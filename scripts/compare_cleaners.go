// compare_cleaners.go - Compare output from different cleaner methods
//
// Usage: go run scripts/compare_cleaners.go <url>
//
// Example:
//   go run scripts/compare_cleaners.go https://www.instructables.com/Build-a-simple-shed-a-complete-guide/
//   go run scripts/compare_cleaners.go https://demo.refyne.uk/products

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jmylchreest/refyne/pkg/cleaner"
	refynecleaner "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/compare_cleaners.go <url>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run scripts/compare_cleaners.go https://www.instructables.com/Build-a-simple-shed-a-complete-guide/")
		fmt.Println("  go run scripts/compare_cleaners.go https://demo.refyne.uk/products")
		os.Exit(1)
	}

	url := os.Args[1]

	fmt.Println("Fetching:", url)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching URL: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "HTTP error: %s\n", resp.Status)
		os.Exit(1)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	fmt.Printf("Input size: %d bytes\n\n", len(html))

	// =====================================================
	// Method 1: Our new refyne cleaner with built-in markdown
	// =====================================================
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("METHOD 1: Refyne cleaner with built-in markdown + frontmatter")
	fmt.Println("=" + strings.Repeat("=", 60))

	cfg1 := refynecleaner.DefaultConfig()
	cfg1.Output = refynecleaner.OutputMarkdown
	cfg1.IncludeFrontmatter = true
	cfg1.ExtractImages = true
	cfg1.ExtractHeadings = true
	cfg1.IncludeHints = true
	cfg1.BaseURL = url

	cleaner1 := refynecleaner.New(cfg1)
	result1 := cleaner1.CleanWithStats(html)

	fmt.Printf("Output size: %d bytes (%.1f%% reduction)\n",
		result1.Stats.OutputBytes, result1.Stats.ReductionPercent())

	if result1.Metadata != nil {
		fmt.Printf("Extracted: %d images, %d headings\n",
			len(result1.Metadata.Images), len(result1.Metadata.Headings))
	}

	// Count markdown images in output
	mdImages1 := strings.Count(result1.Content, "![")
	fmt.Printf("Markdown images in body: %d\n", mdImages1)

	// Save to file for comparison
	os.WriteFile("/tmp/method1_refyne_md.md", []byte(result1.Content), 0644)
	fmt.Println("Saved to: /tmp/method1_refyne_md.md")

	// Show preview
	fmt.Println("\n--- Preview (first 2000 chars) ---")
	preview1 := result1.Content
	if len(preview1) > 2000 {
		preview1 = preview1[:2000] + "\n..."
	}
	fmt.Println(preview1)

	// =====================================================
	// Method 2: Chained refyne(html) -> html-to-markdown
	// =====================================================
	fmt.Println("\n" + strings.Repeat("=", 61))
	fmt.Println("METHOD 2: Refyne cleaner (HTML) -> html-to-markdown library")
	fmt.Println(strings.Repeat("=", 61))

	cfg2 := refynecleaner.DefaultConfig()
	cfg2.Output = refynecleaner.OutputHTML // Output HTML first

	cleaner2 := refynecleaner.New(cfg2)
	result2 := cleaner2.CleanWithStats(html)

	// Chain with markdown cleaner
	mdCleaner := cleaner.NewMarkdown()
	markdown2, _ := mdCleaner.Clean(result2.Content)

	fmt.Printf("After refyne: %d bytes\n", result2.Stats.OutputBytes)
	fmt.Printf("After html-to-markdown: %d bytes (%.1f%% total reduction)\n",
		len(markdown2), float64(len(html)-len(markdown2))/float64(len(html))*100)

	// Count markdown images
	mdImages2 := strings.Count(markdown2, "![")
	fmt.Printf("Markdown images in body: %d\n", mdImages2)

	// Save to file
	os.WriteFile("/tmp/method2_chained_md.md", []byte(markdown2), 0644)
	fmt.Println("Saved to: /tmp/method2_chained_md.md")

	// Show preview
	fmt.Println("\n--- Preview (first 2000 chars) ---")
	preview2 := markdown2
	if len(preview2) > 2000 {
		preview2 = preview2[:2000] + "\n..."
	}
	fmt.Println(preview2)

	// =====================================================
	// Summary
	// =====================================================
	fmt.Println("\n" + strings.Repeat("=", 61))
	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("=", 61))
	fmt.Printf("Method 1 (built-in): %d bytes, %d images in frontmatter\n",
		result1.Stats.OutputBytes, len(result1.Metadata.Images))
	fmt.Printf("Method 2 (chained):  %d bytes, %d images as markdown\n",
		len(markdown2), mdImages2)
	fmt.Println("\nFiles saved for diff:")
	fmt.Println("  /tmp/method1_refyne_md.md")
	fmt.Println("  /tmp/method2_chained_md.md")
}
