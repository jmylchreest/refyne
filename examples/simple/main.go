// Package main demonstrates basic webpage extraction using the Refyne SDK.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jmylchreest/refyne/pkg/refyne"
	"github.com/jmylchreest/refyne/pkg/schema"
)

// WebPage represents basic information extracted from any webpage.
type WebPage struct {
	Title          string   `json:"title" description:"The page title"`
	Summary        string   `json:"summary" description:"Brief summary of main content (2-3 sentences)"`
	KeyPoints      []string `json:"key_points,omitempty" description:"Key facts, data points, or takeaways"`
	LinksMentioned []string `json:"links_mentioned,omitempty" description:"Important external resources mentioned"`
	Author         string   `json:"author,omitempty" description:"Author name if present"`
	Date           string   `json:"date,omitempty" description:"Publication or last updated date"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <url>")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  go run main.go 'https://example.com'")
		os.Exit(1)
	}

	url := os.Args[1]
	ctx := context.Background()

	// Create schema from struct
	s, err := schema.NewSchema[WebPage](
		schema.WithDescription("Extract basic information from any webpage including title, summary, and key points."),
	)
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Create refyne instance (uses default static fetcher)
	r, err := refyne.New()
	if err != nil {
		log.Fatalf("Failed to create refyne: %v", err)
	}
	defer func() { _ = r.Close() }()

	fmt.Printf("Provider: %s\n", r.Provider())
	fmt.Printf("URL: %s\n\n", url)

	// Extract data
	result, err := r.Extract(ctx, url, s)
	if err != nil {
		log.Fatalf("Extraction failed: %v", err)
	}

	page := result.Data.(*WebPage)

	// Print results
	fmt.Printf("Title: %s\n", page.Title)
	fmt.Printf("Summary: %s\n", page.Summary)

	if len(page.KeyPoints) > 0 {
		fmt.Println("\nKey Points:")
		for _, point := range page.KeyPoints {
			fmt.Printf("  - %s\n", point)
		}
	}

	if page.Author != "" {
		fmt.Printf("\nAuthor: %s\n", page.Author)
	}
	if page.Date != "" {
		fmt.Printf("Date: %s\n", page.Date)
	}

	fmt.Printf("\nTokens: %d in, %d out\n", result.TokenUsage.InputTokens, result.TokenUsage.OutputTokens)

	// Output as JSON
	fmt.Println("\nFull JSON:")
	jsonBytes, _ := json.MarshalIndent(page, "", "  ")
	fmt.Println(string(jsonBytes))
}
