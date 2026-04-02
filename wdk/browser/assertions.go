// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package browser

import (
	"fmt"
	"testing"
	"time"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

// Assertion provides a fluent API for checking element properties.
// Create one using Page.Assert(selector).
type Assertion struct {
	// t is the test context for reporting failures and log messages.
	t testing.TB

	// page is the parent Page used for action tracking and context.
	page *Page

	// selector is the CSS selector used to find the element being checked.
	selector string
}

// Exists checks that at least one element matches the selector.
//
// Returns *Assertion which allows for method chaining.
func (a *Assertion) Exists() *Assertion {
	a.runAssertion("Exists", func() error {
		return browser_provider_chromedp.CheckElementCount(a.page.actionCtx(), a.selector, -1)
	})
	return a
}

// NotExists asserts that no elements match the selector.
//
// Returns *Assertion which allows method chaining.
func (a *Assertion) NotExists() *Assertion {
	a.runAssertion("NotExists", func() error {
		return browser_provider_chromedp.CheckElementCount(a.page.actionCtx(), a.selector, 0)
	})
	return a
}

// Count asserts that exactly n elements match the selector.
//
// Takes n (int) which specifies the expected number of matching elements.
//
// Returns *Assertion which allows method chaining.
func (a *Assertion) Count(n int) *Assertion {
	a.runAssertion("Count", func() error {
		return browser_provider_chromedp.CheckElementCount(a.page.actionCtx(), a.selector, n)
	})
	return a
}

// HasText asserts that the element's text content equals the expected value.
// Polls until the text matches or timeout (5s).
//
// Takes expected (string) which is the text content to match.
//
// Returns *Assertion which allows method chaining.
func (a *Assertion) HasText(expected string) *Assertion {
	a.runAssertion("HasText", func() error {
		return browser_provider_chromedp.CheckText(a.page.actionCtx(), a.selector, expected)
	})
	return a
}

// ContainsText asserts that the element's text content contains the substring.
// Polls until the text contains the substring or timeout (5s).
//
// Takes substring (string) which is the text to search for in the element.
//
// Returns *Assertion which allows chaining further assertions.
func (a *Assertion) ContainsText(substring string) *Assertion {
	a.runAssertion("ContainsText", func() error {
		return browser_provider_chromedp.CheckTextContains(a.page.actionCtx(), a.selector, substring)
	})
	return a
}

// HasHTML asserts that the element's inner HTML equals the expected value.
// This is an instant check without polling.
//
// Takes expected (string) which specifies the HTML content to match.
//
// Returns *Assertion which allows method chaining for further assertions.
func (a *Assertion) HasHTML(expected string) *Assertion {
	a.runAssertion("HasHTML", func() error {
		return browser_provider_chromedp.CheckHTML(a.page.actionCtx(), a.selector, expected)
	})
	return a
}

// HasAttribute asserts that the element has an attribute with the expected
// value. This is an instant check without polling.
//
// Takes name (string) which specifies the attribute name to check.
// Takes expected (string) which specifies the expected attribute value.
//
// Returns *Assertion which allows chaining further assertions.
func (a *Assertion) HasAttribute(name, expected string) *Assertion {
	a.runAssertion("HasAttribute", func() error {
		return browser_provider_chromedp.CheckAttribute(a.page.actionCtx(), a.selector, name, expected)
	})
	return a
}

// HasClass asserts that the element has the specified CSS class.
// This is an instant check without polling.
//
// Takes className (string) which specifies the CSS class name to check for.
//
// Returns *Assertion which allows for method chaining.
func (a *Assertion) HasClass(className string) *Assertion {
	a.runAssertion("HasClass", func() error {
		return browser_provider_chromedp.CheckClass(a.page.actionCtx(), a.selector, className)
	})
	return a
}

// HasStyle asserts that the element has the specified computed style.
// This is an instant check without polling.
//
// Takes property (string) which specifies the CSS property name to check.
// Takes expected (string) which specifies the expected computed value.
//
// Returns *Assertion which allows method chaining for further assertions.
func (a *Assertion) HasStyle(property, expected string) *Assertion {
	a.runAssertion("HasStyle", func() error {
		return browser_provider_chromedp.CheckStyle(a.page.actionCtx(), a.selector, property, expected)
	})
	return a
}

// HasValue asserts that an input element has the expected value.
// Polls until the value matches or timeout (5s).
//
// Takes expected (string) which is the value to match against the element.
//
// Returns *Assertion which allows method chaining for further assertions.
func (a *Assertion) HasValue(expected string) *Assertion {
	a.runAssertion("HasValue", func() error {
		return browser_provider_chromedp.CheckValue(a.page.actionCtx(), a.selector, expected)
	})
	return a
}

// IsChecked asserts that a checkbox or radio button is checked.
// This is an instant check without polling.
//
// Returns *Assertion which allows chaining of further assertions.
func (a *Assertion) IsChecked() *Assertion {
	a.runAssertion("IsChecked", func() error {
		return browser_provider_chromedp.CheckChecked(a.page.actionCtx(), a.selector)
	})
	return a
}

// IsUnchecked asserts that a checkbox or radio button is not checked.
// This is an instant check without polling.
//
// Returns *Assertion which allows for method chaining.
func (a *Assertion) IsUnchecked() *Assertion {
	a.runAssertion("IsUnchecked", func() error {
		return browser_provider_chromedp.CheckUnchecked(a.page.actionCtx(), a.selector)
	})
	return a
}

// IsFocused asserts that the element is currently focused.
// This is an instant check without polling.
//
// Returns *Assertion which allows method chaining for further assertions.
func (a *Assertion) IsFocused() *Assertion {
	a.runAssertion("IsFocused", func() error {
		return browser_provider_chromedp.CheckFocused(a.page.actionCtx(), a.selector)
	})
	return a
}

// IsVisible asserts that the element is visible (not hidden).
// This is an instant check without polling.
//
// Returns *Assertion which allows method chaining.
func (a *Assertion) IsVisible() *Assertion {
	a.runAssertion("IsVisible", func() error {
		return browser_provider_chromedp.CheckVisible(a.page.actionCtx(), a.selector)
	})
	return a
}

// IsHidden asserts that the element is hidden (not visible).
// This is an instant check without polling.
//
// Returns *Assertion which allows chaining additional assertions.
func (a *Assertion) IsHidden() *Assertion {
	a.runAssertion("IsHidden", func() error {
		return browser_provider_chromedp.CheckHidden(a.page.actionCtx(), a.selector)
	})
	return a
}

// IsEnabled asserts that the element is enabled (not disabled).
// This is an instant check without polling.
//
// Returns *Assertion which allows chaining additional assertions.
func (a *Assertion) IsEnabled() *Assertion {
	a.runAssertion("IsEnabled", func() error {
		return browser_provider_chromedp.CheckEnabled(a.page.actionCtx(), a.selector)
	})
	return a
}

// IsDisabled asserts that the element is disabled.
// This is an instant check without polling.
//
// Returns *Assertion which allows for method chaining.
func (a *Assertion) IsDisabled() *Assertion {
	a.runAssertion("IsDisabled", func() error {
		return browser_provider_chromedp.CheckDisabled(a.page.actionCtx(), a.selector)
	})
	return a
}

// MatchesGolden compares the element's normalised HTML against a golden file
// stored at testdata/golden/<name>.html relative to the test working
// directory, creating or updating the file when the PIKO_UPDATE_GOLDEN=1
// environment variable is set.
//
// Takes name (string) which specifies the golden file name (without extension).
//
// Returns *Assertion which allows method chaining.
func (a *Assertion) MatchesGolden(name string) *Assertion {
	a.runAssertion("MatchesGolden", func() error {
		html, err := browser_provider_chromedp.CaptureDOM(a.page.actionCtx(), a.selector, true)
		if err != nil {
			return fmt.Errorf("capturing DOM for %q: %w", a.selector, err)
		}
		normalised := browser_provider_chromedp.NormaliseDOM(html, browser_provider_chromedp.DefaultNormaliseOptions())
		return compareGolden(a.t, name, normalised)
	})
	return a
}

// runAssertion wraps an assertion check with TUI tracking and timing.
//
// Takes name (string) which identifies the assertion type for logging.
// Takes check (func() error) which runs the actual assertion.
func (a *Assertion) runAssertion(name string, check func() error) {
	a.page.beforeAction("Assert."+name, a.selector)
	start := time.Now()
	err := check()
	a.page.afterAction("Assert."+name, a.selector, err != nil, time.Since(start))
	if err != nil {
		a.t.Errorf("Assert(%q).%s() failed: %v", a.selector, name, err)
	}
}
