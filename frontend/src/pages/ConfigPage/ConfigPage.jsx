import { useState } from 'react'
import { useSettings } from '../../context/SettingsContext.jsx'
import { notify } from '../../services/notifications.js'
import './ConfigPage.css'

const MODULE_IDS = ['clipboard', 'photos', 'camera', 'terminal', 'notes', 'storage']
const MODULE_ICONS = { clipboard: '#', photos: 'O', camera: '>', terminal: '_', notes: '≡', storage: '⛁' }

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
          {MODULE_IDS.map((id) => {
            const enabled = settings.modules?.[id] !== false
            return (
              <li key={id} className="config-module">
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
