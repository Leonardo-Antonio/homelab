const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''
const ARCHIVE_SEARCH_URL = 'https://archive.org/advancedsearch.php'
const ARCHIVE_METADATA_URL = 'https://archive.org/metadata'
const ARCHIVE_DOWNLOAD_URL = 'https://archive.org/download'
const DAILYMOTION_SEARCH_URL = 'https://api.dailymotion.com/videos'
const ITUNES_SEARCH_URL = 'https://itunes.apple.com/search'

// To add a source, append one object here. The search function receives
// (term, limit) and must return createMovieResult(...) items.
export const MOVIE_SOURCE_CONFIGS = [
  {
    id: 'archive',
    label: 'Archive.org',
    enabled: true,
    limit: 10,
    search: searchArchiveMovies,
  },
  {
    id: 'dailymotion',
    label: 'Dailymotion',
    enabled: true,
    limit: 8,
    search: searchDailymotionVideos,
  },
  {
    id: 'itunes',
    label: 'Apple / iTunes',
    enabled: true,
    limit: 10,
    search: searchItunesMovies,
  },
  {
    id: 'itunes-tv',
    label: 'Series (Apple)',
    enabled: true,
    limit: 10,
    search: searchItunesTvShows,
  },
  {
    id: 'cuevana',
    label: 'Cuevana',
    enabled: true,
    limit: 12,
    search: (term, limit) => searchCinemaSource('cuevana', term, limit),
  },
  {
    id: 'pelisplus',
    label: 'PelisPlus',
    enabled: true,
    limit: 12,
    search: (term, limit) => searchCinemaSource('pelisplus', term, limit),
  },
]

export function createMovieResult({
  id,
  source,
  sourceLabel,
  playbackType = 'external',
  title,
  director = '',
  releaseYear = null,
  genre = '',
  rating = '',
  runtime = null,
  posterUrl = '',
  previewUrl = '',
  sourceUrl = '',
  description = '',
}) {
  return {
    id,
    source,
    sourceLabel,
    playbackType,
    title,
    director,
    releaseYear,
    genre,
    rating,
    runtime,
    posterUrl,
    previewUrl,
    sourceUrl,
    description: description || 'Sin descripcion disponible.',
  }
}

function archiveFileUrl(identifier, fileName) {
  const encodedName = fileName.split('/').map(encodeURIComponent).join('/')
  return `${ARCHIVE_DOWNLOAD_URL}/${encodeURIComponent(identifier)}/${encodedName}`
}

function pickPlayableFile(files = []) {
  const mp4Files = files
    .filter((file) => /\.mp4$/i.test(file.name || ''))
    .filter((file) => !/sample|trailer/i.test(file.name || ''))
    .sort((a, b) => Number(b.size || 0) - Number(a.size || 0))

  return mp4Files.find((file) => /h\.264|mpeg4|mp4/i.test(file.format || '')) || mp4Files[0] || null
}

function cleanText(value) {
  return String(value || '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/\s+/g, ' ')
    .trim()
}

function normalizeSearchDoc(doc, metadata = null) {
  const item = metadata?.metadata || {}
  const playableFile = pickPlayableFile(metadata?.files)
  const releaseYear = Number.parseInt(item.year || doc.date || item.date, 10)
  const subjects = Array.isArray(doc.subject) ? doc.subject : item.subject
  const subjectLabel = Array.isArray(subjects) ? subjects.slice(0, 2).join(' / ') : subjects

  return createMovieResult({
    id: `archive:${doc.identifier}`,
    source: 'archive',
    sourceLabel: 'Archive.org',
    playbackType: playableFile ? 'video' : 'external',
    title: item.title || doc.title || doc.identifier,
    director: item.creator || doc.creator || 'Internet Archive',
    releaseYear: Number.isFinite(releaseYear) ? releaseYear : null,
    genre: subjectLabel || 'Archivo audiovisual',
    rating: 'IA',
    runtime: null,
    posterUrl: `https://archive.org/services/img/${encodeURIComponent(doc.identifier)}`,
    previewUrl: playableFile ? archiveFileUrl(doc.identifier, playableFile.name) : '',
    sourceUrl: `https://archive.org/details/${encodeURIComponent(doc.identifier)}`,
    description: cleanText(item.description || doc.description) || 'Sin sinopsis disponible.',
  })
}

function normalizeDailymotionVideo(video) {
  const releaseYear = video.created_time
    ? new Date(video.created_time * 1000).getFullYear()
    : null

  return createMovieResult({
    id: `dailymotion:${video.id}`,
    source: 'dailymotion',
    sourceLabel: 'Dailymotion',
    playbackType: video.embed_url ? 'iframe' : 'external',
    title: video.title || 'Video de Dailymotion',
    director: video['owner.screenname'] || 'Dailymotion',
    releaseYear: Number.isFinite(releaseYear) ? releaseYear : null,
    genre: video.channel || 'Trailer / video',
    rating: 'DM',
    runtime: null,
    posterUrl: video.thumbnail_360_url || '',
    previewUrl: video.embed_url || '',
    sourceUrl: video.url || `https://www.dailymotion.com/video/${video.id}`,
    description: cleanText(video.description) || 'Sin descripcion disponible.',
  })
}

function itunesArtwork(url) {
  // iTunes returns 100x100 thumbnails; upscale the dimensions in the path.
  return String(url || '').replace(/\/\d+x\d+bb\.(jpg|png)$/i, '/600x600bb.$1')
}

function normalizeItunesItem(item, kind) {
  const isTv = kind === 'tv'
  const title = (isTv ? item.collectionName : item.trackName) || item.trackName || 'Sin titulo'
  const releaseYear = item.releaseDate ? new Date(item.releaseDate).getFullYear() : null
  const runtime = item.trackTimeMillis ? Math.round(item.trackTimeMillis / 60000) : null
  const sourceUrl = item.trackViewUrl || item.collectionViewUrl || ''
  const idValue = item.trackId || item.collectionId || encodeURIComponent(title)

  return createMovieResult({
    id: `itunes:${isTv ? 'tv' : 'movie'}:${idValue}`,
    source: isTv ? 'itunes-tv' : 'itunes',
    sourceLabel: isTv ? 'Series (Apple)' : 'Apple / iTunes',
    playbackType: item.previewUrl ? 'video' : 'external',
    title,
    director: item.artistName || 'Apple TV',
    releaseYear: Number.isFinite(releaseYear) ? releaseYear : null,
    genre: item.primaryGenreName || (isTv ? 'Serie' : 'Pelicula'),
    rating: item.contentAdvisoryRating || 'iTunes',
    runtime: isTv ? null : runtime,
    posterUrl: itunesArtwork(item.artworkUrl100),
    previewUrl: item.previewUrl || '',
    sourceUrl,
    description: cleanText(item.longDescription || item.shortDescription) || 'Sin sinopsis disponible.',
  })
}

async function searchItunes({ term, limit, media, entity, kind }) {
  const params = new URLSearchParams({
    term,
    media,
    limit: String(limit),
    country: 'us',
    explicit: 'No',
  })
  if (entity) params.set('entity', entity)

  const response = await fetch(`${ITUNES_SEARCH_URL}?${params.toString()}`)
  if (!response.ok) return []

  const data = await response.json()
  return (data.results || []).map((item) => normalizeItunesItem(item, kind))
}

async function searchItunesMovies(term, limit) {
  return searchItunes({ term, limit, media: 'movie', entity: 'movie', kind: 'movie' })
}

async function searchItunesTvShows(term, limit) {
  return searchItunes({ term, limit, media: 'tvShow', entity: 'tvSeason', kind: 'tv' })
}

function normalizeCinemaItem(item) {
  return createMovieResult({
    id: item.id || `${item.source}:${item.sourceUrl}`,
    source: item.source,
    sourceLabel: item.sourceLabel,
    playbackType: 'external',
    title: item.title || 'Sin titulo',
    director: item.sourceLabel || '',
    releaseYear: Number.isFinite(item.releaseYear) ? item.releaseYear : null,
    genre: item.kind === 'tv' ? 'Serie' : 'Pelicula',
    rating: '',
    runtime: null,
    posterUrl: item.posterUrl || '',
    previewUrl: '',
    sourceUrl: item.sourceUrl || '',
    description: 'Abre la ficha en la fuente para ver opciones de reproduccion.',
  })
}

// Cuevana / PelisPlus and similar sites have no public API and block direct
// browser requests via CORS, so they are scraped server-side by the backend
// cinema proxy. This calls that endpoint for a single source.
async function searchCinemaSource(sourceId, term, limit) {
  const params = new URLSearchParams({
    q: term,
    source: sourceId,
    limit: String(limit),
  })

  const response = await fetch(`${API_BASE_URL}/api/v1/cinema/search?${params.toString()}`)
  if (!response.ok) return []

  const data = await response.json()
  return (data.items || []).map(normalizeCinemaItem)
}

async function fetchMetadata(identifier) {
  const response = await fetch(`${ARCHIVE_METADATA_URL}/${encodeURIComponent(identifier)}`)
  if (!response.ok) return null
  return response.json()
}

export async function searchMovies(query, { limit = 18 } = {}) {
  const term = query.trim()
  if (!term) return []

  const enabledSources = MOVIE_SOURCE_CONFIGS.filter((source) => source.enabled)
  const resultGroups = await Promise.all(
    enabledSources.map(async (source) => {
      try {
        const sourceLimit = Math.min(source.limit || limit, limit)
        return source.search(term, sourceLimit)
      } catch {
        return []
      }
    }),
  )

  return resultGroups.flat()
}

export function getMovieSourceFilters() {
  return [
    { id: 'all', label: 'Todo' },
    ...MOVIE_SOURCE_CONFIGS
      .filter((source) => source.enabled)
      .map((source) => ({ id: source.id, label: source.label })),
  ]
}

async function searchArchiveMovies(term, limit) {
  const docs = await searchArchive(`collection:(moviesandfilms) AND title:(${term})`, limit)
  const fallbackDocs = docs.length
    ? docs
    : await searchArchive(`collection:(moviesandfilms) AND (${term})`, limit)
  const metadataList = await Promise.all(fallbackDocs.map((doc) => fetchMetadata(doc.identifier)))

  return fallbackDocs.map((doc, index) => normalizeSearchDoc(doc, metadataList[index]))
}

async function searchDailymotionVideos(term, limit) {
  const params = new URLSearchParams({
    search: `${term} official trailer`,
    limit: String(limit),
    explicit: 'false',
    fields: [
      'id',
      'title',
      'thumbnail_360_url',
      'description',
      'created_time',
      'channel',
      'owner.screenname',
      'url',
      'embed_url',
    ].join(','),
  })

  const response = await fetch(`${DAILYMOTION_SEARCH_URL}?${params.toString()}`)
  if (!response.ok) return []

  const data = await response.json()
  return (data.list || []).map(normalizeDailymotionVideo)
}

async function searchArchive(archiveQuery, limit) {
  const params = new URLSearchParams({
    q: archiveQuery,
    rows: String(limit),
    page: '1',
    output: 'json',
    sort: 'downloads desc',
  })
  params.append('fl[]', 'identifier')
  params.append('fl[]', 'title')
  params.append('fl[]', 'description')
  params.append('fl[]', 'date')
  params.append('fl[]', 'creator')
  params.append('fl[]', 'subject')

  const response = await fetch(`${ARCHIVE_SEARCH_URL}?${params.toString()}`)
  if (!response.ok) {
    throw new Error('No se pudo consultar el catalogo.')
  }

  const data = await response.json()
  return data.response?.docs || []
}

export function buildLegalSources(movie) {
  const encodedTitle = encodeURIComponent(movie.title)
  const encodedTitleAndYear = encodeURIComponent(`${movie.title} ${movie.releaseYear || ''}`.trim())
  const sources = [
    {
      id: movie.source || 'source',
      label: movie.sourceLabel || 'Fuente',
      url: movie.sourceUrl,
      note: movie.previewUrl ? 'Reproduccion disponible' : 'Ficha original',
    },
    {
      id: 'apple',
      label: 'Apple TV',
      url: `https://tv.apple.com/search?term=${encodedTitle}`,
      note: 'Compra, renta o ficha oficial',
    },
    {
      id: 'justwatch',
      label: 'JustWatch',
      url: `https://www.justwatch.com/us/search?q=${encodedTitle}`,
      note: 'Disponibilidad por proveedor',
    },
    {
      id: 'youtube',
      label: 'YouTube',
      url: `https://www.youtube.com/results?search_query=${encodedTitleAndYear}+official+trailer`,
      note: 'Trailers y canales oficiales',
    },
    {
      id: 'archive',
      label: 'Archive.org',
      url: `https://archive.org/search?query=${encodedTitle}`,
      note: 'Dominio publico y archivos autorizados',
    },
    {
      id: 'themoviedb',
      label: 'TheMovieDB',
      url: `https://www.themoviedb.org/search?query=${encodedTitle}`,
      note: 'Ficha, reparto y proveedores',
    },
    {
      id: 'cuevana',
      label: 'Cuevana',
      url: `https://cuevana.biz/?s=${encodedTitle}`,
      note: 'Buscador externo (enlaces de terceros)',
    },
    {
      id: 'pelisplus',
      label: 'PelisPlus',
      url: `https://pelisplus.to/search?q=${encodedTitle}`,
      note: 'Buscador externo (enlaces de terceros)',
    },
  ]

  return sources.filter((source) => source.url)
}
