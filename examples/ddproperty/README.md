# DDProperty Thailand Example

Extract structured property listing data from DDProperty.com, a Thai real estate portal. Handles Thai language content with bilingual output and multi-currency price conversion.

## Features

- Multi-currency prices (THB from page, USD/GBP converted)
- Bilingual output: English first, Thai in parentheses for proper nouns
- Thai Buddhist year to Gregorian conversion
- Cloudflare bypass via FlareSolverr

## CLI Usage

### Single listing extraction

```bash
# With FlareSolverr for Cloudflare bypass
refyne scrape \
  -u "https://www.ddproperty.com/property/the-seacraze-..." \
  -s examples/ddproperty/schema.yaml \
  --fetch-mode dynamic \
  --flaresolverr-url "http://localhost:8191/v1" \
  --debug
```

### Crawling search results (Hua Hin)

```bash
refyne scrape \
  -u "https://www.ddproperty.com/%E0%B8%A3%E0%B8%A7%E0%B8%A1%E0%B8%9B%E0%B8%A3%E0%B8%B0%E0%B8%81%E0%B8%B2%E0%B8%A8%E0%B8%82%E0%B8%B2%E0%B8%A2?locale=en&listingType=sale&districtCode=TH7707" \
  -s examples/ddproperty/schema.yaml \
  --follow "a[href*='/property/']" \
  --max-depth 1 \
  --max-urls 10 \
  --fetch-mode dynamic \
  --flaresolverr-url "http://localhost:8191/v1" \
  --delay 2s \
  --format jsonl \
  -o listings.jsonl \
  --debug
```

### Crawling Krabi properties

```bash
refyne scrape \
  -u "https://www.ddproperty.com/%E0%B8%A3%E0%B8%A7%E0%B8%A1%E0%B8%9B%E0%B8%A3%E0%B8%B0%E0%B8%81%E0%B8%B2%E0%B8%A8%E0%B8%82%E0%B8%B2%E0%B8%A2?listingType=sale&regionCode=TH81" \
  -s examples/ddproperty/schema.yaml \
  --follow "a[href*='/property/']" \
  --max-depth 1 \
  --max-urls 5 \
  --fetch-mode dynamic \
  --flaresolverr-url "http://localhost:8191/v1" \
  --stealth \
  --format jsonl \
  --debug
```

## Go SDK Usage

```bash
cd examples/ddproperty
go run main.go "https://www.ddproperty.com/property/..."
```

Or with crawling:

```bash
go run main.go "https://www.ddproperty.com/..." "a[href*='/property/']"
```

## Schema Fields

| Field | Type | Description |
|-------|------|-------------|
| title | object | `{en: "English", th: "ไทย"}` - bilingual title |
| project_name | object | `{en: "Project", th: "โครงการ"}` - development name |
| prices | array | `[{currency: "THB", value: 2890000, source: "page"}, ...]` |
| price_per_sqm | array | Price per sqm in multiple currencies |
| price_qualifier | string | "negotiable", "starting from", etc. |
| location | object | `{address, district, province, subdistrict}` |
| bedrooms | integer | Number of bedrooms |
| bathrooms | integer | Number of bathrooms |
| area_sqm | number | Usable area in square meters |
| land_area_sqm | number | Land area in square meters |
| land_area_rai | number | Land area in rai (1 rai = 1600 sqm) |
| property_type | string | Condo, House, Villa, Townhouse, Land |
| listing_type | string | Sale or Rent |
| ownership | string | Freehold, Leasehold |
| completion_status | string | Completed, Under Construction, Off Plan |
| year_built | integer | Gregorian year (Thai year - 543) |
| furnishing | string | Fully/Partially Furnished, Unfurnished |
| description | string | English description |
| features | array | Property features in English |
| facilities | array | Building facilities in English |
| nearby | array | Nearby places `["Beach (ชายหาด)", ...]` |
| agent | object | `{name, phone, type}` |
| listing_id | string | DDProperty listing ID |
| listing_date | string | ISO date YYYY-MM-DD |
| images | array | Photo URLs (first 10) |
| url | string | Full listing URL |

## Example Output

```json
{
  "_metadata": {
    "url": "https://www.ddproperty.com/property/the-seacraze-...",
    "fetched_at": "2026-01-11T19:30:00Z",
    "model": "openrouter/auto",
    "provider": "openrouter"
  },
  "data": {
    "title": {
      "en": "The Seacraze Condo for Sale - Sea View",
      "th": "เดอะ ซีเครซ คอนโดขาย - วิวทะเล"
    },
    "project_name": {
      "en": "The Seacraze",
      "th": "เดอะ ซีเครซ"
    },
    "prices": [
      {"currency": "THB", "value": 2890000, "source": "page"},
      {"currency": "USD", "value": 82571, "source": "converted"},
      {"currency": "GBP", "value": 64222, "source": "converted"}
    ],
    "price_per_sqm": [
      {"currency": "THB", "value": 66437, "source": "page"},
      {"currency": "USD", "value": 1898, "source": "converted"}
    ],
    "price_qualifier": "negotiable",
    "location": {
      "address": "Nong Kae (หนองแก), Hua Hin (หัวหิน)",
      "district": "Hua Hin (หัวหิน)",
      "province": "Prachuap Khiri Khan (ประจวบคีรีขันธ์)",
      "subdistrict": "Nong Kae (หนองแก)"
    },
    "bedrooms": 1,
    "bathrooms": 1,
    "area_sqm": 43.5,
    "property_type": "Condo",
    "listing_type": "Sale",
    "ownership": "Freehold",
    "completion_status": "Completed",
    "year_built": 2012,
    "furnishing": "Partially Furnished",
    "description": "Condo for sale in great location near the beach. Comfortable living with full facilities including large swimming pool.",
    "features": ["swimming pool", "fitness", "security 24hr", "key card access"],
    "facilities": ["swimming pool", "fitness center", "lobby"],
    "nearby": ["beach", "shops", "restaurants"],
    "agent": {
      "name": "Owner",
      "phone": "0*****",
      "type": "Owner"
    },
    "listing_id": "500022549",
    "listing_date": "2026-01-01",
    "url": "https://www.ddproperty.com/property/the-seacraze-..."
  }
}
```

## Notes

- **FlareSolverr required**: DDProperty uses Cloudflare protection
- Use `--fetch-mode dynamic` as DDProperty is JavaScript-heavy
- Add `--delay 2s` to be respectful to the server
- The LLM translates Thai to English, keeping Thai in parentheses for proper nouns
- Thai Buddhist year (พ.ศ.) is converted to Gregorian by subtracting 543
- Exchange rates: 1 USD = 35 THB, 1 GBP = 45 THB

## Running FlareSolverr

```bash
docker run -d \
  --name flaresolverr \
  -p 8191:8191 \
  -e LOG_LEVEL=info \
  ghcr.io/flaresolverr/flaresolverr:latest
```
