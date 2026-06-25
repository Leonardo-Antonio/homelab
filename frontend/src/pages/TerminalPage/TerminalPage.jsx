import { useCallback } from 'react'
import { Button } from '../../components/Button.jsx'
import { EmptyState } from '../../components/EmptyState.jsx'
import { useTerminalSession } from '../../hooks/useTerminalSession.js'
import './TerminalPage.css'

const STATUS_LABELS = {
  connecting: 'Conectando',
  connected: 'Conectado',
  disconnected: 'Desconectado',
}

// Standalone view: only the terminal, filling the whole window. Opened in a new
// tab via the `?view=standalone` query param so the SPA renders it without the
// app shell, regardless of the current route.
export const STANDALONE_VIEW = 'standalone'

export function TerminalPage({ standalone = false }) {
  const { containerRef, status, info, reconnect } = useTerminalSession()
  const isDisabled = info?.enabled === false

  const openInNewTab = useCallback(() => {
    const url = new URL(window.location.href)
    url.searchParams.set('view', STANDALONE_VIEW)
    window.open(url.toString(), '_blank', 'noopener,noreferrer')
  }, [])

  if (isDisabled) {
    return (
      <section className="terminal-page" aria-labelledby="terminal-title">
        <header className="terminal-header">
          <div>
            <p className="eyebrow">Remote shell</p>
            <h1 id="terminal-title">Terminal SSH.</h1>
          </div>
        </header>
        <EmptyState
          title="Terminal deshabilitado"
          description="Activa SSH_ENABLED en el backend y configura el host para usar la terminal."
        />
      </section>
    )
  }

  const target = info?.host ? `${info.user ? `${info.user}@` : ''}${info.host}:${info.port}` : 'SSH'

  return (
    <section
      className={`terminal-page ${standalone ? 'terminal-page-standalone' : ''}`.trim()}
      aria-labelledby="terminal-title"
    >
      {!standalone ? (
        <header className="terminal-header">
          <div>
            <p className="eyebrow">Remote shell</p>
            <h1 id="terminal-title">Terminal SSH en vivo.</h1>
          </div>
          <div className={`terminal-status terminal-status-${status}`}>{STATUS_LABELS[status]}</div>
        </header>
      ) : null}

      <section className="terminal-console" aria-label="Terminal SSH">
        <div className="terminal-toolbar">
          <span>{target}</span>
          <div className="terminal-toolbar-actions">
            {standalone ? (
              <span className={`terminal-status terminal-status-${status}`}>
                {STATUS_LABELS[status]}
              </span>
            ) : (
              <Button type="button" variant="ghost" onClick={openInNewTab}>
                Abrir en pestaña nueva
              </Button>
            )}
            <Button type="button" variant="ghost" onClick={reconnect}>
              Reconectar
            </Button>
          </div>
        </div>

        <div className="terminal-frame">
          <div className="terminal-mount" ref={containerRef} />
        </div>
      </section>
    </section>
  )
}
