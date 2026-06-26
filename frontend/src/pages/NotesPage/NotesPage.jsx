import { useCallback, useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import rehypeHighlight from 'rehype-highlight'
import remarkGfm from 'remark-gfm'
import { Button } from '../../components/Button.jsx'
import { EmptyState } from '../../components/EmptyState.jsx'
import { useNotes } from '../../hooks/useNotes.js'
import { notify } from '../../services/notifications.js'
import './NotesPage.css'

// ─── Toolbar ────────────────────────────────────────────────────────────────

const FORMATS = [
  { label: 'B',   title: 'Bold',        prefix: '**', suffix: '**', placeholder: 'bold' },
  { label: 'I',   title: 'Italic',      prefix: '_',  suffix: '_',  placeholder: 'italic' },
  { label: 'H1',  title: 'Heading 1',   prefix: '# ', suffix: '',   placeholder: 'Heading', line: true },
  { label: 'H2',  title: 'Heading 2',   prefix: '## ', suffix: '',  placeholder: 'Heading', line: true },
  { label: 'H3',  title: 'Heading 3',   prefix: '### ', suffix: '', placeholder: 'Heading', line: true },
  { label: '`·`', title: 'Inline code', prefix: '`',  suffix: '`',  placeholder: 'code' },
  { label: '```', title: 'Code block',  prefix: '```\n', suffix: '\n```', placeholder: 'code', block: true },
  { label: '——',  title: 'Divider',     prefix: '\n---\n', suffix: '', placeholder: '', block: true },
  { label: '[ ]', title: 'Task',        prefix: '- [ ] ', suffix: '', placeholder: 'task', line: true },
  { label: '≡',   title: 'List',        prefix: '- ',  suffix: '',  placeholder: 'item', line: true },
  { label: '1.',  title: 'Ordered list',prefix: '1. ', suffix: '',  placeholder: 'item', line: true },
  { label: '"',   title: 'Blockquote',  prefix: '> ',  suffix: '',  placeholder: 'quote', line: true },
  { label: '⊞',   title: 'Table',       prefix: '| Col 1 | Col 2 |\n| --- | --- |\n| ', suffix: ' | |', placeholder: 'value', block: true },
  { label: '🔗',  title: 'Link',        prefix: '[',   suffix: '](url)', placeholder: 'label' },
]

function applyFormat(draft, textarea, format) {
  if (!textarea) return draft

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const selected = draft.slice(start, end) || format.placeholder

  let prefix = format.prefix
  let suffix = format.suffix

  if (format.line) {
    const lineStart = draft.lastIndexOf('\n', start - 1) + 1
    const before = draft.slice(0, lineStart)
    const line = draft.slice(lineStart, end)
    const after = draft.slice(end)
    const newValue = before + prefix + (line || format.placeholder) + suffix + after
    return { value: newValue, cursor: { start: lineStart + prefix.length, end: lineStart + prefix.length + (line || format.placeholder).length } }
  }

  const before = draft.slice(0, start)
  const after = draft.slice(end)
  const newValue = before + prefix + selected + suffix + after
  return {
    value: newValue,
    cursor: { start: start + prefix.length, end: start + prefix.length + selected.length },
  }
}

function EditorToolbar({ onFormat }) {
  return (
    <div className="editor-toolbar" role="toolbar" aria-label="Formato Markdown">
      {FORMATS.map((fmt) => (
        <button
          key={fmt.title}
          type="button"
          className="toolbar-btn"
          title={fmt.title}
          onClick={() => onFormat(fmt)}
        >
          {fmt.label}
        </button>
      ))}
    </div>
  )
}

// ─── Note Editor ────────────────────────────────────────────────────────────

function NoteEditor({ note, onSave }) {
  const [draft, setDraft] = useState(note.content)
  const [isSaving, setIsSaving] = useState(false)
  const [view, setView] = useState('split') // 'edit' | 'preview' | 'split'
  const textareaRef = useRef(null)
  const saveTimerRef = useRef(null)
  const cursorRef = useRef(null)

  useEffect(() => {
    setDraft(note.content)
  }, [note.id, note.content])

  // Restore cursor position after React re-render from toolbar action
  useEffect(() => {
    if (cursorRef.current && textareaRef.current) {
      const { start, end } = cursorRef.current
      textareaRef.current.selectionStart = start
      textareaRef.current.selectionEnd = end
      cursorRef.current = null
    }
  })

  function triggerSave(value) {
    clearTimeout(saveTimerRef.current)
    saveTimerRef.current = setTimeout(async () => {
      setIsSaving(true)
      try {
        await onSave(note.id, { name: note.name, content: value, parentId: note.parentId })
      } catch {
        notify.actionFailed('No se pudo guardar', 'Revisa si el backend está disponible.')
      } finally {
        setIsSaving(false)
      }
    }, 800)
  }

  function handleChange(e) {
    const value = e.target.value
    setDraft(value)
    triggerSave(value)
  }

  function handleFormat(format) {
    const result = applyFormat(draft, textareaRef.current, format)
    if (!result) return
    cursorRef.current = result.cursor
    setDraft(result.value)
    triggerSave(result.value)
  }

  return (
    <div className="note-editor">
      <div className="editor-header">
        <h2 className="editor-title">{note.name}</h2>
        <div className="editor-meta">
          {isSaving && <span className="saving-indicator">Guardando…</span>}
          <div className="view-toggle" role="group" aria-label="Vista">
            <button type="button" className={`view-btn ${view === 'edit' ? 'view-btn-active' : ''}`} onClick={() => setView('edit')}>Editar</button>
            <button type="button" className={`view-btn ${view === 'split' ? 'view-btn-active' : ''}`} onClick={() => setView('split')}>Dividir</button>
            <button type="button" className={`view-btn ${view === 'preview' ? 'view-btn-active' : ''}`} onClick={() => setView('preview')}>Vista previa</button>
          </div>
        </div>
      </div>

      <EditorToolbar onFormat={handleFormat} />

      <div className={`editor-body editor-body-${view}`}>
        {view !== 'preview' && (
          <textarea
            ref={textareaRef}
            className="editor-textarea"
            value={draft}
            onChange={handleChange}
            placeholder="Escribe en Markdown… # Titulo, **bold**, `code`, etc."
            spellCheck
          />
        )}
        {view !== 'edit' && (
          <div className="editor-preview">
            {draft.trim() ? (
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[[rehypeHighlight, { detect: false, ignoreMissing: true }]]}
                className="md-content"
              >
                {draft}
              </ReactMarkdown>
            ) : (
              <p className="preview-empty">El preview aparecerá aquí mientras escribes.</p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// ─── File Tree ───────────────────────────────────────────────────────────────

function TreeNode({ node, depth, selectedId, renamingId, onSelect, onToggle, expanded, onRename, onRenameSubmit, onRenameCancel, onDelete, onCreateChild }) {
  const isSelected = selectedId === node.id
  const isRenaming = renamingId === node.id
  const inputRef = useRef(null)

  useEffect(() => {
    if (isRenaming) inputRef.current?.select()
  }, [isRenaming])

  function handleRenameKey(e) {
    if (e.key === 'Enter') onRenameSubmit(node.id, e.target.value.trim())
    if (e.key === 'Escape') onRenameCancel()
  }

  return (
    <li>
      <div
        className={`tree-item ${isSelected ? 'tree-item-active' : ''}`}
        style={{ paddingLeft: depth * 16 + 8 }}
      >
        {node.type === 'dir' ? (
          <button type="button" className="tree-toggle" onClick={() => onToggle(node.id)} aria-label={expanded ? 'Colapsar' : 'Expandir'}>
            {expanded ? '▾' : '▸'}
          </button>
        ) : (
          <span className="tree-spacer" aria-hidden="true" />
        )}

        <span className="tree-icon" aria-hidden="true">{node.type === 'dir' ? '▤' : '≡'}</span>

        {isRenaming ? (
          <input
            ref={inputRef}
            className="tree-rename-input"
            defaultValue={node.name}
            onBlur={(e) => onRenameSubmit(node.id, e.target.value.trim())}
            onKeyDown={handleRenameKey}
          />
        ) : (
          <button
            type="button"
            className="tree-name"
            onClick={() => node.type === 'note' ? onSelect(node.id) : onToggle(node.id)}
          >
            {node.name}
          </button>
        )}

        <div className="tree-actions">
          {node.type === 'dir' && (
            <button type="button" className="tree-action-btn" title="Nueva nota aquí" onClick={() => onCreateChild(node.id, 'note')}>+</button>
          )}
          <button type="button" className="tree-action-btn" title="Renombrar" onClick={() => onRename(node.id)}>✎</button>
          <button type="button" className="tree-action-btn tree-action-danger" title="Eliminar" onClick={() => onDelete(node.id)}>✕</button>
        </div>
      </div>

      {node.type === 'dir' && expanded && node.children.length > 0 && (
        <ul className="tree-children">
          {node.children.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedId={selectedId}
              renamingId={renamingId}
              expanded={false}
              onSelect={onSelect}
              onToggle={onToggle}
              onRename={onRename}
              onRenameSubmit={onRenameSubmit}
              onRenameCancel={onRenameCancel}
              onDelete={onDelete}
              onCreateChild={onCreateChild}
            />
          ))}
        </ul>
      )}
    </li>
  )
}

// The FileTree needs to know which dirs are expanded at its own level
function FileTree({ tree, selectedId, renamingId, onSelect, onRename, onRenameSubmit, onRenameCancel, onDelete, onCreateChild, onCreateRoot }) {
  const [expanded, setExpanded] = useState(new Set())

  function toggle(id) {
    setExpanded((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  return (
    <aside className="notes-tree">
      <div className="notes-tree-header">
        <span className="notes-tree-title">Notas</span>
        <div className="notes-tree-root-actions">
          <button type="button" className="tree-new-btn" title="Nueva nota" onClick={() => onCreateRoot('note')}>+ Nota</button>
          <button type="button" className="tree-new-btn" title="Nueva carpeta" onClick={() => onCreateRoot('dir')}>+ Carpeta</button>
        </div>
      </div>

      {tree.length === 0 ? (
        <p className="tree-empty">Aún no hay notas. Crea la primera con los botones de arriba.</p>
      ) : (
        <ul className="tree-list">
          {tree.map((node) => (
            <TreeNode
              key={node.id}
              node={node}
              depth={0}
              selectedId={selectedId}
              renamingId={renamingId}
              expanded={expanded.has(node.id)}
              onSelect={onSelect}
              onToggle={toggle}
              onRename={onRename}
              onRenameSubmit={onRenameSubmit}
              onRenameCancel={onRenameCancel}
              onDelete={onDelete}
              onCreateChild={(parentId, type) => {
                if (!expanded.has(parentId)) toggle(parentId)
                onCreateChild(parentId, type)
              }}
            />
          ))}
        </ul>
      )}
    </aside>
  )
}

// ─── Page ────────────────────────────────────────────────────────────────────

export function NotesPage() {
  const { tree, selectedId, activeNote, isLoadingTree, isLoadingNote, error, selectNote, createNode, saveNote, renameNode, deleteNode } = useNotes()
  const [renamingId, setRenamingId] = useState(null)

  useEffect(() => {
    if (error) notify.backendUnavailable(error)
  }, [error])

  const handleCreate = useCallback(async (parentId, type) => {
    const name = type === 'dir' ? 'Nueva carpeta' : 'Nueva nota'
    try {
      const node = await createNode(parentId, type, name)
      if (type === 'note') selectNote(node.id)
      setRenamingId(node.id)
    } catch {
      notify.actionFailed('No se pudo crear', 'Revisa si el backend está disponible.')
    }
  }, [createNode, selectNote])

  const handleRenameSubmit = useCallback(async (id, name) => {
    setRenamingId(null)
    if (!name) return
    try {
      await renameNode(id, name)
    } catch {
      notify.actionFailed('No se pudo renombrar', 'Intenta de nuevo.')
    }
  }, [renameNode])

  const handleDelete = useCallback(async (id) => {
    try {
      await deleteNode(id)
    } catch {
      notify.actionFailed('No se pudo eliminar', 'Revisa si el backend está disponible.')
    }
  }, [deleteNode])

  return (
    <div className="notes-page">
      <FileTree
        tree={tree}
        selectedId={selectedId}
        renamingId={renamingId}
        onSelect={selectNote}
        onRename={setRenamingId}
        onRenameSubmit={handleRenameSubmit}
        onRenameCancel={() => setRenamingId(null)}
        onDelete={handleDelete}
        onCreateRoot={(type) => handleCreate(null, type)}
        onCreateChild={handleCreate}
      />

      <div className="notes-main">
        {isLoadingTree ? (
          <div className="notes-loading" aria-label="Cargando notas">
            <span /><span /><span />
          </div>
        ) : isLoadingNote ? (
          <div className="notes-loading" aria-label="Cargando nota">
            <span /><span /><span />
          </div>
        ) : activeNote ? (
          <NoteEditor note={activeNote} onSave={saveNote} />
        ) : (
          <div className="notes-empty-wrap">
            <EmptyState
              title="Selecciona o crea una nota"
              description="Elige una nota del panel izquierdo o crea una nueva para empezar a escribir."
            />
          </div>
        )}
      </div>
    </div>
  )
}
