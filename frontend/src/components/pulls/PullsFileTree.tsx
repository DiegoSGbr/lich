import {useState} from "react"
import {ChevronDown, ChevronRight, Folder, FolderOpen} from "lucide-react"
import {buildTree, type TreeNode} from "@/lib/file-tree"
import {FileIcon} from "@/lib/file-icon"
import {cn} from "@/lib/utils"
import type {DiffFile} from "@/lib/diff"

// PullsFileTree is the changed-files navigator for the Files changed tab: the
// PR's files as a collapsible tree — the same buildTree + FileIcon as the dock's
// Files panel — where clicking a file scrolls its diff into view. Structure
// only; the per-file +/- lives on each diff's header, the way GitHub shows it.
export function PullsFileTree({files, active, onSelect}: {files: DiffFile[]; active: string | null; onSelect: (path: string) => void}) {
  const tree = buildTree(files.map((file) => file.newPath))
  return (
    <div role="tree" className="py-1 font-mono text-xs">
      {tree.map((node) => (
        <TreeRow key={node.path} node={node} depth={0} active={active} onSelect={onSelect}/>
      ))}
    </div>
  )
}

interface TreeRowProps {
  node: TreeNode
  depth: number
  active: string | null
  onSelect: (path: string) => void
}

function TreeRow({node, depth, active, onSelect}: TreeRowProps) {
  const [open, setOpen] = useState(true)
  // The 0.5rem base keeps even top-level rows off the edge.
  const indent = {paddingLeft: `${depth * 0.75 + 0.5}rem`}
  if (node.type === "file") {
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
  const Chevron = open ? ChevronDown : ChevronRight
  const FolderIcon = open ? FolderOpen : Folder
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
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
          <TreeRow key={child.path} node={child} depth={depth + 1} active={active} onSelect={onSelect}/>
        ))}
    </>
  )
}
