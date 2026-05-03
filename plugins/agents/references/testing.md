# Testing

Use this guide when writing tests for Piko components, server actions, E2E browser tests, or benchmarks.

## Overview

Piko provides testing utilities through the `piko.sh/piko` and `piko.sh/piko/wdk/browser` packages. Tests use standard Go testing with three approaches:

- **Component tests** (`piko.NewComponentTester`) - render `.pk` components at the AST level (fast, no browser)
- **Action tests** (`piko.NewActionTester`) - invoke server actions directly
- **E2E browser tests** (`browser.New(t)`) - drive a real browser with ChromeDP (full integration)

New projects scaffolded with `piko new` include example tests for both component testing (`pages/index_test.go`) and E2E browser testing (`e2e/`).

## Building test requests

Use the fluent `NewTestRequest` builder:

```go
req := piko.NewTestRequest("GET", "/customers").
    WithQueryParam("sort", "desc").
    WithPathParam("id", "123").
    WithFormData("name", "Test").
    WithLocale("fr").
    WithHost("example.com").
    Build(ctx)
```

| Method | Purpose |
|--------|---------|
| `WithQueryParam(key, value)` | Add URL query parameter |
| `WithPathParam(key, value)` | Add route path parameter |
| `WithFormData(key, value)` | Add form field |
| `WithLocale(locale)` | Set request locale (default: "en") |
| `WithHost(host)` | Set host header |
| `WithGlobalTranslations(t)` | Set global translations |
| `WithLocalTranslations(t)` | Set component translations |
| `WithCollectionData(data)` | Set collection data |
| `Build(ctx)` | Build `*RequestData` (context required for dependency injection) |

## Mocking dependencies

Pass mocks via context (recommended):

```go
mockRepo := &MockCustomerRepo{Customers: []Customer{{Name: "Acme"}}}
ctx := context.WithValue(context.Background(), "repo", mockRepo)
req := piko.NewTestRequest("GET", "/").Build(ctx)
```

## Component testing

Import the compiled component and create a tester:

```go
import customers "myapp/dist/pages/pages_customers_abc123"

tester := piko.NewComponentTester(t, customers.BuildAST)
view := tester.Render(req, piko.NoProps{})
```

### AST queries (recommended - fast, no RenderService needed)

```go
// Existence
view.QueryAST("h1").Exists()
view.QueryAST(".missing").NotExists()

// Count
view.QueryAST(".customer-row").Count(10)

// Text
view.QueryAST("h1").HasText("Customers")
view.QueryAST("p").ContainsText("Welcome")

// Attributes
view.QueryAST("input[name='email']").HasAttribute("type", "email")
view.QueryAST("button").HasAttributePresent("disabled")
view.QueryAST("div").HasClass("container")

// Indexing and iteration
view.QueryAST(".stat-value").Index(0).HasText("100")
view.QueryAST(".row").Each(func(i int, node *ast_domain.TemplateNode) {
    id, ok := node.GetAttribute("data-id")
    assert.True(t, ok)
})
```

All standard CSS selectors work: tag, class, ID, combinators, attribute selectors, pseudo-classes.

### Metadata assertions

```go
view.AssertTitle("Customers - MyApp")
view.AssertDescription("Page description")
view.AssertStatusCode(200)
view.AssertClientRedirect("/login")
```

### HTML rendering (only when needed)

```go
html := view.HTMLString()
assert.Contains(t, html, "<h1>Customers</h1>")
```

Requires a `RenderService`. AST queries are significantly faster.

## Server action testing

```go
import (
    "context"
    "net/http"

    "myapp/actions"
    "piko.sh/piko"
)

ctx := context.Background()
entry := piko.ActionHandlerEntry{
    Name:   "CustomerCreate",
    Method: http.MethodPost,
    Create: func() any { return &actions.CustomerCreateAction{} },
    Invoke: func(ctx context.Context, action any, arguments map[string]any) (any, error) {
        return action.(*actions.CustomerCreateAction).Call(
            arguments["name"].(string),
            arguments["email"].(string),
        )
    },
}
tester := piko.NewActionTester(t, entry)

result := tester.Invoke(ctx, map[string]any{
    "name":  "Acme",
    "email": "contact@acme.com",
})

// Assertions (called on result, not tester)
result.AssertSuccess()
result.AssertHelper("redirect")
result.AssertError()
```

Note: `NewActionTester` takes an `ActionHandlerEntry` descriptor, not an action struct pointer. The generator emits one entry per action; in production tests prefer constructing the entry inline as above. `Invoke` requires `ctx` first. Assertions are methods on the result object, not the tester.

## Snapshot testing

```go
view := tester.Render(req, piko.NoProps{})
view.MatchSnapshot("customers-page")
```

Snapshots stored in `__snapshots__/<test-directory>/<name>.golden.html`. Update with:

```bash
PIKO_UPDATE_SNAPSHOTS=1 go test ./...
```

## Benchmarks

```go
func BenchmarkCustomersPage(b *testing.B) {
    ctx := context.Background()
    tester := piko.NewComponentTester(b, customers.BuildAST)
    req := piko.NewTestRequest("GET", "/customers").Build(ctx)
    tester.Benchmark(req, piko.NoProps{})
}
```

Run: `go test -bench=. -benchmem ./pages/...`

## Table-driven tests

```go
tests := []struct {
    name          string
    mockData      []Customer
    expectedCount int
}{
    {"empty", []Customer{}, 0},
    {"with data", []Customer{{Name: "Acme"}}, 1},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // ... setup and assert ...
    })
}
```

## E2E browser testing

E2E tests drive a real headless Chromium browser via ChromeDP. They use the `e2e` build tag and live in an `e2e/` directory at the project root.

### Harness setup (TestMain)

Every E2E test package needs a `TestMain` that creates a harness. The harness builds the project, starts the server, and launches the browser:

```go
//go:build e2e

package e2e

import (
    "fmt"
    "os"
    "testing"

    "piko.sh/piko/wdk/browser"
)

func TestMain(m *testing.M) {
    harness := browser.NewHarness(
        browser.WithProjectDir(".."),
        browser.WithServerCommand("go", "run", "./cmd/main"),
    )

    if err := harness.Setup(); err != nil {
        fmt.Fprintf(os.Stderr, "E2E setup failed: %v\n", err)
        os.Exit(1)
    }

    code := m.Run()
    harness.Cleanup()
    os.Exit(code)
}
```

Harness options:

| Option | Purpose |
|--------|---------|
| `WithProjectDir(dir)` | Project root directory |
| `WithServerCommand(args...)` | Custom server command |
| `WithServerArgs(args...)` | Additional server arguments |
| `WithPort(port)` | TCP port (0 = auto-select) |
| `WithHeadless(bool)` | Headless mode (default: true) |
| `WithSkipBuild(bool)` | Skip build for faster reruns |
| `WithEnv(key, value)` | Set environment variables |
| `WithBuildTimeout(d)` | Build timeout (default: 5 min) |

### Writing E2E tests

Each test creates an isolated browser page with `browser.New(t)`:

```go
//go:build e2e

package e2e

import (
    "testing"
    "piko.sh/piko/wdk/browser"
)

func TestHomepage_Title(t *testing.T) {
    p := browser.New(t)
    defer p.Close()

    p.Navigate("/")
    p.WaitFor(".hero-title")

    p.Assert(".hero-title").HasText("Welcome to Piko")
}

func TestLogin_Flow(t *testing.T) {
    p := browser.New(t)
    defer p.Close()

    p.Navigate("/login").
        Fill("#email", "user@example.com").
        Fill("#password", "secret").
        Click("button[type=submit]").
        WaitFor(".dashboard")

    p.Assert(".welcome-msg").ContainsText("user@example.com")
}
```

### Navigation

- `Navigate(path)` - go to URL path
- `Reload()` - refresh page
- `Back()` / `Forward()` - browser history
- `Title()` - get page title
- `URL()` - get current URL

### Interactions (all chainable)

- `Click(selector)` / `DoubleClick(selector)` / `RightClick(selector)`
- `Fill(selector, value)` - type into input (clears first)
- `Clear(selector)` - clear input
- `Type(text)` - type character-by-character (no clear)
- `Press(keys...)` - press keys: `"Enter"`, `"Tab"`, `"Shift+Enter"`, `"Control+b"`
- `Check(selector)` / `Uncheck(selector)` - checkboxes
- `Submit(selector)` - submit form
- `Hover(selector)` / `Focus(selector)` / `Blur(selector)`
- `SetFiles(selector, ...paths)` - file inputs
- `DragAndDrop(source, target)`
- `Scroll(selector, position)`

### Waiting

- `WaitFor(selector)` - wait for element to appear
- `WaitForText(selector, text)` - wait for text content
- `WaitForVisible(selector)` / `WaitForNotVisible(selector)`
- `WaitForEnabled(selector)` / `WaitForDisabled(selector)`
- `WaitForNotPresent(selector)` - wait for removal from DOM
- `WaitForNetworkIdle(duration)` - wait for network quiet
- `WaitStable()` - wait until DOM stops changing
- `Wait(duration)` - explicit pause

`WaitForVisible`, `WaitForNotVisible`, `WaitForEnabled`, `WaitForDisabled`, `WaitForNotPresent`, and `WaitForNetworkIdle` support `WithTimeout(duration)` (default: 5 seconds).

### Assertions

Access via `p.Assert(selector)`:

```go
// Existence and count
p.Assert(".card").Exists()
p.Assert(".missing").NotExists()
p.Assert(".item").Count(5)

// Text (polls up to 5s)
p.Assert("h1").HasText("Exact match")
p.Assert("p").ContainsText("substring")

// Attributes and state
p.Assert("input").HasAttribute("type", "email")
p.Assert("div").HasClass("active")
p.Assert("input").HasValue("hello")
p.Assert("input").IsChecked()
p.Assert("button").IsDisabled()
p.Assert(".modal").IsVisible()
p.Assert(".hidden").IsHidden()

// Golden file comparison
p.Assert(".output").MatchesGolden("expected-output")
```

### Console assertions

```go
p.AssertNoConsoleErrors()
p.AssertNoConsoleWarnings()

p.AssertConsole().HasMessage("loaded").NoErrors().NoWarnings()
```

### Screenshots and output

```go
png := p.ScreenshotViewport()
p.SaveScreenshot(".card", "card.png")
html := p.CaptureDOM(".content")
```

### Shadow DOM piercing

Use `>>>` to pierce shadow DOM boundaries (for PKC components):

```go
p.Click("pp-button >>> .inner-btn")
p.Assert("pp-card >>> .title").HasText("Hello")
```

### Piko-specific features

```go
// Partial reloads
p.TriggerPartialReload("my-partial", nil)
p.WaitForPartialReload("my-partial")

// Event bus
p.TriggerBusEvent("user:updated", map[string]any{"id": 123})
```

### Running E2E tests

```bash
go test -tags=e2e ./e2e/...              # Headless (default)
go test -tags=e2e -v ./e2e/...           # Verbose
go test -tags=e2e -v -run TestLogin ./e2e/...  # Specific test
go test -tags=e2e ./e2e/... -headed      # Watch browser
go test -tags=e2e ./e2e/... -interactive  # Step-through TUI
```

Update E2E golden files (separate from component snapshots which use `PIKO_UPDATE_SNAPSHOTS`):

```bash
PIKO_UPDATE_GOLDEN=1 go test -tags=e2e ./e2e/...
```

## Running tests

```bash
go test ./...                          # All unit tests
go test -v -run TestCustomers ./...    # Specific test
go test -cover ./...                   # With coverage
go test -race ./...                    # Race detection
go test -bench=. ./...                 # Benchmarks
go test -tags=e2e ./e2e/...            # E2E browser tests
```

## LLM mistake checklist

- Using HTML rendering when AST queries would suffice (slower, needs RenderService)
- Forgetting to import the compiled component package (not the source `.pk` file)
- Sharing mutable mock state across parallel subtests
- Testing too many things in one test (unclear failure messages)
- Forgetting `Build(ctx)` on the test request builder (note: `Build` requires a `context.Context` argument)
- Calling a non-existent `WithContext(ctx)` method on the request builder; pass `ctx` to `Build(ctx)` instead
- Passing an action struct pointer (e.g. `&actions.Foo{}`) to `NewActionTester`; it expects an `ActionHandlerEntry` descriptor
- Calling `tester.Invoke(args)` without a `ctx`; the real signature is `Invoke(ctx, args)`
- Using `InvokeExpectError` when the action returns a validation error via response (not a Go error)
- Forgetting `//go:build e2e` tag on E2E test files (they'll run with normal tests)
- Forgetting `defer p.Close()` on browser pages (leaks browser contexts)
- Using `browser.New(t)` without a `TestMain` harness (no server running)
- Using AST queries (`QueryAST`) in E2E tests or browser assertions (`Assert`) in component tests - they are different APIs

## Related

- `references/server-actions.md` - action structure and registration
- `references/pk-file-format.md` - component structure
