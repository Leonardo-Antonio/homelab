package music

// Result is a single normalized Spotify entity (artist, album, track or
// playlist) returned to the frontend. SpotifyURL is the public open.spotify.com
// link the user can copy and share.
type Result struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	ImageURL   string `json:"imageUrl"`
	SpotifyURL string `json:"spotifyUrl"`
}
