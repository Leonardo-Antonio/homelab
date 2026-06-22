import { Toaster } from 'sileo'
import { useState } from 'react'
import { AppShell } from './components/AppShell.jsx'
import { CameraStreamPage } from './pages/CameraStreamPage/CameraStreamPage.jsx'
import { ClipboardPage } from './pages/ClipboardPage/ClipboardPage.jsx'
import { PhotosPage } from './pages/PhotosPage/PhotosPage.jsx'
import './App.css'

const pages = {
  clipboard: <ClipboardPage />,
  photos: <PhotosPage />,
  camera: <CameraStreamPage />,
}

function App() {
  const [activePage, setActivePage] = useState('clipboard')

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
      <AppShell activePage={activePage} onNavigate={setActivePage}>
        {pages[activePage]}
      </AppShell>
    </>
  )
}

export default App
