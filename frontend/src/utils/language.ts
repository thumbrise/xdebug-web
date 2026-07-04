const EXTENSION_MAP: Record<string, string> = {
  '.php': 'php',
  '.js': 'javascript',
  '.mjs': 'javascript',
  '.ts': 'typescript',
  '.tsx': 'typescript',
  '.css': 'css',
  '.html': 'html',
  '.json': 'json',
  '.md': 'markdown',
  '.go': 'go',
  '.yaml': 'yaml',
  '.yml': 'yaml',
  '.sql': 'sql',
  '.xml': 'xml',
}

export function detectLanguage(filename: string): string {
  const dot = filename.lastIndexOf('.')

  if (dot === -1) return 'plaintext'

  const ext = filename.slice(dot)

  return EXTENSION_MAP[ext] ?? 'plaintext'
}