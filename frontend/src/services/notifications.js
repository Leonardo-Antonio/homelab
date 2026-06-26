import { sileo } from 'sileo'

const DEFAULT_ERROR_DESCRIPTION = 'Intenta nuevamente en unos segundos.'

export const notify = {
  clipboardCreated() {
    sileo.success({
      title: 'Snippet agregado',
      description: 'El texto ya esta guardado en tu clipboard.',
    })
  },

  clipboardCopied() {
    sileo.success({
      title: 'Copiado',
      description: 'El contenido esta listo para pegar.',
      duration: 2400,
    })
  },

  clipboardDeleted() {
    sileo.info({
      title: 'Snippet eliminado',
      description: 'Se borro el item seleccionado.',
    })
  },

  clipboardCleared() {
    sileo.warning({
      title: 'Lista limpiada',
      description: 'Se eliminaron todos los snippets.',
    })
  },

  backendUnavailable(message = DEFAULT_ERROR_DESCRIPTION) {
    sileo.error({
      title: 'Backend no disponible',
      description: message,
      duration: 7000,
    })
  },

  actionFailed(title, description = DEFAULT_ERROR_DESCRIPTION) {
    sileo.error({
      title,
      description,
      duration: 6500,
    })
  },

  actionSucceeded(title, description) {
    sileo.success({
      title,
      description,
    })
  },

  photoSaved() {
    sileo.success({
      title: 'Foto guardada',
      description: 'La captura ya esta disponible en la galeria.',
    })
  },

  photoDeleted() {
    sileo.info({
      title: 'Foto eliminada',
      description: 'La imagen se borro de la galeria.',
    })
  },

  cameraFailed() {
    sileo.error({
      title: 'Camara no disponible',
      description: 'Revisa permisos del navegador o conecta una camara.',
      duration: 7000,
    })
  },
}
