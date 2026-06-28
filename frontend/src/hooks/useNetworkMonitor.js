import { useCallback, useEffect, useState } from 'react'
import { getNetworkSnapshot, updateNetworkDevice } from '../services/networkApi.js'

const REFRESH_MS = 5000

export function useNetworkMonitor() {
  const [snapshot, setSnapshot] = useState({ overview: null, devices: [], visits: [] })
  const [isLoading, setIsLoading] = useState(true)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [error, setError] = useState('')
  const [reloadToken, setReloadToken] = useState(0)

  const refresh = useCallback(() => setReloadToken((token) => token + 1), [])

  useEffect(() => {
    let active = true
    let timer

    async function load({ quiet = false } = {}) {
      if (!quiet && active) setIsLoading(true)
      if (quiet && active) setIsRefreshing(true)
      try {
        const data = await getNetworkSnapshot()
        if (!active) return
        setSnapshot({
          overview: data.overview || null,
          devices: Array.isArray(data.devices) ? data.devices : [],
          visits: Array.isArray(data.visits) ? data.visits : [],
        })
        setError('')
      } catch {
        if (active) setError('No se pudo leer el estado actual de la red.')
      } finally {
        if (active) {
          setIsLoading(false)
          setIsRefreshing(false)
          timer = window.setTimeout(() => load({ quiet: true }), REFRESH_MS)
        }
      }
    }

    load()
    return () => {
      active = false
      window.clearTimeout(timer)
    }
  }, [reloadToken])

  const updateDevice = useCallback(async (id, patch) => {
    await updateNetworkDevice(id, patch)
    refresh()
  }, [refresh])

  return {
    ...snapshot,
    isLoading,
    isRefreshing,
    error,
    refresh,
    updateDevice,
  }
}

