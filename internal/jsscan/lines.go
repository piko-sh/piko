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

package jsscan

import (
	"slices"
	"unicode/utf8"
)

// LineIndex turns byte offsets in a script into line and column positions.
type LineIndex struct {
	// source is the script.
	source string

	// lineStarts holds the byte offset where each line begins.
	lineStarts []int
}

// NewLineIndex indexes the lines of a script.
//
// Takes source (string) which is the script.
//
// Returns *LineIndex which places offsets in the script.
func NewLineIndex(source string) *LineIndex {
	lineStarts := []int{0}
	for index := range len(source) {
		if source[index] == '\n' {
			lineStarts = append(lineStarts, index+1)
		}
	}
	return &LineIndex{source: source, lineStarts: lineStarts}
}

// Position returns the line and column of a byte offset, both starting at 1. Columns
// count characters, not bytes.
//
// An offset outside the script is moved to its nearest end.
//
// Takes offset (int) which is the byte offset in the script.
//
// Returns line (int) which is the 1-based line.
// Returns column (int) which is the 1-based column.
func (index *LineIndex) Position(offset int) (line, column int) {
	offset = min(max(offset, 0), len(index.source))
	lineIndex, found := slices.BinarySearch(index.lineStarts, offset)
	if !found {
		lineIndex--
	}
	return lineIndex + 1, utf8.RuneCountInString(index.source[index.lineStarts[lineIndex]:offset]) + 1
}
