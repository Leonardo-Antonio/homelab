import { useRef, useState } from 'react'
import { useSettings } from '../../context/SettingsContext.jsx'
import { notify } from '../../services/notifications.js'
import './ConfigPage.css'

const MODULE_IDS = ['network', 'clipboard', 'photos', 'camera', 'terminal', 'notes', 'storage']
const MODULE_ICONS = { network: '⌁', clipboard: '#', photos: 'O', camera: '>', terminal: '_', notes: '≡', storage: '⛁' }

// orderedModules returns the known module ids in the saved order, appending any
// the stored order happens to omit so the list is always complete.
function orderedModules(savedOrder) {
  const known = (savedOrder || []).filter((id) => MODULE_IDS.includes(id))
  const missing = MODULE_IDS.filter((id) => !known.includes(id))
  return [...known, ...missing]
}

// moveItem returns a new array with `id` moved to `targetId`'s position.
function moveItem(order, id, targetId) {
  if (id === targetId) return order
  const next = order.filter((item) => item !== id)
  const targetIndex = next.indexOf(targetId)
  if (targetIndex < 0) return order
  next.splice(targetIndex, 0, id)
  return next
}

// ─── Segmented control ────────────────────────────────────────────────────────

function Segmented({ value, options, onChange, ariaLabel }) {
  return (
    <div className="seg" role="group" aria-label={ariaLabel}>
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          className={`seg-btn ${value === option.value ? 'seg-btn-active' : ''}`}
          onClick={() => onChange(option.value)}
          aria-pressed={value === option.value}
        >
          {option.icon && <span className="seg-icon" aria-hidden="true">{option.icon}</span>}
          {option.label}
        </button>
      ))}
    </div>
  )
}

// ─── Toggle switch ────────────────────────────────────────────────────────────

function Toggle({ checked, onChange, label }) {
  return (
    <button
      type="button"
      className={`switch ${checked ? 'switch-on' : ''}`}
      role="switch"
      aria-checked={checked}
      aria-label={label}
      onClick={() => onChange(!checked)}
    >
      <span className="switch-knob" />
    </button>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function ConfigPage() {
  const { settings, updateSettings, t } = useSettings()
  const [isSaving, setIsSaving] = useState(false)
  // Reorder drag-and-drop state for the modules list.
  const [draggingId, setDraggingId] = useState(null)
  const [dropTargetId, setDropTargetId] = useState(null)
  const dragIdRef = useRef(null)

  async function apply(patch) {
    setIsSaving(true)
    try {
      await updateSettings(patch)
    } catch {
      notify.actionFailed(t('config.error'), 'Revisa si el backend está disponible.')
    } finally {
      setIsSaving(false)
    }
  }

  const modules = orderedModules(settings.moduleOrder)

  function handleReorderDrop(targetId) {
    const id = dragIdRef.current
    dragIdRef.current = null
    setDraggingId(null)
    setDropTargetId(null)
    if (!id || id === targetId) return
    const next = moveItem(modules, id, targetId)
    apply({ moduleOrder: next })
  }

  const themeOptions = [
    { value: 'light', label: t('config.theme.light'), icon: '☀' },
    { value: 'dark', label: t('config.theme.dark'), icon: '☾' },
    { value: 'system', label: t('config.theme.system'), icon: '◐' },
  ]
  const fontOptions = [
    { value: 'sans', label: t('config.font.sans') },
    { value: 'serif', label: t('config.font.serif') },
    { value: 'mono', label: t('config.font.mono') },
  ]
  const languageOptions = [
    { value: 'es', label: t('config.language.es') },
    { value: 'en', label: t('config.language.en') },
  ]

  return (
    <div className="config-page">
      <header className="config-header">
        <div>
          <h1>{t('config.title')}</h1>
          <p>{t('config.subtitle')}</p>
        </div>
        {isSaving && <span className="config-saving">{t('config.saving')}</span>}
      </header>

      <section className="config-card">
        <h2 className="config-section-title">{t('config.appearance')}</h2>

        <div className="config-row">
          <div className="config-row-label">
            <span className="config-row-name">{t('config.theme')}</span>
          </div>
          <Segmented value={settings.theme} options={themeOptions} ariaLabel={t('config.theme')} onChange={(v) => apply({ theme: v })} />
        </div>

        <div className="config-row">
          <div className="config-row-label">
            <span className="config-row-name">{t('config.font')}</span>
          </div>
          <Segmented value={settings.font} options={fontOptions} ariaLabel={t('config.font')} onChange={(v) => apply({ font: v })} />
        </div>

        <div className="config-preview">
          <span className="config-preview-tag">{t('config.preview')}</span>
          <p className="config-preview-text">{t('config.previewText')}</p>
          <div className="config-preview-swatches">
            <span className="swatch swatch-accent" />
            <span className="swatch swatch-surface" />
            <span className="swatch swatch-text" />
          </div>
        </div>
      </section>

      <section className="config-card">
        <h2 className="config-section-title">{t('config.language')}</h2>
        <div className="config-row">
          <div className="config-row-label">
            <span className="config-row-name">{t('config.language')}</span>
          </div>
          <Segmented value={settings.language} options={languageOptions} ariaLabel={t('config.language')} onChange={(v) => apply({ language: v })} />
        </div>
      </section>

      <section className="config-card">
        <h2 className="config-section-title">{t('config.modules')}</h2>
        <p className="config-section-desc">{t('config.modules.desc')}</p>
        <ul className="config-modules">
          {modules.map((id) => {
            const enabled = settings.modules?.[id] !== false
            const className = [
              'config-module',
              draggingId === id ? 'config-module-dragging' : '',
              dropTargetId === id ? 'config-module-drop' : '',
            ].filter(Boolean).join(' ')
            return (
              <li
                key={id}
                className={className}
                draggable
                onDragStart={(event) => {
                  dragIdRef.current = id
                  setDraggingId(id)
                  event.dataTransfer.effectAllowed = 'move'
                  event.dataTransfer.setData('text/plain', id)
                }}
                onDragEnd={() => { dragIdRef.current = null; setDraggingId(null); setDropTargetId(null) }}
                onDragOver={(event) => {
                  if (!dragIdRef.current || dragIdRef.current === id) return
                  event.preventDefault()
                  event.dataTransfer.dropEffect = 'move'
                  setDropTargetId(id)
                }}
                onDragLeave={() => setDropTargetId((current) => (current === id ? null : current))}
                onDrop={(event) => { event.preventDefault(); handleReorderDrop(id) }}
              >
                <span className="config-module-handle" aria-hidden="true" title={t('config.reorder')}>⠿</span>
                <span className="config-module-icon" aria-hidden="true">{MODULE_ICONS[id]}</span>
                <span className="config-module-name">{t(`nav.${id}`)}</span>
                <Toggle checked={enabled} label={t(`nav.${id}`)} onChange={(value) => apply({ modules: { [id]: value } })} />
              </li>
            )
          })}
        </ul>
      </section>
    </div>
  )
}
