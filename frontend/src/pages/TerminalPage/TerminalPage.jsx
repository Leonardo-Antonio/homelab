import { useCallback, useEffect, useRef, useState } from 'react'
import { Button } from '../../components/Button.jsx'
import { EmptyState } from '../../components/EmptyState.jsx'
import { useTerminalSession } from '../../hooks/useTerminalSession.js'
import './TerminalPage.css'

const STATUS_LABELS = {
  connecting: 'Conectando',
  connected: 'Conectado',
  disconnected: 'Desconectado',
}

export function TerminalPage() {
  const { containerRef, status, info, reconnect } = useTerminalSession()
  const consoleRef = useRef(null)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const isDisabled = info?.enabled === false

  useEffect(() => {
    function handleChange() {
      setIsFullscreen(document.fullscreenElement === consoleRef.current)
    }

    document.addEventListener('fullscreenchange', handleChange)
    return () => document.removeEventListener('fullscreenchange', handleChange)
  }, [])

  const toggleFullscreen = useCallback(() => {
    if (document.fullscreenElement) {
      document.exitFullscreen()
    } else {
      consoleRef.current?.requestFullscreen?.()
    }
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
    <section className="terminal-page" aria-labelledby="terminal-title">
      <header className="terminal-header">
        <div>
          <p className="eyebrow">Remote shell</p>
          <h1 id="terminal-title">Terminal SSH en vivo.</h1>
        </div>
        <div className={`terminal-status terminal-status-${status}`}>{STATUS_LABELS[status]}</div>
      </header>

      <section className="terminal-console" aria-label="Terminal SSH" ref={consoleRef}>
        <div className="terminal-toolbar">
          <span>{target}</span>
          <div className="terminal-toolbar-actions">
            <Button
              type="button"
              variant="ghost"
              onClick={toggleFullscreen}
              aria-pressed={isFullscreen}
            >
              {isFullscreen ? 'Salir de pantalla completa' : 'Pantalla completa'}
            </Button>
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
