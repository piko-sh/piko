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

package markdown_domain

import "bytes"

// positionMapper converts byte offsets to line/column positions in a source file.
type positionMapper interface {
	// Position converts a byte offset to line and column numbers.
	//
	// Takes offset (int) which is the byte position in the source.
	//
	// Returns line (int) which is the line number, starting from one.
	// Returns column (int) which is the column within the line, starting from one.
	Position(offset int) (line, column int)
}

// locationMapper holds a pre-calculated index of newline positions in a
// source file, allowing fast conversion of a byte offset to a line and
// column number. It implements the positionMapper interface.
type locationMapper struct {
	// source holds the original source code bytes for position calculations.
	source []byte

	// lineStarts holds byte offsets where each line begins in the source.
	// The index is the line number (0-indexed) and the value is the offset.
	lineStarts []int
}

var _ positionMapper = (*locationMapper)(nil)

// Position finds the line and column number for a given byte offset.
// Both values are 1-based.
//
// Takes offset (int) which specifies the byte position in the source.
//
// Returns line (int) which is the 1-based line number.
// Returns column (int) which is the 1-based column number in runes.
func (lm *locationMapper) Position(offset int) (line, column int) {
	if lm == nil {
		return 0, 0
	}

	lineIndex := 0
	for i := len(lm.lineStarts) - 1; i >= 0; i-- {
		if lm.lineStarts[i] <= offset {
			lineIndex = i
			break
		}
	}

	lineStartOffset := lm.lineStarts[lineIndex]

	column = len(bytes.Runes(lm.source[lineStartOffset:offset])) + 1

	return lineIndex + 1, column
}

// newLocationMapper creates a mapper by scanning the source for newlines.
//
// Takes source ([]byte) which contains the content to build line mappings for.
//
// Returns *locationMapper which converts byte offsets to line and column
// positions.
func newLocationMapper(source []byte) *locationMapper {
	lineStarts := make([]int, 0, bytes.Count(source, []byte("\n"))+1)
	lineStarts = append(lineStarts, 0)

	for i, b := range source {
		if b == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}

	return &locationMapper{
		source:     source,
		lineStarts: lineStarts,
	}
}
