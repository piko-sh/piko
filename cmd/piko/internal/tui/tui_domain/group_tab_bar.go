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

// GroupTabBar renders the top tab strip showing the four group tabs.
//
// Invisible groups are filtered out, so users running without a watchdog
// provider see three tabs instead of four.
type GroupTabBar struct {
	// theme drives bracket and hotkey colouring; never nil.
	theme *Theme
}

// NewGroupTabBar constructs a tab bar bound to theme.
//
// Takes theme (*Theme) which provides the active style palette.
//
// Returns *GroupTabBar ready to call Render on.
func NewGroupTabBar(theme *Theme) *GroupTabBar {
	return &GroupTabBar{theme: theme}
}

// SetTheme replaces the active theme.
//
// Takes theme (*Theme) which is the new theme; never nil.
func (b *GroupTabBar) SetTheme(theme *Theme) { b.theme = theme }

// Render produces the tab bar row at the supplied width.
//
// Groups whose Visible() reports false are filtered before rendering. The
// active group is wrapped in brackets; inactive groups are dimmed.
//
// Takes groups ([]PanelGroup) which is the ordered group list.
// Takes activeID (GroupID) which identifies the current group.
// Takes width (int) which is the terminal width; the rendered row is
// padded to width.
//
// Returns string with one row of styled tab text.
func (b *GroupTabBar) Render(groups []PanelGroup, activeID GroupID, width int) string {
	visible := make([]PanelGroup, 0, len(groups))
	for _, g := range groups {
		if g != nil && g.Visible() {
			visible = append(visible, g)
		}
	}
	if len(visible) == 0 {
		return strings.Repeat(" ", width)
	}

	tabs := make([]string, 0, len(visible))
	for _, g := range visible {
		hotkey := fmt.Sprintf("F%c", g.Hotkey())
		title := g.Title()

		var tab string
		if g.ID() == activeID {
			tab = b.styleActive("[") + b.styleHotkey(hotkey) + b.styleActive(" "+title) + b.styleActive("]")
		} else {
			tab = b.styleInactive(fmt.Sprintf(" %s %s ", b.styleHotkey(hotkey), title))
		}
		tabs = append(tabs, tab)
	}

	separator := b.styleSeparator("│")
	row := strings.Join(tabs, separator)
	return PadRightANSI(row, width)
}

// styleActive renders text in the active-tab style.
//
// Takes text (string) which is the content to style.
//
// Returns string which is the styled active-tab text.
func (b *GroupTabBar) styleActive(text string) string {
	if b.theme == nil {
		return navItemActiveStyle.Render(text)
	}
	return b.theme.TabActive.Render(text)
}

// styleInactive renders text in the inactive-tab style.
//
// Takes text (string) which is the content to style.
//
// Returns string which is the styled inactive-tab text.
func (b *GroupTabBar) styleInactive(text string) string {
	if b.theme == nil {
		return navItemStyle.Render(text)
	}
	return b.theme.Tab.Render(text)
}

// styleHotkey renders the hotkey rune.
//
// Takes text (string) which is the hotkey label to style.
//
// Returns string which is the styled hotkey text.
func (b *GroupTabBar) styleHotkey(text string) string {
	if b.theme == nil {
		return navItemHotkeyStyle.Render(text)
	}
	return b.theme.TabHotkey.Render(text)
}

// styleSeparator renders the column separator between tabs.
//
// Takes text (string) which is the separator glyph to style.
//
// Returns string which is the styled separator.
func (b *GroupTabBar) styleSeparator(text string) string {
	if b.theme == nil {
		return helpSeparatorStyle.Render(text)
	}
	return b.theme.StatusSep.Render(text)
}
