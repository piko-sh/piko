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
	"github.com/charmbracelet/x/ansi"
)

// TextWidth returns the visible-cell width of s, ignoring ANSI escape
// sequences and accounting for wide characters such as East-Asian glyphs and
// emoji. Callers must use this rather than len() when measuring a string for
// terminal display.
//
// Takes s (string) which is the text to measure.
//
// Returns int which is the number of terminal cells s would occupy when
// rendered.
func TextWidth(s string) int {
	return lipgloss.Width(s)
}

// PadRightANSI returns s padded on the right with spaces until its visible
// width equals width.
//
// If s is wider than width, it is truncated to fit without an ellipsis. ANSI
// escape sequences embedded in s are preserved.
//
// Takes s (string) which is the input string.
// Takes width (int) which is the target visible width in terminal cells.
//
// Returns string which is the padded or truncated result.
func PadRightANSI(s string, width int) string {
	if width <= 0 {
		return ""
	}
	visible := TextWidth(s)
	if visible >= width {
		return ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-visible)
}

// TruncateANSI shortens s so its visible width is at most maxWidth.
//
// Appends an ellipsis when the input was cut. ANSI escape sequences are
// preserved and wide characters are accounted for. When maxWidth is small
// enough that the ellipsis would not fit, the input is cut to fit without one.
//
// Takes s (string) which is the input string.
// Takes maxWidth (int) which is the maximum permitted visible width in cells.
//
// Returns string which is the input shortened to fit within maxWidth, with
// an ellipsis appended when content was lost.
func TruncateANSI(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if TextWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= ellipsisLength {
		return ansi.Truncate(s, maxWidth, "")
	}
	return ansi.Truncate(s, maxWidth, ellipsis)
}
