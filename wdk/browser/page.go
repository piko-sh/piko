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
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
	"piko.sh/piko/wdk/safedisk"
)

// Page wraps a browser page with a fluent testing API and implements
// io.Closer. Each test should create its own Page using New and call Close
// when done.
type Page struct {
	// t is the test context used to report failures and log messages.
	t testing.TB

	// interactiveRunner controls step-by-step test execution; nil when not in
	// interactive mode.
	interactiveRunner InteractiveRunner

	// incognitoPage holds the browser context for running tests in isolation.
	incognitoPage *browser_provider_chromedp.IncognitoPage

	// harness is the parent test harness that owns this page.
	harness *Harness

	// pageHelper captures and retrieves console logs during page execution.
	pageHelper *browser_provider_chromedp.PageHelper

	// dialogHandler tracks the active dialog auto-handler for automatic cleanup.
	dialogHandler *browser_provider_chromedp.DialogAutoHandler

	// outputSandbox controls where screenshots, PDFs, and other test output files
	// are saved. Paths given to Save* methods are relative to this sandbox.
	outputSandbox safedisk.Sandbox

	// baseURL is the server URL used for test actions.
	baseURL string

	// currentPath stores the last URL path visited, used for retries.
	currentPath string
}

const (
	// displayTextMaxLen is the maximum length of text to display before it is
	// truncated.
	displayTextMaxLen = 30

	// fmtKeyValue is the format string for logging key-value pairs in actions.
	fmtKeyValue = "%s = %q"

	// fmtTruncatedText is the suffix added to text that has been shortened
	// for display purposes.
	fmtTruncatedText = "..."

	// outputFilePermissions is the permission mode for output files such as
	// screenshots and PDFs.
	outputFilePermissions = 0600

	// maxNavigateRetries is the number of times to retry navigation with page
	// recreation when the page becomes unresponsive.
	maxNavigateRetries = 2

	// maxWaitForRetries is the number of retry attempts when waiting for a selector
	// with page recreation.
	maxWaitForRetries = 2
)

// truncateRunes shortens s to at most maxRunes runes, appending "..." when the
// original string was longer. The function is rune-aware so it never cuts
// through a multi-byte UTF-8 sequence, which keeps display text valid for
// CJK, accented, and other non-ASCII content.
//
// Takes s (string) which is the input to truncate.
// Takes maxRunes (int) which is the maximum number of runes the result may
// contain before the suffix is appended. Values of zero or below produce an
// empty string.
//
// Returns string which is at most maxRunes runes long when s already fits,
// otherwise the first maxRunes runes followed by "...".
func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + fmtTruncatedText
}

// Navigate goes to a path on the test server.
//
// Takes path (string) which specifies the URL path to visit.
//
// Returns *Page which allows method chaining for fluent test syntax.
func (p *Page) Navigate(path string) *Page {
	p.beforeAction("Navigate", path)
	start := time.Now()

	var lastErr error
	for attempt := range maxNavigateRetries {
		err := browser_provider_chromedp.Navigate(p.actionCtx(), path)
		if err == nil {
			p.currentPath = path
			p.afterAction("Navigate", path, false, time.Since(start))
			return p
		}

		lastErr = err

		if !isUnresponsivePageError(err) {
			break
		}

		if attempt < maxNavigateRetries-1 {
			if recreateErr := p.recreatePage(); recreateErr != nil {
				lastErr = fmt.Errorf("navigate failed and page recreation failed: %w (original: %w)", recreateErr, err)
				break
			}
			continue
		}
	}

	p.afterAction("Navigate", path, true, time.Since(start))
	p.t.Fatalf("Navigate(%q) failed: %v", path, lastErr)
	return p
}

// Reload refreshes the current page in the browser.
//
// Returns *Page which allows method chaining.
func (p *Page) Reload() *Page {
	p.beforeAction("Reload", "")
	start := time.Now()

	timedCtx, cancel := context.WithTimeoutCause(
		p.incognitoPage.Ctx, 5*time.Second,
		errors.New("page Reload exceeded 5s timeout"),
	)
	defer cancel()

	err := chromedp.Run(timedCtx, chromedp.Reload())
	if err == nil {
		_ = browser_provider_chromedp.WaitStable(timedCtx, 500*time.Millisecond)
	}
	p.afterAction("Reload", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Reload() failed: %v", err)
	}
	return p
}

// Back moves to the previous page in the browser history.
//
// Returns *Page which allows method chaining.
func (p *Page) Back() *Page {
	p.beforeAction("Back", "")
	start := time.Now()
	err := browser_provider_chromedp.GoBack(p.actionCtx())
	p.afterAction("Back", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Back() failed: %v", err)
	}
	return p
}

// Forward moves forward in the browser history.
//
// Returns *Page which allows method chaining.
func (p *Page) Forward() *Page {
	p.beforeAction("Forward", "")
	start := time.Now()
	err := browser_provider_chromedp.GoForward(p.actionCtx())
	p.afterAction("Forward", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Forward() failed: %v", err)
	}
	return p
}

// Stop halts the current page from loading.
//
// Returns *Page which allows method chaining.
func (p *Page) Stop() *Page {
	p.beforeAction("Stop", "")
	start := time.Now()
	err := browser_provider_chromedp.Stop(p.actionCtx())
	p.afterAction("Stop", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Stop() failed: %v", err)
	}
	return p
}

// Title returns the title text of the page.
//
// Returns string which is the page title.
func (p *Page) Title() string {
	title, err := browser_provider_chromedp.GetTitle(p.actionCtx())
	if err != nil {
		p.t.Fatalf("Title() failed: %v", err)
	}
	return title
}

// URL returns the full URL of the current page.
//
// Returns string which is the page address.
func (p *Page) URL() string {
	url, err := browser_provider_chromedp.GetURL(p.actionCtx())
	if err != nil {
		p.t.Fatalf("URL() failed: %v", err)
	}
	return url
}

// Wait pauses for the given length of time.
//
// Takes d (time.Duration) which specifies how long to pause.
//
// Returns *Page which allows method chaining.
func (p *Page) Wait(d time.Duration) *Page {
	p.beforeAction("Wait", d.String())
	start := time.Now()
	time.Sleep(d)
	p.afterAction("Wait", d.String(), false, time.Since(start))
	return p
}

// WaitFor waits for an element matching the selector to appear.
//
// Takes selector (string) which is the CSS selector to wait for.
//
// Returns *Page which allows method chaining.
func (p *Page) WaitFor(selector string) *Page {
	p.beforeAction("WaitFor", selector)
	start := time.Now()

	var lastErr error
	for attempt := range maxWaitForRetries {
		err := browser_provider_chromedp.WaitForSelector(p.actionCtx(), selector, 5*time.Second)
		if err == nil {
			p.afterAction("WaitFor", selector, false, time.Since(start))
			return p
		}

		lastErr = err

		if !isUnresponsivePageError(err) {
			break
		}

		if p.currentPath == "" || attempt >= maxWaitForRetries-1 {
			break
		}

		if recreateErr := p.recreatePage(); recreateErr != nil {
			lastErr = fmt.Errorf("wait failed and page recreation failed: %w (original: %w)", recreateErr, err)
			break
		}

		if navErr := browser_provider_chromedp.Navigate(p.actionCtx(), p.currentPath); navErr != nil {
			lastErr = fmt.Errorf("wait failed and re-navigation failed: %w (original: %w)", navErr, err)
			break
		}

		continue
	}

	p.afterAction("WaitFor", selector, true, time.Since(start))
	p.t.Fatalf("WaitFor(%q) failed: %v", selector, lastErr)
	return p
}

// WaitForText waits for text to appear in an element.
//
// Takes selector (string) which specifies the CSS selector for the element.
// Takes text (string) which specifies the text to wait for.
//
// Returns *Page which allows method chaining for fluent test syntax.
func (p *Page) WaitForText(selector, text string) *Page {
	detail := fmt.Sprintf(fmtKeyValue, selector, text)
	p.beforeAction("WaitForText", detail)
	start := time.Now()
	err := browser_provider_chromedp.WaitForText(p.actionCtx(), selector, text, 5*time.Second)
	p.afterAction("WaitForText", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForText(%q, %q) failed: %v", selector, text, err)
	}
	return p
}

// WaitStable waits until the page content stops changing.
//
// Returns *Page which allows method chaining.
func (p *Page) WaitStable() *Page {
	p.beforeAction("WaitStable", "")
	start := time.Now()
	timedCtx, cancel := context.WithTimeoutCause(
		p.incognitoPage.Ctx, 5*time.Second,
		errors.New("page WaitStable exceeded 5s timeout"),
	)
	defer cancel()
	_ = browser_provider_chromedp.WaitStable(timedCtx, 500*time.Millisecond)
	p.afterAction("WaitStable", "", false, time.Since(start))
	return p
}

// WaitForVisible waits for an element to become visible on the page.
//
// Takes selector (string) which identifies the element to wait for.
// Takes opts (...WaitOption) which sets the wait behaviour.
//
// Returns *Page which allows method chaining.
func (p *Page) WaitForVisible(selector string, opts ...WaitOption) *Page {
	config := defaultWaitConfig()
	for _, opt := range opts {
		opt(&config)
	}

	p.beforeAction("WaitForVisible", selector)
	start := time.Now()
	err := browser_provider_chromedp.WaitForVisible(p.actionCtx(), selector, config.timeout)
	p.afterAction("WaitForVisible", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForVisible(%q) failed: %v", selector, err)
	}
	return p
}

// WaitForNotVisible waits for an element to become hidden or not exist.
//
// Takes selector (string) which identifies the element to wait for.
// Takes opts (...WaitOption) which configures the wait behaviour.
//
// Returns *Page which allows method chaining.
func (p *Page) WaitForNotVisible(selector string, opts ...WaitOption) *Page {
	config := defaultWaitConfig()
	for _, opt := range opts {
		opt(&config)
	}

	p.beforeAction("WaitForNotVisible", selector)
	start := time.Now()
	err := browser_provider_chromedp.WaitForNotVisible(p.actionCtx(), selector, config.timeout)
	p.afterAction("WaitForNotVisible", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForNotVisible(%q) failed: %v", selector, err)
	}
	return p
}

// WaitForEnabled waits for an element to become enabled.
//
// Takes selector (string) which identifies the element to wait for.
// Takes opts (...WaitOption) which provides optional wait behaviour controls.
//
// Returns *Page which allows method chaining for fluent test syntax.
func (p *Page) WaitForEnabled(selector string, opts ...WaitOption) *Page {
	config := defaultWaitConfig()
	for _, opt := range opts {
		opt(&config)
	}

	p.beforeAction("WaitForEnabled", selector)
	start := time.Now()
	err := browser_provider_chromedp.WaitForEnabled(p.actionCtx(), selector, config.timeout)
	p.afterAction("WaitForEnabled", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForEnabled(%q) failed: %v", selector, err)
	}
	return p
}

// WaitForDisabled waits for an element to become disabled.
//
// Takes selector (string) which identifies the element to wait for.
// Takes opts (...WaitOption) which provides optional wait behaviour controls.
//
// Returns *Page which allows method chaining.
func (p *Page) WaitForDisabled(selector string, opts ...WaitOption) *Page {
	config := defaultWaitConfig()
	for _, opt := range opts {
		opt(&config)
	}

	p.beforeAction("WaitForDisabled", selector)
	start := time.Now()
	err := browser_provider_chromedp.WaitForDisabled(p.actionCtx(), selector, config.timeout)
	p.afterAction("WaitForDisabled", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForDisabled(%q) failed: %v", selector, err)
	}
	return p
}

// WaitForNotPresent waits for an element to be removed from the DOM.
//
// Takes selector (string) which identifies the element to wait for removal.
// Takes opts (...WaitOption) which provides optional wait behaviour controls.
//
// Returns *Page which allows method chaining for further page actions.
func (p *Page) WaitForNotPresent(selector string, opts ...WaitOption) *Page {
	config := defaultWaitConfig()
	for _, opt := range opts {
		opt(&config)
	}

	p.beforeAction("WaitForNotPresent", selector)
	start := time.Now()
	err := browser_provider_chromedp.WaitForNotPresent(p.actionCtx(), selector, config.timeout)
	p.afterAction("WaitForNotPresent", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForNotPresent(%q) failed: %v", selector, err)
	}
	return p
}

// Assert returns an assertion builder for the given selector.
//
// Takes selector (string) which specifies the CSS selector to query.
//
// Returns *Assertion which provides methods to verify element properties.
func (p *Page) Assert(selector string) *Assertion {
	return &Assertion{
		t:        p.t,
		page:     p,
		selector: selector,
	}
}

// AssertNoConsoleErrors checks that no errors have appeared in the browser
// console.
//
// Returns *Page which allows method chaining.
func (p *Page) AssertNoConsoleErrors() *Page {
	err := browser_provider_chromedp.CheckNoConsoleErrors(p.actionCtx())
	if err != nil {
		p.t.Errorf("AssertNoConsoleErrors failed: %v", err)
	}
	return p
}

// AssertNoConsoleWarnings checks that no console warnings have occurred.
//
// Returns *Page which allows method chaining.
func (p *Page) AssertNoConsoleWarnings() *Page {
	err := browser_provider_chromedp.CheckNoConsoleWarnings(p.actionCtx())
	if err != nil {
		p.t.Errorf("AssertNoConsoleWarnings failed: %v", err)
	}
	return p
}

// ConsoleLogs returns the captured console logs.
//
// Returns []string which contains the console log messages captured during
// page execution.
func (p *Page) ConsoleLogs() []string {
	return p.pageHelper.ConsoleLogs()
}

// ConsoleLogsWithLevel returns the captured console logs with their levels.
//
// Returns []browser_provider_chromedp.ConsoleLog which contains the log
// messages and their
// severity levels.
func (p *Page) ConsoleLogsWithLevel() []browser_provider_chromedp.ConsoleLog {
	return p.pageHelper.ConsoleLogsWithLevel()
}

// HasConsoleErrors reports whether any error-level console messages were
// logged.
//
// Returns bool which is true if console errors exist.
func (p *Page) HasConsoleErrors() bool {
	return p.pageHelper.HasConsoleErrors()
}

// ConsoleErrors returns all error-level console messages.
//
// Returns []browser_provider_chromedp.ConsoleLog which contains the collected
// error messages.
func (p *Page) ConsoleErrors() []browser_provider_chromedp.ConsoleLog {
	return p.pageHelper.ConsoleErrors()
}

// ClearConsole clears the console log buffer.
//
// Returns *Page which allows method chaining.
func (p *Page) ClearConsole() *Page {
	p.pageHelper.ClearConsoleLogs()
	return p
}

// AssertConsole returns a console assertion builder.
//
// Returns *ConsoleAssertion which provides methods for asserting console
// output.
func (p *Page) AssertConsole() *ConsoleAssertion {
	return &ConsoleAssertion{
		t:    p.t,
		page: p,
	}
}

// ConsoleAssertion provides a fluent API for checking console messages.
type ConsoleAssertion struct {
	// t reports test failures when console assertions fail.
	t testing.TB

	// page holds the Page used for console assertions.
	page *Page
}

// HasMessage asserts that a console message containing the substring was
// logged.
//
// Takes contains (string) which specifies the substring to search for in
// console messages.
//
// Returns *ConsoleAssertion which allows chaining further assertions.
func (c *ConsoleAssertion) HasMessage(contains string) *ConsoleAssertion {
	err := browser_provider_chromedp.CheckConsoleMessage(c.page.actionCtx(), "", contains)
	if err != nil {
		c.t.Errorf("AssertConsole().HasMessage(%q) failed: %v", contains, err)
	}
	return c
}

// HasError asserts that an error-level message containing the substring was
// logged.
//
// Takes contains (string) which is the substring to search for in the message.
//
// Returns *ConsoleAssertion which allows method chaining for further checks.
func (c *ConsoleAssertion) HasError(contains string) *ConsoleAssertion {
	err := browser_provider_chromedp.CheckConsoleMessage(c.page.actionCtx(), "error", contains)
	if err != nil {
		c.t.Errorf("AssertConsole().HasError(%q) failed: %v", contains, err)
	}
	return c
}

// HasWarning checks that a warning message containing the given text was
// logged to the console.
//
// Takes contains (string) which specifies the text to search for in warning
// messages.
//
// Returns *ConsoleAssertion which allows chaining further assertions.
func (c *ConsoleAssertion) HasWarning(contains string) *ConsoleAssertion {
	err := browser_provider_chromedp.CheckConsoleMessage(c.page.actionCtx(), "warn", contains)
	if err != nil {
		c.t.Errorf("AssertConsole().HasWarning(%q) failed: %v", contains, err)
	}
	return c
}

// HasLog asserts that a log-level message containing the substring was logged.
//
// Takes contains (string) which is the substring to search for in log messages.
//
// Returns *ConsoleAssertion which allows method chaining for further
// assertions.
func (c *ConsoleAssertion) HasLog(contains string) *ConsoleAssertion {
	err := browser_provider_chromedp.CheckConsoleMessage(c.page.actionCtx(), "log", contains)
	if err != nil {
		c.t.Errorf("AssertConsole().HasLog(%q) failed: %v", contains, err)
	}
	return c
}

// NoErrors checks that no error-level messages were logged to the console.
//
// Returns *ConsoleAssertion which allows chaining further console assertions.
func (c *ConsoleAssertion) NoErrors() *ConsoleAssertion {
	err := browser_provider_chromedp.CheckNoConsoleErrors(c.page.actionCtx())
	if err != nil {
		c.t.Errorf("AssertConsole().NoErrors() failed: %v", err)
	}
	return c
}

// NoWarnings checks that no warning messages were logged to the console.
//
// Returns *ConsoleAssertion which allows method chaining for more checks.
func (c *ConsoleAssertion) NoWarnings() *ConsoleAssertion {
	err := browser_provider_chromedp.CheckNoConsoleWarnings(c.page.actionCtx())
	if err != nil {
		c.t.Errorf("AssertConsole().NoWarnings() failed: %v", err)
	}
	return c
}

// CaptureDOM returns the HTML content of an element found by a CSS selector.
//
// Takes selector (string) which specifies the CSS selector for the element.
//
// Returns string which contains the normalised HTML of the matched element.
func (p *Page) CaptureDOM(selector string) string {
	html, err := browser_provider_chromedp.CaptureDOM(p.actionCtx(), selector, true)
	if err != nil {
		p.t.Fatalf("CaptureDOM(%q) failed: %v", selector, err)
	}
	return browser_provider_chromedp.NormaliseDOM(html, browser_provider_chromedp.DefaultNormaliseOptions())
}

// MatchGolden compares the normalised HTML of an element against a golden file.
//
// Golden files are stored at testdata/golden/<name>.html relative to the test
// working directory. Set PIKO_UPDATE_GOLDEN=1 to create or update golden files.
//
// Takes selector (string) which specifies the CSS selector of the element.
// Takes name (string) which specifies the golden file name (without extension).
//
// Returns *Page which allows method chaining.
func (p *Page) MatchGolden(selector, name string) *Page {
	p.beforeAction("MatchGolden", selector)
	start := time.Now()

	actual := p.CaptureDOM(selector)
	err := compareGolden(p.t, name, actual)

	p.afterAction("MatchGolden", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Errorf("MatchGolden(%q, %q) failed: %v", selector, name, err)
	}
	return p
}

// ScrollIntoView scrolls an element into the visible area of the page.
//
// Takes selector (string) which identifies the element to scroll into view.
//
// Returns *Page which allows method chaining.
func (p *Page) ScrollIntoView(selector string) *Page {
	p.beforeAction("ScrollIntoView", selector)
	start := time.Now()
	err := browser_provider_chromedp.ScrollIntoView(p.actionCtx().Ctx, selector)
	p.afterAction("ScrollIntoView", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("ScrollIntoView(%q) failed: %v", selector, err)
	}
	return p
}

// GetAttribute returns the value of an attribute on an element.
//
// Takes selector (string) which identifies the element to query.
// Takes name (string) which specifies the attribute to retrieve.
//
// Returns string which contains the attribute value, or an empty string if
// the attribute does not exist.
func (p *Page) GetAttribute(selector, name string) string {
	value, err := browser_provider_chromedp.GetElementAttribute(p.actionCtx().Ctx, selector, name)
	if err != nil {
		p.t.Fatalf("GetAttribute(%q, %q) failed: %v", selector, name, err)
	}
	if value == nil {
		return ""
	}
	return *value
}

// GetAttributes returns all attributes of an element as a map.
//
// Takes selector (string) which identifies the element to query.
//
// Returns map[string]string which contains attribute names and their values.
func (p *Page) GetAttributes(selector string) map[string]string {
	attrs, err := browser_provider_chromedp.GetAllAttributes(p.actionCtx().Ctx, selector)
	if err != nil {
		p.t.Fatalf("GetAttributes(%q) failed: %v", selector, err)
	}
	return attrs
}

// SetAttribute sets an attribute on an element.
//
// Takes selector (string) which identifies the element to change.
// Takes name (string) which specifies the attribute name.
// Takes value (string) which provides the attribute value.
//
// Returns *Page which allows method chaining.
func (p *Page) SetAttribute(selector, name, value string) *Page {
	detail := fmt.Sprintf("%s[%s]=%q", selector, name, value)
	p.beforeAction("SetAttribute", detail)
	start := time.Now()
	err := browser_provider_chromedp.SetElementAttribute(p.actionCtx().Ctx, selector, name, value)
	p.afterAction("SetAttribute", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetAttribute(%q, %q, %q) failed: %v", selector, name, value, err)
	}
	return p
}

// RemoveAttribute removes an attribute from an element.
//
// Takes selector (string) which identifies the element to modify.
// Takes name (string) which specifies the attribute to remove.
//
// Returns *Page which allows method chaining.
func (p *Page) RemoveAttribute(selector, name string) *Page {
	detail := fmt.Sprintf("%s[%s]", selector, name)
	p.beforeAction("RemoveAttribute", detail)
	start := time.Now()
	err := browser_provider_chromedp.RemoveElementAttribute(p.actionCtx().Ctx, selector, name)
	p.afterAction("RemoveAttribute", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("RemoveAttribute(%q, %q) failed: %v", selector, name, err)
	}
	return p
}

// Dimensions represents the position and size of an element.
type Dimensions struct {
	// X is the horizontal position in pixels.
	X float64

	// Y is the vertical position in pixels.
	Y float64

	// Width is the element width in pixels.
	Width float64

	// Height is the element height in pixels.
	Height float64
}

// GetDimensions returns the bounding box of an element.
//
// Takes selector (string) which identifies the element to measure.
//
// Returns *Dimensions which contains the position and size of the element.
func (p *Page) GetDimensions(selector string) *Dimensions {
	dims, err := browser_provider_chromedp.GetElementDimensions(p.actionCtx().Ctx, selector)
	if err != nil {
		p.t.Fatalf("GetDimensions(%q) failed: %v", selector, err)
	}
	return &Dimensions{
		X:      dims.X,
		Y:      dims.Y,
		Width:  dims.Width,
		Height: dims.Height,
	}
}

// Eval runs JavaScript code on the page.
//
// Takes js (string) which contains the JavaScript code to run.
//
// Returns *Page which allows method chaining.
func (p *Page) Eval(js string) *Page {
	displayJS := truncateRunes(js, displayTextMaxLen)
	p.beforeAction("Eval", displayJS)
	start := time.Now()
	err := browser_provider_chromedp.Eval(p.actionCtx(), "", js)
	p.afterAction("Eval", displayJS, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Eval() failed: %v", err)
	}
	return p
}

// EvalReturn runs JavaScript code and returns the result.
//
// Takes js (string) which contains the JavaScript code to run.
//
// Returns any which is the result of running the JavaScript code.
func (p *Page) EvalReturn(js string) any {
	displayJS := truncateRunes(js, displayTextMaxLen)
	p.beforeAction("EvalReturn", displayJS)
	start := time.Now()

	var result any
	err := chromedp.Run(p.incognitoPage.Ctx, chromedp.Evaluate(js, &result))
	p.afterAction("EvalReturn", displayJS, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("EvalReturn() failed: %v", err)
	}
	return result
}

// Pause stops test running and waits for the user to press Enter.
// Useful for quick debugging without enabling full interactive mode.
//
// Returns *Page which allows method chaining.
func (p *Page) Pause() *Page {
	var url string
	_ = chromedp.Run(p.incognitoPage.Ctx, chromedp.Location(&url))
	fmt.Printf("\n[e2e] Paused at: %s\n", url)
	_, _ = fmt.Print("[e2e] Press Enter to continue...")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
	return p
}

// Close releases all resources held by the page.
//
// Call this in a defer statement after creating a new page.
func (p *Page) Close() {
	if p.dialogHandler != nil {
		p.dialogHandler.Stop()
		p.dialogHandler = nil
	}

	if p.interactiveRunner != nil {
		p.interactiveRunner.Close()
	}

	if p.pageHelper != nil {
		p.pageHelper.Close()
	}

	if p.incognitoPage != nil {
		_ = p.incognitoPage.CloseContext()
	}

	if p.outputSandbox != nil {
		_ = p.outputSandbox.Close()
	}
}

// DragAndDrop drags an element from source to target selector.
//
// Takes sourceSelector (string) which identifies the element to drag.
// Takes targetSelector (string) which identifies the drop destination.
//
// Returns *Page which allows method chaining.
func (p *Page) DragAndDrop(sourceSelector, targetSelector string) *Page {
	detail := fmt.Sprintf("%s -> %s", sourceSelector, targetSelector)
	p.beforeAction("DragAndDrop", detail)
	start := time.Now()
	err := browser_provider_chromedp.DragAndDrop(p.actionCtx(), sourceSelector, targetSelector)
	p.afterAction("DragAndDrop", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("DragAndDrop(%q, %q) failed: %v", sourceSelector, targetSelector, err)
	}
	return p
}

// DragTo drags an element to specific coordinates.
//
// Takes sourceSelector (string) which identifies the element to drag.
// Takes targetX (float64) which specifies the destination X coordinate.
// Takes targetY (float64) which specifies the destination Y coordinate.
//
// Returns *Page which allows method chaining.
func (p *Page) DragTo(sourceSelector string, targetX, targetY float64) *Page {
	detail := fmt.Sprintf("%s -> (%v, %v)", sourceSelector, targetX, targetY)
	p.beforeAction("DragTo", detail)
	start := time.Now()
	err := browser_provider_chromedp.DragTo(p.actionCtx(), sourceSelector, targetX, targetY)
	p.afterAction("DragTo", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("DragTo(%q, %v, %v) failed: %v", sourceSelector, targetX, targetY, err)
	}
	return p
}

// DragByOffset drags an element by a set amount from its current position.
//
// Takes selector (string) which identifies the element to drag.
// Takes offsetX (float64) which sets how far to move right, in pixels.
// Takes offsetY (float64) which sets how far to move down, in pixels.
//
// Returns *Page which allows method chaining for fluent test assertions.
func (p *Page) DragByOffset(selector string, offsetX, offsetY float64) *Page {
	detail := fmt.Sprintf("%s by (%v, %v)", selector, offsetX, offsetY)
	p.beforeAction("DragByOffset", detail)
	start := time.Now()
	err := browser_provider_chromedp.DragByOffset(p.actionCtx(), selector, offsetX, offsetY)
	p.afterAction("DragByOffset", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("DragByOffset(%q, %v, %v) failed: %v", selector, offsetX, offsetY, err)
	}
	return p
}

// DragAndDropHTML5 performs an HTML5 drag and drop operation using
// dataTransfer.
//
// Takes sourceSelector (string) which identifies the element to drag.
// Takes targetSelector (string) which identifies the drop target element.
//
// Returns *Page which allows method chaining for further actions.
func (p *Page) DragAndDropHTML5(sourceSelector, targetSelector string) *Page {
	detail := fmt.Sprintf("%s -> %s (HTML5)", sourceSelector, targetSelector)
	p.beforeAction("DragAndDropHTML5", detail)
	start := time.Now()
	err := browser_provider_chromedp.DragAndDropHTML5(p.actionCtx(), sourceSelector, targetSelector)
	p.afterAction("DragAndDropHTML5", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("DragAndDropHTML5(%q, %q) failed: %v", sourceSelector, targetSelector, err)
	}
	return p
}

// enableInteractive enables interactive mode for this page.
//
// Takes useTUI (bool) which selects TUI mode when true, or simple mode when
// false.
func (p *Page) enableInteractive(useTUI bool) {
	if useTUI {
		p.interactiveRunner = NewTUIRunner()
	} else {
		p.interactiveRunner = newSimpleRunner()
	}
	_ = p.interactiveRunner.Start(p.t.Name())
}

// beforeAction runs before each action in interactive mode.
//
// Takes action (string) which names the action about to run.
// Takes detail (string) which gives extra information for the action.
func (p *Page) beforeAction(action, detail string) {
	if p.interactiveRunner == nil {
		return
	}
	p.interactiveRunner.BeforeStep(action, detail)
	p.interactiveRunner.WaitForContinue()
}

// afterAction is called after each action when in interactive mode.
//
// Takes action (string) which identifies the action that was performed.
// Takes detail (string) which provides additional context about the action.
// Takes failed (bool) which indicates whether the action failed.
// Takes duration (time.Duration) which records how long the action took.
func (p *Page) afterAction(action, detail string, failed bool, duration time.Duration) {
	if p.interactiveRunner == nil {
		return
	}
	p.interactiveRunner.AfterStep(action, detail, failed, duration)
}

// actionCtx creates an ActionContext for end-to-end core actions.
//
// Returns *browser_provider_chromedp.ActionContext which holds the context for
// running
// actions.
func (p *Page) actionCtx() *browser_provider_chromedp.ActionContext {
	return &browser_provider_chromedp.ActionContext{
		Ctx:            p.incognitoPage.Ctx,
		SrcSandbox:     p.harness.srcSandbox,
		SandboxFactory: p.harness.opts.sandboxFactory,
		PageHelper:     p.pageHelper,
		ServerURL:      p.baseURL,
	}
}

// recreatePage closes the current page and creates a fresh one.
// Recovers from unresponsive CDP connections.
//
// Returns error when the browser fails to create a new incognito page.
func (p *Page) recreatePage() error {
	if p.pageHelper != nil {
		p.pageHelper.Close()
		p.pageHelper = nil
	}

	if p.incognitoPage != nil {
		_ = p.incognitoPage.CloseContext()
		p.incognitoPage = nil
	}

	incognitoPage, err := p.harness.browser.NewIncognitoPage()
	if err != nil {
		return fmt.Errorf("recreating browser page: %w", err)
	}

	p.incognitoPage = incognitoPage
	p.pageHelper = browser_provider_chromedp.NewPageHelper(incognitoPage.Ctx)

	return nil
}

// isUnresponsivePageError checks if the error shows a stuck CDP connection.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true when the error matches known unresponsive
// patterns.
func isUnresponsivePageError(err error) bool {
	if err == nil {
		return false
	}
	errString := err.Error()
	return strings.Contains(errString, "context deadline exceeded") ||
		strings.Contains(errString, "page unresponsive") ||
		strings.Contains(errString, "CDP stuck") ||
		strings.Contains(errString, "page not responsive")
}
