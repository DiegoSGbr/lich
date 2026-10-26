package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/lichdotdev/lich/api"
	"github.com/lichdotdev/lich/internal/chromium"
)

func liveChrome(t *testing.T) (*chromePage, *httptest.Server) {
	t.Helper()
	if _, err := chromium.FindBrowser(exec.LookPath); err != nil {
		t.Skip("no Chromium-family browser on this machine")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<title>Probe</title>
<form onsubmit="event.preventDefault(); document.body.dataset.sent = this.q.value">
<input id="f" name="q" style="position:absolute;left:10px;top:10px;width:200px;height:30px"
 oninput="document.body.dataset.seen = this.value"></form>`)
	}))
	t.Cleanup(srv.Close)
	p, err := chromeLaunch(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	return p.(*chromePage), srv
}

func TestChromeClearReachesTheFieldsListener(t *testing.T) {
	p, srv := liveChrome(t)
	ctx, cancel := context.WithTimeout(context.Background(), pageWait)
	defer cancel()
	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatal(err)
	}
	field := Target{Selector: "#f"}
	if err := p.Type(ctx, "abc", false, field); err != nil {
		t.Fatal(err)
	}
	if err := p.Type(ctx, "", true, field); err != nil {
		t.Fatal(err)
	}
	var seen string
	if err := p.run(ctx, chromedp.Evaluate(`document.body.dataset.seen`, &seen)); err != nil {
		t.Fatal(err)
	}
	if seen != "" {
		t.Fatalf("the page's input listener last saw %q; clearing fired no input event", seen)
	}
}

func TestChromeCookiesKeepASessionCookie(t *testing.T) {
	p, srv := liveChrome(t)
	ctx, cancel := context.WithTimeout(context.Background(), pageWait)
	defer cancel()
	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatal(err)
	}
	in := []*network.CookieParam{{Name: "sid", Value: "1", URL: srv.URL}}
	if err := p.SetCookies(ctx, in); err != nil {
		t.Fatal(err)
	}
	got, err := p.Cookies(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c.Name == "sid" {
			if c.Value != "1" || c.Expires != nil {
				t.Fatalf("sid read back as %+v; a session cookie must stay one", c)
			}
			return
		}
	}
	t.Fatalf("sid missing from %d cookies", len(got))
}

func TestPromoteRelaunchesOnTheProfileWithItsCookies(t *testing.T) {
	_, srv := liveChrome(t)
	// headless both times: what is under test is the relaunch on the same
	// profile, and a headed window needs a display the suite may not have.
	launch := func(dir string, _ bool) (Page, error) { return chromeLaunch(dir, false) }
	s := newService(launch, func(string, string) (string, error) { return t.TempDir(), nil }, time.Now, idleAfter)
	t.Cleanup(func() {
		if err := s.Cleanup(); err != nil {
			t.Error(err)
		}
	})
	if _, err := s.Open("s", srv.URL); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), pageWait)
	defer cancel()
	in := []*network.CookieParam{{Name: "sid", Value: "1", URL: srv.URL}}
	if err := s.byOwner["s"].page.SetCookies(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenVisible("s"); err != nil {
		t.Fatal(err)
	}
	got, err := s.byOwner["s"].page.Cookies(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c.Name == "sid" && c.Value == "1" {
			return
		}
	}
	t.Fatalf("session cookie lost in the promotion: %d cookies", len(got))
}

func TestCookieParamGivesAPersistentCookieItsExpiry(t *testing.T) {
	when := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	got := cookieParam(&network.Cookie{Name: "k", Expires: float64(when.Unix())})
	if got.Expires == nil || !got.Expires.Time().Equal(when) {
		t.Fatalf("expires = %v, want %v", got.Expires, when)
	}
}

func TestChromeScreencastSendsFramesAndPageInfo(t *testing.T) {
	p, srv := liveChrome(t)
	frames := make(chan string, 4)
	titles := make(chan string, 4)
	p.Listen(func(data string) {
		select {
		case frames <- data:
		default:
		}
	}, func(_, title string) {
		select {
		case titles <- title:
		default:
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), pageWait)
	defer cancel()
	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := p.Screencast(ctx, Viewport{W: 400, H: 300, Scale: 1}); err != nil {
		t.Fatal(err)
	}
	select {
	case data := <-frames:
		img, err := base64.StdEncoding.DecodeString(data)
		if err != nil || len(img) < 2 || img[0] != 0xFF || img[1] != 0xD8 {
			t.Fatalf("frame is not a JPEG (err %v, %d bytes)", err, len(img))
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no screencast frame")
	}
	deadline := time.After(10 * time.Second)
	for {
		select {
		case title := <-titles:
			if title == "Probe" {
				return
			}
		case <-deadline:
			t.Fatal("no page info with the document's title")
		}
	}
}

func TestChromeWindowInputTypesAndSubmits(t *testing.T) {
	p, srv := liveChrome(t)
	ctx, cancel := context.WithTimeout(context.Background(), pageWait)
	defer cancel()
	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"press", "release"} {
		if err := p.Mouse(ctx, api.BrowserMouseOptions{Type: kind, X: 50, Y: 25, Button: "left", Clicks: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.InsertText(ctx, "ação"); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"down", "up"} {
		if err := p.Key(ctx, api.BrowserKeyOptions{Type: kind, Key: "Enter", Code: "Enter", KeyCode: 13}); err != nil {
			t.Fatal(err)
		}
	}
	var sent string
	if err := p.run(ctx, chromedp.Evaluate(`document.body.dataset.sent || ""`, &sent)); err != nil {
		t.Fatal(err)
	}
	if sent != "ação" {
		t.Fatalf("form sent %q, want the clicked field's typed text", sent)
	}
}
