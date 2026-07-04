import { useRef, useEffect, useCallback } from 'react'
import Editor, { useMonaco, type OnMount } from '@monaco-editor/react'
import type { editor, IDisposable } from 'monaco-editor'
import type { Breakpoint, DebugVariable, DebugState } from '../types'
import { detectLanguage } from '../utils/language'

interface EditorViewProps {
  path: string | null
  content: string | null
  loading: boolean
  error: string | null
  currentLine: number | null
  breakpoints: Breakpoint[]
  onToggleBreakpoint: (file: string, line: number) => void
  variables?: DebugVariable[]
  debugState: DebugState | null
}

export function EditorView({ path, content, loading, error, currentLine, breakpoints, onToggleBreakpoint, variables, debugState }: EditorViewProps) {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null)
  const bpDeco = useRef<string[]>([])
  const clDeco = useRef<string[]>([])
  const pathRef = useRef(path)
  const onToggleRef = useRef(onToggleBreakpoint)
  const variablesRef = useRef(variables)
  const monaco = useMonaco()

  pathRef.current = path
  onToggleRef.current = onToggleBreakpoint
  variablesRef.current = variables

  const toggle = useCallback((line: number) => {
    if (!pathRef.current) return

    console.log('[breakpoint] toggle', pathRef.current, line)
    onToggleRef.current(pathRef.current, line)
  }, [])

  const syncBreakpoints = useCallback(() => {
    const ed = editorRef.current

    if (!ed || !pathRef.current) return

    const model = ed.getModel()

    if (!model) return

    const fileBps = breakpoints.filter((bp) => bp.file === pathRef.current)

    const decorations = fileBps.map((bp) => ({
      range: {
        startLineNumber: bp.line,
        startColumn: 1,
        endLineNumber: bp.line,
        endColumn: 1,
      },
      options: {
        isWholeLine: true,
        className: 'breakpoint-line',
        glyphMarginClassName: 'breakpoint-glyph',
        glyphMarginHoverMessage: { value: 'Breakpoint' },
      } satisfies editor.IModelDecorationOptions,
    }))

    bpDeco.current = ed.deltaDecorations(bpDeco.current, decorations)
  }, [breakpoints])

  const syncCurrentLine = useCallback(() => {
    const ed = editorRef.current

    if (!ed) return

    if (currentLine === null || currentLine <= 0) {
      clDeco.current = ed.deltaDecorations(clDeco.current, [])
      return
    }

    const decoration = {
      range: {
        startLineNumber: currentLine,
        startColumn: 1,
        endLineNumber: currentLine,
        endColumn: 1,
      },
      options: {
        isWholeLine: true,
        className: 'current-line-highlight',
        glyphMarginClassName: 'current-line-arrow',
      } satisfies editor.IModelDecorationOptions,
    }

    clDeco.current = ed.deltaDecorations(clDeco.current, [decoration])
  }, [currentLine])

  const hoverDisposable = useRef<IDisposable | null>(null)

  const handleMount: OnMount = (editor) => {
    editorRef.current = editor

    editor.onMouseDown((e) => {
      if (e.target.type !== 2 && e.target.type !== 3) return

      const line = e.target.position?.lineNumber

      if (line) toggle(line)
    })

    if (monaco) {
      hoverDisposable.current = monaco.languages.registerHoverProvider('php', {
        provideHover(model, position) {
          const vars = variablesRef.current
          if (!vars || vars.length === 0) return null

          const word = model.getWordAtPosition(position)
          if (!word) return null

          const varName = word.word

          for (const v of vars) {
            if (v.name === varName || v.name === '$' + varName || v.name === varName.replace('$', '')) {
              return {
                contents: [
                  { value: `**${v.name}** = ${v.value} : ${v.type}${v.className ? ' (' + v.className + ')' : ''}` },
                ],
              }
            }
          }

          return null
        },
      })
    }

    syncBreakpoints()
    syncCurrentLine()
  }

  useEffect(() => {
    if (editorRef.current) syncBreakpoints()
  }, [syncBreakpoints])

  useEffect(() => {
    return () => {
      bpDeco.current = []
      clDeco.current = []

      if (hoverDisposable.current) {
        hoverDisposable.current.dispose()
        hoverDisposable.current = null
      }
    }
  }, [])

  useEffect(() => {
    if (!path) {
      bpDeco.current = []
      clDeco.current = []
      return
    }
  }, [path])

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

  const language = path ? detectLanguage(path) : 'plaintext'

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
          glyphMargin: !!debugState,
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
          codeLens: false,
        }}
      />
    </div>
  )
}
