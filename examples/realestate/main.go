// Package main demonstrates real estate listing extraction with crawling.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/refyne/refyne/pkg/refyne"
	"github.com/refyne/refyne/pkg/schema"
)

// PropertyListing represents a real estate listing.
type PropertyListing struct {
	Title       string   `json:"title" description:"Property listing title"`
	Price       int      `json:"price" description:"Listing price (numeric, no currency symbol)"`
	Address     string   `json:"address" description:"Full property address"`
	Bedrooms    int      `json:"bedrooms,omitempty" description:"Number of bedrooms"`
	Bathrooms   float64  `json:"bathrooms,omitempty" description:"Number of bathrooms (can be 1.5, 2.5, etc.)"`
	SquareFeet  int      `json:"square_feet,omitempty" description:"Living area in square feet"`
	Description string   `json:"description,omitempty" description:"Property description"`
	Features    []string `json:"features,omitempty" description:"List of features (pool, garage, etc.)"`
	Images      []string `json:"images,omitempty" description:"URLs of property photos"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <search-url> [link-selector]")
		fmt.Println("Example: go run main.go 'https://example.com/search' 'a.listing-card'")
		os.Exit(1)
	}

	searchURL := os.Args[1]
	linkSelector := "a[href]" // default
	if len(os.Args) > 2 {
		linkSelector = os.Args[2]
	}

	ctx := context.Background()

	// Create schema
	s, err := schema.NewSchema[PropertyListing](
		schema.WithDescription("A real estate property listing page with details about the property for sale"),
	)
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Create refyne instance
	r, err := refyne.New(
		refyne.WithProvider("anthropic"),
	)
	if err != nil {
		log.Fatalf("Failed to create refyne: %v", err)
	}
	defer r.Close()

	fmt.Printf("Crawling from: %s\n", searchURL)
	fmt.Printf("Following links matching: %s\n", linkSelector)

	// Crawl and extract
	results := r.Crawl(ctx, searchURL, s,
		refyne.WithFollowSelector(linkSelector),
		refyne.WithMaxDepth(1),
		refyne.WithDelay(1*time.Second),
		refyne.WithConcurrency(2),
	)

	count := 0
	for result := range results {
		if result.Error != nil {
			fmt.Printf("Error: %s - %v\n", result.URL, result.Error)
			continue
		}

		if result.Data == nil {
			continue
		}

		listing := result.Data.(*PropertyListing)
		count++

		fmt.Printf("\n--- Listing %d ---\n", count)
		fmt.Printf("Title: %s\n", listing.Title)
		fmt.Printf("Price: $%d\n", listing.Price)
		fmt.Printf("Address: %s\n", listing.Address)
		if listing.Bedrooms > 0 {
			fmt.Printf("Bedrooms: %d\n", listing.Bedrooms)
		}
		if listing.Bathrooms > 0 {
			fmt.Printf("Bathrooms: %.1f\n", listing.Bathrooms)
		}
		if listing.SquareFeet > 0 {
			fmt.Printf("Sq Ft: %d\n", listing.SquareFeet)
		}
		if len(listing.Features) > 0 {
			fmt.Printf("Features: %v\n", listing.Features)
		}
		fmt.Printf("URL: %s\n", result.URL)
	}

	fmt.Printf("\nTotal listings extracted: %d\n", count)
}
