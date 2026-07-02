import { useState, useEffect } from 'react'
import type { FileTree } from '../types'

export function useFileTree(): {
  tree: FileTree | null
  loading: boolean
  error: string | null
} {
  const [tree, setTree] = useState<FileTree | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    async function fetchTree() {
      try {
        const res = await fetch('/api/files')

        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`)
        }

        const data: FileTree = await res.json()

        if (!cancelled) {
          setTree(data)
          setError(null)
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load files')
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    fetchTree()

    return () => {
      cancelled = true
    }
  }, [])

  return { tree, loading, error }
}
