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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLineIndex_Position(t *testing.T) {
	t.Parallel()

	source := "ab\nçd\n\nx"
	testCases := []struct {
		name       string
		offset     int
		wantLine   int
		wantColumn int
	}{
		{name: "start of script", offset: 0, wantLine: 1, wantColumn: 1},
		{name: "end of first line", offset: 2, wantLine: 1, wantColumn: 3},
		{name: "start of second line", offset: 3, wantLine: 2, wantColumn: 1},
		{name: "after a two byte character", offset: 5, wantLine: 2, wantColumn: 2},
		{name: "empty line", offset: 7, wantLine: 3, wantColumn: 1},
		{name: "last line", offset: 8, wantLine: 4, wantColumn: 1},
		{name: "offset before the script", offset: -4, wantLine: 1, wantColumn: 1},
		{name: "offset past the script", offset: 99, wantLine: 4, wantColumn: 2},
	}
	index := NewLineIndex(source)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			line, column := index.Position(testCase.offset)
			require.Equal(t, testCase.wantLine, line)
			require.Equal(t, testCase.wantColumn, column)
		})
	}
}
