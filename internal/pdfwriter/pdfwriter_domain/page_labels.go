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

package pdfwriter_domain

import (
	"fmt"
	"strings"
)

// PageLabelStyle specifies the numbering style for page labels.
type PageLabelStyle string

const (
	// LabelDecimal produces decimal numbering: 1, 2, 3, ...
	LabelDecimal PageLabelStyle = "D"

	// LabelRomanUpper produces uppercase Roman numerals: I, II, III, ...
	LabelRomanUpper PageLabelStyle = "R"

	// LabelRomanLower produces lowercase Roman numerals: i, ii, iii, ...
	LabelRomanLower PageLabelStyle = "r"

	// LabelAlphaUpper produces uppercase alphabetic labels: A, B, C, ...
	LabelAlphaUpper PageLabelStyle = "A"

	// LabelAlphaLower produces lowercase alphabetic labels: a, b, c, ...
	LabelAlphaLower PageLabelStyle = "a"

	// LabelNone produces no numbering (prefix only).
	LabelNone PageLabelStyle = ""
)

// PageLabelRange defines a page label range starting at a given page index.
// All pages from PageIndex until the next range use this label configuration.
type PageLabelRange struct {
	// Style is the numbering style for this range.
	Style PageLabelStyle

	// Prefix is an optional string prepended to each page label.
	Prefix string

	// PageIndex is the zero-based page index where this range begins.
	PageIndex int

	// Start is the numeric value of the first page label in this range.
	// Defaults to 1 if zero or negative.
	Start int
}

// buildPageLabelsDict writes the /PageLabels number tree as a PDF object
// and returns the catalogue entry string.
//
// Takes ranges ([]PageLabelRange) which defines the page label ranges.
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
//
// Returns string which is the catalogue entry (e.g. " /PageLabels 5 0 R"),
// or an empty string if ranges is nil or empty.
func buildPageLabelsDict(ranges []PageLabelRange, writer *PdfDocumentWriter) string {
	if len(ranges) == 0 {
		return ""
	}

	var nums strings.Builder
	nums.WriteByte('[')
	for i, r := range ranges {
		if i > 0 {
			nums.WriteByte(' ')
		}
		fmt.Fprintf(&nums, "%d << ", r.PageIndex)
		if r.Style != "" {
			fmt.Fprintf(&nums, "/S /%s ", string(r.Style))
		}
		if r.Prefix != "" {
			fmt.Fprintf(&nums, "/P (%s) ", pdfEscapeString(r.Prefix))
		}
		if r.Start > 1 {
			fmt.Fprintf(&nums, "/St %d ", r.Start)
		}
		nums.WriteString(">>")
	}
	nums.WriteByte(']')

	labelsNumber := writer.AllocateObject()
	writer.WriteObject(labelsNumber, fmt.Sprintf("<< /Nums %s >>", nums.String()))

	return fmt.Sprintf(" /PageLabels %s", FormatReference(labelsNumber))
}
