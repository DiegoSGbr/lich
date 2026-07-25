import {useState} from "react"
import {ChevronDown, ChevronRight, Folder, FolderOpen} from "lucide-react"
import type {TreeNode} from "@/lib/file-tree"
import {FileIcon} from "@/lib/file-icon"
import {cn} from "@/lib/utils"

interface FileTreeProps {
  tree: TreeNode[]
  /** Path of the file to highlight; null for no selection. */
  active?: string | null
  /** Start folders expanded — a pull request's changed files, where every row
   * is part of the review. The repo browser starts collapsed instead. */
  defaultOpen?: boolean
  className?: string
  onSelect: (path: string) => void
}

// FileTree renders a buildTree() result as a collapsible folder/file tree. The
// dock's repo browser and the pull request's changed-files navigator are the
// same widget: they differ only in what a click does, whether folders start
// open, and whether a row reads as selected.
export function FileTree({
  tree,
  active = null,
  defaultOpen = false,
  className,
  onSelect,
}: FileTreeProps) {
  // The set holds the rows toggled *away* from defaultOpen, so a folder's state
  // survives its parent closing and reopening.
  const [toggled, setToggled] = useState<Set<string>>(new Set())
  const toggle = (path: string) =>
    setToggled((prev) => {
      const next = new Set(prev)
      if (!next.delete(path)) {
        next.add(path)
      }
      return next
    })

  return (
    <div role="tree" className={cn("py-1 font-mono text-xs", className)}>
      {tree.map((node) => (
        <TreeRow
          key={node.path}
          node={node}
          depth={0}
          active={active}
          isOpen={(path) => toggled.has(path) !== defaultOpen}
          onToggle={toggle}
          onSelect={onSelect}
        />
      ))}
    </div>
  )
}

interface TreeRowProps {
  node: TreeNode
  depth: number
  active: string | null
  isOpen: (path: string) => boolean
  onToggle: (path: string) => void
  onSelect: (path: string) => void
}

function TreeRow({node, depth, active, isOpen, onToggle, onSelect}: TreeRowProps) {
  // The 0.5rem base keeps even top-level rows off the edge.
  const indent = {paddingLeft: `${depth * 0.75 + 0.5}rem`}
  if (node.type === "file") {
    // A chevron-width spacer keeps file names aligned under their folder's name;
    // FileIcon draws the language's real logo (devicon).
    return (
      <button
        type="button"
        onClick={() => onSelect(node.path)}
        style={indent}
        title={node.path}
        className={cn(
          "flex w-full items-center gap-1.5 py-0.5 pr-2 text-left text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground",
          node.path === active && "bg-accent text-accent-foreground",
        )}
      >
        <span className="size-3.5 shrink-0" aria-hidden/>
        <FileIcon path={node.path}/>
        <span className="truncate">{node.name}</span>
      </button>
    )
  }
  const open = isOpen(node.path)
  const Chevron = open ? ChevronDown : ChevronRight
  const FolderIcon = open ? FolderOpen : Folder
  return (
    <>
      <button
        type="button"
        onClick={() => onToggle(node.path)}
        style={indent}
        aria-expanded={open}
        className="flex w-full items-center gap-1.5 py-0.5 pr-2 text-left font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
      >
        <Chevron className="size-3.5 shrink-0 text-muted-foreground"/>
        <FolderIcon className="size-3.5 shrink-0 text-muted-foreground"/>
        <span className="truncate">{node.name}</span>
      </button>
      {open &&
        node.children.map((child) => (
          <TreeRow
            key={child.path}
            node={child}
            depth={depth + 1}
            active={active}
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={onSelect}
          />
        ))}
    </>
  )
}
