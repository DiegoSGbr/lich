package browser

import (
	"context"
	"log/slog"
	"math"
	"sync"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/lichdotdev/lich/api"
)

// screencastQuality is the JPEG quality of a pane's frames: legible text at a
// fraction of PNG's size, since every frame crosses /events as base64.
const screencastQuality = 70

// listeners are what a chromePage hands its CDP events to; the service sets
// them once, right after the launch.
type listeners struct {
	mu      sync.Mutex
	onFrame func(data string)
	onInfo  func(url, title string)
}

func (p *chromePage) Listen(onFrame func(data string), onInfo func(url, title string)) {
	p.listen.mu.Lock()
	p.listen.onFrame, p.listen.onInfo = onFrame, onInfo
	p.listen.mu.Unlock()
	chromedp.ListenTarget(p.ctx, func(ev any) {
		switch e := ev.(type) {
		case *page.EventScreencastFrame:
			go p.deliver(e)
		case *page.EventFrameNavigated:
			if e.Frame.ParentID == "" {
				go p.reportInfo()
			}
		case *page.EventNavigatedWithinDocument, *page.EventDomContentEventFired:
			go p.reportInfo()
		}
	})
}

// reportInfo reads the URL and title after a navigation: a link followed, a
// history step, a single-page app's pushState, or the document's title
// arriving with its DOM.
func (p *chromePage) reportInfo() {
	p.listen.mu.Lock()
	onInfo := p.listen.onInfo
	p.listen.mu.Unlock()
	if onInfo == nil {
		return
	}
	var info struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	if err := p.run(nil, chromedp.Evaluate(`({url: location.href, title: document.title})`, &info)); err != nil {
		if p.Alive() {
			slog.Warn("browser: read the page's URL and title", "err", err)
		}
		return
	}
	onInfo(info.URL, info.Title)
}

// deliver hands one frame on, then acks it. Chrome sends no next frame until
// the ack, so a window slow to take frames slows the stream instead of
// piling it up.
func (p *chromePage) deliver(frame *page.EventScreencastFrame) {
	p.listen.mu.Lock()
	onFrame := p.listen.onFrame
	p.listen.mu.Unlock()
	if onFrame != nil {
		onFrame(frame.Data)
	}
	ack := chromedp.ActionFunc(func(ctx context.Context) error {
		return page.ScreencastFrameAck(frame.SessionID).Do(ctx)
	})
	if err := p.run(nil, ack); err != nil && p.Alive() {
		slog.Warn("browser: ack a screencast frame", "err", err)
	}
}

func (p *chromePage) Screencast(ctx context.Context, v Viewport) error {
	if err := p.Resize(ctx, v); err != nil {
		return err
	}
	w, h := v.devicePixels()
	return p.run(ctx, page.StartScreencast().
		WithFormat(page.ScreencastFormatJpeg).
		WithQuality(screencastQuality).
		WithMaxWidth(w).
		WithMaxHeight(h))
}

func (p *chromePage) StopScreencast(ctx context.Context) error {
	return p.run(ctx, page.StopScreencast())
}

// Resize lays the page out at the pane's size, as a window of that size would.
func (p *chromePage) Resize(ctx context.Context, v Viewport) error {
	return p.run(ctx, emulation.SetDeviceMetricsOverride(int64(v.W), int64(v.H), v.Scale, false))
}

func (p *chromePage) Mouse(ctx context.Context, m api.BrowserMouseOptions) error {
	kind, ok := mouseTypes[m.Type]
	if !ok {
		return errUnknownInput
	}
	ev := input.DispatchMouseEvent(kind, m.X, m.Y).WithModifiers(input.Modifier(m.Modifiers))
	switch kind {
	case input.MouseWheel:
		ev = ev.WithDeltaX(m.DX).WithDeltaY(m.DY)
	case input.MousePressed, input.MouseReleased:
		ev = ev.WithButton(input.MouseButton(m.Button)).WithClickCount(int64(max(m.Clicks, 1)))
	}
	if kind != input.MouseReleased {
		// buttons is what a drag reads: which button is still held as it moves.
		ev = ev.WithButtons(mouseButtons[m.Button])
	}
	return p.run(ctx, ev)
}

// Key sends a key that types no text. Enter is the exception CDP needs: its
// default action (submit, newline) runs only on a keyDown carrying "\r".
func (p *chromePage) Key(ctx context.Context, k api.BrowserKeyOptions) error {
	var ev *input.DispatchKeyEventParams
	switch {
	case k.Type == "up":
		ev = input.DispatchKeyEvent(input.KeyUp)
	case k.Type != "down":
		return errUnknownInput
	case k.Key == "Enter":
		ev = input.DispatchKeyEvent(input.KeyDown).WithText("\r")
	default:
		ev = input.DispatchKeyEvent(input.KeyRawDown)
	}
	ev = ev.WithKey(k.Key).WithCode(k.Code).
		WithWindowsVirtualKeyCode(int64(k.KeyCode)).
		WithNativeVirtualKeyCode(int64(k.KeyCode)).
		WithModifiers(input.Modifier(k.Modifiers))
	return p.run(ctx, ev)
}

func (p *chromePage) InsertText(ctx context.Context, text string) error {
	return p.run(ctx, input.InsertText(text))
}

var mouseTypes = map[string]input.MouseType{
	"press":   input.MousePressed,
	"release": input.MouseReleased,
	"move":    input.MouseMoved,
	"wheel":   input.MouseWheel,
}

// mouseButtons is CDP's buttons bit set: left 1, right 2, middle 4.
var mouseButtons = map[string]int64{"left": 1, "right": 2, "middle": 4}

func (v Viewport) devicePixels() (w, h int64) {
	return int64(math.Round(float64(v.W) * v.Scale)), int64(math.Round(float64(v.H) * v.Scale))
}
