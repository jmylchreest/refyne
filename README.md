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
| 1 | `OPENROUTER_API_KEY` | openrouter | xiaomi/mimo-v2-flash:free |
| 2 | `ANTHROPIC_API_KEY` | anthropic | claude-opus-4-5-20251101 |
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

## License

MIT
