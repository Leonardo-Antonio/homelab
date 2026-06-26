// Centralised helpers for classifying storage files so the list and the
// preview modal stay in agreement about how to render and what icon to show.

const EXT_GROUPS = {
  image: ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'avif'],
  video: ['mp4', 'mkv', 'mov', 'webm', 'avi', 'm4v'],
  audio: ['mp3', 'wav', 'flac', 'ogg', 'm4a', 'aac'],
  pdf: ['pdf'],
  archive: ['zip', 'tar', 'gz', 'tgz', 'rar', '7z', 'bz2', 'xz'],
  sheet: ['xls', 'xlsx', 'csv', 'ods'],
  doc: ['doc', 'docx', 'rtf', 'odt'],
  text: ['txt', 'md', 'markdown', 'log', 'rst'],
  code: [
    'js', 'jsx', 'ts', 'tsx', 'go', 'py', 'rs', 'rb', 'php', 'java', 'c', 'h',
    'cpp', 'cs', 'sh', 'bash', 'zsh', 'json', 'yaml', 'yml', 'toml', 'xml',
    'html', 'css', 'scss', 'sql', 'dockerfile', 'env', 'ini', 'conf',
  ],
}

const ICONS = {
  dir: '📁',
  image: '🖼️',
  video: '🎬',
  audio: '🎵',
  pdf: '📕',
  archive: '🗜️',
  sheet: '📊',
  doc: '📄',
  text: '📝',
  code: '🧩',
  other: '📦',
}

function extOf(name = '') {
  const dot = name.lastIndexOf('.')
  return dot >= 0 ? name.slice(dot + 1).toLowerCase() : ''
}

// fileKind returns one of: image | video | audio | pdf | archive | sheet |
// doc | text | code | other. Content type wins; extension is the fallback.
export function fileKind(node) {
  if (node.type === 'dir') return 'dir'
  const ct = (node.contentType || '').toLowerCase()
  if (ct.startsWith('image/')) return 'image'
  if (ct.startsWith('video/')) return 'video'
  if (ct.startsWith('audio/')) return 'audio'
  if (ct === 'application/pdf') return 'pdf'

  const ext = extOf(node.name)
  for (const [kind, list] of Object.entries(EXT_GROUPS)) {
    if (list.includes(ext)) return kind
  }
  if (ct.startsWith('text/')) return 'text'
  return 'other'
}

export function iconFor(node) {
  return ICONS[fileKind(node)] ?? ICONS.other
}

// previewable kinds can be shown inline in the modal.
export function isPreviewable(node) {
  return ['image', 'video', 'audio', 'pdf', 'text', 'code'].includes(fileKind(node))
}

export function formatSize(bytes) {
  if (!bytes || bytes <= 0) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value >= 10 || unit === 0 ? Math.round(value) : value.toFixed(1)} ${units[unit]}`
}

const dateFormatter = new Intl.DateTimeFormat('es', { dateStyle: 'medium', timeStyle: 'short' })

export function formatDate(value) {
  try {
    return dateFormatter.format(new Date(value))
  } catch {
    return ''
  }
}
