const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  })

  if (response.status === 204) return null

  const data = await response.json()
  if (!response.ok) throw new Error(data.message || 'No se pudo completar la solicitud.')

  return data
}

export function listNotes() {
  return request('/api/v1/notes')
}

export function getNote(id) {
  return request(`/api/v1/notes/${id}`)
}

export function createNote({ parentId = null, type, name, content = '' }) {
  return request('/api/v1/notes', {
    method: 'POST',
    body: JSON.stringify({ parentId, type, name, content }),
  })
}

export function updateNote(id, { name, content, parentId }) {
  return request(`/api/v1/notes/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ name, content, parentId: parentId ?? null }),
  })
}

export function deleteNote(id) {
  return request(`/api/v1/notes/${id}`, { method: 'DELETE' })
}
