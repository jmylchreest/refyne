# Simple Example

This example extracts basic information from any webpage.

## CLI Usage

```bash
# Basic extraction
refyne scrape -u "https://example.com" -s examples/simple/schema.yaml

# Output as YAML
refyne scrape -u "https://example.com" -s examples/simple/schema.yaml --format yaml

# Use a different provider
refyne scrape -u "https://example.com" -s examples/simple/schema.yaml -p ollama -m llama3.2

# With debug output
refyne scrape -u "https://example.com" -s examples/simple/schema.yaml --debug
```

## Go SDK Usage

```bash
cd examples/simple
go run main.go "https://example.com"
```

## Schema

The schema extracts:
- **title**: Page title
- **summary**: Brief content summary (2-3 sentences)
- **key_points**: Important facts/takeaways
- **links_mentioned**: External resources referenced
- **author**: Author if present
- **date**: Publication date if present

## Example Output

```json
{
  "title": "Example Domain",
  "summary": "This domain is for use in illustrative examples in documents.",
  "key_points": [
    "Reserved for documentation purposes",
    "Can be used without prior coordination"
  ],
  "links_mentioned": [
    "https://www.iana.org/domains/example"
  ]
}
```
