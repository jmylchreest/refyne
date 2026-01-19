// Package main demonstrates UK property listing extraction from Zoopla using the SDK.
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

// ZooplaListing represents a UK property listing from Zoopla.
type ZooplaListing struct {
	Title          string   `json:"title" description:"Property title or headline"`
	Price          int      `json:"price" description:"Listing price in GBP (numeric only)"`
	PriceQualifier string   `json:"price_qualifier,omitempty" description:"Guide price, Offers over, etc."`
	Address        string   `json:"address" description:"Full property address including postcode"`
	Bedrooms       int      `json:"bedrooms,omitempty" description:"Number of bedrooms"`
	Bathrooms      int      `json:"bathrooms,omitempty" description:"Number of bathrooms"`
	Receptions     int      `json:"receptions,omitempty" description:"Number of reception rooms"`
	SquareFeet     int      `json:"square_feet,omitempty" description:"Property size in square feet"`
	PropertyType   string   `json:"property_type,omitempty" description:"detached, semi-detached, terraced, flat, etc."`
	Tenure         string   `json:"tenure,omitempty" description:"Freehold, Leasehold, Share of Freehold"`
	ChainStatus    string   `json:"chain_status,omitempty" description:"Chain free, No onward chain, etc."`
	Status         string   `json:"status,omitempty" description:"New, Reduced, Under offer, Sold STC, etc."`
	Description    string   `json:"description,omitempty" description:"Full property description"`
	Features       []string `json:"features,omitempty" description:"Key features (garden, parking, garage, etc.)"`
	Images         []string `json:"images,omitempty" description:"URLs of property photos"`
	AgentName      string   `json:"agent_name,omitempty" description:"Estate agent name"`
	AgentPhone     string   `json:"agent_phone,omitempty" description:"Estate agent phone number"`
	EPCRating      string   `json:"epc_rating,omitempty" description:"Energy Performance Certificate rating (A-G)"`
	CouncilTaxBand string   `json:"council_tax_band,omitempty" description:"Council tax band (A-H)"`
	URL            string   `json:"url,omitempty" description:"Full URL of the listing"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <url> [link-selector]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Single listing")
		fmt.Println("  go run main.go 'https://www.zoopla.co.uk/for-sale/details/12345678'")
		fmt.Println()
		fmt.Println("  # Crawl search results")
		fmt.Println("  go run main.go 'https://www.zoopla.co.uk/for-sale/property/london/' 'a[href*=\"/for-sale/details/\"]:not([href*=\"/contact/\"])'")
		os.Exit(1)
	}

	url := os.Args[1]
	linkSelector := ""
	if len(os.Args) > 2 {
		linkSelector = os.Args[2]
	}

	ctx := context.Background()

	// Create schema from struct
	s, err := schema.NewSchema[ZooplaListing](
		schema.WithDescription(`A Zoopla UK property listing page. Extract property details including
price in GBP, bedrooms, bathrooms, and UK-specific fields like tenure and chain status.`),
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

		listing := result.Data.(*ZooplaListing)
		count++

		fmt.Printf("--- Listing %d ---\n", count)
		fmt.Printf("Title: %s\n", listing.Title)

		// Price
		if listing.PriceQualifier != "" {
			fmt.Printf("Price: %s £%d\n", listing.PriceQualifier, listing.Price)
		} else {
			fmt.Printf("Price: £%d\n", listing.Price)
		}

		// Address
		fmt.Printf("Address: %s\n", listing.Address)

		// Property details
		if listing.Bedrooms > 0 {
			fmt.Printf("Bedrooms: %d\n", listing.Bedrooms)
		}
		if listing.Bathrooms > 0 {
			fmt.Printf("Bathrooms: %d\n", listing.Bathrooms)
		}
		if listing.SquareFeet > 0 {
			fmt.Printf("Size: %d sq ft\n", listing.SquareFeet)
		}
		if listing.PropertyType != "" {
			fmt.Printf("Type: %s\n", listing.PropertyType)
		}
		if listing.Tenure != "" {
			fmt.Printf("Tenure: %s\n", listing.Tenure)
		}
		if listing.ChainStatus != "" {
			fmt.Printf("Chain: %s\n", listing.ChainStatus)
		}
		if len(listing.Features) > 0 {
			fmt.Printf("Features: %v\n", listing.Features)
		}
		if listing.AgentName != "" {
			fmt.Printf("Agent: %s\n", listing.AgentName)
		}

		fmt.Printf("URL: %s\n", result.URL)
		fmt.Printf("Tokens: %d in, %d out\n", result.TokenUsage.InputTokens, result.TokenUsage.OutputTokens)
		fmt.Println()

		// Output first listing as JSON
		if count == 1 {
			jsonBytes, _ := json.MarshalIndent(listing, "", "  ")
			fmt.Printf("Full JSON:\n%s\n\n", string(jsonBytes))
		}
	}

	fmt.Printf("Total listings: %d\n", count)
}
