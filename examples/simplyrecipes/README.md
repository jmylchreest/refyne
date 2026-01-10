# Simply Recipes Example

Extract structured recipe data from Simply Recipes website.

## Usage

### Single recipe extraction:

```bash
# From repository root
task example:simplyrecipes:cli -- "https://www.simplyrecipes.com/homemade-coffee-creamer-recipe-8363223"

# Or directly
./bin/refyne scrape \
  -u "https://www.simplyrecipes.com/steak-tips-recipe-7972730" \
  -s examples/simplyrecipes/schema.yaml \
  --fetch-mode static
```

### Crawling recipe list:

```bash
# Crawl a category page, following links to individual recipes
task example:simplyrecipes:crawl -- "https://www.simplyrecipes.com/recipes-5090746"

# Or directly with more options
./bin/refyne scrape \
  -u "https://www.simplyrecipes.com/dinner-recipes-5091433" \
  -s examples/simplyrecipes/schema.yaml \
  --follow "a[href*='-recipe-']" \
  --max-depth 1 \
  --delay 1s \
  --fetch-mode static \
  --format jsonl \
  -o recipes.jsonl
```

## Schema Fields

### Basic Info
- **title**: Recipe name
- **description**: Brief introduction
- **author**: Recipe author
- **category**: Recipe category (Dinners, Desserts, etc.)
- **cuisine**: Cuisine type if specified

### Timing
- **prep_time**: Preparation time
- **cook_time**: Cooking time
- **total_time**: Total time
- **servings**: Number of servings

### Recipe Content
- **ingredients**: List of ingredients with amount, name, and notes
- **instructions**: Step-by-step cooking instructions
- **notes**: Tips, variations, or storage info
- **image_url**: Main recipe image

## Example Output

```json
{
  "title": "Steak Tips",
  "description": "Tender, juicy steak tips in a savory sauce...",
  "author": "Summer Miller",
  "prep_time": "10 minutes",
  "cook_time": "20 minutes",
  "total_time": "30 minutes",
  "servings": "4",
  "category": "Dinners",
  "ingredients": [
    {"amount": "1.5 lbs", "name": "sirloin steak", "notes": "cut into 1-inch cubes"},
    {"amount": "2 tbsp", "name": "olive oil"},
    {"amount": "3 cloves", "name": "garlic", "notes": "minced"}
  ],
  "instructions": [
    "Season the steak tips with salt and pepper.",
    "Heat oil in a large skillet over high heat.",
    "Add steak tips and cook until browned, about 3 minutes per side."
  ],
  "notes": [
    "Use a cast iron skillet for the best sear.",
    "Let the steak rest for 5 minutes before serving."
  ],
  "image_url": "https://www.simplyrecipes.com/thmb/..."
}
```

## Notes

- Simply Recipes works well with static fetching (no JavaScript required)
- Recipe links follow the pattern `*-recipe-*` in the URL
- Use `--delay 1s` to be respectful of their servers
