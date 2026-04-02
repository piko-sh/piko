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

import "charm.land/lipgloss/v2"

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
	// colorPrimary is the main colour for focused and selected items.
	colorPrimary = lipgloss.Color("39")

	// colorAccent is the accent colour for highlights and emphasis.
	colorAccent = lipgloss.Color("214")

	// colorSuccess is the ANSI colour for successful status indicators.
	colorSuccess = lipgloss.Color("42")

	// colorWarning is the colour used for warning indicators (orange/amber).
	colorWarning = lipgloss.Color("214")

	// colorError is the colour used for error states and error text.
	colorError = lipgloss.Color("196")

	// colorInfo is the colour used for informational text and status messages.
	colorInfo = lipgloss.Color("39")

	// colorForeground is the standard text colour for normal content.
	colorForeground = lipgloss.Color("252")

	// colorForegroundDim is a muted foreground colour for secondary text.
	colorForegroundDim = lipgloss.Color("240")

	// colorBackground is the background colour for styled elements.
	colorBackground = lipgloss.Color("235")

	// colorBorder is the border colour for UI elements.
	colorBorder = lipgloss.Color("238")

	// colorBorderFocused is the border colour for focused elements.
	colorBorderFocused = lipgloss.Color("39")
)

var (
	titleStyle = lipgloss.NewStyle().

			Foreground(colorPrimary).

			Bold(true).

			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().

			Foreground(colorForegroundDim).

			Background(colorBackground).

			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().

			Border(lipgloss.RoundedBorder()).

			BorderForeground(colorBorder).

			Padding(0, 1)

	panelFocusedStyle = lipgloss.NewStyle().

				Border(lipgloss.RoundedBorder()).

				BorderForeground(colorBorderFocused).

				Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().

			Foreground(colorPrimary).

			Bold(true).

			Padding(0, 1)

	navItemStyle = lipgloss.NewStyle().

			Foreground(colorForegroundDim).

			Padding(0, 1)

	navItemActiveStyle = lipgloss.NewStyle().

				Foreground(colorPrimary).

				Bold(true).

				Padding(0, 1)

	navItemHotkeyStyle = lipgloss.NewStyle().

				Foreground(colorAccent).

				Bold(true)

	statusHealthyStyle = lipgloss.NewStyle().

				Foreground(colorSuccess)

	statusDegradedStyle = lipgloss.NewStyle().

				Foreground(colorWarning)

	statusUnhealthyStyle = lipgloss.NewStyle().

				Foreground(colorError)

	statusUnknownStyle = lipgloss.NewStyle().

				Foreground(colorForegroundDim)

	statusPendingStyle = lipgloss.NewStyle().

				Foreground(colorInfo)

	helpSeparatorStyle = lipgloss.NewStyle().

				Foreground(colorBorder)
)
