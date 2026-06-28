const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  })
  const data = await response.json()
  if (!response.ok) throw new Error(data.message || 'No se pudo completar la solicitud.')
  return data
}

export function getNetworkSnapshot() {
  return request('/api/v1/network/snapshot')
}

export function updateNetworkDevice(id, patch) {
  return request(`/api/v1/network/devices/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(patch),
  })
}

