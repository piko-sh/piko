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
| `WithTimeout(d)` | Maximum time to wait. Default 5 seconds. |

Use with any wait method: `page.Wait(selector, browser.WithTimeout(30 * time.Second))`.

## Page API

A `*Page` exposes the full browser-driving surface. The sections below group methods by role.

### Navigation

`Navigate(url)`, `Reload()`, `Back()`, `Forward()`, `Stop()`, `Title() string`, `URL() string`.

### Waiting

`Wait(selector)`, `WaitFor(selector, condition)`, `WaitForText(selector, text)`, `WaitStable(selector)`, `WaitForVisible(selector)`, `WaitForNotVisible(selector)`, `WaitForEnabled(selector)`, `WaitForDisabled(selector)`, `WaitForNotPresent(selector)`.

### Interaction

| Method | Purpose |
|---|---|
| `Click`, `DoubleClick`, `RightClick` | Mouse interaction. |
| `Fill(selector, value)` | Clear and type. |
| `Clear(selector)` | Empty a field. |
| `Focus`, `Blur` | Focus management. |
| `Check`, `Uncheck` | Checkbox state. |
| `Select(selector, value)` | `<select>` element. |
| `Submit(selector)` | Submit a form. |
| `Type(selector, text)` | Append typed text. |
| `Press(selector, key)` | Send a key event. |
| `Hover(selector)` | Mouse-over. |
| `Scroll(selector)` | Scroll an element into view. |
| `DragAndDrop`, `DragAndDropHTML5` | Drag operations. |
| `SetViewport(width, height)` | Resize the viewport. |

### Frames

`ClickInFrame`, `FillInFrame`, `WaitForElementInFrame`, `GetTextInFrame`, `EvalInFrame`, `GetFrameDocument`, `CountFrames`, `GetFrames`, `IsFrameLoaded`.

### Assertions

`Assert(selector) *Assertion` returns a builder with methods such as `.Text(expected)`, `.ContainsText(substr)`, `.HasClass(name)`, `.HasAttribute(key, value)`, `.IsVisible()`, `.IsEnabled()`, `.Count(n)`. Chain assertions for composite checks.

Console checks: `AssertNoConsoleErrors()`, `AssertNoConsoleWarnings()`.

### Console

`ConsoleLogs()`, `ConsoleLogsWithLevel(level)`, `HasConsoleErrors()`, `ConsoleErrors()`, `ClearConsole()`.

### Storage

Cookies: `SetCookie`, `GetCookie`, `GetAllCookies`, `DeleteCookie`, `HasCookie`.
Local storage: `SetLocalStorageItem`, `GetLocalStorageItem`, `RemoveLocalStorageItem`, `GetAllLocalStorage`, `ClearLocalStorage`.
Session storage: `SetSessionStorageItem`, `GetSessionStorageItem`, `RemoveSessionStorageItem`, `GetAllSessionStorage`, `ClearSessionStorage`.

### Network

| Method | Purpose |
|---|---|
| `InterceptRequest(pattern, handler)` | Hook a request pattern. |
| `RemoveRequestIntercept`, `ClearRequestIntercepts` | Tear down intercepts. |
| `SetBasicAuthHeader`, `SetAuthorisationHeader` | Auth headers. |
| `SetCustomHeader`, `SetExtraHTTPHeaders`, `ClearExtraHTTPHeaders` | Custom headers. |
| `SetAcceptLanguageHeader`, `SetUserAgent` | Locale and UA. |
| `GetResponseHeaders(url)` | Inspect a response. |
| `CheckRequestMade(pattern)` | Assertion helper. |
| `WaitForRequest(pattern)`, `WaitForNetworkIdle()` | Request synchronisation. |

### Output

Screenshots: `Screenshot`, `ScreenshotFull`, `ScreenshotViewport`, `ScreenshotRegion`, `ScreenshotWithOptions`, `ScreenshotJPEG`, `ScreenshotWebP`, `ScreenshotElementWithPadding`, `SaveScreenshot`, `SaveScreenshotViewport`.

Comparison: `CompareScreenshots`, `MatchGolden(name)`.

PDF: `PrintToPDF`, `PrintToPDFA4`, `PrintToPDFLandscape`, `PrintToPDFWithOptions`, `SavePDF`.

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

`Eval(script)`, `EvalReturn(script)`, `EvalInFrame(frameSelector, script)`, `CaptureDOM()`.

### Dialogues

`HandleAlert`, `HandleConfirm`, `HandlePrompt`, `SetupDialogAutoAccept`, `SetupDialogAutoDismiss`, `AcceptDialog`, `DismissDialog` (method names match the underlying Chromedp API).

### Downloads

`WaitForDownload`, `GetDownloadedFile(name)`, `SetDownloadPath(path)`, `TriggerDownload(selector)`, `CreateBlobDownload(data)`.

### Cursor and selection

`GetCursorOffset`, `SetCursor`, `PlaceCursorInElement`, `CollapseToStart`, `CollapseToEnd`, `SelectAll`, `GetSelectionRange`, `SetFiles(selector, paths...)`.

### Piko-specific helpers

For Piko projects under test, the page exposes helpers for the framework's partial refresh and event bus:

| Method | Purpose |
|---|---|
| `PikoBusEmit(event, data)` | Emit a bus event in the browser. |
| `PikoBusWaitForEvent(event, timeout)` | Wait for a bus event. |
| `PikoCheckBusEventReceived(event)` | Assert an event fired. |
| `PikoGetEventLog()`, `PikoSetupEventLog()`, `PikoClearEventLog()` | Event-log plumbing. |
| `TriggerBusEvent(event, data)` | Trigger through the page. |
| `PikoPartialReload(selector)`, `PikoPartialReloadWithLevel(selector, level)` | Force a partial refresh. |
| `PikoWaitForPartialReload(selector, timeout)` | Synchronise with a partial reload. |
| `PikoDispatchFragmentMorph(selector, html)` | Trigger a fragment morph. |
| `PikoGetPartialState(selector)` | Read partial state. |
| `TriggerPartialReload(selector)` | Force a partial reload. |
| `PikoDebugIsAvailable()`, `PikoDebugIsConnected()` | Debug-runtime presence. |
| `PikoDebugGetPartialInfo()`, `PikoDebugGetCleanupCount()`, `PikoDebugGetRegisteredCallbacks()`, `PikoDebugGetAllConnectedPartials()` | Debug-runtime introspection. |

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
| `TestSpec` | Alias for `[]BrowserStep`. |
| `BrowserStep` | Single declarative test step. |
| `CaptureResult` | Page capture output (`URL`, `HTML`, `Screenshot` or `Screenshots`). |
| `CaptureOptions` | Capture configuration. |

## Capturer

```go
func NewCapturer(opts CaptureOptions) (*Capturer, error)
func DefaultCaptureOptions() CaptureOptions
```

Capturer records video or step-by-step screenshots during a run. Useful for regression diffs and review artefacts.

## Declarative test specs

`RunSpec` runs a JSON document enumerating browser actions and assertions. A spec is a sequence of `BrowserStep` objects. Each step specifies an action (navigate, click, assert, screenshot) and optional arguments. Run specs in CI to verify flows without writing Go test code for each scenario.

Typical step shape (JSON):

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

`wdk/browser/spec.go` defines the exact action vocabulary, which mirrors the Page methods listed above.

## See also

- [About browser testing](../explanation/about-browser-testing.md) for the harness-vs-pikotest boundary, when to reach for each, and the rationale for goldens and spec files.
- [How to browser testing](../how-to/browser-testing.md) for scaffolding a first test, using device emulation, writing a spec file, and refreshing goldens.
- [Testing API reference](testing-api.md) for AST-level pikotest unit tests.
- Source: [`wdk/browser/`](https://github.com/piko-sh/piko/tree/master/wdk/browser).
