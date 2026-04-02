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
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
	"piko.sh/piko/wdk/json"
	"piko.sh/piko/wdk/safedisk"
)

// errElementNotFoundFormat is the format string for element-not-found errors.
const errElementNotFoundFormat = "element not found: %s"

// ActionContext holds the context needed for browser action execution.
type ActionContext struct {
	// Ctx is the chromedp browser context for executing actions.
	Ctx context.Context

	// SrcSandbox provides sandboxed access to the source directory for resolving
	// relative file paths with path traversal protection.
	SrcSandbox safedisk.Sandbox

	// SandboxFactory creates sandboxes for filesystem access. When nil,
	// safedisk.NewNoOpSandbox is used as a fallback.
	SandboxFactory safedisk.Factory

	// PageHelper provides console log access for assertions; nil if unavailable.
	PageHelper *PageHelper

	// ServerURL is the base URL of the server, prepended to paths for navigation.
	ServerURL string
}

// clickablePosition holds the centre coordinates of an element and whether it
// is currently receiving pointer events (not obscured by an overlay such as a
// view transition).
type clickablePosition struct {
	// X is the horizontal centre of the element in viewport coordinates.
	X float64

	// Y is the vertical centre of the element in viewport coordinates.
	Y float64

	// Found indicates whether the element exists in the DOM.
	Found bool

	// Clickable indicates whether the element is the topmost element at its
	// centre coordinates, meaning a CDP mouse event will reach it.
	Clickable bool
}

// BoolConditionChecker checks a boolean condition on an element.
type BoolConditionChecker func(ctx context.Context, selector string) (bool, error)

// Click clicks an element by selector.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes selector (string) which identifies the element to click.
//
// Returns error when the element cannot be clicked.
func Click(ctx *ActionContext, selector string) error {
	if strings.Contains(selector, ShadowDOMSeparator) {
		return clickInShadowDOM(ctx.Ctx, selector)
	}

	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("browser Click exceeded %s timeout", DefaultActionTimeout))
	defer cancel()

	position, err := pollForClickableElement(timedCtx, selector)
	if err != nil {
		return fmt.Errorf("clicking element %s: %w", selector, err)
	}

	if err := dispatchMouseClick(timedCtx, position.X, position.Y, 1); err != nil {
		return fmt.Errorf("clicking element %s: %w", selector, err)
	}
	return nil
}

// DoubleClick double-clicks an element by selector.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes selector (string) which identifies the element to double-click.
//
// Returns error when the element cannot be found or the click fails.
func DoubleClick(ctx *ActionContext, selector string) error {
	if strings.Contains(selector, ShadowDOMSeparator) {
		return doubleClickInShadowDOM(ctx.Ctx, selector)
	}

	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("browser DoubleClick exceeded %s timeout", DefaultActionTimeout))
	defer cancel()

	position, err := pollForClickableElement(timedCtx, selector)
	if err != nil {
		return fmt.Errorf("double-clicking element %s: %w", selector, err)
	}

	if err := dispatchMouseClick(timedCtx, position.X, position.Y, 1); err != nil {
		return fmt.Errorf("double-clicking element %s: %w", selector, err)
	}
	if err := dispatchMouseClick(timedCtx, position.X, position.Y, 2); err != nil {
		return fmt.Errorf("double-clicking element %s: %w", selector, err)
	}
	return nil
}

// pollForClickableElement polls until the element identified by selector
// becomes clickable or the context expires.
//
// Takes ctx (context.Context) which controls the polling deadline.
// Takes selector (string) which identifies the target element.
//
// Returns clickablePosition which holds the centre coordinates of the element.
// Returns error when the element is not found or the context expires.
func pollForClickableElement(ctx context.Context, selector string) (clickablePosition, error) {
	js := scripts.MustExecute("click_element.js.tmpl", map[string]any{
		"Selector": selector,
	})

	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	for {
		var position clickablePosition
		if err := chromedp.Run(ctx, chromedp.Evaluate(js, &position)); err != nil {
			return clickablePosition{}, err
		}
		if !position.Found {
			return clickablePosition{}, fmt.Errorf(errElementNotFoundFormat, selector)
		}
		if position.Clickable {
			return position, nil
		}

		select {
		case <-ctx.Done():
			return clickablePosition{}, fmt.Errorf("timed out waiting for element %s to become clickable", selector)
		case <-ticker.C:
		}
	}
}

// Hover moves the mouse over an element.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes selector (string) which identifies the target element.
//
// Returns error when the element cannot be found or the hover action fails.
func Hover(ctx *ActionContext, selector string) error {
	if strings.Contains(selector, ShadowDOMSeparator) {
		return hoverInShadowDOM(ctx.Ctx, selector)
	}

	js := scripts.MustExecute("hover_element.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var hovered bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &hovered))
	if err != nil {
		return fmt.Errorf("hovering over element %s: %w", selector, err)
	}
	if !hovered {
		return fmt.Errorf(errElementNotFoundFormat, selector)
	}
	return nil
}

// RightClick performs a right-click (context menu) on an element.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which specifies the element to right-click, with
// support for shadow DOM selectors.
//
// Returns error when the element cannot be found or the click fails.
func RightClick(ctx *ActionContext, selector string) error {
	if strings.Contains(selector, ShadowDOMSeparator) {
		return rightClickInShadowDOM(ctx.Ctx, selector)
	}

	js := scripts.MustExecute("right_click_element.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var clicked bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &clicked))
	if err != nil {
		return fmt.Errorf("right-clicking element %s: %w", selector, err)
	}
	if !clicked {
		return fmt.Errorf(errElementNotFoundFormat, selector)
	}
	return nil
}

// Fill sets the value of an input element, simulating typing.
// If value is empty, this clears the input using the Clear function.
//
// Takes ctx (*ActionContext) which provides the browser context for actions.
// Takes selector (string) which identifies the target input element.
// Takes value (string) which specifies the text to type into the element.
//
// Returns error when focusing, selecting, typing, or dispatching events fails.
func Fill(ctx *ActionContext, selector, value string) error {
	if value == "" {
		return Clear(ctx, selector)
	}

	if strings.Contains(selector, ShadowDOMSeparator) {
		return fillInShadowDOM(ctx.Ctx, selector, value)
	}

	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("browser Fill exceeded %s timeout", DefaultActionTimeout))
	defer cancel()

	focusJS := scripts.MustExecute("focus.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var focused bool
	if err := chromedp.Run(timedCtx, chromedp.Evaluate(focusJS, &focused)); err != nil {
		return fmt.Errorf("focusing element %s: %w", selector, err)
	}
	if !focused {
		return fmt.Errorf(errElementNotFoundFormat, selector)
	}

	selectJS := scripts.MustExecute("select_all_text.js.tmpl", map[string]any{
		"Selector": selector,
	})
	if err := chromedp.Run(timedCtx, chromedp.Evaluate(selectJS, nil)); err != nil {
		return fmt.Errorf("selecting text on %s: %w", selector, err)
	}

	if err := chromedp.Run(timedCtx, chromedp.ActionFunc(func(actionCtx context.Context) error {
		return input.InsertText(value).Do(actionCtx)
	})); err != nil {
		return fmt.Errorf("typing value on %s: %w", selector, err)
	}

	inputJS := scripts.MustExecute("dispatch_input_event.js.tmpl", map[string]any{
		"Selector": selector,
	})
	if err := chromedp.Run(timedCtx, chromedp.Evaluate(inputJS, nil)); err != nil {
		return fmt.Errorf("dispatching input event on %s: %w", selector, err)
	}

	return nil
}

// Clear clears the value of an input element by selecting all and deleting.
// This simulates user keyboard input rather than directly manipulating the DOM.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which identifies the input element to clear.
//
// Returns error when the element cannot be focused, text cannot be selected,
// or the deletion fails.
func Clear(ctx *ActionContext, selector string) error {
	if strings.Contains(selector, ShadowDOMSeparator) {
		return clearInShadowDOM(ctx.Ctx, selector)
	}

	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("browser Clear exceeded %s timeout", DefaultActionTimeout))
	defer cancel()

	focusJS := scripts.MustExecute("focus.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var focused bool
	if err := chromedp.Run(timedCtx, chromedp.Evaluate(focusJS, &focused)); err != nil {
		return fmt.Errorf("focusing element for clear %s: %w", selector, err)
	}
	if !focused {
		return fmt.Errorf(errElementNotFoundFormat, selector)
	}

	selectJS := scripts.MustExecute("select_all_text.js.tmpl", map[string]any{
		"Selector": selector,
	})
	if err := chromedp.Run(timedCtx, chromedp.Evaluate(selectJS, nil)); err != nil {
		return fmt.Errorf("selecting text on %s: %w", selector, err)
	}

	backspaceKey := keyMap["Backspace"]
	if err := chromedp.Run(timedCtx, chromedp.ActionFunc(func(actionCtx context.Context) error {
		if err := input.DispatchKeyEvent(input.KeyDown).
			WithKey(backspaceKey.Key).
			WithCode(backspaceKey.Code).
			WithWindowsVirtualKeyCode(backspaceKey.KeyCode).
			WithNativeVirtualKeyCode(backspaceKey.KeyCode).
			Do(actionCtx); err != nil {
			return err
		}
		return input.DispatchKeyEvent(input.KeyUp).
			WithKey(backspaceKey.Key).
			WithCode(backspaceKey.Code).
			WithWindowsVirtualKeyCode(backspaceKey.KeyCode).
			WithNativeVirtualKeyCode(backspaceKey.KeyCode).
			Do(actionCtx)
	})); err != nil {
		return fmt.Errorf("deleting text on %s: %w", selector, err)
	}

	inputJS := scripts.MustExecute("dispatch_input_event.js.tmpl", map[string]any{
		"Selector": selector,
	})
	_ = chromedp.Run(timedCtx, chromedp.Evaluate(inputJS, nil))

	return nil
}

// Submit submits a form element.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which identifies the form element to submit.
//
// Returns error when the form cannot be submitted or no form matches the
// selector.
func Submit(ctx *ActionContext, selector string) error {
	if strings.Contains(selector, ShadowDOMSeparator) {
		return submitInShadowDOM(ctx.Ctx, selector)
	}

	js := scripts.MustExecute("submit_form.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var submitted bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &submitted))
	if err != nil {
		return fmt.Errorf("submitting form %s: %w", selector, err)
	}
	if !submitted {
		return fmt.Errorf("no form found for selector: %s", selector)
	}
	return nil
}

// Check checks a checkbox or radio input.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which identifies the element, supporting shadow DOM.
//
// Returns error when the element cannot be found, is not a checkbox or radio,
// or the operation fails.
func Check(ctx *ActionContext, selector string) error {
	if strings.Contains(selector, ShadowDOMSeparator) {
		return checkInShadowDOM(ctx.Ctx, selector)
	}

	js := scripts.MustExecute("check_checkbox.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var checked bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &checked))
	if err != nil {
		return fmt.Errorf("checking element %s: %w", selector, err)
	}
	if !checked {
		return fmt.Errorf("element is not a checkbox/radio or not found: %s", selector)
	}
	return nil
}

// Uncheck unchecks a checkbox input.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which identifies the checkbox element to uncheck.
//
// Returns error when the element cannot be unchecked, is not a checkbox, or
// is not found.
func Uncheck(ctx *ActionContext, selector string) error {
	if strings.Contains(selector, ShadowDOMSeparator) {
		return uncheckInShadowDOM(ctx.Ctx, selector)
	}

	js := scripts.MustExecute("uncheck_checkbox.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var unchecked bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &unchecked))
	if err != nil {
		return fmt.Errorf("unchecking element %s: %w", selector, err)
	}
	if !unchecked {
		return fmt.Errorf("element is not a checkbox or not found: %s", selector)
	}
	return nil
}

// SetFiles sets files on a file input element.
//
// Takes ctx (*ActionContext) which provides the browser context and source
// directory for resolving file paths.
// Takes selector (string) which identifies the file input element, supporting
// shadow DOM selectors.
// Takes filePaths ([]string) which lists relative file paths to set on the
// input.
//
// Returns error when a file does not exist or the upload fails.
func SetFiles(ctx *ActionContext, selector string, filePaths []string) error {
	absolutePaths := make([]string, len(filePaths))
	for i, relPath := range filePaths {
		if _, err := ctx.SrcSandbox.Stat(relPath); err != nil {
			return fmt.Errorf("file not found: %s (resolved to %s)", relPath,
				filepath.Join(ctx.SrcSandbox.Root(), relPath))
		}
		absolutePaths[i] = filepath.Join(ctx.SrcSandbox.Root(), relPath)
	}

	if strings.Contains(selector, ShadowDOMSeparator) {
		return setFilesInShadowDOM(ctx.Ctx, selector, absolutePaths)
	}

	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("browser SetFiles exceeded %s timeout", DefaultActionTimeout))
	defer cancel()

	err := chromedp.Run(timedCtx,
		chromedp.SetUploadFiles(selector, absolutePaths, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("setting files on %s: %w", selector, err)
	}

	js := scripts.MustExecute("dispatch_change_event.js.tmpl", map[string]any{
		"Selector": selector,
	})
	_ = chromedp.Run(timedCtx, chromedp.Evaluate(js, nil))

	return nil
}

// Focus sets keyboard focus on the element matching the given selector.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which specifies the element to focus, supporting
// shadow DOM paths separated by ShadowDOMSeparator.
//
// Returns error when the element cannot be focused or does not exist.
func Focus(ctx *ActionContext, selector string) error {
	return runTimedJSAction(ctx, selector, focusInShadowDOM, "Focus", "focus.js.tmpl", "focusing")
}

// Blur removes focus from an element.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to blur.
//
// Returns error when the element cannot be blurred.
func Blur(ctx *ActionContext, selector string) error {
	return runTimedJSAction(ctx, selector, blurInShadowDOM, "Blur", "blur.js.tmpl", "blurring")
}

// runTimedJSAction executes a JavaScript template action
// with a timeout, falling back to a shadow DOM handler when
// the selector contains a shadow DOM separator.
//
// Takes ctx (*ActionContext) which provides the browser
// context for execution.
// Takes selector (string) which identifies the target
// element.
// Takes shadowFallback (func) which handles shadow DOM
// selectors.
// Takes actionName (string) which labels the action for
// timeout error messages.
// Takes templateName (string) which identifies the JS
// template to execute.
// Takes verb (string) which describes the action for error
// messages.
//
// Returns error when the element is not found or the action
// fails.
func runTimedJSAction(
	ctx *ActionContext,
	selector string,
	shadowFallback func(context.Context, string) error,
	actionName string,
	templateName string,
	verb string,
) error {
	if strings.Contains(selector, ShadowDOMSeparator) {
		return shadowFallback(ctx.Ctx, selector)
	}

	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("browser %s exceeded %s timeout", actionName, DefaultActionTimeout))
	defer cancel()

	js := scripts.MustExecute(templateName, map[string]any{
		"Selector": selector,
	})

	var found bool
	err := chromedp.Run(timedCtx, chromedp.Evaluate(js, &found))
	if err != nil {
		return fmt.Errorf("%s element %s: %w", verb, selector, err)
	}
	if !found {
		return fmt.Errorf(errElementNotFoundFormat, selector)
	}
	return nil
}

// Scroll scrolls the window or an element to a position.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes selector (string) which identifies the element to scroll, or "window"
// for the viewport.
// Takes position (string) which specifies the target scroll position.
//
// Returns error when the scroll operation fails.
func Scroll(ctx *ActionContext, selector string, position string) error {
	var js string
	if selector == "window" || selector == "" {
		js = scripts.MustExecute("window_scroll_to.js.tmpl", map[string]any{
			"Position": position,
		})
	} else if strings.Contains(selector, ShadowDOMSeparator) {
		parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
		js = scripts.MustExecute("shadow_scroll_element.js.tmpl", map[string]any{
			"Host":     parts[0],
			"Shadow":   parts[1],
			"Position": position,
		})
	} else {
		js = scripts.MustExecute("scroll_element.js.tmpl", map[string]any{
			"Selector": selector,
			"Position": position,
		})
	}

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("scrolling: %w", err)
	}
	return nil
}

// Wait pauses execution for a specified duration.
//
// Takes ms (int) which is the number of milliseconds to wait.
func Wait(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// WaitForSelector waits for an element matching the selector to appear.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which specifies the CSS selector to wait for.
// Takes timeout (time.Duration) which sets the maximum wait time.
//
// Returns error when the element does not appear within the timeout.
func WaitForSelector(ctx *ActionContext, selector string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		_, err := FindElement(ctx.Ctx, selector)
		if err == nil {
			return nil
		}
		lastErr = err

		if time.Now().After(deadline) {
			currentURL := getCurrentURL(ctx.Ctx)
			bodyHTML := getBodyPreview(ctx.Ctx)
			return fmt.Errorf("timed out waiting for selector '%s' after %v (url: %s, body preview: %s, last error: %w)",
				selector, timeout, currentURL, bodyHTML, lastErr)
		}

		<-ticker.C
	}
}

// WaitForText waits for specific text to appear in an element.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the target element.
// Takes text (string) which specifies the expected text content.
// Takes timeout (time.Duration) which sets the maximum wait time.
//
// Returns error when the timeout expires before the text appears.
func WaitForText(ctx *ActionContext, selector, text string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	for {
		actualText, err := GetElementText(ctx.Ctx, selector)
		if err == nil && actualText == text {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for text '%s' in '%s'", text, selector)
		}

		<-ticker.C
	}
}

// DispatchEvent dispatches a custom event on an element or document.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which identifies the target element, or "*" or
// "document" for document-level events.
// Takes eventName (string) which specifies the custom event name to dispatch.
// Takes detail (map[string]any) which contains the event detail payload.
//
// Returns error when the detail cannot be marshalled or the event dispatch
// fails.
func DispatchEvent(ctx *ActionContext, selector, eventName string, detail map[string]any) error {
	detailJSON, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("marshalling event detail: %w", err)
	}

	var js string
	if selector == "*" || selector == "document" {
		js = scripts.MustExecute("dispatch_document_event.js.tmpl", map[string]any{
			"EventName":  eventName,
			"DetailJSON": string(detailJSON),
		})
	} else if strings.Contains(selector, ShadowDOMSeparator) {
		parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
		js = scripts.MustExecute("shadow_dispatch_event.js.tmpl", map[string]any{
			"Host":       parts[0],
			"Shadow":     parts[1],
			"EventName":  eventName,
			"DetailJSON": string(detailJSON),
		})
	} else {
		js = scripts.MustExecute("dispatch_custom_event.js.tmpl", map[string]any{
			"Selector":   selector,
			"EventName":  eventName,
			"DetailJSON": string(detailJSON),
		})
	}

	err = chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("dispatching event: %w", err)
	}
	return nil
}

// TriggerPartialReload triggers a Piko partial reload in the browser.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes partialName (string) which identifies the partial to reload.
// Takes data (map[string]any) which contains data to pass to the partial.
// Takes refreshLevel (int) which controls the reload behaviour; zero uses
// default settings.
//
// Returns error when marshalling data fails or browser execution fails.
func TriggerPartialReload(ctx *ActionContext, partialName string, data map[string]any, refreshLevel int) error {
	dataJSON := "{}"
	if data != nil {
		bytes, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshalling data: %w", err)
		}
		dataJSON = string(bytes)
	}

	var jsCall string
	if refreshLevel != 0 {
		jsCall = scripts.MustExecute("partial_reload_with_options.js.tmpl", map[string]any{
			"PartialName": partialName,
			"DataJSON":    dataJSON,
			"Level":       refreshLevel,
		})
	} else {
		jsCall = scripts.MustExecute("partial_reload.js.tmpl", map[string]any{
			"PartialName": partialName,
			"DataJSON":    dataJSON,
		})
	}

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(jsCall, nil))
	if err != nil {
		return fmt.Errorf("triggering partial reload: %w", err)
	}
	return nil
}

// TriggerBusEvent triggers a Piko event bus event.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes eventName (string) which specifies the name of the event to trigger.
// Takes detail (map[string]any) which contains the event payload data.
//
// Returns error when the detail cannot be marshalled to JSON or when the
// browser fails to execute the event trigger.
func TriggerBusEvent(ctx *ActionContext, eventName string, detail map[string]any) error {
	detailJSON, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("marshalling event detail: %w", err)
	}

	jsCall := scripts.MustExecute("bus_emit.js.tmpl", map[string]any{
		"EventName":  eventName,
		"DetailJSON": string(detailJSON),
	})

	err = chromedp.Run(ctx.Ctx, chromedp.Evaluate(jsCall, nil))
	if err != nil {
		return fmt.Errorf("triggering bus event: %w", err)
	}
	return nil
}

// Eval evaluates JavaScript on an element or the page.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which identifies the target element, or "window",
// "document", or empty string for page-level evaluation.
// Takes script (string) which contains the JavaScript code to execute.
//
// Returns error when the script fails to execute.
func Eval(ctx *ActionContext, selector, script string) error {
	var js string
	if selector == "window" || selector == "document" || selector == "" {
		js = script
	} else if strings.Contains(selector, ShadowDOMSeparator) {
		parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
		js = scripts.MustExecute("shadow_eval_statement.js.tmpl", map[string]any{
			"Host":   parts[0],
			"Shadow": parts[1],
			"JS":     script,
		})
	} else {
		js = scripts.MustExecute("eval_statement.js.tmpl", map[string]any{
			"Selector": selector,
			"JS":       script,
		})
	}

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("evaluating script: %w", err)
	}
	return nil
}

// WaitForPartialReload waits for a Piko partial to finish loading.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes partialName (string) which identifies the partial to wait for.
// Takes timeout (time.Duration) which sets the maximum wait time.
//
// Returns error when the timeout is reached before the partial finishes
// loading.
func WaitForPartialReload(ctx *ActionContext, partialName string, timeout time.Duration) error {
	selector := fmt.Sprintf("[data-partial=%q]", partialName)

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	for {
		js := scripts.MustExecute("check_partial_not_loading.js.tmpl", map[string]any{
			"Selector": selector,
		})

		var notLoading bool
		err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &notLoading))
		if err == nil && notLoading {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for partial '%s' to finish loading", partialName)
		}

		<-ticker.C
	}
}

// WaitForVisible waits for an element to become visible.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes selector (string) which identifies the element to wait for.
// Takes timeout (time.Duration) which sets the maximum wait time.
//
// Returns error when the element does not become visible within the timeout.
func WaitForVisible(ctx *ActionContext, selector string, timeout time.Duration) error {
	return waitForBoolCondition(ctx, selector, timeout, IsElementVisible, "visible", "element exists but not visible")
}

// WaitForNotVisible waits for an element to become hidden or not exist.
//
// Takes ctx (*ActionContext) which provides the browser context for queries.
// Takes selector (string) which identifies the element to wait for.
// Takes timeout (time.Duration) which sets the maximum wait time.
//
// Returns error when the element remains visible after the timeout expires.
func WaitForNotVisible(ctx *ActionContext, selector string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	for {
		visible, err := IsElementVisible(ctx.Ctx, selector)
		if err != nil || !visible {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for '%s' to become hidden after %v", selector, timeout)
		}

		<-ticker.C
	}
}

// WaitForEnabled waits for an element to become enabled.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes selector (string) which identifies the target element.
// Takes timeout (time.Duration) which specifies how long to wait.
//
// Returns error when the element does not become enabled within the timeout.
func WaitForEnabled(ctx *ActionContext, selector string, timeout time.Duration) error {
	return waitForBoolCondition(ctx, selector, timeout, IsElementEnabled, "enabled", "element exists but disabled")
}

// WaitForDisabled waits for an element to become disabled.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes selector (string) which identifies the element to wait for.
// Takes timeout (time.Duration) which sets the maximum wait time.
//
// Returns error when the timeout is reached before the element becomes disabled.
func WaitForDisabled(ctx *ActionContext, selector string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		enabled, err := IsElementEnabled(ctx.Ctx, selector)
		if err == nil && !enabled {
			return nil
		}
		if err != nil {
			lastErr = err
		}

		if time.Now().After(deadline) {
			if lastErr != nil {
				return fmt.Errorf("timed out waiting for '%s' to be disabled after %v: %w", selector, timeout, lastErr)
			}
			return fmt.Errorf("timed out waiting for '%s' to be disabled after %v (element exists but enabled)", selector, timeout)
		}

		<-ticker.C
	}
}

// WaitForNotPresent waits for an element to be removed from the DOM.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the element to wait for removal.
// Takes timeout (time.Duration) which sets the maximum wait time.
//
// Returns error when the element remains present after the timeout expires.
func WaitForNotPresent(ctx *ActionContext, selector string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	for {
		exists := ElementExists(ctx.Ctx, selector)
		if !exists {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for '%s' to be removed from DOM after %v", selector, timeout)
		}

		<-ticker.C
	}
}

// submitInShadowDOM submits a form inside shadow DOM.
//
// Takes selector (string) which specifies the shadow DOM path in the format
// "hostSelector>>>shadowSelector".
//
// Returns error when the form cannot be submitted or no form is found.
func submitInShadowDOM(ctx context.Context, selector string) error {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_submit_form.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var submitted bool
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &submitted))
	if err != nil {
		return fmt.Errorf("submitting form in shadow DOM %s: %w", selector, err)
	}
	if !submitted {
		return fmt.Errorf("no form found in shadow DOM: %s", selector)
	}
	return nil
}

// getBodyPreview returns a truncated preview of the page body for diagnostics.
//
// Returns string which contains the body HTML, truncated to DebugHTMLPreviewLength
// characters, or an error message if the body cannot be retrieved.
func getBodyPreview(ctx context.Context) string {
	timedCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("browser getBodyPreview exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	var html string
	err := chromedp.Run(timedCtx,
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	)
	if err != nil {
		return "<no body: " + err.Error() + ">"
	}
	if len(html) > DebugHTMLPreviewLength {
		return html[:DebugHTMLPreviewLength] + "..."
	}
	return html
}

// waitForBoolCondition waits for a boolean condition to become true.
// It polls using the checker function until the condition is met or timeout
// is reached.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes selector (string) which identifies the element to check.
// Takes timeout (time.Duration) which sets the maximum wait time.
// Takes checker (BoolConditionChecker) which evaluates the condition.
// Takes conditionDesc (string) which describes the expected condition.
// Takes existsButFalseDesc (string) which describes the false state.
//
// Returns error when the timeout is reached before the condition becomes true.
func waitForBoolCondition(
	ctx *ActionContext,
	selector string,
	timeout time.Duration,
	checker BoolConditionChecker,
	conditionDesc string,
	existsButFalseDesc string,
) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		result, err := checker(ctx.Ctx, selector)
		if err == nil && result {
			return nil
		}
		if err != nil {
			lastErr = err
		}

		if time.Now().After(deadline) {
			if lastErr != nil {
				return fmt.Errorf("timed out waiting for '%s' to be %s after %v: %w", selector, conditionDesc, timeout, lastErr)
			}
			return fmt.Errorf("timed out waiting for '%s' to be %s after %v (%s)", selector, conditionDesc, timeout, existsButFalseDesc)
		}

		<-ticker.C
	}
}
