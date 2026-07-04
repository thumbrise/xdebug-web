import { useEffect, useRef, useState } from 'react'
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
  expandedPaths: Set<string>
  onToggle: (path: string) => void
  depth: number
}

function getAncestors(path: string): string[] {
  const parts = path.split('/')
  const ancestors: string[] = []

  for (let i = 0; i < parts.length - 1; i++) {
    if (i === 0) {
      ancestors.push(parts[0])
    } else {
      ancestors.push(ancestors[i - 1] + '/' + parts[i])
    }
  }

  return ancestors
}

function TreeNode({ entry, selectedPath, onSelect, expandedPaths, onToggle, depth }: TreeNodeProps) {
  const expanded = expandedPaths.has(entry.path)

  if (entry.type === 'directory') {
    return (
      <div>
        <div
          className={`file-tree-item file-tree-dir ${selectedPath === entry.path ? 'selected' : ''}`}
          style={{ paddingLeft: `${8 + depth * 16}px` }}
          onClick={() => onToggle(entry.path)}
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
            expandedPaths={expandedPaths}
            onToggle={onToggle}
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
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(() => {
    const init = new Set<string>()

    for (const entry of entries) {
      if (entry.type === 'directory') {
        init.add(entry.path)
      }
    }

    return init
  })

  const prevSelected = useRef(selectedPath)

  useEffect(() => {
    if (!selectedPath || selectedPath === prevSelected.current) {
      return
    }

    prevSelected.current = selectedPath

    setExpandedPaths((prev) => {
      const next = new Set(prev)
      let changed = false

      for (const ancestor of getAncestors(selectedPath)) {
        if (!next.has(ancestor)) {
          next.add(ancestor)
          changed = true
        }
      }

      return changed ? next : prev
    })
  }, [selectedPath])

  function toggleExpand(path: string) {
    setExpandedPaths((prev) => {
      const next = new Set(prev)

      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }

      return next
    })
  }

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
            expandedPaths={expandedPaths}
            onToggle={toggleExpand}
            depth={0}
          />
        ))}
      </div>
    </div>
  )
}
