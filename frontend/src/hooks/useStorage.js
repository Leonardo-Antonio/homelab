import { useCallback, useEffect, useState } from 'react'
import {
  createFolder,
  deleteNode,
  listFolder,
  moveNode,
  renameNode,
  uploadFile,
} from '../services/storageApi.js'

// useStorage drives the Drive-like browser: it owns the current folder, its
// children, the breadcrumb trail and in-flight uploads. The backend is the
// single source of truth — every mutation refetches the current folder so the
// view can never drift from persisted state.
export function useStorage() {
  const [currentId, setCurrentId] = useState(null)
  const [items, setItems] = useState([])
  const [breadcrumb, setBreadcrumb] = useState([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [uploads, setUploads] = useState([])
  const [reloadToken, setReloadToken] = useState(0)

  // Refetches the current folder. Mutations bump reloadToken instead of calling
  // setState during render/effects, keeping the backend the source of truth.
  useEffect(() => {
    let active = true

    async function loadFolder() {
      if (active) setIsLoading(true)
      try {
        const data = await listFolder(currentId)
        if (!active) return
        setItems(Array.isArray(data.items) ? data.items : [])
        setBreadcrumb(Array.isArray(data.breadcrumb) ? data.breadcrumb : [])
        setError('')
      } catch {
        if (active) setError('No se pudo cargar el almacenamiento desde el backend.')
      } finally {
        if (active) setIsLoading(false)
      }
    }

    loadFolder()

    return () => {
      active = false
    }
  }, [currentId, reloadToken])

  const reload = useCallback(() => setReloadToken((token) => token + 1), [])

  const open = useCallback((folderId) => setCurrentId(folderId ?? null), [])

  const refresh = reload

  const addFolder = useCallback(async (name) => {
    await createFolder({ parentId: currentId, name })
    reload()
  }, [currentId, reload])

  const rename = useCallback(async (id, name) => {
    await renameNode(id, name)
    reload()
  }, [reload])

  const move = useCallback(async (id, parentId) => {
    await moveNode(id, parentId)
    reload()
  }, [reload])

  const remove = useCallback(async (id) => {
    await deleteNode(id)
    reload()
  }, [reload])

  // upload accepts a FileList/array and uploads sequentially, tracking
  // per-file progress so the UI can show a live list.
  const upload = useCallback(async (files) => {
    const list = Array.from(files)
    if (list.length === 0) return

    const entries = list.map((file) => ({
      key: `${file.name}-${file.size}-${Math.random().toString(36).slice(2)}`,
      name: file.name,
      progress: 0,
      status: 'pending',
    }))
    setUploads((prev) => [...prev, ...entries])

    const patch = (key, next) =>
      setUploads((prev) => prev.map((u) => (u.key === key ? { ...u, ...next } : u)))

    for (let i = 0; i < list.length; i += 1) {
      const file = list[i]
      const { key } = entries[i]
      patch(key, { status: 'uploading' })
      try {
        await uploadFile({
          parentId: currentId,
          file,
          onProgress: (ratio) => patch(key, { progress: ratio }),
        })
        patch(key, { status: 'done', progress: 1 })
      } catch {
        patch(key, { status: 'error' })
        throw new Error(`No se pudo subir "${file.name}".`)
      } finally {
        // Drop the finished entry after a short delay so the user sees it land.
        setTimeout(() => setUploads((prev) => prev.filter((u) => u.key !== key)), 1500)
      }
    }

    reload()
  }, [currentId, reload])

  return {
    currentId,
    items,
    breadcrumb,
    isLoading,
    error,
    uploads,
    open,
    refresh,
    addFolder,
    rename,
    move,
    remove,
    upload,
  }
}
