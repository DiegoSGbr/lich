// Which files of a pull request the reviewer has ticked off, so a long review
// keeps its place while the user jumps to the Overview tab and back. Keyed by
// the pull request (its URL, unique across projects) so two PRs never share
// ticks.
//
// In memory only — a restart clears every tick. That is also what keeps a tick
// honest: nothing here notices a force-push, so a persisted tick could outlive
// the diff it was made against. Persist (keyed on the head SHA) if a review
// ever needs to survive a restart.
//
// Module-level store + a subscribe/getSnapshot pair for useSyncExternalStore,
// matching the app's no-state-library pattern. Each snapshot is a frozen set
// that is replaced, never mutated, so useSyncExternalStore sees a stable
// reference until something actually changes.
const byPullRequest = new Map<string, ReadonlySet<string>>()
const listeners = new Set<() => void>()

const empty: ReadonlySet<string> = new Set()

/** The files ticked off in one pull request; a stable reference between changes. */
export function viewedFiles(pullRequest: string): ReadonlySet<string> {
  return byPullRequest.get(pullRequest) ?? empty
}

/** Tick a file off (or untick it). No-op — and no notification — if it already reads that way. */
export function setViewed(pullRequest: string, path: string, viewed: boolean): void {
  const current = viewedFiles(pullRequest)
  if (current.has(path) === viewed) {
    return
  }
  const next = new Set(current)
  if (viewed) {
    next.add(path)
  } else {
    next.delete(path)
  }
  byPullRequest.set(pullRequest, next)
  for (const listener of listeners) {
    listener()
  }
}

export function subscribeViewed(listener: () => void): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}
