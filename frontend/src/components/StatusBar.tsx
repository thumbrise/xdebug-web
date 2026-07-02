import type { DebugStatus } from '../types'

interface StatusBarProps {
  connected: boolean
  debugStatus: DebugStatus
}

export function StatusBar({ connected, debugStatus }: StatusBarProps) {
  return (
    <div className="status-bar">
      <span className="status-left">
        <span className={`status-indicator ${connected ? 'connected' : 'disconnected'}`} />
        {connected ? 'Connected' : 'Disconnected'}
      </span>
      <span className="status-right">
        Debug: {debugStatus}
      </span>
    </div>
  )
}
