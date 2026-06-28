import { useMemo, useState } from 'react'
import { useNetworkMonitor } from '../../hooks/useNetworkMonitor.js'
import './NetworkPage.css'

const statusLabels = {
  unknown: 'Nuevo',
  trusted: 'Confiable',
  blocked: 'Bloqueado',
  ignored: 'Ignorado',
}

const statusOptions = [
  { value: 'all', label: 'Todos' },
  { value: 'unknown', label: 'Nuevos' },
  { value: 'trusted', label: 'Confiables' },
  { value: 'blocked', label: 'Bloqueados' },
]

function formatTime(value) {
  if (!value) return 'sin datos'
  return new Intl.DateTimeFormat('es', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(value))
}

function SourcePill({ source }) {
  return (
    <span className={`source-pill ${source.available ? 'source-on' : 'source-off'}`}>
      {source.name}
    </span>
  )
}

function DeviceActions({ device, onUpdate }) {
  return (
    <div className="device-actions" aria-label={`Acciones para ${device.name}`}>
      <button type="button" onClick={() => onUpdate(device.id, { status: 'trusted' })}>
        Confiar
      </button>
      <button type="button" onClick={() => onUpdate(device.id, { status: 'blocked' })}>
        Bloquear
      </button>
      <button type="button" onClick={() => onUpdate(device.id, { status: 'ignored' })}>
        Ocultar
      </button>
    </div>
  )
}

export function NetworkPage() {
  const { overview, devices, visits, isLoading, isRefreshing, error, refresh, updateDevice } = useNetworkMonitor()
  const [query, setQuery] = useState('')
  const [status, setStatus] = useState('all')

  const visibleDevices = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return devices.filter((device) => {
      if (device.status === 'ignored') return false
      if (status !== 'all' && device.status !== status) return false
      if (!needle) return true
      return [device.name, device.ip, device.mac, device.interface, device.source]
        .filter(Boolean)
        .some((value) => value.toLowerCase().includes(needle))
    })
  }, [devices, query, status])

  const filteredVisits = useMemo(() => {
    const needle = query.trim().toLowerCase()
    if (!needle) return visits
    return visits.filter((visit) => [visit.domain, visit.clientIp, visit.device]
      .filter(Boolean)
      .some((value) => value.toLowerCase().includes(needle)))
  }, [query, visits])

  return (
    <div className="network-page">
      <header className="network-hero">
        <div>
          <span className="network-kicker">Monitoreo en tiempo actual</span>
          <h1>Estado de red</h1>
          <p>
            Dispositivos vistos por ARP/DHCP, dominios leídos desde DNS y decisiones locales para reconocer, bloquear u ocultar equipos.
          </p>
        </div>
        <button className="network-live-badge" type="button" onClick={refresh}>
          <span />
          {isRefreshing ? 'Actualizando' : 'Live'}
        </button>
      </header>

      {error && <div className="network-error">{error}</div>}

      <section className="network-stats" aria-label="Resumen de red">
        <article className="network-stat network-stat-good">
          <span>Dispositivos activos</span>
          <strong>{overview?.devicesOnline ?? 0}</strong>
          <small>{overview?.devicesTotal ?? 0} registrados</small>
        </article>
        <article className="network-stat network-stat-warn">
          <span>No reconocidos</span>
          <strong>{overview?.devicesUnknown ?? 0}</strong>
          <small>requieren revisión</small>
        </article>
        <article className="network-stat">
          <span>Dominios recientes</span>
          <strong>{overview?.recentVisits ?? 0}</strong>
          <small>{overview?.blockedVisits ?? 0} bloqueados</small>
        </article>
        <article className="network-stat">
          <span>Último scan</span>
          <strong>{formatTime(overview?.generatedAt)}</strong>
          <small>refresco cada 5 s</small>
        </article>
      </section>

      <section className="network-toolbar">
        <input
          type="search"
          placeholder="Filtrar por equipo, IP, MAC o dominio"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <div className="status-tabs" role="group" aria-label="Filtro por estado">
          {statusOptions.map((option) => (
            <button
              className={status === option.value ? 'status-tab-active' : ''}
              key={option.value}
              type="button"
              onClick={() => setStatus(option.value)}
            >
              {option.label}
            </button>
          ))}
        </div>
      </section>

      <div className="network-grid">
        <section className="network-panel network-wide">
          <div className="network-panel-head">
            <div>
              <h2>Dispositivos</h2>
              <p>Marca lo conocido como confiable; bloquea u oculta lo que no reconozcas.</p>
            </div>
          </div>
          <div className="device-table">
            {isLoading && <div className="network-empty">Leyendo red actual…</div>}
            {!isLoading && visibleDevices.length === 0 && <div className="network-empty">No hay dispositivos con ese filtro.</div>}
            {visibleDevices.map((device) => (
              <article className={`device-row device-${device.status}`} key={device.id}>
                <div className="device-main">
                  <span className={`net-dot net-dot-${device.status}`} aria-hidden="true" />
                  <div>
                    <strong>{device.name || device.ip}</strong>
                    <span>{device.ip} · {device.interface || device.source || 'local'}</span>
                  </div>
                </div>
                <span className="device-chip">{statusLabels[device.status] || device.status}</span>
                <span className="device-mac">{device.mac}</span>
                <span className="device-seen">visto {formatTime(device.lastSeen)}</span>
                <DeviceActions device={device} onUpdate={updateDevice} />
              </article>
            ))}
          </div>
        </section>

        <section className="network-panel">
          <div className="network-panel-head">
            <div>
              <h2>Dominios en vivo</h2>
              <p>Consultas DNS leídas desde Pi-hole, dnsmasq o una ruta configurada en el servidor.</p>
            </div>
          </div>
          <div className="domain-list">
            {filteredVisits.length === 0 && <div className="network-empty">Sin eventos DNS disponibles.</div>}
            {filteredVisits.map((visit, index) => (
              <article className="domain-row" key={`${visit.timestamp}-${visit.domain}-${index}`}>
                <div>
                  <strong>{visit.domain}</strong>
                  <span>{visit.device || visit.clientIp || 'cliente desconocido'} · {formatTime(visit.timestamp)}</span>
                </div>
                <span className={`risk-tag risk-${visit.action}`}>{visit.action === 'blocked' ? 'Bloqueado' : 'Permitido'}</span>
              </article>
            ))}
          </div>
        </section>

        <section className="network-panel">
          <div className="network-panel-head">
            <div>
              <h2>Fuentes</h2>
              <p>Qué señales reales está leyendo el backend ahora.</p>
            </div>
          </div>
          <div className="source-list">
            {(overview?.liveSources || []).map((source) => (
              <article className="source-row" key={`${source.name}-${source.path}`}>
                <SourcePill source={source} />
                <div>
                  <strong>{source.path || 'sin ruta configurada'}</strong>
                  <span>{source.detail}</span>
                </div>
              </article>
            ))}
          </div>
          <div className="blocklist-note">
            Bloqueos exportados a: <strong>{overview?.blocklistPath || 'no configurado'}</strong>
          </div>
        </section>
      </div>
    </div>
  )
}
