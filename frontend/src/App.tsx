import { useState, useMemo, useCallback, useEffect, useRef } from 'react'
import { FileExplorer } from './components/FileExplorer'
import { EditorView } from './components/EditorView'
import { DebugPanel } from './components/DebugPanel'
import { StatusBar } from './components/StatusBar'
import { SearchOverlay } from './components/SearchOverlay'
import { useFileTree } from './hooks/useFileTree'
import { useFileContent } from './hooks/useFileContent'
import { useDebugger } from './hooks/useDebugger'
import type { Breakpoint } from './types'

export default function App() {
  const { tree, loading, error } = useFileTree()
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const { content, loading: fileLoading, error: fileError } = useFileContent(selectedPath)
  const [searchOpen, setSearchOpen] = useState(false)
  const { state: debugState, connected, status, send } = useDebugger()
  const prevStatusRef = useRef(debugState?.status)

  const [localBreakpoints, setLocalBreakpoints] = useState<Breakpoint[]>([])

  const breakpoints = useMemo(() => {
    const serverBps = debugState?.breakpoints ?? []
    const serverKeys = new Set(serverBps.map((bp) => `${bp.file}:${bp.line}`))
    const localOnly = localBreakpoints.filter(
      (bp) => !serverKeys.has(`${bp.file}:${bp.line}`)
    )

    return [...serverBps, ...localOnly]
  }, [debugState?.breakpoints, localBreakpoints])

  const handleSelectFile = useCallback((path: string) => {
    setSelectedPath(path)
  }, [])

  const handleToggleBreakpoint = useCallback((file: string, line: number) => {
    setLocalBreakpoints((prev) => {
      const existing = prev.find((bp) => bp.file === file && bp.line === line)

      if (existing) {
        send('breakpoint_remove', { id: existing.id })

        return prev.filter((bp) => bp.id !== existing.id)
      }

      const bp: Breakpoint = { id: `local-${file}-${line}`, file, line }

      send('breakpoint_set', { file, line: String(line) })

      return [...prev, bp]
    })
  }, [send])

  useEffect(() => {
    const prev = prevStatusRef.current
    const cur = debugState?.status

    prevStatusRef.current = cur

    if (cur === 'break' && prev !== 'break' && debugState?.currentFile) {
      setSelectedPath(debugState.currentFile)
    }
  }, [debugState?.status, debugState?.currentFile])

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'p') {
        e.preventDefault()
        setSearchOpen(true)
      }
    }

    window.addEventListener('keydown', handleKeyDown)

    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  return (
    <div className="app">
      <div className="top-bar">
        <span className="app-title">Xdebug-Web</span>
      </div>

      <div className="main-area">
        <aside className="sidebar">
          {loading && <div className="sidebar-message">Loading files...</div>}
          {error && <div className="sidebar-message sidebar-error">{error}</div>}
          {tree && (
            <FileExplorer
              entries={tree.entries}
              selectedPath={selectedPath}
              onSelect={handleSelectFile}
            />
          )}
        </aside>

        <main className="content">
          <EditorView
            path={selectedPath}
            content={content}
            loading={fileLoading}
            error={fileError}
            currentLine={debugState?.currentLine ?? null}
            breakpoints={breakpoints}
            onToggleBreakpoint={handleToggleBreakpoint}
          />
        </main>
      </div>

      <DebugPanel state={debugState} onSend={send} />

      <StatusBar
        connected={connected}
        debugStatus={status}
      />

      {searchOpen && tree && (
        <SearchOverlay
          entries={tree.entries}
          onSelect={handleSelectFile}
          onClose={() => setSearchOpen(false)}
        />
      )}
    </div>
  )
}
