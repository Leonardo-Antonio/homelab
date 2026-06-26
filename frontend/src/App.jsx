import { Toaster } from 'sileo'
import { AppShell } from './components/AppShell.jsx'
import { useSettings } from './context/SettingsContext.jsx'
import { useRoute } from './hooks/useRoute.js'
import { CameraStreamPage } from './pages/CameraStreamPage/CameraStreamPage.jsx'
import { ClipboardPage } from './pages/ClipboardPage/ClipboardPage.jsx'
import { ConfigPage } from './pages/ConfigPage/ConfigPage.jsx'
import { NotesPage } from './pages/NotesPage/NotesPage.jsx'
import { PhotosPage } from './pages/PhotosPage/PhotosPage.jsx'
import { StoragePage } from './pages/StoragePage/StoragePage.jsx'
import { STANDALONE_VIEW, TerminalPage } from './pages/TerminalPage/TerminalPage.jsx'
import './App.css'

const pages = {
  clipboard: <ClipboardPage />,
  photos: <PhotosPage />,
  camera: <CameraStreamPage />,
  terminal: <TerminalPage />,
  notes: <NotesPage />,
  storage: <StoragePage />,
  config: <ConfigPage />,
}

function getStandaloneView() {
  if (typeof window === 'undefined') {
    return null
  }

  return new URLSearchParams(window.location.search).get('view')
}

function App() {
  const { page, navigate } = useRoute()
  const { settings } = useSettings()
  const isStandaloneTerminal = getStandaloneView() === STANDALONE_VIEW

  // If the active page belongs to a disabled module, fall back to Settings so
  // the user is never stranded on a hidden section.
  const isDisabledModule = settings.modules?.[page] === false
  const activePage = isDisabledModule ? 'config' : page

  return (
    <>
      <Toaster
        position="top-right"
        offset={{ top: 18, right: 18 }}
        theme={settings.theme === 'dark' ? 'dark' : 'light'}
        options={{
          duration: 4200,
          roundness: 14,
        }}
      />
      {isStandaloneTerminal ? (
        <TerminalPage standalone />
      ) : (
        <AppShell activePage={activePage} onNavigate={navigate}>
          {pages[activePage] ?? pages.config}
        </AppShell>
      )}
    </>
  )
}

export default App
