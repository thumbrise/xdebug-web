import type { DebugState } from '../types'

interface DebugPanelProps {
  state: DebugState | null
}

function StackTrace({ frames }: { frames: DebugState['stack'] }) {
  if (frames.length === 0) {
    return <div className="debug-empty">No call stack</div>
  }

  return (
    <div className="debug-stack">
      {frames.map((frame) => (
        <div key={frame.level} className="debug-stack-frame">
          <span className="debug-stack-fn">{frame.function}</span>
          <span className="debug-stack-file">
            {frame.filename}:{frame.lineno}
          </span>
        </div>
      ))}
    </div>
  )
}

function VariableList({ name, variables }: { name: string; variables: DebugState['locals'] }) {
  if (variables.length === 0) {
    return null
  }

  return (
    <div className="debug-variables">
      <div className="debug-variables-title">{name}</div>
      {variables.map((v) => (
        <div key={v.name} className="debug-variable">
          <span className="debug-var-name">{v.name}</span>
          <span className="debug-var-sep"> = </span>
          <span className="debug-var-value">{v.value}</span>
          <span className="debug-var-type">{v.type}</span>
        </div>
      ))}
    </div>
  )
}

export function DebugPanel({ state }: DebugPanelProps) {
  if (!state) {
    return (
      <div className="debug-panel">
        <div className="debug-placeholder">No active debug session</div>
      </div>
    )
  }

  return (
    <div className="debug-panel">
      <div className="debug-controls">
        <button className="debug-btn" title="Continue (F5)" disabled={state.status !== 'break'}>
          ▶ Continue
        </button>
        <button className="debug-btn" title="Step Over (F10)" disabled={state.status !== 'break'}>
          ↘ Step Over
        </button>
        <button className="debug-btn" title="Step Into (F11)" disabled={state.status !== 'break'}>
          ↓ Step Into
        </button>
        <button className="debug-btn" title="Step Out (⇧F11)" disabled={state.status !== 'break'}>
          ↑ Step Out
        </button>
        <button className="debug-btn debug-btn-stop" title="Stop">
          ■ Stop
        </button>
      </div>

      <div className="debug-info">
        <StackTrace frames={state.stack} />
        {state.locals.length > 0 && <VariableList name="Locals" variables={state.locals} />}
        {state.globals.length > 0 && <VariableList name="Globals" variables={state.globals} />}
      </div>
    </div>
  )
}
