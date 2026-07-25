import {describe, expect, it, vi} from "vitest"
import {setViewed, subscribeViewed, viewedFiles} from "./pull-request-viewed"

// The store is module-level, so every test uses its own pull request URL.
const pr = (name: string) => `https://github.com/o/l/pull/${name}`

describe("pull-request-viewed", () => {
  it("starts with nothing ticked", () => {
    expect(viewedFiles(pr("empty")).size).toBe(0)
  })

  it("ticks a file off and back on", () => {
    const url = pr("toggle")
    setViewed(url, "a.ts", true)
    expect(viewedFiles(url).has("a.ts")).toBe(true)
    setViewed(url, "a.ts", false)
    expect(viewedFiles(url).has("a.ts")).toBe(false)
  })

  it("keeps pull requests apart", () => {
    setViewed(pr("one"), "a.ts", true)
    expect(viewedFiles(pr("two")).has("a.ts")).toBe(false)
  })

  // useSyncExternalStore re-renders forever if getSnapshot returns a fresh
  // object every call, so the same set must come back until something changes.
  it("returns a stable snapshot between changes", () => {
    const url = pr("stable")
    setViewed(url, "a.ts", true)
    const first = viewedFiles(url)
    expect(viewedFiles(url)).toBe(first)
    setViewed(url, "b.ts", true)
    expect(viewedFiles(url)).not.toBe(first)
    // The old snapshot is never mutated behind a holder's back.
    expect(first.has("b.ts")).toBe(false)
  })

  it("notifies subscribers, and only on a real change", () => {
    const url = pr("notify")
    const listener = vi.fn()
    const off = subscribeViewed(listener)
    setViewed(url, "a.ts", true)
    setViewed(url, "a.ts", true) // already ticked
    setViewed(url, "b.ts", false) // never ticked
    expect(listener).toHaveBeenCalledTimes(1)
    off()
    setViewed(url, "c.ts", true)
    expect(listener).toHaveBeenCalledTimes(1)
  })
})
