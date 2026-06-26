import { useCallback, useEffect, useState } from 'react'
import { createNote, deleteNote, getNote, listNotes, updateNote } from '../services/notesApi.js'

export function buildTree(nodes, parentId = null) {
  return nodes
    .filter((n) => (n.parentId ?? null) === parentId)
    .sort((a, b) => {
      if (a.type !== b.type) return a.type === 'dir' ? -1 : 1
      return a.name.localeCompare(b.name)
    })
    .map((n) => ({ ...n, children: n.type === 'dir' ? buildTree(nodes, n.id) : [] }))
}

function collectSubtree(nodes, rootId) {
  const ids = new Set([rootId])
  let changed = true
  while (changed) {
    changed = false
    for (const n of nodes) {
      if (!ids.has(n.id) && n.parentId && ids.has(n.parentId)) {
        ids.add(n.id)
        changed = true
      }
    }
  }
  return ids
}

export function useNotes() {
  const [nodes, setNodes] = useState([])
  const [selectedId, setSelectedId] = useState(null)
  const [activeNote, setActiveNote] = useState(null)
  const [isLoadingTree, setIsLoadingTree] = useState(true)
  const [isLoadingNote, setIsLoadingNote] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    let isMounted = true

    async function load() {
      try {
        const data = await listNotes()
        if (isMounted) {
          setNodes(Array.isArray(data) ? data : [])
          setError('')
        }
      } catch {
        if (isMounted) setError('No se pudo cargar las notas desde el backend.')
      } finally {
        if (isMounted) setIsLoadingTree(false)
      }
    }

    load()
    return () => { isMounted = false }
  }, [])

  const selectNote = useCallback(async (id) => {
    setSelectedId(id)
    setIsLoadingNote(true)
    try {
      const detail = await getNote(id)
      setActiveNote(detail)
    } catch {
      setActiveNote(null)
    } finally {
      setIsLoadingNote(false)
    }
  }, [])

  const createNode = useCallback(async (parentId, type, name) => {
    const node = await createNote({ parentId, type, name })
    setNodes((prev) => [...prev, node])
    return node
  }, [])

  const saveNote = useCallback(async (id, { name, content, parentId }) => {
    const node = await updateNote(id, { name, content, parentId })
    setNodes((prev) => prev.map((n) => (n.id === id ? node : n)))
    setActiveNote((prev) => (prev?.id === id ? { ...node, content } : prev))
    return node
  }, [])

  const renameNode = useCallback(async (id, name) => {
    const current = nodes.find((n) => n.id === id)
    if (!current) return
    const node = await updateNote(id, {
      name,
      content: activeNote?.id === id ? (activeNote.content ?? '') : '',
      parentId: current.parentId,
    })
    setNodes((prev) => prev.map((n) => (n.id === id ? node : n)))
    if (activeNote?.id === id) setActiveNote((prev) => ({ ...prev, ...node }))
    return node
  }, [nodes, activeNote])

  const deleteNode = useCallback(async (id) => {
    await deleteNote(id)
    const subtree = collectSubtree(nodes, id)
    setNodes((prev) => prev.filter((n) => !subtree.has(n.id)))
    if (subtree.has(selectedId)) {
      setSelectedId(null)
      setActiveNote(null)
    }
  }, [nodes, selectedId])

  return {
    tree: buildTree(nodes),
    nodes,
    selectedId,
    activeNote,
    isLoadingTree,
    isLoadingNote,
    error,
    selectNote,
    createNode,
    saveNote,
    renameNode,
    deleteNode,
  }
}
