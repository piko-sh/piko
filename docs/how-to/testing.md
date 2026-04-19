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
    customers "myapp/dist/pages/pages_customers_abc123"
)

func TestCustomersPage(t *testing.T) {
    mockRepo := &MockCustomerRepo{Customers: []Customer{{Name: "Acme"}}}
    ctx := context.WithValue(context.Background(), "repo", mockRepo)

    request := piko.NewTestRequest("GET", "/customers").Build(ctx)

    tester := piko.NewComponentTester(t, customers.BuildAST)
    view := tester.Render(request, piko.NoProps{})

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

`piko.NewTestRequest(method, path)` returns a fluent builder. The terminating `Build` call takes the context, which is the channel for cancellation, deadlines, and dependency injection:

```go
request := piko.NewTestRequest("GET", "/customers").
    WithQueryParam("sort", "desc").
    WithPathParam("id", "123").
    WithLocale("fr").
    Build(ctx)
```

Every `With*` method appears in [testing API reference](../reference/testing-api.md#request-builder).

## Inject dependencies through context

The cleanest way to swap a repository, clock, or service for a mock is via the context passed to `Build`:

```go
mockRepo := &MockCustomerRepo{Customers: []Customer{{Name: "Acme"}}}
ctx := context.WithValue(context.Background(), "repo", mockRepo)

request := piko.NewTestRequest("GET", "/").Build(ctx)
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

`piko.NewActionTester(t, entry)` drives an action through an `ActionHandlerEntry`, the descriptor that pairs a constructor with its invoke function. Construct one in the test using the action struct from your own package and a small invoke shim, then call `Invoke(ctx, arguments)` with the parameter map. Assertions live on the returned `*ActionResultView`.

```go
package actions_test

import (
    "context"
    "testing"

    "piko.sh/piko"
    "myapp/actions/customer"
)

func TestCustomerCreateAction_Success(t *testing.T) {
    mockRepo := &MockCustomerRepo{}
    ctx := context.WithValue(context.Background(), "repo", mockRepo)

    entry := piko.ActionHandlerEntry{
        Name:   "customer.Create",
        Method: "POST",
        Create: func() any { return &customer.CreateAction{} },
        Invoke: func(ctx context.Context, action any, arguments map[string]any) (any, error) {
            return action.(*customer.CreateAction).Run(ctx, arguments)
        },
    }

    tester := piko.NewActionTester(t, entry)
    result := tester.Invoke(ctx, map[string]any{
        "name":  "Acme Corp",
        "email": "contact@acme.com",
    })

    result.AssertSuccess()
    result.AssertHelper("redirect")
    assert.True(t, mockRepo.CreateCalled)
}
```

The `Invoke` shim adapts the action's typed input to the harness's `map[string]any`. In production the generated registry does this conversion for you. For unit tests it is usually simpler to call the action's exported method directly than to depend on generated code.

`tester.Invoke` accepts `nil` for actions with no parameters. The harness substitutes an empty map. To assert error cases:

```go
result := tester.Invoke(ctx, map[string]any{"email": "bad"})

result.AssertError()
result.AssertErrorContains("invalid email")
```

To assert that the action registered no helpers, use `result.AssertNoHelpers()`. For raw access to the response data, call `result.Data()` or `result.Err()`.

The action assertions are: `AssertSuccess`, `AssertError`, `AssertErrorContains(substr)`, `AssertHelper(name)`, `AssertNoHelpers`. See [testing API reference](../reference/testing-api.md#action-tester) for accessor methods.

## Snapshot unexpected regressions

`MatchSnapshot(name)` writes a golden file on first run and compares on later runs:

```go
view := tester.Render(request, piko.NoProps{})
view.MatchSnapshot("customers-page")
```

Snapshots live in `__snapshots__/<test-directory>/<name>.golden.html`. Regenerate after intentional changes:

```bash
PIKO_UPDATE_SNAPSHOTS=1 go test ./...
```

## Benchmark a page

`tester.Benchmark(req, props)` runs the render in a benchmark loop:

```go
func BenchmarkCustomersPage(b *testing.B) {
    mockRepo := &MockCustomerRepo{Customers: generateMockCustomers(100)}
    ctx := context.WithValue(context.Background(), "repo", mockRepo)

    request := piko.NewTestRequest("GET", "/customers").Build(ctx)
    tester := piko.NewComponentTester(b, customers.BuildAST)

    tester.Benchmark(request, piko.NoProps{})
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
        request := piko.NewTestRequest("GET", "/").Build(ctx)

        view := piko.NewComponentTester(t, customers.BuildAST).Render(request, piko.NoProps{})
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
