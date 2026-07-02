import { useState, useEffect, useRef } from 'react'
import type { FileEntry } from '../types'

interface SearchOverlayProps {
  entries: FileEntry[]
  onSelect: (path: string) => void
  onClose: () => void
}

function flattenFiles(entries: FileEntry[]): FileEntry[] {
  const result: FileEntry[] = []

  for (const entry of entries) {
    if (entry.type === 'file') {
      result.push(entry)
    }

    if (entry.children) {
      result.push(...flattenFiles(entry.children))
    }
  }

  return result
}

export function SearchOverlay({ entries, onSelect, onClose }: SearchOverlayProps) {
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const allFiles = flattenFiles(entries)

  const filtered = query
    ? allFiles.filter((f) => f.name.toLowerCase().includes(query.toLowerCase()))
    : allFiles

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  useEffect(() => {
    setSelectedIndex(0)
  }, [query])

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIndex((prev) => Math.min(prev + 1, filtered.length - 1))
      }

      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIndex((prev) => Math.max(prev - 1, 0))
      }

      if (e.key === 'Enter' && filtered[selectedIndex]) {
        e.preventDefault()
        onSelect(filtered[selectedIndex].path)
        onClose()
      }

      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }

    window.addEventListener('keydown', handleKeyDown)

    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [filtered, selectedIndex, onSelect, onClose])

  return (
    <div className="search-overlay-backdrop" onClick={onClose}>
      <div className="search-overlay" onClick={(e) => e.stopPropagation()}>
        <input
          ref={inputRef}
          className="search-input"
          type="text"
          placeholder="Search files by name..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <div className="search-results">
          {filtered.length === 0 && (
            <div className="search-empty">No files found</div>
          )}
          {filtered.slice(0, 50).map((file, idx) => (
            <div
              key={file.path}
              className={`search-result-item ${idx === selectedIndex ? 'selected' : ''}`}
              onClick={() => {
                onSelect(file.path)
                onClose()
              }}
            >
              <span className="search-result-name">{file.name}</span>
              <span className="search-result-path">{file.path}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
