---
title: Browser testing harness
description: End-to-end browser-driven tests with a fluent page API and optional declarative specs.
nav:
  sidebar:
    section: "reference"
    subsection: "testing"
    order: 240
---

# Browser testing harness

The `wdk/browser` package runs Piko projects in a real browser (Chromedp / headless Chrome) and exposes a fluent, assertion-driven API for end-to-end tests. Tests either drive the browser programmatically through `Page` or load declarative `TestSpec` JSON documents. For the design rationale and the comparison with pikotest, see [about browser testing](../explanation/about-browser-testing.md). For task recipes see [how to browser testing](../how-to/browser-testing.md). Source of truth: [`wdk/browser/`](https://github.com/piko-sh/piko/tree/master/wdk/browser).

## Harness and pages

```go
func NewHarness(opts ...HarnessOption) *Harness
func (h *Harness) Setup() error
func (h *Harness) Cleanup()
func (h *Harness) ServerURL() string
func (h *Harness) IsInteractive() bool
func (h *Harness) NewSession() (*Session, error)

func New(t testing.TB) *Page
func RunSpec(t testing.TB, specPath string, opts ...SpecOption)
```

`NewHarness` builds a reusable harness. Typically called once in `TestMain` so every test shares a single server build. `New(t)` returns a per-test `Page` bound to `t` for automatic cleanup. `NewSession` returns a `Session` for cases that prefer returned errors over `t.Fatalf`. `RunSpec` runs a declarative JSON test spec.

## Harness options

| Option | Purpose |
|---|---|
| `WithProjectDir(dir)` | Project root to compile and serve. Defaults to `.`. |
| `WithOutputDir(dir)` | Destination for screenshots, downloads, and logs. Defaults to CWD. |
| `WithPort(port)` | Explicit listen port. Zero picks a random free port. |
| `WithHeadless(bool)` | Headless vs headful. Defaults to headless unless the caller supplies `-headed` or `-interactive`. |
| `WithInteractive(bool)` | Step-through mode with TUI. Also disables headless. |
| `WithSimpleInteractive()` | Basic ANSI interactive mode (no TUI). |
| `WithBuildTimeout(d)` | Max time for the project build. Default 5 minutes. |
| `WithSkipBuild(bool)` | Reuse a pre-built binary. |
| `WithEnv(k, v)` | Set an environment variable on the server process. |
| `WithServerCommand(args...)` | Alternative server-start command. Replaces the default `go run ./cmd/server`. |
| `WithServerArgs(args...)` | Extra arguments appended to the server command. |
| `WithSandboxFactory(factory)` | `safedisk.Factory` for file-operation sandboxing. |

### Command-line flags

| Flag | Effect |
|---|---|
| `-headed` | Run the browser with a visible window. |
| `-interactive` | Enable step-through mode with TUI. |
| `-interactive-simple` | Use basic ANSI interactive mode. |
| `-update-goldens` | Regenerate golden files instead of comparing. |

## Spec options

| Option | Purpose |
|---|---|
| `WithUpdateGoldens(bool)` | Regenerate golden files. Defaults to the `-update-goldens` flag. |
| `WithSpecTimeout(d)` | Spec-execution timeout. Default 5 minutes. |

## Wait options

| Option | Purpose |
|---|---|
| `WithTimeout(d time.Duration)` | Maximum time to wait. Default 5 seconds. |

Use with any wait method that accepts `opts ...WaitOption`: `page.WaitForVisible("#thing", browser.WithTimeout(30 * time.Second))`.

## Page API

A `*Page` exposes the full browser-driving surface. The sections below group methods by role.

### Navigation

`Navigate(url)`, `Reload()`, `Back()`, `Forward()`, `Stop()`, `Title() string`, `URL() string`.

### Waiting

| Method | Purpose |
|---|---|
| `Wait(d time.Duration)` | Sleep for a fixed duration. |
| `WaitFor(selector string)` | Wait for `selector` to appear in the DOM. |
| `WaitForText(selector, text string)` | Wait until `selector` contains `text`. Hard-coded 5-second timeout; does not accept `WaitOption`. |
| `WaitStable()` | Wait for the DOM to settle (no mutations within the stability window). |
| `WaitForVisible(selector string, opts ...WaitOption)` | Wait until `selector` is visible. |
| `WaitForNotVisible(selector string, opts ...WaitOption)` | Wait until `selector` becomes hidden or leaves the DOM. |
| `WaitForEnabled(selector string, opts ...WaitOption)` | Wait until `selector` becomes enabled. |
| `WaitForDisabled(selector string, opts ...WaitOption)` | Wait until `selector` becomes disabled. |
| `WaitForNotPresent(selector string, opts ...WaitOption)` | Wait until `selector` is no longer in the DOM. |

### Interaction

| Method | Purpose |
|---|---|
| `Click(selector)`, `DoubleClick(selector)`, `RightClick(selector)` | Mouse interaction. |
| `Fill(selector, value)` | Clear and type. |
| `Clear(selector)` | Empty a field. |
| `Focus(selector)`, `Blur(selector)` | Focus management. |
| `Check(selector)`, `Uncheck(selector)` | Checkbox state. |
| `Select(selector string, start, end int)` | Set a text-selection range on `selector` (uses `SetSelection` under the hood). Not a `<select>`-option helper; pick options via `Click(...)` on the option element or `Eval(...)`. |
| `Submit(selector)` | Submit a form. |
| `Type(text string)` | Type `text` into the currently focused element. No selector argument; combine with `Focus(selector)` first. |
| `Press(keys ...string)` | Send a sequence of key chord strings (for example `Press("Enter")`, `Press("Shift+Tab")`, `Press("Control+k")`). No selector argument; operates on the focused element. |
| `PressAndHold(key string)` / `Release(key string)` | Press a key without releasing, then release later. |
| `Hover(selector)` | Mouse-over. |
| `Scroll(selector string, position int)` | Scroll the page so the named element sits at the given pixel Y position. |
| `ScrollIntoView(selector string)` | Scroll the named element into the viewport. |
| `DragAndDrop(sourceSelector, targetSelector)`, `DragAndDropHTML5(sourceSelector, targetSelector)` | Drag operations between elements. |
| `DragTo(sourceSelector string, targetX, targetY float64)` | Drag from `sourceSelector` to absolute viewport coordinates. |
| `DragByOffset(selector string, offsetX, offsetY float64)` | Drag from `selector` by a relative pixel offset. |
| `SetViewport(width, height int64)` | Resize the viewport. |

### Frames

`ClickInFrame`, `FillInFrame`, `WaitForElementInFrame`, `GetTextInFrame`, `EvalInFrame`, `GetFrameDocument`, `CountFrames`, `GetFrames`, `IsFrameLoaded`.

### Assertions

`Assert(selector) *Assertion` returns a builder. Available methods: `.Exists()`, `.NotExists()`, `.Count(n)`, `.HasText(expected)`, `.ContainsText(substring)`, `.HasHTML(expected)`, `.HasAttribute(name, expected)`, `.HasClass(className)`, `.HasStyle(property, expected)`, `.HasValue(expected)`, `.IsChecked()`, `.IsUnchecked()`, `.IsFocused()`, `.IsVisible()`, `.IsHidden()`, `.IsEnabled()`, `.IsDisabled()`, `.MatchesGolden(name)`. Chain assertions for composite checks.

Console checks: `AssertNoConsoleErrors()`, `AssertNoConsoleWarnings()`, and the richer `AssertConsole() *ConsoleAssertion` builder with `.HasMessage(contains)`, `.HasError(contains)`, `.HasWarning(contains)`, `.HasLog(contains)`, `.NoErrors()`, `.NoWarnings()`.

### Console

`ConsoleLogs()`, `ConsoleLogsWithLevel()`, `HasConsoleErrors()`, `ConsoleErrors()`, `ClearConsole()`.

### Storage

Cookies: `SetCookie`, `GetCookie`, `GetCookieValue`, `GetAllCookies`, `DeleteCookie`, `HasCookie`, `ClearCookies`.
Local storage: `SetLocalStorageItem`, `GetLocalStorageItem`, `HasLocalStorageItem`, `RemoveLocalStorageItem`, `GetAllLocalStorage`, `ClearLocalStorage`.
Session storage: `SetSessionStorageItem`, `GetSessionStorageItem`, `HasSessionStorageItem`, `RemoveSessionStorageItem`, `GetAllSessionStorage`, `ClearSessionStorage`.

### Network

| Method | Purpose |
|---|---|
| `InterceptRequest(urlPattern string, response MockResponse)` | Reply to requests matching `urlPattern` with a canned `MockResponse`. |
| `RemoveRequestIntercept(urlPattern string)`, `ClearRequestIntercepts()` | Tear down intercepts. |
| `SetBasicAuthHeader`, `SetAuthorisationHeader` | Auth headers. |
| `SetCustomHeader`, `SetExtraHTTPHeaders`, `ClearExtraHTTPHeaders` | Custom headers. |
| `SetAcceptLanguageHeader`, `SetUserAgent` | Locale and UA. |
| `GetResponseHeaders()` | Inspect the most recent response headers. |
| `CheckRequestMade(matcher RequestMatcher) bool` | Assertion helper; returns `true` if any captured request matched. |
| `WaitForRequest(matcher RequestMatcher, opts ...WaitOption)`, `WaitForNetworkIdle(idleDuration time.Duration, opts ...WaitOption)` | Request synchronisation. |

The internal `browser_provider_chromedp` package defines `RequestMatcher`, `MockResponse`, `NetworkRequest`, and `DownloadInfo`, exposing them only through the `*Page` method surface. Construct them via the helpers Piko provides through `Page` instead of instantiating the chromedp structs directly.

### Output

Screenshots: `Screenshot`, `ScreenshotFull`, `ScreenshotViewport`, `ScreenshotRegion`, `ScreenshotWithOptions`, `ScreenshotJPEG`, `ScreenshotWebP`, `ScreenshotElementWithPadding`, `SaveScreenshot`, `SaveScreenshotViewport`.

Comparison: `CompareScreenshots` diffs PNG bytes. `MatchGolden(selector, name string)` captures the rendered HTML of `selector` and diffs it against the named golden file under `testdata/golden/`. The two helpers operate on different artefacts (image bytes versus DOM HTML).

PDF: `PrintToPDF`, `PrintToPDFA4`, `PrintToPDFLandscape`, `PrintToPDFNoBackground`, `PrintToPDFWithHeaderFooter`, `PrintToPDFPageRange`, `PrintToPDFWithOptions`, `SavePDF`.

### Emulation

| Method | Purpose |
|---|---|
| `EmulateDevice(device)` | Apply a `Device` preset. |
| `EmulateViewportByName(name)` | Named viewport preset. |
| `EmulateDesktop4K`, `EmulateDesktopHD`, `EmulateMobile`, `EmulateTablet` | Common presets. |
| `EmulateIPhone13`, `EmulateIPhone14`, `EmulateIPhone14Pro` | iPhone presets. |
| `EmulateIPad`, `EmulateIPadPro` | iPad presets. |
| `EmulateGalaxyS9` | Android preset. |
| `ResetEmulation` | Clear emulation. |

### Attributes and dimensions

`GetAttribute`, `GetAttributes`, `SetAttribute`, `RemoveAttribute`, `GetDimensions`.

### JavaScript

`Eval(script)`, `EvalReturn(script)`, `EvalInFrame(frameSelector, script)`, `CaptureDOM(selector string) string` (returns the live HTML of `selector`).

### Dialogues

`HandleAlert`, `HandleConfirm`, `HandlePrompt`, `SetupDialogAutoAccept`, `SetupDialogAutoDismiss`, `AcceptDialog`, `DismissDialog` (method names match the underlying Chromedp API).

### Downloads

| Method | Purpose |
|---|---|
| `SetDownloadPath(path string)` | Set the download directory for future downloads. |
| `WaitForDownload(downloadDir string, trigger func()) *DownloadInfo` | Run `trigger`, then wait for a file to appear in `downloadDir`. |
| `TriggerDownload(url, filename string)` | Start a download from `url`, saving as `filename`. |
| `GetDownloadedFile(sandbox safedisk.Sandbox, filename string) []byte` | Read a previously downloaded file via a sandbox. |
| `CreateBlobDownload(content, mimeType, filename string)` | Create and trigger a blob download in the page (test helper). |

### Cursor and selection

`GetCursorOffset`, `SetCursor`, `PlaceCursorInElement`, `CollapseToStart`, `CollapseToEnd`, `SelectAll`, `GetSelectionRange`, `SetFiles(selector, paths...)`.

### Piko-specific helpers

For Piko projects under test, the page exposes helpers for Piko's partial refresh and event bus:

| Method | Purpose |
|---|---|
| `PikoBusEmit(eventName string, detail map[string]any)` | Emit a bus event in the browser. |
| `PikoBusWaitForEvent(eventName string, opts ...WaitOption) map[string]any` | Wait for a bus event; supply a timeout via `browser.WithTimeout`. |
| `PikoCheckBusEventReceived(eventName string) bool` | Assert an event fired. |
| `PikoGetEventLog()`, `PikoSetupEventLog()`, `PikoClearEventLog()` | Event-log plumbing. |
| `TriggerBusEvent(event string, payload any)` | Dispatch a bus event from the test side. |
| `PikoPartialReload(partialName string, data map[string]any)` | Force a partial refresh of the named partial. |
| `PikoPartialReloadWithLevel(partialName string, data map[string]any, refreshLevel int)` | As above, with an explicit refresh level. |
| `WaitForPartialReload(partialName string)` | Synchronise with a partial reload using the hard-coded 5-second timeout (no `WaitOption`). |
| `PikoWaitForPartialReload(partialName string, opts ...WaitOption)` | Synchronise with a partial reload; supply a timeout via `browser.WithTimeout`. |
| `PikoDispatchFragmentMorph(selector, newHTML string)` | Dispatch a fragment morph against `selector`. |
| `PikoGetPartialState(partialName string) map[string]any` | Read the named partial's state. |
| `TriggerPartialReload(name string, data map[string]any)` | Force a partial reload from the test side. |
| `PikoDebugIsAvailable() bool`, `PikoDebugIsConnected(selector string) bool` | Debug-runtime presence. |
| `PikoDebugGetPartialInfo(selector string)`, `PikoDebugGetCleanupCount(selector string)`, `PikoDebugGetRegisteredCallbacks(selector string)`, `PikoDebugGetAllConnectedPartials()` | Debug-runtime introspection. |

### Utility

`Close()`, `Pause()`.

## Types

| Type | Purpose |
|---|---|
| `Harness` | Test environment manager. |
| `Page` | Browser page with fluent API. Bound to a `testing.TB`. |
| `Session` | Standalone session. Returns errors instead of calling `t.Fatalf`. |
| `Assertion` | Assertion builder returned by `Page.Assert`. |
| `Dimensions` | Bounding box (`X`, `Y`, `Width`, `Height` as floats). |
| `Device` | Device configuration (`Name`, `Width`, `Height`, `Scale`, `Mobile`). |
| `Capturer` | Low-level headless driver for page capture. |
| `TestSpec` | Struct describing a declarative test case. Fields: `Description`, `RequestURL`, `BrowserSteps []BrowserStep`, `PKCComponents`, `NetworkMocks`, `ExpectedStatus`, `ShouldError`, `ErrorContains`, `TLS`, `RequiresMarkdown`, `Partials`, `AdditionalGoldenFiles`. |
| `BrowserStep` | Single declarative step inside `TestSpec.BrowserSteps`. |
| `CaptureResult` | Page capture output. Fields: `URL string`, `HTML string`, `Screenshot []byte` (set when `CaptureOptions.ChunkScreenshots` is `false`), `Screenshots []ScreenshotChunk` (set when `ChunkScreenshots` is `true`). |
| `CaptureOptions` | Capture configuration. |

## Capturer

```go
func NewCapturer(opts CaptureOptions) (*Capturer, error)
func DefaultCaptureOptions() CaptureOptions
```

Capturer records video or step-by-step screenshots during a run. Useful for regression diffs and review artefacts.

## Declarative test specs

`RunSpec` runs a JSON document describing a `TestSpec` object. The spec wraps an ordered list of steps under `browserSteps`, plus optional fields for expected status, network mocks, partial configurations, and additional golden files.

Action and assertion names use camelCase. Action handlers live at [`wdk/browser/internal/browser_provider_chromedp/actions_handlers.go`](https://github.com/piko-sh/piko/blob/master/wdk/browser/internal/browser_provider_chromedp/actions_handlers.go). Assertion handlers live alongside them in `assertions.go`.

| Category | Action names |
|---|---|
| Navigation | `navigate`, `goBack`, `goForward`, `stop` |
| Interaction | `click`, `doubleClick`, `rightClick`, `hover`, `fill`, `setValue`, `clear`, `submit`, `check`, `uncheck`, `setFiles`, `focus`, `blur`, `scroll`, `scrollIntoView`, `press`, `type`, `keyDown`, `keyUp`, `setCursor`, `setSelection`, `selectAll`, `collapseSelection` |
| Waiting | `wait`, `waitForSelector`, `waitForText`, `waitForVisible`, `waitForNotVisible`, `waitForEnabled`, `waitForDisabled`, `waitForNotPresent`, `waitForPartialReload` |
| Piko bus / partials | `dispatchEvent`, `triggerPartialReload`, `triggerBusEvent`, `pikoBusEmit`, `pikoPartialReload` |
| Misc | `eval`, `comment`, `clearConsole`, `setAttribute`, `removeAttribute`, `setViewport` |
| Assertions | `checkText`, `checkTextContains`, `checkTextNotContains`, `checkVisible`, `checkHidden`, `checkFocused`, `checkNotFocused`, and other `check*` variants |

Example spec:

```json
{
  "description": "Login flow",
  "requestURL": "/login",
  "expectedStatus": 200,
  "browserSteps": [
    { "action": "fill", "selector": "[name=email]", "value": "alice@example.com" },
    { "action": "fill", "selector": "[name=password]", "value": "hunter2" },
    { "action": "click", "selector": "button[type=submit]" },
    { "action": "waitForText", "selector": "h1", "value": "Welcome" },
    { "action": "checkText", "selector": "h1", "value": "Welcome" }
  ]
}
```

Match-against-golden checks happen via the spec's `additionalGoldenFiles` field, not as a per-step action.

## See also

- [About browser testing](../explanation/about-browser-testing.md) for the harness-vs-pikotest boundary, when to reach for each, and the rationale for goldens and spec files.
- [How to browser testing](../how-to/browser-testing.md) for scaffolding a first test, using device emulation, writing a spec file, and refreshing goldens.
- [Testing API reference](testing-api.md) for AST-level pikotest unit tests.
- Source: [`wdk/browser/`](https://github.com/piko-sh/piko/tree/master/wdk/browser).
