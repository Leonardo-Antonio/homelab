package music

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	tokenURL  = "https://accounts.spotify.com/api/token"
	searchURL = "https://api.spotify.com/v1/search"
)

// ErrNotConfigured is returned when no Spotify credentials are available, so
// the handler can surface a clear "configure your keys" message.
var ErrNotConfigured = errors.New("music: spotify credentials are not configured")

// Config carries the Spotify app credentials (client credentials flow) and the
// HTTP timeout used for token and search requests.
type Config struct {
	ClientID     string
	ClientSecret string
	Timeout      time.Duration
}

type Service struct {
	clientID     string
	clientSecret string
	client       *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func NewService(cfg Config) *Service {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	return &Service{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		client:       &http.Client{Timeout: timeout},
	}
}

// Configured reports whether the Spotify credentials are present.
func (s *Service) Configured() bool {
	return s.clientID != "" && s.clientSecret != ""
}

// Search queries Spotify for the given term and types (e.g. "artist,track").
// It returns the matching entities flattened into a single slice, preserving
// the order Spotify returns within each type.
func (s *Service) Search(ctx context.Context, query, types string, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if !s.Configured() {
		return nil, ErrNotConfigured
	}
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	if types == "" {
		types = "artist,album,track,playlist"
	}

	token, err := s.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("type", types)
	params.Set("limit", fmt.Sprintf("%d", limit))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return nil, fmt.Errorf("music: spotify search returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return payload.results(), nil
}

// accessToken returns a cached client-credentials token, refreshing it when it
// is missing or about to expire.
func (s *Service) accessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && time.Now().Before(s.tokenExpiry) {
		return s.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(s.clientID + ":" + s.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return "", fmt.Errorf("music: spotify token request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", err
	}
	if token.AccessToken == "" {
		return "", errors.New("music: spotify returned an empty access token")
	}

	s.token = token.AccessToken
	// Refresh a minute early to avoid races at the expiry boundary.
	s.tokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn-60) * time.Second)

	return s.token, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// searchResponse mirrors the slice of the Spotify search payload we consume.
type searchResponse struct {
	Artists   itemPage `json:"artists"`
	Albums    itemPage `json:"albums"`
	Tracks    itemPage `json:"tracks"`
	Playlists itemPage `json:"playlists"`
}

type itemPage struct {
	Items []spotifyItem `json:"items"`
}

type spotifyItem struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Images       []spotifyImage `json:"images"`
	Artists      []spotifyRef   `json:"artists"`
	Album        *spotifyAlbum  `json:"album"`
	Owner        *spotifyRef    `json:"owner"`
	ExternalURLs spotifyURLs    `json:"external_urls"`
}

type spotifyAlbum struct {
	Images []spotifyImage `json:"images"`
}

type spotifyRef struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type spotifyImage struct {
	URL string `json:"url"`
}

type spotifyURLs struct {
	Spotify string `json:"spotify"`
}

// results flattens the four entity buckets into normalized Results, skipping
// any item that lacks a public Spotify URL (the whole point of the feature).
func (r searchResponse) results() []Result {
	pages := []itemPage{r.Artists, r.Albums, r.Tracks, r.Playlists}
	out := make([]Result, 0)
	for _, page := range pages {
		for _, item := range page.Items {
			if item.ExternalURLs.Spotify == "" {
				continue
			}
			out = append(out, normalize(item))
		}
	}
	return out
}

func normalize(item spotifyItem) Result {
	return Result{
		ID:         item.ID,
		Type:       item.Type,
		Title:      item.Name,
		Subtitle:   subtitle(item),
		ImageURL:   imageURL(item),
		SpotifyURL: item.ExternalURLs.Spotify,
	}
}

func subtitle(item spotifyItem) string {
	switch item.Type {
	case "artist":
		return "Artista"
	case "album":
		return joinArtists("Álbum", item.Artists)
	case "track":
		return joinArtists("Canción", item.Artists)
	case "playlist":
		if item.Owner != nil && item.Owner.DisplayName != "" {
			return "Playlist · " + item.Owner.DisplayName
		}
		return "Playlist"
	default:
		return item.Type
	}
}

func joinArtists(prefix string, artists []spotifyRef) string {
	names := make([]string, 0, len(artists))
	for _, artist := range artists {
		if artist.Name != "" {
			names = append(names, artist.Name)
		}
	}
	if len(names) == 0 {
		return prefix
	}
	return prefix + " · " + strings.Join(names, ", ")
}

func imageURL(item spotifyItem) string {
	if len(item.Images) > 0 {
		return item.Images[0].URL
	}
	if item.Album != nil && len(item.Album.Images) > 0 {
		return item.Album.Images[0].URL
	}
	return ""
}
