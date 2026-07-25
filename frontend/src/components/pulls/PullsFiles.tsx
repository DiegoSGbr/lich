import type {ReactNode} from "react"
import {usePullRequestDiff} from "@/lib/usePullRequestDiff"
import {FileDiff} from "@/components/diff/FileDiff"

// PullsFiles is the "Files changed" tab of the Pulls screen: the PR's unified
// diff (gh pr diff) rendered with the same FileDiff cards as the Review dock,
// read-only — no discard (these changes are committed and remote). Inject still
// works, so a PR file can be referenced into the session's terminal.
export function PullsFiles({path, onInject}: {path: string; onInject: (text: string) => void}) {
  const {files, error} = usePullRequestDiff(path)

  if (error) {
    return <Notice>Couldn’t load the diff: {error}</Notice>
  }
  if (files === null) {
    return <Notice>Loading…</Notice>
  }
  if (files.length === 0) {
    return <Notice>No file changes</Notice>
  }
  return (
    <div className="flex flex-col p-3 [&>section:not(:first-child)]:mt-2.5 [&>section:not(:first-child)]:border-t [&>section:not(:first-child)]:border-border [&>section:not(:first-child)]:pt-2.5">
      {files.map((file) => (
        <FileDiff key={file.newPath} file={file} onInject={onInject}/>
      ))}
    </div>
  )
}

function Notice({children}: {children: ReactNode}) {
  return <p className="px-4 py-6 text-sm text-muted-foreground">{children}</p>
}
