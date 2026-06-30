const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

// Search types offered in the UI. The id maps to Spotify's `type` parameter;
// "all" sends every type so a single query covers artists, songs and more.
export const MUSIC_SEARCH_TYPES = [
  { id: 'all', label: 'Todo', spotifyType: 'artist,album,track,playlist' },
  { id: 'artist', label: 'Artistas', spotifyType: 'artist' },
  { id: 'track', label: 'Canciones', spotifyType: 'track' },
  { id: 'album', label: 'Álbumes', spotifyType: 'album' },
  { id: 'playlist', label: 'Playlists', spotifyType: 'playlist' },
]

export async function searchMusic(query, { type = 'all', limit = 12 } = {}) {
  const term = query.trim()
  if (!term) return []

  const config = MUSIC_SEARCH_TYPES.find((item) => item.id === type) || MUSIC_SEARCH_TYPES[0]
  const params = new URLSearchParams({
    q: term,
    type: config.spotifyType,
    limit: String(limit),
  })

  const response = await fetch(`${API_BASE_URL}/api/v1/music/search?${params.toString()}`)
  if (!response.ok) {
    let message = 'No se pudo buscar en Spotify.'
    try {
      const data = await response.json()
      if (data?.message) message = data.message
    } catch {
      // Keep the default message if the body is not JSON.
    }
    throw new Error(message)
  }

  const data = await response.json()
  return data.items || []
}
