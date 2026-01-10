# Zoopla Property Listing Example

Extract structured property listing data from Zoopla UK real estate website.

## Usage

### Single listing extraction:

```bash
# From repository root
task example:zoopla:cli -- "https://www.zoopla.co.uk/for-sale/details/12345678/"

# Or directly
./bin/refyne scrape \
  -u "https://www.zoopla.co.uk/for-sale/details/12345678/" \
  -s examples/zoopla/schema.yaml
```

### Crawling search results:

```bash
# Crawl a search results page, following links to individual listings
task example:zoopla:crawl -- "https://www.zoopla.co.uk/for-sale/property/north-yorkshire/?q=york"

# Or directly with more options
./bin/refyne scrape \
  -u "https://www.zoopla.co.uk/for-sale/property/north-yorkshire/?q=york" \
  -s examples/zoopla/schema.yaml \
  --follow "a[href*='/for-sale/details/']" \
  --max-depth 1 \
  --delay 2s \
  --format jsonl \
  -o zoopla-listings.jsonl
```

### Pagination support:

```bash
# Crawl multiple pages of search results
./bin/refyne scrape \
  -u "https://www.zoopla.co.uk/for-sale/property/north-yorkshire/" \
  -s examples/zoopla/schema.yaml \
  --follow "a[href*='/for-sale/details/']" \
  --next "a[data-testid='pagination-next']" \
  --max-pages 5 \
  --max-depth 1 \
  --delay 2s \
  --format jsonl \
  -o zoopla-listings.jsonl
```

## Schema Fields

### Core Property Details
- **title**: Property headline
- **price**: Numeric price in GBP (no currency symbol)
- **price_qualifier**: Guide price, Offers over, etc.
- **address**: Full address including postcode

### Property Specifications
- **bedrooms**: Number of bedrooms
- **bathrooms**: Number of bathrooms
- **receptions**: Number of reception rooms
- **square_feet**: Size in sq ft
- **property_type**: detached, semi-detached, terraced, flat, bungalow

### UK-Specific Fields
- **tenure**: Freehold, Leasehold, Share of Freehold
- **chain_status**: Chain free, No onward chain
- **epc_rating**: Energy Performance Certificate (A-G)
- **council_tax_band**: Council tax band (A-H)

### Listing Information
- **status**: New, Reduced, Under offer, Sold STC
- **description**: Full listing description
- **features**: List of property features
- **images**: URLs of property photos
- **agent_name**: Estate agent name
- **agent_phone**: Agent contact number
- **url**: Listing URL

## Example Output

```json
{
  "title": "3 bedroom semi-detached house for sale",
  "price": 210000,
  "price_qualifier": "",
  "address": "Main Street, Stamford Bridge, York YO41",
  "bedrooms": 3,
  "bathrooms": 1,
  "receptions": 1,
  "square_feet": 1100,
  "property_type": "semi-detached",
  "tenure": "Freehold",
  "chain_status": "Chain free",
  "status": "Reduced",
  "description": "A well-presented three bedroom semi-detached property...",
  "features": ["garden", "driveway", "garage", "double glazing"],
  "images": [
    "https://lid.zoocdn.com/u/2400/1800/abc123.jpg"
  ],
  "agent_name": "Hunters - York",
  "epc_rating": "D",
  "council_tax_band": "C",
  "url": "https://www.zoopla.co.uk/for-sale/details/12345678/"
}
```

## Notes

- Zoopla uses JavaScript rendering, so the `--fetch-mode dynamic` flag may be needed for full content
- Respect Zoopla's robots.txt and rate limits - use `--delay 2s` or higher
- Some fields like EPC rating and council tax band are only shown on detail pages
- Price qualifiers vary: "Guide price", "Offers over", "Offers in region of", "POA"
