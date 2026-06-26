import { useEffect, useState } from 'react'
import { useSettings } from '../context/SettingsContext.jsx'

// Toggleable modules, in nav order. "config" is appended separately so it can
// never be disabled (you always need a way back to Settings).
const moduleItems = [
  { id: 'clipboard', icon: '#' },
  { id: 'photos', icon: 'O' },
  { id: 'camera', icon: '>' },
  { id: 'terminal', icon: '_' },
  { id: 'notes', icon: '≡' },
  { id: 'storage', icon: '⛁' },
]

const configItem = { id: 'config', icon: '⚙' }

const COLLAPSED_STORAGE_KEY = 'homelab.sidebar.collapsed'

function readCollapsedPreference() {
  if (typeof window === 'undefined') {
    return false
  }

  return window.localStorage.getItem(COLLAPSED_STORAGE_KEY) === 'true'
}

export function AppShell({ activePage, children, onNavigate }) {
  const [collapsed, setCollapsed] = useState(readCollapsedPreference)
  const { settings, t } = useSettings()

  useEffect(() => {
    window.localStorage.setItem(COLLAPSED_STORAGE_KEY, String(collapsed))
  }, [collapsed])

  // Order modules by the saved preference (unknown/missing ids fall back to the
  // declared order), then keep only the enabled ones. Settings is always last.
  const byId = new Map(moduleItems.map((item) => [item.id, item]))
  const orderedIds = [
    ...(settings.moduleOrder || []).filter((id) => byId.has(id)),
    ...moduleItems.map((item) => item.id).filter((id) => !(settings.moduleOrder || []).includes(id)),
  ]
  const visibleItems = [
    ...orderedIds
      .map((id) => byId.get(id))
      .filter((item) => settings.modules?.[item.id] !== false),
    configItem,
  ]

  const collapseLabel = collapsed ? t('sidebar.expand') : t('sidebar.collapse')

  return (
    <div className={`app-shell ${collapsed ? 'app-shell-collapsed' : ''}`.trim()}>
      <aside className="sidebar" aria-label="Main navigation">
        <a className="brand" href="/" aria-label="HomeLab home">
          <span className="brand-mark" aria-hidden="true">
            HL
          </span>
          <span className="brand-text">
            <strong>HomeLab</strong>
            <small>{t('brand.subtitle')}</small>
          </span>
        </a>

        <button
          className="sidebar-toggle"
          type="button"
          onClick={() => setCollapsed((current) => !current)}
          aria-pressed={collapsed}
          aria-label={collapseLabel}
          title={collapseLabel}
        >
          <span aria-hidden="true">{collapsed ? '»' : '«'}</span>
          <span className="sidebar-toggle-label">{collapseLabel}</span>
        </button>

        <nav className="nav-list">
          {visibleItems.map((item) => {
            const label = t(`nav.${item.id}`)
            return (
              <button
                className={`nav-item ${activePage === item.id ? 'nav-item-active' : ''}`}
                key={item.id}
                type="button"
                onClick={() => onNavigate(item.id)}
                title={collapsed ? label : undefined}
              >
                <span aria-hidden="true">{item.icon}</span>
                <span className="nav-item-label">{label}</span>
              </button>
            )
          })}
        </nav>
      </aside>

      <main className="main-content">{children}</main>
    </div>
  )
}
