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
	// ModuleOrder is the sidebar order of the modules. A map cannot carry
	// order, so it lives in its own slice; normalize() keeps it a valid
	// permutation of KnownModules.
	ModuleOrder []string `json:"moduleOrder"`
}

// Default returns the baseline settings used before the user customises
// anything (and to backfill any field missing from a stored document).
func Default() Settings {
	modules := make(map[string]bool, len(KnownModules))
	order := make([]string, len(KnownModules))
	for i, id := range KnownModules {
		modules[id] = true
		order[i] = id
	}
	return Settings{
		Theme:       "light",
		Language:    "es",
		Font:        "sans",
		Modules:     modules,
		ModuleOrder: order,
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
	out.ModuleOrder = normalizeOrder(s.ModuleOrder)
	return out
}

// normalizeOrder turns an arbitrary client-supplied order into a valid
// permutation of KnownModules: it keeps the requested order, ignores unknown or
// duplicated ids, then appends any module the client left out (in default
// order) so every module always appears exactly once.
func normalizeOrder(requested []string) []string {
	seen := make(map[string]bool, len(KnownModules))
	order := make([]string, 0, len(KnownModules))
	for _, id := range requested {
		if contains(KnownModules, id) && !seen[id] {
			seen[id] = true
			order = append(order, id)
		}
	}
	for _, id := range KnownModules {
		if !seen[id] {
			order = append(order, id)
		}
	}
	return order
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
