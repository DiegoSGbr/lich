package routes

import (
	"github.com/lichdotdev/lich/api"
	"github.com/lichdotdev/lich/internal/browser"
	"github.com/lichdotdev/lich/internal/rpc"
)

// Browser exposes the session browser as api declares it. Cleanup and
// CloseOwnedBy stay out: one ends every browser this process started, the
// other runs when the session that owns one is closed.
func Browser(d *rpc.Handler, s *browser.Service) {
	rpc.Handle(d, api.BrowserShow, s.Show)
	rpc.Handle(d, api.BrowserOpen, s.OpenURL)
	rpc.Handle(d, api.BrowserInfo, s.Read)
	rpc.Handle(d, api.BrowserClick, s.ClickAt)
	rpc.Handle(d, api.BrowserType, s.TypeText)
	rpc.Handle(d, api.BrowserPress, s.PressKey)
	rpc.Handle(d, api.BrowserScreenshot, s.Capture)
	rpc.Handle(d, api.BrowserNavigate, s.Go)
	rpc.Handle(d, api.BrowserReload, s.ReloadPage)
	rpc.Handle(d, api.BrowserBack, s.GoBack)
	rpc.Handle(d, api.BrowserForward, s.GoForward)
	rpc.Handle(d, api.BrowserScroll, s.ScrollBy)
	rpc.Handle(d, api.BrowserList, s.Handles)
	rpc.Handle(d, api.BrowserClose, s.CloseOne)
}
