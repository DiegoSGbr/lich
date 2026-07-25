import {useMemo, useRef, useState, type ReactNode} from "react"
import {usePullRequestDiff} from "@/lib/usePullRequestDiff"
import {buildTree} from "@/lib/file-tree"
import {FileDiff} from "@/components/diff/FileDiff"
import {FileTree} from "@/components/FileTree"
import {DiffStat} from "@/components/DiffStat"

// PullsFiles is the "Files changed" tab of the Pulls screen: a changed-files
// tree on the left (click to jump) beside the PR's diff, rendered with the same
// FileDiff cards as the Review dock — read-only, no discard. Inject still works,
// so a PR file can be referenced into the session's terminal.
export function PullsFiles({path, onInject}: {path: string; onInject: (text: string) => void}) {
  const {files, error} = usePullRequestDiff(path)
  const rows = useRef<Map<string, HTMLElement>>(new Map())
  const [active, setActive] = useState<string | null>(null)
  // Structure only; the per-file +/- lives on each diff's header, the way
  // GitHub shows it.
  const tree = useMemo(() => buildTree((files ?? []).map((file) => file.newPath)), [files])

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

  const jumpTo = (target: string) => {
    setActive(target)
    rows.current.get(target)?.scrollIntoView({block: "start", behavior: "smooth"})
  }

  return (
    <div className="flex h-full">
      <div className="w-60 shrink-0 overflow-y-auto border-r border-border">
        <FileTree tree={tree} active={active} defaultOpen onSelect={jumpTo}/>
      </div>
      <div className="flex flex-1 flex-col overflow-y-auto">
        <div className="flex items-center justify-end border-b border-border px-3 py-2.5 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <DiffStat added={added} deleted={deleted}/>
          </span>
        </div>
        <div className="flex flex-col p-3 [&>div:not(:first-child)]:mt-2.5 [&>div:not(:first-child)]:border-t [&>div:not(:first-child)]:border-border [&>div:not(:first-child)]:pt-2.5">
          {files.map((file) => (
            <div
              key={file.newPath}
              ref={(el) => {
                if (el) {
                  rows.current.set(file.newPath, el)
                } else {
                  rows.current.delete(file.newPath)
                }
              }}
            >
              <FileDiff file={file} onInject={onInject}/>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function Notice({children}: {children: ReactNode}) {
  return <p className="px-4 py-6 text-sm text-muted-foreground">{children}</p>
}
