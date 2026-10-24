package api

var (
	BrowserShow       = define[SessionIDOptions, BrowserHandle]("browser.Show")
	BrowserOpen       = define[BrowserOpenOptions, BrowserHandle]("browser.Open")
	BrowserInfo       = define[SessionIDOptions, BrowserPageInfo]("browser.Info")
	BrowserClick      = define[BrowserClickOptions, None]("browser.Click")
	BrowserType       = define[BrowserTypeOptions, None]("browser.Type")
	BrowserPress      = define[BrowserPressOptions, None]("browser.Press")
	BrowserScreenshot = define[BrowserScreenshotOptions, string]("browser.Screenshot")
	BrowserNavigate   = define[BrowserOpenOptions, BrowserHandle]("browser.Navigate")
	BrowserReload     = define[SessionIDOptions, None]("browser.Reload")
	BrowserBack       = define[SessionIDOptions, None]("browser.Back")
	BrowserForward    = define[SessionIDOptions, None]("browser.Forward")
	BrowserScroll     = define[BrowserScrollOptions, None]("browser.Scroll")
	BrowserList       = define[SessionIDOptions, []BrowserHandle]("browser.List")
	BrowserClose      = define[SessionIDOptions, None]("browser.Close")
)

// BrowserHandle is one session's browser: the id later calls take, and enough
// for a tool result to say which window it is.
type BrowserHandle struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Title  string `json:"title"`
	Owner  string `json:"owner"`
	Headed bool   `json:"headed"`
}

// BrowserOpenOptions loads a URL in the session's browser, creating a headless
// one when the card has not opened a window yet.
type BrowserOpenOptions struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// BrowserTarget picks an element. Index is the one page-info stamped;
// selector is a CSS query; x/y are viewport coordinates. Coordinates win when
// set, then selector, then index.
type BrowserTarget struct {
	Index    *int     `json:"index,omitempty"`
	Selector string   `json:"selector,omitempty"`
	X        *float64 `json:"x,omitempty"`
	Y        *float64 `json:"y,omitempty"`
}

// BrowserClickOptions clicks in the session's browser.
type BrowserClickOptions struct {
	ID string `json:"id"`
	BrowserTarget
}

// BrowserTypeOptions types into the session's browser. An empty target types
// into whatever is focused.
type BrowserTypeOptions struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Clear bool   `json:"clear"`
	BrowserTarget
}

// BrowserPressOptions presses one named key. An empty target presses into
// whatever is focused.
type BrowserPressOptions struct {
	ID  string `json:"id"`
	Key string `json:"key"`
	BrowserTarget
}

// BrowserScreenshotOptions captures the page to Path, or to a file lich
// chooses when Path is empty.
type BrowserScreenshotOptions struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// BrowserScrollOptions scrolls the page by DY CSS pixels. Positive is down.
type BrowserScrollOptions struct {
	ID string `json:"id"`
	DY int    `json:"dy"`
}

// BrowserRect is an element's viewport box, integer pixels.
type BrowserRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// BrowserInteractive is one clickable or fillable node. It never carries a
// control's value: filled and checked say that typing landed.
type BrowserInteractive struct {
	Index   int         `json:"index"`
	Tag     string      `json:"tag"`
	Type    string      `json:"type,omitempty"`
	Role    string      `json:"role,omitempty"`
	Text    string      `json:"text"`
	Href    string      `json:"href,omitempty"`
	Rect    BrowserRect `json:"rect"`
	Filled  *bool       `json:"filled,omitempty"`
	Checked *bool       `json:"checked,omitempty"`
}

// BrowserPageInfo is what the agent reads instead of a screenshot.
type BrowserPageInfo struct {
	URL         string               `json:"url"`
	Title       string               `json:"title"`
	Text        string               `json:"text"`
	Interactive []BrowserInteractive `json:"interactive"`
	Truncated   bool                 `json:"truncated"`
}
