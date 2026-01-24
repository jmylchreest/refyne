# Refyne

LLM-powered web scraper for structured data extraction.

Define a schema for the data you want, point it at URLs, and get validated, structured output.

## Quick Start

### Prerequisites

1. **Go 1.25+**

2. **Taskfile** (optional, for running examples):
   ```bash
   # macOS
   brew install go-task

   # Linux
   sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin

   # Or see https://taskfile.dev/installation/
   ```

3. **Chrome/Chromium** (for dynamic/JavaScript-rendered pages)

### Environment Variables

Refyne auto-detects your LLM provider based on available API keys:

| Priority | Environment Variable | Provider | Default Model |
|----------|---------------------|----------|---------------|
| 1 | `OPENROUTER_API_KEY` | openrouter | openrouter/auto |
| 2 | `ANTHROPIC_API_KEY` | anthropic | claude-sonnet-4-20250514 |
| 3 | `OPENAI_API_KEY` | openai | gpt-4o |
| 4 | (none needed) | ollama | llama3.2 |

Set your preferred provider's API key:
```bash
# OpenRouter (often has free models)
export OPENROUTER_API_KEY="your-key-here"

# Or Anthropic
export ANTHROPIC_API_KEY="your-key-here"

# Or OpenAI
export OPENAI_API_KEY="your-key-here"
```

You can override auto-detection with flags:
```bash
refyne scrape -u URL -s schema.yaml -p anthropic -m claude-3-haiku
```

Or environment variables:
```bash
export REFYNE_PROVIDER=openrouter
export REFYNE_MODEL=anthropic/claude-sonnet
```

Or a config file at `~/.refyne.yaml`:
```yaml
providers:
  openrouter:
    model: "xiaomi/mimo-v2-flash:free"
    # temperature: 0.1
    # max_tokens: 8192
  anthropic:
    model: "claude-sonnet-4-20250514"
  ollama:
    model: "llama3.2"

# Fallback order when multiple providers are available
fallback_order:
  - openrouter
  - anthropic
  - ollama
```

### Building

```bash
# Build the CLI
task build
# or: go build -o bin/refyne ./cmd/refyne

# Install to $GOPATH/bin
task install
# or: go install ./cmd/refyne
```

## Usage

### Single Page Extraction

```bash
refyne scrape -u "https://example.com/recipe/123" -s schema.yaml
```

### Crawling (List + Detail Pages)

```bash
refyne scrape -u "https://example.com/search" -s schema.yaml \
    --follow "a.item-link" \
    --max-depth 1 \
    --delay 1s
```

### Pagination

```bash
refyne scrape -u "https://example.com/category" -s schema.yaml \
    --follow "a.item" \
    --next "a.pagination-next" \
    --max-pages 10
```

### Output Formats

```bash
# JSON (default)
refyne scrape -u URL -s schema.yaml

# JSONL (one JSON object per line, good for streaming)
refyne scrape -u URL -s schema.yaml --format jsonl -o results.jsonl

# YAML
refyne scrape -u URL -s schema.yaml --format yaml
```

## Running Examples

Examples are in the `examples/` directory. Each has a schema and README.

### Simply Recipes (static HTML)

```bash
# Single recipe
task example:simplyrecipes:cli -- "https://www.simplyrecipes.com/steak-tips-recipe-7972730"

# Crawl a category page
task example:simplyrecipes:crawl -- "https://www.simplyrecipes.com/dinner-recipes-5091433"
```

### Other Recipe Sites

The recipe schema works with many recipe sites. Use the CLI directly:

```bash
# BBC Food
go run ./cmd/refyne scrape \
  -u "https://www.bbc.co.uk/food/recipes/chickentostadas_87570" \
  -s examples/simplyrecipes/schema.yaml \
  --fetch-mode static

# Any recipe site (schema is portable)
go run ./cmd/refyne scrape \
  -u "https://example.com/recipe/123" \
  -s examples/simplyrecipes/schema.yaml \
  --fetch-mode static
```

### Zoopla UK Property Listings (dynamic/JS)

```bash
# Single listing
task example:zoopla:cli -- "https://www.zoopla.co.uk/for-sale/details/12345678"

# Crawl search results
task example:zoopla:crawl -- "https://www.zoopla.co.uk/for-sale/property/london/"
```

### Other Examples

```bash
# List all available tasks
task --list

# Run any example
task example:recipes:cli -- "URL"
task example:realestate:cli -- "URL"
task example:simple -- "URL"
```

## Schema Definition

Schemas define what data to extract. Can be YAML or JSON.

```yaml
name: Recipe
description: |
  A recipe page. Extract the title, ingredients list,
  and step-by-step instructions.

fields:
  - name: title
    type: string
    description: "Recipe name"
    required: true

  - name: ingredients
    type: array
    description: "List of ingredients with amounts"
    items:
      type: object
      properties:
        amount:
          type: string
          description: "Quantity and unit"
        name:
          type: string
          description: "Ingredient name"

  - name: instructions
    type: array
    description: "Step-by-step cooking instructions"
    items:
      type: string
```

## CLI Reference

```
refyne scrape [flags]

Flags:
  -u, --url strings       URL(s) to scrape
  -s, --schema string     Path to schema file (required)
  -p, --provider string   LLM provider (auto-detects from env vars)
  -m, --model string      Model name (uses provider default if not set)
  -k, --api-key string    API key (or use env var)
  -o, --output string     Output file (default: stdout)
      --format string     Output format: json, jsonl, yaml (default "json")
      --fetch-mode string Fetch mode: auto, static, dynamic (default "auto")
      --timeout duration  Request timeout (default 30s)
      --max-retries int   Max extraction retries (default 3)
      --max-content-size string  Max input content size (default "100KB", 0=unlimited)
      --debug             Enable debug logging

Crawling:
      --follow string        CSS selector for links to follow
      --follow-pattern string  Regex pattern for URLs to follow
      --next string          CSS selector for pagination
      --max-depth int        Max link depth (default 1)
      --max-pages int        Max pagination pages (0=unlimited)
      --max-urls int         Max total URLs to process (0=unlimited)
      --delay duration       Delay between requests (default 200ms)
  -c, --concurrency int      Concurrent requests (default 3)

Output:
      --include-metadata     Wrap output with _metadata and data keys (default true)
      --save-training-data   Save input/output pairs for fine-tuning (JSONL file path)
```

## Using Refyne Packages in Your Own Projects

Refyne is more than a CLI - it provides reusable Go packages for HTML cleaning, content extraction, and web scraping that you can use directly in your own applications.

### The Refyne Cleaner Package

The `pkg/cleaner/refyne` package is a highly configurable HTML cleaner optimized for preparing web content for LLM consumption. It reduces token usage while preserving meaningful content.

```bash
go get github.com/jmylchreest/refyne/pkg/cleaner/refyne
```

#### Why Use It?

- **Token reduction**: Strips scripts, styles, ads, tracking, hidden elements - things LLMs don't need
- **Configurable presets**: From minimal cleaning to aggressive content extraction
- **Multiple output formats**: HTML, plain text, or LLM-optimized markdown with structured metadata
- **Image handling**: Extracts images to frontmatter with `{{IMG_001}}` placeholders in the body
- **Lazy-loading support**: Handles `data-src`, `srcset`, and noscript fallbacks for JS-loaded images

#### Basic Usage

```go
package main

import (
    "fmt"
    refynecleaner "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
)

func main() {
    // Create cleaner with default config (HTML output)
    cleaner := refynecleaner.New(nil)

    html := `<html>
        <head><script>tracking();</script></head>
        <body>
            <nav>Menu items...</nav>
            <article>
                <h1>Article Title</h1>
                <p>Important content here.</p>
            </article>
            <div class="ad">Advertisement</div>
        </body>
    </html>`

    cleaned, _ := cleaner.Clean(html)
    fmt.Println(cleaned)
    // Output: Clean HTML with scripts, ads, and navigation removed
}
```

#### Markdown Output with Frontmatter (for LLMs)

The most powerful mode for LLM consumption - generates markdown with YAML frontmatter containing structured metadata:

```go
package main

import (
    "fmt"
    refynecleaner "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
)

func main() {
    cfg := refynecleaner.DefaultConfig()
    cfg.Output = refynecleaner.OutputMarkdown
    cfg.IncludeFrontmatter = true  // Add YAML metadata header
    cfg.ExtractImages = true       // Images become {{IMG_001}} placeholders
    cfg.ExtractHeadings = true     // Include heading structure
    // Optional: resolve relative URLs to absolute (disabled by default to save tokens)
    // cfg.BaseURL = "https://example.com"
    // cfg.ResolveURLs = true

    cleaner := refynecleaner.New(cfg)

    html := `<html><body>
        <h1>Recipe: Chocolate Cake</h1>
        <img src="/images/cake.jpg" alt="Delicious chocolate cake">
        <p>A rich, moist chocolate cake.</p>
        <h2>Ingredients</h2>
        <ul><li>2 cups flour</li><li>1 cup sugar</li></ul>
    </body></html>`

    markdown, _ := cleaner.Clean(html)
    fmt.Println(markdown)
}
```

Output:
```yaml
---
# Content Metadata
# Images are referenced in the body as {{IMG_001}}, {{IMG_002}}, etc.
# Use this map to look up the actual URL and description for each placeholder.
images:
  IMG_001:
    url: "https://example.com/images/cake.jpg"
    alt: "Delicious chocolate cake"

headings:
  - level: 1
    text: "Recipe: Chocolate Cake"
  - level: 2
    text: "Ingredients"

links_count: 0

hints:
  - "Image placeholders like {{IMG_001}} appear in the body where images belong"
  - "Look up each placeholder in the 'images' map above to get the actual URL"
---

# Recipe: Chocolate Cake

{{IMG_001}}

A rich, moist chocolate cake.

## Ingredients

- 2 cups flour
- 1 cup sugar
```

#### Configuration Presets

```go
// Minimal - only removes scripts/styles/comments
cfg := refynecleaner.PresetMinimal()

// Default - removes ads, tracking, hidden elements, preserves content structure
cfg := refynecleaner.DefaultConfig()

// Aggressive - also removes navigation, sidebars, short text blocks
cfg := refynecleaner.PresetAggressive()
```

#### Custom Selectors

```go
cfg := refynecleaner.DefaultConfig()

// Remove specific elements
cfg.RemoveSelectors = append(cfg.RemoveSelectors,
    ".sidebar",
    "#comments",
    "[data-testid='promo-banner']",
)

// Force keep elements (overrides removals)
cfg.KeepSelectors = []string{
    ".product-price",
    ".main-content",
}
```

#### Getting Stats

```go
cleaner := refynecleaner.New(cfg)
result := cleaner.CleanWithStats(html)

fmt.Printf("Input: %d bytes\n", result.Stats.InputBytes)
fmt.Printf("Output: %d bytes\n", result.Stats.OutputBytes)
fmt.Printf("Reduction: %.1f%%\n", result.Stats.ReductionPercent)
fmt.Printf("Parse time: %v\n", result.Stats.ParseDuration)
fmt.Printf("Clean time: %v\n", result.Stats.CleanDuration)

// Access extracted metadata (when using markdown output)
if result.Metadata != nil {
    fmt.Printf("Images found: %d\n", len(result.Metadata.Images))
    fmt.Printf("Headings found: %d\n", len(result.Metadata.Headings))
}
```

### Other Reusable Packages

| Package | Description |
|---------|-------------|
| `pkg/cleaner` | Cleaner interface and implementations (noop, markdown, trafilatura, readability, chain) |
| `pkg/cleaner/refyne` | Configurable HTML cleaner with LLM-optimized output |
| `pkg/fetcher` | HTTP fetching with static and dynamic (headless browser) modes |
| `pkg/extractor` | LLM extraction with provider support (Anthropic, OpenAI, OpenRouter, Ollama) |
| `pkg/schema` | Schema definition and JSON Schema generation |
| `pkg/refyne` | High-level orchestrator combining fetch, clean, and extract |

## Development

```bash
# Run tests
task test

# Run with coverage
task test:coverage

# Format code
task fmt

# Run linter
task lint
```

## Fine-Tuning Local Models

You can fine-tune a local model on your extraction tasks for faster, cheaper, offline operation.

### 1. Generate Training Data

Run extractions with a capable model and save training data:

```bash
refyne scrape \
  -u "https://example.com/search" \
  -s schema.yaml \
  --follow "a.item-link" \
  --max-urls 100 \
  --save-training-data training.jsonl \
  -o extractions.jsonl
```

This creates `training.jsonl` with input/output pairs:
```json
{"url":"https://...","input":"<page content>","output":{"title":"...","price":100}}
```

### 2. Fine-Tune with Unsloth (LoRA)

[Unsloth](https://github.com/unslothai/unsloth) provides fast, memory-efficient LoRA fine-tuning.
LoRA only trains small adapter layers instead of full model weights - faster training, less VRAM, smaller files.

```python
# train_lora.py
from unsloth import FastLanguageModel
import json

# Load base model with LoRA
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name="unsloth/Qwen2.5-3B",
    max_seq_length=4096,
    load_in_4bit=True,  # Use 4-bit quantization to save VRAM
)

# Add LoRA adapters
model = FastLanguageModel.get_peft_model(
    model,
    r=16,              # LoRA rank
    lora_alpha=16,
    target_modules=["q_proj", "k_proj", "v_proj", "o_proj"],
    lora_dropout=0,
    bias="none",
)

# Load training data
with open("training.jsonl") as f:
    data = [json.loads(line) for line in f]

# Format for training (input -> output pairs)
def format_prompt(example):
    return f"Extract structured data:\n{example['input']}\n\nJSON:\n{json.dumps(example['output'])}"

# Train
from trl import SFTTrainer
trainer = SFTTrainer(
    model=model,
    train_dataset=data,
    formatting_func=format_prompt,
    max_seq_length=4096,
)
trainer.train()

# Save LoRA adapter
model.save_pretrained("refyne-lora")
```

**Good base models for structured extraction:**
- `Qwen2.5-3B` or `Qwen2.5-7B` - Best for JSON tasks
- `Llama-3.2-3B` - Fast, good general purpose
- `Mistral-7B` - Solid all-rounder

### 3. Merge LoRA and Convert to GGUF

```bash
# Merge LoRA adapter with base model
python -c "
from unsloth import FastLanguageModel
model, tokenizer = FastLanguageModel.from_pretrained('refyne-lora')
model.save_pretrained_merged('refyne-merged', tokenizer)
"

# Convert to GGUF for Ollama
pip install llama-cpp-python
python -m llama_cpp.convert refyne-merged --outfile refyne-extractor.gguf --outtype q4_k_m

# Create Ollama model
cat > Modelfile << 'EOF'
FROM ./refyne-extractor.gguf
PARAMETER temperature 0.1
PARAMETER num_predict 4096
EOF

ollama create refyne-extractor -f Modelfile
```

### 4. Use Your Fine-Tuned Model

```bash
refyne scrape \
  -u "https://example.com/page" \
  -s schema.yaml \
  -p ollama \
  -m refyne-extractor
```

### Alternative: Axolotl

For more control over LoRA training, use [Axolotl](https://github.com/OpenAccess-AI-Collective/axolotl):

```yaml
# axolotl-config.yaml
base_model: Qwen/Qwen2.5-3B
load_in_4bit: true

adapter: lora
lora_r: 16
lora_alpha: 16
lora_target_modules:
  - q_proj
  - k_proj
  - v_proj
  - o_proj

datasets:
  - path: training.jsonl
    type: completion

output_dir: ./refyne-lora
```

```bash
accelerate launch -m axolotl.cli.train axolotl-config.yaml
```
