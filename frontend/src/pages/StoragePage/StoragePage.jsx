import { useCallback, useEffect, useRef, useState } from 'react'
import { Button } from '../../components/Button.jsx'
import { EmptyState } from '../../components/EmptyState.jsx'
import { useStorage } from '../../hooks/useStorage.js'
import { downloadUrl } from '../../services/storageApi.js'
import { notify } from '../../services/notifications.js'
import './StoragePage.css'

const dateFormatter = new Intl.DateTimeFormat('es', { dateStyle: 'medium', timeStyle: 'short' })

function formatSize(bytes) {
  if (!bytes) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value >= 10 || unit === 0 ? Math.round(value) : value.toFixed(1)} ${units[unit]}`
}

function iconForFile(name, contentType = '') {
  const ext = name.split('.').pop()?.toLowerCase() ?? ''
  if (contentType.startsWith('image/') || ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg'].includes(ext)) return '🖼'
  if (contentType.startsWith('video/') || ['mp4', 'mkv', 'mov', 'webm'].includes(ext)) return '🎬'
  if (contentType.startsWith('audio/') || ['mp3', 'wav', 'flac', 'ogg'].includes(ext)) return '🎵'
  if (['pdf'].includes(ext)) return '📕'
  if (['zip', 'tar', 'gz', 'rar', '7z'].includes(ext)) return '🗜'
  if (['doc', 'docx', 'txt', 'md', 'rtf'].includes(ext)) return '📄'
  if (['xls', 'xlsx', 'csv'].includes(ext)) return '📊'
  if (['js', 'ts', 'jsx', 'tsx', 'go', 'py', 'rs', 'json', 'html', 'css', 'sh'].includes(ext)) return '🧩'
  return '📄'
}

// ─── Item row ─────────────────────────────────────────────────────────────────

function StorageItem({ node, renaming, onOpen, onStartRename, onRenameSubmit, onRenameCancel, onDownload, onDelete }) {
  const isDir = node.type === 'dir'
  const inputRef = useRef(null)

  useEffect(() => {
    if (renaming) inputRef.current?.select()
  }, [renaming])

  function handleKey(event) {
    if (event.key === 'Enter') onRenameSubmit(node.id, event.target.value.trim())
    if (event.key === 'Escape') onRenameCancel()
  }

  return (
    <div className={`storage-item ${isDir ? 'storage-item-dir' : ''}`}>
      <button
        type="button"
        className="storage-item-main"
        onDoubleClick={() => isDir && onOpen(node.id)}
        onClick={() => isDir && onOpen(node.id)}
      >
        <span className="storage-item-icon" aria-hidden="true">
          {isDir ? '📁' : iconForFile(node.name, node.contentType)}
        </span>
        {renaming ? (
          <input
            ref={inputRef}
            className="storage-rename-input"
            defaultValue={node.name}
            onClick={(event) => event.stopPropagation()}
            onBlur={(event) => onRenameSubmit(node.id, event.target.value.trim())}
            onKeyDown={handleKey}
          />
        ) : (
          <span className="storage-item-name" title={node.name}>{node.name}</span>
        )}
      </button>

      <span className="storage-item-meta storage-item-size">{isDir ? 'Carpeta' : formatSize(node.sizeBytes)}</span>
      <span className="storage-item-meta storage-item-date">{dateFormatter.format(new Date(node.updatedAt))}</span>

      <div className="storage-item-actions">
        {!isDir && (
          <a className="storage-action" href={onDownload(node)} title="Descargar" download>↓</a>
        )}
        <button type="button" className="storage-action" title="Renombrar" onClick={() => onStartRename(node.id)}>✎</button>
        <button type="button" className="storage-action storage-action-danger" title="Eliminar" onClick={() => onDelete(node)}>✕</button>
      </div>
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function StoragePage() {
  const { currentId, items, breadcrumb, isLoading, error, uploads, open, addFolder, rename, remove, upload } = useStorage()
  const [renamingId, setRenamingId] = useState(null)
  const [isDragging, setIsDragging] = useState(false)
  const fileInputRef = useRef(null)
  const dragDepth = useRef(0)

  useEffect(() => {
    if (error) notify.backendUnavailable(error)
  }, [error])

  const handleUpload = useCallback(async (files) => {
    if (!files || files.length === 0) return
    try {
      await upload(files)
      notify.actionSucceeded?.('Subida completa', 'Tus archivos ya están guardados.')
    } catch (uploadError) {
      notify.actionFailed('No se pudo subir', uploadError.message)
    }
  }, [upload])

  const handleDrop = useCallback((event) => {
    event.preventDefault()
    dragDepth.current = 0
    setIsDragging(false)
    handleUpload(event.dataTransfer.files)
  }, [handleUpload])

  const handleDragEnter = useCallback((event) => {
    event.preventDefault()
    dragDepth.current += 1
    setIsDragging(true)
  }, [])

  const handleDragLeave = useCallback((event) => {
    event.preventDefault()
    dragDepth.current -= 1
    if (dragDepth.current <= 0) setIsDragging(false)
  }, [])

  const handleNewFolder = useCallback(async () => {
    const name = window.prompt('Nombre de la carpeta', 'Nueva carpeta')
    if (!name?.trim()) return
    try {
      await addFolder(name.trim())
    } catch {
      notify.actionFailed('No se pudo crear la carpeta', 'Quizá ya existe una con ese nombre.')
    }
  }, [addFolder])

  const handleRenameSubmit = useCallback(async (id, name) => {
    setRenamingId(null)
    if (!name) return
    try {
      await rename(id, name)
    } catch {
      notify.actionFailed('No se pudo renombrar', 'Quizá el nombre ya está en uso.')
    }
  }, [rename])

  const handleDelete = useCallback(async (node) => {
    const label = node.type === 'dir' ? 'la carpeta y todo su contenido' : 'el archivo'
    if (!window.confirm(`¿Eliminar ${label} "${node.name}"? Esta acción no se puede deshacer.`)) return
    try {
      await remove(node.id)
    } catch {
      notify.actionFailed('No se pudo eliminar', 'Revisa si el backend está disponible.')
    }
  }, [remove])

  return (
    <div
      className={`storage-page ${isDragging ? 'storage-page-dragging' : ''}`}
      onDragEnter={handleDragEnter}
      onDragOver={(event) => event.preventDefault()}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <header className="storage-header">
        <div className="storage-titles">
          <h1>Drive</h1>
          <p>Tu almacenamiento personal, guardado de forma segura en tu servidor.</p>
        </div>
        <div className="storage-actions-bar">
          <Button variant="ghost" onClick={handleNewFolder}>+ Carpeta</Button>
          <Button variant="primary" onClick={() => fileInputRef.current?.click()}>↑ Subir archivos</Button>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            hidden
            onChange={(event) => {
              handleUpload(event.target.files)
              event.target.value = ''
            }}
          />
        </div>
      </header>

      <nav className="storage-breadcrumb" aria-label="Ruta">
        <button type="button" className="crumb" onClick={() => open(null)} disabled={!currentId}>
          🏠 Inicio
        </button>
        {breadcrumb.map((crumb) => (
          <span key={crumb.id} className="crumb-wrap">
            <span className="crumb-sep" aria-hidden="true">/</span>
            <button
              type="button"
              className="crumb"
              onClick={() => open(crumb.id)}
              disabled={crumb.id === currentId}
            >
              {crumb.name}
            </button>
          </span>
        ))}
      </nav>

      {uploads.length > 0 && (
        <ul className="storage-uploads" aria-label="Subidas en curso">
          {uploads.map((entry) => (
            <li key={entry.key} className={`upload-row upload-${entry.status}`}>
              <span className="upload-name" title={entry.name}>{entry.name}</span>
              <span className="upload-bar">
                <span className="upload-bar-fill" style={{ width: `${Math.round(entry.progress * 100)}%` }} />
              </span>
              <span className="upload-pct">
                {entry.status === 'error' ? '⚠' : `${Math.round(entry.progress * 100)}%`}
              </span>
            </li>
          ))}
        </ul>
      )}

      <section className="storage-panel">
        {isLoading ? (
          <div className="storage-loading" aria-label="Cargando">
            <span /><span /><span />
          </div>
        ) : items.length === 0 ? (
          <div className="storage-empty-wrap">
            <EmptyState
              title="Esta carpeta está vacía"
              description="Arrastra archivos aquí o usa “Subir archivos” para empezar."
            />
          </div>
        ) : (
          <div className="storage-list">
            <div className="storage-list-head">
              <span>Nombre</span>
              <span className="storage-item-size">Tamaño</span>
              <span className="storage-item-date">Modificado</span>
              <span />
            </div>
            {items.map((node) => (
              <StorageItem
                key={node.id}
                node={node}
                renaming={renamingId === node.id}
                onOpen={open}
                onStartRename={setRenamingId}
                onRenameSubmit={handleRenameSubmit}
                onRenameCancel={() => setRenamingId(null)}
                onDownload={downloadUrl}
                onDelete={handleDelete}
              />
            ))}
          </div>
        )}
      </section>

      <div className="storage-dropzone-overlay" aria-hidden={!isDragging}>
        <div className="storage-dropzone-card">
          <span className="storage-dropzone-icon">⬇</span>
          <p>Suelta los archivos para subirlos aquí</p>
        </div>
      </div>
    </div>
  )
}
