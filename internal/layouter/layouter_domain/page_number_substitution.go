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

package layouter_domain

import (
	"fmt"
	"strings"
)

// SubstitutePageNumbers walks the layout box tree and replaces {page} and
// {pages} placeholders in text nodes. After substitution, glyphs are
// re-shaped using the provided font metrics so that the painted text
// matches the new content.
//
// This should be called after pagination (which assigns PageIndex to each
// box) but before painting.
//
// Takes root (*LayoutBox) which is the root of the box tree.
// Takes totalPages (int) which is the total number of pages in the document.
// Takes fontMetrics (FontMetricsPort) which is used to re-shape glyphs
// after text substitution.
func SubstitutePageNumbers(root *LayoutBox, totalPages int, fontMetrics FontMetricsPort) {
	substitutePageNumbersRecursive(root, totalPages, fontMetrics)
}

// substitutePageNumbersRecursive recursively walks the box tree and
// replaces page number placeholders in text nodes.
//
// Takes box (*LayoutBox) which is the current box to process.
// Takes totalPages (int) which is the total number of pages.
// Takes fontMetrics (FontMetricsPort) which is used to re-shape glyphs.
func substitutePageNumbersRecursive(box *LayoutBox, totalPages int, fontMetrics FontMetricsPort) {
	if box.Text != "" && (strings.Contains(box.Text, "{page}") || strings.Contains(box.Text, "{pages}")) {
		pageNumber := fmt.Sprintf("%d", box.PageIndex+1)
		totalPagesStr := fmt.Sprintf("%d", totalPages)
		box.Text = strings.ReplaceAll(box.Text, "{page}", pageNumber)
		box.Text = strings.ReplaceAll(box.Text, "{pages}", totalPagesStr)

		font := FontDescriptor{
			Family: box.Style.FontFamily,
			Weight: box.Style.FontWeight,
			Style:  box.Style.FontStyle,
		}
		box.Glyphs = fontMetrics.ShapeText(font, box.Style.FontSize, box.Text, box.Style.Direction)
	}

	for _, child := range box.Children {
		substitutePageNumbersRecursive(child, totalPages, fontMetrics)
	}
}
