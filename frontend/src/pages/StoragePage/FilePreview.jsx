import { useEffect, useState } from 'react'
import { contentUrl, downloadUrl, fetchTextPreview } from '../../services/storageApi.js'
import { fileKind, formatSize, iconFor, isPreviewable } from '../../utils/fileKinds.js'

// TextPreview lazily fetches just the head of a text/code file via a Range
// request so previewing a large file is cheap.
function TextPreview({ node }) {
  const [state, setState] = useState({ status: 'loading', text: '', truncated: false })

  useEffect(() => {
    let active = true

    async function load() {
      if (active) setState({ status: 'loading', text: '', truncated: false })
      try {
        const { text, truncated } = await fetchTextPreview(node)
        if (active) setState({ status: 'ready', text, truncated })
      } catch {
        if (active) setState({ status: 'error', text: '', truncated: false })
      }
    }

    load()
    return () => {
      active = false
    }
  }, [node])

  if (state.status === 'loading') return <div className="preview-spinner" aria-label="Cargando" />
  if (state.status === 'error') return <p className="preview-fallback-text">No se pudo leer el archivo.</p>

  return (
    <div className="preview-text-wrap">
      <pre className="preview-text">{state.text}</pre>
      {state.truncated && <p className="preview-truncated">Vista previa truncada — descarga el archivo para verlo completo.</p>}
    </div>
  )
}

function PreviewBody({ node }) {
  const kind = fileKind(node)
  const url = contentUrl(node)

  switch (kind) {
    case 'image':
      return <img className="preview-image" src={url} alt={node.name} />
    case 'video':
      return <video className="preview-media" src={url} controls autoPlay />
    case 'audio':
      return (
        <div className="preview-audio-wrap">
          <span className="preview-audio-icon" aria-hidden="true">🎵</span>
          <audio src={url} controls autoPlay />
        </div>
      )
    case 'pdf':
      return <iframe className="preview-frame" src={url} title={node.name} />
    case 'text':
    case 'code':
      return <TextPreview node={node} />
    default:
      return (
        <div className="preview-fallback">
          <span className="preview-fallback-icon" aria-hidden="true">{iconFor(node)}</span>
          <p>No hay vista previa para este tipo de archivo.</p>
        </div>
      )
  }
}

export function FilePreview({ node, hasPrev, hasNext, onPrev, onNext, onClose, onDelete }) {
  useEffect(() => {
    function onKey(event) {
      if (event.key === 'Escape') onClose()
      if (event.key === 'ArrowLeft' && hasPrev) onPrev()
      if (event.key === 'ArrowRight' && hasNext) onNext()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [hasPrev, hasNext, onPrev, onNext, onClose])

  if (!node) return null

  return (
    <div className="preview-backdrop" onClick={onClose} role="dialog" aria-modal="true" aria-label={node.name}>
      <div className="preview-shell" onClick={(event) => event.stopPropagation()}>
        <header className="preview-header">
          <div className="preview-title">
            <span aria-hidden="true">{iconFor(node)}</span>
            <span className="preview-name" title={node.name}>{node.name}</span>
            <span className="preview-size">{formatSize(node.sizeBytes)}</span>
          </div>
          <div className="preview-tools">
            <a className="preview-tool" href={contentUrl(node)} target="_blank" rel="noreferrer" title="Abrir en pestaña nueva">↗</a>
            <a className="preview-tool" href={downloadUrl(node)} download title="Descargar">↓</a>
            <button type="button" className="preview-tool preview-tool-danger" onClick={() => onDelete(node)} title="Eliminar">✕</button>
            <button type="button" className="preview-tool" onClick={onClose} title="Cerrar (Esc)">⨯</button>
          </div>
        </header>

        <div className="preview-stage">
          {hasPrev && (
            <button type="button" className="preview-nav preview-nav-prev" onClick={onPrev} aria-label="Anterior">‹</button>
          )}
          <div className="preview-content">
            {isPreviewable(node) ? (
              <PreviewBody node={node} />
            ) : (
              <div className="preview-fallback">
                <span className="preview-fallback-icon" aria-hidden="true">{iconFor(node)}</span>
                <p>No hay vista previa para este archivo.</p>
                <a className="preview-download-btn" href={downloadUrl(node)} download>Descargar</a>
              </div>
            )}
          </div>
          {hasNext && (
            <button type="button" className="preview-nav preview-nav-next" onClick={onNext} aria-label="Siguiente">›</button>
          )}
        </div>
      </div>
    </div>
  )
}
