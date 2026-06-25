const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

const INFO_PATH = '/api/v1/terminal/info'
const WS_PATH = '/api/v1/terminal/ws'

export async function fetchTerminalInfo() {
  const response = await fetch(`${API_BASE_URL}${INFO_PATH}`)
  if (!response.ok) {
    throw new Error('No se pudo obtener la configuracion del terminal.')
  }

  return response.json()
}

function buildWebSocketUrl() {
  const base = API_BASE_URL || window.location.origin
  const url = new URL(WS_PATH, base)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  return url.toString()
}

/**
 * Opens the terminal WebSocket and adapts the wire sub-protocol:
 * - binary frames are raw PTY output forwarded through onOutput
 * - text frames are JSON status/error messages forwarded through onStatus
 *
 * Returns helpers to send input/resize and to close the socket.
 */
export function openTerminalSocket({ onOutput, onStatus, onClose }) {
  const socket = new WebSocket(buildWebSocketUrl())
  socket.binaryType = 'arraybuffer'

  socket.addEventListener('message', (event) => {
    if (typeof event.data === 'string') {
      try {
        onStatus?.(JSON.parse(event.data))
      } catch {
        // Ignore malformed status frames.
      }
      return
    }

    onOutput?.(new Uint8Array(event.data))
  })

  socket.addEventListener('close', (event) => {
    onClose?.(event)
  })

  function send(message) {
    if (socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(message))
    }
  }

  return {
    socket,
    sendInput(data) {
      send({ type: 'stdin', data })
    },
    sendResize(cols, rows) {
      send({ type: 'resize', cols, rows })
    },
    close() {
      socket.close()
    },
  }
}
