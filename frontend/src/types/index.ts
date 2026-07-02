export interface FileEntry {
  name: string
  type: 'file' | 'directory'
  path: string
  children?: FileEntry[]
}

export interface FileTree {
  root: string
  entries: FileEntry[]
}

export interface StackFrame {
  level: number
  function: string
  filename: string
  lineno: number
}

export interface DebugVariable {
  name: string
  value: string
  type: string
  children?: DebugVariable[]
}

export type DebugStatus = 'idle' | 'running' | 'break' | 'stopped'

export interface DebugState {
  status: DebugStatus
  stack: StackFrame[]
  locals: DebugVariable[]
  globals: DebugVariable[]
  superglobals: DebugVariable[]
  currentFile: string
  currentLine: number
}
