import { useCallback, useState } from 'react'
import type { DebugState, DebugVariable } from '../types'

interface DebugPanelProps {
  state: DebugState | null
  onSend: (cmd: string, args?: Record<string, string>) => void
  propertyData: Record<string, DebugVariable[]>
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

interface VariableRowProps {
  variable: DebugVariable
  level: number
  expanded: Set<string>
  onToggle: (fullName: string) => void
  children?: DebugVariable[]
}

function VariableRow({ variable, level, expanded, onToggle, children: childVars }: VariableRowProps) {
  const key = variable.fullName ?? variable.name
  const hasChildren = variable.numChildren > 0
  const isExpanded = expanded.has(key)

  return (
    <div className="debug-variable">
      <span
        className="debug-var-indent"
        style={{ paddingLeft: `${level * 16}px` }}
      />

      {hasChildren && (
        <span className="debug-var-toggle" onClick={() => onToggle(key)}>
          {isExpanded ? '▼' : '▶'}
        </span>
      )}

      {!hasChildren && (
        <span className="debug-var-toggle-spacer" />
      )}

      <span className="debug-var-value">{variable.value}</span>
      <span className="debug-var-sep"> </span>
      <span className="debug-var-type">{variable.type}</span>
      <span className="debug-var-name">{variable.name}</span>

      {hasChildren && isExpanded && childVars && childVars.length > 0 && (
        <div className="debug-variable-children">
          {childVars.map((child) => (
            <VariableRow
              key={child.fullName ?? child.name}
              variable={child}
              level={level + 1}
              expanded={expanded}
              onToggle={onToggle}
            />
          ))}
        </div>
      )}
    </div>
  )
}

interface VariableListProps {
  name: string
  variables: DebugVariable[]
  propertyData: Record<string, DebugVariable[]>
  onExpand: (fullName: string) => void
}

function VariableList({ name, variables, propertyData, onExpand }: VariableListProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  const handleToggle = useCallback(
    (fullName: string) => {
      setExpanded((prev) => {
        const next = new Set(prev)

        if (next.has(fullName)) {
          next.delete(fullName)
        } else {
          next.add(fullName)
          onExpand(fullName)
        }

        return next
      })
    },
    [onExpand],
  )

  if (variables.length === 0) {
    return null
  }

  return (
    <div className="debug-variables">
      <div className="debug-variables-title">{name}</div>

      {variables.map((v) => (
        <VariableRow
          key={v.fullName ?? v.name}
          variable={v}
          level={0}
          expanded={expanded}
          onToggle={handleToggle}
          children={propertyData[v.fullName ?? v.name]}
        />
      ))}
    </div>
  )
}

export function DebugPanel({ state, onSend, propertyData }: DebugPanelProps) {
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

  const handleExpand = useCallback(
    (fullName: string) => {
      onSend('property_get', { name: fullName })
    },
    [onSend],
  )

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

        {locals.length > 0 && (
          <VariableList
            name="Locals"
            variables={locals}
            propertyData={propertyData}
            onExpand={handleExpand}
          />
        )}

        {globals.length > 0 && (
          <VariableList
            name="Globals"
            variables={globals}
            propertyData={propertyData}
            onExpand={handleExpand}
          />
        )}
      </div>
    </div>
  )
}
