import { useEffect, useState } from 'react'
import {
  createClipboardItem,
  deleteAllClipboardItems,
  deleteClipboardItem,
  listClipboardItems,
} from '../services/clipboardApi.js'

const DEFAULT_PAGE_SIZE = 15

function formatDate(value) {
  return new Intl.DateTimeFormat('es', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

function normalizeItems(value) {
  if (!Array.isArray(value)) {
    return []
  }

  return value
    .filter((item) => typeof item?.id === 'string' && typeof item?.text === 'string')
    .map((item) => ({
      id: item.id,
      text: item.text,
      createdAt: item.createdAt || new Date().toISOString(),
    }))
}

export function useClipboardItems() {
  const [items, setItems] = useState([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [reloadToken, setReloadToken] = useState(0)
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
    pages: 0,
    total: 0,
    hasNext: false,
    hasPrevious: false,
  })

  useEffect(() => {
    let isMounted = true

    async function loadItems() {
      try {
        if (isMounted) {
          setIsLoading(true)
        }

        const response = await listClipboardItems({
          page: pagination.page,
          pageSize: pagination.pageSize,
        })
        if (isMounted) {
          setItems(normalizeItems(response.items))
          setPagination((currentPagination) => ({
            ...currentPagination,
            page: response.page,
            pageSize: response.pageSize,
            pages: response.pages,
            total: response.total,
            hasNext: response.hasNext,
            hasPrevious: response.hasPrevious,
          }))
          setError('')
        }
      } catch {
        if (isMounted) {
          setError('No se pudo cargar la lista desde el backend.')
        }
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    loadItems()

    return () => {
      isMounted = false
    }
  }, [pagination.page, pagination.pageSize, reloadToken])

  async function addItem(text) {
    await createClipboardItem(text)
    setPagination((currentPagination) => ({
      ...currentPagination,
      page: 1,
    }))
    setReloadToken((currentToken) => currentToken + 1)
  }

  async function removeItem(itemId) {
    await deleteClipboardItem(itemId)
    setItems((currentItems) => currentItems.filter((item) => item.id !== itemId))
    setPagination((currentPagination) => {
      const nextTotal = Math.max(currentPagination.total - 1, 0)
      const nextPages = Math.ceil(nextTotal / currentPagination.pageSize)
      const nextPage = Math.min(currentPagination.page, Math.max(nextPages, 1))

      return {
        ...currentPagination,
        page: nextPage,
        pages: nextPages,
        total: nextTotal,
        hasNext: nextPage < nextPages,
        hasPrevious: nextPage > 1 && nextPages > 0,
      }
    })
    setReloadToken((currentToken) => currentToken + 1)
  }

  async function clearItems() {
    await deleteAllClipboardItems()
    setItems([])
    setPagination((currentPagination) => ({
      ...currentPagination,
      page: 1,
      pages: 0,
      total: 0,
      hasNext: false,
      hasPrevious: false,
    }))
  }

  function goToNextPage() {
    setPagination((currentPagination) => ({
      ...currentPagination,
      page: currentPagination.hasNext ? currentPagination.page + 1 : currentPagination.page,
    }))
  }

  function goToPreviousPage() {
    setPagination((currentPagination) => ({
      ...currentPagination,
      page: currentPagination.hasPrevious ? currentPagination.page - 1 : currentPagination.page,
    }))
  }

  return {
    addItem,
    clearItems,
    error,
    goToNextPage,
    goToPreviousPage,
    isLoading,
    items: items.map((item) => ({
      ...item,
      createdAtLabel: formatDate(item.createdAt),
    })),
    pagination,
    removeItem,
  }
}
