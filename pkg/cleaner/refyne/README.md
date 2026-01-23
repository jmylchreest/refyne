# Refyne Content Cleaner

A highly configurable, heuristic-based HTML content cleaner designed to reduce token usage while preserving meaningful content for LLM extraction.

## Features

- **Configurable**: Every operation is opt-in/opt-out
- **Heuristic-based**: No ML dependencies, pure Go
- **Content-agnostic**: Works for articles, e-commerce, recipes, listings
- **Graceful degradation**: Never fails - returns original on errors with warnings
- **Detailed stats**: Track exactly what was removed and why
- **Presets**: Ready-to-use configurations for common use cases

## Installation

```go
import "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
```

## Quick Start

```go
// Use default config (safe for all content types)
cleaner := refyne.New(nil)
result := cleaner.CleanWithStats(htmlContent)

fmt.Printf("Reduced from %d to %d bytes (%.1f%%)\n",
    result.Stats.InputBytes,
    result.Stats.OutputBytes,
    result.Stats.ReductionPercent())

fmt.Println(result.Content)
```

## Configuration

### Default Config

The default configuration is designed to work safely across all content types:

```go
cfg := refyne.DefaultConfig()
```

**What it removes:**
- `<script>` tags and contents
- `<style>` tags and `style=""` attributes
- HTML comments
- `<noscript>`, `<svg>`, `<iframe>` elements
- Hidden elements (`[hidden]`, `aria-hidden="true"`, `display:none`)
- Event handlers (`onclick`, `onload`, etc.)
- `data-*` and `aria-*` attributes

**What it preserves:**
- Links, images, tables
- Forms (important for recipe checkboxes, filters)
- Lists (ul, ol, li, dl)
- Semantic HTML5 elements
- Classes and IDs (needed for selectors)

### Custom Configuration

```go
cfg := &refyne.Config{
    // Remove scripts and styles
    StripScripts: true,
    StripStyles:  true,

    // Remove specific elements by selector
    RemoveSelectors: []string{
        "nav",
        "footer",
        ".advertisement",
        "[role='navigation']",
    },

    // Always keep these even if matched by RemoveSelectors
    KeepSelectors: []string{
        ".main-content",
        "#product-details",
    },

    // Enable aggressive heuristics
    RemoveByLinkDensity:  true,
    LinkDensityThreshold: 0.5, // Remove if >50% of text is links

    // Output format
    Output: refyne.OutputHTML, // or OutputText, OutputMarkdown
}

cleaner := refyne.New(cfg)
```

### Presets

Pre-configured settings for common use cases:

```go
// Minimal cleaning - maximum content preservation
cfg := refyne.PresetMinimal()

// Aggressive - for articles/blog posts
cfg := refyne.PresetAggressive()

// E-commerce - preserves product tables, images, prices
cfg := refyne.PresetEcommerce()

// Recipes - preserves forms, lists, nutrition tables
cfg := refyne.PresetRecipe()
```

## Configuration Reference

### Removal Options

| Option | Default | Description |
|--------|---------|-------------|
| `StripScripts` | `true` | Remove `<script>` tags |
| `StripStyles` | `true` | Remove `<style>` tags and `style=""` attributes |
| `StripComments` | `true` | Remove HTML comments |
| `StripHiddenElements` | `true` | Remove `display:none`, `[hidden]`, `aria-hidden` |
| `StripEmptyElements` | `false` | Remove elements with no content |
| `StripEventHandlers` | `true` | Remove `onclick`, `onload`, etc. |
| `StripSVGContent` | `true` | Remove inline SVG (often decorative) |
| `StripIframes` | `true` | Remove iframes (ads, embeds) |
| `StripNoscript` | `true` | Remove noscript fallbacks |

### Attribute Cleaning

| Option | Default | Description |
|--------|---------|-------------|
| `StripDataAttributes` | `true` | Remove `data-*` attributes |
| `StripClasses` | `false` | Remove `class=""` attributes |
| `StripIDs` | `false` | Remove `id=""` attributes |
| `StripARIA` | `true` | Remove `aria-*` attributes |
| `StripMicrodata` | `false` | Remove `itemscope`, `itemprop`, etc. |

### Preservation (Override Removals)

| Option | Default | Description |
|--------|---------|-------------|
| `PreserveLinks` | `true` | Keep `<a>` elements |
| `PreserveImages` | `true` | Keep `<img>` elements |
| `PreserveTables` | `true` | Keep table structure |
| `PreserveForms` | `true` | Keep form elements |
| `PreserveLists` | `true` | Keep ul/ol/li/dl structure |
| `PreserveSemanticTags` | `true` | Keep article, main, section, etc. |

### Selector-Based Rules

| Option | Type | Description |
|--------|------|-------------|
| `RemoveSelectors` | `[]string` | CSS selectors to always remove |
| `KeepSelectors` | `[]string` | CSS selectors to always keep (overrides removals) |

**Supported selector syntax:**
- Tag: `nav`, `footer`, `aside`
- Class: `.advertisement`, `.sidebar`
- ID: `#navigation`, `#footer`
- Attribute: `[hidden]`, `[role='navigation']`
- Attribute contains: `[class*='ad']`
- Descendant: `nav a`, `footer .links`
- Child: `ul > li`
- Multiple: `nav, footer, .sidebar`

### Heuristics

| Option | Default | Description |
|--------|---------|-------------|
| `RemoveByLinkDensity` | `false` | Remove high link-density elements |
| `LinkDensityThreshold` | `0.5` | Ratio of link text to total text |
| `RemoveShortText` | `false` | Remove elements with little text |
| `MinTextLength` | `20` | Minimum characters to keep |

### Output Options

| Option | Default | Description |
|--------|---------|-------------|
| `Output` | `OutputHTML` | Output format: `html`, `text`, or `markdown` |
| `CollapseWhitespace` | `true` | Normalize multiple spaces to single |
| `TrimElements` | `true` | Trim leading/trailing whitespace |
| `Debug` | `false` | Enable verbose logging |

## Working with Stats

The cleaner provides detailed statistics about what was done:

```go
result := cleaner.CleanWithStats(html)

fmt.Printf("Size: %d -> %d bytes (%.1f%% reduction)\n",
    result.Stats.InputBytes,
    result.Stats.OutputBytes,
    result.Stats.ReductionPercent())

fmt.Printf("Elements removed: %d\n", result.Stats.TotalElementsRemoved())
fmt.Printf("Attributes removed: %d\n", result.Stats.AttributesRemoved)

// Breakdown by tag
for tag, count := range result.Stats.ElementsRemoved {
    fmt.Printf("  %s: %d\n", tag, count)
}

// Timing
fmt.Printf("Parse: %v, Transform: %v, Total: %v\n",
    result.Stats.ParseDuration,
    result.Stats.TransformDuration,
    result.Stats.TotalDuration)
```

## Handling Warnings

The cleaner never fails - it returns original content on errors:

```go
result := cleaner.CleanWithStats(html)

if result.HasWarnings() {
    for _, warn := range result.Warnings {
        log.Printf("[%s] %s: %s", warn.Phase, warn.Message, warn.Context)
    }
}

// Content is always usable
processContent(result.Content)
```

## Supported Input Formats

- **HTML**: Full HTML documents or fragments
- **XHTML**: XML-style HTML (self-closing tags handled)
- **Malformed HTML**: Gracefully handled via goquery's parser

## Integration with Cleaner Chain

The refyne cleaner implements the `cleaner.Cleaner` interface:

```go
import (
    "github.com/jmylchreest/refyne/pkg/cleaner"
    "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
)

// Use in a chain
chain := cleaner.NewChain(
    refyne.New(refyne.DefaultConfig()),
    cleaner.NewMarkdown(), // Convert to markdown after cleaning
)

cleaned, err := chain.Clean(html)
```

## Performance Considerations

1. **Single pass where possible**: Most transformations happen in one DOM traversal
2. **Selector caching**: Compiled selectors are reused
3. **Empty element removal**: Uses multiple passes (configurable) to catch nested empties
4. **Memory**: Uses goquery which builds a full DOM tree

Typical performance on a complex e-commerce page (~200KB):
- Parse: ~5-10ms
- Transform: ~10-20ms
- Output: ~2-5ms
- Total: ~20-35ms

## Comparison to Other Cleaners

| Feature | Refyne | Trafilatura | Readability |
|---------|--------|-------------|-------------|
| Configurable | Highly | Limited | Limited |
| Preserves forms | Yes | No | No |
| Preserves tables | Yes | Optional | Yes |
| Product pages | Good | Poor | Poor |
| Articles | Good | Excellent | Excellent |
| Token reduction | 50-80% | 80-95% | 70-90% |
| False positives | Low | Medium | Medium |

## Future Enhancements

- [ ] Markdown output support
- [ ] Learning/feedback mechanism for domain-specific tuning
- [ ] Per-domain rule storage
- [ ] Automatic threshold adjustment based on extraction outcomes
