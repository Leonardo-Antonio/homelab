import './NetworkPage.css'

const networkStats = [
  { label: 'Estado WAN', value: 'Online', detail: '18 ms a 1.1.1.1', tone: 'good' },
  { label: 'Dispositivos', value: '24', detail: '19 activos ahora', tone: 'neutral' },
  { label: 'Bloqueos DNS', value: '1,284', detail: 'últimas 24 h', tone: 'warn' },
  { label: 'Tráfico hoy', value: '86.4 GB', detail: '61% descarga', tone: 'neutral' },
]

const devices = [
  { name: 'nas-lab', ip: '192.168.1.10', mac: 'B8:27:EB:41:21:90', group: 'Infra', usage: '21.8 GB', status: 'online' },
  { name: 'workstation-leo', ip: '192.168.1.34', mac: '7C:8B:CA:19:4D:22', group: 'Personal', usage: '14.2 GB', status: 'online' },
  { name: 'sala-tv', ip: '192.168.1.51', mac: '44:65:0D:A9:72:C1', group: 'IoT', usage: '9.6 GB', status: 'online' },
  { name: 'camera-garage', ip: '192.168.1.72', mac: 'A4:CF:12:08:99:02', group: 'Cámaras', usage: '4.1 GB', status: 'online' },
  { name: 'phone-guest', ip: '192.168.1.88', mac: '92:F1:AC:12:73:DE', group: 'Invitados', usage: '860 MB', status: 'idle' },
]

const visitedDomains = [
  { domain: 'youtube.com', device: 'sala-tv', hits: 428, risk: 'Normal', time: 'hace 2 min' },
  { domain: 'github.com', device: 'workstation-leo', hits: 184, risk: 'Normal', time: 'hace 4 min' },
  { domain: 'cloudflare-dns.com', device: 'nas-lab', hits: 96, risk: 'Sistema', time: 'hace 8 min' },
  { domain: 'doubleclick.net', device: 'phone-guest', hits: 62, risk: 'Bloqueado', time: 'hace 13 min' },
  { domain: 'ntp.ubuntu.com', device: 'nas-lab', hits: 44, risk: 'Sistema', time: 'hace 19 min' },
]

const alerts = [
  { title: 'Nuevo dispositivo detectado', body: 'phone-guest recibió IP por DHCP en la VLAN principal.', level: 'warning' },
  { title: 'Pico de descarga', body: 'sala-tv consumió 5.2 GB entre 20:00 y 21:00.', level: 'info' },
  { title: 'DNS bloqueado', body: '62 consultas a trackers publicitarios fueron rechazadas.', level: 'blocked' },
]

const protocolShare = [
  { label: 'HTTPS', value: 64 },
  { label: 'Streaming', value: 18 },
  { label: 'DNS', value: 8 },
  { label: 'SSH/VPN', value: 6 },
  { label: 'Otros', value: 4 },
]

function StatusDot({ status }) {
  return <span className={`net-dot net-dot-${status}`} aria-hidden="true" />
}

function TrafficBars() {
  return (
    <div className="traffic-bars" aria-label="Distribución de tráfico por protocolo">
      {protocolShare.map((item) => (
        <div className="traffic-row" key={item.label}>
          <span>{item.label}</span>
          <div className="traffic-track">
            <span style={{ width: `${item.value}%` }} />
          </div>
          <strong>{item.value}%</strong>
        </div>
      ))}
    </div>
  )
}

export function NetworkPage() {
  return (
    <div className="network-page">
      <header className="network-hero">
        <div>
          <span className="network-kicker">Observabilidad local</span>
          <h1>Estado de red</h1>
          <p>
            Vista central para seguir actividad DNS, equipos conectados, consumo y señales raras dentro de tu HomeLab.
          </p>
        </div>
        <div className="network-live-badge" aria-label="Monitoreo activo">
          <span />
          Live
        </div>
      </header>

      <section className="network-stats" aria-label="Resumen de red">
        {networkStats.map((stat) => (
          <article className={`network-stat network-stat-${stat.tone}`} key={stat.label}>
            <span>{stat.label}</span>
            <strong>{stat.value}</strong>
            <small>{stat.detail}</small>
          </article>
        ))}
      </section>

      <div className="network-grid">
        <section className="network-panel network-map-panel">
          <div className="network-panel-head">
            <div>
              <h2>Topología</h2>
              <p>WAN, router, servicios y clientes principales.</p>
            </div>
          </div>
          <div className="network-map" aria-label="Topología de red">
            <div className="map-node map-wan">WAN</div>
            <div className="map-line map-line-router" />
            <div className="map-node map-router">Router</div>
            <div className="map-line map-line-core" />
            <div className="map-cluster">
              <span>NAS</span>
              <span>DNS</span>
              <span>IoT</span>
              <span>Guest</span>
            </div>
          </div>
        </section>

        <section className="network-panel">
          <div className="network-panel-head">
            <div>
              <h2>Tráfico</h2>
              <p>Distribución aproximada por tipo.</p>
            </div>
          </div>
          <TrafficBars />
        </section>

        <section className="network-panel network-wide">
          <div className="network-panel-head">
            <div>
              <h2>Dispositivos</h2>
              <p>Clientes observados por DHCP, DNS o ARP.</p>
            </div>
            <button className="network-icon-btn" type="button" title="Actualizar" aria-label="Actualizar dispositivos">↻</button>
          </div>
          <div className="device-table">
            {devices.map((device) => (
              <article className="device-row" key={device.mac}>
                <div className="device-main">
                  <StatusDot status={device.status} />
                  <div>
                    <strong>{device.name}</strong>
                    <span>{device.ip}</span>
                  </div>
                </div>
                <span className="device-chip">{device.group}</span>
                <span className="device-mac">{device.mac}</span>
                <strong className="device-usage">{device.usage}</strong>
              </article>
            ))}
          </div>
        </section>

        <section className="network-panel">
          <div className="network-panel-head">
            <div>
              <h2>Dominios recientes</h2>
              <p>Páginas y servicios consultados desde la red.</p>
            </div>
          </div>
          <div className="domain-list">
            {visitedDomains.map((item) => (
              <article className="domain-row" key={`${item.domain}-${item.device}`}>
                <div>
                  <strong>{item.domain}</strong>
                  <span>{item.device} · {item.time}</span>
                </div>
                <div className="domain-meta">
                  <strong>{item.hits}</strong>
                  <span className={`risk-tag risk-${item.risk.toLowerCase()}`}>{item.risk}</span>
                </div>
              </article>
            ))}
          </div>
        </section>

        <section className="network-panel">
          <div className="network-panel-head">
            <div>
              <h2>Alertas</h2>
              <p>Eventos que conviene revisar.</p>
            </div>
          </div>
          <div className="alert-list">
            {alerts.map((alert) => (
              <article className={`network-alert alert-${alert.level}`} key={alert.title}>
                <strong>{alert.title}</strong>
                <span>{alert.body}</span>
              </article>
            ))}
          </div>
        </section>
      </div>
    </div>
  )
}
