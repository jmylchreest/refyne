// cleaner-compare compares all available cleaners on the same input.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jmylchreest/refyne/pkg/cleaner"
	refynecleaner "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: cleaner-compare <url-or-file>\n")
		os.Exit(1)
	}

	input := os.Args[1]
	var html string
	var err error

	if strings.HasPrefix(input, "http") {
		html, err = fetchURL(input)
	} else {
		data, err := os.ReadFile(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		html = string(data)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Input: %d bytes\n\n", len(html))
	fmt.Printf("%-25s %10s %8s %10s\n", "Cleaner", "Output", "Reduce%", "Time")
	fmt.Printf("%-25s %10s %8s %10s\n", "-------", "------", "-------", "----")

	// Test each cleaner
	cleaners := []struct {
		name    string
		cleaner cleaner.Cleaner
	}{
		{"noop", cleaner.NewNoop()},
		{"markdown", cleaner.NewMarkdown()},
		{"trafilatura (html)", cleaner.NewTrafilatura(&cleaner.TrafilaturaConfig{
			Output: cleaner.OutputHTML,
			Tables: cleaner.Include,
			Links:  cleaner.Include,
		})},
		{"trafilatura (text)", cleaner.NewTrafilatura(&cleaner.TrafilaturaConfig{
			Output: cleaner.OutputText,
		})},
		{"readability (html)", cleaner.NewReadability(&cleaner.ReadabilityConfig{
			Output: cleaner.OutputHTML,
		})},
		{"readability (text)", cleaner.NewReadability(&cleaner.ReadabilityConfig{
			Output: cleaner.OutputText,
		})},
		{"refyne (default)", refynecleaner.New(nil)},
		{"refyne (minimal)", refynecleaner.New(refynecleaner.PresetMinimal())},
		{"refyne (aggressive)", refynecleaner.New(refynecleaner.PresetAggressive())},
		// Chains
		{"refyne -> markdown", cleaner.NewChain(
			refynecleaner.New(nil),
			cleaner.NewMarkdown(),
		)},
		{"trafilatura -> markdown", cleaner.NewChain(
			cleaner.NewTrafilatura(&cleaner.TrafilaturaConfig{
				Output: cleaner.OutputHTML,
				Tables: cleaner.Include,
				Links:  cleaner.Include,
			}),
			cleaner.NewMarkdown(),
		)},
	}

	for _, c := range cleaners {
		start := time.Now()
		output, err := c.cleaner.Clean(html)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("%-25s %10s %8s %10v (error: %v)\n",
				c.name, "ERROR", "-", duration.Round(time.Millisecond), err)
			continue
		}

		reduction := float64(len(html)-len(output)) / float64(len(html)) * 100
		fmt.Printf("%-25s %10d %7.1f%% %10v\n",
			c.name, len(output), reduction, duration.Round(time.Millisecond))
	}
}

func fetchURL(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "cleaner-compare/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	return string(data), err
}
