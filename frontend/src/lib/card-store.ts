// A parked-card tracker: the sidebar keeps a card (Settings, Pull request) in
// the session list while the user works in a terminal, so it can be clicked
// back to. Pure UI state, never persisted — a restart starts with no cards
// parked; the screens themselves are route-driven.
//
// Module-level set + a subscribe/getSnapshot pair for useSyncExternalStore,
// matching the app's no-state-library pattern. Each createCardStore() call owns
// its own set, so one store's keys never leak into another's.

export interface CardStore {
  /** Park a card. No-op (and no notification) if the key is empty or already parked. */
  open(key: string): void
  /** Remove a card. No-op (and no notification) if it was not parked. */
  close(key: string): void
  isOpen(key: string): boolean
  subscribe(listener: () => void): () => void
}

export function createCardStore(): CardStore {
  const openKeys = new Set<string>()
  const listeners = new Set<() => void>()

  const emit = (): void => {
    for (const listener of listeners) {
      listener()
    }
  }

  return {
    open(key) {
      if (!key || openKeys.has(key)) {
        return
      }
      openKeys.add(key)
      emit()
    },
    close(key) {
      if (!openKeys.delete(key)) {
        return
      }
      emit()
    },
    isOpen: (key) => openKeys.has(key),
    subscribe(listener) {
      listeners.add(listener)
      return () => {
        listeners.delete(listener)
      }
    },
  }
}
