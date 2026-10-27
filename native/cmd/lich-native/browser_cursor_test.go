package main

import (
	"image"
	"image/color"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/lichdotdev/lich/native/ui"
)

func clickMark(at time.Time, x, y float64) browserMark { return browserMark{at: at, x: &x, y: &y} }

func TestAgentCursorAppearsAtItsFirstClickAndGlidesToTheNext(t *testing.T) {
	var c agentCursor
	t0 := time.Unix(100, 0)
	c.take(clickMark(t0, 50, 40))
	if !c.shown || c.position(t0) != f32.Pt(50, 40) {
		t.Fatalf("first click at %v, shown %v", c.position(t0), c.shown)
	}
	t1 := t0.Add(time.Second)
	c.take(clickMark(t1, 150, 40))
	mid := c.position(t1.Add(cursorGlide / 2))
	if mid.X <= 50 || mid.X >= 150 {
		t.Fatalf("halfway through the glide at %v, want between the two clicks", mid)
	}
	if got := c.position(t1.Add(cursorGlide)); got != f32.Pt(150, 40) {
		t.Fatalf("after the glide at %v, want the new click", got)
	}
}

func TestAgentCursorGlidesFromWhereItIsWhenAClickInterruptsAGlide(t *testing.T) {
	var c agentCursor
	t0 := time.Unix(100, 0)
	c.take(clickMark(t0, 0, 0))
	t1 := t0.Add(time.Second)
	c.take(clickMark(t1, 100, 0))
	t2 := t1.Add(cursorGlide / 2)
	was := c.position(t2)
	c.take(clickMark(t2, 100, 100))
	if c.from != was {
		t.Fatalf("new glide starts at %v, want where the cursor was, %v", c.from, was)
	}
}

func TestAgentCursorStaysUpWhileTheAgentActsAndThenFades(t *testing.T) {
	var c agentCursor
	t0 := time.Unix(100, 0)
	c.take(clickMark(t0, 10, 10))
	c.take(browserMark{at: t0.Add(time.Second)})
	if a := c.alpha(t0.Add(time.Second + agentHold - time.Millisecond)); a != 1 {
		t.Fatalf("alpha %v before the glow ends, want 1: an action without a point keeps it up", a)
	}
	if a := c.alpha(t0.Add(time.Second + agentHold + cursorFade)); a != 0 {
		t.Fatalf("alpha %v after the fade, want 0", a)
	}
	if c.position(t0.Add(2*time.Second)) != f32.Pt(10, 10) {
		t.Fatal("an action without a point moved the cursor")
	}
}

func TestAgentCursorTakesEachMarkOnce(t *testing.T) {
	var c agentCursor
	t0 := time.Unix(100, 0)
	m := clickMark(t0, 10, 10)
	c.take(m)
	c.take(clickMark(t0.Add(time.Second), 90, 90))
	c.take(m)
	if c.to != f32.Pt(90, 90) {
		t.Fatalf("an old mark taken again moved the cursor to %v", c.to)
	}
}

func TestInkOnKeepsTheThemesPairAndReadsOnCardColors(t *testing.T) {
	white, black := color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{A: 255}
	if got := inkOn(ui.Dark.Primary, ui.Dark.PrimaryForeground); got != ui.Dark.PrimaryForeground {
		t.Fatalf("on the dark theme's primary, ink %v, want its primary-foreground", got)
	}
	for name, c := range cardColors {
		ink := inkOn(c, ui.Dark.PrimaryForeground)
		if ink != white && ink != black && ink != ui.Dark.PrimaryForeground {
			t.Fatalf("%s: ink %v", name, ink)
		}
		if contrast(c, ink) < 3 {
			t.Errorf("%s: badge text contrast %.2f, want at least 3", name, contrast(c, ink))
		}
	}
}

func TestDrawAgentCursorLaysOutWithoutAPage(t *testing.T) {
	th := &ui.Theme{Palette: ui.Dark, Shaper: text.NewShaper(text.WithCollection(gofont.Collection())), Surface: ui.Dark.Canvas}
	u := &window{theme: th}
	var c agentCursor
	t0 := time.Unix(100, 0)
	c.take(clickMark(t0, 40, 30))
	for _, d := range []time.Duration{0, cursorGlide, cursorGlide + cursorDip/2, agentHold + cursorFade/2, agentHold + cursorFade} {
		gtx := layout.Context{Ops: new(op.Ops), Now: t0.Add(d), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(400, 300))}
		u.drawAgentCursor(gtx, &c, 1, th.Primary, th.PrimaryForeground, "Claude Code", providerIcon("claude"))
	}
	if c.shown {
		t.Fatal("cursor still shown after it faded out")
	}
}

func TestAgentCursorScrollShowsItsDirectionInsteadOfAClick(t *testing.T) {
	var c agentCursor
	t0 := time.Unix(100, 0)
	x, y := 200.0, 150.0
	c.take(browserMark{at: t0, x: &x, y: &y, action: "scroll", dy: -300})
	if c.scroll != -1 || c.to != f32.Pt(200, 150) {
		t.Fatalf("scroll up: scroll %d to %v", c.scroll, c.to)
	}
	c.take(clickMark(t0.Add(time.Second), 10, 10))
	if c.scroll != 0 {
		t.Fatalf("a click after a scroll kept scroll %d", c.scroll)
	}
}

func TestAgentCursorStaysWhileTheTurnRunsAndHoldsAfterItEnds(t *testing.T) {
	var c agentCursor
	t0 := time.Unix(100, 0)
	c.take(clickMark(t0, 10, 10))
	thinking := t0.Add(time.Minute)
	c.observe(thinking, true)
	if a := c.alpha(thinking); a != 1 {
		t.Fatalf("alpha %v a minute into the turn, want 1: the agent is thinking between tools", a)
	}
	ended := thinking.Add(time.Second)
	c.observe(ended, false)
	if a := c.alpha(ended.Add(agentHold - time.Millisecond)); a != 1 {
		t.Fatalf("alpha %v within the hold after the turn ended, want 1", a)
	}
	if a := c.alpha(ended.Add(agentHold + cursorFade)); a != 0 {
		t.Fatalf("alpha %v after the hold and the fade, want 0", a)
	}
}
