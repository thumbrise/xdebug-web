import { useRef, useEffect } from 'react'
import Editor, { type OnMount } from '@monaco-editor/react'
import type { editor } from 'monaco-editor'

interface EditorViewProps {
  path: string | null
  content: string | null
  loading: boolean
  error: string | null
  currentLine: number | null
}

export function EditorView({ path, content, loading, error, currentLine }: EditorViewProps) {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null)

  const handleMount: OnMount = (editor) => {
    editorRef.current = editor
  }

  useEffect(() => {
    if (editorRef.current && currentLine !== null) {
      editorRef.current.revealLineInCenter(currentLine)
      editorRef.current.setPosition({ lineNumber: currentLine, column: 1 })
      editorRef.current.focus()
    }
  }, [currentLine])

  const filename = path ? path.split('/').pop() || path : ''

  const language = filename.endsWith('.php')
    ? 'php'
    : filename.endsWith('.js') || filename.endsWith('.mjs')
      ? 'javascript'
      : filename.endsWith('.ts') || filename.endsWith('.tsx')
        ? 'typescript'
        : filename.endsWith('.css')
          ? 'css'
          : filename.endsWith('.html')
            ? 'html'
            : filename.endsWith('.json')
              ? 'json'
              : filename.endsWith('.md')
                ? 'markdown'
                : filename.endsWith('.go')
                  ? 'go'
                  : filename.endsWith('.yaml') || filename.endsWith('.yml')
                    ? 'yaml'
                    : filename.endsWith('.sql')
                      ? 'sql'
                      : filename.endsWith('.xml')
                        ? 'xml'
                        : 'plaintext'

  if (!path) {
    return (
      <div className="editor-view">
        <div className="editor-placeholder">Select a file to view</div>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="editor-view">
        <div className="editor-placeholder">Loading...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="editor-view">
        <div className="editor-placeholder editor-error">{error}</div>
      </div>
    )
  }

  return (
    <div className="editor-view">
      <div className="editor-header">{path}</div>
      <Editor
        key={path}
        height="100%"
        language={language}
        value={content || ''}
        theme="vs-dark"
        onMount={handleMount}
        options={{
          readOnly: true,
          minimap: { enabled: false },
          fontSize: 13,
          lineNumbers: 'on',
          scrollBeyondLastLine: false,
          automaticLayout: true,
          wordWrap: 'on',
          tabSize: 4,
          renderWhitespace: 'selection',
          glyphMargin: false,
          folding: true,
          lineDecorationsWidth: 8,
          lineNumbersMinChars: 3,
          overviewRulerLanes: 0,
          hideCursorInOverviewRuler: true,
          overviewRulerBorder: false,
          scrollbar: {
            verticalScrollbarSize: 10,
            horizontalScrollbarSize: 10,
          },
        }}
      />
    </div>
  )
}
