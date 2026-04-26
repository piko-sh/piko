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
	"image/color"

	"charm.land/lipgloss/v2"
)

const (
	// statusIndicatorDot is the dot symbol used to show health status.
	statusIndicatorDot = "●"

	// cursorArrow is the arrow character that marks the selected item.
	cursorArrow = "▸"

	// cursorIndicator is the cursor symbol with a trailing space for list selection.
	cursorIndicator = cursorArrow + " "

	// stringNewline is the newline character used to join and trim strings.
	stringNewline = "\n"

	// stringSpace is the space character used for padding.
	stringSpace = " "

	// keyEnter is the Enter key name used in key binding descriptions.
	keyEnter = "Enter"

	// logKeyProvider is the log field key for provider names.
	logKeyProvider = "provider"
)

var (
	// colourPrimary is the main colour for focused and selected items.
	colourPrimary = lipgloss.Color("39")

	// colourAccent is the accent colour for highlights and emphasis.
	colourAccent = lipgloss.Color("214")

	// colourSuccess is the ANSI colour for successful status indicators.
	colourSuccess = lipgloss.Color("42")

	// colourWarning is the colour used for warning indicators (orange/amber).
	colourWarning = lipgloss.Color("214")

	// colourError is the colour used for error states and error text.
	colourError = lipgloss.Color("196")

	// colourInfo is the colour used for informational text and status messages.
	colourInfo = lipgloss.Color("39")

	// colourForeground is the standard text colour for normal content.
	colourForeground = lipgloss.Color("252")

	// colourForegroundDim is a muted foreground colour for secondary text.
	colourForegroundDim = lipgloss.Color("240")

	// colourBackground is the background colour for styled elements.
	colourBackground = lipgloss.Color("235")

	// colourBorder is the border colour for UI elements.
	colourBorder = lipgloss.Color("238")

	// colourBorderFocused is the border colour for focused elements.
	colourBorderFocused = lipgloss.Color("39")
)

var (
	// titleStyle defines the Lip Gloss style for the main title text.
	titleStyle = lipgloss.NewStyle().
			Foreground(colourPrimary).
			Bold(true).
			Padding(0, 1)

	// statusBarStyle defines the Lip Gloss style for the bottom status bar.
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colourForegroundDim).
			Background(colourBackground).
			Padding(0, 1)

	// panelStyle defines the Lip Gloss style for unfocused panel borders.
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colourBorder).
			Padding(0, 1)

	// panelFocusedStyle defines the Lip Gloss style for focused panel borders.
	panelFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colourBorderFocused).
				Padding(0, 1)

	// panelTitleStyle defines the Lip Gloss style for panel heading text.
	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colourPrimary).
			Bold(true).
			Padding(0, 1)

	// navItemStyle defines the Lip Gloss style for inactive navigation items.
	navItemStyle = lipgloss.NewStyle().
			Foreground(colourForegroundDim).
			Padding(0, 1)

	// navItemActiveStyle defines the Lip Gloss style for the selected navigation item.
	navItemActiveStyle = lipgloss.NewStyle().
				Foreground(colourPrimary).
				Bold(true).
				Padding(0, 1)

	// navItemHotkeyStyle defines the Lip Gloss style for hotkey characters in navigation labels.
	navItemHotkeyStyle = lipgloss.NewStyle().
				Foreground(colourAccent).
				Bold(true)

	// statusHealthyStyle defines the Lip Gloss style for healthy status indicators.
	statusHealthyStyle = lipgloss.NewStyle().
				Foreground(colourSuccess)

	// statusDegradedStyle defines the Lip Gloss style for degraded status indicators.
	statusDegradedStyle = lipgloss.NewStyle().
				Foreground(colourWarning)

	// statusUnhealthyStyle defines the Lip Gloss style for unhealthy status indicators.
	statusUnhealthyStyle = lipgloss.NewStyle().
				Foreground(colourError)

	// statusUnknownStyle defines the Lip Gloss style for unknown status indicators.
	statusUnknownStyle = lipgloss.NewStyle().
				Foreground(colourForegroundDim)

	// statusPendingStyle defines the Lip Gloss style for pending status indicators.
	statusPendingStyle = lipgloss.NewStyle().
				Foreground(colourInfo)

	// helpSeparatorStyle defines the Lip Gloss style for separators in the help bar.
	helpSeparatorStyle = lipgloss.NewStyle().
				Foreground(colourBorder)
)

// applyThemeToLegacyGlobals rebinds the package-level colour and style
// variables from the supplied theme.
//
// Existing panels that read the legacy globals directly therefore inherit
// theme switches without per-panel migration. Callers (typically
// Model.SetTheme) invoke this every time the active theme changes.
//
// Takes theme (*Theme) which is the new theme. Nil theme is a no-op.
func applyThemeToLegacyGlobals(theme *Theme) {
	if theme == nil {
		return
	}

	colourPrimary = paletteColour(theme.Palette.Primary)
	colourAccent = paletteColour(theme.Palette.Accent)
	colourSuccess = paletteColour(theme.Palette.Success)
	colourWarning = paletteColour(theme.Palette.Warning)
	colourError = paletteColour(theme.Palette.Danger)
	colourInfo = paletteColour(theme.Palette.Info)
	colourForeground = paletteColour(theme.Palette.Foreground)
	colourForegroundDim = paletteColour(theme.Palette.ForegroundDim)
	colourBackground = paletteColour(theme.Palette.SurfaceHigh)
	colourBorder = paletteColour(theme.Palette.Border)
	colourBorderFocused = paletteColour(theme.Palette.BorderFocused)

	titleStyle = theme.Title
	statusBarStyle = theme.StatusBar
	panelStyle = theme.Panel
	panelFocusedStyle = theme.PanelFocused
	panelTitleStyle = theme.PanelTitle
	navItemStyle = theme.Tab
	navItemActiveStyle = theme.TabActive
	navItemHotkeyStyle = theme.TabHotkey
	statusHealthyStyle = theme.StatusHealthy
	statusDegradedStyle = theme.StatusDegraded
	statusUnhealthyStyle = theme.StatusUnhealthy
	statusUnknownStyle = theme.StatusUnknown
	statusPendingStyle = theme.StatusPending
	helpSeparatorStyle = theme.StatusSep
}

// paletteColour returns the palette colour as a color.Color suitable for
// assignment into the legacy globals. The legacy globals are inferred to
// the return type of lipgloss.Color(), which is color.Color, so the
// signatures align.
//
// Takes c (color.Color) which is the palette colour.
//
// Returns color.Color (potentially nil -> no colour).
func paletteColour(c color.Color) color.Color {
	if c == nil {
		return lipgloss.NoColor{}
	}
	return c
}
