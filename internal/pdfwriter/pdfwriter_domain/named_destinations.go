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

// nameTreePair is a single entry in a PDF name tree, pairing an already-encoded key
// string with its serialised PDF value.
type nameTreePair struct {
	// key holds the PDF string used as the lookup name, already delimited by
	// encodePdfTextString.
	key string

	// value holds the serialised PDF value associated with the key.
	value string
}

// buildNamedDestsDict writes a /Dests name tree for internal link targets and returns the
// catalogue entry string.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes pageObjNumbers ([]int) which maps page indices to their PDF object numbers.
//
// Returns string which is the catalogue entry, or an empty string if no named
// destinations were collected.
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

	pairs := make([]nameTreePair, 0, len(painter.namedDests))
	for _, dest := range painter.namedDests {
		pageRef := ""
		if dest.pageIndex >= 0 && dest.pageIndex < len(pageObjNumbers) {
			pageRef = FormatReference(pageObjNumbers[dest.pageIndex])
		}
		value := fmt.Sprintf("[%s /XYZ 0 %s null]", pageRef, FormatNumber(dest.y))
		pairs = append(pairs, nameTreePair{key: encodePdfTextString(dest.name), value: value})
	}

	destsNumber := buildNameTreeDict(writer, pairs)
	return fmt.Sprintf(" /Dests %s", FormatReference(destsNumber))
}

// buildNameTreeDict writes a single-node PDF name tree object from the given pairs and
// returns its object number.
//
// The caller must sort pairs by key, as the PDF name-tree specification requires. It is
// shared by the named-destinations and embedded-files trees.
//
// Takes writer (*PdfDocumentWriter) which receives the object.
// Takes pairs ([]nameTreePair) which holds the sorted name/value entries.
//
// Returns int which is the allocated object number.
func buildNameTreeDict(writer *PdfDocumentWriter, pairs []nameTreePair) int {
	var names strings.Builder
	names.WriteByte('[')
	for i, pair := range pairs {
		if i > 0 {
			names.WriteByte(' ')
		}
		fmt.Fprintf(&names, "%s %s", pair.key, pair.value)
	}
	names.WriteByte(']')

	number := writer.AllocateObject()
	writer.WriteObject(number, fmt.Sprintf("<< /Names %s >>", names.String()))
	return number
}
