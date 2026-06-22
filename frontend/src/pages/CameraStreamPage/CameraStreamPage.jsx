import { useMemo, useState } from 'react'
import { Button } from '../../components/Button.jsx'
import { notify } from '../../services/notifications.js'
import './CameraStreamPage.css'

const DEFAULT_CAMERA_STREAM_URL = 'http://192.168.31.67/stream'

export function CameraStreamPage() {
  const [reloadToken, setReloadToken] = useState(0)
  const [isOnline, setIsOnline] = useState(false)
  const streamUrl = import.meta.env.VITE_CAMERA_STREAM_URL || DEFAULT_CAMERA_STREAM_URL

  const streamSrc = useMemo(() => {
    const separator = streamUrl.includes('?') ? '&' : '?'
    return `${streamUrl}${separator}t=${reloadToken}`
  }, [reloadToken, streamUrl])

  function handleStreamError() {
    setIsOnline(false)
    notify.actionFailed('Stream no disponible', 'No se pudo cargar la camara de casa.')
  }

  function handleReload() {
    setIsOnline(false)
    setReloadToken((currentToken) => currentToken + 1)
  }

  return (
    <section className="camera-stream-page" aria-labelledby="camera-stream-title">
      <header className="camera-stream-header">
        <div>
          <p className="eyebrow">Home camera</p>
          <h1 id="camera-stream-title">Camara de casa en vivo.</h1>
        </div>
        <div className={`stream-status ${isOnline ? 'stream-status-online' : ''}`}>
          {isOnline ? 'En vivo' : 'Conectando'}
        </div>
      </header>

      <section className="stream-console" aria-label="Stream de camara">
        <div className="stream-toolbar">
          <span>{streamUrl}</span>
          <Button type="button" variant="ghost" onClick={handleReload}>
            Recargar
          </Button>
        </div>

        <div className="stream-frame">
          <img
            key={streamSrc}
            src={streamSrc}
            alt="Stream en vivo de la camara de casa"
            onLoad={() => setIsOnline(true)}
            onError={handleStreamError}
          />
          {!isOnline ? (
            <div className="stream-overlay">
              <span aria-hidden="true" />
              <strong>Esperando video</strong>
            </div>
          ) : null}
        </div>
      </section>
    </section>
  )
}
