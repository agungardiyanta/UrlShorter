import { Clipboard, Clock3, ExternalLink, Link2, Loader2, RefreshCw, Scissors, Trash2 } from 'lucide-react'
import { FormEvent, useEffect, useMemo, useState } from 'react'

type LinkRecord = {
  code: string
  targetUrl: string
  shortUrl: string
  clicks: number
  expiresAt?: string
  expired: boolean
  createdAt: string
}

const apiBase = import.meta.env.VITE_API_BASE_URL ?? ''

export function App() {
  const [targetUrl, setTargetUrl] = useState('')
  const [customCode, setCustomCode] = useState('')
  const [links, setLinks] = useState<LinkRecord[]>([])
  const [result, setResult] = useState<LinkRecord | null>(null)
  const [loading, setLoading] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [deletingCode, setDeletingCode] = useState('')
  const [currentTime, setCurrentTime] = useState(() => new Date().getTime())
  const [error, setError] = useState('')

  const canSubmit = useMemo(() => targetUrl.trim().length > 0 && !loading, [targetUrl, loading])

  async function loadLinks() {
    setRefreshing(true)
    try {
      const response = await fetch(`${apiBase}/api/links`)
      if (!response.ok) throw new Error('Could not load links')
      const payload = (await response.json()) as { links: LinkRecord[] }
      setCurrentTime(new Date().getTime())
      setLinks(payload.links)
    } finally {
      setRefreshing(false)
    }
  }

  useEffect(() => {
    void loadLinks()
  }, [])

  async function createLink(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)
    setError('')
    setResult(null)

    try {
      const response = await fetch(`${apiBase}/api/links`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          targetUrl: targetUrl.trim(),
          customCode: customCode.trim() || undefined,
        }),
      })
      const payload = await response.json()
      if (!response.ok) throw new Error(payload.error ?? 'Could not create short link')
      setResult(payload)
      setTargetUrl('')
      setCustomCode('')
      await loadLinks()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  async function copy(value: string) {
    await navigator.clipboard.writeText(value)
  }

  async function deleteLink(code: string) {
    setDeletingCode(code)
    setError('')
    try {
      const response = await fetch(`${apiBase}/api/links/${code}`, { method: 'DELETE' })
      if (!response.ok && response.status !== 404) {
        const payload = await response.json()
        throw new Error(payload.error ?? 'Could not delete link')
      }
      setLinks((current) => current.filter((link) => link.code !== code))
      if (result?.code === code) setResult(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setDeletingCode('')
    }
  }

  function formatExpiry(value: string | undefined, now: number) {
    if (!value) return 'No expiration'
    const expiresAt = new Date(value)
    if (Number.isNaN(expiresAt.getTime())) return 'No expiration'
    const expired = expiresAt.getTime() <= now
    return `${expired ? 'Expired' : 'Expires'} ${expiresAt.toLocaleString()}`
  }

  return (
    <main className="shell">
      <section className="workspace" aria-label="UrlShorter workspace">
        <div className="compose-panel">
          <div className="brand-row">
            <div className="brand-mark" aria-hidden="true">
              <Scissors size={22} />
            </div>
            <div>
              <h1>UrlShorter</h1>
              <p>Short links backed by Fiber, Postgres, Redis, Kafka, and OpenTelemetry.</p>
            </div>
          </div>

          <form className="shorten-form" onSubmit={createLink}>
            <label>
              Destination URL
              <input
                type="url"
                value={targetUrl}
                onChange={(event) => setTargetUrl(event.target.value)}
                placeholder="https://example.com/long/path"
                required
              />
            </label>
            <label>
              Custom code
              <input
                value={customCode}
                onChange={(event) => setCustomCode(event.target.value)}
                placeholder="launch"
                maxLength={64}
              />
            </label>
            <button type="submit" disabled={!canSubmit}>
              {loading ? <Loader2 className="spin" size={18} /> : <Link2 size={18} />}
              Shorten
            </button>
          </form>

          {error && <div className="notice error">{error}</div>}
          {result && (
            <div className="notice success">
              <span>{result.shortUrl}</span>
              <button type="button" onClick={() => copy(result.shortUrl)} aria-label="Copy short URL">
                <Clipboard size={16} />
              </button>
            </div>
          )}
        </div>

        <div className="list-panel">
          <div className="list-header">
            <div>
              <h2>Recent links</h2>
              <p>{links.length} saved links</p>
            </div>
            <button type="button" className="icon-button" onClick={loadLinks} aria-label="Refresh links">
              <RefreshCw className={refreshing ? 'spin' : ''} size={18} />
            </button>
          </div>

          <div className="link-list">
            {links.map((link) => (
              <article className="link-row" key={link.code}>
                <div className="link-main">
                  <a href={link.shortUrl} target="_blank" rel="noreferrer">
                    {link.shortUrl}
                    <ExternalLink size={14} />
                  </a>
                  <span>{link.targetUrl}</span>
                  <span className="expiry">
                    <Clock3 size={14} />
                    {formatExpiry(link.expiresAt, currentTime)}
                  </span>
                </div>
                <div className="link-actions">
                  <strong>{link.clicks}</strong>
                  <span>clicks</span>
                  <button type="button" onClick={() => copy(link.shortUrl)} aria-label={`Copy ${link.shortUrl}`}>
                    <Clipboard size={16} />
                  </button>
                  <button
                    type="button"
                    className="danger-button"
                    onClick={() => deleteLink(link.code)}
                    disabled={deletingCode === link.code}
                    aria-label={`Delete ${link.shortUrl}`}
                  >
                    {deletingCode === link.code ? <Loader2 className="spin" size={16} /> : <Trash2 size={16} />}
                  </button>
                </div>
              </article>
            ))}
            {links.length === 0 && <div className="empty-state">No links yet.</div>}
          </div>
        </div>
      </section>
    </main>
  )
}
