import { useState, useEffect } from 'react'

export function useFileContent(path: string | null): {
  content: string | null
  loading: boolean
  error: string | null
} {
  const [content, setContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!path) {
      setContent(null)
      setLoading(false)
      setError(null)

      return
    }

    const filePath: string = path

    let cancelled = false

    async function fetchContent() {
      setLoading(true)
      setError(null)

      try {
        const res = await fetch(`/api/file?path=${encodeURIComponent(filePath)}`)

        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`)
        }

        const text = await res.text()

        if (!cancelled) {
          setContent(text)
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load file')
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    fetchContent()

    return () => {
      cancelled = true
    }
  }, [path])

  return { content, loading, error }
}
