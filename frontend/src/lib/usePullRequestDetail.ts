import {useCallback, useEffect, useRef, useState} from "react"
import {ProjectService} from "./rpc"
import type {PullRequestDetail} from "./api-types"
import {errorText} from "./utils"

export type {PullRequestDetail}

export interface PullRequestState {
  detail: PullRequestDetail | null
  loading: boolean
  error: string | null
  refresh: () => void
}

// usePullRequestDetail resolves the active branch's open PR in full for the
// Pulls dock tab. Like the footer badge it is not polled — a PR changes rarely
// and each lookup is a gh network round-trip — but it refetches on window focus
// (so opening or merging in the browser reflects on return) and exposes
// refresh() so an in-app merge updates the panel at once. detail is null with no
// error when the branch simply has no open PR: the panel's empty state.
export function usePullRequestDetail(path: string, branch: string): PullRequestState {
  const [detail, setDetail] = useState<PullRequestDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const seq = useRef(0)

  const refresh = useCallback(() => {
    if (!path) {
      setDetail(null)
      setError(null)
      setLoading(false)
      return
    }
    const mine = ++seq.current
    setLoading(true)
    ProjectService.PullRequestDetail(path)
      .then((result) => {
        if (mine !== seq.current) return
        setDetail(result)
        setError(null)
      })
      .catch((err: unknown) => {
        if (mine !== seq.current) return
        setDetail(null)
        setError(errorText(err))
      })
      .finally(() => {
        if (mine === seq.current) setLoading(false)
      })
  }, [path])

  useEffect(() => {
    refresh()
    window.addEventListener("focus", refresh)
    return () => {
      // Invalidate any in-flight lookup so a late response can't land on the
      // next branch/path or after unmount.
      seq.current++
      window.removeEventListener("focus", refresh)
    }
  }, [refresh, branch])

  return {detail, loading, error, refresh}
}
