package cinema

// Source describes an external catalog site that is scraped server-side.
// Selectors are intentionally tolerant: these sites change their markup
// often, so parsing relies on generic article/anchor/image extraction
// rather than brittle per-site CSS selectors.
type Source struct {
	ID        string
	Label     string
	Enabled   bool
	BaseURL   string // used to resolve relative links, e.g. https://cuevana.biz
	SearchURL string // contains a single %s placeholder for the URL-encoded query
}

// Result is a single catalog entry returned to the frontend. The JSON shape
// maps directly onto createMovieResult(...) on the client.
type Result struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	SourceLabel string `json:"sourceLabel"`
	Title       string `json:"title"`
	ReleaseYear int    `json:"releaseYear,omitempty"`
	Kind        string `json:"kind"` // "movie" or "tv"
	PosterURL   string `json:"posterUrl"`
	SourceURL   string `json:"sourceUrl"`
}
