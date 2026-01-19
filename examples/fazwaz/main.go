// Package main demonstrates Thai property listing extraction from FazWaz using the SDK.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	clifetcher "github.com/jmylchreest/refyne/cmd/refyne/fetcher"
	"github.com/jmylchreest/refyne/pkg/refyne"
	"github.com/jmylchreest/refyne/pkg/schema"
)

// Price represents a price in a specific currency.
type Price struct {
	Currency string `json:"currency" description:"Currency code (USD, GBP, THB, EUR, etc.)"`
	Value    int    `json:"value" description:"Price value as integer"`
	Source   string `json:"source,omitempty" description:"'page' if from page, 'converted' if calculated"`
}

// FazWazListing represents a Thailand property listing from FazWaz.
type FazWazListing struct {
	Title            string   `json:"title" description:"Property title or headline"`
	Prices           []Price  `json:"prices" description:"Prices in all currencies shown, plus USD (converted if needed)"`
	PricePerSqm      []Price  `json:"price_per_sqm,omitempty" description:"Price per sqm in available currencies"`
	Address          string   `json:"address" description:"Property location (district, city, province)"`
	Province         string   `json:"province,omitempty" description:"Thai province name"`
	District         string   `json:"district,omitempty" description:"District name"`
	ProjectName      string   `json:"project_name,omitempty" description:"Development or project name"`
	Bedrooms         int      `json:"bedrooms,omitempty" description:"Number of bedrooms"`
	Bathrooms        int      `json:"bathrooms,omitempty" description:"Number of bathrooms"`
	AreaSqm          float64  `json:"area_sqm,omitempty" description:"Property area in square meters"`
	LandAreaSqm      float64  `json:"land_area_sqm,omitempty" description:"Land area in sqm (for houses/villas)"`
	PropertyType     string   `json:"property_type,omitempty" description:"House, Condo, Villa, Townhouse, Land"`
	Ownership        string   `json:"ownership,omitempty" description:"Freehold, Leasehold, Foreign Freehold"`
	CompletionStatus string   `json:"completion_status,omitempty" description:"Off Plan, Under Construction, Completed"`
	YearBuilt        int      `json:"year_built,omitempty" description:"Year built or expected completion"`
	Description      string   `json:"description,omitempty" description:"Property description (first 500 chars)"`
	Features         []string `json:"features,omitempty" description:"Key features: pool, gym, parking, etc."`
	NearbyPlaces     []string `json:"nearby_places,omitempty" description:"Nearby amenities, beaches, etc."`
	AgentName        string   `json:"agent_name,omitempty" description:"Listing agent or agency name"`
	ListingURL       string   `json:"listing_url,omitempty" description:"Full URL of this listing"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <url> [link-selector]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Single listing")
		fmt.Println("  go run main.go 'https://www.fazwaz.com/property-for-sale/thailand/123456'")
		fmt.Println()
		fmt.Println("  # Crawl search results")
		fmt.Println("  go run main.go 'https://www.fazwaz.com/property-for-sale/thailand/phuket' 'a[href*=\"/property-for-sale/\"]'")
		os.Exit(1)
	}

	url := os.Args[1]
	linkSelector := ""
	if len(os.Args) > 2 {
		linkSelector = os.Args[2]
	}

	ctx := context.Background()

	// Create schema from struct
	s, err := schema.NewSchema[FazWazListing](
		schema.WithDescription(`A FazWaz Thailand property listing page.
Extract all prices shown in their respective currencies.
ALWAYS include USD - convert from THB if not shown (1 USD = 35 THB).`),
	)
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Create dynamic fetcher for JS-heavy site
	fetcher, err := clifetcher.NewDynamicFetcher(clifetcher.Config{
		Timeout: 60 * time.Second,
		Stealth: true,
	})
	if err != nil {
		log.Fatalf("Failed to create fetcher: %v", err)
	}

	// Create refyne instance
	r, err := refyne.New(
		refyne.WithFetcher(fetcher),
	)
	if err != nil {
		log.Fatalf("Failed to create refyne: %v", err)
	}
	defer func() { _ = r.Close() }()

	fmt.Printf("Provider: %s\n", r.Provider())
	fmt.Printf("URL: %s\n", url)
	if linkSelector != "" {
		fmt.Printf("Following: %s\n", linkSelector)
	}
	fmt.Println()

	// Single extraction or crawl
	var results <-chan *refyne.Result
	if linkSelector == "" {
		// Single URL extraction
		result, err := r.Extract(ctx, url, s)
		if err != nil {
			log.Fatalf("Extraction failed: %v", err)
		}
		ch := make(chan *refyne.Result, 1)
		ch <- result
		close(ch)
		results = ch
	} else {
		// Crawl with link following
		results = r.Crawl(ctx, url, s,
			refyne.WithFollowSelector(linkSelector),
			refyne.WithFollowPattern(`/[0-9]+$`), // Only follow property detail pages
			refyne.WithMaxDepth(1),
			refyne.WithMaxURLs(10),
			refyne.WithDelay(2*time.Second),
			refyne.WithConcurrency(2),
		)
	}

	count := 0
	for result := range results {
		if result.Error != nil {
			fmt.Printf("Error: %s - %v\n", result.URL, result.Error)
			continue
		}

		if result.Data == nil {
			continue
		}

		listing := result.Data.(*FazWazListing)
		count++

		fmt.Printf("--- Listing %d ---\n", count)
		fmt.Printf("Title: %s\n", listing.Title)

		// Print prices
		for _, p := range listing.Prices {
			src := ""
			if p.Source != "" {
				src = fmt.Sprintf(" (%s)", p.Source)
			}
			fmt.Printf("Price: %s %d%s\n", p.Currency, p.Value, src)
		}

		// Location
		fmt.Printf("Location: %s\n", listing.Address)
		if listing.Province != "" {
			fmt.Printf("Province: %s\n", listing.Province)
		}
		if listing.ProjectName != "" {
			fmt.Printf("Project: %s\n", listing.ProjectName)
		}

		// Property details
		if listing.Bedrooms > 0 {
			fmt.Printf("Bedrooms: %d\n", listing.Bedrooms)
		}
		if listing.Bathrooms > 0 {
			fmt.Printf("Bathrooms: %d\n", listing.Bathrooms)
		}
		if listing.AreaSqm > 0 {
			fmt.Printf("Area: %.1f sqm\n", listing.AreaSqm)
		}
		if listing.PropertyType != "" {
			fmt.Printf("Type: %s\n", listing.PropertyType)
		}
		if listing.Ownership != "" {
			fmt.Printf("Ownership: %s\n", listing.Ownership)
		}
		if len(listing.Features) > 0 {
			fmt.Printf("Features: %v\n", listing.Features)
		}

		fmt.Printf("URL: %s\n", result.URL)
		fmt.Printf("Tokens: %d in, %d out\n", result.TokenUsage.InputTokens, result.TokenUsage.OutputTokens)
		fmt.Println()

		// Output first listing as JSON for verification
		if count == 1 {
			jsonBytes, _ := json.MarshalIndent(listing, "", "  ")
			fmt.Printf("Full JSON:\n%s\n\n", string(jsonBytes))
		}
	}

	fmt.Printf("Total listings: %d\n", count)
}
