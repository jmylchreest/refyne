# Recipe Extraction Example

Extract structured recipe data from cooking websites.

## Usage

### Using the CLI with the schema file:

```bash
# From repository root
task example:recipes:cli -- "https://www.allrecipes.com/recipe/10813/best-chocolate-chip-cookies/"

# Or directly
./bin/refyne scrape \
  -u "https://www.allrecipes.com/recipe/10813/best-chocolate-chip-cookies/" \
  -s examples/recipes/schema.yaml \
  --format yaml
```

### Using the Go example code:

```bash
task example:recipes -- "https://www.allrecipes.com/recipe/10813/best-chocolate-chip-cookies/"
```

## Schema Fields

- **title**: Recipe name
- **description**: Brief description of the dish
- **prep_time**: Preparation time
- **cook_time**: Cooking time
- **servings**: Number of servings
- **ingredients**: List of ingredients with name, amount, and notes
- **steps**: Step-by-step instructions

## Example Output

```yaml
title: Best Chocolate Chip Cookies
description: These chocolate chip cookies are the best I have ever made.
prep_time: 20 minutes
cook_time: 10 minutes
servings: 24
ingredients:
  - name: all-purpose flour
    amount: 2 1/4 cups
  - name: butter
    amount: 1 cup
    notes: softened
  - name: chocolate chips
    amount: 2 cups
steps:
  - Preheat oven to 375 degrees F.
  - Combine flour, baking soda and salt in small bowl.
  - Beat butter, sugars and vanilla in large mixer bowl.
  ...
```
