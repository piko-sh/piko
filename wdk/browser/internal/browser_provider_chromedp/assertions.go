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

package browser_provider_chromedp

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// AssertionError represents a failed assertion and implements the error
// interface.
type AssertionError struct {
	// Selector is the AST path where the assertion failed.
	Selector string

	// Expected holds the value that was expected in the comparison.
	Expected string

	// Actual is the value that was found during the assertion.
	Actual string

	// Message describes what the assertion failed.
	Message string
}

// Error returns the formatted error message for the assertion failure.
//
// Returns string which contains the message, selector, and expected and actual
// values when they are set.
func (e *AssertionError) Error() string {
	if e.Expected != "" && e.Actual != "" {
		return fmt.Sprintf("%s: expected %q, got %q (selector: %s)", e.Message, e.Expected, e.Actual, e.Selector)
	}
	return fmt.Sprintf("%s (selector: %s)", e.Message, e.Selector)
}

// assertionHandler is a function type that runs an assertion step.
type assertionHandler func(ctx *ActionContext, step *BrowserStep) error

// assertionHandlers maps assertion action names to their handlers.
var assertionHandlers = map[string]assertionHandler{
	"checkText": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckText(ctx, step.Selector, step.ExpectedString())
	},
	"checkTextContains": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckTextContains(ctx, step.Selector, step.ExpectedString())
	},
	"checkTextNotContains": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckTextNotContains(ctx, step.Selector, step.ExpectedString())
	},
	"checkValue": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckValue(ctx, step.Selector, step.ExpectedString())
	},
	"checkAttribute":            executeCheckAttribute,
	"checkAttributeContains":    executeCheckAttributeContains,
	"checkAttributeNotContains": executeCheckAttributeNotContains,
	"checkClass": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckClass(ctx, step.Selector, step.ExpectedString())
	},
	"checkStyle": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckStyle(ctx, step.Selector, step.Name, step.ExpectedString())
	},
	"checkFocused":    func(ctx *ActionContext, step *BrowserStep) error { return CheckFocused(ctx, step.Selector) },
	"checkNotFocused": func(ctx *ActionContext, step *BrowserStep) error { return CheckNotFocused(ctx, step.Selector) },
	"checkVisible":    func(ctx *ActionContext, step *BrowserStep) error { return CheckVisible(ctx, step.Selector) },
	"checkHidden":     func(ctx *ActionContext, step *BrowserStep) error { return CheckHidden(ctx, step.Selector) },
	"checkEnabled":    func(ctx *ActionContext, step *BrowserStep) error { return CheckEnabled(ctx, step.Selector) },
	"checkDisabled":   func(ctx *ActionContext, step *BrowserStep) error { return CheckDisabled(ctx, step.Selector) },
	"checkChecked":    func(ctx *ActionContext, step *BrowserStep) error { return CheckChecked(ctx, step.Selector) },
	"checkUnchecked":  func(ctx *ActionContext, step *BrowserStep) error { return CheckUnchecked(ctx, step.Selector) },
	"checkElementCount": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckElementCount(ctx, step.Selector, step.ExpectedInt())
	},
	"checkHTML": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckHTML(ctx, step.Selector, step.ExpectedString())
	},
	"checkFormData": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckFormData(ctx, step.Selector, step.ExpectedMap())
	},
	"checkConsoleMessage": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckConsoleMessage(ctx, step.Level, step.Message)
	},
	"checkNoConsoleMessage": func(ctx *ActionContext, step *BrowserStep) error {
		return CheckNoConsoleMessage(ctx, step.Level, step.Message)
	},
	"checkNoConsoleErrors":   func(ctx *ActionContext, _ *BrowserStep) error { return CheckNoConsoleErrors(ctx) },
	"checkNoConsoleWarnings": func(ctx *ActionContext, _ *BrowserStep) error { return CheckNoConsoleWarnings(ctx) },
}

// DeviceParams holds the settings for device emulation in a browser.
type DeviceParams struct {
	// UserAgent is the HTTP user agent string sent with requests.
	UserAgent string

	// Width specifies the viewport width in pixels.
	Width int64

	// Height is the viewport height in pixels.
	Height int64

	// Scale is the device pixel ratio; 0 defaults to 1.0.
	Scale float64

	// Mobile enables mobile device emulation when set to true.
	Mobile bool
}

// CheckText checks that an element contains the expected text.
//
// Takes ctx (*ActionContext) which provides the browser context for the check.
// Takes selector (string) which identifies the element to look at.
// Takes expected (string) which is the text content to match.
//
// Returns error when the element text does not match within the timeout.
func CheckText(ctx *ActionContext, selector, expected string) error {
	var actualText string

	deadline := time.Now().Add(DefaultAssertionTimeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	for {
		text, found := tryGetElementText(ctx, selector)
		if found {
			actualText = text
			if actualText == expected {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return &AssertionError{
				Selector: selector,
				Expected: expected,
				Actual:   actualText,
				Message:  "text mismatch",
			}
		}

		<-ticker.C
	}
}

// CheckTextContains checks that an element contains the expected text.
//
// Takes ctx (*ActionContext) which provides the browser context for the check.
// Takes selector (string) which identifies the element to inspect.
// Takes substring (string) which specifies the text to search for.
//
// Returns error when the element does not contain the substring within the
// default timeout.
func CheckTextContains(ctx *ActionContext, selector, substring string) error {
	var actualText string

	deadline := time.Now().Add(DefaultAssertionTimeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	for {
		text, found := tryGetElementText(ctx, selector)
		if found {
			actualText = text
			if strings.Contains(actualText, substring) {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return &AssertionError{
				Selector: selector,
				Expected: fmt.Sprintf("contains %q", substring),
				Actual:   actualText,
				Message:  "text does not contain expected substring",
			}
		}

		<-ticker.C
	}
}

// CheckTextNotContains verifies that an element does NOT contain the substring.
//
// Takes ctx (*ActionContext) which provides the browser context for the check.
// Takes selector (string) which identifies the element to inspect.
// Takes substring (string) which specifies the text that should NOT be present.
//
// Returns error when the element contains the substring.
func CheckTextNotContains(ctx *ActionContext, selector, substring string) error {
	text, found := tryGetElementText(ctx, selector)
	if !found {
		return &AssertionError{
			Selector: selector,
			Expected: "element to exist",
			Actual:   "element not found",
			Message:  "element not found",
		}
	}

	if strings.Contains(text, substring) {
		return &AssertionError{
			Selector: selector,
			Expected: fmt.Sprintf("does not contain %q", substring),
			Actual:   text,
			Message:  "text contains unexpected substring",
		}
	}

	return nil
}

// CheckValue checks that an input element has the expected value.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes selector (string) which identifies the input element to check.
// Takes expected (string) which is the value to match against.
//
// Returns error when the element value does not match the expected value
// within the default assertion timeout.
func CheckValue(ctx *ActionContext, selector, expected string) error {
	var actualValue string

	deadline := time.Now().Add(DefaultAssertionTimeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	for {
		value, found := tryGetElementValue(ctx, selector)
		if found {
			actualValue = value
			if actualValue == expected {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return &AssertionError{
				Selector: selector,
				Expected: expected,
				Actual:   actualValue,
				Message:  "value mismatch",
			}
		}

		<-ticker.C
	}
}

// CheckAttribute verifies that an element has the expected attribute value.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to check.
// Takes attributeName (string) which specifies the attribute name to verify.
// Takes expected (string) which is the expected attribute value.
//
// Returns error when the element cannot be found, the attribute cannot be
// read, or the attribute value does not match the expected value.
func CheckAttribute(ctx *ActionContext, selector, attributeName, expected string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	attributeValue, err := GetElementAttribute(ctx.Ctx, selector, attributeName)
	if err != nil {
		return fmt.Errorf("getting attribute %s: %w", attributeName, err)
	}

	if expected == NullAttributeValue {
		if attributeValue != nil {
			return &AssertionError{
				Selector: selector,
				Expected: NullAttributeValue,
				Actual:   *attributeValue,
				Message:  fmt.Sprintf("attribute '%s' should be null", attributeName),
			}
		}
		return nil
	}

	if attributeValue == nil {
		return &AssertionError{
			Selector: selector,
			Expected: expected,
			Actual:   NullAttributeValue,
			Message:  fmt.Sprintf("attribute '%s' should exist", attributeName),
		}
	}

	if *attributeValue != expected {
		return &AssertionError{
			Selector: selector,
			Expected: expected,
			Actual:   *attributeValue,
			Message:  fmt.Sprintf("attribute '%s' mismatch", attributeName),
		}
	}

	return nil
}

// CheckAttributeContains verifies that an element's attribute contains a
// substring.
//
// Takes ctx (*ActionContext) which provides the browser context for the check.
// Takes selector (string) which identifies the element to inspect.
// Takes attributeName (string) which specifies the attribute to examine.
// Takes substring (string) which is the expected substring to find.
//
// Returns error when the element cannot be found, the attribute does not exist,
// or the attribute value does not contain the expected substring.
func CheckAttributeContains(ctx *ActionContext, selector, attributeName, substring string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	attributeValue, err := GetElementAttribute(ctx.Ctx, selector, attributeName)
	if err != nil {
		return fmt.Errorf("getting attribute %s: %w", attributeName, err)
	}

	if attributeValue == nil {
		return &AssertionError{
			Selector: selector,
			Expected: fmt.Sprintf("contains %q", substring),
			Actual:   NullAttributeValue,
			Message:  fmt.Sprintf("attribute '%s' should exist", attributeName),
		}
	}

	if !strings.Contains(*attributeValue, substring) {
		return &AssertionError{
			Selector: selector,
			Expected: fmt.Sprintf("contains %q", substring),
			Actual:   *attributeValue,
			Message:  fmt.Sprintf("attribute '%s' does not contain expected substring", attributeName),
		}
	}

	return nil
}

// CheckAttributeNotContains verifies that an element's attribute does NOT
// contain a substring.
//
// Takes ctx (*ActionContext) which provides the browser context for the check.
// Takes selector (string) which identifies the element to inspect.
// Takes attributeName (string) which specifies the attribute to examine.
// Takes substring (string) which is the substring that should NOT be present.
//
// Returns error when the element cannot be found, or the attribute value
// contains the unexpected substring.
func CheckAttributeNotContains(ctx *ActionContext, selector, attributeName, substring string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	attributeValue, err := GetElementAttribute(ctx.Ctx, selector, attributeName)
	if err != nil {
		return fmt.Errorf("getting attribute %s: %w", attributeName, err)
	}

	if attributeValue == nil {
		return nil
	}

	if strings.Contains(*attributeValue, substring) {
		return &AssertionError{
			Selector: selector,
			Expected: fmt.Sprintf("does not contain %q", substring),
			Actual:   *attributeValue,
			Message:  fmt.Sprintf("attribute '%s' contains unexpected substring", attributeName),
		}
	}

	return nil
}

// CheckClass checks if an element has a given CSS class.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to check.
// Takes className (string) which is the CSS class to look for.
//
// Returns error when the element cannot be found, the class attribute cannot
// be read, or the element does not have the given class.
func CheckClass(ctx *ActionContext, selector, className string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	classAttr, err := GetElementAttribute(ctx.Ctx, selector, "class")
	if err != nil {
		return fmt.Errorf("getting class attribute: %w", err)
	}

	classes := ""
	if classAttr != nil {
		classes = *classAttr
	}

	if !strings.Contains(classes, className) {
		return &AssertionError{
			Selector: selector,
			Expected: fmt.Sprintf("has class %q", className),
			Actual:   classes,
			Message:  "class not found",
		}
	}

	return nil
}

// CheckStyle verifies that an element has a specific computed style value.
//
// Takes ctx (*ActionContext) which provides the browser context for the check.
// Takes selector (string) which identifies the element to inspect.
// Takes property (string) which specifies the CSS property name to check.
// Takes expected (string) which is the expected computed value of the property.
//
// Returns error when the element cannot be found, the style cannot be read,
// or the actual computed value does not match the expected value.
func CheckStyle(ctx *ActionContext, selector, property, expected string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	js := scripts.MustExecute("get_computed_style.js.tmpl", map[string]any{
		"Property": property,
	})
	result, err := EvalOnElement(ctx.Ctx, selector, js)
	if err != nil {
		return fmt.Errorf("getting computed style %s: %w", property, err)
	}

	actual := fmt.Sprintf("%v", result)
	if actual != expected {
		return &AssertionError{
			Selector: selector,
			Expected: expected,
			Actual:   actual,
			Message:  fmt.Sprintf("style '%s' mismatch", property),
		}
	}

	return nil
}

// CheckFocused checks whether the specified element has focus.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to check.
//
// Returns error when the element cannot be found, the focus check fails,
// or the element does not have focus.
func CheckFocused(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	result, err := EvalOnElement(ctx.Ctx, selector, scripts.MustGet("check_focused.js"))
	if err != nil {
		return fmt.Errorf("checking focus: %w", err)
	}

	if b, ok := result.(bool); !ok || !b {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should be focused",
		}
	}

	return nil
}

// CheckNotFocused checks that an element does not have focus.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to check.
//
// Returns error when the element cannot be found, the focus check fails, or
// the element has focus.
func CheckNotFocused(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	result, err := EvalOnElement(ctx.Ctx, selector, scripts.MustGet("check_not_focused.js"))
	if err != nil {
		return fmt.Errorf("checking focus: %w", err)
	}

	if b, ok := result.(bool); !ok || !b {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should not be focused",
		}
	}

	return nil
}

// CheckElementCount verifies the number of elements matching a selector.
// Use expected = -1 to check for "at least one element exists".
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which specifies the CSS selector to match.
// Takes expected (int) which is the expected element count, or -1 for at least
// one.
//
// Returns error when the element count does not match within the timeout.
func CheckElementCount(ctx *ActionContext, selector string, expected int) error {
	deadline := time.Now().Add(DefaultAssertionTimeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	for {
		elements, err := FindElements(ctx.Ctx, selector)
		count := len(elements)

		if elementCountMatches(count, expected, err) {
			return nil
		}

		if time.Now().After(deadline) {
			return newElementCountError(selector, expected, count)
		}

		<-ticker.C
	}
}

// CheckHTML verifies the innerHTML of an element matches the expected value.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to check.
// Takes expected (string) which specifies the expected HTML content.
//
// Returns error when the element cannot be found, the HTML cannot be read,
// or the HTML does not match the expected value.
func CheckHTML(ctx *ActionContext, selector, expected string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	html, err := GetElementHTML(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf("getting HTML: %w", err)
	}

	if html != expected {
		return &AssertionError{
			Selector: selector,
			Expected: expected,
			Actual:   html,
			Message:  "HTML mismatch",
		}
	}

	return nil
}

// CheckFormData checks that a form contains the expected field values.
//
// Takes ctx (*ActionContext) which provides the browser context for evaluation.
// Takes selector (string) which identifies the form element to check.
// Takes expectedFields (map[string]any) which specifies the expected field
// values.
//
// Returns error when the form element cannot be found, the form data cannot be
// read, or any field value does not match the expected value.
func CheckFormData(ctx *ActionContext, selector string, expectedFields map[string]any) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf("finding form element %s: %w", selector, err)
	}

	result, err := EvalOnElement(ctx.Ctx, selector, scripts.MustGet("get_form_data.js"))
	if err != nil {
		return fmt.Errorf("getting form data: %w", err)
	}

	actualFormData, ok := result.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected form data result type: %T", result)
	}

	for key, expectedValue := range expectedFields {
		actualValue := actualFormData[key]
		if actualValue != expectedValue {
			return &AssertionError{
				Selector: selector,
				Expected: fmt.Sprintf("%v", expectedValue),
				Actual:   fmt.Sprintf("%v", actualValue),
				Message:  fmt.Sprintf("form field '%s' mismatch", key),
			}
		}
	}

	return nil
}

// CheckExists verifies that an element matching the given selector exists.
//
// Takes ctx (*ActionContext) which provides the browser context for element
// lookup.
// Takes selector (string) which specifies the CSS selector to find.
//
// Returns error when no element matches the selector.
func CheckExists(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should exist",
		}
	}
	return nil
}

// CheckNotExists checks that no element matches the given selector.
//
// Takes ctx (*ActionContext) which provides the browser context for the check.
// Takes selector (string) which specifies the CSS selector to search for.
//
// Returns error when an element is found, as an AssertionError.
func CheckNotExists(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err == nil {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should not exist",
		}
	}
	return nil
}

// CheckVisible checks that an element is visible on the page.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to check.
//
// Returns error when the element cannot be found, visibility cannot be
// checked, or the element is not visible.
func CheckVisible(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	visible, err := IsElementVisible(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf("checking visibility: %w", err)
	}

	if !visible {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should be visible",
		}
	}

	return nil
}

// CheckHidden checks that an element is not visible on the page.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to check.
//
// Returns error when the visibility check fails or the element is visible.
func CheckHidden(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return nil
	}

	visible, err := IsElementVisible(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf("checking visibility: %w", err)
	}

	if visible {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should be hidden",
		}
	}

	return nil
}

// CheckEnabled checks that an element is enabled.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to check.
//
// Returns error when the element cannot be found, when the enabled state
// cannot be checked, or when the element is not enabled.
func CheckEnabled(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	enabled, err := IsElementEnabled(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf("checking enabled state: %w", err)
	}

	if !enabled {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should be enabled",
		}
	}

	return nil
}

// CheckDisabled verifies that an element is disabled.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to check.
//
// Returns error when the element cannot be found, its enabled state cannot be
// determined, or the element is enabled when it should be disabled.
func CheckDisabled(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	enabled, err := IsElementEnabled(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf("checking enabled state: %w", err)
	}

	if enabled {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should be disabled",
		}
	}

	return nil
}

// CheckChecked verifies that a checkbox or radio element is checked.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to verify.
//
// Returns error when the element cannot be found, its checked state cannot be
// read, or the element is not checked.
func CheckChecked(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	checked, err := IsElementChecked(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf("checking checked state: %w", err)
	}

	if !checked {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should be checked",
		}
	}

	return nil
}

// CheckUnchecked checks that a checkbox or radio button is not selected.
//
// Takes ctx (*ActionContext) which provides the browser context for the check.
// Takes selector (string) which identifies the element to check.
//
// Returns error when the element cannot be found, its state cannot be read, or
// the element is selected when it should not be.
func CheckUnchecked(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	checked, err := IsElementChecked(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf("checking checked state: %w", err)
	}

	if checked {
		return &AssertionError{
			Selector: selector,
			Expected: "",
			Actual:   "",
			Message:  "element should be unchecked",
		}
	}

	return nil
}

// CheckConsoleMessage verifies that a console message was logged.
//
// If level is empty, matches any level. If messageContains is empty,
// matches any message.
//
// Takes ctx (*ActionContext) which provides access to the page helper.
// Takes level (string) which specifies the log level to match, or empty
// for any level.
// Takes messageContains (string) which specifies a substring to find in
// the message, or empty for any message.
//
// Returns error when no matching console message is found or the page
// helper is nil.
func CheckConsoleMessage(ctx *ActionContext, level, messageContains string) error {
	if ctx.PageHelper == nil {
		return errors.New("checkConsoleMessage requires PageHelper in ActionContext")
	}

	logs := ctx.PageHelper.ConsoleLogsWithLevel()

	for _, log := range logs {
		levelMatches := level == "" || log.Level == level
		messageMatches := messageContains == "" || strings.Contains(log.Message, messageContains)

		if levelMatches && messageMatches {
			return nil
		}
	}

	return &AssertionError{
		Selector: "",
		Expected: fmt.Sprintf("console message (level=%q, contains=%q)", level, messageContains),
		Actual:   fmt.Sprintf("%d console messages", len(logs)),
		Message:  "console message not found",
	}
}

// CheckNoConsoleMessage checks that no console message matches the given
// criteria.
//
// When level is empty, checks all levels. When messageContains is empty,
// checks all messages.
//
// Takes ctx (*ActionContext) which provides the page helper for console access.
// Takes level (string) which filters by log level, or empty for all levels.
// Takes messageContains (string) which filters by message content, or empty
// for all messages.
//
// Returns error when a matching console message is found or when PageHelper
// is nil.
func CheckNoConsoleMessage(ctx *ActionContext, level, messageContains string) error {
	if ctx.PageHelper == nil {
		return errors.New("checkNoConsoleMessage requires PageHelper in ActionContext")
	}

	logs := ctx.PageHelper.ConsoleLogsWithLevel()

	for _, log := range logs {
		levelMatches := level == "" || log.Level == level
		messageMatches := messageContains == "" || strings.Contains(log.Message, messageContains)

		if levelMatches && messageMatches {
			return &AssertionError{
				Selector: "",
				Expected: fmt.Sprintf("no console message (level=%q, contains=%q)", level, messageContains),
				Actual:   fmt.Sprintf("found: [%s] %s", log.Level, log.Message),
				Message:  "unexpected console message found",
			}
		}
	}

	return nil
}

// CheckNoConsoleErrors verifies that no error-level console messages were
// logged.
//
// Takes ctx (*ActionContext) which provides the browser context to check.
//
// Returns error when an error-level console message is found.
func CheckNoConsoleErrors(ctx *ActionContext) error {
	return CheckNoConsoleMessage(ctx, "error", "")
}

// CheckNoConsoleWarnings verifies that no warning-level console messages were
// logged.
//
// Takes ctx (*ActionContext) which provides the browser context to check.
//
// Returns error when a warning-level console message is found.
func CheckNoConsoleWarnings(ctx *ActionContext) error {
	return CheckNoConsoleMessage(ctx, "warn", "")
}

// ExecuteAssertion runs an assertion step against the browser.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which specifies the assertion to run.
//
// Returns error when the assertion action is unknown or the handler fails.
func ExecuteAssertion(ctx *ActionContext, step *BrowserStep) error {
	handler, ok := assertionHandlers[step.Action]
	if !ok {
		return fmt.Errorf("unknown assertion action: %s", step.Action)
	}
	return handler(ctx, step)
}

// IsAssertionAction reports whether the given action is an assertion action.
//
// Takes action (string) which is the action name to check.
//
// Returns bool which is true if the action is an assertion action.
//
// Note: The captureDOM action is not included here as it requires special
// handling by test harnesses for golden file comparison.
func IsAssertionAction(action string) bool {
	switch action {
	case "checkText", "checkTextNotContains", "checkTextContains",
		"checkValue", "checkAttribute", "checkAttributeContains", "checkAttributeNotContains",
		"checkClass", "checkStyle",
		"checkFocused", "checkNotFocused", "checkVisible", "checkHidden",
		"checkEnabled", "checkDisabled", "checkChecked", "checkUnchecked",
		"checkElementCount", "checkHTML", "checkFormData",
		"checkConsoleMessage", "checkNoConsoleMessage",
		"checkNoConsoleErrors", "checkNoConsoleWarnings":
		return true
	default:
		return false
	}
}

// CaptureDOM captures the outer HTML of an element for golden file comparison.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which identifies the target element using CSS syntax.
// Takes includeShadowRoots (bool) which, when true, serialises shadow DOM
// content as <template shadowrootmode="open"> elements inside the captured
// HTML using the browser's getHTML API.
//
// Returns string which contains the outer HTML of the matched element.
// Returns error when the selector times out or the DOM capture fails.
func CaptureDOM(ctx *ActionContext, selector string, includeShadowRoots bool) (string, error) {
	if err := WaitForSelector(ctx, selector, DefaultAssertionTimeout); err != nil {
		return "", fmt.Errorf("waiting for selector %s: %w", selector, err)
	}

	if !includeShadowRoots {
		var html string
		err := chromedp.Run(ctx.Ctx,
			chromedp.OuterHTML(selector, &html, chromedp.ByQuery),
		)
		if err != nil {
			return "", fmt.Errorf("capturing DOM of %s: %w", selector, err)
		}
		return html, nil
	}

	var html string
	js := scripts.MustExecute("capture_dom_with_shadow.js.tmpl", map[string]any{
		"Selector": selector,
	})

	err := chromedp.Run(ctx.Ctx,
		chromedp.Evaluate(js, &html),
	)
	if err != nil {
		return "", fmt.Errorf("capturing DOM with shadow roots of %s: %w", selector, err)
	}

	return html, nil
}

// ScreenshotElement captures a PNG screenshot of an element.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to capture.
//
// Returns []byte which contains the PNG image data.
// Returns error when the element is not found or the screenshot fails.
func ScreenshotElement(ctx *ActionContext, selector string) ([]byte, error) {
	if err := WaitForSelector(ctx, selector, DefaultAssertionTimeout); err != nil {
		return nil, fmt.Errorf("waiting for selector %s: %w", selector, err)
	}

	var buffer []byte
	err := chromedp.Run(ctx.Ctx,
		chromedp.Screenshot(selector, &buffer, chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("taking screenshot of %s: %w", selector, err)
	}

	return buffer, nil
}

// ScreenshotViewport captures a PNG screenshot of the current viewport.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// screenshot operation.
//
// Returns []byte which contains the PNG image data.
// Returns error when the screenshot cannot be captured.
func ScreenshotViewport(ctx *ActionContext) ([]byte, error) {
	var buffer []byte
	err := chromedp.Run(ctx.Ctx,
		chromedp.CaptureScreenshot(&buffer),
	)
	if err != nil {
		return nil, fmt.Errorf("taking viewport screenshot: %w", err)
	}

	return buffer, nil
}

// ScreenshotFull captures a PNG screenshot of the entire page, scrolling if
// necessary.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// screenshot operation.
//
// Returns []byte which contains the PNG image data.
// Returns error when the screenshot operation fails.
func ScreenshotFull(ctx *ActionContext) ([]byte, error) {
	var buffer []byte
	err := chromedp.Run(ctx.Ctx,
		chromedp.FullScreenshot(&buffer, ScreenshotQualityFull),
	)
	if err != nil {
		return nil, fmt.Errorf("taking full page screenshot: %w", err)
	}

	return buffer, nil
}

// SetViewport sets the viewport dimensions.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes width (int64) which specifies the viewport width in pixels.
// Takes height (int64) which specifies the viewport height in pixels.
//
// Returns error when the viewport cannot be set.
func SetViewport(ctx *ActionContext, width, height int64) error {
	err := chromedp.Run(ctx.Ctx,
		chromedp.EmulateViewport(width, height),
	)
	if err != nil {
		return fmt.Errorf("setting viewport to %dx%d: %w", width, height, err)
	}
	return nil
}

// EmulateDevice emulates a device with the given parameters.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes params (DeviceParams) which specifies the device dimensions and scale.
//
// Returns error when the viewport emulation fails.
func EmulateDevice(ctx *ActionContext, params DeviceParams) error {
	scale := params.Scale
	if scale == 0 {
		scale = 1.0
	}

	opts := []chromedp.EmulateViewportOption{
		chromedp.EmulateScale(scale),
	}

	if params.Mobile {
		opts = append(opts, chromedp.EmulateLandscape)
	}

	err := chromedp.Run(ctx.Ctx,
		chromedp.EmulateViewport(params.Width, params.Height, opts...),
	)
	if err != nil {
		return fmt.Errorf("emulating device %dx%d: %w", params.Width, params.Height, err)
	}
	return nil
}

// ResetEmulation resets device emulation to defaults.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// operation.
//
// Returns error when the emulation reset fails.
func ResetEmulation(ctx *ActionContext) error {
	err := chromedp.Run(ctx.Ctx,
		chromedp.EmulateReset(),
	)
	if err != nil {
		return fmt.Errorf("resetting emulation: %w", err)
	}
	return nil
}

// tryGetElementText retrieves the text content of an element by selector.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to query.
//
// Returns string which is the trimmed text content of the element.
// Returns bool which is true if the element was found, false otherwise.
func tryGetElementText(ctx *ActionContext, selector string) (string, bool) {
	text, err := GetElementText(ctx.Ctx, selector)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(text), true
}

// tryGetElementValue gets the value property of an input element.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes selector (string) which identifies the target input element.
//
// Returns string which contains the element value if found.
// Returns bool which is true if the value was retrieved, false otherwise.
func tryGetElementValue(ctx *ActionContext, selector string) (string, bool) {
	value, err := GetElementValue(ctx.Ctx, selector)
	if err != nil {
		return "", false
	}
	return value, true
}

// elementCountMatches checks if the element count matches the expected value.
// A value of -1 for expected means at least one element exists.
//
// Takes count (int) which is the actual number of elements found.
// Takes expected (int) which is the required count, or -1 for at least one.
// Takes err (error) which shows if the element lookup failed.
//
// Returns bool which is true when the count meets the expected condition.
func elementCountMatches(count, expected int, err error) bool {
	if err != nil {
		return expected == 0
	}
	if expected == -1 {
		return count >= 1
	}
	return count == expected
}

// newElementCountError creates an AssertionError for element count mismatch.
//
// Takes selector (string) which identifies the element being checked.
// Takes expected (int) which specifies the expected count, or -1 for at least one.
// Takes actual (int) which specifies the actual count found.
//
// Returns *AssertionError which describes the count mismatch.
func newElementCountError(selector string, expected, actual int) *AssertionError {
	expectedString := fmt.Sprintf("%d elements", expected)
	if expected == -1 {
		expectedString = "at least 1 element"
	}
	return &AssertionError{
		Selector: selector,
		Expected: expectedString,
		Actual:   fmt.Sprintf("%d elements", actual),
		Message:  "element count mismatch",
	}
}

// executeCheckAttribute handles the checkAttribute action with conditional
// logic.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which defines the attribute check to perform.
//
// Returns error when the attribute check fails.
func executeCheckAttribute(ctx *ActionContext, step *BrowserStep) error {
	if step.Contains != "" {
		return CheckAttributeContains(ctx, step.Selector, step.AttributeName(), step.Contains)
	}
	return CheckAttribute(ctx, step.Selector, step.AttributeName(), step.ExpectedString())
}

// executeCheckAttributeContains runs the checkAttributeContains action.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which defines the attribute check to perform.
//
// Returns error when the attribute does not contain the expected substring.
func executeCheckAttributeContains(ctx *ActionContext, step *BrowserStep) error {
	return CheckAttributeContains(ctx, step.Selector, step.AttributeName(), step.ExpectedString())
}

// executeCheckAttributeNotContains checks that an element attribute does not
// contain a given substring.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which defines the attribute check to perform.
//
// Returns error when the attribute contains the unwanted substring.
func executeCheckAttributeNotContains(ctx *ActionContext, step *BrowserStep) error {
	return CheckAttributeNotContains(ctx, step.Selector, step.AttributeName(), step.ExpectedString())
}
