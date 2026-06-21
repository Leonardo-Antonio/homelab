import { useEffect, useMemo, useState } from 'react'
import { Button } from '../../components/Button.jsx'
import { EmptyState } from '../../components/EmptyState.jsx'
import { useClipboardItems } from '../../hooks/useClipboardItems.js'
import { notify } from '../../services/notifications.js'
import { copyTextToClipboard } from '../../utils/clipboard.js'
import './ClipboardPage.css'

const MAX_PREVIEW_LENGTH = 220

function getPreviewText(value) {
  if (value.length <= MAX_PREVIEW_LENGTH) {
    return value
  }

  return `${value.slice(0, MAX_PREVIEW_LENGTH).trimEnd()}...`
}

export function ClipboardPage() {
  const [draft, setDraft] = useState('')
  const [copiedId, setCopiedId] = useState(null)
  const {
    items,
    addItem,
    removeItem,
    clearItems,
    goToNextPage,
    goToPreviousPage,
    pagination,
    isLoading,
    error,
  } = useClipboardItems()
  const trimmedDraft = draft.trim()
  const canAdd = trimmedDraft.length > 0

  useEffect(() => {
    if (error) {
      notify.backendUnavailable(error)
    }
  }, [error])

  const itemCountLabel = useMemo(() => {
    if (pagination.total === 1) {
      return '1 snippet listo'
    }

    return `${pagination.total} snippets listos`
  }, [pagination.total])

  async function handleSubmit(event) {
    event.preventDefault()

    if (!canAdd) {
      return
    }

    try {
      await addItem(trimmedDraft)
      setDraft('')
      notify.clipboardCreated()
    } catch {
      notify.actionFailed('No se pudo guardar', 'Revisa si el backend esta disponible.')
    }
  }

  async function handleCopy(item) {
    try {
      await copyTextToClipboard(item.text)
      setCopiedId(item.id)
      notify.clipboardCopied()
      window.setTimeout(() => setCopiedId(null), 1400)
    } catch {
      notify.actionFailed('No se pudo copiar', 'Revisa permisos del navegador e intenta de nuevo.')
    }
  }

  async function handleRemove(itemId) {
    try {
      await removeItem(itemId)
      notify.clipboardDeleted()
    } catch {
      notify.actionFailed('No se pudo borrar', 'El item no pudo eliminarse del backend.')
    }
  }

  async function handleClear() {
    try {
      await clearItems()
      notify.clipboardCleared()
    } catch {
      notify.actionFailed('No se pudo limpiar', 'La lista no pudo limpiarse en el backend.')
    }
  }

  return (
    <section className="clipboard-page" aria-labelledby="clipboard-title">
      <header className="page-header">
        <div>
          <p className="eyebrow">Copy paste clipboard</p>
          <h1 id="clipboard-title">Guarda textos rápidos y cópialos al instante.</h1>
        </div>
        <div className="status-pill" aria-label={itemCountLabel}>
          {itemCountLabel}
        </div>
      </header>

      <form className="composer" onSubmit={handleSubmit}>
        <label htmlFor="clipboard-input">Texto</label>
        <textarea
          id="clipboard-input"
          className="composer-input"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          placeholder="Escribe cualquier texto, comando, nota o respuesta frecuente..."
          rows={8}
        />
        <div className="composer-actions">
          <span className="input-meta">{trimmedDraft.length} caracteres</span>
          <Button type="submit" disabled={!canAdd}>
            Agregar
          </Button>
        </div>
      </form>

      <section className="snippet-section" aria-labelledby="snippet-list-title">
        <div className="section-heading">
          <div>
            <h2 id="snippet-list-title">Lista</h2>
            <p>
              Pagina {pagination.pages === 0 ? 0 : pagination.page} de {pagination.pages}
            </p>
          </div>
          {items.length > 0 ? (
            <Button type="button" variant="ghost" onClick={handleClear}>
              Limpiar
            </Button>
          ) : null}
        </div>

        {isLoading ? (
          <div className="snippet-skeleton" aria-label="Cargando snippets">
            <span />
            <span />
            <span />
          </div>
        ) : items.length > 0 ? (
          <ul className="snippet-list">
            {items.map((item) => (
              <li className="snippet-card" key={item.id}>
                <p>{getPreviewText(item.text)}</p>
                <div className="snippet-actions">
                  <time dateTime={item.createdAt}>{item.createdAtLabel}</time>
                  <div className="snippet-buttons">
                    <Button
                      type="button"
                      variant={copiedId === item.id ? 'success' : 'secondary'}
                      onClick={() => handleCopy(item)}
                    >
                      {copiedId === item.id ? 'Copiado' : 'Copiar'}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      onClick={() => handleRemove(item.id)}
                      aria-label="Eliminar snippet"
                    >
                      Borrar
                    </Button>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <EmptyState
            title="Aun no hay textos guardados"
            description="Agrega tu primer snippet para tenerlo disponible en un clic."
          />
        )}

        {!isLoading && pagination.total > 0 ? (
          <nav className="pagination" aria-label="Paginacion de snippets">
            <Button
              type="button"
              variant="ghost"
              onClick={goToPreviousPage}
              disabled={!pagination.hasPrevious}
            >
              Anterior
            </Button>
            <span>
              {items.length} de {pagination.total}
            </span>
            <Button
              type="button"
              variant="ghost"
              onClick={goToNextPage}
              disabled={!pagination.hasNext}
            >
              Siguiente
            </Button>
          </nav>
        ) : null}
      </section>
    </section>
  )
}
