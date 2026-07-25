import {useState, type ReactNode} from "react"
import {toast} from "sonner"
import {
  Check,
  ChevronDown,
  CircleDashed,
  Clock,
  ExternalLink,
  GitBranch,
  GitMerge,
  GitPullRequestArrow,
  X,
  type LucideIcon,
} from "lucide-react"
import {ProjectService, System} from "@/lib/rpc"
import type {ChecksRollup, PullRequestDetail} from "@/lib/api-types"
import {useActiveSession} from "@/lib/useActiveSession"
import {useGitStatus} from "@/lib/useGitStatus"
import {usePullRequestDetail} from "@/lib/usePullRequestDetail"
import {cn, errorText} from "@/lib/utils"
import {Markdown} from "@/components/Markdown"
import {Button} from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {Input} from "@/components/ui/input"
import {Label} from "@/components/ui/label"

// PullsPanel is the Pulls tab of the right dock: the active session's branch and
// its open GitHub pull request, with merge and open-on-GitHub actions. It
// follows the active session like Files and Review — a worktree session shows
// its checkout's PR — and reuses the gh CLI already behind the footer badge. The
// dock (RightDock) owns the surrounding chrome; this stays a pure body.
export function PullsPanel() {
  const {path} = useActiveSession()
  const status = useGitStatus(path)
  const branch = status?.branch ?? ""
  const {detail, loading, error, refresh} = usePullRequestDetail(path, branch)

  if (!path) {
    return <Notice>No repository</Notice>
  }
  if (error) {
    return <Notice>Couldn’t load the pull request: {error}</Notice>
  }
  if (detail) {
    return <PullRequestView path={path} detail={detail} onMerged={refresh}/>
  }
  if (loading) {
    return <Notice>Loading…</Notice>
  }
  return <EmptyState path={path} branch={branch} onOpened={refresh}/>
}

// EditState carries a pending "edit commit message" merge: which method to run
// and the message the dialog is editing. null when the dialog is closed.
interface EditState {
  method: string
  title: string
  subject: string
  body: string
}

interface PullRequestViewProps {
  path: string
  detail: PullRequestDetail
  onMerged: () => void
}

function PullRequestView({path, detail, onMerged}: PullRequestViewProps) {
  const [merging, setMerging] = useState(false)
  const [edit, setEdit] = useState<EditState | null>(null)
  const blocked = detail.isDraft
    ? "Pull request is a draft"
    : detail.mergeable === "CONFLICTING"
      ? `Conflicts with ${detail.baseRefName}`
      : null

  const merge = async (method: string, subject = "", body = "") => {
    setMerging(true)
    try {
      await ProjectService.MergePullRequest(path, method, subject, body)
      toast.success(`Merged #${detail.number} into ${detail.baseRefName}`)
      setEdit(null)
      onMerged()
    } catch (err: unknown) {
      toast.error(`Merge failed: ${errorText(err)}`)
    } finally {
      setMerging(false)
    }
  }

  // Prefill the message the way GitHub does: a squash takes the PR title/body, a
  // merge commit its own "Merge pull request …" subject.
  const openEdit = (method: string) => {
    setEdit(
      method === "squash"
        ? {
            method,
            title: "Squash and merge",
            subject: `${detail.title} (#${detail.number})`,
            body: detail.body,
          }
        : {
            method,
            title: "Create a merge commit",
            subject: `Merge pull request #${detail.number} from ${detail.headRefName}`,
            body: "",
          },
    )
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button size="sm" disabled={merging || blocked !== null}>
                <GitMerge/>
                {merging ? "Merging…" : "Merge"}
                <ChevronDown/>
              </Button>
            }
          />
          <DropdownMenuContent align="start">
            <DropdownMenuItem onClick={() => void merge("squash")}>
              Squash and merge
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => void merge("merge")}>
              Create a merge commit
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => void merge("rebase")}>
              Rebase and merge
            </DropdownMenuItem>
            <DropdownMenuSeparator/>
            <DropdownMenuItem onClick={() => openEdit("squash")}>
              Squash and merge, edit message…
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => openEdit("merge")}>
              Create a merge commit, edit message…
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => void System.OpenExternal(detail.url)}
        >
          <ExternalLink/>
          Open
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="flex flex-col gap-2.5 p-3">
          <div className="text-sm font-medium leading-snug">
            <span className="text-muted-foreground">#{detail.number}</span> {detail.title}
          </div>

          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
            <span
              className={cn(
                "flex items-center gap-1 font-medium",
                detail.isDraft ? "text-amber-500" : "text-emerald-500",
              )}
            >
              <GitPullRequestArrow className="size-3.5"/>
              {detail.isDraft ? "Draft" : "Open"}
            </span>
            <span className="flex items-center gap-1 truncate font-mono text-muted-foreground">
              <GitBranch className="size-3.5 shrink-0"/>
              {detail.headRefName} → {detail.baseRefName}
            </span>
          </div>

          <div className="flex flex-col gap-1.5 pt-0.5">
            <ChecksRow checks={detail.checks}/>
            <MergeableRow mergeable={detail.mergeable} base={detail.baseRefName}/>
          </div>
        </div>

        {detail.body.trim() !== "" && (
          <div className="border-t border-border px-3 py-3">
            <Markdown>{detail.body}</Markdown>
          </div>
        )}
      </div>

      {edit && (
        <MergeMessageDialog
          edit={edit}
          merging={merging}
          onChange={setEdit}
          onCancel={() => setEdit(null)}
          onConfirm={() => void merge(edit.method, edit.subject, edit.body)}
        />
      )}
    </div>
  )
}

interface MergeMessageDialogProps {
  edit: EditState
  merging: boolean
  onChange: (next: EditState) => void
  onCancel: () => void
  onConfirm: () => void
}

function MergeMessageDialog({edit, merging, onChange, onCancel, onConfirm}: MergeMessageDialogProps) {
  return (
    <Dialog open onOpenChange={(next) => !next && onCancel()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{edit.title}</DialogTitle>
          <DialogDescription>Edit the commit message, then merge.</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="merge-subject">Commit message</Label>
            <Input
              id="merge-subject"
              value={edit.subject}
              onChange={(e) => onChange({...edit, subject: e.target.value})}
              autoFocus
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="merge-body">Extended description</Label>
            <textarea
              id="merge-body"
              value={edit.body}
              onChange={(e) => onChange({...edit, body: e.target.value})}
              rows={6}
              placeholder="Optional"
              className="min-h-24 w-full resize-y rounded-md border border-input bg-transparent px-2.5 py-1.5 text-sm shadow-xs outline-none transition-[color,box-shadow] placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
          <Button onClick={onConfirm} disabled={merging || edit.subject.trim() === ""}>
            <GitMerge/>
            {merging ? "Merging…" : edit.method === "squash" ? "Squash and merge" : "Merge"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type Tone = "pass" | "fail" | "pending" | "muted"

const toneClass: Record<Tone, string> = {
  pass: "text-emerald-500",
  fail: "text-destructive",
  pending: "text-amber-500",
  muted: "text-muted-foreground",
}

function StatusRow({icon: Icon, tone, children}: {icon: LucideIcon; tone: Tone; children: ReactNode}) {
  return (
    <span className={cn("flex items-center gap-1.5 text-xs", toneClass[tone])}>
      <Icon className="size-3.5 shrink-0"/>
      {children}
    </span>
  )
}

function ChecksRow({checks}: {checks: ChecksRollup}) {
  const {passed, failed, pending, total} = checks
  if (total === 0) {
    return <StatusRow icon={CircleDashed} tone="muted">No checks</StatusRow>
  }
  if (failed > 0) {
    return (
      <StatusRow icon={X} tone="fail">
        {failed} of {total} checks failing
      </StatusRow>
    )
  }
  if (pending > 0) {
    return (
      <StatusRow icon={Clock} tone="pending">
        {pending} of {total} checks running
      </StatusRow>
    )
  }
  return (
    <StatusRow icon={Check} tone="pass">
      {passed === 1 ? "1 check passed" : `${passed} checks passed`}
    </StatusRow>
  )
}

function MergeableRow({mergeable, base}: {mergeable: string; base: string}) {
  if (mergeable === "CONFLICTING") {
    return (
      <StatusRow icon={X} tone="fail">
        Conflicts with {base}
      </StatusRow>
    )
  }
  if (mergeable === "MERGEABLE") {
    return (
      <StatusRow icon={GitMerge} tone="pass">
        Mergeable — no conflicts with {base}
      </StatusRow>
    )
  }
  return <StatusRow icon={CircleDashed} tone="muted">Checking if mergeable…</StatusRow>
}

interface EmptyStateProps {
  path: string
  branch: string
  onOpened: () => void
}

function EmptyState({path, branch, onOpened}: EmptyStateProps) {
  const [opening, setOpening] = useState(false)
  const openPR = async () => {
    setOpening(true)
    try {
      await ProjectService.CreatePullRequest(path)
      // gh pushed the branch and opened the browser; refetch on focus-return
      // catches the new PR, but nudge once now for a quick creation.
      onOpened()
    } catch (err: unknown) {
      toast.error(`Couldn’t open a pull request: ${errorText(err)}`)
    } finally {
      setOpening(false)
    }
  }
  return (
    <div className="flex flex-col items-start gap-3 px-3 py-4">
      <p className="text-xs text-muted-foreground">
        {branch ? (
          <>
            No open pull request for{" "}
            <span className="font-medium text-foreground">{branch}</span>.
          </>
        ) : (
          "No open pull request."
        )}
      </p>
      <Button
        variant="outline"
        size="sm"
        onClick={() => void openPR()}
        disabled={opening}
      >
        <GitPullRequestArrow/>
        {opening ? "Opening…" : "Open pull request"}
        <ExternalLink/>
      </Button>
    </div>
  )
}

function Notice({children}: {children: ReactNode}) {
  return <p className="px-3 py-4 text-xs text-muted-foreground">{children}</p>
}
