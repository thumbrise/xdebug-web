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
  const { state: debugState, connected, status, send, propertyData } = useDebugger()
  const prevStatusRef = useRef(debugState?.status)
  const [panelHeight, setPanelHeight] = useState(180)
  const dragRef = useRef(false)
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

  const handleDragStart = useCallback(() => {
    dragRef.current = true

    const onMouseMove = (e: MouseEvent) => {
      if (!dragRef.current) return
      const newHeight = window.innerHeight - e.clientY - 24

      if (newHeight >= 80 && newHeight <= 600) {
        setPanelHeight(newHeight)
      }
    }

    const onMouseUp = () => {
      dragRef.current = false
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }

    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
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
            currentLine={debugState?.status === 'break' ? (debugState?.currentLine ?? null) : null}
            breakpoints={breakpoints}
            onToggleBreakpoint={handleToggleBreakpoint}
            variables={debugState?.status === 'break' ? [...(debugState.locals ?? []), ...(debugState.globals ?? [])] : undefined}
            debugState={debugState}
          />
        </main>
      </div>

      <div className="resize-handle" onMouseDown={handleDragStart} />

      <div className="debug-panel-wrapper" style={{ height: panelHeight }}>
        <DebugPanel state={debugState} onSend={send} propertyData={propertyData} />
      </div>

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
