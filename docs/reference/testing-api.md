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

The `piko.sh/piko` package exposes a unit-test harness for pages, PKC components, and server actions. This page lists every public symbol. Source of truth: [`pikotest.go`](https://github.com/piko-sh/piko/blob/master/pikotest.go). For recipes see [how to testing](../how-to/testing.md). For browser-driven end-to-end tests see [browser testing harness reference](browser-testing.md).

## Request builder

```go
func NewTestRequest(method, target string) *TestRequestBuilder
```

`NewTestRequest` returns a fluent builder. Chain `With*` calls and finish with `Build()` to obtain a `*RequestData`, or `BuildHTTPRequest()` for both `*http.Request` and `*RequestData`.

| Method | Purpose |
|---|---|
| `WithContext(ctx)` | Attach a context (for dependency injection). |
| `WithQueryParam(key, value)` | Add a single URL query parameter. |
| `WithQueryParams(map[string][]string)` | Add multiple query parameters. |
| `WithPathParam(key, value)` | Add a single route path parameter. |
| `WithPathParams(map[string]string)` | Add multiple path parameters. |
| `WithFormData(key, value)` | Add a single form field. |
| `WithFormDataMap(map[string][]string)` | Add multiple form fields. |
| `WithLocale(locale)` | Set the request locale (default `en`). |
| `WithDefaultLocale(locale)` | Set the fallback locale (default `en`). |
| `WithHost(host)` | Set the Host header (default `localhost`). |
| `WithHeader(key, value)` | Add an HTTP header. |
| `WithGlobalTranslations(map)` | Seed global translations. |
| `WithLocalTranslations(map)` | Seed component-scoped translations. |
| `WithCollectionData(any)` | Seed collection data for `p-collection` pages. |
| `Build()` | Return `*RequestData`. |
| `BuildHTTPRequest()` | Return `(*http.Request, *RequestData)`. |

## Component tester

```go
func NewComponentTester(t testing.TB, buildAST BuildASTFunc, opts ...ComponentTesterOption) *ComponentTester
```

`buildAST` is the generated `BuildAST` function from the compiled component package at `dist/pages/.../`.

| Option | Purpose |
|---|---|
| `WithPageID(id)` | Attach a page identifier to error messages. |

| Method | Purpose |
|---|---|
| `Render(req, props)` | Return a `*ComponentView` bound to the rendered output. |
| `Benchmark(req, props)` | Run the render in a benchmark loop. Calls `b.ResetTimer()` internally. |

## Component view

A `*ComponentView` exposes the rendered AST, metadata, and helper assertions.

### AST queries

```go
func (v *ComponentView) QueryAST(selector string) *ASTQueryResult
```

Accepts any standard CSS selector. Supported forms include tag, class, ID, attribute selectors, combinators, and pseudo-classes (`:first-child`, `:nth-child(2n+1)`, `:not(...)`, etc.).

| Method on `*ASTQueryResult` | Purpose |
|---|---|
| `Exists()` | Assert at least one match. |
| `NotExists()` | Assert zero matches. |
| `Count(n)` | Assert exact match count. |
| `MinCount(n)` / `MaxCount(n)` | Bound the match count. |
| `HasText(text)` | Assert the first match's text content. |
| `ContainsText(text)` | Assert substring match. |
| `HasAttribute(key, value)` | Exact attribute match. |
| `HasAttributeContaining(key, substr)` | Attribute substring match. |
| `HasAttributePresent(key)` | Attribute exists (any value). |
| `HasClass(name)` | Assert CSS class. |
| `HasTag(name)` | Assert tag name. |
| `First()` / `Last()` / `At(i)` | Return `*TemplateNode`. |
| `FirstResult()` / `Index(i)` | Return a chainable `*ASTQueryResult`. |
| `Len()` | Return match count. |
| `Nodes()` | Return `[]*TemplateNode`. |
| `Each(func(int, *TemplateNode))` | Iterate. |
| `Filter(func(*TemplateNode) bool)` | Return a filtered `*ASTQueryResult`. |
| `Map(func(*TemplateNode) any)` | Return a slice of derived values. |
| `Dump()` | Log matched nodes to test output. |

### Metadata assertions

| Method | Purpose |
|---|---|
| `AssertTitle(s)` | Assert `piko.Metadata.Title`. |
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
| `AssertJSScriptURLContains(substr)` | Assert script URL substring. |
| `Metadata()` | Return the raw `piko.Metadata`. |

### HTML rendering

```go
func (v *ComponentView) HTML() []byte
func (v *ComponentView) HTMLString() string
func (v *ComponentView) WriteTo(io.Writer) (int64, error)
func (v *ComponentView) MatchSnapshot(name string)
```

HTML rendering requires a configured `RenderService`. AST queries run faster and skip this setup.

`MatchSnapshot` writes to `__snapshots__/<test-directory>/<name>.golden.html` on first run. Subsequent runs compare and fail on difference. Set `PIKO_UPDATE_SNAPSHOTS=1` (or `UPDATE_SNAPSHOTS=1`) to regenerate.

## Action tester

```go
func NewActionTester(t testing.TB, action any) *ActionTester
```

`action` is a pointer to an action struct value (for example `&actions.CustomerCreate{}`).

| Method | Purpose |
|---|---|
| `Invoke(req)` | Run `Call`; return the typed response. |
| `InvokeExpectError(req)` | Run `Call`; return the returned error. |

### Action assertions

| Method | Purpose |
|---|---|
| `AssertSuccess(resp)` | Assert 2xx status and no field errors. |
| `AssertError(resp)` | Assert non-2xx or has errors. |
| `AssertStatusCode(resp, n)` | Assert HTTP status. |
| `AssertErrorMessage(resp, msg)` | Assert top-level error message. |
| `AssertFieldError(resp, field, msg)` | Assert a field-level error message. |
| `AssertHasFieldError(resp, field)` | Assert a field has any error. |
| `AssertNoFieldError(resp, field)` | Assert a field has no error. |
| `AssertHasHelper(resp, name)` | Assert the action queued a response helper (`redirect`, `toast`, etc.). |
| `AssertHelperArg(resp, name, index, value)` | Assert the helper argument at a position. |
| `GetHelper(resp, name)` | Return the helper entry for inspection. |

## Environment variables

| Variable | Effect |
|---|---|
| `PIKO_UPDATE_SNAPSHOTS=1` | Rewrite golden files in `__snapshots__/`. |
| `UPDATE_SNAPSHOTS=1` | Alias for `PIKO_UPDATE_SNAPSHOTS`. |

## See also

- [How to testing](../how-to/testing.md) for scaffolding, mocking, and action-test recipes.
- [Browser testing harness reference](browser-testing.md) for end-to-end tests.
- [Errors reference](errors.md) for the error types that action assertions inspect.
- Source: [`pikotest.go`](https://github.com/piko-sh/piko/blob/master/pikotest.go).
