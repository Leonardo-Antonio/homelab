import { Toaster } from 'sileo'
import { useState } from 'react'
import { AppShell } from './components/AppShell.jsx'
import { CameraStreamPage } from './pages/CameraStreamPage/CameraStreamPage.jsx'
import { ClipboardPage } from './pages/ClipboardPage/ClipboardPage.jsx'
import { PhotosPage } from './pages/PhotosPage/PhotosPage.jsx'
import { STANDALONE_VIEW, TerminalPage } from './pages/TerminalPage/TerminalPage.jsx'
import './App.css'

const pages = {
  clipboard: <ClipboardPage />,
  photos: <PhotosPage />,
  camera: <CameraStreamPage />,
  terminal: <TerminalPage />,
}

function getStandaloneView() {
  if (typeof window === 'undefined') {
    return null
  }

  return new URLSearchParams(window.location.search).get('view')
}

function App() {
  const [activePage, setActivePage] = useState('clipboard')
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
        <AppShell activePage={activePage} onNavigate={setActivePage}>
          {pages[activePage]}
        </AppShell>
      )}
    </>
  )
}

export default App
