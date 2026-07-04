export interface InitInfo {
  language: string
  fileUri: string
  appId: string
  ideKey: string
}

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
  filename: string
  lineno: number
  where: string
}

export interface Breakpoint {
  id: string
  file: string
  line: number
}

export interface DebugVariable {
  name: string
  value: string
  type: string
  className?: string
  numChildren: number
  children?: DebugVariable[]
}

export type DebugStatus = 'idle' | 'starting' | 'running' | 'break' | 'stopped'

export interface DebugState {
  status: DebugStatus
  initInfo?: InitInfo
  currentFile: string
  currentLine: number
  stack: StackFrame[]
  locals: DebugVariable[]
  globals: DebugVariable[]
  breakpoints: Breakpoint[]
}

export interface OutgoingMessage {
  type: string
  data?: DebugState
}

export interface IncomingCommand {
  type: string
  args?: Record<string, string>
}
