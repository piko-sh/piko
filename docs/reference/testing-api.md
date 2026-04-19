---
title: Testing API
description: The pikotest surface - request builder, component and action testers, view assertions, and AST query results.
nav:
  sidebar:
    section: "reference"
    subsection: "testing"
    order: 230
---

# Testing API

The `piko.sh/piko` package exposes a unit-test harness for pages, PKC components, and server actions. This page lists every public symbol. For recipes see [how to testing](../how-to/testing.md). For browser-driven end-to-end tests see [browser testing harness reference](browser-testing.md).

## Request builder

```go
func NewTestRequest(method, path string) *RequestBuilder
```

`NewTestRequest` returns a fluent `*RequestBuilder`. Chain `With*` calls and finish with `Build(ctx)` to obtain a `*RequestData`, or `BuildHTTPRequest(ctx)` for both `*http.Request` and `*RequestData`. The context is the terminus argument - there is no `WithContext`.

| Method | Purpose |
|---|---|
| `WithQueryParam(key, value)` | Add a single URL query parameter. |
| `WithQueryParams(map[string][]string)` | Add multiple query parameters. |
| `WithPathParam(key, value)` | Add a single route path parameter. |
| `WithPathParams(map[string]string)` | Add multiple path parameters. |
| `WithFormData(key, value)` | Add a single form field. |
| `WithFormDataMap(map[string][]string)` | Add multiple form fields. |
| `WithLocale(locale)` | Set the request locale. |
| `WithDefaultLocale(locale)` | Set the fallback locale. |
| `WithHost(host)` | Set the Host header. |
| `WithHeader(key, value)` | Add an HTTP header. |
| `WithGlobalTranslations(map)` | Seed global translations. |
| `WithLocalTranslations(map)` | Seed component-scoped translations. |
| `WithCollectionData(any)` | Seed collection data for `p-collection` pages. |
| `Build(ctx)` | Return `*RequestData`. |
| `BuildHTTPRequest(ctx)` | Return `(*http.Request, *RequestData)`. |

## Component tester

```go
func NewComponentTester(tb testing.TB, buildAST BuildASTFunc, opts ...ComponentOption) *ComponentTester
```

`buildAST` is the generated `BuildAST` function from the compiled component package at `dist/pages/.../`.

| Method | Purpose |
|---|---|
| `Render(req, props)` | Return a `*TestView` bound to the rendered output. |
| `Benchmark(req, props)` | Run the render in a benchmark loop (the method calls `b.ResetTimer()` internally). |

## Test view

A `*TestView` exposes the rendered AST, metadata, and helper assertions.

### State and metadata

| Method | Purpose |
|---|---|
| `State() any` | Return the rendered component state. |
| `AssertState(func(state any))` | Run a callback against the typed state for fine-grained assertions. |
| `Metadata()` | Return the internal metadata struct. |
| `AssertTitle(s)` | Assert `Metadata.Title`. |
| `AssertDescription(s)` | Assert `Description`. |
| `AssertLanguage(s)` | Assert `Language`. |
| `AssertCanonicalURL(s)` | Assert `CanonicalURL`. |
| `AssertStatusCode(n)` | Assert `StatusCode`. |
| `AssertDefaultStatusCode()` | Assert the render did not set a custom status. |
| `AssertClientRedirect(url)` | Assert a 302/303-style client redirect. |
| `AssertServerRedirect(url)` | Assert an internal redirect. |
| `AssertHasMetaTag(name, content)` | Assert a `<meta>` entry. |
| `AssertHasOGTag(property, content)` | Assert an Open Graph tag. |
| `AssertHasJSScript()` / `AssertNoJSScript()` | Assert client-script presence. |
| `AssertJSScriptURLs(expected []string)` | Assert the exact set of script URLs. |
| `AssertJSScriptURLContains(substr)` | Assert any script URL contains the substring. |

### AST queries

```go
func (v *TestView) AST() *TemplateAST
func (v *TestView) QueryAST(selector string) *ASTQueryResult
```

`QueryAST` accepts any standard CSS selector. Supported forms include tag, class, ID, attribute selectors, combinators, and pseudo-classes (`:first-child`, `:nth-child(2n+1)`, `:not(...)`, etc.).

| Method on `*ASTQueryResult` | Purpose |
|---|---|
| `Exists()` | Assert at least one match. |
| `NotExists()` | Assert zero matches. |
| `Count(n)` | Assert exact match count. |
| `MinCount(n)` / `MaxCount(n)` | Bound the match count. |
| `HasText(text)` | Assert the first match's text content equals. |
| `ContainsText(text)` | Assert substring match in any of the matched nodes. |
| `HasAttribute(key, value)` | Exact attribute match. |
| `HasAttributeContaining(key, substr)` | Attribute substring match. |
| `HasAttributePresent(key)` | Attribute exists (any value). |
| `HasClass(name)` | Assert CSS class. |
| `HasTag(name)` | Assert tag name. |
| `First()` / `Last()` / `At(i)` | Return a `*TemplateNode`. |
| `FirstResult()` / `Index(i)` | Return a chainable `*ASTQueryResult` (for further assertions). |
| `Len()` | Return the match count. |
| `Nodes()` | Return `[]*TemplateNode`. |
| `Each(func(int, *TemplateNode))` | Iterate. |
| `Filter(func(*TemplateNode) bool)` | Return a filtered `*ASTQueryResult`. |
| `Map(func(*TemplateNode) any) []any` | Project each match to a derived value. |
| `Dump()` | Log matched nodes to test output (debugging aid). |

> **Note:** `First()` returns a raw `*TemplateNode` and is not chainable for assertions. Use `FirstResult()` when you want to chain `.ContainsText(...)` etc.

### HTML rendering and snapshots

```go
func (v *TestView) HTML() []byte
func (v *TestView) HTMLString() string
func (v *TestView) WriteTo(w io.Writer) (int64, error)
func (v *TestView) MatchSnapshot(name string)
```

HTML rendering requires a configured `RenderService`. AST queries run faster and skip this setup.

`MatchSnapshot` writes to `__snapshots__/<test-directory>/<name>.golden.html` on first run. Subsequent runs compare and fail on difference. Set `PIKO_UPDATE_SNAPSHOTS=1` (or `UPDATE_SNAPSHOTS=1`) to regenerate.

## Action tester

```go
func NewActionTester(tb testing.TB, entry ActionHandlerEntry) *ActionTester
```

`entry` is a generated `piko.ActionHandlerEntry` (an alias for `daemon_adapters.ActionHandlerEntry`) that describes the action. It carries its `Name`, a `Create` factory that returns a fresh action instance, and an `Invoke` function that calls the action's `Call` method against a parsed argument map. Generated code emits one `ActionHandlerEntry` per action. Pass that value (for example `actions.CustomerCreate`), not a pointer to a struct literal.

```go
func TestCustomerCreate(t *testing.T) {
    tester := piko.NewActionTester(t, actions.CustomerCreate)
    result := tester.Invoke(context.Background(), map[string]any{"name": "Acme Corp"})
    result.AssertSuccess()
}
```

| Method | Purpose |
|---|---|
| `Invoke(ctx, args map[string]any)` | Run the action through `entry.Invoke`. Returns `*ActionResultView`. The map is the form-style argument set the action would receive in production; pass `nil` for actions with no parameters. |

## Action result view

A `*ActionResultView` exposes the action's response and helper assertions.

| Method | Purpose |
|---|---|
| `AssertSuccess()` | Assert the action returned no error. |
| `AssertError()` | Assert the action returned an error. |
| `AssertErrorContains(substr)` | Assert the error's user-safe message contains `substr` (use this for field-level validation messages too). |
| `AssertHelper(name)` | Assert the action queued a response helper (`redirect`, `toast`, etc.). |
| `AssertNoHelpers()` | Assert the action queued no helpers. |
| `Data() any` | Return the typed response (cast to your `Response` struct pointer). |
| `Err() error` | Return the raw error returned from `Call` (useful for type-checking). |
| `Result() *ActionResult` | Return the full `ActionResult` (status, helpers, body). |

## Environment variables

| Variable | Effect |
|---|---|
| `PIKO_UPDATE_SNAPSHOTS=1` | Rewrite golden files in `__snapshots__/`. |
| `UPDATE_SNAPSHOTS=1` | Alias for `PIKO_UPDATE_SNAPSHOTS`. |

## See also

- [How to testing](../how-to/testing.md) for scaffolding, mocking, and action-test recipes.
- [Browser testing harness reference](browser-testing.md) for end-to-end tests.
- [Errors reference](errors.md) for the error types that action assertions inspect.
- Tutorial: [Testing what you built](../tutorials/06-testing-what-you-built.md).
