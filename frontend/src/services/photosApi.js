const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, options)

  if (response.status === 204) {
    return null
  }

  const data = await response.json()
  if (!response.ok) {
    throw new Error(data.message || 'No se pudo completar la solicitud.')
  }

  return data
}

export function buildPhotoUrl(path) {
  return `${API_BASE_URL}${path}`
}

export function listPhotos({ page = 1, pageSize = 15 } = {}) {
  const params = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  })

  return request(`/api/v1/photos?${params.toString()}`)
}

export function createPhoto(blob) {
  const formData = new FormData()
  formData.append('photo', blob, `homelab-photo-${Date.now()}.jpg`)

  return request('/api/v1/photos', {
    method: 'POST',
    body: formData,
  })
}

export function deletePhoto(photoId) {
  return request(`/api/v1/photos/${photoId}`, {
    method: 'DELETE',
  })
}
