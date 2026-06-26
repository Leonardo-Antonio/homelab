import { Toaster } from 'sileo'
import { AppShell } from './components/AppShell.jsx'
import { useRoute } from './hooks/useRoute.js'
import { CameraStreamPage } from './pages/CameraStreamPage/CameraStreamPage.jsx'
import { ClipboardPage } from './pages/ClipboardPage/ClipboardPage.jsx'
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
}

function getStandaloneView() {
  if (typeof window === 'undefined') {
    return null
  }

  return new URLSearchParams(window.location.search).get('view')
}

function App() {
  const { page, navigate } = useRoute()
  const isStandaloneTerminal = getStandaloneView() === STANDALONE_VIEW

  return (
    <>
      <Toaster
        position="top-right"
        offset={{ top: 18, right: 18 }}
        theme="light"
        options={{
          duration: 4200,
          roundness: 14,
        }}
      />
      {isStandaloneTerminal ? (
        <TerminalPage standalone />
      ) : (
        <AppShell activePage={page} onNavigate={navigate}>
          {pages[page]}
        </AppShell>
      )}
    </>
  )
}

export default App
