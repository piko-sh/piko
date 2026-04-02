---
title: Testing framework
description: Guide to testing Piko components and server actions
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 140
---

# Testing framework

Piko provides testing utilities for writing unit tests and benchmarks for `.pk` components. The testing utilities are exposed through the `piko.sh/piko` package, allowing you to test components using standard Go testing tools without JavaScript frameworks.

## Quick example

```go
package pages_test

import (
    "context"
    "testing"

    "piko.sh/piko"
    "github.com/stretchr/testify/assert"
    customers "myapp/dist/pages/pages_customers_abc123"
)

func TestCustomersPage(t *testing.T) {
    // Setup mock
    mockDB := &MockDB{Customers: []Customer{{Name: "Acme"}}}
    ctx := context.WithValue(context.Background(), "db", mockDB)

    // Build request
    req := piko.NewTestRequest("GET", "/customers").
        WithContext(ctx).
        Build()

    // Test component
    tester := piko.NewComponentTester(t, customers.BuildAST)
    view := tester.Render(req, piko.NoProps{})

    // Assert using AST queries
    view.QueryAST(".customer-row").Count(1)
    view.QueryAST("h1").HasText("Customers")
    view.AssertTitle("Customers - MyApp")
}
```

## Test file structure

Place test files next to your compiled components in your output directory:

```text
dist/
└── pages/
    └── pages_customers_abc123/
        ├── component.go      // Generated component
        └── component_test.go // Your test file
```

Alternatively, place tests wherever makes sense for your project structure, as long as you import the generated component package.

## Building test requests

Use `piko.NewTestRequest` with a fluent API:

```go
req := piko.NewTestRequest("GET", "/customers").
    WithQueryParam("sort", "desc").
    WithQueryParam("filter", "active").
    WithPathParam("id", "123").
    WithFormData("name", "Test").
    WithContext(ctx).
    WithLocale("fr").
    WithHost("example.com").
    Build()
```

### Request builder methods

| Method | Purpose |
|--------|---------|
| `WithContext(ctx)` | Inject dependencies via context |
| `WithQueryParam(key, value)` | Add a single URL query parameter |
| `WithQueryParams(params)` | Add multiple query parameters from `map[string][]string` |
| `WithPathParam(key, value)` | Add a single route path parameter |
| `WithPathParams(params)` | Add multiple path parameters from `map[string]string` |
| `WithFormData(key, value)` | Add a single form field |
| `WithFormDataMap(data)` | Add multiple form fields from `map[string][]string` |
| `WithLocale(locale)` | Set request locale (default: "en") |
| `WithDefaultLocale(locale)` | Set fallback locale (default: "en") |
| `WithHost(host)` | Set host header (default: "localhost") |
| `WithHeader(key, value)` | Add HTTP header |
| `WithGlobalTranslations(translations)` | Set global translation map |
| `WithLocalTranslations(translations)` | Set component-specific translations |
| `WithCollectionData(data)` | Set collection data for `p-collection` pages |
| `Build()` | Build the final `*RequestData` |
| `BuildHTTPRequest()` | Build both `*http.Request` and `*RequestData` |

## Mocking dependencies

### Context injection (recommended)

Pass mocks via context for clean, isolated tests:

```go
// In your test
mockRepo := &MockCustomerRepo{
    Customers: []Customer{{Name: "Acme"}},
}
ctx := context.WithValue(context.Background(), "repo", mockRepo)
req := piko.NewTestRequest("GET", "/").WithContext(ctx).Build()
```

Your component's Render function accesses the mock via context:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    repo := r.Context().Value("repo").(CustomerRepository)
    customers := repo.GetAll()
    // ...
}
```

### Mock example

```go
type MockCustomerRepo struct {
    Customers    []Customer
    GetAllCalled bool
    ShouldError  bool
}

func (m *MockCustomerRepo) GetAll() []Customer {
    m.GetAllCalled = true
    return m.Customers
}

func (m *MockCustomerRepo) GetByID(id string) (*Customer, error) {
    if m.ShouldError {
        return nil, errors.New("database error")
    }
    for _, c := range m.Customers {
        if c.ID == id {
            return &c, nil
        }
    }
    return nil, errors.New("not found")
}
```

## Creating component testers

Import your pre-compiled component and create a tester:

```go
import customers "myapp/dist/pages/pages_customers_abc123"

tester := piko.NewComponentTester(t, customers.BuildAST)
```

### Component tester options

```go
// Set a page ID for error messages and debugging
tester := piko.NewComponentTester(t, customers.BuildAST,
    piko.WithPageID("pages/customers"),
)
```

### Rendering components

```go
// Render without props
view := tester.Render(req, piko.NoProps{})

// Render with props
view := tester.Render(req, CustomerProps{ID: "123"})
```

## AST queries (recommended)

Query the template AST using CSS selectors. This is significantly faster than rendering HTML and provides direct access to the component's output structure.

### Basic queries

```go
// Check existence
view.QueryAST("h1").Exists()
view.QueryAST(".non-existent").NotExists()

// Count elements
view.QueryAST(".customer-row").Count(10)
view.QueryAST(".item").MinCount(1)
view.QueryAST(".item").MaxCount(100)

// Check text content
view.QueryAST("h1").HasText("Customers")
view.QueryAST("p").ContainsText("Welcome")
```

### Attribute assertions

```go
// Check exact attribute value
view.QueryAST("input[name='email']").
    HasAttribute("type", "email").
    HasAttribute("required", "true")

// Check attribute contains substring
view.QueryAST(".product").HasAttributeContaining("data-id", "prod-")

// Check attribute exists (any value)
view.QueryAST("button").HasAttributePresent("disabled")

// Check CSS class
view.QueryAST("div").HasClass("container")

// Check tag name
view.QueryAST("#header").HasTag("header")
```

### CSS selector support

All standard CSS selectors are supported:

```go
// Basic selectors
view.QueryAST("div")           // Tag
view.QueryAST(".classname")    // Class
view.QueryAST("#id")           // ID
view.QueryAST("*")             // Universal

// Combinators
view.QueryAST("div > p")       // Direct child
view.QueryAST("div p")         // Descendant
view.QueryAST("div + p")       // Adjacent sibling
view.QueryAST("div ~ p")       // General sibling

// Attribute selectors
view.QueryAST("[name]")                   // Has attribute
view.QueryAST("[name='email']")           // Exact match
view.QueryAST("[name~='word']")           // Word in list
view.QueryAST("[lang|='en']")             // Dash-prefix match
view.QueryAST("[name^='user']")           // Starts with
view.QueryAST("[name$='_id']")            // Ends with
view.QueryAST("[name*='search']")         // Contains

// Pseudo-classes
view.QueryAST("li:first-child")
view.QueryAST("li:last-child")
view.QueryAST("li:only-child")
view.QueryAST("li:nth-child(2)")
view.QueryAST("li:nth-child(odd)")
view.QueryAST("li:nth-child(2n+1)")
view.QueryAST("li:nth-last-child(2)")
view.QueryAST("p:first-of-type")
view.QueryAST("p:last-of-type")
view.QueryAST("p:only-of-type")
view.QueryAST("p:nth-of-type(2)")
view.QueryAST("p:nth-last-of-type(2)")
view.QueryAST(":not(.hidden)")

// Combined selectors
view.QueryAST("div.container#main")
view.QueryAST("ul.list > li.item:first-child")
```

### Working with query results

```go
// Get specific nodes
firstRow := view.QueryAST(".customer-row").First()      // Returns *TemplateNode
lastRow := view.QueryAST(".customer-row").Last()        // Returns *TemplateNode
thirdRow := view.QueryAST(".customer-row").At(2)        // Returns *TemplateNode (0-indexed)

// Get wrapped result for chaining
firstResult := view.QueryAST(".stat-value").FirstResult()  // Returns *ASTQueryResult
firstResult.HasText("100")

// Index into results (returns *ASTQueryResult for chaining)
view.QueryAST(".stat-value").Index(0).HasText("100")
view.QueryAST(".stat-value").Index(1).HasText("42")

// Get raw node count
count := view.QueryAST(".item").Len()

// Get raw nodes slice
nodes := view.QueryAST(".item").Nodes()  // Returns []*TemplateNode

// Iterate over results
view.QueryAST(".customer-row").Each(func(i int, node *ast_domain.TemplateNode) {
    id, ok := node.GetAttribute("data-id")
    assert.True(t, ok, "Row %d missing data-id", i)
    assert.NotEmpty(t, id)
})

// Filter results
activeRows := view.QueryAST(".customer-row").Filter(func(node *ast_domain.TemplateNode) bool {
    status, _ := node.GetAttribute("data-status")
    return status == "active"
})
activeRows.Count(5)

// Map to values
customerIDs := view.QueryAST(".customer-row").Map(func(node *ast_domain.TemplateNode) any {
    id, _ := node.GetAttribute("data-customer-id")
    return id
})
assert.Contains(t, customerIDs, "123")

// Debug output
view.QueryAST(".customer-row").Dump()  // Logs matched nodes to test output
```

## Metadata assertions

Test SEO metadata, status codes, and redirects:

```go
// SEO metadata
view.AssertTitle("Page Title")
view.AssertDescription("Page description")
view.AssertLanguage("en")
view.AssertCanonicalURL("https://example.com/page")

// HTTP status
view.AssertStatusCode(200)
view.AssertDefaultStatusCode()  // Asserts no custom status was set

// Redirects
view.AssertClientRedirect("/login")
view.AssertServerRedirect("/internal-page")

// Meta tags
view.AssertHasMetaTag("author", "John Doe")
view.AssertHasOGTag("og:title", "Page Title")

// JavaScript scripts
view.AssertHasJSScript()
view.AssertNoJSScript()
view.AssertJSScriptURLContains("customers")

// Access raw metadata
metadata := view.Metadata()
```

## HTML rendering (when needed)

For integration tests or snapshot testing, render full HTML:

```go
// Get rendered HTML
html := view.HTML()
assert.Contains(t, string(html), "<h1>Customers</h1>")

// Get as string
htmlStr := view.HTMLString()

// Write to a writer (for debugging)
view.WriteTo(os.Stdout)
```

**Note:** HTML rendering requires a `RenderService` to be configured. AST queries are significantly faster and don't require this setup. Only render HTML when you specifically need to test the full rendering pipeline or use snapshot testing.

## Server action testing

Test action handlers with full validation:

```go
import "myapp/actions"

func TestCustomerCreateAction_Success(t *testing.T) {
    // Setup mock
    mockRepo := &MockCustomerRepo{}
    ctx := context.WithValue(context.Background(), "repo", mockRepo)

    // Create action tester
    tester := piko.NewActionTester(t, &actions.CustomerCreate{})

    // Build request with form data
    req := piko.NewTestRequest("POST", "/actions/customer_create").
        WithContext(ctx).
        WithFormData("name", "Acme Corp").
        WithFormData("email", "contact@acme.com").
        Build()

    // Invoke action
    resp := tester.Invoke(req)

    // Assert success
    tester.AssertSuccess(resp)
    assert.Equal(t, "Customer created successfully", resp.Message)

    // Check redirect helper
    tester.AssertHasHelper(resp, "redirect")
    tester.AssertHelperArg(resp, "redirect", 0, "/customers/123")

    // Verify mock was called
    assert.True(t, mockRepo.CreateCalled)
}
```

### Action assertions

```go
// Success/Error
tester.AssertSuccess(resp)           // Status 2xx, no errors
tester.AssertError(resp)             // Status non-2xx or has errors
tester.AssertStatusCode(resp, 422)
tester.AssertErrorMessage(resp, "Validation failed")

// Validation errors
tester.AssertFieldError(resp, "email", "Invalid email format")
tester.AssertHasFieldError(resp, "name")
tester.AssertNoFieldError(resp, "phone")

// Response helpers
tester.AssertHasHelper(resp, "redirect")
tester.AssertHelperArg(resp, "redirect", 0, "/customers/123")

helper := tester.GetHelper(resp, "toast")
assert.NotNil(t, helper)
```

### Testing validation errors

```go
func TestCustomerCreateAction_ValidationError(t *testing.T) {
    mockRepo := &MockCustomerRepo{}
    ctx := context.WithValue(context.Background(), "repo", mockRepo)

    tester := piko.NewActionTester(t, &actions.CustomerCreate{})

    // Missing required field
    req := piko.NewTestRequest("POST", "/").
        WithContext(ctx).
        WithFormData("email", "contact@acme.com").  // No name
        Build()

    resp := tester.Invoke(req)

    tester.AssertError(resp)
    tester.AssertStatusCode(resp, 422)
    tester.AssertFieldError(resp, "name", "Name is required")

    // Verify action didn't proceed
    assert.False(t, mockRepo.CreateCalled)
}
```

### Testing expected errors

```go
func TestCustomerCreateAction_NoRepository(t *testing.T) {
    tester := piko.NewActionTester(t, &actions.CustomerCreate{})

    // No context with repository
    req := piko.NewTestRequest("POST", "/").
        WithFormData("name", "Acme").
        Build()

    err := tester.InvokeExpectError(req)
    assert.Contains(t, err.Error(), "repository not found")
}
```

## Snapshot testing

Detect unintended changes with golden file comparison:

```go
func TestCustomersPage_Snapshot(t *testing.T) {
    // ... setup ...
    view := tester.Render(req, piko.NoProps{})
    view.MatchSnapshot("customers-page")
}
```

Snapshots are stored in `__snapshots__/<test-directory>/<name>.golden.html`.

### Updating snapshots

When making intentional changes:

```bash
PIKO_UPDATE_SNAPSHOTS=1 go test ./...
# or
UPDATE_SNAPSHOTS=1 go test ./...
```

**Note:** Snapshot testing requires HTML rendering, which needs a `RenderService` to be configured.

## Performance benchmarking

Measure component render performance:

```go
func BenchmarkCustomersPage(b *testing.B) {
    // Setup (excluded from benchmark)
    mockDB := &MockDB{Customers: generateMockCustomers(100)}
    ctx := context.WithValue(context.Background(), "db", mockDB)
    req := piko.NewTestRequest("GET", "/customers").
        WithContext(ctx).
        Build()

    tester := piko.NewComponentTester(b, customers.BuildAST)

    // Run benchmark - calls b.ResetTimer() internally
    tester.Benchmark(req, piko.NoProps{})
}
```

Run benchmarks:

```bash
go test -bench=. ./pages/...
go test -bench=BenchmarkCustomersPage -benchmem ./pages/...
```

## Table-driven tests

Organise multiple test cases efficiently:

```go
func TestCustomersPage(t *testing.T) {
    tests := []struct {
        name          string
        mockCustomers []Customer
        expectedCount int
        expectedTitle string
    }{
        {
            name:          "empty state",
            mockCustomers: []Customer{},
            expectedCount: 0,
            expectedTitle: "No Customers",
        },
        {
            name: "with customers",
            mockCustomers: []Customer{
                {Name: "Acme"},
                {Name: "Beta"},
            },
            expectedCount: 2,
            expectedTitle: "Customers",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockDB := &MockDB{Customers: tt.mockCustomers}
            ctx := context.WithValue(context.Background(), "db", mockDB)

            req := piko.NewTestRequest("GET", "/customers").
                WithContext(ctx).
                Build()

            tester := piko.NewComponentTester(t, customers.BuildAST)
            view := tester.Render(req, piko.NoProps{})

            view.QueryAST(".customer-row").Count(tt.expectedCount)
            view.QueryAST("h1").HasText(tt.expectedTitle)
        })
    }
}
```

## Complete example

A full test demonstrating component and action testing:

```go
package pages_test

import (
    "context"
    "errors"
    "testing"

    "piko.sh/piko"
    "github.com/stretchr/testify/assert"
    customers "myapp/dist/pages/pages_customers_abc123"
    "myapp/actions"
)

// Mock repository
type MockCustomerRepo struct {
    Customers     []Customer
    CreateCalled  bool
    ShouldError   bool
}

func (m *MockCustomerRepo) GetAll() []Customer {
    return m.Customers
}

func (m *MockCustomerRepo) Create(name, email string) (*Customer, error) {
    m.CreateCalled = true
    if m.ShouldError {
        return nil, errors.New("database error")
    }
    return &Customer{ID: "123", Name: name, Email: email}, nil
}

// Test component with multiple scenarios
func TestCustomersPage(t *testing.T) {
    tests := []struct {
        name          string
        mockCustomers []Customer
        sortParam     string
        expectedCount int
        expectedTitle string
    }{
        {
            name:          "empty state",
            mockCustomers: []Customer{},
            sortParam:     "asc",
            expectedCount: 0,
            expectedTitle: "No Customers",
        },
        {
            name: "with customers sorted descending",
            mockCustomers: []Customer{
                {ID: "1", Name: "Acme"},
                {ID: "2", Name: "Beta"},
            },
            sortParam:     "desc",
            expectedCount: 2,
            expectedTitle: "Customers",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mock
            mockRepo := &MockCustomerRepo{Customers: tt.mockCustomers}
            ctx := context.WithValue(context.Background(), "repo", mockRepo)

            // Build request
            req := piko.NewTestRequest("GET", "/customers").
                WithContext(ctx).
                WithQueryParam("sort", tt.sortParam).
                Build()

            // Render component
            tester := piko.NewComponentTester(t, customers.BuildAST,
                piko.WithPageID("pages/customers"),
            )
            view := tester.Render(req, piko.NoProps{})

            // Assert DOM
            view.QueryAST(".customer-row").Count(tt.expectedCount)
            view.QueryAST("h1").HasText(tt.expectedTitle)

            // Assert metadata
            view.AssertTitle(tt.expectedTitle + " - MyApp")
            view.AssertStatusCode(200)
        })
    }
}

// Test server action
func TestCustomerCreateAction(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        mockRepo := &MockCustomerRepo{}
        ctx := context.WithValue(context.Background(), "repo", mockRepo)

        tester := piko.NewActionTester(t, &actions.CustomerCreate{})
        req := piko.NewTestRequest("POST", "/").
            WithContext(ctx).
            WithFormData("name", "Acme Corp").
            WithFormData("email", "contact@acme.com").
            Build()

        resp := tester.Invoke(req)

        tester.AssertSuccess(resp)
        tester.AssertHasHelper(resp, "redirect")
        tester.AssertHelperArg(resp, "redirect", 0, "/customers/123")
        assert.True(t, mockRepo.CreateCalled)
    })

    t.Run("validation error", func(t *testing.T) {
        mockRepo := &MockCustomerRepo{}
        ctx := context.WithValue(context.Background(), "repo", mockRepo)

        tester := piko.NewActionTester(t, &actions.CustomerCreate{})
        req := piko.NewTestRequest("POST", "/").
            WithContext(ctx).
            WithFormData("name", "").  // Empty name
            Build()

        resp := tester.Invoke(req)

        tester.AssertError(resp)
        tester.AssertStatusCode(resp, 422)
        tester.AssertFieldError(resp, "name", "Name is required")
        assert.False(t, mockRepo.CreateCalled)
    })

    t.Run("database error", func(t *testing.T) {
        mockRepo := &MockCustomerRepo{ShouldError: true}
        ctx := context.WithValue(context.Background(), "repo", mockRepo)

        tester := piko.NewActionTester(t, &actions.CustomerCreate{})
        req := piko.NewTestRequest("POST", "/").
            WithContext(ctx).
            WithFormData("name", "Acme Corp").
            WithFormData("email", "contact@acme.com").
            Build()

        resp := tester.Invoke(req)

        tester.AssertError(resp)
        tester.AssertStatusCode(resp, 500)
    })
}

// Benchmark
func BenchmarkCustomersPage(b *testing.B) {
    mockRepo := &MockCustomerRepo{
        Customers: generateMockCustomers(100),
    }
    ctx := context.WithValue(context.Background(), "db", mockRepo)

    tester := piko.NewComponentTester(b, customers.BuildAST)
    req := piko.NewTestRequest("GET", "/customers").
        WithContext(ctx).
        Build()

    tester.Benchmark(req, piko.NoProps{})
}

func generateMockCustomers(count int) []Customer {
    customers := make([]Customer, count)
    for i := range count {
        customers[i] = Customer{
            ID:   fmt.Sprintf("%d", i+1),
            Name: fmt.Sprintf("Customer %d", i+1),
        }
    }
    return customers
}
```
## Running tests

```bash
# Run all tests
go test ./...

# Run specific package
go test ./pages/...

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./...
go test -bench=. -benchmem ./...

# Update snapshots
PIKO_UPDATE_SNAPSHOTS=1 go test ./...

# Verbose output
go test -v ./...

# Run specific test
go test -run TestCustomersPage ./pages/...
go test -run TestCustomersPage/empty_state ./pages/...

# Race detection
go test -race ./...
```

## Next steps

- [Server actions](/docs/guide/server-actions) → Learn about testing server actions
- [Directives](/docs/guide/directives) → Understand conditional rendering for testing
- [Metadata & SEO](/docs/guide/metadata) → Test metadata and SEO tags
