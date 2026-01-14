package crawler

import (
	"os"
	"path/filepath"
	"testing"
)

// readTestdata reads a file from the testdata directory
func readTestdata(t *testing.T, filename string) string {
	t.Helper()
	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read testdata %s: %v", filename, err)
	}
	return string(data)
}

// --- LinkSelector Tests ---

func TestNewLinkSelector_ValidPattern(t *testing.T) {
	ls, err := NewLinkSelector("a.product", "/product/\\d+")
	if err != nil {
		t.Fatalf("NewLinkSelector() error = %v", err)
	}

	if ls.CSSSelector != "a.product" {
		t.Errorf("expected CSSSelector 'a.product', got %q", ls.CSSSelector)
	}

	if ls.URLPattern == nil {
		t.Error("expected URLPattern to be set")
	}
}

func TestNewLinkSelector_InvalidPattern(t *testing.T) {
	_, err := NewLinkSelector("a", "[invalid")
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestNewLinkSelector_EmptyPattern(t *testing.T) {
	ls, err := NewLinkSelector("a", "")
	if err != nil {
		t.Fatalf("NewLinkSelector() error = %v", err)
	}

	if ls.URLPattern != nil {
		t.Error("expected URLPattern to be nil for empty pattern")
	}
}

func TestLinkSelector_ExtractLinks_DefaultSelector(t *testing.T) {
	html := readTestdata(t, "links.html")

	ls, _ := NewLinkSelector("", "") // Default selector: a[href]
	links, err := ls.ExtractLinks(html, "https://example.com/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	if len(links) == 0 {
		t.Error("expected some links to be extracted")
	}

	// Should include product links
	hasProduct := false
	for _, link := range links {
		if link == "https://example.com/product/1" {
			hasProduct = true
			break
		}
	}

	if !hasProduct {
		t.Errorf("expected product link, got %v", links)
	}
}

func TestLinkSelector_ExtractLinks_CustomCSSSelector(t *testing.T) {
	html := readTestdata(t, "links.html")

	ls, _ := NewLinkSelector("a.product-link", "")
	links, err := ls.ExtractLinks(html, "https://example.com/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	// Should only get product links
	if len(links) != 3 {
		t.Errorf("expected 3 product links, got %d: %v", len(links), links)
	}

	for _, link := range links {
		if link != "https://example.com/product/1" &&
			link != "https://example.com/product/2" &&
			link != "https://example.com/product/3" {
			t.Errorf("unexpected link: %s", link)
		}
	}
}

func TestLinkSelector_ExtractLinks_URLPattern(t *testing.T) {
	html := readTestdata(t, "links.html")

	ls, _ := NewLinkSelector("", "/product/\\d+$")
	links, err := ls.ExtractLinks(html, "https://example.com/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	// Should only get URLs matching the pattern
	for _, link := range links {
		if link != "https://example.com/product/1" &&
			link != "https://example.com/product/2" &&
			link != "https://example.com/product/3" {
			t.Errorf("unexpected link: %s (should match /product/\\d+)", link)
		}
	}
}

func TestLinkSelector_ExtractLinks_RelativeURLs(t *testing.T) {
	html := `<a href="relative/path">Link</a>`

	ls, _ := NewLinkSelector("", "")
	links, err := ls.ExtractLinks(html, "https://example.com/base/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}

	expected := "https://example.com/base/relative/path"
	if links[0] != expected {
		t.Errorf("expected %q, got %q", expected, links[0])
	}
}

func TestLinkSelector_ExtractLinks_ParentPath(t *testing.T) {
	html := `<a href="../other/path">Link</a>`

	ls, _ := NewLinkSelector("", "")
	links, err := ls.ExtractLinks(html, "https://example.com/base/page/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}

	expected := "https://example.com/base/other/path"
	if links[0] != expected {
		t.Errorf("expected %q, got %q", expected, links[0])
	}
}

func TestLinkSelector_ExtractLinks_SkipsFragments(t *testing.T) {
	html := `
		<a href="#section">Fragment</a>
		<a href="/page#section">Page with Fragment</a>
	`

	ls, _ := NewLinkSelector("", "")
	links, err := ls.ExtractLinks(html, "https://example.com/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	// Should skip pure fragment link but include page with fragment (fragment removed)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d: %v", len(links), links)
	}

	// Fragment should be removed from the URL
	if links[0] != "https://example.com/page" {
		t.Errorf("expected fragment removed, got %q", links[0])
	}
}

func TestLinkSelector_ExtractLinks_SkipsJavaScript(t *testing.T) {
	html := `
		<a href="javascript:void(0)">JS Link</a>
		<a href="javascript:doSomething()">Another JS</a>
		<a href="/real-page">Real Link</a>
	`

	ls, _ := NewLinkSelector("", "")
	links, err := ls.ExtractLinks(html, "https://example.com/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d: %v", len(links), links)
	}

	if links[0] != "https://example.com/real-page" {
		t.Errorf("expected real link, got %q", links[0])
	}
}

func TestLinkSelector_ExtractLinks_Deduplicates(t *testing.T) {
	html := `
		<a href="/page">Link 1</a>
		<a href="/page">Link 2</a>
		<a href="/page">Link 3</a>
	`

	ls, _ := NewLinkSelector("", "")
	links, err := ls.ExtractLinks(html, "https://example.com/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	if len(links) != 1 {
		t.Errorf("expected 1 deduplicated link, got %d: %v", len(links), links)
	}
}

func TestLinkSelector_ExtractLinks_SkipsEmptyHref(t *testing.T) {
	html := `
		<a href="">Empty</a>
		<a>No Href</a>
		<a href="/page">Real</a>
	`

	ls, _ := NewLinkSelector("", "")
	links, err := ls.ExtractLinks(html, "https://example.com/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	if len(links) != 1 {
		t.Errorf("expected 1 link, got %d: %v", len(links), links)
	}
}

func TestLinkSelector_ExtractLinks_FromTestdata(t *testing.T) {
	html := readTestdata(t, "links.html")

	ls, _ := NewLinkSelector("a.product-link", "")
	links, err := ls.ExtractLinks(html, "https://example.com/")
	if err != nil {
		t.Fatalf("ExtractLinks() error = %v", err)
	}

	expected := map[string]bool{
		"https://example.com/product/1": true,
		"https://example.com/product/2": true,
		"https://example.com/product/3": true,
	}

	if len(links) != len(expected) {
		t.Errorf("expected %d links, got %d: %v", len(expected), len(links), links)
	}

	for _, link := range links {
		if !expected[link] {
			t.Errorf("unexpected link: %s", link)
		}
	}
}

// --- PaginationSelector Tests ---

func TestNewPaginationSelector(t *testing.T) {
	ps := NewPaginationSelector("a.next")
	if ps.NextSelector != "a.next" {
		t.Errorf("expected NextSelector 'a.next', got %q", ps.NextSelector)
	}
}

func TestPaginationSelector_FindNextPage_Found(t *testing.T) {
	html := readTestdata(t, "pagination.html")

	ps := NewPaginationSelector("a.next")
	nextURL, found := ps.FindNextPage(html, "https://example.com/")

	if !found {
		t.Fatal("expected to find next page")
	}

	expected := "https://example.com/search?page=3"
	if nextURL != expected {
		t.Errorf("expected %q, got %q", expected, nextURL)
	}
}

func TestPaginationSelector_FindNextPage_NotFound(t *testing.T) {
	html := `<nav><a href="/prev" class="prev">Prev</a></nav>`

	ps := NewPaginationSelector("a.next")
	nextURL, found := ps.FindNextPage(html, "https://example.com/")

	if found {
		t.Errorf("expected not to find next page, got %q", nextURL)
	}
}

func TestPaginationSelector_FindNextPage_EmptySelector(t *testing.T) {
	html := readTestdata(t, "pagination.html")

	ps := NewPaginationSelector("")
	nextURL, found := ps.FindNextPage(html, "https://example.com/")

	if found {
		t.Errorf("expected not to find next page with empty selector, got %q", nextURL)
	}
}

func TestPaginationSelector_FindNextPage_RelativeURL(t *testing.T) {
	html := `<a href="next-page" class="next">Next</a>`

	ps := NewPaginationSelector("a.next")
	nextURL, found := ps.FindNextPage(html, "https://example.com/current/")

	if !found {
		t.Fatal("expected to find next page")
	}

	expected := "https://example.com/current/next-page"
	if nextURL != expected {
		t.Errorf("expected %q, got %q", expected, nextURL)
	}
}

func TestPaginationSelector_FindNextPage_SkipsFragment(t *testing.T) {
	html := `<a href="#next" class="next">Next</a>`

	ps := NewPaginationSelector("a.next")
	_, found := ps.FindNextPage(html, "https://example.com/")

	if found {
		t.Error("expected not to find next page for fragment-only link")
	}
}

func TestPaginationSelector_FindNextPage_SkipsJavaScript(t *testing.T) {
	html := `<a href="javascript:loadMore()" class="next">Next</a>`

	ps := NewPaginationSelector("a.next")
	_, found := ps.FindNextPage(html, "https://example.com/")

	if found {
		t.Error("expected not to find next page for javascript link")
	}
}

func TestPaginationSelector_FindNextPage_AlternateSelector(t *testing.T) {
	html := readTestdata(t, "pagination.html")

	ps := NewPaginationSelector("a[rel='next']")
	nextURL, found := ps.FindNextPage(html, "https://example.com/")

	if !found {
		t.Fatal("expected to find next page with rel='next' selector")
	}

	expected := "https://example.com/results/page/3"
	if nextURL != expected {
		t.Errorf("expected %q, got %q", expected, nextURL)
	}
}

func TestPaginationSelector_FindNextPage_FirstMatch(t *testing.T) {
	html := `
		<a href="/first" class="next">First Next</a>
		<a href="/second" class="next">Second Next</a>
	`

	ps := NewPaginationSelector("a.next")
	nextURL, found := ps.FindNextPage(html, "https://example.com/")

	if !found {
		t.Fatal("expected to find next page")
	}

	// Should return first match
	expected := "https://example.com/first"
	if nextURL != expected {
		t.Errorf("expected first match %q, got %q", expected, nextURL)
	}
}

func TestPaginationSelector_FindNextPage_EmptyHref(t *testing.T) {
	html := `<a href="" class="next">Next</a>`

	ps := NewPaginationSelector("a.next")
	_, found := ps.FindNextPage(html, "https://example.com/")

	if found {
		t.Error("expected not to find next page for empty href")
	}
}

func TestPaginationSelector_FindNextPage_InvalidBaseURL(t *testing.T) {
	html := `<a href="/next" class="next">Next</a>`

	ps := NewPaginationSelector("a.next")
	_, found := ps.FindNextPage(html, "://invalid")

	if found {
		t.Error("expected not to find next page with invalid base URL")
	}
}
