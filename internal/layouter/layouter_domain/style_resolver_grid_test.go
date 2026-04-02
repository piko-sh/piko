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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGridTrackList(t *testing.T) {
	ctx := defaultResolutionContext()

	testCases := []struct {
		name     string
		input    string
		expected []GridTrack
	}{
		{
			name:     "none returns nil",
			input:    "none",
			expected: nil,
		},
		{
			name:  "single pixel value is converted to points",
			input: "100px",
			expected: []GridTrack{
				{Value: 75, Unit: GridTrackPoints},
			},
		},
		{
			name:  "single fractional unit",
			input: "1fr",
			expected: []GridTrack{
				{Value: 1, Unit: GridTrackFr},
			},
		},
		{
			name:  "auto keyword",
			input: "auto",
			expected: []GridTrack{
				{Unit: GridTrackAuto},
			},
		},
		{
			name:  "min-content keyword",
			input: "min-content",
			expected: []GridTrack{
				{Unit: GridTrackMinContent},
			},
		},
		{
			name:  "max-content keyword",
			input: "max-content",
			expected: []GridTrack{
				{Unit: GridTrackMaxContent},
			},
		},
		{
			name:  "multiple tracks separated by spaces",
			input: "100px 1fr auto",
			expected: []GridTrack{
				{Value: 75, Unit: GridTrackPoints},
				{Value: 1, Unit: GridTrackFr},
				{Unit: GridTrackAuto},
			},
		},
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseGridTrackList(tc.input, ctx)
			assert.Equal(t, tc.expected, result.tracks)
			assert.Nil(t, result.autoRepeat)
		})
	}
}

func TestParseGridTrackList_Repeat(t *testing.T) {
	ctx := defaultResolutionContext()

	testCases := []struct {
		name     string
		input    string
		expected []GridTrack
	}{
		{
			name:  "repeat with single track",
			input: "repeat(3, 1fr)",
			expected: []GridTrack{
				{Value: 1, Unit: GridTrackFr},
				{Value: 1, Unit: GridTrackFr},
				{Value: 1, Unit: GridTrackFr},
			},
		},
		{
			name:  "repeat with multiple tracks",
			input: "repeat(2, 100px 1fr)",
			expected: []GridTrack{
				{Value: 75, Unit: GridTrackPoints},
				{Value: 1, Unit: GridTrackFr},
				{Value: 75, Unit: GridTrackPoints},
				{Value: 1, Unit: GridTrackFr},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseGridTrackList(tc.input, ctx)
			assert.Equal(t, tc.expected, result.tracks)
			assert.Nil(t, result.autoRepeat)
		})
	}
}

func TestParseGridTrackList_AutoRepeat(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("auto-fill stores deferred repeat", func(t *testing.T) {
		result := parseGridTrackList("repeat(auto-fill, 200px)", ctx)
		assert.Empty(t, result.tracks)
		assert.NotNil(t, result.autoRepeat)
		assert.Equal(t, GridAutoRepeatFill, result.autoRepeat.Type)
		assert.Equal(t, []GridTrack{{Value: 150, Unit: GridTrackPoints}}, result.autoRepeat.Pattern)
		assert.Equal(t, 0, result.autoRepeat.InsertIndex)
	})

	t.Run("auto-fit stores deferred repeat", func(t *testing.T) {
		result := parseGridTrackList("repeat(auto-fit, 200px)", ctx)
		assert.Empty(t, result.tracks)
		assert.NotNil(t, result.autoRepeat)
		assert.Equal(t, GridAutoRepeatFit, result.autoRepeat.Type)
		assert.Equal(t, []GridTrack{{Value: 150, Unit: GridTrackPoints}}, result.autoRepeat.Pattern)
	})

	t.Run("fixed tracks before and after auto-fill", func(t *testing.T) {
		result := parseGridTrackList("100px repeat(auto-fill, 200px) 100px", ctx)
		assert.Equal(t, []GridTrack{
			{Value: 75, Unit: GridTrackPoints},
			{Value: 75, Unit: GridTrackPoints},
		}, result.tracks)
		assert.NotNil(t, result.autoRepeat)
		assert.Equal(t, GridAutoRepeatFill, result.autoRepeat.Type)
		assert.Equal(t, 1, result.autoRepeat.InsertIndex)
	})

	t.Run("auto-fill with multiple pattern tracks", func(t *testing.T) {
		result := parseGridTrackList("repeat(auto-fill, 100px 200px)", ctx)
		assert.Empty(t, result.tracks)
		assert.NotNil(t, result.autoRepeat)
		assert.Equal(t, []GridTrack{
			{Value: 75, Unit: GridTrackPoints},
			{Value: 150, Unit: GridTrackPoints},
		}, result.autoRepeat.Pattern)
	})
}

func TestParseGridTrackToken(t *testing.T) {
	ctx := defaultResolutionContext()

	testCases := []struct {
		name     string
		input    string
		expected GridTrack
	}{
		{
			name:     "auto keyword",
			input:    "auto",
			expected: GridTrack{Unit: GridTrackAuto},
		},
		{
			name:     "min-content keyword",
			input:    "min-content",
			expected: GridTrack{Unit: GridTrackMinContent},
		},
		{
			name:     "max-content keyword",
			input:    "max-content",
			expected: GridTrack{Unit: GridTrackMaxContent},
		},
		{
			name:     "integer fractional unit",
			input:    "1fr",
			expected: GridTrack{Value: 1, Unit: GridTrackFr},
		},
		{
			name:     "decimal fractional unit",
			input:    "2.5fr",
			expected: GridTrack{Value: 2.5, Unit: GridTrackFr},
		},
		{
			name:     "percentage value",
			input:    "50%",
			expected: GridTrack{Value: 50, Unit: GridTrackPercentage},
		},
		{
			name:     "pixel value converted to points",
			input:    "100px",
			expected: GridTrack{Value: 75, Unit: GridTrackPoints},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseGridTrackToken(tc.input, ctx)
			assert.InDelta(t, tc.expected.Value, result.Value, 0.001)
			assert.Equal(t, tc.expected.Unit, result.Unit)
		})
	}
}

func TestParseGridLine(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected GridLine
	}{
		{
			name:     "empty string returns default",
			input:    "",
			expected: DefaultGridLine(),
		},
		{
			name:     "auto returns default",
			input:    "auto",
			expected: DefaultGridLine(),
		},
		{
			name:     "positive line number",
			input:    "2",
			expected: GridLine{Line: 2},
		},
		{
			name:     "negative line number",
			input:    "-1",
			expected: GridLine{Line: -1},
		},
		{
			name:     "span with count 2",
			input:    "span 2",
			expected: GridLine{Span: 2},
		},
		{
			name:     "span with count 3",
			input:    "span 3",
			expected: GridLine{Span: 3},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseGridLine(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseGridShorthand(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedStart GridLine
		expectedEnd   GridLine
	}{
		{
			name:          "single value gives start line only",
			input:         "1",
			expectedStart: GridLine{Line: 1},
			expectedEnd:   DefaultGridLine(),
		},
		{
			name:          "two values separated by slash",
			input:         "1 / 3",
			expectedStart: GridLine{Line: 1},
			expectedEnd:   GridLine{Line: 3},
		},
		{
			name:          "span start and line end",
			input:         "span 2 / 4",
			expectedStart: GridLine{Span: 2},
			expectedEnd:   GridLine{Line: 4},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := parseGridShorthand(tc.input)
			assert.Equal(t, tc.expectedStart, start)
			assert.Equal(t, tc.expectedEnd, end)
		})
	}
}

func TestParseGridTemplateAreas(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected [][]string
	}{
		{
			name:  "three row layout with spanning areas",
			input: `"header header" "sidebar main" "footer footer"`,
			expected: [][]string{
				{"header", "header"},
				{"sidebar", "main"},
				{"footer", "footer"},
			},
		},
		{
			name:  "simple two by two grid",
			input: `"a b" "c d"`,
			expected: [][]string{
				{"a", "b"},
				{"c", "d"},
			},
		},
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseGridTemplateAreas(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseGridAutoFlow(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected GridAutoFlowType
	}{
		{
			name:     "row keyword",
			input:    "row",
			expected: GridAutoFlowRow,
		},
		{
			name:     "column keyword",
			input:    "column",
			expected: GridAutoFlowColumn,
		},
		{
			name:     "row dense",
			input:    "row dense",
			expected: GridAutoFlowRowDense,
		},
		{
			name:     "dense row is equivalent to row dense",
			input:    "dense row",
			expected: GridAutoFlowRowDense,
		},
		{
			name:     "column dense",
			input:    "column dense",
			expected: GridAutoFlowColumnDense,
		},
		{
			name:     "dense column is equivalent to column dense",
			input:    "dense column",
			expected: GridAutoFlowColumnDense,
		},
		{
			name:     "dense alone defaults to row dense",
			input:    "dense",
			expected: GridAutoFlowRowDense,
		},
		{
			name:     "unknown value defaults to row",
			input:    "unknown",
			expected: GridAutoFlowRow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseGridAutoFlow(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
