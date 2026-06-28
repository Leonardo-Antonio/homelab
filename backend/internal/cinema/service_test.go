package cinema

import "testing"

const sampleHTML = `
<div class="search-page">
  <article class="item movies">
    <div class="poster">
      <img data-src="/wp-content/uploads/batman.jpg" alt="Batman Begins">
      <a href="https://cuevana.biz/pelicula/batman-begins/"></a>
    </div>
    <div class="data">
      <h3><a href="https://cuevana.biz/pelicula/batman-begins/">Batman Begins</a></h3>
      <span class="year">2005</span>
    </div>
  </article>
  <article class="item tvshows">
    <div class="poster">
      <img src="//image.tmdb.org/penguin.jpg" alt="The Penguin">
      <a href="/serie/the-penguin/"></a>
    </div>
    <div class="data">
      <h3><a href="/serie/the-penguin/">The Penguin</a></h3>
      <span class="year">2024</span>
    </div>
  </article>
  <article class="item movies">
    <a href="https://cuevana.biz/category/accion/">Accion</a>
  </article>
</div>`

func TestParseResults(t *testing.T) {
	source := Source{ID: "cuevana", Label: "Cuevana", BaseURL: "https://cuevana.biz"}
	results := parseResults(sampleHTML, source, 10)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}

	// Sorted newest-first by year.
	if results[0].Title != "The Penguin" {
		t.Errorf("expected first result 'The Penguin', got %q", results[0].Title)
	}
	if results[0].Kind != "tv" {
		t.Errorf("expected kind tv for series, got %q", results[0].Kind)
	}
	if results[0].PosterURL != "https://image.tmdb.org/penguin.jpg" {
		t.Errorf("protocol-relative poster not resolved: %q", results[0].PosterURL)
	}
	if results[0].SourceURL != "https://cuevana.biz/serie/the-penguin/" {
		t.Errorf("relative link not resolved: %q", results[0].SourceURL)
	}

	movie := results[1]
	if movie.Title != "Batman Begins" || movie.ReleaseYear != 2005 || movie.Kind != "movie" {
		t.Errorf("unexpected movie result: %+v", movie)
	}
}

func TestParseResultsSkipsCategoryOnly(t *testing.T) {
	source := Source{ID: "cuevana", Label: "Cuevana", BaseURL: "https://cuevana.biz"}
	html := `<article><a href="https://cuevana.biz/category/accion/">Accion</a><h3>Accion</h3></article>`
	if results := parseResults(html, source, 10); len(results) != 0 {
		t.Errorf("expected category-only article to be skipped, got %+v", results)
	}
}
