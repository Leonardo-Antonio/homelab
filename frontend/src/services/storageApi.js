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

// listFolder returns { parent, breadcrumb, items } for a folder. A null/empty
// parentId browses the root.
export function listFolder(parentId) {
  const query = parentId ? `?parentId=${encodeURIComponent(parentId)}` : ''
  return request(`/api/v1/storage/nodes${query}`)
}

export function createFolder({ parentId = null, name }) {
  return request('/api/v1/storage/folders', {
    method: 'POST',
    body: JSON.stringify({ parentId, name }),
  })
}

export function renameNode(id, name) {
  return request(`/api/v1/storage/nodes/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ name }),
  })
}

export function moveNode(id, parentId) {
  return request(`/api/v1/storage/nodes/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ parentId: parentId ?? null }),
  })
}

export function deleteNode(id) {
  return request(`/api/v1/storage/nodes/${id}`, { method: 'DELETE' })
}

// uploadFile streams a File/Blob to the backend via XHR so we can report
// progress. Resolves with the created node.
export function uploadFile({ parentId = null, file, onProgress } = {}) {
  return new Promise((resolve, reject) => {
    const formData = new FormData()
    if (parentId) formData.append('parentId', parentId)
    formData.append('file', file, file.name)

    const xhr = new XMLHttpRequest()
    xhr.open('POST', `${API_BASE_URL}/api/v1/storage/files`)

    xhr.upload.addEventListener('progress', (event) => {
      if (onProgress && event.lengthComputable) {
        onProgress(event.loaded / event.total)
      }
    })

    xhr.addEventListener('load', () => {
      let payload
      try {
        payload = JSON.parse(xhr.responseText)
      } catch {
        payload = null
      }
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(payload)
      } else {
        reject(new Error(payload?.message || 'No se pudo subir el archivo.'))
      }
    })

    xhr.addEventListener('error', () => reject(new Error('Error de red al subir el archivo.')))
    xhr.addEventListener('abort', () => reject(new Error('Subida cancelada.')))

    xhr.send(formData)
  })
}

// downloadUrl forces a browser download (Content-Disposition: attachment).
export function downloadUrl(node) {
  return `${API_BASE_URL}${node.downloadUrl}?download=1`
}

// contentUrl serves the file inline — used by <img>, <video>, <iframe>, etc.
export function contentUrl(node) {
  return node.downloadUrl ? `${API_BASE_URL}${node.downloadUrl}` : ''
}

// thumbUrl is a small cached preview for image files (empty for everything else).
export function thumbUrl(node) {
  return node.thumbnailUrl ? `${API_BASE_URL}${node.thumbnailUrl}` : ''
}

// fetchTextPreview pulls only the first `maxBytes` of a file via a Range
// request, so previewing a huge log never downloads the whole thing.
export async function fetchTextPreview(node, maxBytes = 256 * 1024) {
  const response = await fetch(`${API_BASE_URL}${node.downloadUrl}`, {
    headers: { Range: `bytes=0-${maxBytes - 1}` },
  })
  if (!response.ok && response.status !== 206) {
    throw new Error('No se pudo leer el archivo.')
  }
  const text = await response.text()
  const truncated = (node.sizeBytes ?? 0) > maxBytes
  return { text, truncated }
}
