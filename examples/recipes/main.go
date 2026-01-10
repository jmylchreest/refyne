// Package main demonstrates recipe extraction from cooking websites.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/refyne/refyne/pkg/refyne"
	"github.com/refyne/refyne/pkg/schema"
)

// Recipe represents a cooking recipe.
type Recipe struct {
	Title       string       `json:"title" description:"The name of the recipe"`
	Description string       `json:"description,omitempty" description:"Brief description of the dish"`
	PrepTime    string       `json:"prep_time,omitempty" description:"Preparation time (e.g., '15 minutes')"`
	CookTime    string       `json:"cook_time,omitempty" description:"Cooking time (e.g., '30 minutes')"`
	Servings    int          `json:"servings,omitempty" description:"Number of servings"`
	Ingredients []Ingredient `json:"ingredients" description:"List of ingredients needed"`
	Steps       []string     `json:"steps" description:"Step-by-step cooking instructions"`
}

// Ingredient represents a single ingredient.
type Ingredient struct {
	Name   string `json:"name" description:"Ingredient name"`
	Amount string `json:"amount,omitempty" description:"Quantity (e.g., '2 cups', '1 tablespoon')"`
	Notes  string `json:"notes,omitempty" description:"Additional notes (e.g., 'finely chopped')"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <recipe-url>")
		fmt.Println("Example: go run main.go https://www.allrecipes.com/recipe/10813/best-chocolate-chip-cookies/")
		os.Exit(1)
	}

	url := os.Args[1]
	ctx := context.Background()

	// Create schema from Go struct
	s, err := schema.NewSchema[Recipe](
		schema.WithDescription("A cooking recipe page with ingredients and step-by-step instructions"),
	)
	if err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Create refyne instance (uses ANTHROPIC_API_KEY env var by default)
	r, err := refyne.New(
		refyne.WithProvider("anthropic"),
	)
	if err != nil {
		log.Fatalf("Failed to create refyne: %v", err)
	}
	defer func() { _ = r.Close() }()

	fmt.Printf("Extracting recipe from: %s\n", url)

	// Extract data
	result, err := r.Extract(ctx, url, s)
	if err != nil {
		log.Fatalf("Extraction failed: %v", err)
	}

	// Print the extracted recipe
	recipe := result.Data.(*Recipe)

	fmt.Printf("\n=== %s ===\n", recipe.Title)
	if recipe.Description != "" {
		fmt.Printf("Description: %s\n", recipe.Description)
	}
	if recipe.PrepTime != "" {
		fmt.Printf("Prep Time: %s\n", recipe.PrepTime)
	}
	if recipe.CookTime != "" {
		fmt.Printf("Cook Time: %s\n", recipe.CookTime)
	}
	if recipe.Servings > 0 {
		fmt.Printf("Servings: %d\n", recipe.Servings)
	}

	fmt.Println("\nIngredients:")
	for _, ing := range recipe.Ingredients {
		if ing.Amount != "" {
			fmt.Printf("  - %s %s", ing.Amount, ing.Name)
		} else {
			fmt.Printf("  - %s", ing.Name)
		}
		if ing.Notes != "" {
			fmt.Printf(" (%s)", ing.Notes)
		}
		fmt.Println()
	}

	fmt.Println("\nInstructions:")
	for i, step := range recipe.Steps {
		fmt.Printf("  %d. %s\n", i+1, step)
	}

	fmt.Printf("\nTokens used: %d input, %d output\n",
		result.TokenUsage.InputTokens, result.TokenUsage.OutputTokens)
}
