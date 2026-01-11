# DDProperty Thailand Example

Extract structured property listing data from DDProperty.com, a Thai real estate portal. Handles Thai language content and converts prices to both THB and approximate GBP.

## CLI Usage

### Single listing extraction

```bash
refyne scrape \
  -u "https://www.ddproperty.com/property/..." \
  -s examples/ddproperty/schema.yaml \
  --fetch-mode dynamic
```

### Crawling search results

```bash
# Crawl Krabi properties for sale
refyne scrape \
  -u "https://www.ddproperty.com/%E0%B8%A3%E0%B8%A7%E0%B8%A1%E0%B8%9B%E0%B8%A3%E0%B8%B0%E0%B8%81%E0%B8%B2%E0%B8%A8%E0%B8%82%E0%B8%B2%E0%B8%A2?listingType=sale&regionCode=TH81" \
  -s examples/ddproperty/schema.yaml \
  --follow "a[href*='/property/']" \
  --max-depth 1 \
  --max-urls 10 \
  --fetch-mode dynamic \
  --delay 2s \
  --format jsonl \
  -o listings.jsonl
```

### With metadata

```bash
refyne scrape \
  -u "https://www.ddproperty.com/..." \
  -s examples/ddproperty/schema.yaml \
  --follow "a[href*='/property/']" \
  --max-depth 1 \
  --fetch-mode dynamic \
  --include-metadata \
  --format jsonl
```

## Go Library Usage

```bash
cd examples/ddproperty
go run main.go "https://www.ddproperty.com/..." "a[href*='/property/']"
```

## Schema Fields

| Field | Type | Description |
|-------|------|-------------|
| title | string | Property title (may be in Thai) |
| price_thb | integer | Price in Thai Baht |
| price_gbp_approx | integer | Approximate GBP value (1 GBP = 45 THB) |
| address | string | Property location |
| province | string | Thai province (Krabi, Phuket, etc.) |
| district | string | District/amphoe name |
| bedrooms | integer | Number of bedrooms |
| bathrooms | integer | Number of bathrooms |
| land_area_sqm | number | Land area in square meters |
| land_area_rai | number | Land area in rai (1 rai = 1600 sqm) |
| building_area_sqm | number | Building area in square meters |
| property_type | string | house, condo, land, villa, townhouse |
| listing_type | string | sale or rent |
| description | string | Description (translated to English) |
| features | array | Features (pool, garden, sea view, etc.) |
| nearby | array | Nearby amenities |
| agent_name | string | Agent/developer name |
| agent_phone | string | Contact phone |
| images | array | Photo URLs |
| url | string | Listing URL |

## Example Output

```json
{
  "_metadata": {
    "url": "https://www.ddproperty.com/property/12345",
    "fetched_at": "2025-01-11T10:30:00Z",
    "model": "claude-sonnet-4-20250514",
    "provider": "anthropic"
  },
  "data": {
    "title": "Beautiful Villa with Sea View in Ao Nang",
    "price_thb": 15000000,
    "price_gbp_approx": 333333,
    "address": "Ao Nang, Mueang Krabi",
    "province": "Krabi",
    "bedrooms": 3,
    "bathrooms": 3,
    "land_area_rai": 0.5,
    "building_area_sqm": 250,
    "property_type": "villa",
    "listing_type": "sale",
    "features": ["pool", "sea view", "garden", "parking"],
    "nearby": ["beach", "restaurants", "airport 30 min"]
  }
}
```

## Notes

- Use `--fetch-mode dynamic` as DDProperty is JavaScript-heavy
- Add `--delay 2s` to be respectful to the server
- The LLM will translate Thai descriptions to English
- Currency conversion uses approximate rate of 1 GBP = 45 THB
