---
title: How to write browser tests
description: Set up the harness, drive the page through the fluent API, use device emulation, and lock in regressions with goldens and spec files.
nav:
  sidebar:
    section: "how-to"
    subsection: "testing"
    order: 150
---

# How to write browser tests

This guide walks through end-to-end browser tests with `wdk/browser`. Tests drive a real Chromedp browser against a built Piko binary, so they exercise the full request path (server, actions, client components). For the full method surface see [browser testing harness reference](../reference/browser-testing.md). For the comparison with pikotest AST tests see [about browser testing](../explanation/about-browser-testing.md).

## Set up the harness in TestMain

The harness compiles the project once and shares the browser across every test in the package. Create `TestMain`:

```go
package browsertests

import (
    "os"
    "testing"

    "piko.sh/piko/wdk/browser"
)

func TestMain(m *testing.M) {
    harness := browser.NewHarness(
        browser.WithProjectDir("../.."),
        browser.WithOutputDir("./testdata/output"),
        browser.WithBuildTimeout(2 * time.Minute),
    )
    if err := harness.Setup(); err != nil {
        harness.Cleanup()
        os.Exit(1)
    }
    code := m.Run()
    harness.Cleanup()
    os.Exit(code)
}
```

`Setup` builds the binary, starts the server on a free port, and boots headless Chrome. `Cleanup` shuts everything down.

## Write a first test

```go
func TestLoginFlow(t *testing.T) {
    page := browser.New(t)

    page.Navigate("/login")
    page.Fill("[name=email]", "alice@example.com")
    page.Fill("[name=password]", "hunter2")
    page.Click("button[type=submit]")
    page.WaitForText("h1", "Welcome, Alice")

    page.Assert("h1").Text("Welcome, Alice")
    page.AssertNoConsoleErrors()
}
```

`browser.New(t)` returns a page bound to the test. Failures call `t.Fatalf` so the first failed assertion stops the test.

## Wait for the page to settle

Piko pages typically load fast. Interactive flows with client components sometimes need explicit synchronisation:

```go
page.WaitForVisible(".product-card")   // element appears in the DOM
page.WaitForText("h1", "Products")     // text content matches
page.WaitForNetworkIdle()              // no in-flight requests
page.WaitStable(".live-counter")       // text stops changing
```

Each wait defaults to 5 seconds. Override with `browser.WithTimeout`:

```go
page.Wait(".slow-load", browser.WithTimeout(30 * time.Second))
```

## Test a form submission

```go
func TestContactForm(t *testing.T) {
    page := browser.New(t)

    page.Navigate("/contact")
    page.Fill("[name=name]", "Alice Smith")
    page.Fill("[name=email]", "alice@example.com")
    page.Fill("textarea[name=message]", "Hello from a test")
    page.Click("button[type=submit]")

    page.WaitForText(".toast", "Message sent")
    page.Assert(".toast").HasClass("success")
}
```

## Emulate a device

Viewport and user-agent presets ship for common devices:

```go
func TestMobileLayout(t *testing.T) {
    page := browser.New(t)
    page.EmulateIPhone14()

    page.Navigate("/")
    page.Assert(".mobile-menu").IsVisible()
    page.Assert(".desktop-nav").IsVisible().Count(0)
}
```

Presets include `EmulateIPhone13`, `EmulateIPhone14Pro`, `EmulateIPad`, `EmulateIPadPro`, `EmulateGalaxyS9`, `EmulateDesktop4K`, `EmulateDesktopHD`. `ResetEmulation()` restores the default viewport.

## Intercept network requests

```go
page.InterceptRequest("/api/inventory", func(req *browser.InterceptedRequest) {
    req.RespondJSON(200, map[string]any{"stock": 0})
})
page.Navigate("/products/42")
page.Assert(".stock-label").Text("Out of stock")
```

Intercepts isolate front-end behaviour from a downstream API during tests.

## Drive Piko-specific behaviour

For pages that use the event bus or partial refresh, the page exposes helpers that reach into the framework's client runtime:

```go
page.PikoSetupEventLog()
page.Click(".chat-input [type=submit]")
page.PikoBusWaitForEvent("chat:message-sent", 5 * time.Second)

messages := page.PikoGetEventLog()
if len(messages) != 1 {
    t.Fatalf("expected one bus event, got %d", len(messages))
}
```

Partial refresh:

```go
page.TriggerPartialReload(".product-list")
page.PikoWaitForPartialReload(".product-list", 5 * time.Second)
```

## Lock visual regressions with goldens

`MatchGolden(name)` stores a screenshot on first run and compares on later runs:

```go
func TestDashboardLook(t *testing.T) {
    page := browser.New(t)
    page.Navigate("/dashboard")
    page.WaitForNetworkIdle()
    page.MatchGolden("dashboard")
}
```

Golden files live under `testdata/goldens/`. After an intentional visual change, regenerate:

```bash
go test ./... -update-goldens
```

Or programmatically inside a spec:

```go
browser.RunSpec(t, "testdata/dashboard.spec.json",
    browser.WithUpdateGoldens(true))
```

## Write a declarative spec

For flows with no test-specific Go logic, specs are faster to write and review. Create `testdata/login.spec.json`:

```json
[
  { "action": "navigate", "target": "/login" },
  { "action": "fill", "selector": "[name=email]", "value": "alice@example.com" },
  { "action": "fill", "selector": "[name=password]", "value": "hunter2" },
  { "action": "click", "selector": "button[type=submit]" },
  { "action": "wait_for_text", "selector": "h1", "value": "Welcome" },
  { "action": "assert_text", "selector": "h1", "value": "Welcome" },
  { "action": "match_golden", "name": "login-success" }
]
```

Run it:

```go
func TestLoginSpec(t *testing.T) {
    browser.RunSpec(t, "testdata/login.spec.json")
}
```

The spec vocabulary mirrors the `Page` API. See [browser testing harness reference](../reference/browser-testing.md) for the full action list.

## Debug a failing test interactively

Pass `-interactive` and the harness pauses at each step with a TUI:

```bash
go test -run TestLoginFlow -interactive
```

`-headed` opens a visible browser window without the TUI. Combine with `-headed` for debugging CSS without the step-through UI.

Inside a test, `page.Pause()` drops into the interactive UI on demand.

## Capture screenshots and PDFs

```go
page.SaveScreenshot("login-page.png")
page.SaveScreenshotViewport("login-viewport.png")
page.SavePDF("login.pdf")
```

Outputs go under the harness's `WithOutputDir` directory.

## Troubleshoot

- Build failures. The harness runs `go build` of the project. Inspect the harness output or set `WithSkipBuild(true)` to reuse an existing binary.
- Port conflicts. `WithPort(0)` picks a free port. Use an explicit port only when the test depends on a specific URL.
- Flaky waits. Prefer `WaitForVisible`, `WaitForText`, and `WaitStable` over `time.Sleep`. Waits poll up to their timeout and return as soon as the condition holds.
- Missing Chrome. Chromedp shells out to a local Chrome binary. Install Chrome or Chromium on the machine running tests.

## See also

- [Browser testing harness reference](../reference/browser-testing.md) for the complete Page, Harness, and Spec API.
- [About browser testing](../explanation/about-browser-testing.md) for when to reach for browser tests versus pikotest unit tests.
- [Testing API reference](../reference/testing-api.md) for the AST-level pikotest surface.
- [How to testing](testing.md) for pikotest recipes.
