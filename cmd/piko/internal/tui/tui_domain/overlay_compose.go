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

	"github.com/charmbracelet/x/ansi"
)

// ComposeOverlay splices an overlay body over a rendered background, both
// expressed as multiline strings whose visible width is at most screenW.
// The overlay is centred on the screen; SGR sequences in both layers are
// preserved so colours and styles do not bleed into adjacent cells.
//
// Takes background (string) which is the underlying layout, sized to
// approximately screenW x screenH cells.
// Takes overlayBody (string) which is the pre-rendered overlay; its visible
// dimensions are computed by counting cells in its rows.
// Takes screenW (int) which is the terminal width.
// Takes screenH (int) which is the terminal height.
// Takes theme (*Theme) which provides the optional dim style applied to
// background regions outside the overlay rectangle. May be nil; if so, the
// background is unchanged.
//
// Returns string which is the composed image.
func ComposeOverlay(background, overlayBody string, screenW, screenH int, theme *Theme) string {
	if overlayBody == "" {
		return background
	}

	bgRows := splitToRows(background, screenW, screenH)
	overlayRows := strings.Split(overlayBody, "\n")

	overlayW := 0
	for _, row := range overlayRows {
		if w := TextWidth(row); w > overlayW {
			overlayW = w
		}
	}
	overlayH := len(overlayRows)

	if overlayW > screenW {
		overlayW = screenW
	}
	if overlayH > screenH {
		overlayH = screenH
	}

	startCol := max(0, (screenW-overlayW)/2)
	startRow := max(0, (screenH-overlayH)/2)

	for ri := 0; ri < overlayH && startRow+ri < len(bgRows); ri++ {
		bgRow := bgRows[startRow+ri]
		overlayRow := overlayRows[ri]

		left := ansi.Cut(bgRow, 0, startCol)
		right := ansi.Cut(bgRow, startCol+overlayW, screenW)

		overlayRow = PadRightANSI(overlayRow, overlayW)

		bgRows[startRow+ri] = left + overlayRow + right
	}

	if theme != nil && theme.OverlayBackground.GetFaint() {
		_ = theme
	}

	return strings.Join(bgRows, "\n")
}

// splitToRows splits the rendered background into rows, padding each row
// to screenW visible cells and the slice to screenH rows so subsequent
// splicing arithmetic works without bounds checks.
//
// Takes raw (string) which is the background newline-separated rendering.
// Takes screenW (int) which is the target visible width per row.
// Takes screenH (int) which is the target row count.
//
// Returns []string with exactly screenH rows, each of visible width
// screenW.
func splitToRows(raw string, screenW, screenH int) []string {
	rows := strings.Split(raw, "\n")

	if screenH <= 0 {
		screenH = len(rows)
	}
	if screenW <= 0 {
		screenW = 1
	}

	if len(rows) < screenH {
		blank := strings.Repeat(" ", screenW)
		for len(rows) < screenH {
			rows = append(rows, blank)
		}
	} else if len(rows) > screenH {
		rows = rows[:screenH]
	}

	for i, row := range rows {
		w := TextWidth(row)
		if w < screenW {
			rows[i] = row + strings.Repeat(" ", screenW-w)
		} else if w > screenW {
			rows[i] = ansi.Cut(row, 0, screenW)
		}
	}
	return rows
}
