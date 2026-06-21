const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  })

  if (response.status === 204) {
    return null
  }

  const data = await response.json()
  if (!response.ok) {
    throw new Error(data.message || 'No se pudo completar la solicitud.')
  }

  return data
}

export async function listClipboardItems({ page = 1, pageSize = 15 } = {}) {
  const params = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  })

  return request(`/api/v1/clipboard-items?${params.toString()}`)
}

export async function createClipboardItem(text) {
  return request('/api/v1/clipboard-items', {
    method: 'POST',
    body: JSON.stringify({ text }),
  })
}

export async function deleteClipboardItem(itemId) {
  return request(`/api/v1/clipboard-items/${itemId}`, {
    method: 'DELETE',
  })
}

export async function deleteAllClipboardItems() {
  return request('/api/v1/clipboard-items', {
    method: 'DELETE',
  })
}
