# `/e2e`: End-to-end browser tests

This directory contains end-to-end (E2E) tests that run in a real browser to verify your application works correctly from a user's perspective. You'll find two files already scaffolded: `e2e_test.go` (the test harness) and `homepage_test.go` (tests for the welcome page).

These tests use the Piko browser testing package, which provides a fluent API for browser automation.

---

## Running tests

E2E tests are gated behind a build tag to prevent them from running during normal `go test` runs. To run them:

```bash
# Run all E2E tests (headless)
go test -tags=e2e ./e2e/...

# Run with verbose output
go test -tags=e2e -v ./e2e/...

# Run a specific test
go test -tags=e2e -v -run TestHomepage_DefaultTitle ./e2e/...
```

---

## Test structure

Your E2E tests use a harness that automatically starts the Piko server before tests run and shuts it down afterwards.

- **`e2e_test.go`**: contains `TestMain` which sets up the test harness (starts the server, launches browser).
- **`*_test.go`**: individual test files grouped by feature.

### Test harness setup

The harness is configured in `TestMain`:

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

### Writing tests

Individual tests create a `browser.New(t)` page, navigate, and make assertions:

```go
//go:build e2e

package e2e

import (
    "testing"
    "piko.sh/piko/wdk/browser"
)

func TestMyFeature(t *testing.T) {
    p := browser.New(t)
    defer p.Close()

    p.Navigate("/my-page")
    p.WaitFor("h1")

    p.Assert("h1").HasText("Welcome")
    p.Assert(".success-message").Exists()
    p.Assert("#result").ContainsText("success")
}
```

---

## Available assertions

```go
p.Assert(selector).Exists()           // Element exists in DOM
p.Assert(selector).NotExists()        // Element does not exist
p.Assert(selector).HasText("exact")   // Exact text match
p.Assert(selector).ContainsText("x")  // Text contains substring
p.Assert(selector).HasAttribute("href", "/path")
p.Assert(selector).HasClass("active")
p.Assert(selector).IsVisible()
p.Assert(selector).Count(4)           // Number of matching elements
p.AssertNoConsoleErrors()             // No JavaScript errors in console
```

---

## Available actions

```go
p.Navigate("/path")           // Navigate to URL
p.Click("#button")            // Click an element
p.Fill("#input", "value")     // Fill a text input
p.Focus("#input")             // Focus an element
p.Blur("#input")              // Blur (unfocus) an element
p.WaitFor("#element")         // Wait for element to exist
p.WaitForText("#el", "text")  // Wait for text content
p.Wait(time.Second)           // Wait for duration
```

---

### To learn more

Refer to the official Piko documentation:

-   **[Testing guide](https://piko.sh/docs/how-to/testing)**
