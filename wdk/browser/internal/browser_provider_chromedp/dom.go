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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"

	"piko.sh/piko/internal/formatter/formatter_domain"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

var (
	// goldenFileFormatter is a pre-configured formatter for formatting captured
	// DOM snapshots in golden files. It uses RawHTMLMode to treat p-* attributes
	// as regular HTML (not directives), since the captured DOM contains runtime
	// values like p-key="r.0:0" that aren't valid expressions.
	goldenFileFormatter = formatter_domain.NewFormatterServiceWithOptions(&formatter_domain.FormatOptions{
		FileFormat:          formatter_domain.FormatHTML,
		IndentSize:          2,
		PreserveEmptyLines:  false,
		SortAttributes:      false,
		MaxLineLength:       120,
		AttributeWrapIndent: 1,
		RawHTMLMode:         true,
	})

	// uuidPattern matches UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx.
	uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

// NormaliseOptions controls how DOM normalisation is performed.
type NormaliseOptions struct {
	// ReplaceUUIDs enables replacing UUIDs with a placeholder for stable comparison.
	ReplaceUUIDs bool

	// FormatHTML enables HTML formatting for better readability and clearer diffs.
	FormatHTML bool
}

// Dimensions represents the position and size of an element.
type Dimensions struct {
	// X is the horizontal position in pixels.
	X float64 `json:"x"`

	// Y is the vertical position coordinate.
	Y float64 `json:"y"`

	// Width is the element width in pixels.
	Width float64 `json:"width"`

	// Height is the element height in pixels.
	Height float64 `json:"height"`
}

// DefaultNormaliseOptions returns the default normalisation options.
//
// Returns NormaliseOptions which contains sensible defaults for text
// normalisation with UUID replacement and HTML formatting enabled.
func DefaultNormaliseOptions() NormaliseOptions {
	return NormaliseOptions{
		ReplaceUUIDs: true,
		FormatHTML:   true,
	}
}

// NormaliseDOM cleans HTML for deterministic comparison.
//
// Takes html (string) which is the raw HTML content to normalise.
// Takes opts (NormaliseOptions) which controls normalisation behaviour.
//
// Returns string which is the cleaned and optionally formatted HTML.
func NormaliseDOM(html string, opts NormaliseOptions) string {
	normalised := html

	if opts.ReplaceUUIDs {
		normalised = uuidPattern.ReplaceAllString(normalised, "[UUID]")
	}

	if opts.FormatHTML {
		formatted, err := goldenFileFormatter.FormatWithOptions(
			context.Background(),
			[]byte(normalised),
			&formatter_domain.FormatOptions{
				FileFormat:          formatter_domain.FormatHTML,
				IndentSize:          2,
				PreserveEmptyLines:  false,
				SortAttributes:      false,
				MaxLineLength:       120,
				AttributeWrapIndent: 1,
				RawHTMLMode:         true,
			},
		)
		if err != nil {
			return strings.TrimSpace(normalised)
		}
		return strings.TrimSpace(string(formatted))
	}

	return strings.TrimSpace(normalised)
}

// FindElement finds DOM elements matching the given selector, with support for
// shadow DOM piercing using the " >>> " syntax.
//
// The " >>> " syntax allows piercing into shadow roots:
// "host-element >>> .shadow-content"
//
// Takes selector (string) which specifies the CSS selector to match
// elements.
//
// Returns []*cdp.Node which contains the matching DOM nodes.
// Returns error when no elements match or the selector is invalid.
func FindElement(ctx context.Context, selector string) ([]*cdp.Node, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM FindElement exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return findElementInShadowDOM(fastCtx, selector)
	}

	var nodes []*cdp.Node
	err := chromedp.Run(fastCtx,
		chromedp.Nodes(selector, &nodes, chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("finding element %s: %w", selector, err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf(ErrFmtElementNotFound, selector)
	}

	return nodes, nil
}

// FindElements finds multiple elements by CSS selector, with support for
// shadow DOM piercing. Use " >>> " to pierce into shadow roots, for example
// "host-element >>> .shadow-content".
//
// Takes selector (string) which specifies the CSS selector to match.
//
// Returns []*cdp.Node which contains the matching elements, or an empty slice
// if none are found.
// Returns error when the selector query fails.
func FindElements(ctx context.Context, selector string) ([]*cdp.Node, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM FindElements exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return findElementsInShadowDOM(fastCtx, selector)
	}

	var nodes []*cdp.Node
	err := chromedp.Run(fastCtx,
		chromedp.Nodes(selector, &nodes, chromedp.ByQueryAll),
	)
	if err != nil {
		return nil, fmt.Errorf("finding elements %s: %w", selector, err)
	}

	return nodes, nil
}

// ElementExists checks if an element exists without error.
//
// Takes selector (string) which specifies the CSS selector to find.
//
// Returns bool which is true if the element was found, false otherwise.
func ElementExists(ctx context.Context, selector string) bool {
	_, err := FindElement(ctx, selector)
	return err == nil
}

// GetElementText gets the text content of an element.
//
// Takes selector (string) which specifies the CSS selector for the element.
//
// Returns string which is the trimmed text content of the element.
// Returns error when the text cannot be retrieved from the element.
func GetElementText(ctx context.Context, selector string) (string, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM GetElementText exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return getTextFromShadowDOM(fastCtx, selector)
	}

	var text string
	err := chromedp.Run(fastCtx,
		chromedp.Text(selector, &text, chromedp.ByQuery),
	)
	if err != nil {
		return "", fmt.Errorf("getting text from %s: %w", selector, err)
	}
	return strings.TrimSpace(text), nil
}

// GetElementAttribute gets an attribute value from an element.
//
// Takes selector (string) which specifies the CSS selector for the element.
// Takes attributeName (string) which specifies the name of the attribute to get.
//
// Returns *string which is the attribute value, or nil if the attribute does
// not exist.
// Returns error when the attribute cannot be retrieved.
func GetElementAttribute(ctx context.Context, selector, attributeName string) (*string, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM GetElementAttribute exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return getAttributeFromShadowDOM(fastCtx, selector, attributeName)
	}

	var value string
	var ok bool
	err := chromedp.Run(fastCtx,
		chromedp.AttributeValue(selector, attributeName, &value, &ok, chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("getting attribute %s from %s: %w", attributeName, selector, err)
	}
	if !ok {
		return nil, nil
	}
	return &value, nil
}

// GetElementHTML gets the outer HTML of an element.
//
// When the selector contains a shadow DOM separator, the HTML is retrieved
// from the shadow DOM instead.
//
// Takes selector (string) which specifies the CSS selector for the element.
//
// Returns string which contains the outer HTML of the matched element.
// Returns error when the element cannot be found or the operation times out.
func GetElementHTML(ctx context.Context, selector string) (string, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM GetElementHTML exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return getHTMLFromShadowDOM(fastCtx, selector)
	}

	var html string
	err := chromedp.Run(fastCtx,
		chromedp.OuterHTML(selector, &html, chromedp.ByQuery),
	)
	if err != nil {
		return "", fmt.Errorf("getting HTML from %s: %w", selector, err)
	}
	return html, nil
}

// GetElementValue gets the value property of an input element.
//
// Takes selector (string) which identifies the target element, supporting
// shadow DOM paths using the shadow DOM separator.
//
// Returns string which is the current value of the input element.
// Returns error when the element cannot be found or the value cannot be read.
func GetElementValue(ctx context.Context, selector string) (string, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM GetElementValue exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return getValueFromShadowDOM(fastCtx, selector)
	}

	var value string
	err := chromedp.Run(fastCtx,
		chromedp.Value(selector, &value, chromedp.ByQuery),
	)
	if err != nil {
		return "", fmt.Errorf("getting value from %s: %w", selector, err)
	}
	return value, nil
}

// IsElementVisible checks if an element is visible in the browser.
//
// Takes selector (string) which specifies the CSS selector for the element.
//
// Returns bool which indicates whether the element is visible.
// Returns error when the visibility check fails.
func IsElementVisible(ctx context.Context, selector string) (bool, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM IsElementVisible exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return isVisibleInShadowDOM(fastCtx, selector)
	}

	js := scripts.MustExecute("is_element_visible.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var visible bool
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &visible))
	if err != nil {
		return false, fmt.Errorf("checking visibility of %s: %w", selector, err)
	}
	return visible, nil
}

// IsElementChecked checks if a checkbox or radio element is checked.
//
// Takes selector (string) which identifies the element to check.
//
// Returns bool which is true if the element is checked, false otherwise.
// Returns error when the element is not found or is not a checkbox or radio.
func IsElementChecked(ctx context.Context, selector string) (bool, error) {
	return queryDOMBoolProperty(
		ctx, selector,
		"check_element_checked.js.tmpl",
		"shadow_check_element_checked.js.tmpl",
		"checking checked state",
		"element not found or not a checkbox/radio: %s",
		"element not found or not a checkbox/radio in shadow DOM: %s",
	)
}

// IsElementEnabled checks if an element is enabled (not disabled).
//
// Takes selector (string) which specifies the CSS selector for the element.
//
// Returns bool which is true if the element is enabled, false otherwise.
// Returns error when the element is not found or the query fails.
func IsElementEnabled(ctx context.Context, selector string) (bool, error) {
	return queryDOMBoolProperty(
		ctx, selector,
		"check_element_enabled.js.tmpl",
		"shadow_check_element_enabled.js.tmpl",
		"checking enabled state",
		"element not found: %s",
		ErrFmtElementNotFoundShadow,
	)
}

// EvalOnElement evaluates JavaScript on an element matching the selector.
//
// Takes selector (string) which identifies the target element, supporting
// shadow DOM selectors with a separator.
// Takes js (string) which contains the JavaScript code to execute.
// Takes arguments (...any) which provides arguments to pass to the JavaScript.
//
// Returns any which is the result of the JavaScript evaluation.
// Returns error when the element cannot be found or the script fails.
func EvalOnElement(ctx context.Context, selector, js string, arguments ...any) (any, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 2*time.Second, fmt.Errorf("DOM EvalOnElement exceeded %s timeout", 2*time.Second))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return evalOnShadowDOMElement(fastCtx, selector, js, arguments...)
	}

	fullJS := scripts.MustExecute("eval_on_element.js.tmpl", map[string]any{
		"Selector": selector,
		"JS":       js,
	})

	var result any
	err := chromedp.Run(fastCtx, chromedp.Evaluate(fullJS, &result))
	if err != nil {
		return nil, fmt.Errorf("evaluating JS on %s: %w", selector, err)
	}
	return result, nil
}

// ScrollIntoView scrolls an element into the viewport.
//
// Takes selector (string) which identifies the element to scroll into view.
//
// Returns error when the element is not found or scrolling fails.
func ScrollIntoView(ctx context.Context, selector string) error {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 2*time.Second, fmt.Errorf("DOM ScrollIntoView exceeded %s timeout", 2*time.Second))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return scrollIntoViewInShadowDOM(fastCtx, selector)
	}

	js := scripts.MustExecute("scroll_into_view.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var found bool
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &found))
	if err != nil {
		return fmt.Errorf("scrolling %s into view: %w", selector, err)
	}
	if !found {
		return fmt.Errorf(ErrFmtElementNotFound, selector)
	}
	return nil
}

// GetAllAttributes gets all attributes of an element as a map.
//
// Takes selector (string) which identifies the element to query, supporting
// shadow DOM selectors.
//
// Returns map[string]string which contains all attribute name-value pairs.
// Returns error when the element is not found or the query fails.
func GetAllAttributes(ctx context.Context, selector string) (map[string]string, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM GetAllAttributes exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return getAllAttributesInShadowDOM(fastCtx, selector)
	}

	js := scripts.MustExecute("get_all_attributes.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var result map[string]any
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting attributes from %s: %w", selector, err)
	}
	if result == nil {
		return nil, fmt.Errorf(ErrFmtElementNotFound, selector)
	}

	attrs := make(map[string]string)
	for k, v := range result {
		attrs[k] = fmt.Sprintf("%v", v)
	}
	return attrs, nil
}

// SetElementAttribute sets an attribute on an element.
//
// Takes selector (string) which identifies the target element using a CSS
// selector, with support for shadow DOM paths separated by ShadowDOMSeparator.
// Takes attributeName (string) which specifies the name of the attribute to set.
// Takes attributeValue (string) which provides the value to assign
// to the attribute.
//
// Returns error when the element cannot be found or the attribute cannot be
// set.
func SetElementAttribute(ctx context.Context, selector, attributeName, attributeValue string) error {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM SetElementAttribute exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return setAttributeInShadowDOM(fastCtx, selector, attributeName, attributeValue)
	}

	js := scripts.MustExecute("set_attribute.js.tmpl", map[string]any{
		"Selector":  selector,
		"AttrName":  attributeName,
		"AttrValue": attributeValue,
	})

	var found bool
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &found))
	if err != nil {
		return fmt.Errorf("setting attribute %s on %s: %w", attributeName, selector, err)
	}
	if !found {
		return fmt.Errorf(ErrFmtElementNotFound, selector)
	}
	return nil
}

// RemoveElementAttribute removes an attribute from an element.
//
// Takes selector (string) which identifies the target element, supporting
// shadow DOM selectors with the ShadowDOMSeparator.
// Takes attributeName (string) which specifies the attribute to remove.
//
// Returns error when the element is not found or the attribute cannot be
// removed.
func RemoveElementAttribute(ctx context.Context, selector, attributeName string) error {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM RemoveElementAttribute exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return removeAttributeInShadowDOM(fastCtx, selector, attributeName)
	}

	js := scripts.MustExecute("remove_attribute.js.tmpl", map[string]any{
		"Selector": selector,
		"AttrName": attributeName,
	})

	var found bool
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &found))
	if err != nil {
		return fmt.Errorf("removing attribute %s from %s: %w", attributeName, selector, err)
	}
	if !found {
		return fmt.Errorf(ErrFmtElementNotFound, selector)
	}
	return nil
}

// GetElementDimensions gets the bounding box of an element.
//
// Takes selector (string) which specifies the CSS selector for the element.
//
// Returns *Dimensions which contains the position and size of the element.
// Returns error when the element cannot be found or the query fails.
func GetElementDimensions(ctx context.Context, selector string) (*Dimensions, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM GetElementDimensions exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return getDimensionsInShadowDOM(fastCtx, selector)
	}

	js := scripts.MustExecute("get_element_rect.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var result map[string]float64
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting dimensions of %s: %w", selector, err)
	}
	if result == nil {
		return nil, fmt.Errorf(ErrFmtElementNotFound, selector)
	}

	return &Dimensions{
		X:      result["x"],
		Y:      result["y"],
		Width:  result["width"],
		Height: result["height"],
	}, nil
}

// HasShadowRoot checks if an element has a shadow root.
//
// Takes hostSelector (string) which specifies the CSS selector for the host
// element to check.
//
// Returns bool which is true if the element has a shadow root attached.
// Returns error when the shadow root check fails or times out.
func HasShadowRoot(ctx context.Context, hostSelector string) (bool, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM HasShadowRoot exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	js := scripts.MustExecute("shadow_has_root.js.tmpl", map[string]any{
		"Host": hostSelector,
	})

	var hasShadow bool
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &hasShadow))
	if err != nil {
		return false, fmt.Errorf("checking shadow root for %s: %w", hostSelector, err)
	}
	return hasShadow, nil
}

// GetShadowRootHTML gets the innerHTML of an element's shadow root.
//
// Takes hostSelector (string) which specifies the CSS selector for the host
// element containing the shadow root.
//
// Returns string which is the innerHTML of the shadow root, or empty string
// if the element has no shadow root.
// Returns error when the element cannot be found or the shadow root cannot
// be accessed.
func GetShadowRootHTML(ctx context.Context, hostSelector string) (string, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM GetShadowRootHTML exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	js := scripts.MustExecute("shadow_get_root_html.js.tmpl", map[string]any{
		"Host": hostSelector,
	})

	var html *string
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &html))
	if err != nil {
		return "", fmt.Errorf("getting shadow root HTML for %s: %w", hostSelector, err)
	}
	if html == nil {
		return "", fmt.Errorf("element %s has no shadow root", hostSelector)
	}
	return *html, nil
}

// GetFormData returns the form data from a form element (or its closest
// ancestor form) as a map, returning only the last value for multi-value
// fields (use EvalOnElement for full control).
//
// Takes ctx (context.Context) which controls the evaluation timeout.
// Takes selector (string) which identifies the form element.
//
// Returns map[string]any which maps field names to their values.
// Returns error when the element cannot be found or the script fails.
func GetFormData(ctx context.Context, selector string) (map[string]any, error) {
	js := scripts.MustGet("get_form_data.js")
	result, err := EvalOnElement(ctx, selector, js)
	if err != nil {
		return nil, fmt.Errorf("getting form data from %s: %w", selector, err)
	}
	if result == nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}
	data, ok := result.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return data, nil
}

// ListenForEvent attaches an event listener to the element matching selector,
// capturing e.detail into a window-scoped variable. The captured detail can
// later be retrieved with GetEventDetail using the same eventName.
//
// Takes ctx (context.Context) which controls the evaluation timeout.
// Takes selector (string) which identifies the target element.
// Takes eventName (string) which specifies the DOM event to listen for.
//
// Returns error when the element cannot be found or the script fails.
func ListenForEvent(ctx context.Context, selector, eventName string) error {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 2*time.Second, fmt.Errorf("DOM ListenForEvent exceeded %s timeout", 2*time.Second))
	defer cancel()

	callbackName := fmt.Sprintf("__eventCallback_%s", eventName)

	var findElementJS string
	if strings.Contains(selector, ShadowDOMSeparator) {
		parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
		findElementJS = fmt.Sprintf(`document.querySelector(%s).shadowRoot.querySelector(%s)`,
			strconv.Quote(parts[0]), strconv.Quote(parts[1]))
	} else {
		findElementJS = fmt.Sprintf(`document.querySelector(%s)`, strconv.Quote(selector))
	}

	js := scripts.MustExecute("listen_for_event.js.tmpl", map[string]any{
		"CallbackName":  callbackName,
		"FindElementJS": findElementJS,
		"EventName":     eventName,
	})

	var ok bool
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &ok))
	if err != nil {
		return fmt.Errorf("listening for event %q on %s: %w", eventName, selector, err)
	}
	if !ok {
		return fmt.Errorf("element not found: %s", selector)
	}
	return nil
}

// GetEventDetail returns the event detail captured by a prior ListenForEvent
// call. Returns nil if the event has not been received yet.
//
// Takes ctx (context.Context) which controls the evaluation timeout.
// Takes eventName (string) which specifies the event name to check.
//
// Returns any which is the captured e.detail, or nil if not yet received.
// Returns error when the script fails.
func GetEventDetail(ctx context.Context, eventName string) (any, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 2*time.Second, fmt.Errorf("DOM GetEventDetail exceeded %s timeout", 2*time.Second))
	defer cancel()

	callbackName := fmt.Sprintf("__eventCallback_%s", eventName)

	js := scripts.MustExecute("check_event_received.js.tmpl", map[string]any{
		"CallbackName": callbackName,
	})

	var result any
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("checking event %q: %w", eventName, err)
	}
	return result, nil
}

// findElementInShadowDOM handles the >>> shadow DOM piercing syntax.
//
// Takes selector (string) which contains the host and shadow selectors
// separated by the shadow DOM separator.
//
// Returns []*cdp.Node which is a placeholder slice; actual operations use
// JavaScript directly.
// Returns error when the shadow host cannot be found or the element does not
// exist within the shadow DOM.
func findElementInShadowDOM(ctx context.Context, selector string) ([]*cdp.Node, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_find_element.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var result any
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("finding shadow host %s: %w", hostSelector, err)
	}
	if result == nil {
		return nil, fmt.Errorf(ErrFmtElementNotFoundShadow, selector)
	}

	return []*cdp.Node{{}}, nil
}

// findElementsInShadowDOM handles the >>> shadow DOM piercing syntax for
// multiple elements.
//
// Takes selector (string) which contains the host and shadow selectors
// separated by the shadow DOM separator.
//
// Returns []*cdp.Node which contains placeholder nodes matching the count of
// found elements.
// Returns error when the shadow host cannot be found or evaluated.
func findElementsInShadowDOM(ctx context.Context, selector string) ([]*cdp.Node, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_find_elements_count.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var count int
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &count))
	if err != nil {
		return nil, fmt.Errorf("finding shadow host %s: %w", hostSelector, err)
	}

	nodes := make([]*cdp.Node, count)
	for i := range nodes {
		nodes[i] = &cdp.Node{}
	}
	return nodes, nil
}

// getTextFromShadowDOM gets text from an element inside shadow DOM.
//
// Takes selector (string) which specifies the combined host and shadow element
// selector separated by ShadowDOMSeparator.
//
// Returns string which is the trimmed text content of the shadow DOM element.
// Returns error when the shadow DOM element cannot be found or evaluated.
func getTextFromShadowDOM(ctx context.Context, selector string) (string, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_get_text.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var text string
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &text))
	if err != nil {
		return "", fmt.Errorf("getting text from shadow DOM %s: %w", selector, err)
	}
	return strings.TrimSpace(text), nil
}

// getAttributeFromShadowDOM gets an attribute from an element inside shadow
// DOM.
//
// Takes selector (string) which specifies the element path using a shadow DOM
// separator to split host and shadow selectors.
// Takes attributeName (string) which specifies the attribute name to retrieve.
//
// Returns *string which contains the attribute value, or nil if the element
// or attribute does not exist.
// Returns error when the JavaScript execution fails.
func getAttributeFromShadowDOM(ctx context.Context, selector, attributeName string) (*string, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_get_attribute.js.tmpl", map[string]any{
		"Host":     hostSelector,
		"Shadow":   shadowSelector,
		"AttrName": attributeName,
	})

	var result any
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting attribute from shadow DOM: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	return new(fmt.Sprintf("%v", result)), nil
}

// getHTMLFromShadowDOM gets HTML from an element inside shadow DOM.
//
// Takes selector (string) which specifies the element path using shadow DOM
// separator format (host selector + separator + shadow selector).
//
// Returns string which contains the HTML content of the matched element.
// Returns error when the shadow DOM evaluation fails.
func getHTMLFromShadowDOM(ctx context.Context, selector string) (string, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_get_html.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var html string
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &html))
	if err != nil {
		return "", fmt.Errorf("getting HTML from shadow DOM: %w", err)
	}
	return html, nil
}

// getValueFromShadowDOM gets the value from an input inside shadow DOM.
//
// Takes selector (string) which specifies the combined host and shadow
// selector separated by the shadow DOM separator.
//
// Returns string which is the value of the matched input element.
// Returns error when the JavaScript execution fails.
func getValueFromShadowDOM(ctx context.Context, selector string) (string, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_get_value.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var value string
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &value))
	if err != nil {
		return "", fmt.Errorf("getting value from shadow DOM: %w", err)
	}
	return value, nil
}

// isVisibleInShadowDOM checks visibility of an element inside shadow DOM.
//
// Takes selector (string) which specifies the element using a shadow DOM
// separator to split the host and shadow selectors.
//
// Returns bool which indicates whether the element is visible.
// Returns error when the visibility check fails.
func isVisibleInShadowDOM(ctx context.Context, selector string) (bool, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_is_visible.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var visible bool
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &visible))
	if err != nil {
		return false, fmt.Errorf("checking visibility in shadow DOM: %w", err)
	}
	return visible, nil
}

// queryDOMBoolProperty checks a boolean DOM property by running a JS template
// and its shadow DOM equivalent.
//
// Takes selector (string) which identifies the element to query.
// Takes jsTemplate (string) which specifies the JS template for normal DOM.
// Takes shadowJSTemplate (string) which specifies the JS template for shadow
// DOM.
// Takes errorContext (string) which describes the property for error messages.
// Takes nilErrorFmt (string) which is the format string for nil result errors.
// Takes shadowNilErrorFmt (string) which is the format string for shadow DOM
// nil result errors.
//
// Returns bool which is the property value.
// Returns error when the element is not found or the query fails.
func queryDOMBoolProperty(
	ctx context.Context,
	selector, jsTemplate, shadowJSTemplate, errorContext, nilErrorFmt string,
	shadowNilErrorFmt string,
) (bool, error) {
	fastCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("DOM queryDOMBoolProperty exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	if strings.Contains(selector, ShadowDOMSeparator) {
		return queryBoolPropertyInShadowDOM(fastCtx, selector, shadowJSTemplate, errorContext, shadowNilErrorFmt)
	}

	js := scripts.MustExecute(jsTemplate, map[string]any{
		"Selector": selector,
	})

	var result any
	err := chromedp.Run(fastCtx, chromedp.Evaluate(js, &result))
	if err != nil {
		return false, fmt.Errorf("%s of %s: %w", errorContext, selector, err)
	}
	if result == nil {
		return false, fmt.Errorf(nilErrorFmt, selector)
	}

	boolResult, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("unexpected result type for %s: %T", selector, result)
	}
	return boolResult, nil
}

// queryBoolPropertyInShadowDOM checks a boolean property on an element inside
// shadow DOM.
//
// Takes selector (string) which specifies the combined host and shadow
// selector separated by the shadow DOM separator.
// Takes shadowJSTemplate (string) which specifies the JS template for shadow
// DOM queries.
// Takes errorContext (string) which describes the property for error messages.
// Takes nilErrorFmt (string) which is the format string for nil result errors.
//
// Returns bool which is the property value.
// Returns error when the element cannot be found or the result type is
// unexpected.
func queryBoolPropertyInShadowDOM(ctx context.Context, selector, shadowJSTemplate, errorContext, nilErrorFmt string) (bool, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute(shadowJSTemplate, map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var result any
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return false, fmt.Errorf("%s in shadow DOM: %w", errorContext, err)
	}
	if result == nil {
		return false, fmt.Errorf(nilErrorFmt, selector)
	}

	boolResult, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("unexpected result type for shadow DOM element: %T", result)
	}
	return boolResult, nil
}

// evalOnShadowDOMElement evaluates JS on an element inside shadow DOM.
//
// Takes selector (string) which specifies the combined host and shadow
// selector separated by the shadow DOM separator.
// Takes js (string) which contains the JavaScript code to execute on
// the matched element.
//
// Returns any which is the result of the JavaScript evaluation.
// Returns error when the shadow DOM element cannot be found or the
// script fails.
func evalOnShadowDOMElement(ctx context.Context, selector, js string, _ ...any) (any, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	fullJS := scripts.MustExecute("shadow_eval_on_element.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
		"JS":     js,
	})

	var result any
	err := chromedp.Run(ctx, chromedp.Evaluate(fullJS, &result))
	if err != nil {
		return nil, fmt.Errorf("evaluating JS on shadow DOM element: %w", err)
	}
	return result, nil
}

// scrollIntoViewInShadowDOM scrolls a shadow DOM element into view.
//
// Takes selector (string) which specifies the element path in the format
// "hostSelector>>>shadowSelector".
//
// Returns error when the element cannot be found or the scroll fails.
func scrollIntoViewInShadowDOM(ctx context.Context, selector string) error {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_scroll_into_view.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var found bool
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &found))
	if err != nil {
		return fmt.Errorf("scrolling shadow DOM element into view: %w", err)
	}
	if !found {
		return fmt.Errorf(ErrFmtElementNotFoundShadow, selector)
	}
	return nil
}

// getAllAttributesInShadowDOM gets all attributes from a shadow DOM element.
//
// Takes selector (string) which specifies the element path in the format
// "hostSelector>>>shadowSelector".
//
// Returns map[string]string which contains all attribute name-value pairs.
// Returns error when the element is not found or script execution fails.
func getAllAttributesInShadowDOM(ctx context.Context, selector string) (map[string]string, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_get_all_attributes.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var result map[string]any
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting attributes from shadow DOM: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf(ErrFmtElementNotFoundShadow, selector)
	}

	attrs := make(map[string]string)
	for k, v := range result {
		attrs[k] = fmt.Sprintf("%v", v)
	}
	return attrs, nil
}

// setAttributeInShadowDOM sets an attribute on a shadow DOM element.
//
// Takes selector (string) which specifies the element path using
// ShadowDOMSeparator to separate host and shadow selectors.
// Takes attributeName (string) which is the name of the attribute to set.
// Takes attributeValue (string) which is the value to assign to the attribute.
//
// Returns error when the JavaScript execution fails or the element is not
// found in the shadow DOM.
func setAttributeInShadowDOM(ctx context.Context, selector, attributeName, attributeValue string) error {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_set_attribute.js.tmpl", map[string]any{
		"Host":      hostSelector,
		"Shadow":    shadowSelector,
		"AttrName":  attributeName,
		"AttrValue": attributeValue,
	})

	var found bool
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &found))
	if err != nil {
		return fmt.Errorf("setting attribute in shadow DOM: %w", err)
	}
	if !found {
		return fmt.Errorf(ErrFmtElementNotFoundShadow, selector)
	}
	return nil
}

// removeAttributeInShadowDOM removes an attribute from a shadow DOM element.
//
// Takes selector (string) which specifies the element path using the shadow
// DOM separator to identify the host and shadow elements.
// Takes attributeName (string) which is the name of the attribute to remove.
//
// Returns error when the attribute cannot be removed or the element is not
// found.
func removeAttributeInShadowDOM(ctx context.Context, selector, attributeName string) error {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_remove_attribute.js.tmpl", map[string]any{
		"Host":     hostSelector,
		"Shadow":   shadowSelector,
		"AttrName": attributeName,
	})

	var found bool
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &found))
	if err != nil {
		return fmt.Errorf("removing attribute from shadow DOM: %w", err)
	}
	if !found {
		return fmt.Errorf(ErrFmtElementNotFoundShadow, selector)
	}
	return nil
}

// getDimensionsInShadowDOM gets dimensions of a shadow DOM element.
//
// Takes selector (string) which specifies the element path using shadow DOM
// separator notation (host selector + separator + shadow selector).
//
// Returns *Dimensions which contains the position and size of the element.
// Returns error when the JavaScript execution fails or the element is not
// found.
func getDimensionsInShadowDOM(ctx context.Context, selector string) (*Dimensions, error) {
	parts := strings.SplitN(selector, ShadowDOMSeparator, 2)
	hostSelector := parts[0]
	shadowSelector := parts[1]

	js := scripts.MustExecute("shadow_get_rect.js.tmpl", map[string]any{
		"Host":   hostSelector,
		"Shadow": shadowSelector,
	})

	var result map[string]float64
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting dimensions from shadow DOM: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf(ErrFmtElementNotFoundShadow, selector)
	}

	return &Dimensions{
		X:      result["x"],
		Y:      result["y"],
		Width:  result["width"],
		Height: result["height"],
	}, nil
}
