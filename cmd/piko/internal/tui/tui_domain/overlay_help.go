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

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// HelpOverlay is a centred modal listing the global key bindings and any
// bindings advertised by the active panel.
type HelpOverlay struct {
	// theme drives the overlay's frame and text colours.
	theme *Theme

	// globalKeys are the model-wide bindings shown in the global section.
	globalKeys []KeyBinding

	// panelTitle names the active panel; empty suppresses the panel section.
	panelTitle string

	// panelKeys are bindings advertised by the active panel.
	panelKeys []KeyBinding

	// commands are command-palette entries listed in the overlay.
	commands []Command

	// dismissed is set true once the user closes the overlay.
	dismissed bool
}

// NewHelpOverlay constructs a help overlay for the supplied bindings.
//
// Takes theme (*Theme) which sets the overlay's frame and text styles.
// Takes globalKeys ([]KeyBinding) which are the global key bindings.
// Takes panelTitle (string) which names the active panel; empty string
// suppresses the panel-specific section.
// Takes panelKeys ([]KeyBinding) which are the panel's bindings; empty
// slice suppresses the panel-specific section.
// Takes commands ([]Command) which are the available command-palette
// entries; nil or empty suppresses the section.
//
// Returns *HelpOverlay ready to push onto the OverlayManager stack.
func NewHelpOverlay(theme *Theme, globalKeys []KeyBinding, panelTitle string, panelKeys []KeyBinding, commands []Command) *HelpOverlay {
	return &HelpOverlay{
		theme:      theme,
		globalKeys: globalKeys,
		panelTitle: panelTitle,
		panelKeys:  panelKeys,
		commands:   commands,
	}
}

// ID returns the overlay identifier.
//
// Returns string equal to "help".
func (*HelpOverlay) ID() string { return "help" }

// MinSize returns the minimum overlay width and height.
//
// Returns width (int) which is the minimum overlay width.
// Returns height (int) which is the minimum overlay height.
func (*HelpOverlay) MinSize() (width, height int) {
	return HelpOverlayMinWidth, HelpOverlayMinHeight
}

// KeyMap describes the keys the help overlay accepts.
//
// Returns []KeyBinding listing the dismiss keys.
func (*HelpOverlay) KeyMap() []KeyBinding {
	return []KeyBinding{
		{Key: "?", Description: "Close help"},
		{Key: "Esc", Description: "Close help"},
		{Key: "q", Description: "Close help"},
	}
}

// Dismiss reports whether the overlay has been closed by the user.
//
// Returns bool which is true after Esc/?/q.
func (h *HelpOverlay) Dismiss() bool { return h.dismissed }

// Update consumes input intended for the help overlay. ?, Esc, and q each
// dismiss the overlay; other keys are ignored so they do not propagate to
// the underlying panel.
//
// Takes msg (tea.Msg) which is the message to handle.
//
// Returns Overlay which is the (possibly updated) overlay.
// Returns tea.Cmd which is always nil; the overlay produces no commands.
func (h *HelpOverlay) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "?", "esc", "q":
			h.dismissed = true
		}
	}
	return h, nil
}

// Render produces the help overlay body sized to the supplied dimensions.
//
// Takes width (int) which is the overlay width including border.
// Takes height (int) which is the overlay height including border.
//
// Returns string which is the framed help content.
func (h *HelpOverlay) Render(width, height int) string {
	innerWidth := max(1, width-PanelChromeWidth)

	rows := []string{
		h.titleRow("Piko TUI Help", innerWidth),
		"",
		h.sectionHeader("Global", innerWidth),
	}
	rows = append(rows, h.bindingRows(h.globalKeys, innerWidth)...)

	if h.panelTitle != "" && len(h.panelKeys) > 0 {
		rows = append(rows, "", h.sectionHeader("Panel: "+h.panelTitle, innerWidth))
		rows = append(rows, h.bindingRows(h.panelKeys, innerWidth)...)
	}

	if len(h.commands) > 0 {
		rows = append(rows, "", h.sectionHeader("Command palette ( : )", innerWidth))
		rows = append(rows, h.commandRows(h.commands, innerWidth)...)
	}

	rows = append(rows, "", h.dim("Press ? or Esc to close.", innerWidth))

	body := strings.Join(rows, "\n")

	return RenderPaneFrame(PaneFrameOpts{
		Theme:   h.theme,
		Title:   "",
		Body:    body,
		Width:   width,
		Height:  height,
		Focused: true,
	})
}

// titleRow renders the centred header row.
//
// Takes label (string) which is the title text.
// Takes width (int) which is the available cell width inside the frame.
//
// Returns string which is the title row.
func (h *HelpOverlay) titleRow(label string, width int) string {
	style := paneFrameStyle(PaneFrameOpts{Theme: h.theme, Focused: true})
	_ = style
	titleStyle := paneFrameTitleStyle(h.theme)
	pad := max(0, (width-TextWidth(label))/2)
	return strings.Repeat(" ", pad) + titleStyle.Render(label)
}

// sectionHeader renders a section heading row.
//
// Takes label (string) which is the heading text.
// Takes width (int) which is the available width.
//
// Returns string which is the styled heading.
func (h *HelpOverlay) sectionHeader(label string, width int) string {
	headingStyle := paneFrameTitleStyle(h.theme)
	heading := headingStyle.Render(label)
	return PadRightANSI(heading, width)
}

// bindingRows renders one row per binding with a key column and a
// description column.
//
// Takes bindings ([]KeyBinding) which is the list to render.
// Takes width (int) which is the available width.
//
// Returns []string with one entry per binding.
func (h *HelpOverlay) bindingRows(bindings []KeyBinding, width int) []string {
	keyWidth := 0
	for _, b := range bindings {
		if w := TextWidth(b.Key); w > keyWidth {
			keyWidth = w
		}
	}
	keyWidth += 2

	rows := make([]string, 0, len(bindings))
	for _, b := range bindings {
		key := h.styledKey(b.Key, keyWidth)
		desc := h.styledDesc(b.Description)
		row := "  " + key + " " + desc
		rows = append(rows, PadRightANSI(row, width))
	}
	return rows
}

// commandRows renders one row per palette command with a `:name` column
// and a description column. Aliases are appended to the name column.
//
// Takes commands ([]Command) which is the list to render.
// Takes width (int) which is the available width.
//
// Returns []string with one entry per command.
func (h *HelpOverlay) commandRows(commands []Command, width int) []string {
	labels := make([]string, len(commands))
	keyWidth := 0
	for i, c := range commands {
		label := ":" + c.Name
		if len(c.Aliases) > 0 {
			label += " (" + strings.Join(c.Aliases, ", ") + ")"
		}
		labels[i] = label
		if w := TextWidth(label); w > keyWidth {
			keyWidth = w
		}
	}
	keyWidth += 2

	rows := make([]string, 0, len(commands))
	for i, c := range commands {
		key := h.styledKey(labels[i], keyWidth)
		desc := h.styledDesc(c.Description)
		row := "  " + key + " " + desc
		rows = append(rows, PadRightANSI(row, width))
	}
	return rows
}

// styledKey renders the key column with theme styling.
//
// Takes key (string) which is the key glyph.
// Takes width (int) which is the column width.
//
// Returns string which is the styled, padded key column.
func (h *HelpOverlay) styledKey(key string, width int) string {
	if h.theme != nil {
		return h.theme.StatusKey.Render(PadRightANSI(key, width))
	}
	return navItemHotkeyStyle.Render(PadRightANSI(key, width))
}

// styledDesc renders the description column with theme styling.
//
// Takes desc (string) which is the description text.
//
// Returns string which is the styled description.
func (h *HelpOverlay) styledDesc(desc string) string {
	if h.theme != nil {
		return h.theme.StatusDesc.Render(desc)
	}
	return navItemStyle.Render(desc)
}

// dim renders a footer hint in the dimmed theme style.
//
// Takes text (string) which is the hint.
// Takes width (int) which is the available width.
//
// Returns string which is the styled, padded hint.
func (h *HelpOverlay) dim(text string, width int) string {
	if h.theme != nil {
		return PadRightANSI(h.theme.Dim.Render(text), width)
	}
	return PadRightANSI(text, width)
}

// paneFrameTitleStyle returns the theme's panel-title style, falling back
// to the legacy global when no theme is supplied.
//
// Takes theme (*Theme) which is the active theme; may be nil.
//
// Returns lipgloss.Style used for the help heading.
func paneFrameTitleStyle(theme *Theme) lipgloss.Style {
	if theme != nil {
		return theme.PanelTitle
	}
	return panelTitleStyle
}
