import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button } from '../../components/Button.jsx'
import { EmptyState } from '../../components/EmptyState.jsx'
import { useStorage } from '../../hooks/useStorage.js'
import { downloadUrl, thumbUrl } from '../../services/storageApi.js'
import { notify } from '../../services/notifications.js'
import { fileKind, formatDate, formatSize, iconFor } from '../../utils/fileKinds.js'
import { FilePreview } from './FilePreview.jsx'
import './StoragePage.css'

const VIEW_KEY = 'homelab.drive.view'
const SORTS = {
  name: (a, b) => a.name.localeCompare(b.name, 'es', { numeric: true }),
  size: (a, b) => (b.sizeBytes ?? 0) - (a.sizeBytes ?? 0),
  date: (a, b) => new Date(b.updatedAt) - new Date(a.updatedAt),
}

function readView() {
  if (typeof window === 'undefined') return 'grid'
  return window.localStorage.getItem(VIEW_KEY) === 'list' ? 'list' : 'grid'
}

// ─── Thumbnail / icon tile ────────────────────────────────────────────────────

function Thumb({ node, size }) {
  const [failed, setFailed] = useState(false)
  const url = thumbUrl(node)
  const showImage = url && !failed && fileKind(node) === 'image'

  if (showImage) {
    return (
      <img
        className={`thumb thumb-${size}`}
        src={url}
        alt=""
        loading="lazy"
        onError={() => setFailed(true)}
      />
    )
  }
  return (
    <span className={`thumb thumb-icon thumb-${size}`} aria-hidden="true">
      {iconFor(node)}
    </span>
  )
}

// ─── Inline rename input ──────────────────────────────────────────────────────

function RenameInput({ defaultValue, onSubmit, onCancel }) {
  const ref = useRef(null)
  useEffect(() => {
    const input = ref.current
    if (!input) return
    input.focus()
    const dot = defaultValue.lastIndexOf('.')
    input.setSelectionRange(0, dot > 0 ? dot : defaultValue.length)
  }, [defaultValue])

  return (
    <input
      ref={ref}
      className="rename-input"
      defaultValue={defaultValue}
      onClick={(event) => event.stopPropagation()}
      onDoubleClick={(event) => event.stopPropagation()}
      onBlur={(event) => onSubmit(event.target.value.trim())}
      onKeyDown={(event) => {
        if (event.key === 'Enter') onSubmit(event.target.value.trim())
        if (event.key === 'Escape') onCancel()
      }}
    />
  )
}

// ─── Grid card ────────────────────────────────────────────────────────────────

function GridCard({ node, isRenaming, onActivate, menu, dnd }) {
  const isDir = node.type === 'dir'
  const className = [
    'grid-card',
    isDir ? 'grid-card-dir' : '',
    dnd.isDragging ? 'dnd-dragging' : '',
    dnd.isDropTarget ? 'dnd-drop-target' : '',
  ].filter(Boolean).join(' ')

  return (
    <div
      className={className}
      onDoubleClick={() => onActivate(node)}
      tabIndex={0}
      onKeyDown={(event) => event.key === 'Enter' && onActivate(node)}
      {...dnd.itemProps}
      {...dnd.folderDropProps}
    >
      <button type="button" className="grid-card-body" onClick={() => onActivate(node)}>
        <span className="grid-card-thumb">
          <Thumb node={node} size="lg" />
        </span>
      </button>
      <div className="grid-card-foot">
        {isRenaming ? (
          <RenameInput {...menu.renameProps} />
        ) : (
          <span className="grid-card-name" title={node.name}>{node.name}</span>
        )}
        <span className="grid-card-meta">{isDir ? 'Carpeta' : formatSize(node.sizeBytes)}</span>
      </div>
      {menu.element}
    </div>
  )
}

// ─── List row ─────────────────────────────────────────────────────────────────

function ListRow({ node, isRenaming, onActivate, menu, dnd }) {
  const isDir = node.type === 'dir'
  const className = [
    'list-row',
    dnd.isDragging ? 'dnd-dragging' : '',
    dnd.isDropTarget ? 'dnd-drop-target' : '',
  ].filter(Boolean).join(' ')

  return (
    <div className={className} onDoubleClick={() => onActivate(node)} {...dnd.itemProps} {...dnd.folderDropProps}>
      <button type="button" className="list-row-main" onClick={() => onActivate(node)}>
        <Thumb node={node} size="sm" />
        {isRenaming ? (
          <RenameInput {...menu.renameProps} />
        ) : (
          <span className="list-row-name" title={node.name}>{node.name}</span>
        )}
      </button>
      <span className="list-cell list-cell-size">{isDir ? 'Carpeta' : formatSize(node.sizeBytes)}</span>
      <span className="list-cell list-cell-date">{formatDate(node.updatedAt)}</span>
      <div className="list-cell list-cell-actions">{menu.inlineActions}</div>
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function StoragePage() {
  const { currentId, items, breadcrumb, isLoading, error, uploads, open, addFolder, rename, move, remove, upload } = useStorage()
  const [view, setView] = useState(readView)
  const [sort, setSort] = useState('name')
  const [query, setQuery] = useState('')
  const [renamingId, setRenamingId] = useState(null)
  const [previewId, setPreviewId] = useState(null)
  const [isDragging, setIsDragging] = useState(false)
  // Internal move drag-and-drop: which item is being dragged and which folder /
  // breadcrumb crumb is currently the hovered drop target.
  const [draggingId, setDraggingId] = useState(null)
  const [dropTargetId, setDropTargetId] = useState(null)
  const dragNodeRef = useRef(null)
  const fileInputRef = useRef(null)
  const dragDepth = useRef(0)

  useEffect(() => {
    window.localStorage.setItem(VIEW_KEY, view)
  }, [view])

  useEffect(() => {
    if (error) notify.backendUnavailable(error)
  }, [error])

  // Folders always first, then the chosen sort, then an optional name filter.
  const visibleItems = useMemo(() => {
    const needle = query.trim().toLowerCase()
    const filtered = needle ? items.filter((n) => n.name.toLowerCase().includes(needle)) : items
    return [...filtered].sort((a, b) => {
      if (a.type !== b.type) return a.type === 'dir' ? -1 : 1
      return SORTS[sort](a, b)
    })
  }, [items, query, sort])

  const fileList = useMemo(() => visibleItems.filter((n) => n.type === 'file'), [visibleItems])
  const previewIndex = fileList.findIndex((n) => n.id === previewId)
  const previewNode = previewIndex >= 0 ? fileList[previewIndex] : null

  // ── actions ──────────────────────────────────────────────────────────────

  const handleUpload = useCallback(async (files) => {
    if (!files || files.length === 0) return
    try {
      await upload(files)
      notify.actionSucceeded('Subida completa', 'Tus archivos ya están guardados en el servidor.')
    } catch (uploadError) {
      notify.actionFailed('No se pudo subir', uploadError.message)
    }
  }, [upload])

  const handleActivate = useCallback((node) => {
    if (node.type === 'dir') open(node.id)
    else setPreviewId(node.id)
  }, [open])

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
      setPreviewId((current) => (current === node.id ? null : current))
    } catch {
      notify.actionFailed('No se pudo eliminar', 'Revisa si el backend está disponible.')
    }
  }, [remove])

  // Builds the per-item menu/actions shared by both views.
  const buildMenu = useCallback((node) => {
    const renameProps = {
      defaultValue: node.name,
      onSubmit: (name) => handleRenameSubmit(node.id, name),
      onCancel: () => setRenamingId(null),
    }
    const inlineActions = (
      <>
        {node.type === 'file' && (
          <a className="row-action" href={downloadUrl(node)} download title="Descargar" onClick={(e) => e.stopPropagation()}>↓</a>
        )}
        <button type="button" className="row-action" title="Renombrar" onClick={(e) => { e.stopPropagation(); setRenamingId(node.id) }}>✎</button>
        <button type="button" className="row-action row-action-danger" title="Eliminar" onClick={(e) => { e.stopPropagation(); handleDelete(node) }}>✕</button>
      </>
    )
    return {
      renameProps,
      inlineActions,
      element: <div className="grid-card-actions" onClick={(e) => e.stopPropagation()}>{inlineActions}</div>,
    }
  }, [handleDelete, handleRenameSubmit])

  // ── move via drag and drop ───────────────────────────────────────────────

  // A move is valid unless the item is dropped on itself or back into the
  // folder it already lives in. Siblings can never be descendants of each
  // other, and breadcrumb targets are ancestors, so no cycle is reachable from
  // this UI — the backend still enforces it as a backstop.
  const canDropOn = useCallback((node, targetId) => {
    if (!node) return false
    if (node.id === targetId) return false
    if ((node.parentId ?? null) === (targetId ?? null)) return false
    return true
  }, [])

  const handleMove = useCallback(async (node, targetId, targetName) => {
    if (!canDropOn(node, targetId)) return
    try {
      await move(node.id, targetId)
      notify.actionSucceeded('Movido', `“${node.name}” → ${targetName}.`)
    } catch {
      notify.actionFailed('No se pudo mover', 'Quizá ya existe un elemento con ese nombre en el destino.')
    }
  }, [canDropOn, move])

  // Drop-zone handlers shared by folder cards/rows and breadcrumb crumbs.
  // targetId null means the Drive root.
  const buildDropZone = useCallback((targetId, targetName) => ({
    onDragOver: (event) => {
      const node = dragNodeRef.current
      if (!node || !canDropOn(node, targetId)) return
      event.preventDefault()
      event.stopPropagation()
      event.dataTransfer.dropEffect = 'move'
      setDropTargetId(targetId ?? 'root')
    },
    onDragLeave: (event) => {
      event.stopPropagation()
      setDropTargetId((current) => (current === (targetId ?? 'root') ? null : current))
    },
    onDrop: (event) => {
      const node = dragNodeRef.current
      if (!node) return
      event.preventDefault()
      event.stopPropagation()
      setDropTargetId(null)
      handleMove(node, targetId, targetName)
    },
  }), [canDropOn, handleMove])

  const buildDnd = useCallback((node) => {
    const isDir = node.type === 'dir'
    return {
      isDragging: draggingId === node.id,
      isDropTarget: isDir && dropTargetId === node.id,
      itemProps: {
        draggable: renamingId !== node.id,
        onDragStart: (event) => {
          dragNodeRef.current = node
          setDraggingId(node.id)
          event.dataTransfer.effectAllowed = 'move'
          event.dataTransfer.setData('application/x-homelab-node', node.id)
          event.dataTransfer.setData('text/plain', node.name)
        },
        onDragEnd: () => {
          dragNodeRef.current = null
          setDraggingId(null)
          setDropTargetId(null)
        },
      },
      folderDropProps: isDir ? buildDropZone(node.id, node.name) : {},
    }
  }, [draggingId, dropTargetId, renamingId, buildDropZone])

  // ── drag & drop (file upload from OS) ───────────────────────────────────────

  const handleDrop = useCallback((event) => {
    event.preventDefault()
    dragDepth.current = 0
    setIsDragging(false)
    handleUpload(event.dataTransfer.files)
  }, [handleUpload])

  const handleDragEnter = useCallback((event) => {
    if (![...event.dataTransfer.types].includes('Files')) return
    event.preventDefault()
    dragDepth.current += 1
    setIsDragging(true)
  }, [])

  const handleDragLeave = useCallback((event) => {
    event.preventDefault()
    dragDepth.current -= 1
    if (dragDepth.current <= 0) setIsDragging(false)
  }, [])

  // ── render ──────────────────────────────────────────────────────────────────

  return (
    <div
      className={`storage-page ${isDragging ? 'storage-page-dragging' : ''}`}
      onDragEnter={handleDragEnter}
      onDragOver={(event) => event.preventDefault()}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <header className="drive-toolbar">
        <nav className="drive-breadcrumb" aria-label="Ruta">
          <button
            type="button"
            className={`crumb crumb-home ${dropTargetId === 'root' ? 'dnd-drop-target' : ''}`}
            onClick={() => open(null)}
            disabled={!currentId}
            {...buildDropZone(null, 'Drive')}
          >
            <span aria-hidden="true">⛁</span> Drive
          </button>
          {breadcrumb.map((crumb) => (
            <span key={crumb.id} className="crumb-wrap">
              <span className="crumb-sep" aria-hidden="true">›</span>
              <button
                type="button"
                className={`crumb ${dropTargetId === crumb.id ? 'dnd-drop-target' : ''}`}
                onClick={() => open(crumb.id)}
                disabled={crumb.id === currentId}
                {...buildDropZone(crumb.id, crumb.name)}
              >
                {crumb.name}
              </button>
            </span>
          ))}
        </nav>

        <div className="drive-toolbar-actions">
          <label className="drive-search">
            <span aria-hidden="true">🔍</span>
            <input
              type="search"
              placeholder="Buscar aquí…"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
          </label>

          <select className="drive-sort" value={sort} onChange={(event) => setSort(event.target.value)} aria-label="Ordenar">
            <option value="name">Nombre</option>
            <option value="date">Recientes</option>
            <option value="size">Tamaño</option>
          </select>

          <div className="view-toggle" role="group" aria-label="Vista">
            <button type="button" className={`view-btn ${view === 'grid' ? 'view-btn-active' : ''}`} onClick={() => setView('grid')} title="Cuadrícula">▦</button>
            <button type="button" className={`view-btn ${view === 'list' ? 'view-btn-active' : ''}`} onClick={() => setView('list')} title="Lista">≣</button>
          </div>

          <Button variant="ghost" onClick={handleNewFolder}>+ Carpeta</Button>
          <Button variant="primary" onClick={() => fileInputRef.current?.click()}>↑ Subir</Button>
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

      {uploads.length > 0 && (
        <ul className="upload-tray" aria-label="Subidas en curso">
          {uploads.map((entry) => (
            <li key={entry.key} className={`upload-row upload-${entry.status}`}>
              <span className="upload-name" title={entry.name}>{entry.name}</span>
              <span className="upload-bar"><span className="upload-bar-fill" style={{ width: `${Math.round(entry.progress * 100)}%` }} /></span>
              <span className="upload-pct">{entry.status === 'error' ? '⚠' : `${Math.round(entry.progress * 100)}%`}</span>
            </li>
          ))}
        </ul>
      )}

      <section className="drive-surface">
        {isLoading ? (
          <div className="drive-loading" aria-label="Cargando"><span /><span /><span /></div>
        ) : visibleItems.length === 0 ? (
          <div className="drive-empty">
            <EmptyState
              title={query ? 'Sin resultados' : 'Esta carpeta está vacía'}
              description={query ? 'Prueba con otro término de búsqueda.' : 'Arrastra archivos aquí o usa “Subir” para empezar.'}
            />
          </div>
        ) : view === 'grid' ? (
          <div className="drive-grid">
            {visibleItems.map((node) => (
              <GridCard key={node.id} node={node} isRenaming={renamingId === node.id} onActivate={handleActivate} menu={buildMenu(node)} dnd={buildDnd(node)} />
            ))}
          </div>
        ) : (
          <div className="drive-list">
            <div className="list-head">
              <span>Nombre</span>
              <span className="list-cell-size">Tamaño</span>
              <span className="list-cell-date">Modificado</span>
              <span />
            </div>
            {visibleItems.map((node) => (
              <ListRow key={node.id} node={node} isRenaming={renamingId === node.id} onActivate={handleActivate} menu={buildMenu(node)} dnd={buildDnd(node)} />
            ))}
          </div>
        )}
      </section>

      <div className="drive-dropzone" aria-hidden={!isDragging}>
        <div className="drive-dropzone-card">
          <span className="drive-dropzone-icon">⬇</span>
          <p>Suelta los archivos para subirlos aquí</p>
        </div>
      </div>

      {previewNode && (
        <FilePreview
          node={previewNode}
          hasPrev={previewIndex > 0}
          hasNext={previewIndex < fileList.length - 1}
          onPrev={() => setPreviewId(fileList[previewIndex - 1].id)}
          onNext={() => setPreviewId(fileList[previewIndex + 1].id)}
          onClose={() => setPreviewId(null)}
          onDelete={handleDelete}
        />
      )}
    </div>
  )
}
