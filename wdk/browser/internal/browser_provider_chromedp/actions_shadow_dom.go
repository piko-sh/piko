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
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// shadowDOMSelectors holds the parts of a shadow DOM selector after parsing.
type shadowDOMSelectors struct {
	// Host is the CSS selector for the shadow DOM host element.
	Host string

	// Shadow is the CSS selector for the element within the shadow root.
	Shadow string
}

// scrollSettleDelay is the time to wait after scrolling for the page to settle.
const scrollSettleDelay = 50 * time.Millisecond

// shadowDOMActionFunc is a function that performs an action on a shadow DOM
// element. It receives the context, the full selector for error messages, and
// the parsed selectors.
type shadowDOMActionFunc func(ctx context.Context, selector string, selectors shadowDOMSelectors) error

// elementPosition holds the centre coordinates of an element.
type elementPosition struct {
	// X is the horizontal coordinate for mouse interactions.
	X float64

	// Y is the vertical coordinate of the element position.
	Y float64

	// Found indicates whether the element was located in the shadow DOM.
	Found bool
}

// parseShadowDOMSelector splits a shadow DOM selector into host and shadow
// parts.
//
// Takes selector (string) which contains the combined selector with a shadow
// DOM separator.
//
// Returns shadowDOMSelectors which holds the parsed host and shadow parts.
func parseShadowDOMSelector(selector string) shadowDOMSelectors {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	return shadowDOMSelectors{
		Host:   parts[0],
		Shadow: parts[1],
	}
}

// scrollShadowDOMElementIntoView scrolls a shadow DOM element into view and
// waits for it to settle.
//
// Takes selector (string) which identifies the element being scrolled.
// Takes selectors (shadowDOMSelectors) which provides the host
// and shadow selectors.
//
// Returns error when the element is not found or the scroll fails.
func scrollShadowDOMElementIntoView(ctx context.Context, selector string, selectors shadowDOMSelectors) error {
	scrollJS := fmt.Sprintf(jsShadowDOMScrollIntoView, selectors.Host, selectors.Shadow)

	var scrolled bool
	err := chromedp.Run(ctx, chromedp.Evaluate(scrollJS, &scrolled))
	if err != nil {
		return fmt.Errorf(ErrFmtScrollingShadowDOMElement, selector, err)
	}
	if !scrolled {
		return fmt.Errorf(ErrFmtElementNotFoundShadow, selector)
	}

	_ = chromedp.Run(ctx, chromedp.Sleep(scrollSettleDelay))

	return nil
}

// withShadowDOMScroll scrolls a shadow DOM element into view and then
// executes the given action. This pattern is common across many shadow DOM
// operations.
//
// Takes selector (string) which specifies the shadow DOM element path.
// Takes action (shadowDOMActionFunc) which performs the operation after
// scrolling.
//
// Returns error when the element cannot be scrolled into view or the action
// fails.
func withShadowDOMScroll(ctx context.Context, selector string, action shadowDOMActionFunc) error {
	selectors := parseShadowDOMSelector(selector)

	if err := scrollShadowDOMElementIntoView(ctx, selector, selectors); err != nil {
		return err
	}

	return action(ctx, selector, selectors)
}

// simpleShadowDOMAction executes a JS template against a shadow DOM element
// with scrolling, returning an error if the element is not found.
//
// Takes selector (string) which identifies the shadow DOM element.
// Takes templateName (string) which specifies the JS template to execute.
// Takes actionVerb (string) which describes the action for error messages.
// Takes notFoundFmt (string) which is the format string for the not-found
// error, receiving the selector as its sole argument.
//
// Returns error when the element cannot be found or the action fails.
func simpleShadowDOMAction(ctx context.Context, selector, templateName, actionVerb, notFoundFmt string) error {
	return withShadowDOMScroll(ctx, selector, func(ctx context.Context, selector string, selectors shadowDOMSelectors) error {
		js := scripts.MustExecute(templateName, map[string]any{
			"Host":   selectors.Host,
			"Shadow": selectors.Shadow,
		})

		var result bool
		err := chromedp.Run(ctx, chromedp.Evaluate(js, &result))
		if err != nil {
			return fmt.Errorf("%s shadow DOM element %s: %w", actionVerb, selector, err)
		}
		if !result {
			return fmt.Errorf(notFoundFmt, selector)
		}
		return nil
	})
}

// hoverInShadowDOM triggers hover on an element inside shadow DOM.
// Scrolls the element into view first to ensure accurate coordinates.
//
// Takes selector (string) which identifies the element within the shadow DOM.
//
// Returns error when the element cannot be found or the hover action fails.
func hoverInShadowDOM(ctx context.Context, selector string) error {
	return simpleShadowDOMAction(ctx, selector, "shadow_hover.js.tmpl", "hovering over", ErrFmtElementNotFoundShadow)
}

// rightClickInShadowDOM performs a right-click on an element inside shadow DOM.
// It scrolls the element into view first and includes proper coordinates in the
// event.
//
// Takes selector (string) which identifies the element within shadow DOM.
//
// Returns error when the element cannot be found or the click fails.
func rightClickInShadowDOM(ctx context.Context, selector string) error {
	return simpleShadowDOMAction(ctx, selector, "shadow_right_click.js.tmpl", "right-clicking", ErrFmtElementNotFoundShadow)
}

// clearInShadowDOM clears an input element inside shadow DOM.
// Scrolls the element into view first to ensure proper interaction.
//
// Takes selector (string) which identifies the shadow DOM element to clear.
//
// Returns error when the element cannot be found or cleared.
func clearInShadowDOM(ctx context.Context, selector string) error {
	return simpleShadowDOMAction(ctx, selector, "shadow_clear_input.js.tmpl", "clearing", ErrFmtElementNotFoundShadow)
}

// checkInShadowDOM checks a checkbox inside shadow DOM.
// Scrolls the element into view first to ensure proper interaction.
//
// Takes selector (string) which identifies the checkbox element.
//
// Returns error when the element cannot be checked or is not a valid checkbox.
func checkInShadowDOM(ctx context.Context, selector string) error {
	return simpleShadowDOMAction(ctx, selector, "shadow_check_checkbox.js.tmpl", "checking",
		"element is not a checkbox/radio or not found in shadow DOM: %s")
}

// uncheckInShadowDOM unchecks a checkbox inside shadow DOM.
// Scrolls the element into view first to ensure proper interaction.
//
// Takes selector (string) which identifies the checkbox element within the
// shadow DOM.
//
// Returns error when the element cannot be unchecked, is not a checkbox, or
// is not found in the shadow DOM.
func uncheckInShadowDOM(ctx context.Context, selector string) error {
	return simpleShadowDOMAction(ctx, selector, "shadow_uncheck_checkbox.js.tmpl", "unchecking",
		"element is not a checkbox or not found in shadow DOM: %s")
}

// fillInShadowDOM fills an input element inside shadow DOM.
// Scrolls the element into view first to ensure proper interaction.
//
// Takes selector (string) which specifies the shadow DOM element path.
// Takes value (string) which is the text to fill into the input element.
//
// Returns error when the element cannot be found or the fill operation fails.
func fillInShadowDOM(ctx context.Context, selector, value string) error {
	selectors := parseShadowDOMSelector(selector)

	if err := scrollShadowDOMElementIntoView(ctx, selector, selectors); err != nil {
		return err
	}

	js := scripts.MustExecute("shadow_fill_input.js.tmpl", map[string]any{
		"Host":   selectors.Host,
		"Shadow": selectors.Shadow,
		"Value":  value,
	})

	var filled bool
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &filled))
	if err != nil {
		return fmt.Errorf("filling shadow DOM element %s: %w", selector, err)
	}
	if !filled {
		return fmt.Errorf(ErrFmtElementNotFoundShadow, selector)
	}
	return nil
}

// focusInShadowDOM focuses an element inside shadow DOM.
// Scrolls the element into view first to ensure proper interaction.
//
// Takes selector (string) which specifies the element to focus.
//
// Returns error when the element cannot be found or focusing fails.
func focusInShadowDOM(ctx context.Context, selector string) error {
	return simpleShadowDOMAction(ctx, selector, "shadow_focus.js.tmpl", "focusing", ErrFmtElementNotFoundShadow)
}

// blurInShadowDOM blurs an element inside shadow DOM.
// Scrolls the element into view first to ensure proper interaction.
//
// Takes selector (string) which identifies the element to blur.
//
// Returns error when the element cannot be found or the blur fails.
func blurInShadowDOM(ctx context.Context, selector string) error {
	return simpleShadowDOMAction(ctx, selector, "shadow_blur.js.tmpl", "blurring", ErrFmtElementNotFoundShadow)
}

// getShadowDOMElementPosition returns the centre coordinates of a shadow DOM
// element.
//
// Takes selector (string) which identifies the element for error messages.
// Takes selectors (shadowDOMSelectors) which specifies the host
// and shadow selectors.
//
// Returns elementPosition which contains the centre coordinates of the element.
// Returns error when the element cannot be found or evaluation fails.
func getShadowDOMElementPosition(ctx context.Context, selector string, selectors shadowDOMSelectors) (elementPosition, error) {
	js := scripts.MustExecute("shadow_get_centre.js.tmpl", map[string]any{
		"Host":   selectors.Host,
		"Shadow": selectors.Shadow,
	})

	var result elementPosition
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &result)); err != nil {
		return result, fmt.Errorf("getting shadow DOM element position %s: %w", selector, err)
	}
	if !result.Found {
		return result, fmt.Errorf(ErrFmtElementNotFoundShadow, selector)
	}
	return result, nil
}

// dispatchMouseClick dispatches mouse down and up events at the specified
// coordinates.
//
// Takes x (float64) which specifies the horizontal position.
// Takes y (float64) which specifies the vertical position.
// Takes clickCount (int64) which specifies the number of clicks to simulate.
//
// Returns error when the mouse event cannot be dispatched.
func dispatchMouseClick(ctx context.Context, x, y float64, clickCount int64) error {
	return chromedp.Run(ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		if err := input.DispatchMouseEvent(input.MousePressed, x, y).
			WithButton(input.Left).
			WithClickCount(clickCount).
			Do(ctx2); err != nil {
			return err
		}
		return input.DispatchMouseEvent(input.MouseReleased, x, y).
			WithButton(input.Left).
			WithClickCount(clickCount).
			Do(ctx2)
	}))
}

// clickInShadowDOM clicks an element inside shadow DOM using proper mouse
// event simulation. This means clicking on contenteditable elements
// correctly positions the cursor.
//
// Takes selector (string) which specifies the shadow DOM element path.
//
// Returns error when the element cannot be scrolled into view, its position
// cannot be determined, or the click event fails to dispatch.
func clickInShadowDOM(ctx context.Context, selector string) error {
	selectors := parseShadowDOMSelector(selector)

	if err := scrollShadowDOMElementIntoView(ctx, selector, selectors); err != nil {
		return err
	}

	position, err := getShadowDOMElementPosition(ctx, selector, selectors)
	if err != nil {
		return err
	}

	if err := dispatchMouseClick(ctx, position.X, position.Y, 1); err != nil {
		return fmt.Errorf("clicking shadow DOM element %s: %w", selector, err)
	}
	return nil
}

// doubleClickInShadowDOM double-clicks an element inside shadow DOM using
// proper mouse event simulation. This means double-clicking on
// contenteditable elements correctly positions the cursor.
//
// Takes selector (string) which specifies the shadow DOM element path.
//
// Returns error when the element cannot be scrolled into view, its position
// cannot be determined, or the mouse click events fail to dispatch.
func doubleClickInShadowDOM(ctx context.Context, selector string) error {
	selectors := parseShadowDOMSelector(selector)

	if err := scrollShadowDOMElementIntoView(ctx, selector, selectors); err != nil {
		return err
	}

	position, err := getShadowDOMElementPosition(ctx, selector, selectors)
	if err != nil {
		return err
	}

	if err := dispatchMouseClick(ctx, position.X, position.Y, 1); err != nil {
		return fmt.Errorf("double-clicking shadow DOM element %s: %w", selector, err)
	}
	if err := dispatchMouseClick(ctx, position.X, position.Y, 2); err != nil {
		return fmt.Errorf("double-clicking shadow DOM element %s: %w", selector, err)
	}
	return nil
}

// setFilesInShadowDOM sets files on a file input inside shadow DOM.
// Uses CDP Runtime and DOM commands to properly set files on the element.
//
// Takes selector (string) which specifies the shadow DOM selector path.
// Takes absolutePaths ([]string) which contains the file paths to set.
//
// Returns error when scrolling into view fails or when setting files fails.
func setFilesInShadowDOM(ctx context.Context, selector string, absolutePaths []string) error {
	selectors := parseShadowDOMSelector(selector)

	if err := scrollShadowDOMElementIntoView(ctx, selector, selectors); err != nil {
		return err
	}

	return setFilesCDPRuntime(ctx, selector, selectors, absolutePaths)
}

// findShadowDOMFileInput locates a file input element within a shadow DOM.
//
// Takes selector (string) which identifies the file input element.
// Takes selectors (shadowDOMSelectors) which provides the host
// and shadow selectors.
//
// Returns cdp.BackendNodeID which is the backend node ID of the file input.
// Returns error when the document cannot be retrieved, the host element is not
// found, no shadow root exists, or the file input cannot be located.
func findShadowDOMFileInput(ctx context.Context, selector string, selectors shadowDOMSelectors) (cdp.BackendNodeID, error) {
	document, err := dom.GetDocument().WithDepth(-1).WithPierce(true).Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting document: %w", err)
	}

	hostNodeID, err := dom.QuerySelector(document.NodeID, selectors.Host).Do(ctx)
	if err != nil || hostNodeID == 0 {
		return 0, fmt.Errorf("host element not found: %s", selectors.Host)
	}

	hostNode, err := dom.DescribeNode().WithNodeID(hostNodeID).WithDepth(-1).WithPierce(true).Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("describing host node: %w", err)
	}

	if len(hostNode.ShadowRoots) == 0 {
		return 0, fmt.Errorf("no shadow root found on host: %s", selectors.Host)
	}

	shadowRootID := hostNode.ShadowRoots[0].NodeID
	fileInputNodeID, err := dom.QuerySelector(shadowRootID, selectors.Shadow).Do(ctx)
	if err != nil || fileInputNodeID == 0 {
		return 0, fmt.Errorf("file input not found in shadow DOM: %s", selector)
	}

	nodeInfo, err := dom.DescribeNode().WithNodeID(fileInputNodeID).Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("describing file input node: %w", err)
	}

	return nodeInfo.BackendNodeID, nil
}

// setFilesCDPRuntime uses CDP Runtime.evaluate to get a reference and set
// files.
//
// Takes selector (string) which identifies the target file input element.
// Takes selectors (shadowDOMSelectors) which specifies the shadow
// DOM host and shadow selectors.
// Takes absolutePaths ([]string) which contains the file paths to set on the
// input.
//
// Returns error when the shadow DOM file input cannot be found or when setting
// the files via CDP fails.
func setFilesCDPRuntime(ctx context.Context, selector string, selectors shadowDOMSelectors, absolutePaths []string) error {
	return chromedp.Run(ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		backendNodeID, err := findShadowDOMFileInput(ctx2, selector, selectors)
		if err != nil {
			return err
		}

		if err := dom.SetFileInputFiles(absolutePaths).WithBackendNodeID(backendNodeID).Do(ctx2); err != nil {
			return fmt.Errorf("setting files via CDP on %s: %w", selector, err)
		}

		changeJS := scripts.MustExecute("shadow_dispatch_change.js.tmpl", map[string]any{
			"Host":   selectors.Host,
			"Shadow": selectors.Shadow,
		})
		_ = chromedp.Evaluate(changeJS, nil).Do(ctx2)

		return nil
	}))
}
