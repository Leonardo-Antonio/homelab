import { useEffect, useState } from 'react'
import { createPhoto, deletePhoto, listPhotos } from '../services/photosApi.js'

const DEFAULT_PAGE_SIZE = 15

function formatDate(value) {
  return new Intl.DateTimeFormat('es', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

function normalizePhotos(value) {
  if (!Array.isArray(value)) {
    return []
  }

  return value.filter((photo) => typeof photo?.id === 'string' && typeof photo?.url === 'string')
}

export function usePhotos() {
  const [photos, setPhotos] = useState([])
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

    async function loadPhotos() {
      try {
        if (isMounted) {
          setIsLoading(true)
        }

        const response = await listPhotos({
          page: pagination.page,
          pageSize: pagination.pageSize,
        })

        if (isMounted) {
          setPhotos(normalizePhotos(response.items))
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
          setError('No se pudo cargar la galeria desde el backend.')
        }
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    loadPhotos()

    return () => {
      isMounted = false
    }
  }, [pagination.page, pagination.pageSize, reloadToken])

  async function savePhoto(blob) {
    await createPhoto(blob)
    setPagination((currentPagination) => ({ ...currentPagination, page: 1 }))
    setReloadToken((currentToken) => currentToken + 1)
  }

  async function removePhoto(photoId) {
    await deletePhoto(photoId)
    setPhotos((currentPhotos) => currentPhotos.filter((photo) => photo.id !== photoId))
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
    error,
    goToNextPage,
    goToPreviousPage,
    isLoading,
    pagination,
    photos: photos.map((photo) => ({
      ...photo,
      createdAtLabel: formatDate(photo.createdAt),
    })),
    removePhoto,
    savePhoto,
  }
}
