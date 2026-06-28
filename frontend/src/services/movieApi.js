const ARCHIVE_SEARCH_URL = 'https://archive.org/advancedsearch.php'
const ARCHIVE_METADATA_URL = 'https://archive.org/metadata'
const ARCHIVE_DOWNLOAD_URL = 'https://archive.org/download'
const DAILYMOTION_SEARCH_URL = 'https://api.dailymotion.com/videos'

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

  return {
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
  }
}

function normalizeDailymotionVideo(video) {
  const releaseYear = video.created_time
    ? new Date(video.created_time * 1000).getFullYear()
    : null

  return {
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
  }
}

async function fetchMetadata(identifier) {
  const response = await fetch(`${ARCHIVE_METADATA_URL}/${encodeURIComponent(identifier)}`)
  if (!response.ok) return null
  return response.json()
}

export async function searchMovies(query, { limit = 18 } = {}) {
  const term = query.trim()
  if (!term) return []

  const archiveLimit = Math.max(8, Math.ceil(limit * 0.66))
  const dailymotionLimit = Math.max(6, limit - archiveLimit)
  const [archiveMovies, dailymotionVideos] = await Promise.all([
    searchArchiveMovies(term, archiveLimit),
    searchDailymotionVideos(term, dailymotionLimit),
  ])

  return [...archiveMovies, ...dailymotionVideos]
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
  ]

  return sources.filter((source) => source.url)
}
