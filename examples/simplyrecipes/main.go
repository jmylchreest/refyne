// Package main demonstrates recipe extraction from Simply Recipes using the SDK.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/refyne/refyne/pkg/refyne"
	"github.com/refyne/refyne/pkg/schema"
)

// Ingredient represents a single ingredient with amount and notes.
type Ingredient struct {
	Amount string `json:"amount" description:"Quantity and unit (e.g., '2 cups', '1 tbsp')"`
	Name   string `json:"name" description:"Ingredient name"`
	Notes  string `json:"notes,omitempty" description:"Additional notes (e.g., 'softened', 'chopped')"`
}

// Recipe represents a recipe extracted from a cooking website.
type Recipe struct {
	Title        string       `json:"title" description:"Recipe name/title"`
	Description  string       `json:"description,omitempty" description:"Brief description or introduction"`
	Author       string       `json:"author,omitempty" description:"Recipe author name"`
	PrepTime     string       `json:"prep_time,omitempty" description:"Preparation time (e.g., '20 minutes')"`
	CookTime     string       `json:"cook_time,omitempty" description:"Cooking/baking time"`
	TotalTime    string       `json:"total_time,omitempty" description:"Total time from start to finish"`
	Servings     string       `json:"servings,omitempty" description:"Number of servings or yield"`
	Ingredients  []Ingredient `json:"ingredients,omitempty" description:"List of ingredients with amounts"`
	Instructions []string     `json:"instructions,omitempty" description:"Step-by-step cooking instructions"`
	Notes        []string     `json:"notes,omitempty" description:"Recipe tips, variations, or storage instructions"`
	Category     string       `json:"category,omitempty" description:"Recipe category (Dinners, Desserts, etc.)"`
	Cuisine      string       `json:"cuisine,omitempty" description:"Cuisine type (Italian, Mexican, etc.)"`
	ImageURL     string       `json:"image_url,omitempty" description:"URL of the main recipe image"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <url> [link-selector]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Single recipe")
		fmt.Println("  go run main.go 'https://www.simplyrecipes.com/recipes/...'")
		fmt.Println()
		fmt.Println("  # Crawl recipe category")
		fmt.Println("  go run main.go 'https://www.simplyrecipes.com/dinner-recipes' 'a[href*=\"/recipes/\"]'")
		os.Exit(1)
	}

	url := os.Args[1]
	linkSelector := ""
	if len(os.Args) > 2 {
		linkSelector = os.Args[2]
	}

	ctx := context.Background()

	// Create schema from struct
	s, err := schema.NewSchema[Recipe](
		schema.WithDescription(`A recipe page. Extract title, ingredients with amounts, and step-by-step instructions.
Ingredients should include amount, name, and any preparation notes (like 'softened' or 'chopped').`),
	)
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Create refyne instance (uses default static fetcher - good for recipe sites)
	r, err := refyne.New()
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
			refyne.WithMaxURLs(5),
			refyne.WithDelay(1*time.Second),
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

		recipe := result.Data.(*Recipe)
		count++

		fmt.Printf("--- Recipe %d ---\n", count)
		fmt.Printf("Title: %s\n", recipe.Title)
		if recipe.Author != "" {
			fmt.Printf("Author: %s\n", recipe.Author)
		}

		// Timing
		if recipe.PrepTime != "" || recipe.CookTime != "" || recipe.TotalTime != "" {
			fmt.Printf("Time: ")
			if recipe.PrepTime != "" {
				fmt.Printf("Prep %s", recipe.PrepTime)
			}
			if recipe.CookTime != "" {
				if recipe.PrepTime != "" {
					fmt.Printf(" | ")
				}
				fmt.Printf("Cook %s", recipe.CookTime)
			}
			if recipe.TotalTime != "" {
				if recipe.PrepTime != "" || recipe.CookTime != "" {
					fmt.Printf(" | ")
				}
				fmt.Printf("Total %s", recipe.TotalTime)
			}
			fmt.Println()
		}

		if recipe.Servings != "" {
			fmt.Printf("Servings: %s\n", recipe.Servings)
		}

		// Ingredients summary
		if len(recipe.Ingredients) > 0 {
			fmt.Printf("Ingredients: %d items\n", len(recipe.Ingredients))
			for i, ing := range recipe.Ingredients {
				if i < 5 { // Show first 5
					fmt.Printf("  - %s %s", ing.Amount, ing.Name)
					if ing.Notes != "" {
						fmt.Printf(" (%s)", ing.Notes)
					}
					fmt.Println()
				}
			}
			if len(recipe.Ingredients) > 5 {
				fmt.Printf("  ... and %d more\n", len(recipe.Ingredients)-5)
			}
		}

		// Instructions summary
		if len(recipe.Instructions) > 0 {
			fmt.Printf("Instructions: %d steps\n", len(recipe.Instructions))
		}

		fmt.Printf("URL: %s\n", result.URL)
		fmt.Printf("Tokens: %d in, %d out\n", result.TokenUsage.InputTokens, result.TokenUsage.OutputTokens)
		fmt.Println()

		// Output first recipe as JSON
		if count == 1 {
			jsonBytes, _ := json.MarshalIndent(recipe, "", "  ")
			fmt.Printf("Full JSON:\n%s\n\n", string(jsonBytes))
		}
	}

	fmt.Printf("Total recipes: %d\n", count)
}
