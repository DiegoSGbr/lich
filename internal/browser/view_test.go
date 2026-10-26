package browser

import (
	"testing"
	"time"

	"github.com/lichdotdev/lich/api"
)

func viewService(t *testing.T) (*Service, *fakeLauncher, *emitted) {
	t.Helper()
	launch := &fakeLauncher{}
	s := testService(t, launch)
	out := &emitted{}
	s.emit = out.emit
	return s, launch, out
}

func TestAgentOpenTellsTheWindowItWasTheAgent(t *testing.T) {
	s, _, out := viewService(t)
	if _, err := s.Open("s", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	states := out.states()
	if len(states) != 1 || !states[0].Open || !states[0].ByAgent || states[0].Owner != "s" {
		t.Fatalf("states = %+v, want one open state by the agent", states)
	}
}

func TestAttachWatchesTheAgentsPageWithoutRelaunching(t *testing.T) {
	s, launch, out := viewService(t)
	if _, err := s.Open("s", "https://example.com/app"); err != nil {
		t.Fatal(err)
	}
	v := Viewport{W: 640, H: 480, Scale: 1.5}
	h, err := s.Attach("s", v)
	if err != nil {
		t.Fatal(err)
	}
	if len(launch.pages) != 1 {
		t.Fatalf("attach launched %d browsers, want the agent's one", len(launch.pages))
	}
	if h.URL != "https://example.com/app" {
		t.Fatalf("attached handle = %+v", h)
	}
	page := launch.pages[0]
	if !page.casting || page.viewports[0] != v {
		t.Fatalf("screencast not started at %+v: casting=%v viewports=%v", v, page.casting, page.viewports)
	}
	page.onFrame("anBlZw==")
	frames := out.named(api.BrowserFrameEvents.Of("s").Name)
	if len(frames) != 1 || frames[0] != "anBlZw==" {
		t.Fatalf("frames = %v", frames)
	}
}

func TestAttachWithoutABrowserStartsOneTheWindowOwns(t *testing.T) {
	s, launch, out := viewService(t)
	if _, err := s.Attach("s", Viewport{W: 800, H: 600, Scale: 1}); err != nil {
		t.Fatal(err)
	}
	if len(launch.pages) != 1 || launch.headed[0] {
		t.Fatalf("attach launched headed=%v, want one headless", launch.headed)
	}
	if st := out.states(); len(st) != 1 || st[0].ByAgent {
		t.Fatalf("states = %+v, want one opened by the window", st)
	}
}

func TestAttachRefusesABrowserInItsOwnWindow(t *testing.T) {
	s, _, _ := viewService(t)
	if _, err := s.OpenVisible("s"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Attach("s", Viewport{W: 800, H: 600, Scale: 1}); err != errInWindow {
		t.Fatalf("attach on a headed browser = %v, want errInWindow", err)
	}
}

func TestReaperLeavesAWatchedBrowserUntilDetach(t *testing.T) {
	s, launch, _ := viewService(t)
	if _, err := s.Attach("s", Viewport{W: 800, H: 600, Scale: 1}); err != nil {
		t.Fatal(err)
	}
	later := time.Date(2026, 8, 16, 13, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return later }
	s.Reap()
	if launch.pages[0].closed {
		t.Fatal("reaper closed a browser a pane is showing")
	}
	if err := s.Detach("s"); err != nil {
		t.Fatal(err)
	}
	if launch.pages[0].casting {
		t.Fatal("detach left the screencast running")
	}
	s.now = func() time.Time { return later.Add(time.Hour) }
	s.Reap()
	if !launch.pages[0].closed {
		t.Fatal("reaper kept an unwatched idle browser")
	}
}

func TestPageNavigatingOnItsOwnUpdatesTheWindow(t *testing.T) {
	s, launch, out := viewService(t)
	if _, err := s.Attach("s", Viewport{W: 800, H: 600, Scale: 1}); err != nil {
		t.Fatal(err)
	}
	launch.pages[0].onInfo("https://example.com/next", "Next")
	states := out.states()
	last := states[len(states)-1]
	if last.URL != "https://example.com/next" || last.Title != "Next" || !last.Open {
		t.Fatalf("last state = %+v", last)
	}
	if run := s.Running(); len(run) != 1 || run[0].Title != "Next" {
		t.Fatalf("running = %+v", run)
	}
}

func TestCloseTellsTheWindow(t *testing.T) {
	s, _, out := viewService(t)
	if _, err := s.Open("s", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseOwnedBy("s"); err != nil {
		t.Fatal(err)
	}
	states := out.states()
	if last := states[len(states)-1]; last.Open || last.Owner != "s" {
		t.Fatalf("last state = %+v, want closed", last)
	}
	if run := s.Running(); len(run) != 0 {
		t.Fatalf("running after close = %+v", run)
	}
}

func TestAgentClickReportsWhereAndWindowInputDoesNot(t *testing.T) {
	s, launch, out := viewService(t)
	if _, err := s.Attach("s", Viewport{W: 800, H: 600, Scale: 1}); err != nil {
		t.Fatal(err)
	}
	launch.pages[0].clickAt = [2]float64{120, 48}
	sel := "#go"
	if err := s.Click("s", Target{Selector: sel}); err != nil {
		t.Fatal(err)
	}
	acts := out.named(api.BrowserAgentEvent.Name)
	if len(acts) != 1 {
		t.Fatalf("agent acts = %v", acts)
	}
	act := acts[0].(api.BrowserAgentAct)
	if act.X == nil || *act.X != 120 || *act.Y != 48 {
		t.Fatalf("agent click at %+v, want 120,48", act)
	}

	if err := s.Nav("s", "reload", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MouseInput(t.Context(), api.BrowserMouseOptions{ID: "s", Type: "press", X: 1, Y: 2, Button: "left"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.KeyInput(t.Context(), api.BrowserKeyOptions{ID: "s", Type: "down", Key: "Enter"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TextInput(t.Context(), api.BrowserTextOptions{ID: "s", Text: "ção"}); err != nil {
		t.Fatal(err)
	}
	if n := len(out.named(api.BrowserAgentEvent.Name)); n != 1 {
		t.Fatalf("window input lit the agent chip: %d agent acts", n)
	}
	p := launch.pages[0]
	if p.reloads != 1 || len(p.mice) != 1 || len(p.keys) != 1 || len(p.texts) != 1 || p.texts[0] != "ção" {
		t.Fatalf("page got reloads=%d mice=%v keys=%v texts=%v", p.reloads, p.mice, p.keys, p.texts)
	}
}

func TestNavRejectsBlockedURLAndUnknownAction(t *testing.T) {
	s, _, _ := viewService(t)
	if _, err := s.Attach("s", Viewport{W: 800, H: 600, Scale: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Nav("s", "go", "file:///etc/passwd"); err == nil {
		t.Fatal("nav loaded a file: URL")
	}
	if err := s.Nav("s", "jump", ""); err == nil {
		t.Fatal("nav accepted an unknown action")
	}
}
