package settings

import "errors"

var ErrInvalidSettings = errors.New("settings contain an invalid value")

// Allowed values for each preference. Kept here so both validation and the
// default factory share a single source of truth.
var (
	ValidThemes    = []string{"light", "dark", "system"}
	ValidLanguages = []string{"es", "en"}
	ValidFonts     = []string{"sans", "serif", "mono"}
	// KnownModules are the toggleable navigation modules. "config" is
	// intentionally excluded so the settings page can never be hidden.
	KnownModules = []string{"clipboard", "photos", "camera", "terminal", "notes", "storage"}
)

// Settings is the application-wide preferences document persisted server-side.
type Settings struct {
	Theme    string          `json:"theme"`
	Language string          `json:"language"`
	Font     string          `json:"font"`
	Modules  map[string]bool `json:"modules"`
}

// Default returns the baseline settings used before the user customises
// anything (and to backfill any field missing from a stored document).
func Default() Settings {
	modules := make(map[string]bool, len(KnownModules))
	for _, id := range KnownModules {
		modules[id] = true
	}
	return Settings{
		Theme:    "light",
		Language: "es",
		Font:     "sans",
		Modules:  modules,
	}
}

// normalize fills in missing fields from the defaults and drops unknown module
// keys, so a stored document is always complete and forward-compatible.
func (s Settings) normalize() Settings {
	out := Default()
	if s.Theme != "" {
		out.Theme = s.Theme
	}
	if s.Language != "" {
		out.Language = s.Language
	}
	if s.Font != "" {
		out.Font = s.Font
	}
	for _, id := range KnownModules {
		if enabled, ok := s.Modules[id]; ok {
			out.Modules[id] = enabled
		}
	}
	return out
}

// validate ensures every field holds an allowed value.
func (s Settings) validate() error {
	if !contains(ValidThemes, s.Theme) ||
		!contains(ValidLanguages, s.Language) ||
		!contains(ValidFonts, s.Font) {
		return ErrInvalidSettings
	}
	return nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
