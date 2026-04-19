---
title: How to test pages, components, and actions
description: Write unit tests and benchmarks for .pk pages, PKC components, and server actions with the pikotest harness.
nav:
  sidebar:
    section: "how-to"
    subsection: "testing"
    order: 140
---

# How to test pages, components, and actions

This guide shows how to scaffold unit tests against compiled `.pk` pages and server actions. The full API surface (every builder method, assertion, and option) lives in [testing API reference](../reference/testing-api.md). For end-to-end browser tests see [browser testing harness reference](../reference/browser-testing.md).

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
    mockDB := &MockDB{Customers: []Customer{{Name: "Acme"}}}
    ctx := context.WithValue(context.Background(), "db", mockDB)

    req := piko.NewTestRequest("GET", "/customers").
        WithContext(ctx).
        Build()

    tester := piko.NewComponentTester(t, customers.BuildAST)
    view := tester.Render(req, piko.NoProps{})

    view.QueryAST(".customer-row").Count(1)
    view.QueryAST("h1").HasText("Customers")
    view.AssertTitle("Customers - MyApp")
}
```

## Lay out test files

Place test files next to the compiled component package in the `dist/` output directory:

```
dist/
  pages/
    pages_customers_abc123/
      component.go       // generated component
      component_test.go  // your test
```

Alternatively, put tests wherever fits the project, as long as the test imports the generated component package.

## Build a test request

`piko.NewTestRequest(method, path)` returns a fluent builder. The common pattern:

```go
req := piko.NewTestRequest("GET", "/customers").
    WithContext(ctx).
    WithQueryParam("sort", "desc").
    WithPathParam("id", "123").
    WithLocale("fr").
    Build()
```

Every `With*` method appears in [testing API reference](../reference/testing-api.md#request-builder).

## Inject dependencies through context

The cleanest way to swap a repository, clock, or service for a mock is `WithContext`:

```go
mockRepo := &MockCustomerRepo{Customers: []Customer{{Name: "Acme"}}}
ctx := context.WithValue(context.Background(), "repo", mockRepo)

req := piko.NewTestRequest("GET", "/").WithContext(ctx).Build()
```

`Render` reads the mock from the context:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    repo := r.Context().Value("repo").(CustomerRepository)
    customers := repo.GetAll()
    // ...
}
```

A minimal mock records call flags so the test can assert the code path ran:

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
```

## Assert against the template AST

`view.QueryAST(selector)` runs a CSS selector against the rendered template. AST queries are faster than rendering HTML and do not require a configured `RenderService`. The query result chains assertion methods such as `Count`, `HasText`, `HasAttribute`, and `HasClass`. The full list is in [testing API reference](../reference/testing-api.md#ast-queries).

```go
view.QueryAST("h1").HasText("Customers")
view.QueryAST(".customer-row").Count(10)
view.QueryAST("input[name='email']").HasAttribute("type", "email")
view.QueryAST(":not(.hidden)").MinCount(1)
```

CSS selector support includes combinators, attribute selectors, and pseudo-classes (`:nth-child`, `:not`, `:first-of-type`).

For targeted work on a subset, narrow with `First()`, `At(i)`, or `Filter`:

```go
active := view.QueryAST(".customer-row").Filter(func(n *ast_domain.TemplateNode) bool {
    status, _ := n.GetAttribute("data-status")
    return status == "active"
})
active.Count(5)
```

## Assert metadata

`piko.Metadata` (title, description, status code, redirects, Open Graph) surfaces through dedicated `Assert*` methods on the view:

```go
view.AssertTitle("Customers - MyApp")
view.AssertStatusCode(200)
view.AssertCanonicalURL("https://example.com/customers")
view.AssertHasOGTag("og:title", "Customers")
```

The [testing API reference](../reference/testing-api.md#metadata-assertions) lists every assertion. For the underlying field definitions see [metadata fields reference](../reference/metadata-fields.md).

## Render HTML when you need it

AST queries cover most cases. When a test genuinely needs the full rendered HTML (snapshot testing, regex checks on rendered attributes), call `HTML()`:

```go
html := view.HTMLString()
assert.Contains(t, html, "<h1>Customers</h1>")
```

HTML rendering requires a configured `RenderService`. See [testing API reference](../reference/testing-api.md#html-rendering).

## Test server actions

`piko.NewActionTester(t, &actions.CustomerCreate{})` drives an action and returns its typed response. The test builds a request with form data, invokes the action, and asserts on success, helpers, or field-level errors.

```go
func TestCustomerCreateAction_Success(t *testing.T) {
    mockRepo := &MockCustomerRepo{}
    ctx := context.WithValue(context.Background(), "repo", mockRepo)

    tester := piko.NewActionTester(t, &actions.CustomerCreate{})

    req := piko.NewTestRequest("POST", "/actions/customer_create").
        WithContext(ctx).
        WithFormData("name", "Acme Corp").
        WithFormData("email", "contact@acme.com").
        Build()

    resp := tester.Invoke(req)

    tester.AssertSuccess(resp)
    tester.AssertHasHelper(resp, "redirect")
    tester.AssertHelperArg(resp, "redirect", 0, "/customers/123")
    assert.True(t, mockRepo.CreateCalled)
}
```

For field-level errors, use `AssertFieldError`:

```go
tester.AssertError(resp)
tester.AssertStatusCode(resp, 422)
tester.AssertFieldError(resp, "name", "Name is required")
```

For errors the action returns (not validation), use `InvokeExpectError`:

```go
err := tester.InvokeExpectError(req)
assert.Contains(t, err.Error(), "repository not found")
```

See [testing API reference](../reference/testing-api.md#action-tester) for every action assertion.

## Snapshot unexpected regressions

`MatchSnapshot(name)` writes a golden file on first run and compares on later runs:

```go
view := tester.Render(req, piko.NoProps{})
view.MatchSnapshot("customers-page")
```

Snapshots live in `__snapshots__/<test-directory>/<name>.golden.html`. Regenerate after intentional changes:

```bash
PIKO_UPDATE_SNAPSHOTS=1 go test ./...
```

## Benchmark a page

`tester.Benchmark(req, props)` runs the render in a benchmark loop. It calls `b.ResetTimer()` internally:

```go
func BenchmarkCustomersPage(b *testing.B) {
    mockDB := &MockDB{Customers: generateMockCustomers(100)}
    ctx := context.WithValue(context.Background(), "db", mockDB)

    req := piko.NewTestRequest("GET", "/customers").WithContext(ctx).Build()
    tester := piko.NewComponentTester(b, customers.BuildAST)

    tester.Benchmark(req, piko.NoProps{})
}
```

Run with:

```bash
go test -bench=. -benchmem ./pages/...
```

## Table-driven tests

The component and action testers compose naturally with `t.Run` sub-tests:

```go
tests := []struct {
    name     string
    mockData []Customer
    want     int
}{
    {"empty", []Customer{}, 0},
    {"two customers", []Customer{{Name: "Acme"}, {Name: "Beta"}}, 2},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        ctx := context.WithValue(context.Background(), "repo",
            &MockCustomerRepo{Customers: tt.mockData})
        req := piko.NewTestRequest("GET", "/").WithContext(ctx).Build()

        view := piko.NewComponentTester(t, customers.BuildAST).Render(req, piko.NoProps{})
        view.QueryAST(".customer-row").Count(tt.want)
    })
}
```

## Running tests

```bash
go test ./...                              # everything
go test -v ./pages/...                     # one package, verbose
go test -run TestCustomersPage ./pages/... # one test
go test -cover ./...                       # coverage
go test -bench=. -benchmem ./...           # benchmarks
go test -race ./...                        # race detection
PIKO_UPDATE_SNAPSHOTS=1 go test ./...      # refresh snapshots
```

## See also

- [testing API reference](../reference/testing-api.md) for every builder method, assertion, and option.
- [browser testing harness reference](../reference/browser-testing.md) for end-to-end browser tests.
- [server actions reference](../reference/server-actions.md) for the action surface under test.
- [metadata fields reference](../reference/metadata-fields.md) for the fields metadata assertions inspect.
