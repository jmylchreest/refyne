package crawler

import (
	"sync"
	"testing"
)

// --- URLQueue Tests ---

func TestURLQueue_Add_NewURL(t *testing.T) {
	q := NewURLQueue()

	added := q.Add("https://example.com/page1", 0)
	if !added {
		t.Error("Add() should return true for new URL")
	}

	if q.Len() != 1 {
		t.Errorf("expected queue length 1, got %d", q.Len())
	}
}

func TestURLQueue_Add_DuplicateURL(t *testing.T) {
	q := NewURLQueue()

	q.Add("https://example.com/page1", 0)
	added := q.Add("https://example.com/page1", 1)

	if added {
		t.Error("Add() should return false for duplicate URL")
	}

	if q.Len() != 1 {
		t.Errorf("expected queue length 1, got %d", q.Len())
	}
}

func TestURLQueue_Add_InvalidURL(t *testing.T) {
	q := NewURLQueue()

	added := q.Add("://invalid", 0)
	if added {
		t.Error("Add() should return false for invalid URL")
	}
}

func TestURLQueue_Pop_Empty(t *testing.T) {
	q := NewURLQueue()

	url, depth, ok := q.Pop()
	if ok {
		t.Error("Pop() should return false for empty queue")
	}

	if url != "" {
		t.Errorf("expected empty URL, got %q", url)
	}

	if depth != 0 {
		t.Errorf("expected depth 0, got %d", depth)
	}
}

func TestURLQueue_Pop_ReturnsFirst(t *testing.T) {
	q := NewURLQueue()

	q.Add("https://example.com/first", 0)
	q.Add("https://example.com/second", 1)
	q.Add("https://example.com/third", 2)

	url, depth, ok := q.Pop()
	if !ok {
		t.Fatal("Pop() should return true")
	}

	if url != "https://example.com/first" {
		t.Errorf("expected first URL, got %q", url)
	}

	if depth != 0 {
		t.Errorf("expected depth 0, got %d", depth)
	}

	// Queue should have 2 items left
	if q.Len() != 2 {
		t.Errorf("expected queue length 2, got %d", q.Len())
	}
}

func TestURLQueue_Pop_FIFO_Order(t *testing.T) {
	q := NewURLQueue()

	urls := []string{
		"https://example.com/1",
		"https://example.com/2",
		"https://example.com/3",
	}

	for i, url := range urls {
		q.Add(url, i)
	}

	for i, expected := range urls {
		url, depth, ok := q.Pop()
		if !ok {
			t.Fatalf("Pop() returned false at index %d", i)
		}
		if url != expected {
			t.Errorf("expected %q, got %q", expected, url)
		}
		if depth != i {
			t.Errorf("expected depth %d, got %d", i, depth)
		}
	}
}

func TestURLQueue_Len(t *testing.T) {
	q := NewURLQueue()

	if q.Len() != 0 {
		t.Errorf("expected length 0, got %d", q.Len())
	}

	q.Add("https://example.com/1", 0)
	if q.Len() != 1 {
		t.Errorf("expected length 1, got %d", q.Len())
	}

	q.Add("https://example.com/2", 0)
	if q.Len() != 2 {
		t.Errorf("expected length 2, got %d", q.Len())
	}

	q.Pop()
	if q.Len() != 1 {
		t.Errorf("expected length 1 after pop, got %d", q.Len())
	}
}

func TestURLQueue_IsVisited_NotVisited(t *testing.T) {
	q := NewURLQueue()

	if q.IsVisited("https://example.com/page") {
		t.Error("IsVisited() should return false for unvisited URL")
	}
}

func TestURLQueue_IsVisited_AfterAdd(t *testing.T) {
	q := NewURLQueue()

	q.Add("https://example.com/page", 0)

	if !q.IsVisited("https://example.com/page") {
		t.Error("IsVisited() should return true after Add()")
	}
}

func TestURLQueue_MarkVisited(t *testing.T) {
	q := NewURLQueue()

	q.MarkVisited("https://example.com/page")

	if !q.IsVisited("https://example.com/page") {
		t.Error("IsVisited() should return true after MarkVisited()")
	}

	// MarkVisited should not add to queue
	if q.Len() != 0 {
		t.Errorf("expected queue length 0, got %d", q.Len())
	}
}

func TestURLQueue_MarkVisited_PreventsAdd(t *testing.T) {
	q := NewURLQueue()

	q.MarkVisited("https://example.com/page")
	added := q.Add("https://example.com/page", 0)

	if added {
		t.Error("Add() should return false for visited URL")
	}

	if q.Len() != 0 {
		t.Errorf("expected queue length 0, got %d", q.Len())
	}
}

// --- normalizeURL Tests ---

func TestNormalizeURL_RemovesFragment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/page#section", "https://example.com/page"},
		{"https://example.com/page#", "https://example.com/page"},
		{"https://example.com/page", "https://example.com/page"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeURL(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeURL_RemovesTrailingSlash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/page/", "https://example.com/page"},
		{"https://example.com/path/to/page/", "https://example.com/path/to/page"},
		{"https://example.com/page", "https://example.com/page"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeURL(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeURL_PreservesRootSlash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/", "https://example.com/"},
		{"https://example.com", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeURL(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeURL_InvalidURL(t *testing.T) {
	got := normalizeURL("://invalid")
	if got != "" {
		t.Errorf("normalizeURL(invalid) = %q, want empty string", got)
	}
}

// --- IsSameDomain Tests ---

func TestIsSameDomain_Same(t *testing.T) {
	tests := []struct {
		url1     string
		url2     string
		expected bool
	}{
		{"https://example.com/page1", "https://example.com/page2", true},
		{"https://example.com/", "https://example.com/path/to/page", true},
		{"http://example.com/", "https://example.com/", true}, // Different scheme, same host
	}

	for _, tt := range tests {
		t.Run(tt.url1+" vs "+tt.url2, func(t *testing.T) {
			got := IsSameDomain(tt.url1, tt.url2)
			if got != tt.expected {
				t.Errorf("IsSameDomain(%q, %q) = %v, want %v", tt.url1, tt.url2, got, tt.expected)
			}
		})
	}
}

func TestIsSameDomain_Different(t *testing.T) {
	tests := []struct {
		url1     string
		url2     string
		expected bool
	}{
		{"https://example.com/", "https://other.com/", false},
		{"https://www.example.com/", "https://example.com/", false}, // subdomain difference
		{"https://example.com/", "https://sub.example.com/", false},
	}

	for _, tt := range tests {
		t.Run(tt.url1+" vs "+tt.url2, func(t *testing.T) {
			got := IsSameDomain(tt.url1, tt.url2)
			if got != tt.expected {
				t.Errorf("IsSameDomain(%q, %q) = %v, want %v", tt.url1, tt.url2, got, tt.expected)
			}
		})
	}
}

func TestIsSameDomain_WithPorts(t *testing.T) {
	tests := []struct {
		url1     string
		url2     string
		expected bool
	}{
		{"https://example.com:443/", "https://example.com:443/page", true},
		{"https://example.com:8080/", "https://example.com:8080/page", true},
		{"https://example.com:8080/", "https://example.com:443/page", false}, // Different ports
	}

	for _, tt := range tests {
		t.Run(tt.url1+" vs "+tt.url2, func(t *testing.T) {
			got := IsSameDomain(tt.url1, tt.url2)
			if got != tt.expected {
				t.Errorf("IsSameDomain(%q, %q) = %v, want %v", tt.url1, tt.url2, got, tt.expected)
			}
		})
	}
}

func TestIsSameDomain_InvalidURL(t *testing.T) {
	tests := []struct {
		url1 string
		url2 string
	}{
		{"://invalid", "https://example.com/"},
		{"https://example.com/", "://invalid"},
		{"://invalid", "://invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.url1+" vs "+tt.url2, func(t *testing.T) {
			got := IsSameDomain(tt.url1, tt.url2)
			if got {
				t.Errorf("IsSameDomain(%q, %q) = true, want false for invalid URLs", tt.url1, tt.url2)
			}
		})
	}
}

// --- Concurrency Safety Tests ---

func TestURLQueue_ConcurrentAccess(t *testing.T) {
	q := NewURLQueue()
	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			q.Add("https://example.com/page"+string(rune('0'+n%10)), n)
		}(i)
	}

	// Concurrent pops
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Pop()
		}()
	}

	// Concurrent IsVisited/MarkVisited
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			q.IsVisited("https://example.com/check" + string(rune('0'+n%10)))
			q.MarkVisited("https://example.com/mark" + string(rune('0'+n%10)))
		}(i)
	}

	wg.Wait()

	// Should not panic - test passes if we get here
}

// --- URLQueue Normalization Integration Tests ---

func TestURLQueue_NormalizesURLs(t *testing.T) {
	q := NewURLQueue()

	// Add URL with trailing slash
	q.Add("https://example.com/page/", 0)

	// Same URL without trailing slash should be considered duplicate
	added := q.Add("https://example.com/page", 0)
	if added {
		t.Error("Add() should normalize URLs and detect duplicates")
	}
}

func TestURLQueue_NormalizesFragments(t *testing.T) {
	q := NewURLQueue()

	// Add URL without fragment
	q.Add("https://example.com/page", 0)

	// Same URL with fragment should be considered duplicate
	added := q.Add("https://example.com/page#section", 0)
	if added {
		t.Error("Add() should normalize fragments and detect duplicates")
	}
}
