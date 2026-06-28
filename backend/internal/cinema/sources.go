package cinema

import "strings"

// SourcesFromAllowlist returns the default sources filtered by a comma
// separated allowlist of ids. An empty allowlist enables every source.
func SourcesFromAllowlist(allowlist string) []Source {
	sources := DefaultSources()
	allowlist = strings.TrimSpace(allowlist)
	if allowlist == "" {
		return sources
	}

	allowed := make(map[string]struct{})
	for _, id := range strings.Split(allowlist, ",") {
		if id = strings.TrimSpace(id); id != "" {
			allowed[id] = struct{}{}
		}
	}

	filtered := make([]Source, 0, len(sources))
	for _, source := range sources {
		if _, ok := allowed[source.ID]; ok {
			filtered = append(filtered, source)
		}
	}
	return filtered
}

// DefaultSources are the built-in catalog sites. They can be toggled with
// CINEMA_SOURCES (comma separated list of ids); an empty value enables all.
func DefaultSources() []Source {
	return []Source{
		{
			ID:        "cuevana",
			Label:     "Cuevana",
			Enabled:   true,
			BaseURL:   "https://cuevana.biz",
			SearchURL: "https://cuevana.biz/?s=%s",
		},
		{
			ID:        "pelisplus",
			Label:     "PelisPlus",
			Enabled:   true,
			BaseURL:   "https://pelisplus.to",
			SearchURL: "https://pelisplus.to/search?q=%s",
		},
	}
}
