import { useCallback, useEffect, useRef, useState } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { fetchTerminalInfo, openTerminalSocket } from '../services/terminalSocket.js'

const TERMINAL_THEME = {
  background: '#171717',
  foreground: '#fffefa',
  cursor: '#f0c15a',
}

/**
 * Owns the xterm instance and its WebSocket bridge for a single terminal view.
 * The terminal is mounted into the returned containerRef. Status reflects the
 * SSH session lifecycle so the page can render a connection indicator.
 */
export function useTerminalSession() {
  const containerRef = useRef(null)
  const terminalRef = useRef(null)
  const fitAddonRef = useRef(null)
  const connectionRef = useRef(null)
  const [status, setStatus] = useState('connecting')
  const [info, setInfo] = useState(null)
  const [attempt, setAttempt] = useState(0)

  const reconnect = useCallback(() => {
    setStatus('connecting')
    setAttempt((current) => current + 1)
  }, [])

  useEffect(() => {
    let isMounted = true
    fetchTerminalInfo()
      .then((value) => {
        if (isMounted) {
          setInfo(value)
        }
      })
      .catch(() => {
        if (isMounted) {
          setInfo(null)
        }
      })

    return () => {
      isMounted = false
    }
  }, [])

  useEffect(() => {
    if (!containerRef.current) {
      return undefined
    }

    const terminal = new Terminal({
      convertEol: false,
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
      theme: TERMINAL_THEME,
    })
    const fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    terminal.open(containerRef.current)
    fitAddon.fit()

    terminalRef.current = terminal
    fitAddonRef.current = fitAddon

    const connection = openTerminalSocket({
      onOutput: (bytes) => terminal.write(bytes),
      onStatus: (message) => {
        if (message.type === 'status' && message.state === 'connected') {
          setStatus('connected')
          fitAddon.fit()
          connection.sendResize(terminal.cols, terminal.rows)
        }
        if (message.type === 'error') {
          terminal.writeln(`\r\n\x1b[31m${message.message || 'Error de conexion.'}\x1b[0m`)
        }
      },
      onClose: () => setStatus('disconnected'),
    })
    connectionRef.current = connection

    const inputSubscription = terminal.onData((data) => connection.sendInput(data))

    const resizeObserver = new ResizeObserver(() => {
      try {
        fitAddon.fit()
        connection.sendResize(terminal.cols, terminal.rows)
      } catch {
        // The container may be detached mid-resize; ignore.
      }
    })
    resizeObserver.observe(containerRef.current)

    return () => {
      resizeObserver.disconnect()
      inputSubscription.dispose()
      connection.close()
      terminal.dispose()
      terminalRef.current = null
      fitAddonRef.current = null
      connectionRef.current = null
    }
  }, [attempt])

  return { containerRef, status, info, reconnect }
}
