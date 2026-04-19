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
	"strings"
	"time"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

// Click clicks an element by CSS selector.
//
// Takes selector (string) which specifies the CSS selector of the element.
//
// Returns *Page which allows method chaining for further actions.
func (p *Page) Click(selector string) *Page {
	p.beforeAction("Click", selector)
	start := time.Now()
	err := browser_provider_chromedp.Click(p.actionCtx(), selector)
	p.afterAction("Click", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Click(%q) failed: %v", selector, err)
	}
	return p
}

// DoubleClick double-clicks an element by CSS selector.
//
// Takes selector (string) which specifies the CSS selector of the element to
// double-click.
//
// Returns *Page which allows method chaining for further actions.
func (p *Page) DoubleClick(selector string) *Page {
	p.beforeAction("DoubleClick", selector)
	start := time.Now()
	err := browser_provider_chromedp.DoubleClick(p.actionCtx(), selector)
	p.afterAction("DoubleClick", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("DoubleClick(%q) failed: %v", selector, err)
	}
	return p
}

// Hover moves the mouse over an element.
//
// Takes selector (string) which identifies the element to hover over.
//
// Returns *Page which allows method chaining.
func (p *Page) Hover(selector string) *Page {
	p.beforeAction("Hover", selector)
	start := time.Now()
	err := browser_provider_chromedp.Hover(p.actionCtx(), selector)
	p.afterAction("Hover", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Hover(%q) failed: %v", selector, err)
	}
	return p
}

// RightClick performs a right-click (context menu) on an element.
//
// Takes selector (string) which identifies the element to right-click.
//
// Returns *Page which allows method chaining.
func (p *Page) RightClick(selector string) *Page {
	p.beforeAction("RightClick", selector)
	start := time.Now()
	err := browser_provider_chromedp.RightClick(p.actionCtx(), selector)
	p.afterAction("RightClick", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("RightClick(%q) failed: %v", selector, err)
	}
	return p
}

// Fill sets the value of an input element, simulating typing.
//
// Takes selector (string) which identifies the input element to fill.
// Takes value (string) which specifies the text to enter.
//
// Returns *Page which allows method chaining.
func (p *Page) Fill(selector, value string) *Page {
	detail := fmt.Sprintf(fmtKeyValue, selector, value)
	p.beforeAction("Fill", detail)
	start := time.Now()
	err := browser_provider_chromedp.Fill(p.actionCtx(), selector, value)
	p.afterAction("Fill", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Fill(%q, %q) failed: %v", selector, value, err)
	}
	return p
}

// Clear clears the value of an input or textarea element.
//
// Takes selector (string) which identifies the element to clear.
//
// Returns *Page which allows method chaining.
func (p *Page) Clear(selector string) *Page {
	p.beforeAction("Clear", selector)
	start := time.Now()
	err := browser_provider_chromedp.Clear(p.actionCtx(), selector)
	p.afterAction("Clear", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Clear(%q) failed: %v", selector, err)
	}
	return p
}

// Submit submits a form by clicking a submit button or triggering form
// submission.
//
// Takes selector (string) which identifies the submit button or form element.
//
// Returns *Page which enables method chaining.
func (p *Page) Submit(selector string) *Page {
	p.beforeAction("Submit", selector)
	start := time.Now()
	err := browser_provider_chromedp.Submit(p.actionCtx(), selector)
	p.afterAction("Submit", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Submit(%q) failed: %v", selector, err)
	}
	return p
}

// Check checks a checkbox or radio button.
//
// Takes selector (string) which identifies the element to check.
//
// Returns *Page which allows method chaining.
func (p *Page) Check(selector string) *Page {
	p.beforeAction("Check", selector)
	start := time.Now()
	err := browser_provider_chromedp.Check(p.actionCtx(), selector)
	p.afterAction("Check", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Check(%q) failed: %v", selector, err)
	}
	return p
}

// Uncheck unchecks a checkbox identified by the given selector.
//
// Takes selector (string) which identifies the checkbox element to uncheck.
//
// Returns *Page which allows method chaining for further actions.
func (p *Page) Uncheck(selector string) *Page {
	p.beforeAction("Uncheck", selector)
	start := time.Now()
	err := browser_provider_chromedp.Uncheck(p.actionCtx(), selector)
	p.afterAction("Uncheck", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Uncheck(%q) failed: %v", selector, err)
	}
	return p
}

// Focus sets keyboard focus on the element matching the selector.
//
// Takes selector (string) which identifies the element to focus.
//
// Returns *Page which allows method chaining.
func (p *Page) Focus(selector string) *Page {
	p.beforeAction("Focus", selector)
	start := time.Now()
	err := browser_provider_chromedp.Focus(p.actionCtx(), selector)
	p.afterAction("Focus", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Focus(%q) failed: %v", selector, err)
	}
	return p
}

// Blur removes focus from an element.
//
// Takes selector (string) which identifies the element to blur.
//
// Returns *Page which allows method chaining.
func (p *Page) Blur(selector string) *Page {
	p.beforeAction("Blur", selector)
	start := time.Now()
	err := browser_provider_chromedp.Blur(p.actionCtx(), selector)
	p.afterAction("Blur", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Blur(%q) failed: %v", selector, err)
	}
	return p
}

// SetFiles sets files on a file input element.
//
// Takes selector (string) which identifies the file input element.
// Takes paths (...string) which specifies the file paths to set.
//
// Returns *Page which allows method chaining.
func (p *Page) SetFiles(selector string, paths ...string) *Page {
	detail := fmt.Sprintf("%s (%d files)", selector, len(paths))
	p.beforeAction("SetFiles", detail)
	start := time.Now()
	err := browser_provider_chromedp.SetFiles(p.actionCtx(), selector, paths)
	p.afterAction("SetFiles", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetFiles(%q) failed: %v", selector, err)
	}
	return p
}

// Scroll scrolls the window or an element to a position.
//
// Takes selector (string) which identifies the element to scroll.
// Takes position (int) which specifies the scroll offset in pixels.
//
// Returns *Page which allows method chaining.
func (p *Page) Scroll(selector string, position int) *Page {
	detail := fmt.Sprintf("%s -> %d", selector, position)
	p.beforeAction("Scroll", detail)
	start := time.Now()
	err := browser_provider_chromedp.Scroll(p.actionCtx(), selector, fmt.Sprintf("%d", position))
	p.afterAction("Scroll", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Scroll(%q, %d) failed: %v", selector, position, err)
	}
	return p
}

// Press presses one or more keys, supporting modifiers.
//
// Key format: "Enter", "Tab", "Shift+Enter", "Control+b", "Meta+k".
// Multiple keys can be pressed in sequence by passing multiple arguments.
//
// Takes keys (...string) which specifies the keys to press in sequence.
//
// Returns *Page which allows method chaining.
func (p *Page) Press(keys ...string) *Page {
	detail := strings.Join(keys, ", ")
	p.beforeAction("Press", detail)
	start := time.Now()
	err := browser_provider_chromedp.Press(p.actionCtx(), keys...)
	p.afterAction("Press", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Press(%q) failed: %v", detail, err)
	}
	return p
}

// Type enters text character by character at the current cursor position.
// Unlike Fill, this does not clear existing content.
//
// Takes text (string) which is the text to type character by character.
//
// Returns *Page which allows method chaining.
func (p *Page) Type(text string) *Page {
	displayText := truncateRunes(text, displayTextMaxLen)
	p.beforeAction("Type", displayText)
	start := time.Now()
	err := browser_provider_chromedp.Type(p.actionCtx(), text)
	p.afterAction("Type", displayText, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Type(%q) failed: %v", displayText, err)
	}
	return p
}

// PressAndHold holds a key down for modifier combinations or hold actions.
// Use Release to let go of the held key.
//
// Takes key (string) which specifies the key to hold down.
//
// Returns *Page which allows method chaining.
func (p *Page) PressAndHold(key string) *Page {
	p.beforeAction("PressAndHold", key)
	start := time.Now()
	err := browser_provider_chromedp.KeyDown(p.actionCtx(), key)
	p.afterAction("PressAndHold", key, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PressAndHold(%q) failed: %v", key, err)
	}
	return p
}

// Release releases a key that was held down with PressAndHold.
//
// Takes key (string) which specifies the key to release.
//
// Returns *Page which allows method chaining.
func (p *Page) Release(key string) *Page {
	p.beforeAction("Release", key)
	start := time.Now()
	err := browser_provider_chromedp.KeyUp(p.actionCtx(), key)
	p.afterAction("Release", key, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("Release(%q) failed: %v", key, err)
	}
	return p
}

// SetCursor sets the cursor position within a contenteditable element.
// The offset is counted as characters from the start of the text content.
//
// Takes selector (string) which identifies the contenteditable element.
// Takes offset (int) which specifies the character position from the start.
//
// Returns *Page which allows method chaining.
func (p *Page) SetCursor(selector string, offset int) *Page {
	detail := fmt.Sprintf("%s @ %d", selector, offset)
	p.beforeAction("SetCursor", detail)
	start := time.Now()
	err := browser_provider_chromedp.SetCursorPosition(p.actionCtx(), selector, offset)
	p.afterAction("SetCursor", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SetCursor(%q, %d) failed: %v", selector, offset, err)
	}
	return p
}

// Select selects text within an element by start and end offsets.
// Offsets are counted as characters from the start of the text content.
//
// Takes selector (string) which identifies the element containing the text.
// Takes start (int) which specifies the character offset to begin selection.
// Takes end (int) which specifies the character offset to end selection.
//
// Returns *Page which enables method chaining.
func (p *Page) Select(selector string, start, end int) *Page {
	detail := fmt.Sprintf("%s [%d:%d]", selector, start, end)
	p.beforeAction("Select", detail)
	startTime := time.Now()
	err := browser_provider_chromedp.SetSelection(p.actionCtx(), selector, start, end)
	p.afterAction("Select", detail, err != nil, time.Since(startTime))
	if err != nil {
		p.t.Fatalf("Select(%q, %d, %d) failed: %v", selector, start, end, err)
	}
	return p
}

// SelectAll selects all content within the element matching the selector.
//
// Takes selector (string) which identifies the target element.
//
// Returns *Page which enables method chaining.
func (p *Page) SelectAll(selector string) *Page {
	p.beforeAction("SelectAll", selector)
	start := time.Now()
	err := browser_provider_chromedp.SelectAll(p.actionCtx(), selector)
	p.afterAction("SelectAll", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("SelectAll(%q) failed: %v", selector, err)
	}
	return p
}

// CollapseToStart collapses the current selection to its start.
//
// Returns *Page which allows method chaining.
func (p *Page) CollapseToStart() *Page {
	p.beforeAction("CollapseToStart", "")
	start := time.Now()
	err := browser_provider_chromedp.CollapseSelection(p.actionCtx(), false)
	p.afterAction("CollapseToStart", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("CollapseToStart() failed: %v", err)
	}
	return p
}

// CollapseToEnd collapses the current selection to its end.
//
// Returns *Page which allows method chaining.
func (p *Page) CollapseToEnd() *Page {
	p.beforeAction("CollapseToEnd", "")
	start := time.Now()
	err := browser_provider_chromedp.CollapseSelection(p.actionCtx(), true)
	p.afterAction("CollapseToEnd", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("CollapseToEnd() failed: %v", err)
	}
	return p
}

// PlaceCursorInElement places the cursor inside a child element of a parent
// element, programmatically positioning the cursor inside inline elements
// where Click() may not reliably position the cursor due to browser caret
// behaviour.
//
// Takes parentSelector (string) which is the selector for the parent/container
// element (supports shadow DOM with >>>).
// Takes childSelector (string) which is the CSS selector for the child element
// within the parent.
//
// Returns *Page which enables method chaining.
//
// The cursor is placed at offset 1 within the first text node of the child
// element, or at offset 0 if the text is empty.
func (p *Page) PlaceCursorInElement(parentSelector, childSelector string) *Page {
	detail := fmt.Sprintf("%s -> %s", parentSelector, childSelector)
	p.beforeAction("PlaceCursorInElement", detail)
	start := time.Now()
	err := browser_provider_chromedp.PlaceCursorInElement(p.actionCtx(), parentSelector, childSelector)
	p.afterAction("PlaceCursorInElement", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PlaceCursorInElement(%q, %q) failed: %v", parentSelector, childSelector, err)
	}
	return p
}

// GetCursorOffset returns the current cursor position within an element.
// Returns -1 if the cursor is not within the element.
//
// Takes selector (string) which identifies the element to check.
//
// Returns int which is the cursor offset position, or -1 if not found.
func (p *Page) GetCursorOffset(selector string) int {
	offset, err := browser_provider_chromedp.GetCursorPosition(p.actionCtx(), selector)
	if err != nil {
		p.t.Fatalf("GetCursorOffset(%q) failed: %v", selector, err)
	}
	return offset
}

// GetSelectionRange returns the current selection start and end offsets within
// an element. Returns (-1, -1) if there is no selection within the element.
//
// Takes selector (string) which identifies the element to query.
//
// Returns start (int) which is the offset of the selection start.
// Returns end (int) which is the offset of the selection end.
func (p *Page) GetSelectionRange(selector string) (start, end int) {
	start, end, err := browser_provider_chromedp.GetSelection(p.actionCtx(), selector)
	if err != nil {
		p.t.Fatalf("GetSelectionRange(%q) failed: %v", selector, err)
	}
	return start, end
}
