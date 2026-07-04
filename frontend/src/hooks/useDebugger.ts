import { useCallback, useEffect, useRef, useState } from 'react'
import type { DebugState, DebugStatus } from '../types'

interface UseDebuggerReturn {
  state: DebugState | null
  connected: boolean
  status: DebugStatus
  send: (cmd: string, args?: Record<string, string>) => void
}

const WS_RECONNECT_DELAY = 3000

export function useDebugger(): UseDebuggerReturn {
  const [state, setState] = useState<DebugState | null>(null)
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectRef = useRef<ReturnType<typeof setTimeout>>(null)
  const mountedRef = useRef(false)

  const connect = useCallback(() => {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${proto}//${window.location.host}/api/debug`

    const ws = new WebSocket(url)

    ws.onopen = () => {
      setConnected(true)
    }

    ws.onclose = () => {
      if (!mountedRef.current || wsRef.current !== ws) return

      setConnected(false)
      reconnectRef.current = setTimeout(connect, WS_RECONNECT_DELAY)
    }

    ws.onerror = () => {
      ws.close()
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)

        if (msg.type === 'state' && msg.data) {
          setState(msg.data)
        }
      } catch {
        // ignore malformed messages
      }
    }

    wsRef.current = ws
  }, [])

  useEffect(() => {
    mountedRef.current = true
    connect()

    return () => {
      mountedRef.current = false

      if (reconnectRef.current !== null) {
        clearTimeout(reconnectRef.current)
        reconnectRef.current = null
      }

      const ws = wsRef.current

      // Only close if OPEN – closing a CONNECTING socket emits a spurious error
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close()
      }

      wsRef.current = null
    }
  }, [connect])

  const send = useCallback((cmd: string, args?: Record<string, string>) => {
    const ws = wsRef.current

    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: cmd, args }))
    }
  }, [])

  const status: DebugStatus = state?.status ?? 'idle'

  return { state, connected, status, send }
}