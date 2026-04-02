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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newLocationMapper(t *testing.T) {
	t.Run("EmptySource", func(t *testing.T) {
		mapper := newLocationMapper([]byte(""))
		assert.NotNil(t, mapper)
		assert.NotNil(t, mapper.lineStarts)
		assert.Equal(t, 1, len(mapper.lineStarts), "Should have one line start at offset 0")
		assert.Equal(t, 0, mapper.lineStarts[0])
	})

	t.Run("SingleLine", func(t *testing.T) {
		source := []byte("Hello World")
		mapper := newLocationMapper(source)
		assert.NotNil(t, mapper)
		assert.Equal(t, 1, len(mapper.lineStarts), "Single line should have one line start")
		assert.Equal(t, 0, mapper.lineStarts[0])
	})

	t.Run("MultipleLines", func(t *testing.T) {
		source := []byte("Line 1\nLine 2\nLine 3")
		mapper := newLocationMapper(source)
		assert.NotNil(t, mapper)
		assert.Equal(t, 3, len(mapper.lineStarts), "Three lines should have three line starts")
		assert.Equal(t, 0, mapper.lineStarts[0], "Line 1 starts at offset 0")
		assert.Equal(t, 7, mapper.lineStarts[1], "Line 2 starts at offset 7 (after 'Line 1\\n')")
		assert.Equal(t, 14, mapper.lineStarts[2], "Line 3 starts at offset 14")
	})

	t.Run("TrailingNewline", func(t *testing.T) {
		source := []byte("Line 1\nLine 2\n")
		mapper := newLocationMapper(source)
		assert.Equal(t, 3, len(mapper.lineStarts), "Trailing newline creates an empty third line")
		assert.Equal(t, 0, mapper.lineStarts[0])
		assert.Equal(t, 7, mapper.lineStarts[1])
		assert.Equal(t, 14, mapper.lineStarts[2], "Empty line 3 starts at offset 14")
	})
}

func Test_locationMapper_Position(t *testing.T) {
	t.Run("NilMapper", func(t *testing.T) {
		var mapper *locationMapper
		line, column := mapper.Position(0)
		assert.Equal(t, 0, line, "Nil mapper should return 0 for line")
		assert.Equal(t, 0, column, "Nil mapper should return 0 for column")
	})

	t.Run("SingleLineOffsets", func(t *testing.T) {
		source := []byte("Hello World")
		mapper := newLocationMapper(source)

		tests := []struct {
			name           string
			offset         int
			expectedLine   int
			expectedColumn int
		}{
			{
				name:           "Start of line",
				offset:         0,
				expectedLine:   1,
				expectedColumn: 1,
			},
			{
				name:           "Middle of line",
				offset:         6,
				expectedLine:   1,
				expectedColumn: 7,
			},
			{
				name:           "End of line",
				offset:         11,
				expectedLine:   1,
				expectedColumn: 12,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				line, column := mapper.Position(tt.offset)
				assert.Equal(t, tt.expectedLine, line, "Line number mismatch")
				assert.Equal(t, tt.expectedColumn, column, "Column number mismatch")
			})
		}
	})

	t.Run("MultiLineOffsets", func(t *testing.T) {

		source := []byte("Line 1\nLine 2\nLine 3")
		mapper := newLocationMapper(source)

		tests := []struct {
			name           string
			offset         int
			expectedLine   int
			expectedColumn int
		}{
			{
				name:           "Start of line 1",
				offset:         0,
				expectedLine:   1,
				expectedColumn: 1,
			},
			{
				name:           "Middle of line 1",
				offset:         3,
				expectedLine:   1,
				expectedColumn: 4,
			},
			{
				name:           "Newline after line 1",
				offset:         6,
				expectedLine:   1,
				expectedColumn: 7,
			},
			{
				name:           "Start of line 2",
				offset:         7,
				expectedLine:   2,
				expectedColumn: 1,
			},
			{
				name:           "Middle of line 2",
				offset:         10,
				expectedLine:   2,
				expectedColumn: 4,
			},
			{
				name:           "Start of line 3",
				offset:         14,
				expectedLine:   3,
				expectedColumn: 1,
			},
			{
				name:           "End of line 3",
				offset:         20,
				expectedLine:   3,
				expectedColumn: 7,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				line, column := mapper.Position(tt.offset)
				assert.Equal(t, tt.expectedLine, line, "Line number mismatch for offset %d", tt.offset)
				assert.Equal(t, tt.expectedColumn, column, "Column number mismatch for offset %d", tt.offset)
			})
		}
	})

	t.Run("MultiByteCharacters", func(t *testing.T) {

		source := []byte("Hello 世界\nNext line")
		mapper := newLocationMapper(source)

		tests := []struct {
			name           string
			offset         int
			expectedLine   int
			expectedColumn int
		}{
			{
				name:           "Before multi-byte char",
				offset:         6,
				expectedLine:   1,
				expectedColumn: 7,
			},
			{
				name:           "First byte of 世",
				offset:         6,
				expectedLine:   1,
				expectedColumn: 7,
			},
			{
				name:           "After first multi-byte char (世)",
				offset:         9,
				expectedLine:   1,
				expectedColumn: 8,
			},
			{
				name:           "After second multi-byte char (界)",
				offset:         12,
				expectedLine:   1,
				expectedColumn: 9,
			},
			{
				name:           "Start of second line",
				offset:         13,
				expectedLine:   2,
				expectedColumn: 1,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				line, column := mapper.Position(tt.offset)
				assert.Equal(t, tt.expectedLine, line, "Line number mismatch")
				assert.Equal(t, tt.expectedColumn, column, "Column should count runes, not bytes")
			})
		}
	})

	t.Run("EmptyLines", func(t *testing.T) {
		source := []byte("Line 1\n\nLine 3")
		mapper := newLocationMapper(source)

		tests := []struct {
			name           string
			offset         int
			expectedLine   int
			expectedColumn int
		}{
			{
				name:           "Start of line 1",
				offset:         0,
				expectedLine:   1,
				expectedColumn: 1,
			},
			{
				name:           "Start of empty line 2",
				offset:         7,
				expectedLine:   2,
				expectedColumn: 1,
			},
			{
				name:           "Start of line 3",
				offset:         8,
				expectedLine:   3,
				expectedColumn: 1,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				line, column := mapper.Position(tt.offset)
				assert.Equal(t, tt.expectedLine, line)
				assert.Equal(t, tt.expectedColumn, column)
			})
		}
	})
}
