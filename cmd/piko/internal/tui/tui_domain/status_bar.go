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

// StatusBarSegments holds the three segments of the modern status bar.
// The status bar renders Left and Right at the screen edges and centres
// Middle (typically a transient toast) in the remaining space.
type StatusBarSegments struct {
	// Left is the keymap-hint segment shown at the left edge.
	Left string

	// Middle is the transient-message segment shown in the centre.
	Middle string

	// Right is the provider/clock segment shown at the right edge.
	Right string
}

// StatusBarRenderer composes the three segments into a single bar sized to
// the screen width. Truncation is applied to the left segment first when
// the bar runs out of space.
type StatusBarRenderer struct {
	// theme drives the bar's colour palette.
	theme *Theme
}

// NewStatusBarRenderer returns a renderer bound to the supplied theme.
//
// Takes theme (*Theme) which supplies styling. May be nil.
//
// Returns *StatusBarRenderer ready to call Render on.
func NewStatusBarRenderer(theme *Theme) *StatusBarRenderer {
	return &StatusBarRenderer{theme: theme}
}

// SetTheme replaces the renderer's theme. Used when the user switches
// themes via the command bar.
//
// Takes theme (*Theme) which becomes the new theme.
func (r *StatusBarRenderer) SetTheme(theme *Theme) {
	r.theme = theme
}

// Render returns the status bar string sized to width. The themed style
// is applied without horizontal padding so the output cell-width matches
// width exactly; callers wanting padding should pass a smaller width.
//
// Takes segments (StatusBarSegments) which are the three segments to
// arrange.
// Takes width (int) which is the terminal width.
//
// Returns string which is the styled status bar of exactly width cells.
func (r *StatusBarRenderer) Render(segments StatusBarSegments, width int) string {
	if width <= 0 {
		return ""
	}

	left := segments.Left
	mid := segments.Middle
	right := segments.Right

	leftWidth := TextWidth(left)
	midWidth := TextWidth(mid)
	rightWidth := TextWidth(right)

	body := r.composeBody(left, mid, right, leftWidth, midWidth, rightWidth, width)
	return r.applyStyle(body)
}

// composeBody picks the segment arrangement that fits in width.
//
// Takes left, mid, right (string) which are the three segments.
// Takes leftWidth, midWidth, rightWidth (int) which are pre-computed widths.
// Takes width (int) which is the total target width.
//
// Returns string of exactly width cells.
func (*StatusBarRenderer) composeBody(left, mid, right string, leftWidth, midWidth, rightWidth, width int) string {
	if leftWidth+midWidth+rightWidth+2 <= width {
		leftGap := max(1, (width-leftWidth-midWidth-rightWidth)/2)
		body := left + strings.Repeat(SingleSpace, leftGap) + mid
		bodyWidth := leftWidth + leftGap + midWidth
		rightGap := max(1, width-bodyWidth-rightWidth)
		body += strings.Repeat(SingleSpace, rightGap) + right
		return PadRightANSI(body, width)
	}

	if leftWidth+rightWidth+1 <= width {
		gap := width - leftWidth - rightWidth
		body := left + strings.Repeat(SingleSpace, gap) + right
		return PadRightANSI(body, width)
	}

	if rightWidth+1 < width {
		leftBudget := width - rightWidth - 1
		body := TruncateANSI(left, leftBudget) + SingleSpace + right
		return PadRightANSI(body, width)
	}

	return PadRightANSI(TruncateANSI(right, width), width)
}

// applyStyle wraps content in the theme's StatusBar style without altering
// the visible cell width. The theme's padding is unset so the output
// fills exactly the requested width.
//
// Takes content (string) which is the pre-padded bar content.
//
// Returns string which is the styled bar.
func (r *StatusBarRenderer) applyStyle(content string) string {
	if r.theme != nil {
		return r.theme.StatusBar.UnsetPadding().Render(content)
	}
	return statusBarStyle.UnsetPadding().Render(content)
}
