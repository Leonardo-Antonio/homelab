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

export function getSettings() {
  return request('/api/v1/settings')
}

export function saveSettings(settings) {
  return request('/api/v1/settings', {
    method: 'PUT',
    body: JSON.stringify(settings),
  })
}
