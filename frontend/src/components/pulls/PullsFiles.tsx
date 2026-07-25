import type {ReactNode} from "react"
import {usePullRequestDiff} from "@/lib/usePullRequestDiff"
import {FileDiff} from "@/components/diff/FileDiff"
import {DiffStat} from "@/components/DiffStat"

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

  const added = files.reduce((sum, file) => sum + file.added, 0)
  const deleted = files.reduce((sum, file) => sum + file.deleted, 0)

  return (
    <div>
      <div className="flex items-center justify-end gap-3 border-b border-border px-3 py-2.5 text-xs text-muted-foreground">
        <span>
          <span className="font-medium text-foreground">{files.length}</span>{" "}
          {files.length === 1 ? "file changed" : "files changed"}
        </span>
        <span className="flex items-center gap-1.5">
          <DiffStat added={added} deleted={deleted}/>
        </span>
      </div>
      <div className="flex flex-col p-3 [&>section:not(:first-child)]:mt-2.5 [&>section:not(:first-child)]:border-t [&>section:not(:first-child)]:border-border [&>section:not(:first-child)]:pt-2.5">
        {files.map((file) => (
          <FileDiff key={file.newPath} file={file} onInject={onInject}/>
        ))}
      </div>
    </div>
  )
}

function Notice({children}: {children: ReactNode}) {
  return <p className="px-4 py-6 text-sm text-muted-foreground">{children}</p>
}
