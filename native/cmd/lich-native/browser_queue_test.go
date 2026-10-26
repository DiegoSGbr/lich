package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/lichdotdev/lich/api"
)

func TestBrowserQueueMergesWhatPilesUpBehindACall(t *testing.T) {
	q := newBrowserQueue()
	var sent []string
	call := func(m api.BrowserMouseOptions) browserCall {
		return browserCall{mouse: &m, send: func(context.Context) error {
			sent = append(sent, m.Type)
			return nil
		}}
	}
	q.push(call(api.BrowserMouseOptions{Type: "move", X: 1}))
	q.push(call(api.BrowserMouseOptions{Type: "move", X: 2}))
	q.push(call(api.BrowserMouseOptions{Type: "press"}))
	q.push(call(api.BrowserMouseOptions{Type: "wheel", DY: 10}))
	q.push(call(api.BrowserMouseOptions{Type: "wheel", DY: 15}))
	q.push(browserCall{send: func(context.Context) error { sent = append(sent, "key"); return nil }})
	q.push(call(api.BrowserMouseOptions{Type: "move", X: 3}))

	if len(q.calls) != 5 || q.calls[0].mouse.X != 2 || q.calls[2].mouse.DY != 25 {
		t.Fatalf("queued %d calls: first move x=%v wheel dy=%v", len(q.calls), q.calls[0].mouse.X, q.calls[2].mouse.DY)
	}
	for c, ok := q.next(); ok; c, ok = q.next() {
		if err := c.send(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if want := []string{"move", "press", "wheel", "key", "move"}; !reflect.DeepEqual(sent, want) {
		t.Fatalf("sent %v, want %v", sent, want)
	}
}

func TestBrowserQueueKeepsOnlyTheLastPendingResize(t *testing.T) {
	q := newBrowserQueue()
	var sizes []int
	for w := 1; w <= 3; w++ {
		q.push(browserCall{resize: true, send: func(context.Context) error { sizes = append(sizes, w); return nil }})
	}
	for c, ok := q.next(); ok; c, ok = q.next() {
		_ = c.send(context.Background())
	}
	if !reflect.DeepEqual(sizes, []int{3}) {
		t.Fatalf("sent sizes %v, want the last one", sizes)
	}
}
