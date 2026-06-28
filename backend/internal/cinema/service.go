package cinema

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/124.0 Safari/537.36"

var (
	articleRe = regexp.MustCompile(`(?is)<article\b[^>]*>(.*?)</article>`)
	hrefRe    = regexp.MustCompile(`(?is)href="([^"#]+)"`)
	imgRe     = regexp.MustCompile(`(?is)<img[^>]+(?:data-src|data-lazy-src|src)="([^"]+)"`)
	altRe     = regexp.MustCompile(`(?is)alt="([^"]+)"`)
	titleRe   = regexp.MustCompile(`(?is)<h[23][^>]*>(?:\s*<a[^>]*>)?\s*([^<]+)`)
	yearRe    = regexp.MustCompile(`(?is)(?:year|date)["'][^>]*>\s*(\d{4})|>\s*((?:19|20)\d{2})\s*<`)
	classRe   = regexp.MustCompile(`(?is)class="([^"]*)"`)
	tagRe     = regexp.MustCompile(`(?is)<[^>]+>`)
)

// Config tunes the HTTP client used to fetch catalog pages.
type Config struct {
	Timeout   time.Duration
	UserAgent string
}

type Service struct {
	client    *http.Client
	userAgent string
	sources   map[string]Source
	order     []string
}

func NewService(sources []Source, cfg Config) *Service {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	agent := cfg.UserAgent
	if agent == "" {
		agent = defaultUserAgent
	}

	svc := &Service{
		client:    &http.Client{Timeout: timeout},
		userAgent: agent,
		sources:   make(map[string]Source, len(sources)),
	}
	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		svc.sources[source.ID] = source
		svc.order = append(svc.order, source.ID)
	}

	return svc
}

// Sources returns the enabled sources in declaration order.
func (s *Service) Sources() []Source {
	out := make([]Source, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.sources[id])
	}
	return out
}

// Search queries one source (sourceID empty means all enabled sources) and
// returns up to limit results per source. Individual source failures are
// swallowed so one broken site never blocks the others.
func (s *Service) Search(ctx context.Context, query, sourceID string, limit int) []Result {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if limit <= 0 {
		limit = 12
	}

	var targets []Source
	if sourceID != "" {
		source, ok := s.sources[sourceID]
		if !ok {
			return nil
		}
		targets = []Source{source}
	} else {
		targets = s.Sources()
	}

	var results []Result
	for _, source := range targets {
		items, err := s.searchSource(ctx, source, query, limit)
		if err != nil {
			continue
		}
		results = append(results, items...)
	}

	return results
}

func (s *Service) searchSource(ctx context.Context, source Source, query string, limit int) ([]Result, error) {
	target := fmt.Sprintf(source.SearchURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cinema: %s returned status %d", source.ID, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	return parseResults(string(body), source, limit), nil
}

func parseResults(body string, source Source, limit int) []Result {
	blocks := articleRe.FindAllStringSubmatch(body, -1)
	results := make([]Result, 0, len(blocks))
	seen := make(map[string]struct{})

	for _, block := range blocks {
		item, ok := parseBlock(block[1], source)
		if !ok {
			continue
		}
		if _, dup := seen[item.SourceURL]; dup {
			continue
		}
		seen[item.SourceURL] = struct{}{}
		results = append(results, item)
		if len(results) >= limit {
			break
		}
	}

	// Stable order: newest first when a year is available.
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].ReleaseYear > results[j].ReleaseYear
	})

	return results
}

func parseBlock(block string, source Source) (Result, bool) {
	link := firstDetailLink(block, source.BaseURL)
	if link == "" {
		return Result{}, false
	}

	title := ""
	if m := titleRe.FindStringSubmatch(block); m != nil {
		title = cleanText(m[1])
	}
	poster := ""
	if m := imgRe.FindStringSubmatch(block); m != nil {
		poster = absoluteURL(m[1], source.BaseURL)
		if title == "" {
			if alt := altRe.FindStringSubmatch(m[0]); alt != nil {
				title = cleanText(alt[1])
			}
		}
	}
	if title == "" {
		return Result{}, false
	}

	year := 0
	if m := yearRe.FindStringSubmatch(block); m != nil {
		raw := m[1]
		if raw == "" {
			raw = m[2]
		}
		if parsed, err := strconv.Atoi(raw); err == nil {
			year = parsed
		}
	}

	kind := "movie"
	lower := strings.ToLower(block + " " + link)
	if strings.Contains(lower, "tvshow") || strings.Contains(lower, "serie") || strings.Contains(lower, "/series") {
		kind = "tv"
	}

	return Result{
		ID:          source.ID + ":" + slug(link),
		Source:      source.ID,
		SourceLabel: source.Label,
		Title:       title,
		ReleaseYear: year,
		Kind:        kind,
		PosterURL:   poster,
		SourceURL:   link,
	}, true
}

// firstDetailLink picks the first internal anchor that points at a detail page
// rather than a category, tag or pagination link.
func firstDetailLink(block, baseURL string) string {
	for _, m := range hrefRe.FindAllStringSubmatch(block, -1) {
		raw := html.UnescapeString(m[1])
		if raw == "" || strings.HasPrefix(raw, "javascript:") {
			continue
		}
		abs := absoluteURL(raw, baseURL)
		lower := strings.ToLower(abs)
		if !strings.Contains(lower, strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(baseURL, "https://"), "http://"))) {
			continue
		}
		if strings.Contains(lower, "/category/") || strings.Contains(lower, "/genero/") ||
			strings.Contains(lower, "/tag/") || strings.Contains(lower, "/page/") ||
			strings.Contains(lower, "#") {
			continue
		}
		return abs
	}
	return ""
}

func absoluteURL(raw, baseURL string) string {
	raw = html.UnescapeString(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "/") {
		return strings.TrimRight(baseURL, "/") + raw
	}
	return strings.TrimRight(baseURL, "/") + "/" + raw
}

func cleanText(value string) string {
	value = tagRe.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	return strings.Join(strings.Fields(value), " ")
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(value string) string {
	value = strings.ToLower(value)
	value = slugRe.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}
