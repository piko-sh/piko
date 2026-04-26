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
)

// breadcrumbSeparator is the chevron-style separator between breadcrumb
// segments. Drawn from the Unicode "right-pointing single guillemet" so it
// renders without a Nerd Font.
const breadcrumbSeparator = " › "

// watchIndicatorActive marks an active live data feed.
const watchIndicatorActive = "◉"

// watchIndicatorIdle marks a stalled or unconnected feed.
const watchIndicatorIdle = "◌"

// Breadcrumb describes the chrome line above the layout area showing the
// user's current location and global context.
type Breadcrumb struct {
	// Title is the leading label rendered on the left side.
	Title string

	// Endpoint is the resolved server endpoint shown on the right side.
	Endpoint string

	// Scope is the active scope/cluster identifier shown on the right.
	Scope string

	// PanelChain is the ordered list of panel segments forming the path.
	PanelChain []string

	// Watch indicates whether a live data feed is active.
	Watch bool
}

// Render produces the breadcrumb line sized to width. Long inputs are
// truncated; SGR sequences are preserved.
//
// Takes theme (*Theme) which provides the breadcrumb styles.
// Takes width (int) which is the terminal width.
//
// Returns string which is the rendered breadcrumb row.
func (b *Breadcrumb) Render(theme *Theme, width int) string {
	if width <= 0 {
		return ""
	}

	leftPath := b.renderLeft(theme)
	right := b.renderRight(theme)

	leftWidth := TextWidth(leftPath)
	rightWidth := TextWidth(right)
	available := width - leftWidth - rightWidth - 2

	gap := " "
	if available > 0 {
		gap = strings.Repeat(" ", available)
	} else if available < 0 {
		leftPath = TruncateANSI(leftPath, max(1, width-rightWidth-2))
		gap = " "
	}

	return PadRightANSI(" "+leftPath+gap+right+" ", width)
}

// renderLeft assembles the path side of the breadcrumb.
//
// Takes theme (*Theme) which provides the styles. May be nil.
//
// Returns string which is the styled left half.
func (b *Breadcrumb) renderLeft(theme *Theme) string {
	titleStyle := paneFrameTitleStyle(theme)
	dimStyle := dimStyleFor(theme)

	parts := []string{}
	if b.Title != "" {
		parts = append(parts, titleStyle.Render(b.Title))
	}
	for _, segment := range b.PanelChain {
		if segment == "" {
			continue
		}
		parts = append(parts, dimStyle.Render(segment))
	}

	if len(parts) == 0 {
		return ""
	}

	separator := dimStyle.Render(breadcrumbSeparator)
	return strings.Join(parts, separator)
}

// renderRight assembles the endpoint+watch side of the breadcrumb.
//
// Takes theme (*Theme) which provides the styles. May be nil.
//
// Returns string which is the styled right half.
func (b *Breadcrumb) renderRight(theme *Theme) string {
	dimStyle := dimStyleFor(theme)
	healthyStyle := healthyStyleFor(theme)
	subtleStyle := subtleStyleFor(theme)

	parts := []string{}
	if b.Scope != "" {
		parts = append(parts, subtleStyle.Render(b.Scope))
	}
	if b.Endpoint != "" {
		parts = append(parts, dimStyle.Render(b.Endpoint))
	}
	if b.Watch {
		parts = append(parts, healthyStyle.Render(watchIndicatorActive))
	} else {
		parts = append(parts, dimStyle.Render(watchIndicatorIdle))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, SingleSpace)
}

// dimStyleFor returns the theme's dim style with a fallback for nil
// themes.
//
// Takes theme (*Theme) which is the active theme; may be nil.
//
// Returns the lipgloss style used for dim/secondary text.
func dimStyleFor(theme *Theme) interface{ Render(...string) string } {
	if theme != nil {
		return theme.Dim
	}
	return navItemStyle
}

// subtleStyleFor returns the theme's subtle style with a fallback.
//
// Takes theme (*Theme) which is the active theme; may be nil.
//
// Returns the lipgloss style used for subtle text.
func subtleStyleFor(theme *Theme) interface{ Render(...string) string } {
	if theme != nil {
		return theme.Subtle
	}
	return statusBarStyle
}

// healthyStyleFor returns the theme's healthy status style with a fallback.
//
// Takes theme (*Theme) which is the active theme; may be nil.
//
// Returns the lipgloss style used for healthy/positive indicators.
func healthyStyleFor(theme *Theme) interface{ Render(...string) string } {
	if theme != nil {
		return theme.StatusHealthy
	}
	return statusHealthyStyle
}
