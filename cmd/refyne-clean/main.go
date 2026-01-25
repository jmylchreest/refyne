// refyne-clean is a standalone CLI tool for testing and developing the refyne content cleaner.
//
// Usage:
//
//	refyne-clean [options] <url-or-file>
//
// Examples:
//
//	# Clean a URL and show stats
//	refyne-clean https://example.com/article
//
//	# Clean with aggressive preset
//	refyne-clean -preset aggressive https://example.com/blog
//
//	# Clean from file
//	refyne-clean -f page.html
//
//	# Custom selectors to remove
//	refyne-clean -remove "nav,footer,.ads" https://example.com
//
//	# Output to file
//	refyne-clean -o cleaned.html https://example.com
//
//	# Show only stats, don't output content
//	refyne-clean -stats-only https://example.com
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jmylchreest/refyne/pkg/cleaner/refyne"
)

var (
	// Input options
	fileInput = flag.String("f", "", "Read HTML from file instead of URL")

	// Config options
	preset       = flag.String("preset", "", "Use preset: minimal, aggressive")
	remove       = flag.String("remove", "", "Comma-separated selectors to remove")
	keep         = flag.String("keep", "", "Comma-separated selectors to keep")
	outputFormat = flag.String("format", "html", "Output format: html, text")

	// Heuristics
	linkDensity = flag.Bool("link-density", false, "Enable link density heuristic")
	shortText   = flag.Bool("short-text", false, "Enable short text removal")

	// Output options
	outputFile = flag.String("o", "", "Write cleaned output to file")
	statsOnly  = flag.Bool("stats-only", false, "Only show stats, don't output content")
	jsonStats  = flag.Bool("json", false, "Output stats as JSON")
	verbose    = flag.Bool("v", false, "Verbose output (show warnings)")
	quiet      = flag.Bool("q", false, "Quiet mode (no stats, only content)")

	// Compare mode
	compare = flag.Bool("compare", false, "Compare different presets")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "refyne-clean - Test tool for the refyne content cleaner\n\n")
		fmt.Fprintf(os.Stderr, "Usage: refyne-clean [options] <url-or-file>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  refyne-clean https://example.com/article\n")
		fmt.Fprintf(os.Stderr, "  refyne-clean -preset aggressive https://example.com/blog\n")
		fmt.Fprintf(os.Stderr, "  refyne-clean -remove 'nav,footer,.ads' https://example.com\n")
		fmt.Fprintf(os.Stderr, "  refyne-clean -compare https://example.com\n")
	}

	flag.Parse()

	// Get input source
	var html string
	var source string
	var err error

	if *fileInput != "" {
		html, err = readFile(*fileInput)
		source = *fileInput
	} else if flag.NArg() > 0 {
		url := flag.Arg(0)
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			html, err = fetchURL(url)
			source = url
		} else {
			html, err = readFile(url)
			source = url
		}
	} else {
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		html = string(data)
		source = "stdin"
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(html) == 0 {
		fmt.Fprintf(os.Stderr, "Error: empty input\n")
		os.Exit(1)
	}

	// Compare mode
	if *compare {
		runComparison(html, source)
		return
	}

	// Build config
	cfg := buildConfig()

	// Run cleaner
	cleaner := refyne.New(cfg)
	result := cleaner.CleanWithStats(html)

	// Output stats
	if !*quiet {
		if *jsonStats {
			outputJSONStats(result, source)
		} else {
			outputTextStats(result, source)
		}
	}

	// Output warnings
	if *verbose && result.HasWarnings() {
		fmt.Fprintf(os.Stderr, "\nWarnings:\n")
		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "  %s\n", w.String())
		}
	}

	// Output content
	if !*statsOnly {
		if *outputFile != "" {
			if err := os.WriteFile(*outputFile, []byte(result.Content), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
				os.Exit(1)
			}
			if !*quiet {
				fmt.Fprintf(os.Stderr, "\nWritten to %s\n", *outputFile)
			}
		} else if !*quiet {
			fmt.Println("\n--- Cleaned Content ---")
			fmt.Println(result.Content)
		} else {
			fmt.Println(result.Content)
		}
	}
}

func buildConfig() *refyne.Config {
	var cfg *refyne.Config

	// Start with preset or default
	switch *preset {
	case "minimal":
		cfg = refyne.PresetMinimal()
	case "aggressive":
		cfg = refyne.PresetAggressive()
	default:
		cfg = refyne.DefaultConfig()
	}

	// Override with flags
	if *remove != "" {
		selectors := strings.Split(*remove, ",")
		for i := range selectors {
			selectors[i] = strings.TrimSpace(selectors[i])
		}
		cfg.RemoveSelectors = append(cfg.RemoveSelectors, selectors...)
	}

	if *keep != "" {
		selectors := strings.Split(*keep, ",")
		for i := range selectors {
			selectors[i] = strings.TrimSpace(selectors[i])
		}
		cfg.KeepSelectors = append(cfg.KeepSelectors, selectors...)
	}

	if *linkDensity {
		cfg.RemoveByLinkDensity = true
	}

	if *shortText {
		cfg.RemoveShortText = true
	}

	switch *outputFormat {
	case "text":
		cfg.Output = refyne.OutputText
	case "markdown":
		cfg.Output = refyne.OutputMarkdown
	default:
		cfg.Output = refyne.OutputHTML
	}

	return cfg
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}
	return string(data), nil
}

func fetchURL(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "refyne-clean/1.0 (testing tool)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return string(data), nil
}

func outputTextStats(result *refyne.Result, source string) {
	fmt.Fprintf(os.Stderr, "\n=== Refyne Cleaner Stats ===\n")
	fmt.Fprintf(os.Stderr, "Source: %s\n", source)
	fmt.Fprintf(os.Stderr, "%s", result.Stats.String())
}

func outputJSONStats(result *refyne.Result, source string) {
	stats := struct {
		Source  string        `json:"source"`
		Stats   *refyne.Stats `json:"stats"`
		Reduced float64       `json:"reduction_percent"`
	}{
		Source:  source,
		Stats:   result.Stats,
		Reduced: result.Stats.ReductionPercent(),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(stats)
}

func runComparison(html string, source string) {
	presets := []struct {
		name string
		cfg  *refyne.Config
	}{
		{"default", refyne.DefaultConfig()},
		{"minimal", refyne.PresetMinimal()},
		{"aggressive", refyne.PresetAggressive()},
	}

	fmt.Printf("\n=== Preset Comparison for %s ===\n", source)
	fmt.Printf("Input size: %d bytes\n\n", len(html))
	fmt.Printf("%-12s %10s %10s %8s %10s\n", "Preset", "Output", "Removed", "Reduce%", "Time")
	fmt.Printf("%-12s %10s %10s %8s %10s\n", "------", "------", "-------", "-------", "----")

	for _, p := range presets {
		cleaner := refyne.New(p.cfg)
		result := cleaner.CleanWithStats(html)

		fmt.Printf("%-12s %10d %10d %7.1f%% %10v\n",
			p.name,
			result.Stats.OutputBytes,
			result.Stats.TotalElementsRemoved(),
			result.Stats.ReductionPercent(),
			result.Stats.TotalDuration.Round(time.Millisecond))
	}

	fmt.Println()
}
