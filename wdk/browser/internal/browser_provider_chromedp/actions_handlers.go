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
	"strconv"
	"time"
)

// stepHandler is a function that runs a browser step action.
type stepHandler func(*ActionContext, *BrowserStep) error

// stepHandlers maps action names to their handler functions.
var stepHandlers = map[string]stepHandler{
	"navigate":             handleNavigate,
	"goBack":               handleGoBack,
	"goForward":            handleGoForward,
	"stop":                 handleStop,
	"click":                handleClick,
	"doubleClick":          handleDoubleClick,
	"hover":                handleHover,
	"rightClick":           handleRightClick,
	"fill":                 handleFill,
	"setValue":             handleFill,
	"clear":                handleClear,
	"submit":               handleSubmit,
	"check":                handleCheck,
	"uncheck":              handleUncheck,
	"setFiles":             handleSetFiles,
	"focus":                handleFocus,
	"blur":                 handleBlur,
	"scroll":               handleScroll,
	"scrollIntoView":       handleScrollIntoView,
	"press":                handlePress,
	"type":                 handleType,
	"keyDown":              handleKeyDown,
	"keyUp":                handleKeyUp,
	"setCursor":            handleSetCursor,
	"setSelection":         handleSetSelection,
	"selectAll":            handleSelectAll,
	"collapseSelection":    handleCollapseSelection,
	"wait":                 handleWait,
	"waitForSelector":      handleWaitForSelector,
	"waitForText":          handleWaitForText,
	"waitForVisible":       handleWaitForVisible,
	"waitForNotVisible":    handleWaitForNotVisible,
	"waitForEnabled":       handleWaitForEnabled,
	"waitForDisabled":      handleWaitForDisabled,
	"waitForNotPresent":    handleWaitForNotPresent,
	"waitForPartialReload": handleWaitForPartialReload,
	"dispatchEvent":        handleDispatchEvent,
	"triggerPartialReload": handleTriggerPartialReload,
	"triggerBusEvent":      handleTriggerBusEvent,
	"pikoBusEmit":          handleTriggerBusEvent,
	"pikoPartialReload":    handleTriggerPartialReload,
	"eval":                 handleEval,
	"comment":              handleComment,
	"clearConsole":         handleClearConsole,
	"setAttribute":         handleSetAttribute,
	"removeAttribute":      handleRemoveAttribute,
	"setViewport":          handleSetViewport,
}

// ExecuteStep executes a single browser step.
//
// Takes ctx (*ActionContext) which provides the browser context for the step.
// Takes step (*BrowserStep) which defines the action to perform.
//
// Returns error when the action is unknown or the handler fails.
func ExecuteStep(ctx *ActionContext, step *BrowserStep) error {
	handler, ok := stepHandlers[step.Action]
	if !ok {
		return fmt.Errorf("unknown or unhandled action: %s (assertions should use ExecuteAssertion)", step.Action)
	}
	return handler(ctx, step)
}

// handleNavigate navigates the browser to the URL specified in the step.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the target URL in its Value field.
//
// Returns error when the navigation fails.
func handleNavigate(ctx *ActionContext, step *BrowserStep) error {
	return Navigate(ctx, step.Value)
}

// handleGoBack navigates the browser back one page in history.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns error when the navigation fails.
func handleGoBack(ctx *ActionContext, _ *BrowserStep) error {
	return GoBack(ctx)
}

// handleGoForward navigates the browser forward one page in history.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns error when the navigation fails.
func handleGoForward(ctx *ActionContext, _ *BrowserStep) error {
	return GoForward(ctx)
}

// handleStop stops the current browser page load.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns error when stopping the page load fails.
func handleStop(ctx *ActionContext, _ *BrowserStep) error {
	return Stop(ctx)
}

// handleClick performs a click action on the element matching the step selector.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes step (*BrowserStep) which contains the selector for the element to click.
//
// Returns error when the click action fails.
func handleClick(ctx *ActionContext, step *BrowserStep) error {
	return Click(ctx, step.Selector)
}

// handleDoubleClick performs a double-click on the element matching the
// selector in the browser step.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector to target.
//
// Returns error when the double-click action fails.
func handleDoubleClick(ctx *ActionContext, step *BrowserStep) error {
	return DoubleClick(ctx, step.Selector)
}

// handleHover hovers over the element matching the step selector.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector to hover over.
//
// Returns error when the hover action fails.
func handleHover(ctx *ActionContext, step *BrowserStep) error {
	return Hover(ctx, step.Selector)
}

// handleRightClick performs a right-click action on the specified element.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which contains the selector for the target element.
//
// Returns error when the right-click action fails.
func handleRightClick(ctx *ActionContext, step *BrowserStep) error {
	return RightClick(ctx, step.Selector)
}

// handleFill fills a form element with a value.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector and value to fill.
//
// Returns error when the fill operation fails.
func handleFill(ctx *ActionContext, step *BrowserStep) error {
	return Fill(ctx, step.Selector, step.Value)
}

// handleClear clears the content of an element matching the given selector.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector to clear.
//
// Returns error when the clear operation fails.
func handleClear(ctx *ActionContext, step *BrowserStep) error {
	return Clear(ctx, step.Selector)
}

// handleSubmit submits a form element matching the step selector.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector for the form element.
//
// Returns error when the form submission fails.
func handleSubmit(ctx *ActionContext, step *BrowserStep) error {
	return Submit(ctx, step.Selector)
}

// handleCheck verifies that an element matching the selector exists.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector to check.
//
// Returns error when the check fails or the element is not found.
func handleCheck(ctx *ActionContext, step *BrowserStep) error {
	return Check(ctx, step.Selector)
}

// handleUncheck unchecks a checkbox element matching the step selector.
//
// Takes ctx (*ActionContext) which provides the browser automation context.
// Takes step (*BrowserStep) which contains the selector for the checkbox.
//
// Returns error when the uncheck operation fails.
func handleUncheck(ctx *ActionContext, step *BrowserStep) error {
	return Uncheck(ctx, step.Selector)
}

// handleSetFiles sets files on a file input element.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector and file paths.
//
// Returns error when no files are specified or the operation fails.
func handleSetFiles(ctx *ActionContext, step *BrowserStep) error {
	filePaths := step.Files
	if len(filePaths) == 0 {
		if step.Value == "" {
			return errors.New("setFiles requires 'files' array or 'value' with file path")
		}
		filePaths = []string{step.Value}
	}
	return SetFiles(ctx, step.Selector, filePaths)
}

// handleFocus sets the browser focus to the element matching the step selector.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which contains the selector for the target element.
//
// Returns error when the element cannot be found or focused.
func handleFocus(ctx *ActionContext, step *BrowserStep) error {
	return Focus(ctx, step.Selector)
}

// handleBlur removes keyboard focus from the element matching the selector.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector for the element.
//
// Returns error when the blur operation fails.
func handleBlur(ctx *ActionContext, step *BrowserStep) error {
	return Blur(ctx, step.Selector)
}

// handleScroll scrolls the browser viewport to the specified selector.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which specifies the scroll target and value.
//
// Returns error when the scroll operation fails.
func handleScroll(ctx *ActionContext, step *BrowserStep) error {
	return Scroll(ctx, step.Selector, step.Value)
}

// handleWait pauses execution for the number of milliseconds specified
// in the step value.
//
// Takes step (*BrowserStep) which provides the wait duration as a
// string in its Value field.
//
// Returns error when the value is not a valid integer.
func handleWait(_ *ActionContext, step *BrowserStep) error {
	ms, err := strconv.Atoi(step.Value)
	if err != nil {
		return fmt.Errorf("invalid wait value %s: %w", step.Value, err)
	}
	Wait(ms)
	return nil
}

// handleWaitForSelector waits for a CSS selector to appear in the browser.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which contains the selector and timeout settings.
//
// Returns error when the selector does not appear within the timeout period.
func handleWaitForSelector(ctx *ActionContext, step *BrowserStep) error {
	timeout := getStepTimeout(step)
	return WaitForSelector(ctx, step.Selector, timeout)
}

// handleWaitForText waits for specific text to appear in a browser element.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which defines the selector and expected text.
//
// Returns error when the text does not appear within the timeout period.
func handleWaitForText(ctx *ActionContext, step *BrowserStep) error {
	timeout := getStepTimeout(step)
	return WaitForText(ctx, step.Selector, step.ExpectedString(), timeout)
}

// handleWaitForPartialReload waits for a partial page reload to complete.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which specifies the partial name and timeout.
//
// Returns error when the partial reload fails or times out.
func handleWaitForPartialReload(ctx *ActionContext, step *BrowserStep) error {
	timeout := getStepTimeout(step)
	return WaitForPartialReload(ctx, step.PartialName, timeout)
}

// handleWaitForVisible waits for the element matching the step selector to
// become visible.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which specifies the selector and timeout settings.
//
// Returns error when the element does not become visible within the timeout.
func handleWaitForVisible(ctx *ActionContext, step *BrowserStep) error {
	timeout := getStepTimeout(step)
	return WaitForVisible(ctx, step.Selector, timeout)
}

// handleWaitForNotVisible waits until the specified element is no longer visible.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector and timeout settings.
//
// Returns error when the element remains visible after the timeout expires.
func handleWaitForNotVisible(ctx *ActionContext, step *BrowserStep) error {
	timeout := getStepTimeout(step)
	return WaitForNotVisible(ctx, step.Selector, timeout)
}

// handleWaitForEnabled waits for the element matching the selector to become
// enabled.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes step (*BrowserStep) which specifies the selector and timeout settings.
//
// Returns error when the element does not become enabled within the timeout.
func handleWaitForEnabled(ctx *ActionContext, step *BrowserStep) error {
	timeout := getStepTimeout(step)
	return WaitForEnabled(ctx, step.Selector, timeout)
}

// handleWaitForDisabled waits until the specified element becomes disabled.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which specifies the selector and timeout settings.
//
// Returns error when the element does not become disabled within the timeout.
func handleWaitForDisabled(ctx *ActionContext, step *BrowserStep) error {
	timeout := getStepTimeout(step)
	return WaitForDisabled(ctx, step.Selector, timeout)
}

// handleWaitForNotPresent waits for an element to be removed from the page.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes step (*BrowserStep) which contains the selector and timeout settings.
//
// Returns error when the element remains present after the timeout.
func handleWaitForNotPresent(ctx *ActionContext, step *BrowserStep) error {
	timeout := getStepTimeout(step)
	return WaitForNotPresent(ctx, step.Selector, timeout)
}

// handleDispatchEvent dispatches a browser event to the specified element.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector and event details.
//
// Returns error when the event dispatch fails.
func handleDispatchEvent(ctx *ActionContext, step *BrowserStep) error {
	return DispatchEvent(ctx, step.Selector, step.EventName, step.EventDetail)
}

// handleTriggerPartialReload triggers a partial reload in the browser.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the partial name, data, and refresh
// level.
//
// Returns error when the partial reload fails.
func handleTriggerPartialReload(ctx *ActionContext, step *BrowserStep) error {
	return TriggerPartialReload(ctx, step.PartialName, step.Data, step.RefreshLevel)
}

// handleTriggerBusEvent triggers a bus event with the name and detail from the
// given step.
//
// Takes ctx (*ActionContext) which provides the action execution context.
// Takes step (*BrowserStep) which contains the event name and detail to trigger.
//
// Returns error when the bus event cannot be triggered.
func handleTriggerBusEvent(ctx *ActionContext, step *BrowserStep) error {
	return TriggerBusEvent(ctx, step.EventName, step.EventDetail)
}

// handleEval runs a JavaScript expression on a selected browser element.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which contains the selector and expression.
//
// Returns error when the evaluation fails.
func handleEval(ctx *ActionContext, step *BrowserStep) error {
	return Eval(ctx, step.Selector, step.Value)
}

// handleComment handles comment actions on the current page.
//
// Returns error when the comment action fails.
func handleComment(_ *ActionContext, _ *BrowserStep) error {
	return nil
}

// handleClearConsole clears the browser console logs via the page
// helper. No-ops if the page helper is nil.
//
// Takes ctx (*ActionContext) which provides the page helper.
//
// Returns error which is always nil.
func handleClearConsole(ctx *ActionContext, _ *BrowserStep) error {
	if ctx.PageHelper != nil {
		ctx.PageHelper.ClearConsoleLogs()
	}
	return nil
}

// handlePress sends a key press event using the specified key.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes step (*BrowserStep) which contains the key to press in Key or Value.
//
// Returns error when neither Key nor Value is set, or when the press fails.
func handlePress(ctx *ActionContext, step *BrowserStep) error {
	key := step.Key
	if key == "" {
		key = step.Value
	}
	if key == "" {
		return errors.New("press requires 'key' or 'value' field")
	}
	return Press(ctx, key)
}

// handleType types text into the currently focused element.
//
// Takes ctx (*ActionContext) which provides the browser automation context.
// Takes step (*BrowserStep) which contains the text to type in its Value field.
//
// Returns error when the Value field is empty or the type action fails.
func handleType(ctx *ActionContext, step *BrowserStep) error {
	if step.Value == "" {
		return errors.New("type requires 'value' field with text to type")
	}
	return Type(ctx, step.Value)
}

// handleKeyDown sends a key down event for the specified key.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the key in either Key or Value field.
//
// Returns error when neither key nor value field is set.
func handleKeyDown(ctx *ActionContext, step *BrowserStep) error {
	key := step.Key
	if key == "" {
		key = step.Value
	}
	if key == "" {
		return errors.New("keyDown requires 'key' or 'value' field")
	}
	return KeyDown(ctx, key)
}

// handleKeyUp releases a keyboard key that was previously pressed down.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which specifies the key to release via Key or Value.
//
// Returns error when neither key nor value field is provided.
func handleKeyUp(ctx *ActionContext, step *BrowserStep) error {
	key := step.Key
	if key == "" {
		key = step.Value
	}
	if key == "" {
		return errors.New("keyUp requires 'key' or 'value' field")
	}
	return KeyUp(ctx, key)
}

// handleSetCursor sets the cursor position within an element.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which specifies the selector and offset.
//
// Returns error when the selector field is empty or positioning fails.
func handleSetCursor(ctx *ActionContext, step *BrowserStep) error {
	if step.Selector == "" {
		return errors.New("setCursor requires 'selector' field")
	}
	return SetCursorPosition(ctx, step.Selector, step.Offset)
}

// handleSetSelection sets a text selection range on an element.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which contains the selector and selection range.
//
// Returns error when the selector field is empty or the selection fails.
func handleSetSelection(ctx *ActionContext, step *BrowserStep) error {
	if step.Selector == "" {
		return errors.New("setSelection requires 'selector' field")
	}
	return SetSelection(ctx, step.Selector, step.Start, step.End)
}

// handleSelectAll selects all content within the element matching the selector.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which contains the selector for the target element.
//
// Returns error when the selector field is empty or selection fails.
func handleSelectAll(ctx *ActionContext, step *BrowserStep) error {
	if step.Selector == "" {
		return errors.New("selectAll requires 'selector' field")
	}
	return SelectAll(ctx, step.Selector)
}

// handleCollapseSelection collapses the current selection in the browser.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which specifies whether to collapse to the end.
//
// Returns error when the selection cannot be collapsed.
func handleCollapseSelection(ctx *ActionContext, step *BrowserStep) error {
	return CollapseSelection(ctx, step.ToEnd)
}

// handleScrollIntoView scrolls the element matching the selector into view.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes step (*BrowserStep) which contains the selector to scroll to.
//
// Returns error when the selector field is empty or the scroll fails.
func handleScrollIntoView(ctx *ActionContext, step *BrowserStep) error {
	if step.Selector == "" {
		return errors.New("scrollIntoView requires 'selector' field")
	}
	return ScrollIntoView(ctx.Ctx, step.Selector)
}

// handleSetAttribute sets an HTML attribute on the element matching the
// selector.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes step (*BrowserStep) which contains the selector, attribute name, and
// value.
//
// Returns error when selector is empty, attribute name is missing, or the
// element cannot be found.
func handleSetAttribute(ctx *ActionContext, step *BrowserStep) error {
	if step.Selector == "" {
		return errors.New("setAttribute requires 'selector' field")
	}
	attributeName := step.AttributeName()
	if attributeName == "" {
		return errors.New("setAttribute requires 'name' or 'attribute' field")
	}
	return SetElementAttribute(ctx.Ctx, step.Selector, attributeName, step.Value)
}

// handleRemoveAttribute removes an attribute from an element matching the
// selector.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes step (*BrowserStep) which specifies the selector and attribute name.
//
// Returns error when the selector or attribute name is empty.
func handleRemoveAttribute(ctx *ActionContext, step *BrowserStep) error {
	if step.Selector == "" {
		return errors.New("removeAttribute requires 'selector' field")
	}
	attributeName := step.AttributeName()
	if attributeName == "" {
		return errors.New("removeAttribute requires 'name' or 'attribute' field")
	}
	return RemoveElementAttribute(ctx.Ctx, step.Selector, attributeName)
}

// handleSetViewport sets the browser viewport dimensions.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes step (*BrowserStep) which specifies the viewport width and height.
//
// Returns error when width or height is not positive.
func handleSetViewport(ctx *ActionContext, step *BrowserStep) error {
	if step.Width <= 0 {
		return errors.New("setViewport requires positive 'width' field")
	}
	if step.Height <= 0 {
		return errors.New("setViewport requires positive 'height' field")
	}
	return SetViewport(ctx, int64(step.Width), int64(step.Height))
}

// getStepTimeout returns the timeout for a step, using the default if not
// specified.
//
// Takes step (*BrowserStep) which contains the timeout configuration.
//
// Returns time.Duration which is the step timeout or DefaultAssertionTimeout.
func getStepTimeout(step *BrowserStep) time.Duration {
	if step.Timeout > 0 {
		return time.Duration(step.Timeout) * time.Millisecond
	}
	return DefaultAssertionTimeout
}
