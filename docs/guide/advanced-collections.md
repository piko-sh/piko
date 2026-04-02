---
title: Advanced collections
description: Advanced collection features including filtering, sorting, pagination, and navigation trees
nav:
  sidebar:
    section: "guide"
    subsection: "advanced"
    order: 720
---

# Advanced collections

This guide covers advanced collection operations not found in the [basic Collections guide](/docs/guide/collections). You should be familiar with `piko.GetData[T](r)` and basic collection usage before reading this.

Import the `collection` package for the utilities described in this guide:

```go
import "piko.sh/piko/wdk/collection"
```

## Filtering collections

### Filter structure

The `Filter` struct defines a single filter condition:

```go
type Filter struct {
    Field    string         // Metadata field name to match
    Operator FilterOperator // Comparison operator
    Value    any            // Value to compare against
}
```

### Filter operators

Piko supports these comparison operators:

| Operator | Constant | Description |
|----------|----------|-------------|
| `eq` | `FilterOpEquals` | Exact value matching |
| `ne` | `FilterOpNotEquals` | Inequality comparison |
| `gt` | `FilterOpGreaterThan` | Greater than |
| `gte` | `FilterOpGreaterEqual` | Greater than or equal |
| `lt` | `FilterOpLessThan` | Less than |
| `lte` | `FilterOpLessEqual` | Less than or equal |
| `contains` | `FilterOpContains` | String contains substring |
| `startsWith` | `FilterOpStartsWith` | String starts with prefix |
| `endsWith` | `FilterOpEndsWith` | String ends with suffix |
| `in` | `FilterOpIn` | Value exists in array |
| `notIn` | `FilterOpNotIn` | Value not in array |
| `exists` | `FilterOpExists` | Field is present |
| `fuzzy` | `FilterOpFuzzyMatch` | Approximate text matching |

### Filter examples

**Filter by exact value:**

```go
filter := collection.Filter{
    Field:    "featured",
    Operator: collection.FilterOpEquals,
    Value:    true,
}
```

**Filter by date range:**

```go
filter := collection.Filter{
    Field:    "publishedAt",
    Operator: collection.FilterOpGreaterThan,
    Value:    "2025-01-01",
}
```

**Filter by category (in array):**

```go
filter := collection.Filter{
    Field:    "category",
    Operator: collection.FilterOpIn,
    Value:    []string{"tutorial", "guide", "reference"},
}
```

**Text search with fuzzy matching:**

```go
// Approximate title matching (useful for search)
// Uses a similarity threshold of 0.3
filter := collection.Filter{
    Field:    "title",
    Operator: collection.FilterOpFuzzyMatch,
    Value:    "getting started",
}
```

**Check if field exists:**

```go
filter := collection.Filter{
    Field:    "author",
    Operator: collection.FilterOpExists,
    Value:    true, // true = field must exist, false = field must not exist
}
```

### Filter groups

Combine multiple filters with AND/OR logic using `FilterGroup`:

```go
type FilterGroup struct {
    Logic   string   // "AND" or "OR"
    Filters []Filter
}
```

**Example: AND logic (all conditions must match)**

```go
filterGroup := &collection.FilterGroup{
    Logic: "AND",
    Filters: []collection.Filter{
        {Field: "published", Operator: collection.FilterOpEquals, Value: true},
        {Field: "category", Operator: collection.FilterOpEquals, Value: "tutorial"},
    },
}
```

**Example: OR logic (any condition matches)**

```go
filterGroup := &collection.FilterGroup{
    Logic: "OR",
    Filters: []collection.Filter{
        {Field: "author", Operator: collection.FilterOpEquals, Value: "Alice"},
        {Field: "author", Operator: collection.FilterOpEquals, Value: "Bob"},
    },
}
```

### Applying filters

Use `collection.ApplyFilterGroup` to check if an item matches:

```go
item := &collection.ContentItem{
    Metadata: map[string]any{
        "published": true,
        "category":  "tutorial",
    },
}

filterGroup := &collection.FilterGroup{
    Logic: "AND",
    Filters: []collection.Filter{
        {Field: "published", Operator: collection.FilterOpEquals, Value: true},
    },
}

if collection.ApplyFilterGroup(item, filterGroup) {
    // Item matches the filter criteria
}
```

## Sorting collections

### Sort options

The `SortOption` struct specifies field ordering:

```go
type SortOption struct {
    Field string    // Metadata field name to sort by
    Order SortOrder // SortAsc, SortDesc, or SortRandom
}
```

### Sort order constants

```go
const (
    SortAsc    SortOrder = "asc"    // Ascending (A-Z, 0-9, oldest first)
    SortDesc   SortOrder = "desc"   // Descending (Z-A, 9-0, newest first)
    SortRandom SortOrder = "random" // Shuffle items randomly
)
```

### Sorting examples

**Sort by single field:**

```go
sortOptions := []collection.SortOption{
    {Field: "publishedAt", Order: collection.SortDesc},
}
```

**Sort by multiple fields (primary, then secondary):**

```go
sortOptions := []collection.SortOption{
    {Field: "category", Order: collection.SortAsc},    // First by category
    {Field: "publishedAt", Order: collection.SortDesc}, // Then by date
}
```

**Random ordering:**

```go
// When the first sort option is random, items are shuffled
// and any subsequent sort options are ignored
sortOptions := []collection.SortOption{
    {Field: "", Order: collection.SortRandom},
}
```

### Applying sorting

Use `collection.SortItems` to sort a slice of items in place:

```go
items := []*collection.ContentItem{...}

sortOptions := []collection.SortOption{
    {Field: "publishedAt", Order: collection.SortDesc},
}

// Sorts items in place
collection.SortItems(items, sortOptions)
```

### Type-aware sorting

Sorting handles different types intelligently:

- **Strings.** Case-insensitive lexicographic comparison.
- **Numbers.** Numeric comparison (works with int, float64, etc.).
- **Dates.** Chronological comparison (supports RFC3339, `YYYY-MM-DD`, `YYYY-MM-DD HH:MM:SS`).
- **Booleans.** `false` sorts before `true`.
- **Missing fields.** Items without a field sort before items with the field.

## Pagination

### Pagination options

The `PaginationOptions` struct supports both page-based and offset-based pagination:

```go
type PaginationOptions struct {
    // Page-based pagination
    Page     int // Page number, starting from 1
    PageSize int // Number of items per page

    // Offset-based pagination (alternative to Page)
    Offset int // Items to skip
    Limit  int // Maximum items to return
}
```

### Page-based pagination

```go
pagination := &collection.PaginationOptions{
    Page:     1,  // First page
    PageSize: 10, // 10 items per page
}
```

### Offset-based pagination

```go
pagination := &collection.PaginationOptions{
    Offset: 20, // Skip first 20 items
    Limit:  10, // Return next 10 items
}
```

### Applying pagination

Use `collection.PaginateItems` to apply pagination:

```go
items := []*collection.ContentItem{...}

pagination := &collection.PaginationOptions{
    Page:     2,
    PageSize: 10,
}

// Returns a new slice with the requested page
paginatedItems := collection.PaginateItems(items, pagination)
```

### Pagination metadata

Calculate pagination metadata for building UI controls:

```go
type PaginationMeta struct {
    CurrentPage int  // Current page number (1-indexed)
    PageSize    int  // Items per page
    TotalItems  int  // Total number of items
    TotalPages  int  // Total number of pages
    HasNextPage bool // Whether there's a next page
    HasPrevPage bool // Whether there's a previous page
}
```

Use `collection.CalculatePaginationMeta` to compute metadata:

```go
totalItems := 95
pagination := &collection.PaginationOptions{
    Page:     2,
    PageSize: 10,
}

meta := collection.CalculatePaginationMeta(totalItems, pagination)
// meta.CurrentPage = 2
// meta.TotalPages = 10
// meta.HasNextPage = true
// meta.HasPrevPage = true
```

## Navigation trees

Navigation trees provide hierarchical navigation for documentation sites, organised into sections and subsections.

### Navigation structure

```go
// NavigationGroups contains multiple named navigation structures
type NavigationGroups struct {
    Groups map[string]*NavigationTree // Named navigation trees
}

// NavigationTree represents a hierarchy for a specific locale
type NavigationTree struct {
    Locale   string            // ISO 639-1 language code
    Sections []*NavigationNode // Top-level navigation groups
}

// NavigationNode represents a single node in the hierarchy
type NavigationNode struct {
    ID          string            // Unique identifier
    Title       string            // Display name
    URL         string            // Link target (empty for category nodes)
    Section     string            // Top-level section name
    Subsection  string            // Secondary section name
    Icon        string            // Optional icon name
    Order       int               // Sort position (lower = first)
    Level       int               // Depth in hierarchy (0 = root)
    Hidden      bool              // Whether to exclude from navigation
    Children    []*NavigationNode // Child nodes
    Parent      *NavigationNode   // Parent node reference
    ContentItem *ContentItem      // Full content item (if applicable)
}
```

### Navigation frontmatter

Configure navigation in your markdown frontmatter:

```yaml
---
title: Installation Guide
nav:
  sidebar:
    section: "getting-started"
    subsection: "basics"
    order: 1
    label: "Install"       # Override title in navigation
    icon: "download"       # Optional icon
    hidden: false          # Whether to hide from navigation
---
```

### Navigation node methods

```go
// Check if this node is a category (grouping) node
if node.IsCategory() {
    // Render as expandable section (no URL)
}

// Check if this node has no children
if node.IsLeaf() {
    // Render as single item
}

// Check if this node represents actual content
if node.HasContent() {
    // Access node.ContentItem for full data
}

// Get breadcrumb trail from root to this node
breadcrumb := node.GetBreadcrumb()
// Returns: [root, parent, ..., this]

// Count all descendant nodes
count := node.CountDescendants()
// Use for "Section (42 articles)"

// Find a specific node by ID
activeNode := rootNode.FindNodeByID("installation")
```

### Navigation configuration

```go
type NavigationConfig struct {
    Locale         string // Filter to specific locale
    IncludeHidden  bool   // Include hidden items (default: false)
    DefaultOrder   int    // Order for items without explicit order (default: 999)
    GroupBySection bool   // Create section nodes even if empty (default: true)
}

// Get default configuration
config := collection.DefaultNavigationConfig()
```

## Table of contents

Generate a table of contents from content headings using `piko.GetSections` or `piko.GetSectionsTree`.

### Flat section list

```go
import "piko.sh/piko"

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    doc := piko.GetData[Doc](r)

    // Get flat list of headings
    sections := piko.GetSections(r)

    return Response{
        Title:    doc.Title,
        Sections: sections,
    }, piko.Metadata{}, nil
}
```

### Hierarchical section tree

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    doc := piko.GetData[Doc](r)

    // Get hierarchical tree (defaults: h2-h4)
    tree := piko.GetSectionsTree(r)

    // Or with custom level filtering
    customTree := piko.GetSectionsTree(r,
        piko.WithMinLevel(2), // Start from h2
        piko.WithMaxLevel(3), // Include up to h3 only
    )

    return Response{
        Title:           doc.Title,
        TableOfContents: tree,
    }, piko.Metadata{}, nil
}
```

### Section node structure

```go
type SectionNode struct {
    Title    string        // Heading text
    Slug     string        // URL-safe anchor ID
    Level    int           // Heading level (2-6)
    Children []SectionNode // Nested sections
}
```

### Section tree configuration

```go
type SectionTreeConfig struct {
    MinLevel int // Minimum heading level (default: 2, skip h1)
    MaxLevel int // Maximum heading level (default: 4)
}

config := collection.DefaultSectionTreeConfig()
// MinLevel: 2 (start at h2)
// MaxLevel: 4 (include h2, h3, h4)
```

### Table of contents template example

```piko
<template>
  <!-- Render table of contents -->
  <nav class="toc">
    <h3>On This Page</h3>
    <ul>
      <li p-for="(_, section) in state.TableOfContents">
        <a :href="'#' + section.Slug">{{ section.Title }}</a>
        <ul p-if="len(section.Children) > 0">
          <li p-for="(_, child) in section.Children">
            <a :href="'#' + child.Slug">{{ child.Title }}</a>
          </li>
        </ul>
      </li>
    </ul>
  </nav>

  <!-- Main content -->
  <article>
    <piko:content />
  </article>
</template>
```

## Hybrid caching (ISR)

Hybrid mode combines static generation with runtime revalidation for optimal performance.

### How hybrid mode works

1. **Build-time.** Generate static snapshot with computed ETag.
2. **Runtime.** Serve snapshot immediately, validate ETag in background.
3. **Revalidation.** Only refetch when ETag changes.
4. **Fallback.** Serve stale content if revalidation fails.

### Hybrid configuration

```go
type HybridConfig struct {
    // ETagSource specifies how to compute the ETag
    // Options: "modtime-hash", "content-hash", "provider-etag"
    // Default: "modtime-hash"
    ETagSource string

    // RevalidationTTL is how long before checking if cached data is stale
    // Default: 60 seconds
    // Zero value: check every request (not recommended for production)
    RevalidationTTL time.Duration

    // MaxStaleAge is the maximum age before forcing a refresh
    // Default: 0 (no maximum, content can be arbitrarily stale)
    MaxStaleAge time.Duration

    // StaleIfError serves stale content when revalidation fails
    // Default: true
    StaleIfError bool
}

// Get default configuration
config := collection.DefaultHybridConfig()
```

### ETag sources

| Source | Description | Use case |
|--------|-------------|----------|
| `modtime-hash` | Hash of file modification times | Local markdown files (fast, default) |
| `content-hash` | Hash of file contents | When modtimes aren't reliable |
| `provider-etag` | ETag from provider | HTTP-based CMS providers |

### ETag source constants

```go
const (
    ETagSourceModtimeHash  = "modtime-hash"
    ETagSourceContentHash  = "content-hash"
    ETagSourceProviderETag = "provider-etag"
)
```

## Multi-locale collections

Collections support multiple locales for internationalisation.

### Locale organisation

Piko supports several locale patterns:

**Language-first pattern:**
```text
content/
├── en/
│   └── blog/
│       └── hello.md
└── fr/
    └── blog/
        └── hello.md
```

**Suffix pattern:**
```text
content/
└── blog/
    ├── hello.en.md
    └── hello.fr.md
```

**Content-first pattern:**
```text
content/
└── blog/
    └── hello/
        ├── en.md
        └── fr.md
```

### Translation keys

Items across locales are linked by translation key:

```go
type ContentItem struct {
    // ...other fields
    TranslationKey string // Links translations (e.g., "blog/hello")
    Locale         string // ISO 639-1 code (e.g., "en", "fr")
}
```

## Content item structure

Every collection item includes standard metadata fields:

```go
type ContentItem struct {
    ID             string         // Unique identifier
    Slug           string         // URL-safe identifier
    URL            string         // Full URL path
    Locale         string         // ISO 639-1 language code
    TranslationKey string         // Links translations
    CreatedAt      string         // Creation timestamp (ISO 8601)
    UpdatedAt      string         // Last update timestamp (ISO 8601)
    PublishedAt    string         // Publication timestamp (ISO 8601)
    ReadingTime    int            // Estimated reading time (minutes)
    Metadata       map[string]any // Custom frontmatter fields
}
```

### Helper methods

```go
// Get metadata with type safety and fallback
title := item.GetMetadataString("title", "Untitled")
viewCount := item.GetMetadataInt("views", 0)
isDraft := item.GetMetadataBool("draft", false)
tags := item.GetMetadataStringSlice("tags", nil)

// Check field existence
if item.HasMetadata("author") {
    // Field exists
}

// Check publication status
if item.IsPublished() {
    // Has PublishedAt timestamp
}

if item.IsDraft() {
    // No PublishedAt or draft: true
}
```

## FetchOptions

The `FetchOptions` struct configures how collections are fetched:

```go
type FetchOptions struct {
    // ProviderName explicitly specifies which provider to use
    // If empty, the default provider from config is used
    ProviderName string

    // Locale specifies a single locale to fetch
    // Mutually exclusive with ExplicitLocales and AllLocales
    Locale string

    // ExplicitLocales specifies multiple locales to fetch
    // Cannot be used with Locale or AllLocales
    ExplicitLocales []string

    // AllLocales fetches content for all configured locales
    // Cannot be used with Locale or ExplicitLocales
    AllLocales bool

    // FilterGroup holds structured filter conditions
    FilterGroup *FilterGroup

    // Sort specifies sorting options
    // Multiple sort options are applied in order
    Sort []SortOption

    // Pagination specifies the starting position and number of results
    Pagination *PaginationOptions

    // Cache specifies caching settings
    // If nil, the provider's default caching behaviour is used
    Cache *CacheConfig

    // Filters holds provider-specific filter values as key-value pairs
    // The supported keys depend on the provider
    Filters map[string]any

    // Projection specifies which fields to include/exclude
    // Reduces payload size by omitting fields the client doesn't need
    Projection *FieldProjection
}
```

### Cache configuration

```go
type CacheConfig struct {
    // Strategy defines the caching approach
    // Options: "cache-first", "network-first", "stale-while-revalidate", "no-cache"
    Strategy string

    // Key is an optional custom cache key
    // If empty, a key is generated from collection name, locale, and filters
    Key string

    // Tags lists cache invalidation tags
    Tags []string

    // TTL is the cache time-to-live in seconds
    TTL int
}
```

### Field projection

```go
type FieldProjection struct {
    // IncludeFields lists metadata fields to include (empty = all)
    IncludeFields []string

    // ExcludeFields lists metadata fields to exclude
    ExcludeFields []string

    // MaxArrayItems limits items in array fields (0 = no limit)
    MaxArrayItems int32
}
```

## Next steps

- [Collections](/docs/guide/collections) → Basic collection setup, declarative collections, piko:content
