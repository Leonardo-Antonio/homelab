import { useEffect, useMemo, useRef, useState } from 'react'
import { Button } from '../../components/Button.jsx'
import { EmptyState } from '../../components/EmptyState.jsx'
import { usePhotos } from '../../hooks/usePhotos.js'
import { buildPhotoUrl } from '../../services/photosApi.js'
import { notify } from '../../services/notifications.js'
import './PhotosPage.css'

export function PhotosPage() {
  const fileInputRef = useRef(null)
  const [isSavingPhoto, setIsSavingPhoto] = useState(false)
  const [selectedPhoto, setSelectedPhoto] = useState(null)
  const {
    error,
    goToNextPage,
    goToPreviousPage,
    isLoading,
    pagination,
    photos,
    removePhoto,
    savePhoto,
  } = usePhotos()

  const photoCountLabel = useMemo(() => {
    if (pagination.total === 1) {
      return '1 foto guardada'
    }

    return `${pagination.total} fotos guardadas`
  }, [pagination.total])

  useEffect(() => {
    if (error) {
      notify.backendUnavailable(error)
    }
  }, [error])

  function openNativeCamera() {
    fileInputRef.current?.click()
  }

  async function handleNativeCapture(event) {
    const [file] = event.target.files
    event.target.value = ''

    if (!file) {
      return
    }

    try {
      setIsSavingPhoto(true)
      await savePhoto(file)
      notify.photoSaved()
    } catch {
      notify.actionFailed('No se pudo guardar', 'La foto no pudo enviarse al backend.')
    } finally {
      setIsSavingPhoto(false)
    }
  }

  async function handleRemove(photoId) {
    try {
      await removePhoto(photoId)
      if (selectedPhoto?.id === photoId) {
        setSelectedPhoto(null)
      }
      notify.photoDeleted()
    } catch {
      notify.actionFailed('No se pudo eliminar', 'La foto no pudo borrarse del backend.')
    }
  }

  return (
    <section className="photos-page" aria-labelledby="photos-title">
      <header className="photos-header">
        <div>
          <p className="eyebrow">Photo capture</p>
          <h1 id="photos-title">Toma fotos y mantenlas en una galeria privada.</h1>
        </div>
        <div className="status-pill" aria-label={photoCountLabel}>
          {photoCountLabel}
        </div>
      </header>

      <section className="capture-panel" aria-label="Camara">
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          capture="environment"
          onChange={handleNativeCapture}
        />
        <div className="capture-art" aria-hidden="true">
          <span />
        </div>
        <div>
          <h2>Captura directa</h2>
          <p>
            En movil se abrira la camara nativa. Al tomar la foto, HomeLab la guardara
            automaticamente en la galeria.
          </p>
        </div>
        <Button type="button" onClick={openNativeCamera} disabled={isSavingPhoto}>
          {isSavingPhoto ? 'Guardando...' : 'Tomar foto'}
        </Button>
      </section>

      <section className="gallery-section" aria-labelledby="gallery-title">
        <div className="section-heading">
          <div>
            <h2 id="gallery-title">Galeria</h2>
            <p>
              Pagina {pagination.pages === 0 ? 0 : pagination.page} de {pagination.pages}
            </p>
          </div>
        </div>

        {isLoading ? (
          <div className="photo-skeleton" aria-label="Cargando fotos">
            <span />
            <span />
            <span />
          </div>
        ) : photos.length > 0 ? (
          <ul className="photo-grid">
            {photos.map((photo) => (
              <li className="photo-card" key={photo.id}>
                <button type="button" onClick={() => setSelectedPhoto(photo)}>
                  <img src={buildPhotoUrl(photo.url)} alt={`Foto tomada el ${photo.createdAtLabel}`} />
                </button>
                <div className="photo-card-meta">
                  <span>{photo.createdAtLabel}</span>
                  <Button type="button" variant="ghost" onClick={() => handleRemove(photo.id)}>
                    Borrar
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <EmptyState
            title="Aun no hay fotos"
            description="Toma una foto con la camara del dispositivo y aparecera en esta galeria."
          />
        )}

        {!isLoading && pagination.total > 0 ? (
          <nav className="pagination" aria-label="Paginacion de fotos">
            <Button
              type="button"
              variant="ghost"
              onClick={goToPreviousPage}
              disabled={!pagination.hasPrevious}
            >
              Anterior
            </Button>
            <span>
              {photos.length} de {pagination.total}
            </span>
            <Button
              type="button"
              variant="ghost"
              onClick={goToNextPage}
              disabled={!pagination.hasNext}
            >
              Siguiente
            </Button>
          </nav>
        ) : null}
      </section>

      {selectedPhoto ? (
        <div className="photo-viewer" role="dialog" aria-modal="true" aria-label="Foto seleccionada">
          <button className="photo-viewer-backdrop" type="button" onClick={() => setSelectedPhoto(null)} />
          <div className="photo-viewer-content">
            <img
              src={buildPhotoUrl(selectedPhoto.url)}
              alt={`Foto tomada el ${selectedPhoto.createdAtLabel}`}
            />
            <div className="photo-viewer-actions">
              <span>{selectedPhoto.createdAtLabel}</span>
              <Button type="button" variant="secondary" onClick={() => setSelectedPhoto(null)}>
                Cerrar
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  )
}
