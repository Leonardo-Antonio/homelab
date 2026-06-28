import { useEffect, useMemo, useState } from 'react'
import { Button } from '../../components/Button.jsx'
import { EmptyState } from '../../components/EmptyState.jsx'
import { buildLegalSources, searchMovies } from '../../services/movieApi.js'
import { notify } from '../../services/notifications.js'
import './CinemaPage.css'

const DEFAULT_QUERY = 'sherlock jr'

export function CinemaPage() {
  const [query, setQuery] = useState(DEFAULT_QUERY)
  const [submittedQuery, setSubmittedQuery] = useState(DEFAULT_QUERY)
  const [movies, setMovies] = useState([])
  const [selectedMovie, setSelectedMovie] = useState(null)
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    let active = true

    async function loadMovies() {
      try {
        setIsLoading(true)
        const results = await searchMovies(submittedQuery)
        if (!active) return
        setMovies(results)
        setSelectedMovie((current) => {
          if (current && results.some((movie) => movie.id === current.id)) return current
          return results[0] || null
        })
      } catch (error) {
        if (active) {
          setMovies([])
          setSelectedMovie(null)
          notify.actionFailed('Busqueda no disponible', error.message)
        }
      } finally {
        if (active) setIsLoading(false)
      }
    }

    loadMovies()
    return () => {
      active = false
    }
  }, [submittedQuery])

  const selectedSources = useMemo(
    () => (selectedMovie ? buildLegalSources(selectedMovie) : []),
    [selectedMovie],
  )

  function handleSearch(event) {
    event.preventDefault()
    const nextQuery = query.trim()
    if (!nextQuery) return
    setSubmittedQuery(nextQuery)
  }

  return (
    <section className="cinema-page" aria-labelledby="cinema-title">
      <header className="cinema-hero">
        <div className="cinema-hero-copy">
          <p className="eyebrow">Movie signal</p>
          <h1 id="cinema-title">Busca una pelicula y encuentra donde verla.</h1>
        </div>
        <form className="cinema-search" onSubmit={handleSearch} role="search">
          <label htmlFor="cinema-search-input">Pelicula</label>
          <div>
            <input
              id="cinema-search-input"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Nombre de la pelicula"
              type="search"
            />
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Buscando...' : 'Buscar'}
            </Button>
          </div>
        </form>
      </header>

      <section className="cinema-stage" aria-label="Resultado seleccionado">
        <div className="cinema-player">
          {selectedMovie?.previewUrl ? (
            <video
              key={selectedMovie.previewUrl}
              controls
              playsInline
              poster={selectedMovie.posterUrl}
              src={selectedMovie.previewUrl}
            />
          ) : (
            <div className="cinema-player-empty">
              <span aria-hidden="true">▶</span>
              <p>Preview no disponible para este resultado.</p>
            </div>
          )}
        </div>

        <aside className="cinema-detail">
          {selectedMovie ? (
            <>
              <div>
                <p className="eyebrow">{selectedMovie.genre}</p>
                <h2>{selectedMovie.title}</h2>
                <p className="cinema-meta">
                  {[selectedMovie.releaseYear, selectedMovie.rating, selectedMovie.runtime ? `${selectedMovie.runtime} min` : null]
                    .filter(Boolean)
                    .join(' / ')}
                </p>
              </div>
              <p className="cinema-description">{selectedMovie.description}</p>
              <div className="source-list" aria-label="Fuentes disponibles">
                {selectedSources.map((source) => (
                  <a key={source.id} href={source.url} target="_blank" rel="noreferrer">
                    <strong>{source.label}</strong>
                    <span>{source.note}</span>
                  </a>
                ))}
              </div>
            </>
          ) : (
            <EmptyState
              title="Sin seleccion"
              description="Busca una pelicula para ver resultados y fuentes disponibles."
            />
          )}
        </aside>
      </section>

      <section className="cinema-results" aria-labelledby="cinema-results-title">
        <div className="section-heading">
          <div>
            <h2 id="cinema-results-title">Alternativas</h2>
            <p>{isLoading ? 'Consultando catalogos...' : `${movies.length} resultados para "${submittedQuery}"`}</p>
          </div>
        </div>

        {isLoading ? (
          <div className="cinema-skeleton" aria-label="Cargando peliculas">
            <span />
            <span />
            <span />
            <span />
          </div>
        ) : movies.length > 0 ? (
          <ul className="movie-grid">
            {movies.map((movie) => (
              <li key={movie.id}>
                <button
                  className={`movie-card ${selectedMovie?.id === movie.id ? 'movie-card-active' : ''}`}
                  type="button"
                  onClick={() => setSelectedMovie(movie)}
                >
                  <span className="movie-poster">
                    {movie.posterUrl ? <img src={movie.posterUrl} alt="" loading="lazy" /> : null}
                  </span>
                  <span className="movie-card-copy">
                    <strong>{movie.title}</strong>
                    <small>{[movie.releaseYear, movie.genre].filter(Boolean).join(' / ')}</small>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        ) : (
          <EmptyState
            title="No hubo resultados"
            description="Prueba con otro titulo o revisa la ortografia."
          />
        )}
      </section>
    </section>
  )
}
