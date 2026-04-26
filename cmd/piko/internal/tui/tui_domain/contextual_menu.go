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
)

// MenuMarker is the cursor glyph rendered next to the active menu item.
//
// Centralised so future tweaks (e.g. a Nerd Font glyph) land in one
// place.
const MenuMarker = "▸"

// ContextualMenu renders the left-column item list for the active group.
//
// It owns a cursor (highlighted-but-not-active position) and an active
// item id (the item whose centre/detail is currently displayed). The
// two can differ: pressing j/k moves the cursor without committing;
// pressing Enter commits the highlighted item as active.
type ContextualMenu struct {
	// theme provides the styles for label / cursor / hotkey / badge.
	theme *Theme
}

// NewContextualMenu returns a menu renderer bound to theme.
//
// Takes theme (*Theme) which is the active theme.
//
// Returns *ContextualMenu ready to call Render on.
func NewContextualMenu(theme *Theme) *ContextualMenu {
	return &ContextualMenu{theme: theme}
}

// SetTheme replaces the active theme.
//
// Takes theme (*Theme) which is the new theme; never nil.
func (m *ContextualMenu) SetTheme(theme *Theme) { m.theme = theme }

// Render produces the menu body sized to (width, height). Items beyond
// height are clipped; future work introduces scrolling for very tall
// groups (none of the four built-in groups exceeds eight items so this
// is deferred).
//
// Takes items ([]MenuItem) which is the group's ordered item list.
// Takes activeID (ItemID) which is the currently-active item.
// Takes cursor (int) which is the highlighted-but-not-active index.
// Takes width (int) and height (int) which are the column dimensions.
//
// Returns string with the rendered menu body.
func (m *ContextualMenu) Render(items []MenuItem, activeID ItemID, cursor, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	rows := make([]string, 0, height)
	for i, item := range items {
		if i >= height {
			break
		}
		rows = append(rows, m.renderItem(item, i, activeID, cursor, width))
	}

	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", width))
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

// renderItem renders a single menu row.
//
// Takes item (MenuItem) which is the row being rendered.
// Takes index (int) which is the row's position in the list.
// Takes activeID (ItemID) which is the currently-active item id.
// Takes cursor (int) which is the highlighted-but-not-active index.
// Takes width (int) which is the column width to pad the row to.
//
// Returns string with the styled, padded row.
func (m *ContextualMenu) renderItem(item MenuItem, index int, activeID ItemID, cursor, width int) string {
	hotkey := displayHotkey(item.Hotkey)
	label := item.Label
	badge := item.Badge

	prefix := DoubleSpace
	switch {
	case item.ID == activeID:
		prefix = MenuMarker + SingleSpace
	case index == cursor:
		prefix = "› "
	}

	line := prefix
	if hotkey != "" {
		line += m.styleHotkey(hotkey) + SingleSpace
	}
	line += label
	if !badge.IsEmpty() {
		line += SingleSpace + m.renderBadge(badge)
	}

	switch {
	case item.ID == activeID:
		line = m.styleActive(line)
	case index == cursor:
		line = m.styleCursor(line)
	default:
		line = m.styleIdle(line)
	}

	return PadRightANSI(line, width)
}

// renderBadge renders a Badge with the configured theme.
//
// Takes badge (Badge) which is the badge to render.
//
// Returns string which is the styled badge, or empty when the badge
// has no glyph and no count.
func (m *ContextualMenu) renderBadge(badge Badge) string {
	if badge.Glyph != "" {
		return m.severityStyle(badge.Severity).Render(badge.Glyph)
	}
	if badge.Count > 0 {
		return m.severityStyle(badge.Severity).Render(fmt.Sprintf("(%d)", badge.Count))
	}
	return ""
}

// severityStyle returns a render-capable style matching severity.
//
// Takes s (Severity) which is the severity to look up.
//
// Returns interface{ Render(...string) string } which is the matching
// theme style.
func (m *ContextualMenu) severityStyle(s Severity) interface{ Render(...string) string } {
	if m.theme == nil {
		return navItemStyle
	}
	return m.theme.SeverityFor(s)
}

// styleHotkey applies the theme's hotkey style to text, falling back to
// the legacy global style when no theme is configured.
//
// Takes text (string) which is the row text to style.
//
// Returns string which is the styled text.
func (m *ContextualMenu) styleHotkey(text string) string {
	if m.theme == nil {
		return navItemHotkeyStyle.Render(text)
	}
	return m.theme.TabHotkey.Render(text)
}

// styleActive applies the theme's "selected" style to text, falling
// back to the legacy global style when no theme is configured.
//
// Takes text (string) which is the row text to style.
//
// Returns string which is the styled text.
func (m *ContextualMenu) styleActive(text string) string {
	if m.theme == nil {
		return navItemActiveStyle.Render(text)
	}
	return m.theme.Selected.Render(text)
}

// styleCursor applies the theme's cursor style to text, falling back to
// the legacy global style when no theme is configured.
//
// Takes text (string) which is the row text to style.
//
// Returns string which is the styled text.
func (m *ContextualMenu) styleCursor(text string) string {
	if m.theme == nil {
		return navItemStyle.Render(text)
	}
	return m.theme.Cursor.Render(text)
}

// styleIdle applies the theme's subtle (idle) style to text, falling
// back to the legacy global style when no theme is configured.
//
// Takes text (string) which is the row text to style.
//
// Returns string which is the styled text.
func (m *ContextualMenu) styleIdle(text string) string {
	if m.theme == nil {
		return navItemStyle.Render(text)
	}
	return m.theme.Subtle.Render(text)
}

// displayHotkey converts a MenuItem.Hotkey accelerator string to its
// short menu-row form. Single-character hotkeys ("1"-"9", "0") render
// verbatim; "shift+N" collapses to "^N"; other modifiers fall back to
// the raw string so they remain searchable in the help overlay.
//
// Takes hotkey (string) which is the accelerator definition.
//
// Returns string suitable for the left-column hotkey column.
func displayHotkey(hotkey string) string {
	if hotkey == "" {
		return ""
	}
	if rest, ok := strings.CutPrefix(hotkey, "shift+"); ok {
		return "^" + rest
	}
	return hotkey
}
