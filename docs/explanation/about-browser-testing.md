---
title: About browser testing
description: Where browser tests sit versus pikotest, why the harness is separate, and the trade-offs of goldens and declarative specs.
nav:
  sidebar:
    section: "explanation"
    subsection: "testing"
    order: 90
---

# About browser testing

Piko ships two testing substrates. Pikotest runs unit tests against compiled templates without a browser, hitting the AST directly. The browser harness in `wdk/browser` runs the full compiled server in a real headless Chrome and drives it through Chromedp. Each has a role. This page explains where the boundary sits and why both exist.

## The fast layer and the true layer

Pikotest is fast. A test renders a component's AST, asserts on the structure, and moves on. No browser boots, no server starts, no network round-trip. A table-driven pikotest pass over twenty scenarios runs in milliseconds. The cost is that pikotest does not exercise the client-side runtime. PKC reactive state, the event bus, partial refresh, and server actions do not run under pikotest.

The browser harness is slow. A single test compiles the project, starts the server, boots Chrome, loads a page, runs scripts, and asserts against the rendered DOM. Round-trip time for one assertion is hundreds of milliseconds even on a fast machine. The cost buys truth. The test exercises the real browser on the real server.

The trade-off is familiar from other testing disciplines. Unit tests are fast and local, integration tests are slow and global, and most codebases need both. Pikotest handles the unit layer. The browser harness handles the integration layer.

## Where each layer shines

Pikotest shines for:

- Assertions about what the template produces for a given data shape. "Given this customer, the customer card renders with their name and their plan."
- Branching logic that would be tedious to cover at the browser level. "Given an empty list, the empty state shows. Given one item, the singular-item layout shows. Given five items, the list layout shows."
- Action validation. "Given this input, the action returns a field-level error on `email`."
- Metadata assertions. "Given this page, the `<title>` and Open Graph tags are correct."

The browser harness shines for:

- PKC component behaviour. "When the user clicks the button, the counter increments and the DOM reflects the change."
- End-to-end flows. "The user fills the form, submits, sees a toast, and lands on the success page."
- Partial refresh. "When the server pushes an update through the event bus, the partial re-renders without the rest of the page reloading."
- Cross-page navigation with state. "The user logs in, visits a protected page, logs out, and the protected page now redirects."
- Visual regression. "The dashboard at this breakpoint looks exactly like the reference screenshot."

The boundary works in practice as follows. If a test answers itself from the AST without running JavaScript, use pikotest. If the answer requires the runtime, use the browser harness.

## Why the harness is separate from pikotest

Pikotest runs in-process with the server code. A pikotest fails fast because it does not go through the network. Adding browser support to pikotest would pull Chromedp into the unit-test dependency tree, which in turn drags a Chrome installation into every build. The dependency cost is too high for the volume of unit tests the framework wants to encourage.

Keeping the harness separate makes the trade explicit at the import level. A package that imports `pikotest` runs fast tests. A package that imports `wdk/browser` runs slow tests. Build engineers can split the two into separate CI jobs, or run pikotest on every commit and browser tests on pull requests.

## Declarative specs as the cheap starting point

The programmatic Page API is expressive. Every assertion is Go code, every condition a Go expression. But writing Go for a test that is a linear script (navigate, fill, click, assert) feels heavy. The declarative `TestSpec` JSON format is a cheaper option for those cases.

A spec is a list of steps. Each step names an action (navigate, fill, click, assert_text, match_golden) and its arguments. A QA engineer without deep Go knowledge can write a spec after seeing one example. A non-technical product owner can often read one and understand what the test does. Specs live as data files in `testdata/` alongside the goldens they reference.

The term "gateway drug" is deliberate. Specs work for linear flows. When a test needs branching, conditional assertions, or loops, it needs the full Page API. Starting with a spec and promoting to Go is a natural path. Starting with Go is the right default for complex flows.

The spec runner is a thin wrapper around the Page API. Every spec action corresponds to a Page method. This means a new Page method does not automatically appear as a spec action. The spec vocabulary grows deliberately.

## Goldens as regression triggers

Visual regression is a class of bug where the code is correct but the render is wrong. A CSS change reorders elements. A font replacement changes line heights. A minor template tweak pushes content off the screen. Unit tests do not catch these. They pass because the data shape stays the same.

Golden files are pre-recorded screenshots compared on each run. A failing golden test is a strong signal that something changed visually. The signal is noisy (a one-pixel anti-aliasing difference fails a naive byte comparison) and the fix-up cost is non-trivial (review the diff, regenerate the golden, commit). But for pages where visual presentation matters, the trade is worth it.

Piko's golden support is deliberately simple. `MatchGolden(name)` saves a screenshot on first run and compares on later runs. `match_golden` is the corresponding spec action. Regeneration uses `-update-goldens` at the test command. There is no built-in diff-viewer tooling. Teams usually bring their own.

Common failure modes:

- **Flaky goldens**. Time-sensitive content (a "last updated" timestamp) fails every run. Fix: either freeze time in the test environment, or exclude the region from the golden comparison.
- **Environment-sensitive goldens**. The browser renders slightly differently across operating systems. Fix: generate goldens in CI and run comparisons in CI; do not generate locally and compare in CI.
- **Breakpoint-sensitive goldens**. A desktop golden fails on a mobile viewport. Fix: always apply a viewport preset before the assertion.

## Piko-specific hooks

The harness exposes methods starting with `Piko*` that reach into the framework's client runtime. These are pragmatic concessions. Testing a PKC component's reactive state is much easier if the test can inspect the state directly. The test can also trigger a bus event from the harness or watch the event log.

The alternative would be to test only through observable DOM behaviour. That works for most assertions but becomes awkward when the test wants to verify that a specific event fired even though the DOM response is subtle. The `PikoBus*` and `PikoPartial*` helpers cut through the indirection.

Using the hooks has a cost. A test that asserts on bus events takes a dependency on the framework's event model. If the event model changes, the test has to change. This is acceptable because the test is framework-aware by its nature. It tests Piko behaviour, not browser behaviour in general.

## The cost of keeping the harness running

A browser test is a wall-clock budget. Ten browser tests that each take two seconds cost twenty seconds. A hundred such tests cost over three minutes, which starts to hurt developer iteration. Two mitigations ship with the harness:

- Shared harness. `TestMain` sets up one harness for the whole package. Every test reuses the compiled binary, the running server, and the browser process. Setup cost amortises over the full test list.
- `WithSkipBuild(true)`. If the binary is already built, skip the recompile. Useful when iterating on test code without touching application code.

Watch the longer-term cost. A thousand browser tests that each take two seconds run for over half an hour. At that scale, split the suite and run most tests on pull requests and a smaller smoke subset on every commit.

## When not to reach for the browser harness

Interactive debugging of a broken page is usually faster in a real browser than in the harness. The `-interactive` flag opens a TUI that pauses at each step, but the browser's dev tools (breakpoint in JavaScript, inspect DOM, live-edit CSS) still offer more control. Use the harness for automated assertions, not for exploratory debugging.

Load testing is a different discipline. A browser harness measures correctness, not performance. A single browser driving a Piko server does not produce the concurrency profile of production traffic. For load testing use a dedicated tool (k6, vegeta, ab) against the HTTP endpoint.

Accessibility testing through the harness is possible but partial. Automated accessibility checks (axe-core run from the browser) catch a wide class of violations. They do not replace a manual screen-reader walkthrough, and a clean automated pass does not imply an accessible page.

## See also

- [Browser testing harness reference](../reference/browser-testing.md) for the complete Page, Harness, and Spec API.
- [How to browser testing](../how-to/browser-testing.md) for scaffolding a first test, using device emulation, writing a spec file, and refreshing goldens.
- [Testing API reference](../reference/testing-api.md) for the AST-level pikotest surface.
- [How to testing](../how-to/testing.md) for pikotest recipes.
