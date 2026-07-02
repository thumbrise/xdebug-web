import { useState } from 'react'
import type { FileEntry } from '../types'

interface FileExplorerProps {
  entries: FileEntry[]
  selectedPath: string | null
  onSelect: (path: string) => void
}

interface TreeNodeProps {
  entry: FileEntry
  selectedPath: string | null
  onSelect: (path: string) => void
  depth: number
}

function TreeNode({ entry, selectedPath, onSelect, depth }: TreeNodeProps) {
  const [expanded, setExpanded] = useState(depth < 2)

  if (entry.type === 'directory') {
    return (
      <div>
        <div
          className={`file-tree-item file-tree-dir ${selectedPath === entry.path ? 'selected' : ''}`}
          style={{ paddingLeft: `${8 + depth * 16}px` }}
          onClick={() => {
            setExpanded(!expanded)
            onSelect(entry.path)
          }}
        >
          <span className="file-tree-arrow">{expanded ? '▼' : '▶'}</span>
          <span className="file-tree-icon">📁</span>
          <span className="file-tree-name">{entry.name}</span>
        </div>
        {expanded && entry.children?.map((child) => (
          <TreeNode
            key={child.path}
            entry={child}
            selectedPath={selectedPath}
            onSelect={onSelect}
            depth={depth + 1}
          />
        ))}
      </div>
    )
  }

  return (
    <div
      className={`file-tree-item file-tree-file ${selectedPath === entry.path ? 'selected' : ''}`}
      style={{ paddingLeft: `${8 + depth * 16}px` }}
      onClick={() => onSelect(entry.path)}
    >
      <span className="file-tree-arrow" />
      <span className="file-tree-icon">
        {entry.name.endsWith('.php') ? '🐘' : '📄'}
      </span>
      <span className="file-tree-name">{entry.name}</span>
    </div>
  )
}

export function FileExplorer({ entries, selectedPath, onSelect }: FileExplorerProps) {
  return (
    <div className="file-explorer">
      <div className="file-explorer-header">Files</div>
      <div className="file-tree">
        {entries.map((entry) => (
          <TreeNode
            key={entry.path}
            entry={entry}
            selectedPath={selectedPath}
            onSelect={onSelect}
            depth={0}
          />
        ))}
      </div>
    </div>
  )
}
