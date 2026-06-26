import { useEffect, useState } from 'react'

const navItems = [
  { id: 'clipboard', label: 'Clipboard', icon: '#' },
  { id: 'photos', label: 'Fotos', icon: 'O' },
  { id: 'camera', label: 'Camara', icon: '>' },
  { id: 'terminal', label: 'Terminal', icon: '_' },
  { id: 'notes', label: 'Notas', icon: '≡' },
]

const COLLAPSED_STORAGE_KEY = 'homelab.sidebar.collapsed'

function readCollapsedPreference() {
  if (typeof window === 'undefined') {
    return false
  }

  return window.localStorage.getItem(COLLAPSED_STORAGE_KEY) === 'true'
}

export function AppShell({ activePage, children, onNavigate }) {
  const [collapsed, setCollapsed] = useState(readCollapsedPreference)

  useEffect(() => {
    window.localStorage.setItem(COLLAPSED_STORAGE_KEY, String(collapsed))
  }, [collapsed])

  return (
    <div className={`app-shell ${collapsed ? 'app-shell-collapsed' : ''}`.trim()}>
      <aside className="sidebar" aria-label="Main navigation">
        <a className="brand" href="/" aria-label="HomeLab home">
          <span className="brand-mark" aria-hidden="true">
            HL
          </span>
          <span className="brand-text">
            <strong>HomeLab</strong>
            <small>Personal utilities</small>
          </span>
        </a>

        <button
          className="sidebar-toggle"
          type="button"
          onClick={() => setCollapsed((current) => !current)}
          aria-pressed={collapsed}
          aria-label={collapsed ? 'Expandir menu' : 'Colapsar menu'}
          title={collapsed ? 'Expandir menu' : 'Colapsar menu'}
        >
          <span aria-hidden="true">{collapsed ? '»' : '«'}</span>
          <span className="sidebar-toggle-label">Colapsar</span>
        </button>

        <nav className="nav-list">
          {navItems.map((item) => (
            <button
              className={`nav-item ${activePage === item.id ? 'nav-item-active' : ''}`}
              key={item.id}
              type="button"
              onClick={() => onNavigate(item.id)}
              title={collapsed ? item.label : undefined}
            >
              <span aria-hidden="true">{item.icon}</span>
              <span className="nav-item-label">{item.label}</span>
            </button>
          ))}
        </nav>
      </aside>

      <main className="main-content">{children}</main>
    </div>
  )
}
