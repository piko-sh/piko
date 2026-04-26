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

package tui_domain

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var _ ItemRenderer[any] = (*SimpleRenderer[any])(nil)

// ItemRenderer defines how to render items of a specific type.
// Implement this interface for each panel's data type.
type ItemRenderer[T any] interface {
	// RenderRow renders the main row for an item in the list view.
	//
	// Takes item (T) which is the item to render.
	// Takes lineIndex (int) which is the line position in the view for cursor
	// comparison.
	// Takes selected (bool) which indicates if this item is at the cursor position.
	// Takes focused (bool) which indicates if the panel is focused.
	// Takes width (int) which is the available content width.
	//
	// Returns string which is the rendered row content.
	RenderRow(item T, lineIndex int, selected, focused bool, width int) string

	// RenderExpanded renders detail lines for an expanded item. Returns nil or an
	// empty slice if the item is not expandable or has no details.
	//
	// Takes item (T) which is the item to render in expanded form.
	// Takes width (int) which specifies the available width in characters.
	//
	// Returns []string which contains one line per element for display.
	RenderExpanded(item T, width int) []string

	// GetID returns a unique identifier for the item, used for tracking
	// expansion state.
	//
	// Takes item (T) which is the item to get the identifier for.
	//
	// Returns string which is the unique identifier.
	GetID(item T) string

	// MatchesFilter checks whether the item matches the search query.
	// The query is already in lowercase.
	//
	// Takes item (T) which is the item to check.
	// Takes query (string) which is the lowercase search text.
	//
	// Returns bool which is true if the item matches the query.
	MatchesFilter(item T, query string) bool

	// IsExpandable returns true if the given item can be expanded to show more
	// details.
	//
	// Takes item (T) which is the item to check.
	//
	// Returns bool which is true if the item can be expanded, false otherwise.
	IsExpandable(item T) bool

	// ExpandedLineCount returns the number of detail lines when expanded.
	// This should match len(RenderExpanded(item, width)) for correct behaviour.
	//
	// Takes item (T) which is the item to count lines for.
	//
	// Returns int which is the number of lines when expanded.
	ExpandedLineCount(item T) int
}

// ScrollContext tracks visible lines and handles newlines when rendering
// scrollable content.
type ScrollContext struct {
	// content builds rendered lines for the visible part of the view.
	content *strings.Builder

	// visibleStart is the index of the first visible line in the scroll window.
	visibleStart int

	// visibleEnd is the line index where the visible area ends (exclusive).
	visibleEnd int

	// lineIndex tracks the current line position during rendering.
	lineIndex int

	// firstLine indicates whether the next write is the first line of output.
	firstLine bool
}

// NewScrollContext creates a new scroll context for rendering.
// visibleStart and visibleEnd define the range of visible line indices.
//
// Takes content (*strings.Builder) which holds the rendered output.
// Takes scrollOffset (int) which specifies the first visible line index.
// Takes visibleHeight (int) which specifies the number of visible lines.
//
// Returns *ScrollContext which is ready for use in scroll-aware rendering.
func NewScrollContext(content *strings.Builder, scrollOffset, visibleHeight int) *ScrollContext {
	return &ScrollContext{
		content:      content,
		visibleStart: scrollOffset,
		visibleEnd:   scrollOffset + visibleHeight,
		lineIndex:    0,
		firstLine:    true,
	}
}

// WriteLineIfVisible writes a line if it falls within the visible range.
//
// The lineFunc is only called if the line is visible. This avoids extra work
// when the line is outside the viewport.
//
// Takes lineFunc (func() string) which provides the line content when called.
func (ctx *ScrollContext) WriteLineIfVisible(lineFunc func() string) {
	if ctx.lineIndex < ctx.visibleStart || ctx.lineIndex >= ctx.visibleEnd {
		ctx.lineIndex++
		return
	}

	if !ctx.firstLine {
		ctx.content.WriteString(stringNewline)
	}
	ctx.firstLine = false

	if line := lineFunc(); line != "" {
		ctx.content.WriteString(line)
	}
	ctx.lineIndex++
}

// WriteLine writes a line to the output if it falls within the visible range.
// Use this when the line content is ready.
//
// Takes line (string) which is the content to write.
func (ctx *ScrollContext) WriteLine(line string) {
	if ctx.lineIndex < ctx.visibleStart || ctx.lineIndex >= ctx.visibleEnd {
		ctx.lineIndex++
		return
	}

	if !ctx.firstLine {
		ctx.content.WriteString(stringNewline)
	}
	ctx.firstLine = false

	if line != "" {
		ctx.content.WriteString(line)
	}
	ctx.lineIndex++
}

// SkipLines moves the line index forward without writing anything.
// Use this for lines that are known to be outside the visible range.
//
// Takes count (int) which specifies the number of lines to skip.
func (ctx *ScrollContext) SkipLines(count int) {
	ctx.lineIndex += count
}

// LineIndex returns the current line index.
//
// Returns int which is the zero-based index of the current line.
func (ctx *ScrollContext) LineIndex() int {
	return ctx.lineIndex
}

// IsVisible reports whether the current line index is within the visible range.
//
// Returns bool which is true when the line is visible, false otherwise.
func (ctx *ScrollContext) IsVisible() bool {
	return ctx.lineIndex >= ctx.visibleStart && ctx.lineIndex < ctx.visibleEnd
}

// SimpleRenderer provides a minimal ItemRenderer implementation using functions.
// This reduces boilerplate for panels that don't need a full struct-based
// renderer.
//
// Required fields: GetIDFunction, MatchesFilterFunction, RenderRowFunction.
// Optional fields: RenderExpandedFunction, IsExpandableFunction,
// ExpandedCountFunction.
//
// Example usage:
//
//	Renderer: &SimpleRenderer[MyItem]{
//	    GetIDFunction:         func(item MyItem) string { return item.ID },
//	    MatchesFilterFunction: func(item MyItem, q string) bool {
//	        return strings.Contains(item.Name, q)
//	    },
//	    RenderRowFunction:     func(item MyItem, _, selected, focused bool, w int) string {
//	        return RenderCursorStyled(selected, focused, DefaultCursorConfig()) +
//	            item.Name
//	    },
//	},
type SimpleRenderer[T any] struct {
	// GetIDFunction returns a unique identifier for the item. Required.
	GetIDFunction func(T) string

	// MatchesFilterFunction returns true if the item matches the
	// search query; required. The query is already lowercased.
	MatchesFilterFunction func(T, string) bool

	// RenderRowFunction renders the main row for an item; this field is required.
	RenderRowFunction func(T, int, bool, bool, int) string

	// RenderExpandedFunction renders detail lines for an expanded
	// item. Optional; return nil if it has no content to expand.
	RenderExpandedFunction func(T, int) []string

	// IsExpandableFunction checks whether an item can be expanded. Optional; defaults
	// to false if nil.
	IsExpandableFunction func(T) bool

	// ExpandedCountFunction returns the number of detail lines when
	// expanded. Optional; defaults to 0 if nil.
	ExpandedCountFunction func(T) int
}

// GetID returns the unique identifier for the given item. Implements
// ItemRenderer.
//
// Takes item (T) which is the item to get the identifier for.
//
// Returns string which is the unique identifier for the item.
func (r *SimpleRenderer[T]) GetID(item T) string {
	return r.GetIDFunction(item)
}

// MatchesFilter implements ItemRenderer.
//
// Takes item (T) which is the item to test against the filter.
// Takes query (string) which is the filter query to match against.
//
// Returns bool which is true when the item matches the query.
func (r *SimpleRenderer[T]) MatchesFilter(item T, query string) bool {
	return r.MatchesFilterFunction(item, query)
}

// RenderRow implements ItemRenderer.
//
// Takes item (T) which is the data item to render.
// Takes lineIndex (int) which is the position in the list.
// Takes selected (bool) which indicates if the item is selected.
// Takes focused (bool) which indicates if the item has focus.
// Takes width (int) which is the available width in characters.
//
// Returns string which is the rendered row content.
func (r *SimpleRenderer[T]) RenderRow(item T, lineIndex int, selected, focused bool, width int) string {
	return r.RenderRowFunction(item, lineIndex, selected, focused, width)
}

// RenderExpanded implements ItemRenderer.
//
// Takes item (T) which is the item to render in expanded form.
// Takes width (int) which specifies the available width in characters.
//
// Returns []string which contains the rendered lines, or nil if no render
// function is configured.
func (r *SimpleRenderer[T]) RenderExpanded(item T, width int) []string {
	if r.RenderExpandedFunction == nil {
		return nil
	}
	return r.RenderExpandedFunction(item, width)
}

// IsExpandable implements ItemRenderer.
//
// Takes item (T) which is the item to check for expandability.
//
// Returns bool which is true if the item can be expanded.
func (r *SimpleRenderer[T]) IsExpandable(item T) bool {
	if r.IsExpandableFunction == nil {
		return false
	}
	return r.IsExpandableFunction(item)
}

// ExpandedLineCount returns the number of lines when an item is expanded.
// Implements ItemRenderer.
//
// Takes item (T) which is the item to measure.
//
// Returns int which is the line count, or zero if no expand function is set.
func (r *SimpleRenderer[T]) ExpandedLineCount(item T) int {
	if r.ExpandedCountFunction == nil {
		return 0
	}
	return r.ExpandedCountFunction(item)
}

// RenderCursor returns the cursor indicator string.
//
// Takes selected (bool) which specifies whether the item is selected.
// Takes focused (bool) which specifies whether the item has focus.
//
// Returns string which is a styled indicator if selected and focused, plain
// if selected but unfocused, or spaces if not selected.
//
// This is a convenience wrapper around RenderCursorStyled with
// DefaultCursorConfig.
func RenderCursor(selected, focused bool) string {
	return RenderCursorStyled(selected, focused, DefaultCursorConfig())
}

// RenderCursorWithIndent returns the cursor indicator with custom indent strings
// based on selection and focus state.
//
// Takes selected (bool) which indicates whether the item is currently selected.
// Takes focused (bool) which indicates whether the item has focus.
// Takes inactiveIndent (string) which is shown when the item is not selected.
// Takes activeIndent (string) which is combined with the cursor when selected.
//
// Returns string which contains the appropriate indent and cursor indicator.
func RenderCursorWithIndent(selected, focused bool, inactiveIndent, activeIndent string) string {
	if !selected {
		return inactiveIndent
	}
	if focused {
		return activeIndent + lipgloss.NewStyle().Foreground(colourPrimary).Render(cursorIndicator)
	}
	return activeIndent + cursorIndicator
}

// RenderExpandIndicator returns the expand or collapse indicator symbol.
//
// Takes expanded (bool) which sets whether to show the expanded or collapsed
// state.
//
// Returns string which contains SymbolExpanded if expanded, or SymbolCollapsed
// if collapsed, styled with a dim colour.
func RenderExpandIndicator(expanded bool) string {
	indicator := SymbolCollapsed
	if expanded {
		indicator = SymbolExpanded
	}
	return lipgloss.NewStyle().Foreground(colourForegroundDim).Render(indicator)
}

// RenderName formats an item name for display with optional bold styling.
//
// The name is truncated if longer than maxWidth and padded with spaces to fill
// the width. When both selected and focused are true, the text is styled bold.
//
// Takes name (string) which is the text to display.
// Takes maxWidth (int) which is the width to truncate and pad to.
// Takes selected (bool) which shows if the item is selected.
// Takes focused (bool) which shows if the item has focus.
//
// Returns string which is the styled name, truncated and padded to maxWidth.
func RenderName(name string, maxWidth int, selected, focused bool) string {
	truncated := TruncateString(name, maxWidth)
	padded := PadRight(truncated, maxWidth)
	if selected && focused {
		return lipgloss.NewStyle().Bold(true).Render(padded)
	}
	return padded
}

// RenderDimText renders text with a dim foreground colour.
//
// Takes text (string) which is the text to render.
//
// Returns string which is the styled text with dim colour applied.
func RenderDimText(text string) string {
	return lipgloss.NewStyle().Foreground(colourForegroundDim).Render(text)
}

// RenderItalicDimText renders text with dim colour and italic styling.
//
// Takes text (string) which is the content to style.
//
// Returns string which is the styled text.
func RenderItalicDimText(text string) string {
	return lipgloss.NewStyle().
		Foreground(colourForegroundDim).
		Italic(true).
		Render(text)
}

// RenderErrorText renders text with the error colour style.
//
// Takes text (string) which is the content to render.
//
// Returns string which is the text styled with the error colour.
func RenderErrorText(text string) string {
	return lipgloss.NewStyle().Foreground(colourError).Render(text)
}

// RenderInfoText applies the info colour style to the given text.
//
// Takes text (string) which is the text to style.
//
// Returns string which is the text with the info colour applied.
func RenderInfoText(text string) string {
	return lipgloss.NewStyle().Foreground(colourInfo).Render(text)
}

// RenderEmptyState renders a standard empty state message.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes hasFilter (bool) which shows if a filter is active.
// Takes itemName (string) which names the type of item being shown.
func RenderEmptyState(content *strings.Builder, hasFilter bool, itemName string) {
	message := "No " + itemName + " available"
	if hasFilter {
		message = "No " + itemName + " match filter"
	}
	content.WriteString(lipgloss.NewStyle().
		Foreground(colourForegroundDim).
		Render(message))
}

// RenderErrorState writes a styled error message to the given builder.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes err (error) which provides the error to display.
func RenderErrorState(content *strings.Builder, err error) {
	content.WriteString(lipgloss.NewStyle().
		Foreground(colourError).
		Render("Error: " + err.Error()))
	content.WriteString(stringNewline)
}
