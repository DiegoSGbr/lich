package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lichdotdev/lich/api"
)

// browserTools drive this session's browser, the page its tab in the lich
// window shows. Appended to mcpTools, so tools/list cannot drift from the
// commands.
var browserTools = []mcpTool{
	{
		Name: "browser_open",
		Description: "Open a page in this lich session's browser, creating the browser if " +
			"this session has none. It runs headless: the user sees the same page in a tab " +
			"beside this session, which lich offers to open, so no window of its own is needed " +
			"(headed is false, and that is expected). The URL must be http, https, or omitted " +
			"(about:blank). file, javascript and data URLs are refused.",
		Schema: schema(map[string]any{
			"url": property("string", "Page to load. Omit for about:blank."),
		}),
		Run: func(ctx context.Context, c *client, a mcpArgs) (string, error) {
			id, err := c.browserOwner()
			if err != nil {
				return "", err
			}
			h, err := callMethod(ctx, c, api.BrowserOpen, api.BrowserOpenOptions{ID: id, URL: a.text("url")}, openCall)
			if err != nil {
				return "", err
			}
			return encodeJSON(h)
		},
	},
	{
		Name: "browser_info",
		Description: "Read this session's current page: URL, title, the text in the viewport, and " +
			"interactive elements there (index, tag, label, href, box). Form values are never " +
			"included — filled/checked say that typing landed. Call this before click/type " +
			"so indexes match. Read-only.",
		Schema:   schema(map[string]any{}),
		ReadOnly: true,
		Run:      mcpBrowserInfo,
	},
	{
		Name: "browser_click",
		Description: "Click in this session's browser. Prefer index from browser_info, or a " +
			"CSS selector, or viewport x and y. Sends a real mouse event, not element.click().",
		Schema: schema(map[string]any{
			"index":    property("integer", "Index from the last browser_info."),
			"selector": property("string", "CSS selector."),
			"x":        property("number", "Viewport X."),
			"y":        property("number", "Viewport Y."),
		}),
		Run: func(ctx context.Context, c *client, a mcpArgs) (string, error) {
			return mcpBrowserClick(ctx, c, a)
		},
	},
	{
		Name: "browser_type",
		Description: "Type into this session's browser. Target as for browser_click; omit it " +
			"to type into whatever is focused. clear replaces the field first. The text never " +
			"comes back through browser_info.",
		Schema: schema(map[string]any{
			"text":     property("string", "Keys to send."),
			"clear":    property("boolean", "Replace the field before typing."),
			"index":    property("integer", "Index from the last browser_info."),
			"selector": property("string", "CSS selector."),
			"x":        property("number", "Viewport X."),
			"y":        property("number", "Viewport Y."),
		}, "text"),
		Run: func(ctx context.Context, c *client, a mcpArgs) (string, error) {
			id, err := c.browserOwner()
			if err != nil {
				return "", err
			}
			_, err = callMethod(ctx, c, api.BrowserType, api.BrowserTypeOptions{
				ID: id, Text: a.text("text"), Clear: a.flag("clear"), BrowserTarget: mcpTarget(a),
			}, shortCall)
			if err != nil {
				return "", err
			}
			return "ok", nil
		},
	},
	{
		Name: "browser_press",
		Description: "Press one named key in this session's browser (Enter, Tab, Escape, " +
			"Backspace, Delete, Space, ArrowUp/Down/Left/Right, Home, End, PageUp, PageDown). " +
			"Real key event, not typed text — use browser_type for that. Optional target " +
			"focuses first, as for browser_type.",
		Schema: schema(map[string]any{
			"key":      property("string", "Named key: Enter, Tab, Escape, …"),
			"index":    property("integer", "Index from the last browser_info."),
			"selector": property("string", "CSS selector."),
			"x":        property("number", "Viewport X."),
			"y":        property("number", "Viewport Y."),
		}, "key"),
		Run: func(ctx context.Context, c *client, a mcpArgs) (string, error) {
			id, err := c.browserOwner()
			if err != nil {
				return "", err
			}
			_, err = callMethod(ctx, c, api.BrowserPress, api.BrowserPressOptions{
				ID: id, Key: a.text("key"), BrowserTarget: mcpTarget(a),
			}, shortCall)
			if err != nil {
				return "", err
			}
			return "ok", nil
		},
	},
	{
		Name: "browser_screenshot",
		Description: "Capture this session's page to a PNG file and return the path. Do not " +
			"inline the bytes. Optional path; otherwise lich writes one next to the browser profile.",
		Schema: schema(map[string]any{
			"path": property("string", "Destination PNG. Omit to let lich choose."),
		}),
		ReadOnly: true,
		Run: func(ctx context.Context, c *client, a mcpArgs) (string, error) {
			id, err := c.browserOwner()
			if err != nil {
				return "", err
			}
			return callMethod(ctx, c, api.BrowserScreenshot, api.BrowserScreenshotOptions{ID: id, Path: a.text("path")}, openCall)
		},
	},
	{
		Name:        "browser_navigate",
		Description: "Load a URL in this session's already-open browser. Same allowlist as browser_open.",
		Schema: schema(map[string]any{
			"url": property("string", "http(s) URL, or about:blank."),
		}, "url"),
		Run: func(ctx context.Context, c *client, a mcpArgs) (string, error) {
			id, err := c.browserOwner()
			if err != nil {
				return "", err
			}
			h, err := callMethod(ctx, c, api.BrowserNavigate, api.BrowserOpenOptions{ID: id, URL: a.text("url")}, openCall)
			if err != nil {
				return "", err
			}
			return encodeJSON(h)
		},
	},
	{
		Name:        "browser_reload",
		Description: "Reload this session's current page.",
		Schema:      schema(map[string]any{}),
		Run: func(ctx context.Context, c *client, _ mcpArgs) (string, error) {
			return mcpBrowserID(ctx, c, api.BrowserReload)
		},
	},
	{
		Name:        "browser_back",
		Description: "Go back in this session's browser history.",
		Schema:      schema(map[string]any{}),
		Run: func(ctx context.Context, c *client, _ mcpArgs) (string, error) {
			return mcpBrowserID(ctx, c, api.BrowserBack)
		},
	},
	{
		Name:        "browser_forward",
		Description: "Go forward in this session's browser history.",
		Schema:      schema(map[string]any{}),
		Run: func(ctx context.Context, c *client, _ mcpArgs) (string, error) {
			return mcpBrowserID(ctx, c, api.BrowserForward)
		},
	},
	{
		Name:        "browser_scroll",
		Description: "Scroll this session's page. dy is CSS pixels; positive is down. Default 400.",
		Schema: schema(map[string]any{
			"dy": property("integer", "Pixels to scroll. Default 400."),
		}),
		Run: func(ctx context.Context, c *client, a mcpArgs) (string, error) {
			id, err := c.browserOwner()
			if err != nil {
				return "", err
			}
			dy := 400
			if n := a.optionalInt("dy"); n != nil {
				dy = *n
			}
			_, err = callMethod(ctx, c, api.BrowserScroll, api.BrowserScrollOptions{ID: id, DY: dy}, shortCall)
			if err != nil {
				return "", err
			}
			return "ok", nil
		},
	},
	{
		Name:        "browser_list",
		Description: "This session's browser, or an empty list. Read-only.",
		Schema:      schema(map[string]any{}),
		ReadOnly:    true,
		Run:         mcpBrowserList,
	},
	{
		Name: "browser_close",
		Description: "Close this session's browser, and with it the page's tab in the lich window. " +
			"Closing the session card also closes it.",
		Schema: schema(map[string]any{}),
		Run: func(ctx context.Context, c *client, _ mcpArgs) (string, error) {
			return mcpBrowserID(ctx, c, api.BrowserClose)
		},
	},
}

func (c *client) browserOwner() (string, error) {
	id := c.sessionID()
	if id == "" {
		return "", fmt.Errorf("browser tools run inside a lich session")
	}
	return id, nil
}

func mcpTarget(a mcpArgs) api.BrowserTarget {
	return api.BrowserTarget{
		Selector: a.text("selector"),
		Index:    a.optionalInt("index"),
		X:        a.optionalFloat("x"),
		Y:        a.optionalFloat("y"),
	}
}

func mcpBrowserClick(ctx context.Context, c *client, a mcpArgs) (string, error) {
	id, err := c.browserOwner()
	if err != nil {
		return "", err
	}
	_, err = callMethod(ctx, c, api.BrowserClick, api.BrowserClickOptions{ID: id, BrowserTarget: mcpTarget(a)}, shortCall)
	if err != nil {
		return "", err
	}
	return "ok", nil
}

func mcpBrowserID(ctx context.Context, c *client, method api.Method[api.SessionIDOptions, api.None]) (string, error) {
	id, err := c.browserOwner()
	if err != nil {
		return "", err
	}
	if _, err := callMethod(ctx, c, method, api.SessionIDOptions{ID: id}, shortCall); err != nil {
		return "", err
	}
	return "ok", nil
}

func encodeJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func mcpBrowserInfo(ctx context.Context, c *client, _ mcpArgs) (string, error) {
	id, err := c.browserOwner()
	if err != nil {
		return "", err
	}
	info, err := callMethod(ctx, c, api.BrowserInfo, api.SessionIDOptions{ID: id}, openCall)
	if err != nil {
		return "", err
	}
	return encodeJSON(info)
}

func mcpBrowserList(ctx context.Context, c *client, _ mcpArgs) (string, error) {
	id, err := c.browserOwner()
	if err != nil {
		return "", err
	}
	list, err := callMethod(ctx, c, api.BrowserList, api.SessionIDOptions{ID: id}, shortCall)
	if err != nil {
		return "", err
	}
	if list == nil {
		list = []api.BrowserHandle{}
	}
	return encodeJSON(list)
}

// optionalInt reads a number that may be absent. Missing, empty, or a type
// this cannot parse is nil rather than zero: zero is a valid element index.
func (a mcpArgs) optionalInt(key string) *int {
	v, ok := a[key]
	if !ok || v == nil {
		return nil
	}
	n := 0
	switch x := v.(type) {
	case float64:
		n = int(x)
	case string:
		if x == "" {
			return nil
		}
		if _, err := fmt.Sscanf(x, "%d", &n); err != nil {
			return nil
		}
	default:
		return nil
	}
	return &n
}

// optionalFloat is optionalInt for viewport coordinates.
func (a mcpArgs) optionalFloat(key string) *float64 {
	v, ok := a[key]
	if !ok || v == nil {
		return nil
	}
	n := 0.0
	switch x := v.(type) {
	case float64:
		n = x
	case string:
		if x == "" {
			return nil
		}
		if _, err := fmt.Sscanf(x, "%f", &n); err != nil {
			return nil
		}
	default:
		return nil
	}
	return &n
}
