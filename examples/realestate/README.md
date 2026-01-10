# Real Estate Listing Example

Extract structured property listing data from real estate websites, with support for crawling multiple listings.

## Usage

### Single listing extraction:

```bash
# From repository root
task example:realestate:cli -- "https://www.realtor.com/realestateandhomes-detail/123-Main-St"

# Or directly
./bin/refyne scrape \
  -u "https://example-realestate.com/listing/12345" \
  -s examples/realestate/schema.yaml
```

### Crawling multiple listings:

```bash
# Crawl a search results page, following links to individual listings
task example:realestate:crawl -- "https://example-realestate.com/search?city=austin" "a.listing-card"

# Or directly with more options
./bin/refyne scrape \
  -u "https://example-realestate.com/search" \
  -s examples/realestate/schema.yaml \
  --follow "a.property-link" \
  --max-depth 1 \
  --delay 1s \
  --format jsonl \
  -o listings.jsonl
```

### Using the Go example code:

```bash
task example:realestate -- "https://example-realestate.com/search" "a.listing-card"
```

## Schema Fields

- **title**: Listing headline
- **price**: Numeric price (no currency symbols)
- **address**: Full property address
- **bedrooms**: Number of bedrooms
- **bathrooms**: Number of bathrooms (can be 1.5, 2.5, etc.)
- **square_feet**: Living area in sq ft
- **property_type**: house, apartment, condo, townhouse, land
- **description**: Full listing description
- **features**: List of features (pool, garage, etc.)
- **images**: URLs of property photos

## Example Output

```json
{
  "title": "Charming 3BR Home in Downtown Austin",
  "price": 450000,
  "address": "123 Main Street, Austin, TX 78701",
  "bedrooms": 3,
  "bathrooms": 2.5,
  "square_feet": 1850,
  "property_type": "house",
  "features": ["garage", "backyard", "updated kitchen"],
  "images": [
    "https://example.com/photos/listing-123-1.jpg",
    "https://example.com/photos/listing-123-2.jpg"
  ]
}
```
