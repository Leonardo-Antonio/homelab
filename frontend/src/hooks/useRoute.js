import { useCallback, useEffect, useState } from 'react'
import { pageToPath, pathToPage } from '../router.js'

// Lightweight path-based routing on top of the History API: keeps the active
// page in sync with the URL and supports the browser back/forward buttons.
export function useRoute() {
  const [page, setPage] = useState(() => pathToPage(window.location.pathname))

  useEffect(() => {
    function syncFromLocation() {
      setPage(pathToPage(window.location.pathname))
    }

    window.addEventListener('popstate', syncFromLocation)
    return () => window.removeEventListener('popstate', syncFromLocation)
  }, [])

  const navigate = useCallback((nextPage) => {
    const path = pageToPath(nextPage)
    if (window.location.pathname !== path) {
      window.history.pushState({}, '', path)
    }
    setPage(nextPage)
  }, [])

  return { page, navigate }
}
