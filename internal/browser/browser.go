package browser

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
)

// idleAfter is how long a headless browser sits unused before the reaper
// closes it. Five minutes matches the "walked away from the card" window
// without holding a Chromium for a session that finished hours ago. A headed
// window is never reaped: lich cannot see a person using it, so only the card
// closing ends it.
const idleAfter = 5 * time.Minute

// reapEvery is how often the reaper looks. Shorter than idleAfter so a
// browser that went idle is gone within a fraction of the idle window, not a
// full extra one.
const reapEvery = 30 * time.Second

// pageWait bounds one CDP action. Under the 90 seconds an MCP tool call may
// block for, long enough for a cold page, short enough that a hung Chrome
// does not take the tool with it.
const pageWait = 30 * time.Second

var (
	errNoOwner = errors.New("browser needs a lich session id")
	errStopped = errors.New("browser service is shutting down")
)

// Service is one Chromium context per lich session: the window Browser tab
// opens and the page the agent drives are the same one. Headless until
// someone asks for a window; headed after OpenVisible.
type Service struct {
	mu      sync.Mutex
	launch  Launcher
	mkTmp   func(dir, pattern string) (string, error)
	now     func() time.Time
	idle    time.Duration
	byOwner map[string]*session
	byID    map[string]*session
	// owners serializes one session's launch, promotion and reuse, so a slow
	// Chrome start holds that card alone and mu is never held across CDP.
	owners map[string]*sync.Mutex
	// dead holds what the locked code took out of the maps; unlock closes it
	// after mu is released, because closing waits for Chrome to exit.
	dead    []*session
	stop    chan struct{}
	stopped bool
}

type session struct {
	id, owner, dir string
	url, title     string
	headed         bool
	page           Page
	lastUsed       time.Time
}

// New starts a service that launches the system Chromium (chromium.FindBrowser)
// into an ephemeral profile, never the lich --app one.
func New() *Service {
	s := newService(chromeLaunch, os.MkdirTemp, time.Now, idleAfter)
	s.startReaper()
	return s
}

func newService(launch Launcher, mkTmp func(string, string) (string, error), now func() time.Time, idle time.Duration) *Service {
	return &Service{
		launch:  launch,
		mkTmp:   mkTmp,
		now:     now,
		idle:    idle,
		byOwner: make(map[string]*session),
		byID:    make(map[string]*session),
		owners:  make(map[string]*sync.Mutex),
		stop:    make(chan struct{}),
	}
}

// Open navigates this session's browser to url, creating a headless one if
// none exists. A headed window the user already opened is reused, not replaced.
func (s *Service) Open(owner, rawURL string) (Handle, error) {
	if owner == "" {
		return Handle{}, errNoOwner
	}
	u, err := allowURL(rawURL)
	if err != nil {
		return Handle{}, err
	}
	lock := s.ownerLock(owner)
	lock.Lock()
	defer lock.Unlock()
	sess := s.current(owner)
	if sess == nil {
		return s.launchFresh(owner, u, false)
	}
	if err := runPage(func(ctx context.Context) error {
		return sess.page.Navigate(ctx, u)
	}); err != nil {
		s.drop(sess)
		return Handle{}, err
	}
	s.remember(sess, u, "")
	return s.handleOf(sess), nil
}

// OpenVisible brings this session's browser on screen. No browser yet: a
// headed window at about:blank. Already headed: focus it. Headless only:
// reopen it as a window (promote).
func (s *Service) OpenVisible(owner string) (Handle, error) {
	if owner == "" {
		return Handle{}, errNoOwner
	}
	lock := s.ownerLock(owner)
	lock.Lock()
	defer lock.Unlock()
	sess := s.current(owner)
	switch {
	case sess == nil:
		return s.launchFresh(owner, "about:blank", true)
	case !sess.headed:
		return s.promote(sess)
	}
	if err := runPage(sess.page.Focus); err != nil {
		s.drop(sess)
		return Handle{}, err
	}
	s.touch(sess)
	return s.handleOf(sess), nil
}

// promote reopens a headless browser as a window on the same profile and
// carries its cookies across: chromedp cannot show a headless process in
// place, and a session cookie lives only in the process. What else lives only
// in the page (a half-filled form, sessionStorage) does not survive.
func (s *Service) promote(sess *session) (Handle, error) {
	var cookies []*network.CookieParam
	if err := runPage(func(ctx context.Context) error {
		var err error
		cookies, err = sess.page.Cookies(ctx)
		return err
	}); err != nil {
		s.drop(sess)
		return Handle{}, err
	}
	s.mu.Lock()
	s.detachLocked(sess)
	s.unlock()
	if err := sess.page.Close(); err != nil {
		return Handle{}, errors.Join(err, os.RemoveAll(sess.dir))
	}
	u := sess.url
	if u == "" {
		u = "about:blank"
	}
	return s.start(sess.owner, u, true, sess.dir, cookies)
}

// List returns this session's browser, or an empty list. Never null: a
// script should not have to tell those apart.
func (s *Service) List(owner string) ([]Handle, error) {
	if owner == "" {
		return nil, errNoOwner
	}
	out := []Handle{}
	if sess := s.current(owner); sess != nil {
		out = append(out, s.handleOf(sess))
	}
	return out, nil
}

// Close tears down one browser, by handle id or by the lich session that owns it.
func (s *Service) Close(id string) error {
	s.mu.Lock()
	sess := s.lookupLocked(id)
	if sess != nil {
		s.detachLocked(sess)
	}
	s.unlock()
	if sess == nil {
		return nil
	}
	return sess.destroy()
}

// CloseOwnedBy tears down the browser of a lich session that is going away.
// Missing is not an error: most sessions never opened one.
func (s *Service) CloseOwnedBy(owner string) error {
	return s.Close(owner)
}

// Cleanup closes every browser this process started. Wired from main so a
// window-close exit does not leave Chromiums behind.
func (s *Service) Cleanup() error {
	s.mu.Lock()
	if !s.stopped {
		s.stopped = true
		close(s.stop)
	}
	all := append([]*session{}, s.dead...)
	s.dead = nil
	for _, sess := range s.byID {
		s.detachLocked(sess)
		all = append(all, sess)
	}
	s.mu.Unlock()
	var errs []error
	for _, sess := range all {
		errs = append(errs, sess.destroy())
	}
	return errors.Join(errs...)
}

func (s *Service) launchFresh(owner, u string, headed bool) (Handle, error) {
	dir, err := s.mkTmp("", "lich-browser-")
	if err != nil {
		return Handle{}, err
	}
	return s.start(owner, u, headed, dir, nil)
}

// start launches a browser on dir, loads cookies and then u, and registers it.
// The caller holds owner's lock; mu is taken only to register.
func (s *Service) start(owner, u string, headed bool, dir string, cookies []*network.CookieParam) (Handle, error) {
	page, err := s.launch(dir, headed)
	if err != nil {
		return Handle{}, errors.Join(err, os.RemoveAll(dir))
	}
	if err := runPage(func(ctx context.Context) error {
		if len(cookies) > 0 {
			if err := page.SetCookies(ctx, cookies); err != nil {
				return err
			}
		}
		return page.Navigate(ctx, u)
	}); err != nil {
		return Handle{}, errors.Join(err, page.Close(), os.RemoveAll(dir))
	}
	sess := &session{
		id: newID(), owner: owner, dir: dir, url: u, headed: headed, page: page, lastUsed: s.now(),
	}
	s.mu.Lock()
	defer s.unlock()
	if s.stopped {
		s.dead = append(s.dead, sess)
		return Handle{}, errStopped
	}
	s.byOwner[owner] = sess
	s.byID[sess.id] = sess
	return sess.handle(), nil
}

func (s *Service) ownerLock(owner string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock := s.owners[owner]
	if lock == nil {
		lock = &sync.Mutex{}
		s.owners[owner] = lock
	}
	return lock
}

// current is owner's live browser after a reap, or nil.
func (s *Service) current(owner string) *session {
	s.mu.Lock()
	defer s.unlock()
	s.reapLocked()
	return s.liveByOwner(owner)
}

func (s *Service) handleOf(sess *session) Handle {
	s.mu.Lock()
	defer s.mu.Unlock()
	return sess.handle()
}

func (s *Service) liveByOwner(owner string) *session {
	sess := s.byOwner[owner]
	if sess == nil {
		return nil
	}
	if !sess.alive() {
		s.buryLocked(sess)
		return nil
	}
	return sess
}

func (s *Service) lookupLocked(id string) *session {
	for _, sess := range []*session{s.byID[id], s.byOwner[id]} {
		if sess == nil {
			continue
		}
		if sess.alive() {
			return sess
		}
		s.buryLocked(sess)
	}
	return nil
}

func (s *Service) detachLocked(sess *session) {
	delete(s.byID, sess.id)
	if s.byOwner[sess.owner] == sess {
		delete(s.byOwner, sess.owner)
	}
}

// buryLocked takes sess out of the maps and leaves it for unlock to close.
func (s *Service) buryLocked(sess *session) {
	s.detachLocked(sess)
	s.dead = append(s.dead, sess)
}

// unlock releases mu, then closes what the locked section buried. Nobody waits
// on these closes, so a failure is logged rather than returned.
func (s *Service) unlock() {
	dead := s.dead
	s.dead = nil
	s.mu.Unlock()
	for _, sess := range dead {
		if err := sess.destroy(); err != nil {
			slog.Warn("close a session browser", "owner", sess.owner, "err", err)
		}
	}
}

func (s *session) destroy() error {
	return errors.Join(s.page.Close(), os.RemoveAll(s.dir))
}

func (s *session) alive() bool {
	return s != nil && s.page != nil && s.page.Alive()
}

func (s *session) handle() Handle {
	return Handle{ID: s.id, URL: s.url, Title: s.title, Owner: s.owner, Headed: s.headed}
}

func (s *Service) startReaper() {
	go func() {
		tick := time.NewTicker(reapEvery)
		defer tick.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-tick.C:
				s.Reap()
			}
		}
	}()
}

// Reap closes headless browsers that have sat idle past the idle window.
func (s *Service) Reap() {
	s.mu.Lock()
	defer s.unlock()
	s.reapLocked()
}

func (s *Service) reapLocked() {
	cutoff := s.now().Add(-s.idle)
	for _, sess := range s.byID {
		if !sess.headed && sess.lastUsed.Before(cutoff) {
			s.buryLocked(sess)
		}
	}
}

func runPage(fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), pageWait)
	defer cancel()
	return fn(ctx)
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
