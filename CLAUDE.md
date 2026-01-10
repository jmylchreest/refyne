# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Refyne is a Go 1.25 web scraper library and CLI that uses LLMs for structured data extraction. Users define schemas describing what data to extract, and the LLM interprets webpage content to produce validated structured output.

## Build Commands

```bash
# Build the CLI
task build

# Run tests
task test

# Run tests with coverage
task test:coverage

# Install to $GOPATH/bin
task install

# Format code
task fmt

# Update dependencies
task deps

# Build for all platforms
task release

# Run directly without building
task run -- scrape -u "https://example.com" -s schema.json

# Run a specific test
go test -v -run TestSchemaFromStruct ./pkg/schema
```

## Architecture

### Directory Structure

- `cmd/refyne/` - CLI entry point and commands (Cobra-based)
- `internal/llm/` - LLM provider abstraction (Anthropic, OpenAI, OpenRouter, Ollama)
- `internal/scraper/` - Web fetching (static via Colly, dynamic via Chromedp)
- `internal/crawler/` - Multi-page crawling with link following and pagination
- `internal/extractor/` - LLM extraction with retry logic
- `internal/output/` - Output writers (JSON, JSONL, YAML)
- `pkg/refyne/` - Public library API
- `pkg/schema/` - Schema definition and JSON Schema generation
- `examples/` - Usage examples

### Key Abstractions

**Provider Interface** (`internal/llm/provider.go`):
```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    Name() string
    SupportsJSONSchema() bool
}
```

**Schema** (`pkg/schema/schema.go`):
- Created from Go structs via `schema.NewSchema[T]()`
- Loaded from JSON/YAML files via `schema.FromFile(path)`
- Converts to JSON Schema for LLM structured output
- Validates extracted data with go-playground/validator

**Fetcher Interface** (`internal/scraper/fetcher.go`):
```go
type Fetcher interface {
    Fetch(ctx context.Context, url string, opts FetchOptions) (PageContent, error)
    Close() error
    Type() string
}
```

### Data Flow

1. **Fetch**: URL → Colly (static) or Chromedp (dynamic) → HTML + Text
2. **Extract**: Text + Schema → LLM Prompt → JSON Response
3. **Validate**: JSON → Schema validation → Retry if errors
4. **Output**: Validated data → JSON/JSONL/YAML writer

## Environment Variables

- `ANTHROPIC_API_KEY` - Anthropic API key (default provider)
- `OPENAI_API_KEY` - OpenAI API key
- `OPENROUTER_API_KEY` - OpenRouter API key
- `REFYNE_PROVIDER` - Default provider (anthropic, openai, openrouter, ollama)
- `REFYNE_MODEL` - Default model

## CLI Usage

```bash
# Single page extraction
refyne scrape -u "https://example.com/page" -s schema.json

# Crawl with link following
refyne scrape -u "https://example.com/search" -s schema.json \
    --follow "a.listing-link" --max-depth 1

# Pagination
refyne scrape -u "https://example.com/results" -s schema.json \
    --follow "a.item" --next "a.next-page" --max-pages 5

# Different providers
refyne scrape -u "..." -s schema.json -p openrouter -m anthropic/claude-sonnet
refyne scrape -u "..." -s schema.json -p ollama -m llama3.2
```

## Library Usage

```go
// Define schema as Go struct
type Recipe struct {
    Title       string   `json:"title" description:"Recipe name"`
    Ingredients []string `json:"ingredients" description:"Ingredients list"`
}

// Create and extract
s, _ := schema.NewSchema[Recipe](schema.WithDescription("A cooking recipe"))
r, _ := refyne.New(refyne.WithProvider("anthropic"))
defer r.Close()

result, _ := r.Extract(ctx, url, s)
recipe := result.Data.(*Recipe)
```

## Schema Files

Schemas can be JSON or YAML with a description field for NLP context:

```yaml
name: PropertyListing
description: |
  A real estate property listing page. Extract property details
  including address, price, and features.
fields:
  - name: price
    type: integer
    description: "Listing price in dollars"
    required: true
  - name: bedrooms
    type: integer
    description: "Number of bedrooms"
```

## Adding New LLM Providers

1. Create `internal/llm/newprovider.go` implementing the `Provider` interface
2. Register in `init()`:
   ```go
   func init() {
       RegisterProvider("newprovider", func(cfg ProviderConfig) (Provider, error) {
           return NewProvider(cfg)
       })
   }
   ```
3. Implement `Complete()` with JSON schema support via the provider's structured output API
