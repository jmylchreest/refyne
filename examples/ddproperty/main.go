// Package main demonstrates Thai property listing extraction from DDProperty.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/refyne/refyne/internal/scraper"
	"github.com/refyne/refyne/pkg/refyne"
	"github.com/refyne/refyne/pkg/schema"
)

// DDPropertyListing represents a Thai property listing.
type DDPropertyListing struct {
	Title           string   `json:"title" description:"Property title (may be in Thai)"`
	PriceTHB        int      `json:"price_thb" description:"Listing price in Thai Baht (numeric only)"`
	PriceGBPApprox  int      `json:"price_gbp_approx,omitempty" description:"Approximate GBP value (1 GBP = 45 THB)"`
	Address         string   `json:"address" description:"Property address or location"`
	Province        string   `json:"province,omitempty" description:"Thai province (e.g. Krabi, Phuket)"`
	District        string   `json:"district,omitempty" description:"District or amphoe name"`
	Bedrooms        int      `json:"bedrooms,omitempty" description:"Number of bedrooms"`
	Bathrooms       int      `json:"bathrooms,omitempty" description:"Number of bathrooms"`
	LandAreaSqm     float64  `json:"land_area_sqm,omitempty" description:"Land area in square meters"`
	LandAreaRai     float64  `json:"land_area_rai,omitempty" description:"Land area in rai (1 rai = 1600 sqm)"`
	BuildingAreaSqm float64  `json:"building_area_sqm,omitempty" description:"Building area in square meters"`
	PropertyType    string   `json:"property_type,omitempty" description:"house, condo, land, villa, townhouse"`
	ListingType     string   `json:"listing_type,omitempty" description:"sale or rent"`
	Description     string   `json:"description,omitempty" description:"Property description (translate to English if Thai)"`
	Features        []string `json:"features,omitempty" description:"Features (pool, garden, sea view, etc.)"`
	Nearby          []string `json:"nearby,omitempty" description:"Nearby amenities or landmarks"`
	AgentName       string   `json:"agent_name,omitempty" description:"Agent or developer name"`
	AgentPhone      string   `json:"agent_phone,omitempty" description:"Agent phone number"`
	Images          []string `json:"images,omitempty" description:"URLs of property photos"`
	URL             string   `json:"url,omitempty" description:"Full listing URL"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <search-url> [link-selector]")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  go run main.go 'https://www.ddproperty.com/...' 'a[href*=\"/property/\"]'")
		os.Exit(1)
	}

	searchURL := os.Args[1]
	linkSelector := "a[href*='/property/']" // default for ddproperty
	if len(os.Args) > 2 {
		linkSelector = os.Args[2]
	}

	ctx := context.Background()

	// Create schema from struct
	s, err := schema.NewSchema[DDPropertyListing](
		schema.WithDescription("A DDProperty Thailand property listing. Content may be in Thai. Extract prices in THB and convert to approximate GBP at 1 GBP = 45 THB."),
	)
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Create refyne instance with dynamic fetching for JS-heavy site
	r, err := refyne.New(
		refyne.WithProvider("anthropic"),
		refyne.WithFetchMode(scraper.FetchModeDynamic),
	)
	if err != nil {
		log.Fatalf("Failed to create refyne: %v", err)
	}
	defer func() { _ = r.Close() }()

	fmt.Printf("Crawling DDProperty from: %s\n", searchURL)
	fmt.Printf("Following links matching: %s\n", linkSelector)

	// Crawl and extract
	results := r.Crawl(ctx, searchURL, s,
		refyne.WithFollowSelector(linkSelector),
		refyne.WithMaxDepth(1),
		refyne.WithMaxURLs(10),
		refyne.WithDelay(2*time.Second),
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

		listing := result.Data.(*DDPropertyListing)
		count++

		fmt.Printf("\n--- Listing %d ---\n", count)
		fmt.Printf("Title: %s\n", listing.Title)
		fmt.Printf("Price: %d THB", listing.PriceTHB)
		if listing.PriceGBPApprox > 0 {
			fmt.Printf(" (~%d GBP)", listing.PriceGBPApprox)
		}
		fmt.Println()
		fmt.Printf("Location: %s", listing.Address)
		if listing.Province != "" {
			fmt.Printf(", %s", listing.Province)
		}
		fmt.Println()
		if listing.Bedrooms > 0 {
			fmt.Printf("Bedrooms: %d\n", listing.Bedrooms)
		}
		if listing.Bathrooms > 0 {
			fmt.Printf("Bathrooms: %d\n", listing.Bathrooms)
		}
		if listing.LandAreaRai > 0 {
			fmt.Printf("Land: %.2f rai\n", listing.LandAreaRai)
		} else if listing.LandAreaSqm > 0 {
			fmt.Printf("Land: %.0f sqm\n", listing.LandAreaSqm)
		}
		if listing.PropertyType != "" {
			fmt.Printf("Type: %s\n", listing.PropertyType)
		}
		if len(listing.Features) > 0 {
			fmt.Printf("Features: %v\n", listing.Features)
		}
		fmt.Printf("URL: %s\n", result.URL)
	}

	fmt.Printf("\nTotal listings extracted: %d\n", count)
}
