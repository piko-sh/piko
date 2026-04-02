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
	"fmt"

	"github.com/chromedp/chromedp"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// SetCursorPosition sets the cursor to a specific offset within a
// contenteditable element. The offset is counted as characters from the start
// of the text content.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the contenteditable element.
// Takes offset (int) which specifies the character position for the cursor.
//
// Returns error when the element cannot be found, the cursor cannot be set,
// or the offset is out of range.
func SetCursorPosition(ctx *ActionContext, selector string, offset int) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	js := scripts.MustExecute("set_cursor_position.js.tmpl", map[string]any{"Offset": int64(offset)})
	result, err := EvalOnElement(ctx.Ctx, selector, js)
	if err != nil {
		return fmt.Errorf("setting cursor position: %w", err)
	}

	if b, ok := result.(bool); !ok || !b {
		return fmt.Errorf("failed to set cursor at offset %d (offset may be out of range)", offset)
	}

	return nil
}

// SetSelection selects text within an element by start and end offsets.
// Offsets are counted as characters from the start of the text content.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes selector (string) which identifies the target element.
// Takes start (int) which specifies the starting character offset.
// Takes end (int) which specifies the ending character offset.
//
// Returns error when the element cannot be found or the selection fails.
func SetSelection(ctx *ActionContext, selector string, start, end int) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	js := scripts.MustExecute("set_selection.js.tmpl", map[string]any{"Start": int64(start), "End": int64(end)})
	result, err := EvalOnElement(ctx.Ctx, selector, js)
	if err != nil {
		return fmt.Errorf("setting selection: %w", err)
	}

	if b, ok := result.(bool); !ok || !b {
		return fmt.Errorf("failed to set selection from %d to %d", start, end)
	}

	return nil
}

// GetCursorPosition returns the current cursor offset within an element.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes selector (string) which identifies the target element.
//
// Returns int which is the cursor position, or -1 if the cursor is not within
// the element.
// Returns error when the element cannot be found or the script fails.
func GetCursorPosition(ctx *ActionContext, selector string) (int, error) {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return -1, fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	result, err := EvalOnElement(ctx.Ctx, selector, scripts.MustGet("get_cursor_position.js"))
	if err != nil {
		return -1, fmt.Errorf("getting cursor position: %w", err)
	}

	switch v := result.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return -1, nil
	}
}

// GetSelection returns the current selection start and end offsets within an
// element.
// Returns (-1, -1) if there is no selection within the element.
//
// Takes ctx (*ActionContext) which provides the browser context for
// the action.
// Takes selector (string) which identifies the target element.
//
// Returns start (int) which is the starting character offset of the
// selection, or -1 if there is no selection.
// Returns end (int) which is the ending character offset of the
// selection, or -1 if there is no selection.
// Returns err (error) when the element cannot be found or the script
// fails.
func GetSelection(ctx *ActionContext, selector string) (start, end int, err error) {
	_, err = FindElement(ctx.Ctx, selector)
	if err != nil {
		return -1, -1, fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	result, err := EvalOnElement(ctx.Ctx, selector, scripts.MustGet("get_selection.js"))
	if err != nil {
		return -1, -1, fmt.Errorf("getting selection: %w", err)
	}

	selectionRange, ok := result.(map[string]any)
	if !ok {
		return -1, -1, nil
	}

	startVal, startOK := selectionRange["start"].(float64)
	endVal, endOK := selectionRange["end"].(float64)
	if !startOK || !endOK {
		return -1, -1, nil
	}
	return int(startVal), int(endVal), nil
}

// SelectAll selects all content within an element.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes selector (string) which identifies the target element.
//
// Returns error when the element cannot be found or selection fails.
func SelectAll(ctx *ActionContext, selector string) error {
	_, err := FindElement(ctx.Ctx, selector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, selector, err)
	}

	_, err = EvalOnElement(ctx.Ctx, selector, scripts.MustGet("select_all.js"))
	if err != nil {
		return fmt.Errorf("selecting all content: %w", err)
	}

	return nil
}

// CollapseSelection collapses the current selection to its start or end.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes toEnd (bool) which when true collapses to the end, otherwise to the
// start.
//
// Returns error when the selection cannot be collapsed.
func CollapseSelection(ctx *ActionContext, toEnd bool) error {
	js := scripts.MustGet("collapse_selection_to_start.js")
	if toEnd {
		js = scripts.MustGet("collapse_selection_to_end.js")
	}

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("collapsing selection: %w", err)
	}

	return nil
}

// PlaceCursorInElement places the cursor inside a child element of a parent
// element.
//
// Use it to programmatically position the cursor inside inline elements
// where Click() may not reliably position the cursor due to browser caret
// behaviour. The cursor is placed at offset 1 within the first text node
// of the child element, or at offset 0 if the text is empty. This handles
// Shadow DOM by using the shadow root's getSelection() method when available.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes parentSelector (string) which selects the parent container element
// (supports shadow DOM with >>>).
// Takes childSelector (string) which is a CSS selector for the child element
// within the parent.
//
// Returns error when the parent element cannot be found, the cursor placement
// script fails, or the child element does not exist.
func PlaceCursorInElement(ctx *ActionContext, parentSelector, childSelector string) error {
	_, err := FindElement(ctx.Ctx, parentSelector)
	if err != nil {
		return fmt.Errorf(ErrFmtFindingElement, parentSelector, err)
	}

	js := scripts.MustExecute("place_cursor_in_element.js.tmpl", map[string]any{"ChildSelector": childSelector})
	result, err := EvalOnElement(ctx.Ctx, parentSelector, js)
	if err != nil {
		return fmt.Errorf("placing cursor in element: %w", err)
	}

	if b, ok := result.(bool); !ok || !b {
		return fmt.Errorf("failed to place cursor in child element %q (element may not exist)", childSelector)
	}

	return nil
}
