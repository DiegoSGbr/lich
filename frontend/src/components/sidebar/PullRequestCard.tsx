import {GitPullRequestArrow} from "lucide-react"
import {cn} from "@/lib/utils"
import {useGitStatus} from "@/lib/useGitStatus"
import {usePullRequest} from "@/lib/usePullRequest"

interface PullRequestCardProps {
  // The worktree checkout whose branch PR this entry opens.
  path: string
  active: boolean
  onSelect: () => void
}

// PullRequestCard is a worktree group's pull-request entry — a peer of its
// session cards — opening the full-screen Pulls view for the worktree's branch.
// It shows the open PR's number when there is one, and otherwise reads as the
// door to open one (the create flow lives on the screen's empty state).
export function PullRequestCard({path, active, onSelect}: PullRequestCardProps) {
  const git = useGitStatus(path)
  const pr = usePullRequest(path, git?.branch ?? "")
  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "group relative flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-sm font-medium text-foreground transition-colors hover:bg-accent/60",
        active && "bg-accent text-accent-foreground",
      )}
    >
      <GitPullRequestArrow className="size-4 shrink-0 text-muted-foreground"/>
      <span>Pull request</span>
      {pr && (
        <span className="ml-auto flex items-center gap-1.5 text-xs">
          <span className="text-muted-foreground">#{pr.number}</span>
          <span className="text-emerald-500">Open</span>
        </span>
      )}
    </button>
  )
}
