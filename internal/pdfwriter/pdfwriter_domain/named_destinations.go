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
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// buildNamedDestsDict writes a /Dests name tree for internal link targets
// and returns the catalogue entry string.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes pageObjNumbers ([]int) which maps page indices to their PDF
// object numbers.
//
// Returns string which is the catalogue entry, or an empty string if no
// named destinations were collected.
func (painter *PdfPainter) buildNamedDestsDict(writer *PdfDocumentWriter, pageObjNumbers []int) string {
	if len(painter.namedDests) == 0 {
		return ""
	}

	slices.SortFunc(painter.namedDests, func(a, b namedDestination) int {
		return cmp.Compare(a.name, b.name)
	})

	seen := make(map[string]bool, len(painter.namedDests))
	unique := painter.namedDests[:0]
	for _, dest := range painter.namedDests {
		if !seen[dest.name] {
			seen[dest.name] = true
			unique = append(unique, dest)
		}
	}
	painter.namedDests = unique

	var names strings.Builder
	names.WriteByte('[')
	for i, dest := range painter.namedDests {
		if i > 0 {
			names.WriteByte(' ')
		}
		pageRef := ""
		if dest.pageIndex >= 0 && dest.pageIndex < len(pageObjNumbers) {
			pageRef = FormatReference(pageObjNumbers[dest.pageIndex])
		}
		escapedName := pdfEscapeString(dest.name)
		fmt.Fprintf(&names, "(%s) [%s /XYZ 0 %s null]",
			escapedName, pageRef, FormatNumber(dest.y))
	}
	names.WriteByte(']')

	destsNumber := writer.AllocateObject()
	writer.WriteObject(destsNumber, fmt.Sprintf(
		"<< /Names %s >>", names.String()))

	return fmt.Sprintf(" /Dests %s", FormatReference(destsNumber))
}
