// Package main demonstrates Thai property listing extraction from DDProperty using the SDK.
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

// BilingualText represents text with English and optional Thai versions.
type BilingualText struct {
	EN string `json:"en" description:"English text (translated if original was Thai)"`
	TH string `json:"th,omitempty" description:"Original Thai text if present"`
}

// Price represents a price in a specific currency.
type Price struct {
	Currency string `json:"currency" description:"Currency code: THB, USD, or GBP"`
	Value    int    `json:"value" description:"Price value as integer"`
	Source   string `json:"source" description:"'page' if from page, 'converted' if calculated"`
}

// Location represents property location details.
type Location struct {
	Address     string `json:"address" description:"Full address in English with Thai in parentheses"`
	District    string `json:"district,omitempty" description:"District - format: 'English (ไทย)'"`
	Province    string `json:"province,omitempty" description:"Province - format: 'English (ไทย)'"`
	Subdistrict string `json:"subdistrict,omitempty" description:"Subdistrict if shown"`
}

// Agent represents agent/seller information.
type Agent struct {
	Name  string `json:"name,omitempty" description:"Agent or seller name"`
	Phone string `json:"phone,omitempty" description:"Phone number if displayed"`
	Type  string `json:"type,omitempty" description:"Agent, Developer, or Owner"`
}

// DDPropertyListing represents a Thai property listing with bilingual support.
type DDPropertyListing struct {
	Title            BilingualText  `json:"title" description:"Property title with English and Thai"`
	ProjectName      *BilingualText `json:"project_name,omitempty" description:"Development/project name"`
	Prices           []Price        `json:"prices" description:"Prices in THB (page) and USD/GBP (converted)"`
	PricePerSqm      []Price        `json:"price_per_sqm,omitempty" description:"Price per sqm in multiple currencies"`
	PriceQualifier   string         `json:"price_qualifier,omitempty" description:"negotiable, starting from, etc."`
	Location         Location       `json:"location" description:"Property location details"`
	Bedrooms         int            `json:"bedrooms,omitempty" description:"Number of bedrooms"`
	Bathrooms        int            `json:"bathrooms,omitempty" description:"Number of bathrooms"`
	AreaSqm          float64        `json:"area_sqm,omitempty" description:"Usable area in square meters"`
	LandAreaSqm      float64        `json:"land_area_sqm,omitempty" description:"Land area in square meters"`
	LandAreaRai      float64        `json:"land_area_rai,omitempty" description:"Land area in rai (1 rai = 1600 sqm)"`
	PropertyType     string         `json:"property_type,omitempty" description:"Condo, House, Villa, Townhouse, Land"`
	ListingType      string         `json:"listing_type,omitempty" description:"Sale or Rent"`
	Ownership        string         `json:"ownership,omitempty" description:"Freehold, Leasehold"`
	CompletionStatus string         `json:"completion_status,omitempty" description:"Completed, Under Construction, Off Plan"`
	YearBuilt        int            `json:"year_built,omitempty" description:"Gregorian year (Thai year - 543)"`
	Furnishing       string         `json:"furnishing,omitempty" description:"Fully/Partially Furnished, Unfurnished"`
	Description      string         `json:"description,omitempty" description:"English description"`
	Features         []string       `json:"features,omitempty" description:"Property features in English"`
	Facilities       []string       `json:"facilities,omitempty" description:"Building facilities in English"`
	Nearby           []string       `json:"nearby,omitempty" description:"Nearby places"`
	Agent            *Agent         `json:"agent,omitempty" description:"Agent/seller information"`
	ListingID        string         `json:"listing_id,omitempty" description:"DDProperty listing ID"`
	ListingDate      string         `json:"listing_date,omitempty" description:"ISO date YYYY-MM-DD"`
	Images           []string       `json:"images,omitempty" description:"Photo URLs (first 10)"`
	URL              string         `json:"url,omitempty" description:"Full listing URL"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <url> [link-selector]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Single listing")
		fmt.Println("  go run main.go 'https://www.ddproperty.com/property/...'")
		fmt.Println()
		fmt.Println("  # Crawl search results")
		fmt.Println("  go run main.go 'https://www.ddproperty.com/...' 'a[href*=\"/property/\"]'")
		os.Exit(1)
	}

	url := os.Args[1]
	linkSelector := ""
	if len(os.Args) > 2 {
		linkSelector = os.Args[2]
	}

	ctx := context.Background()

	// Create schema from struct with description for LLM
	s, err := schema.NewSchema[DDPropertyListing](
		schema.WithDescription(`A DDProperty Thailand property listing.
OUTPUT REQUIREMENTS:
1. ALL text in English
2. For proper nouns (project names, places): "English (ไทย)" format
3. Include THB price from page, plus USD and GBP converted (1 USD = 35 THB, 1 GBP = 45 THB)
4. Convert Thai Buddhist year to Gregorian by subtracting 543`),
	)
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Create dynamic fetcher for JS-heavy site with FlareSolverr for Cloudflare
	fetcher, err := clifetcher.NewDynamicFetcher(clifetcher.Config{
		Timeout:         60 * time.Second,
		Stealth:         true,
		FlareSolverrURL: os.Getenv("FLARESOLVERR_URL"), // Set to http://localhost:8191/v1
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

		listing := result.Data.(*DDPropertyListing)
		count++

		fmt.Printf("--- Listing %d ---\n", count)
		fmt.Printf("Title: %s\n", listing.Title.EN)
		if listing.Title.TH != "" {
			fmt.Printf("       (%s)\n", listing.Title.TH)
		}

		// Print prices
		for _, p := range listing.Prices {
			fmt.Printf("Price: %s %d (%s)\n", p.Currency, p.Value, p.Source)
		}

		// Location
		fmt.Printf("Location: %s\n", listing.Location.Address)
		if listing.Location.Province != "" {
			fmt.Printf("Province: %s\n", listing.Location.Province)
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

		// Also output as JSON for verification
		if count == 1 {
			jsonBytes, _ := json.MarshalIndent(listing, "", "  ")
			fmt.Printf("Full JSON:\n%s\n\n", string(jsonBytes))
		}
	}

	fmt.Printf("Total listings: %d\n", count)
}
