import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { getSettings, saveSettings } from '../services/settingsApi.js'
import { createTranslator } from '../i18n.js'

const MODULE_ORDER = ['network', 'clipboard', 'photos', 'cinema', 'camera', 'terminal', 'notes', 'storage']

const DEFAULT_SETTINGS = {
  theme: 'light',
  language: 'es',
  font: 'sans',
  modules: {
    network: true,
    clipboard: true,
    photos: true,
    cinema: true,
    camera: true,
    terminal: true,
    notes: true,
    storage: true,
  },
  moduleOrder: MODULE_ORDER,
}

const SettingsContext = createContext(null)

// resolveTheme turns the stored preference into the concrete theme to paint,
// following the OS setting when "system" is chosen.
function resolveTheme(theme) {
  if (theme === 'system') {
    if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark'
    }
    return 'light'
  }
  return theme === 'dark' ? 'dark' : 'light'
}

// applyToDocument reflects the active preferences onto <html> so the global CSS
// (data-theme / data-font) and the browser (lang) pick them up.
function applyToDocument(settings) {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.dataset.theme = resolveTheme(settings.theme)
  root.dataset.font = settings.font
  root.lang = settings.language
}

// withDefaults backfills any missing field so the document is always complete.
function withDefaults(value) {
  const order = Array.isArray(value?.moduleOrder) && value.moduleOrder.length
    ? value.moduleOrder
    : DEFAULT_SETTINGS.moduleOrder
  return {
    ...DEFAULT_SETTINGS,
    ...value,
    modules: { ...DEFAULT_SETTINGS.modules, ...(value?.modules || {}) },
    moduleOrder: order,
  }
}

export function SettingsProvider({ children }) {
  const [settings, setSettings] = useState(DEFAULT_SETTINGS)
  const [isLoading, setIsLoading] = useState(true)
  // Mirror of the latest settings so updateSettings never reads a stale closure.
  const settingsRef = useRef(settings)

  // Load persisted settings from the backend once on startup.
  useEffect(() => {
    let active = true
    async function load() {
      try {
        const stored = await getSettings()
        if (active && stored) setSettings(withDefaults(stored))
      } catch {
        // Keep defaults if the backend is unavailable.
      } finally {
        if (active) setIsLoading(false)
      }
    }
    load()
    return () => { active = false }
  }, [])

  // Apply preferences to <html> and refresh the ref whenever they change.
  useEffect(() => {
    settingsRef.current = settings
    applyToDocument(settings)
  }, [settings])

  // Re-apply when the OS theme flips while "system" is selected.
  useEffect(() => {
    if (settings.theme !== 'system' || typeof window === 'undefined') return
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => applyToDocument(settings)
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [settings])

  // updateSettings merges a patch, applies it optimistically (so the theme
  // flips instantly), then persists. On failure it rolls back so the UI never
  // drifts from the server.
  const updateSettings = useCallback(async (patch) => {
    const previous = settingsRef.current
    const merged = {
      ...previous,
      ...patch,
      modules: { ...previous.modules, ...(patch.modules || {}) },
    }
    setSettings(merged)

    try {
      const saved = await saveSettings(merged)
      setSettings(withDefaults(saved))
      return true
    } catch (error) {
      setSettings(previous)
      throw error
    }
  }, [])

  const t = useMemo(() => createTranslator(settings.language), [settings.language])

  const value = useMemo(
    () => ({ settings, isLoading, updateSettings, t }),
    [settings, isLoading, updateSettings, t],
  )

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useSettings() {
  const context = useContext(SettingsContext)
  if (!context) {
    throw new Error('useSettings must be used within a SettingsProvider')
  }
  return context
}
