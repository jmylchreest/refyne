# FazWaz Thailand Example

Extract structured property listing data from FazWaz.com, a Thailand-focused English-language property portal popular with expats.

## Features

- Multi-currency price extraction with source tracking
- Automatic USD conversion if not shown on page
- English content (FazWaz is primarily English)
- Rich property metadata (ownership, completion status, etc.)

## CLI Usage

### Single listing extraction

```bash
refyne scrape \
  -u "https://www.fazwaz.com/property-for-sale/thailand/123456" \
  -s examples/fazwaz/schema.yaml \
  --fetch-mode dynamic \
  --debug
```

### Crawling Hua Hin listings

```bash
refyne scrape \
  -u "https://www.fazwaz.com/property-for-sale/thailand/prachuap-khiri-khan/hua-hin" \
  -s examples/fazwaz/schema.yaml \
  --follow "a[href*='/property-for-sale/thailand/']" \
  --follow-pattern "/[0-9]+$" \
  --max-depth 1 \
  --max-urls 10 \
  --fetch-mode dynamic \
  --delay 2s \
  --format jsonl \
  -o listings.jsonl \
  --debug
```

### Crawling condos in Phuket

```bash
refyne scrape \
  -u "https://www.fazwaz.com/condo-for-sale/thailand/phuket" \
  -s examples/fazwaz/schema.yaml \
  --follow "a[href*='/condo-for-sale/']" \
  --follow-pattern "/[0-9]+$" \
  --max-depth 1 \
  --max-urls 5 \
  --fetch-mode dynamic \
  --delay 2s \
  --format jsonl
```

## Go SDK Usage

```bash
cd examples/fazwaz
go run main.go "https://www.fazwaz.com/property-for-sale/thailand/123456"
```

Or with crawling:

```bash
go run main.go "https://www.fazwaz.com/property-for-sale/thailand/phuket" "a[href*='/property-for-sale/']"
```

## Schema Fields

| Field | Type | Description |
|-------|------|-------------|
| title | string | Property title/headline |
| prices | array | `[{currency: "THB", value: 5000000, source: "page"}, ...]` |
| price_per_sqm | array | Price per sqm in available currencies |
| address | string | Location (district, city, province) |
| province | string | Thai province name |
| district | string | District name |
| project_name | string | Development/project name |
| bedrooms | integer | Number of bedrooms |
| bathrooms | integer | Number of bathrooms |
| area_sqm | number | Property area in square meters |
| land_area_sqm | number | Land area (for houses/villas) |
| property_type | string | House, Condo, Villa, Townhouse, Land |
| ownership | string | Freehold, Leasehold, Foreign Freehold |
| completion_status | string | Off Plan, Under Construction, Completed |
| year_built | integer | Year built or expected completion |
| description | string | Property description (500 chars) |
| features | array | Key features (pool, gym, etc.) |
| nearby_places | array | Nearby amenities |
| agent_name | string | Agent or agency name |
| listing_url | string | Full listing URL |

## Example Output

```json
{
  "_metadata": {
    "url": "https://www.fazwaz.com/property-for-sale/thailand/123456",
    "fetched_at": "2026-01-11T19:30:00Z",
    "model": "openrouter/auto",
    "provider": "openrouter"
  },
  "data": {
    "title": "3 Bedroom Villa in Hua Hin with Private Pool",
    "prices": [
      {"currency": "THB", "value": 15000000, "source": "page"},
      {"currency": "USD", "value": 428571, "source": "page"},
      {"currency": "GBP", "value": 340000, "source": "converted"}
    ],
    "price_per_sqm": [
      {"currency": "THB", "value": 45000},
      {"currency": "USD", "value": 1286}
    ],
    "address": "Nong Kae, Hua Hin, Prachuap Khiri Khan",
    "province": "Prachuap Khiri Khan",
    "district": "Hua Hin",
    "project_name": "Palm Hills Golf Resort",
    "bedrooms": 3,
    "bathrooms": 3,
    "area_sqm": 333,
    "land_area_sqm": 600,
    "property_type": "Villa",
    "ownership": "Freehold",
    "completion_status": "Completed",
    "year_built": 2020,
    "description": "Stunning 3-bedroom villa with private pool...",
    "features": ["private pool", "garden", "parking", "security"],
    "nearby_places": ["beach 5 min", "golf course", "Bluport mall"],
    "agent_name": "Thailand Property Agency"
  }
}
```

## Notes

- FazWaz is primarily English so no translation needed
- Use `--fetch-mode dynamic` as FazWaz uses JavaScript
- The `--follow-pattern "/[0-9]+$"` ensures only property detail pages are followed
- FazWaz shows prices in multiple currencies - all are extracted
- USD is always included (converted if not on page)
