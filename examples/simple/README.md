# Simple Example

This example extracts basic information from any webpage.

## Usage

From the repository root:

```bash
# Using task
task example:simple -- "https://example.com"

# Or directly with the CLI
./bin/refyne scrape -u "https://example.com" -s examples/simple/schema.yaml

# Output as YAML
./bin/refyne scrape -u "https://example.com" -s examples/simple/schema.yaml --format yaml

# Use a different provider
./bin/refyne scrape -u "https://example.com" -s examples/simple/schema.yaml -p ollama -m llama3.2
```

## Schema

The schema extracts:
- **title**: Page title
- **summary**: Brief content summary
- **key_points**: Important facts/takeaways
- **links_mentioned**: External resources referenced
- **author**: Author if present
- **date**: Publication date if present
