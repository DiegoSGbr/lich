package browser

import (
	"context"

	"github.com/lichdotdev/lich/api"
)

// The methods below are the RPC surface (api.Browser*). The ones they call
// keep the shape the package tests. Neither Cleanup nor CloseOwnedBy is
// routed: one ends every browser, the other is the session-close hook.

// Show brings this session's browser on screen.
func (s *Service) Show(_ context.Context, opts api.SessionIDOptions) (api.BrowserHandle, error) {
	h, err := s.OpenVisible(opts.ID)
	return handleAPI(h), err
}

// OpenURL navigates this session's browser, creating a headless one when the
// card has not opened a window.
func (s *Service) OpenURL(_ context.Context, opts api.BrowserOpenOptions) (api.BrowserHandle, error) {
	h, err := s.Open(opts.ID, opts.URL)
	return handleAPI(h), err
}

// Read returns the page the agent is looking at.
func (s *Service) Read(_ context.Context, opts api.SessionIDOptions) (api.BrowserPageInfo, error) {
	info, err := s.Info(opts.ID)
	return infoAPI(info), err
}

// ClickAt clicks in this session's browser.
func (s *Service) ClickAt(_ context.Context, opts api.BrowserClickOptions) (api.None, error) {
	return api.None{}, s.Click(opts.ID, targetOf(opts.BrowserTarget))
}

// TypeText types into this session's browser.
func (s *Service) TypeText(_ context.Context, opts api.BrowserTypeOptions) (api.None, error) {
	return api.None{}, s.Type(opts.ID, opts.Text, opts.Clear, targetOf(opts.BrowserTarget))
}

// PressKey presses one named key in this session's browser.
func (s *Service) PressKey(_ context.Context, opts api.BrowserPressOptions) (api.None, error) {
	return api.None{}, s.Press(opts.ID, opts.Key, targetOf(opts.BrowserTarget))
}

// Capture writes a PNG of this session's page and returns its path.
func (s *Service) Capture(_ context.Context, opts api.BrowserScreenshotOptions) (string, error) {
	return s.Screenshot(opts.ID, opts.Path)
}

// Go loads a URL in this session's already-open browser.
func (s *Service) Go(_ context.Context, opts api.BrowserOpenOptions) (api.BrowserHandle, error) {
	h, err := s.Navigate(opts.ID, opts.URL)
	return handleAPI(h), err
}

// ReloadPage reloads this session's page.
func (s *Service) ReloadPage(_ context.Context, opts api.SessionIDOptions) (api.None, error) {
	return api.None{}, s.Reload(opts.ID)
}

// GoBack goes back in this session's history.
func (s *Service) GoBack(_ context.Context, opts api.SessionIDOptions) (api.None, error) {
	return api.None{}, s.Back(opts.ID)
}

// GoForward goes forward in this session's history.
func (s *Service) GoForward(_ context.Context, opts api.SessionIDOptions) (api.None, error) {
	return api.None{}, s.Forward(opts.ID)
}

// ScrollBy scrolls this session's page.
func (s *Service) ScrollBy(_ context.Context, opts api.BrowserScrollOptions) (api.None, error) {
	return api.None{}, s.Scroll(opts.ID, opts.DY)
}

// Handles returns this session's browser, or an empty list.
func (s *Service) Handles(_ context.Context, opts api.SessionIDOptions) ([]api.BrowserHandle, error) {
	list, err := s.List(opts.ID)
	if err != nil {
		return nil, err
	}
	out := make([]api.BrowserHandle, len(list))
	for i, h := range list {
		out[i] = handleAPI(h)
	}
	return out, nil
}

// CloseOne tears down this session's browser.
func (s *Service) CloseOne(_ context.Context, opts api.SessionIDOptions) (api.None, error) {
	return api.None{}, s.Close(opts.ID)
}

func handleAPI(h Handle) api.BrowserHandle {
	return api.BrowserHandle{ID: h.ID, URL: h.URL, Title: h.Title, Owner: h.Owner, Headed: h.Headed}
}

func infoAPI(p PageInfo) api.BrowserPageInfo {
	out := api.BrowserPageInfo{URL: p.URL, Title: p.Title, Text: p.Text, Truncated: p.Truncated}
	if p.Interactive == nil {
		return out
	}
	out.Interactive = make([]api.BrowserInteractive, len(p.Interactive))
	for i, n := range p.Interactive {
		out.Interactive[i] = api.BrowserInteractive{
			Index: n.Index, Tag: n.Tag, Type: n.Type, Role: n.Role, Text: n.Text, Href: n.Href,
			Rect:    api.BrowserRect{X: n.Rect.X, Y: n.Rect.Y, W: n.Rect.W, H: n.Rect.H},
			Filled:  n.Filled,
			Checked: n.Checked,
		}
	}
	return out
}

func targetOf(t api.BrowserTarget) Target {
	return Target{Index: t.Index, Selector: t.Selector, X: t.X, Y: t.Y}
}
