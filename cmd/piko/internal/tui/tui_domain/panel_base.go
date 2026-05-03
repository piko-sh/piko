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
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	// ellipsis is the text shown when content is cut short to fit a width limit.
	ellipsis = "..."

	// ellipsisLength is the length of the ellipsis string in characters.
	ellipsisLength = len(ellipsis)

	// metadataDefaultIndent is the default number of spaces used to indent
	// metadata values.
	metadataDefaultIndent = 6

	// metadataDefaultWidthAdj is the default width adjustment for metadata rows.
	metadataDefaultWidthAdj = 10

	// metadataSelectedOffset is the number of spaces to reduce from the indent
	// when an item is selected.
	metadataSelectedOffset = 2
)

// BasePanel provides shared behaviour for all panels in the TUI.
// Embed in panel types to get default key bindings, focus handling, and scroll
// support.
type BasePanel struct {
	// id is the unique identifier for this panel.
	id string

	// title is the text shown in the panel header.
	title string

	// titleSuffix is appended to the title in RenderFrame so panels
	// can surface live status (e.g. scroll position) without changing
	// the stable panel title.
	titleSuffix string

	// keyBindings holds the key bindings for this panel.
	keyBindings []KeyBinding

	// width is the panel width in columns.
	width int

	// height is the panel height in lines.
	height int

	// cursor is the zero-based position of the selected item in the
	// list.
	cursor int

	// scrollOffset is the current vertical scroll position for
	// scrollable content.
	scrollOffset int

	// focused indicates whether the panel has focus.
	focused bool
}

// NewBasePanel creates a new BasePanel with the given ID and title.
//
// Takes id (string) which specifies the unique identifier for the panel.
// Takes title (string) which specifies the display title for the panel.
//
// Returns BasePanel which is initialised with default values for dimensions,
// cursor position, and focus state.
func NewBasePanel(id, title string) BasePanel {
	return BasePanel{
		id:           id,
		title:        title,
		keyBindings:  nil,
		width:        0,
		height:       0,
		cursor:       0,
		scrollOffset: 0,
		focused:      false,
	}
}

// SetTitleSuffix replaces the dim suffix appended to the panel title in
// RenderFrame. Pass an empty string to clear it.
//
// Takes suffix (string) which is the new title suffix.
func (p *BasePanel) SetTitleSuffix(suffix string) {
	p.titleSuffix = suffix
}

// ID returns the panel identifier.
//
// Returns string which is the unique identifier for this panel.
func (p *BasePanel) ID() string {
	return p.id
}

// Title returns the display title for this panel.
//
// Returns string which is the panel title shown in the user interface.
func (p *BasePanel) Title() string {
	return p.title
}

// Focused returns whether the panel is focused.
//
// Returns bool which is true if the panel currently has focus.
func (p *BasePanel) Focused() bool {
	return p.focused
}

// SetFocused sets the panel's focus state.
//
// Takes focused (bool) which indicates whether the panel has focus.
func (p *BasePanel) SetFocused(focused bool) {
	p.focused = focused
}

// SetSize sets the panel dimensions.
//
// Takes width (int) which specifies the panel width.
// Takes height (int) which specifies the panel height.
func (p *BasePanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Width returns the panel width.
//
// Returns int which is the width in pixels.
func (p *BasePanel) Width() int {
	return p.width
}

// Height returns the panel height in pixels.
//
// Returns int which is the height in pixels.
func (p *BasePanel) Height() int {
	return p.height
}

// ContentHeight returns the height available for content, accounting for
// borders and title.
//
// Returns int which is the usable height in lines, or zero if the panel is
// too small.
func (p *BasePanel) ContentHeight() int {
	return max(0, p.height-PanelChromeHeight)
}

// ContentWidth returns the width available for content, after accounting for
// borders and padding.
//
// Returns int which is the usable width inside the panel.
func (p *BasePanel) ContentWidth() int {
	return max(0, p.width-PanelChromeWidth)
}

// Cursor returns the current cursor position.
//
// Returns int which is the zero-based cursor position within the panel.
func (p *BasePanel) Cursor() int {
	return p.cursor
}

// SetCursor sets the cursor position.
//
// Takes position (int) which specifies the new cursor position.
func (p *BasePanel) SetCursor(position int) {
	p.cursor = position
}

// ScrollOffset returns the current vertical scroll position.
//
// Returns int which is the current vertical scroll offset.
func (p *BasePanel) ScrollOffset() int {
	return p.scrollOffset
}

// SetScrollOffset sets the scroll offset.
//
// Takes offset (int) which specifies the new scroll position.
func (p *BasePanel) SetScrollOffset(offset int) {
	p.scrollOffset = offset
}

// KeyMap returns the panel's key bindings.
//
// Returns []KeyBinding which contains the configured key bindings for this
// panel.
func (p *BasePanel) KeyMap() []KeyBinding {
	return p.keyBindings
}

// SetKeyMap sets the key bindings for this panel.
//
// Takes bindings ([]KeyBinding) which specifies the key bindings to use.
func (p *BasePanel) SetKeyMap(bindings []KeyBinding) {
	p.keyBindings = bindings
}

// Init returns nil to indicate there is no initial command.
//
// Returns tea.Cmd which is nil by default.
func (*BasePanel) Init() tea.Cmd {
	return nil
}

// DetailView is the default implementation of Panel.DetailView.
//
// It returns the empty string so the composer falls back to its
// placeholder. Panels that produce real detail content shadow this
// method with their own implementation.
//
// Takes width (int) and height (int) which are ignored by the
// default; concrete implementations use them.
//
// Returns string which is always "" for the default.
func (*BasePanel) DetailView(_, _ int) string {
	return ""
}

// Selection is the default implementation of Panel.Selection.
//
// It returns the empty Selection so panels without selectable rows can embed
// BasePanel without adding boilerplate. Panels with row selection shadow the
// default.
//
// Returns Selection which is the zero value.
func (*BasePanel) Selection() Selection {
	return Selection{}
}

// HandleNavigation handles common navigation keys for cursor movement.
//
// Takes message (tea.KeyPressMsg) which is the key press to handle.
// Takes itemCount (int) which is the total number of items to navigate.
//
// Returns bool which is true if the key was handled.
func (p *BasePanel) HandleNavigation(message tea.KeyPressMsg, itemCount int) bool {
	keyString := message.String()

	switch keyString {
	case "up", "k":
		if p.cursor > 0 {
			p.cursor--
			p.ensureCursorVisible(itemCount)
		}
		return true
	case "down", "j":
		if p.cursor < itemCount-1 {
			p.cursor++
			p.ensureCursorVisible(itemCount)
		}
		return true
	case "g":
		p.cursor = 0
		p.scrollOffset = 0
		return true
	case "G":
		p.cursor = max(0, itemCount-1)
		p.ensureCursorVisible(itemCount)
		return true
	case "pgup", "ctrl+u":
		p.cursor = max(0, p.cursor-p.ContentHeight())
		p.ensureCursorVisible(itemCount)
		return true
	case "pgdown", "ctrl+d":
		p.cursor = min(itemCount-1, p.cursor+p.ContentHeight())
		p.ensureCursorVisible(itemCount)
		return true
	}
	return false
}

// RenderFrame renders the panel frame with a title and content.
//
// Takes content (string) which is the body text to display inside the panel.
//
// Returns string which is the styled panel with its title and content.
func (p *BasePanel) RenderFrame(content string) string {
	style := panelStyle
	if p.focused {
		style = panelFocusedStyle
	}

	title := panelTitleStyle.Render(p.title)
	if p.titleSuffix != "" {
		dim := lipgloss.NewStyle().Foreground(colourForegroundDim)
		title += " " + dim.Render(p.titleSuffix)
	}

	full := lipgloss.JoinVertical(lipgloss.Left, title, content)

	return style.
		Width(p.ContentWidth()).
		Height(p.ContentHeight() + 1).
		Render(full)
}

// ensureCursorVisible adjusts the scroll offset so the cursor stays visible.
//
// Takes itemCount (int) which specifies the total number of items in the list.
func (p *BasePanel) ensureCursorVisible(itemCount int) {
	visibleHeight := p.ContentHeight()
	if visibleHeight <= 0 {
		return
	}

	p.cursor = max(0, min(p.cursor, itemCount-1))

	if p.cursor < p.scrollOffset {
		p.scrollOffset = p.cursor
	}

	if p.cursor >= p.scrollOffset+visibleHeight {
		p.scrollOffset = p.cursor - visibleHeight + 1
	}

	maxScroll := max(0, itemCount-visibleHeight)
	p.scrollOffset = max(0, min(p.scrollOffset, maxScroll))
}

// MetadataRowConfig holds settings for displaying a metadata row.
type MetadataRowConfig struct {
	// IndentSpaces is the number of spaces for indentation; 0 uses the default.
	IndentSpaces int

	// WidthAdjustment adjusts the width for content; 0 uses the default.
	WidthAdjustment int

	// Selected indicates whether this row is the current active item.
	Selected bool

	// Focused indicates whether the parent panel has keyboard focus.
	Focused bool

	// ContentWidth is the maximum width in characters for the content area.
	ContentWidth int
}

// StatusIndicator returns a styled symbol that shows the health state of a
// resource.
//
// Takes status (ResourceStatus) which specifies the health state to display.
//
// Returns string which is the styled indicator symbol.
func StatusIndicator(status ResourceStatus) string {
	switch status {
	case ResourceStatusHealthy:
		return statusHealthyStyle.Render(SymbolStatusFilled)
	case ResourceStatusDegraded:
		return statusDegradedStyle.Render(SymbolStatusFilled)
	case ResourceStatusUnhealthy:
		return statusUnhealthyStyle.Render(SymbolStatusFilled)
	case ResourceStatusPending:
		return statusPendingStyle.Render(SymbolStatusEmpty)
	default:
		return statusUnknownStyle.Render(SymbolStatusEmpty)
	}
}

// StatusStyle returns the visual style for a resource status.
//
// Takes status (ResourceStatus) which specifies the status to style.
//
// Returns lipgloss.Style which is the style for the given status.
func StatusStyle(status ResourceStatus) lipgloss.Style {
	switch status {
	case ResourceStatusHealthy:
		return statusHealthyStyle
	case ResourceStatusDegraded:
		return statusDegradedStyle
	case ResourceStatusUnhealthy:
		return statusUnhealthyStyle
	case ResourceStatusPending:
		return statusPendingStyle
	default:
		return statusUnknownStyle
	}
}

// TruncateString shortens a string to fit within a given visible width,
// adding an ellipsis if needed. The width is measured in terminal cells, not
// bytes, so multi-byte UTF-8 sequences and embedded ANSI escapes are handled
// correctly.
//
// Takes s (string) which is the string to shorten.
// Takes maxWidth (int) which is the maximum number of terminal cells allowed.
//
// Returns string which is the shortened string with an ellipsis, or the
// original string if it fits within maxWidth.
func TruncateString(s string, maxWidth int) string {
	return TruncateANSI(s, maxWidth)
}

// PadRight pads a string to the given visible width with spaces on the right.
// The width is measured in terminal cells; multi-byte UTF-8 sequences and
// embedded ANSI escapes are handled correctly.
//
// Takes s (string) which is the string to pad.
// Takes width (int) which is the target width in terminal cells.
//
// Returns string which is the padded string. If the input is longer than the
// target width, it is cut short to fit.
func PadRight(s string, width int) string {
	return PadRightANSI(s, width)
}

// RenderMetadataRow renders a metadata key-value row with styling and
// indentation for terminal display.
//
// Takes key (string) which is the label for the row.
// Takes value (string) which is the content to display.
// Takes config (MetadataRowConfig) which controls indentation, width, and
// focus state.
//
// Returns string which is the styled row ready for display.
func RenderMetadataRow(key, value string, config MetadataRowConfig) string {
	indentSpaces := config.IndentSpaces
	if indentSpaces == 0 {
		indentSpaces = metadataDefaultIndent
	}

	cursor := buildMetadataCursor(indentSpaces, config.Selected, config.Focused)

	widthAdj := config.WidthAdjustment
	if widthAdj == 0 {
		widthAdj = metadataDefaultWidthAdj
	}
	content := fmt.Sprintf("%s: %s", key, TruncateString(value, config.ContentWidth-TextWidth(key)-widthAdj))

	style := lipgloss.NewStyle().Foreground(colourForegroundDim)
	if config.Selected && config.Focused {
		style = style.Foreground(colourForeground)
	}

	return cursor + style.Render(content)
}

// buildMetadataCursor creates the cursor string for a metadata row.
//
// Takes indentSpaces (int) which sets how many spaces to use for indentation.
// Takes selected (bool) which shows whether the row is selected.
// Takes focused (bool) which shows whether the row has focus.
//
// Returns string which contains the cursor with the correct indent and style.
func buildMetadataCursor(indentSpaces int, selected, focused bool) string {
	if !selected {
		return strings.Repeat(SingleSpace, indentSpaces)
	}

	selectedIndent := max(0, indentSpaces-metadataSelectedOffset)
	if focused {
		return strings.Repeat(SingleSpace, selectedIndent) + lipgloss.NewStyle().Foreground(colourPrimary).Render("▸ ")
	}
	return strings.Repeat(SingleSpace, selectedIndent) + "▸ "
}
