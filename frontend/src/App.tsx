import { useState, useCallback, useEffect } from 'react'
import { FileExplorer } from './components/FileExplorer'
import { EditorView } from './components/EditorView'
import { DebugPanel } from './components/DebugPanel'
import { StatusBar } from './components/StatusBar'
import { SearchOverlay } from './components/SearchOverlay'
import { useFileTree } from './hooks/useFileTree'
import { useFileContent } from './hooks/useFileContent'
import { useDebugger } from './hooks/useDebugger'

export default function App() {
  const { tree, loading, error } = useFileTree()
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const { content, loading: fileLoading, error: fileError } = useFileContent(selectedPath)
  const [searchOpen, setSearchOpen] = useState(false)
  const { state: debugState, connected, status, send } = useDebugger()

  const handleSelectFile = useCallback((path: string) => {
    setSelectedPath(path)
  }, [])

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
