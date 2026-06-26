const navItems = [
  { id: 'clipboard', label: 'Clipboard', icon: '#' },
  { id: 'photos', label: 'Fotos', icon: 'O' },
  { id: 'camera', label: 'Camara', icon: '>' },
  { id: 'terminal', label: 'Terminal', icon: '_' },
  { id: 'notes', label: 'Notas', icon: '≡' },
]

export function AppShell({ activePage, children, onNavigate }) {
  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="Main navigation">
        <a className="brand" href="/" aria-label="HomeLab home">
          <span className="brand-mark" aria-hidden="true">
            HL
          </span>
          <span>
            <strong>HomeLab</strong>
            <small>Personal utilities</small>
          </span>
        </a>

        <nav className="nav-list">
          {navItems.map((item) => (
            <button
              className={`nav-item ${activePage === item.id ? 'nav-item-active' : ''}`}
              key={item.id}
              type="button"
              onClick={() => onNavigate(item.id)}
            >
              <span aria-hidden="true">{item.icon}</span>
              {item.label}
            </button>
          ))}
        </nav>
      </aside>

      <main className="main-content">{children}</main>
    </div>
  )
}
