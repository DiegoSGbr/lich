# Decision: the session browser is a sidecar window

**Status: the card's Browser item opens it.** Right-click a session card →
**Browser**, above Open in. That opens a Chromium window owned by that session.
The agent drives the same page through `lich browser` and the `browser_*` MCP
tools.

## Why not a page inside lich

lich's window is native (`docs/native-window.md`). It does not embed Chromium.
The product is a **second Chromium process** on `chromium.FindBrowser` (system
Chrome, Edge, Chromium, Brave or Vivaldi), with its own user-data directory.
Closing that window does not quit lich. Closing the session closes the sidecar.

## One browser per card

- none yet → headed window at `about:blank`
- already headed → focus that window
- headless only (the agent opened first) → replace it with a headed window at
  the same URL. chromedp cannot show a headless process in place

The agent without the menu uses `browser_open`: headless if nobody asked for a
window, and the headed one if the person already did. MCP and `lich browser`
target `LICH_SESSION_ID`.

## What the agent may load

`http`, `https`, `about:blank`. Not `file:`, `javascript:`, `data:`, `chrome:`.
Page info never includes form values. Screenshots are files. Click, type and
press are real input events. `press` is named keys only. Arbitrary JavaScript
in the page is not a tool.

## Providers

Claude Code and Codex receive the tools at spawn. Antigravity, oh-my-pi,
Crush, Cursor CLI and Kiro receive them from the lich MCP server the plugin
registers. opencode's plugin defines its own tool list, so it does not list
`browser_*`; `lich browser` still runs in that session. The gap is in
`docs/ceilings.md`.
