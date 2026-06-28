// Lightweight i18n. The dictionary covers the shell (navigation, brand) and the
// Config page. Add keys here and reference them through the `t()` helper exposed
// by the settings context; missing keys fall back to the Spanish string or the
// key itself, so partial coverage degrades gracefully.

const dictionaries = {
  es: {
    'brand.subtitle': 'Utilidades personales',
    'sidebar.collapse': 'Colapsar menú',
    'sidebar.expand': 'Expandir menú',

    'nav.clipboard': 'Clipboard',
    'nav.network': 'Red',
    'nav.photos': 'Fotos',
    'nav.camera': 'Cámara',
    'nav.terminal': 'Terminal',
    'nav.notes': 'Notas',
    'nav.storage': 'Drive',
    'nav.config': 'Ajustes',

    'config.title': 'Ajustes',
    'config.subtitle': 'Preferencias guardadas en tu servidor; te siguen en cualquier navegador o dispositivo.',
    'config.appearance': 'Apariencia',
    'config.theme': 'Tema',
    'config.theme.light': 'Claro',
    'config.theme.dark': 'Oscuro',
    'config.theme.system': 'Sistema',
    'config.font': 'Tipografía',
    'config.font.sans': 'Moderna',
    'config.font.serif': 'Clásica',
    'config.font.mono': 'Monoespaciada',
    'config.language': 'Idioma',
    'config.language.es': 'Español',
    'config.language.en': 'Inglés',
    'config.modules': 'Módulos',
    'config.modules.desc': 'Activa o desactiva las secciones, y arrástralas para cambiar su orden en el menú.',
    'config.reorder': 'Arrastra para reordenar',
    'config.saving': 'Guardando…',
    'config.saved': 'Ajustes guardados',
    'config.error': 'No se pudieron guardar los ajustes',
    'config.preview': 'Vista previa',
    'config.previewText': 'Así se ve el texto con la tipografía y el tema elegidos.',
  },
  en: {
    'brand.subtitle': 'Personal utilities',
    'sidebar.collapse': 'Collapse menu',
    'sidebar.expand': 'Expand menu',

    'nav.clipboard': 'Clipboard',
    'nav.network': 'Network',
    'nav.photos': 'Photos',
    'nav.camera': 'Camera',
    'nav.terminal': 'Terminal',
    'nav.notes': 'Notes',
    'nav.storage': 'Drive',
    'nav.config': 'Settings',

    'config.title': 'Settings',
    'config.subtitle': 'Preferences saved on your server; they follow you across browsers and devices.',
    'config.appearance': 'Appearance',
    'config.theme': 'Theme',
    'config.theme.light': 'Light',
    'config.theme.dark': 'Dark',
    'config.theme.system': 'System',
    'config.font': 'Font',
    'config.font.sans': 'Modern',
    'config.font.serif': 'Classic',
    'config.font.mono': 'Monospace',
    'config.language': 'Language',
    'config.language.es': 'Spanish',
    'config.language.en': 'English',
    'config.modules': 'Modules',
    'config.modules.desc': 'Turn the sidebar sections on or off, and drag them to reorder the menu.',
    'config.reorder': 'Drag to reorder',
    'config.saving': 'Saving…',
    'config.saved': 'Settings saved',
    'config.error': 'Could not save settings',
    'config.preview': 'Preview',
    'config.previewText': 'This is how text looks with the chosen font and theme.',
  },
}

export function createTranslator(language) {
  const dict = dictionaries[language] || dictionaries.es
  return (key, fallback) => dict[key] ?? dictionaries.es[key] ?? fallback ?? key
}
