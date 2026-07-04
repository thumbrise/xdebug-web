import type { DebugState } from '../types'

interface DebugPanelProps {
  state: DebugState | null
  onSend: (cmd: string) => void
}

function StackTrace({ frames }: { frames: DebugState['stack'] }) {
  if (frames.length === 0) {
    return <div className="debug-empty">No call stack</div>
  }

  return (
    <div className="debug-stack">
      {frames.map((frame) => (
        <div key={frame.level} className="debug-stack-frame">
          <span className="debug-stack-fn">{frame.where}</span>
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

export function DebugPanel({ state, onSend }: DebugPanelProps) {
  if (!state) {
    return (
      <div className="debug-panel">
        <div className="debug-placeholder">No active debug session</div>
      </div>
    )
  }

  const stack = state.stack ?? []
  const locals = state.locals ?? []
  const globals = state.globals ?? []
  const isBreak = state.status === 'break'

  return (
    <div className="debug-panel">
      <div className="debug-controls">
        <button
          className="debug-btn"
          title="Continue (F5)"
          disabled={!isBreak}
          onClick={() => onSend('run')}
        >
          ▶ Continue
        </button>
        <button
          className="debug-btn"
          title="Step Over (F10)"
          disabled={!isBreak}
          onClick={() => onSend('step_over')}
        >
          ↘ Step Over
        </button>
        <button
          className="debug-btn"
          title="Step Into (F11)"
          disabled={!isBreak}
          onClick={() => onSend('step_into')}
        >
          ↓ Step Into
        </button>
        <button
          className="debug-btn"
          title="Step Out (⇧F11)"
          disabled={!isBreak}
          onClick={() => onSend('step_out')}
        >
          ↑ Step Out
        </button>
        <button className="debug-btn debug-btn-stop" title="Stop" onClick={() => onSend('stop')}>
          ■ Stop
        </button>
      </div>

      <div className="debug-info">
        <StackTrace frames={stack} />
        {locals.length > 0 && <VariableList name="Locals" variables={locals} />}
        {globals.length > 0 && <VariableList name="Globals" variables={globals} />}
      </div>
    </div>
  )
}
