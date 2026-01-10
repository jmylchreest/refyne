// Package crawler handles multi-page crawling and navigation.
package crawler

import (
	"net/url"
	"sync"
)

// URLQueue manages URLs to be crawled with deduplication.
type URLQueue struct {
	mu      sync.Mutex
	queue   []queueItem
	visited map[string]bool
}

type queueItem struct {
	URL   string
	Depth int
}

// NewURLQueue creates a new URL queue.
func NewURLQueue() *URLQueue {
	return &URLQueue{
		queue:   make([]queueItem, 0),
		visited: make(map[string]bool),
	}
}

// Add adds a URL to the queue if not already visited.
func (q *URLQueue) Add(rawURL string, depth int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Normalize URL
	normalized := normalizeURL(rawURL)
	if normalized == "" {
		return false
	}

	// Check if already visited or queued
	if q.visited[normalized] {
		return false
	}

	q.visited[normalized] = true
	q.queue = append(q.queue, queueItem{URL: normalized, Depth: depth})
	return true
}

// Pop removes and returns the next URL from the queue.
func (q *URLQueue) Pop() (string, int, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		return "", 0, false
	}

	item := q.queue[0]
	q.queue = q.queue[1:]
	return item.URL, item.Depth, true
}

// Len returns the number of items in the queue.
func (q *URLQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue)
}

// IsVisited checks if a URL has been visited.
func (q *URLQueue) IsVisited(rawURL string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.visited[normalizeURL(rawURL)]
}

// MarkVisited marks a URL as visited without adding to queue.
func (q *URLQueue) MarkVisited(rawURL string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.visited[normalizeURL(rawURL)] = true
}

// normalizeURL normalizes a URL for comparison.
func normalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	// Remove fragment
	parsed.Fragment = ""

	// Remove trailing slash from path (unless it's just "/")
	if len(parsed.Path) > 1 && parsed.Path[len(parsed.Path)-1] == '/' {
		parsed.Path = parsed.Path[:len(parsed.Path)-1]
	}

	return parsed.String()
}

// IsSameDomain checks if two URLs are on the same domain.
func IsSameDomain(url1, url2 string) bool {
	parsed1, err := url.Parse(url1)
	if err != nil {
		return false
	}
	parsed2, err := url.Parse(url2)
	if err != nil {
		return false
	}
	return parsed1.Host == parsed2.Host
}
