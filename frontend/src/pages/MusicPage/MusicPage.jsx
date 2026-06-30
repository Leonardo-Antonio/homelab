import { useEffect, useState } from 'react'
import { Button } from '../../components/Button.jsx'
import { EmptyState } from '../../components/EmptyState.jsx'
import { MUSIC_SEARCH_TYPES, searchMusic } from '../../services/musicApi.js'
import { notify } from '../../services/notifications.js'
import { copyTextToClipboard } from '../../utils/clipboard.js'
import './MusicPage.css'

const DEFAULT_QUERY = 'Bad Bunny'

export function MusicPage() {
  const [query, setQuery] = useState(DEFAULT_QUERY)
  const [submittedQuery, setSubmittedQuery] = useState(DEFAULT_QUERY)
  const [typeFilter, setTypeFilter] = useState('all')
  const [results, setResults] = useState([])
  const [isLoading, setIsLoading] = useState(false)
  const [copiedId, setCopiedId] = useState(null)

  useEffect(() => {
    let active = true

    async function loadResults() {
      try {
        setIsLoading(true)
        const items = await searchMusic(submittedQuery, { type: typeFilter, limit: 24 })
        if (!active) return
        setResults(items)
      } catch (error) {
        if (active) {
          setResults([])
          notify.actionFailed('Búsqueda no disponible', error.message)
        }
      } finally {
        if (active) setIsLoading(false)
      }
    }

    loadResults()
    return () => {
      active = false
    }
  }, [submittedQuery, typeFilter])

  function handleSearch(event) {
    event.preventDefault()
    const nextQuery = query.trim()
    if (!nextQuery) return
    setSubmittedQuery(nextQuery)
  }

  async function handleCopy(item) {
    try {
      await copyTextToClipboard(item.spotifyUrl)
      setCopiedId(item.id)
      notify.clipboardCopied()
      window.setTimeout(() => {
        setCopiedId((current) => (current === item.id ? null : current))
      }, 2000)
    } catch {
      notify.actionFailed('No se pudo copiar', 'Copia el enlace manualmente.')
    }
  }

  return (
    <section className="music-page" aria-labelledby="music-title">
      <header className="music-hero">
        <div className="music-hero-copy">
          <p className="eyebrow">Spotify</p>
          <h1 id="music-title">Busca música, artistas y playlists; copia su enlace.</h1>
        </div>
        <form className="music-search" onSubmit={handleSearch} role="search">
          <label htmlFor="music-search-input">Búsqueda</label>
          <div>
            <input
              id="music-search-input"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Artista, canción, álbum o playlist"
              type="search"
            />
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Buscando...' : 'Buscar'}
            </Button>
          </div>
        </form>
      </header>

      <div className="music-type-tabs" aria-label="Tipo de búsqueda">
        {MUSIC_SEARCH_TYPES.map((type) => (
          <button
            className={typeFilter === type.id ? 'type-tab type-tab-active' : 'type-tab'}
            key={type.id}
            type="button"
            onClick={() => setTypeFilter(type.id)}
          >
            {type.label}
          </button>
        ))}
      </div>

      <section className="music-results" aria-labelledby="music-results-title">
        <div className="section-heading">
          <div>
            <h2 id="music-results-title">Resultados</h2>
            <p>
              {isLoading
                ? 'Consultando Spotify...'
                : `${results.length} resultados para "${submittedQuery}"`}
            </p>
          </div>
        </div>

        {isLoading ? (
          <div className="music-skeleton" aria-label="Cargando resultados">
            <span />
            <span />
            <span />
            <span />
          </div>
        ) : results.length > 0 ? (
          <ul className="music-grid">
            {results.map((item) => (
              <li key={`${item.type}:${item.id}`} className="music-card">
                <span className={`music-cover ${item.type === 'artist' ? 'music-cover-round' : ''}`}>
                  {item.imageUrl ? <img src={item.imageUrl} alt="" loading="lazy" /> : null}
                </span>
                <span className="music-card-copy">
                  <em>{item.subtitle}</em>
                  <strong>{item.title}</strong>
                </span>
                <div className="music-card-actions">
                  <Button
                    type="button"
                    variant={copiedId === item.id ? 'primary' : 'ghost'}
                    onClick={() => handleCopy(item)}
                  >
                    {copiedId === item.id ? 'Copiado ✓' : 'Copiar link'}
                  </Button>
                  <a
                    className="music-open-link"
                    href={item.spotifyUrl}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Abrir
                  </a>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <EmptyState
            title="Sin resultados"
            description="Prueba con otro término o revisa la ortografía."
          />
        )}
      </section>
    </section>
  )
}
