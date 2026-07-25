// Tracks which worktree checkouts have their Pull request card parked in the
// sidebar. Like the Settings card, opening the PR view parks a card that stays
// in the worktree's group (inactive) while the user works, so it can be clicked
// back to; the X removes it. Keyed by the worktree path — each worktree parks
// its own. A pure UI concern, never persisted: the screen itself is route-driven.
//
// Module-level store + a subscribe/getSnapshot pair for useSyncExternalStore,
// matching the app's no-state-library pattern.
const openPaths = new Set<string>()
const listeners = new Set<() => void>()

function emit(): void {
  for (const listener of listeners) {
    listener()
  }
}

/** Park a worktree's Pull request card. No-op if the path is empty or already parked. */
export function openPulls(path: string): void {
  if (!path || openPaths.has(path)) {
    return
  }
  openPaths.add(path)
  emit()
}

/** Remove a worktree's Pull request card. No-op if it was not parked. */
export function closePulls(path: string): void {
  if (!openPaths.delete(path)) {
    return
  }
  emit()
}

export function isPullsOpen(path: string): boolean {
  return openPaths.has(path)
}

export function subscribePullsCard(listener: () => void): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}
