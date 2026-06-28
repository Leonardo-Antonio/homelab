const ARCHIVE_SEARCH_URL = 'https://archive.org/advancedsearch.php'
const ARCHIVE_METADATA_URL = 'https://archive.org/metadata'
const ARCHIVE_DOWNLOAD_URL = 'https://archive.org/download'

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
    id: doc.identifier,
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

async function fetchMetadata(identifier) {
  const response = await fetch(`${ARCHIVE_METADATA_URL}/${encodeURIComponent(identifier)}`)
  if (!response.ok) return null
  return response.json()
}

export async function searchMovies(query, { limit = 18 } = {}) {
  const term = query.trim()
  if (!term) return []

  const docs = await searchArchive(`mediatype:(movies) AND title:(${term})`, limit)
  const fallbackDocs = docs.length ? docs : await searchArchive(`mediatype:(movies) AND (${term})`, limit)
  const metadataList = await Promise.all(fallbackDocs.map((doc) => fetchMetadata(doc.identifier)))

  return fallbackDocs.map((doc, index) => normalizeSearchDoc(doc, metadataList[index]))
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
      id: 'apple',
      label: 'Apple TV',
      url: movie.sourceUrl,
      note: movie.previewUrl ? 'Preview oficial disponible' : 'Ficha oficial',
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
