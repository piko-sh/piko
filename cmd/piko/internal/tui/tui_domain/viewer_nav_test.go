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
	"testing"
)

func TestCursorToItemIndex(t *testing.T) {
	testCases := []struct {
		expanded  map[int]int
		name      string
		cursor    int
		itemCount int
		mode      NavigationMode
		expected  int
	}{
		{
			name:      "cursor on first item",
			cursor:    0,
			itemCount: 3,
			expanded:  nil,
			mode:      NavigationSimple,
			expected:  0,
		},
		{
			name:      "cursor on second item",
			cursor:    1,
			itemCount: 3,
			expanded:  nil,
			mode:      NavigationSimple,
			expected:  1,
		},
		{
			name:      "cursor beyond items",
			cursor:    10,
			itemCount: 3,
			expanded:  nil,
			mode:      NavigationSimple,
			expected:  -1,
		},
		{
			name:      "cursor in expanded detail simple mode",
			cursor:    2,
			itemCount: 3,
			expanded:  map[int]int{0: 3},
			mode:      NavigationSimple,
			expected:  0,
		},
		{
			name:      "cursor after expanded in skip mode",
			cursor:    4,
			itemCount: 3,
			expanded:  map[int]int{0: 3},
			mode:      NavigationSkipLine,
			expected:  1,
		},
		{
			name:      "cursor on last expanded detail line",
			cursor:    3,
			itemCount: 2,
			expanded:  map[int]int{0: 3},
			mode:      NavigationSimple,
			expected:  0,
		},
		{
			name:      "cursor on second item after first expanded",
			cursor:    4,
			itemCount: 2,
			expanded:  map[int]int{0: 3},
			mode:      NavigationSimple,
			expected:  1,
		},
		{
			name:      "empty list",
			cursor:    0,
			itemCount: 0,
			expanded:  nil,
			mode:      NavigationSimple,
			expected:  -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isExpanded := func(index int) bool {
				_, ok := tc.expanded[index]
				return ok
			}
			expandedLineCount := func(index int) int {
				return tc.expanded[index]
			}

			result := CursorToItemIndex(tc.cursor, tc.itemCount, isExpanded, expandedLineCount, tc.mode)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestCalculateLineCount(t *testing.T) {
	testCases := []struct {
		expanded  map[int]int
		name      string
		itemCount int
		expected  int
	}{
		{
			name:      "empty list",
			itemCount: 0,
			expanded:  nil,
			expected:  0,
		},
		{
			name:      "no expansions",
			itemCount: 5,
			expanded:  nil,
			expected:  5,
		},
		{
			name:      "one item expanded",
			itemCount: 3,
			expanded:  map[int]int{1: 3},
			expected:  6,
		},
		{
			name:      "multiple items expanded",
			itemCount: 3,
			expanded:  map[int]int{0: 2, 2: 4},
			expected:  9,
		},
		{
			name:      "all items expanded",
			itemCount: 2,
			expanded:  map[int]int{0: 1, 1: 1},
			expected:  4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isExpanded := func(index int) bool {
				_, ok := tc.expanded[index]
				return ok
			}
			expandedLineCount := func(index int) int {
				return tc.expanded[index]
			}

			result := CalculateLineCount(tc.itemCount, isExpanded, expandedLineCount)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestHandleNavigationKey(t *testing.T) {
	positions := []int{0, 1, 2, 3, 4}
	lineCount := 5
	visibleHeight := 3

	testCases := []struct {
		name           string
		key            string
		cursor         int
		scrollOffset   int
		expectedCursor int
		handled        bool
	}{
		{
			name:           "move down with j",
			key:            "j",
			cursor:         0,
			scrollOffset:   0,
			expectedCursor: 1,
			handled:        true,
		},
		{
			name:           "move up with k",
			key:            "k",
			cursor:         2,
			scrollOffset:   0,
			expectedCursor: 1,
			handled:        true,
		},
		{
			name:           "go to top with g",
			key:            "g",
			cursor:         3,
			scrollOffset:   1,
			expectedCursor: 0,
			handled:        true,
		},
		{
			name:           "go to bottom with G",
			key:            "G",
			cursor:         0,
			scrollOffset:   0,
			expectedCursor: 4,
			handled:        true,
		},
		{
			name:           "unknown key not handled",
			key:            "x",
			cursor:         2,
			scrollOffset:   0,
			expectedCursor: 2,
			handled:        false,
		},
		{
			name:           "at top cannot go up",
			key:            "k",
			cursor:         0,
			scrollOffset:   0,
			expectedCursor: 0,
			handled:        true,
		},
		{
			name:           "at bottom cannot go down",
			key:            "j",
			cursor:         4,
			scrollOffset:   0,
			expectedCursor: 4,
			handled:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message := createTestKeyMessage(tc.key)
			newCursor, _, handled := HandleNavigationKey(message, tc.cursor, tc.scrollOffset, visibleHeight, positions, lineCount)

			if handled != tc.handled {
				t.Errorf("expected handled=%v, got %v", tc.handled, handled)
			}
			if newCursor != tc.expectedCursor {
				t.Errorf("expected cursor=%d, got %d", tc.expectedCursor, newCursor)
			}
		})
	}
}

func TestHandleNavigationKey_EmptyPositions(t *testing.T) {
	message := createTestKeyMessage("j")
	newCursor, newScroll, handled := HandleNavigationKey(message, 0, 0, 10, nil, 0)

	if handled {
		t.Error("expected not handled for empty positions")
	}
	if newCursor != 0 {
		t.Errorf("expected cursor unchanged, got %d", newCursor)
	}
	if newScroll != 0 {
		t.Errorf("expected scroll unchanged, got %d", newScroll)
	}
}

func TestHandleNavigationKey_ScrollResetOnGoToTop(t *testing.T) {
	positions := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	message := createTestKeyMessage("g")

	_, newScroll, handled := HandleNavigationKey(message, 5, 5, 5, positions, 10)

	if !handled {
		t.Error("expected handled")
	}
	if newScroll != 0 {
		t.Errorf("expected scroll reset to 0, got %d", newScroll)
	}
}

func TestHandlePageUp(t *testing.T) {
	positions := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	testCases := []struct {
		name          string
		cursor        int
		visibleHeight int
		expected      int
	}{
		{
			name:          "page up from middle",
			cursor:        5,
			visibleHeight: 3,
			expected:      2,
		},
		{
			name:          "page up from near top",
			cursor:        2,
			visibleHeight: 5,
			expected:      0,
		},
		{
			name:          "page up from top",
			cursor:        0,
			visibleHeight: 3,
			expected:      0,
		},
		{
			name:          "page up exactly one page",
			cursor:        6,
			visibleHeight: 3,
			expected:      3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handlePageUp(tc.cursor, tc.visibleHeight, positions)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestHandlePageDown(t *testing.T) {
	positions := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	lineCount := 10

	testCases := []struct {
		name          string
		cursor        int
		visibleHeight int
		expected      int
	}{
		{
			name:          "page down from start",
			cursor:        0,
			visibleHeight: 3,
			expected:      3,
		},
		{
			name:          "page down from middle",
			cursor:        5,
			visibleHeight: 3,
			expected:      8,
		},
		{
			name:          "page down from near end",
			cursor:        7,
			visibleHeight: 5,
			expected:      9,
		},
		{
			name:          "page down at end",
			cursor:        9,
			visibleHeight: 3,
			expected:      9,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handlePageDown(tc.cursor, tc.visibleHeight, lineCount, positions)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestIsKeyUp(t *testing.T) {
	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
		{name: "k key", key: "k", expected: true},
		{name: "up key", key: "up", expected: true},
		{name: "j key", key: "j", expected: false},
		{name: "other key", key: "x", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isKeyUp(tc.key)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestIsKeyDown(t *testing.T) {
	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
		{name: "j key", key: "j", expected: true},
		{name: "down key", key: "down", expected: true},
		{name: "k key", key: "k", expected: false},
		{name: "other key", key: "x", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isKeyDown(tc.key)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestIsKeyPageUp(t *testing.T) {
	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
		{name: "pgup key", key: "pgup", expected: true},
		{name: "ctrl+u key", key: "ctrl+u", expected: true},
		{name: "other key", key: "x", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isKeyPageUp(tc.key)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestIsKeyPageDown(t *testing.T) {
	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
		{name: "pgdown key", key: "pgdown", expected: true},
		{name: "ctrl+d key", key: "ctrl+d", expected: true},
		{name: "other key", key: "x", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isKeyPageDown(tc.key)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestHandleMoveUp(t *testing.T) {
	positions := []int{0, 2, 4, 6}

	testCases := []struct {
		name     string
		cursor   int
		expected int
	}{
		{name: "from middle", cursor: 4, expected: 2},
		{name: "from top", cursor: 0, expected: 0},
		{name: "between positions", cursor: 3, expected: 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handleMoveUp(tc.cursor, positions)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestHandleMoveDown(t *testing.T) {
	positions := []int{0, 2, 4, 6}

	testCases := []struct {
		name     string
		cursor   int
		expected int
	}{
		{name: "from start", cursor: 0, expected: 2},
		{name: "from middle", cursor: 2, expected: 4},
		{name: "from end", cursor: 6, expected: 6},
		{name: "between positions", cursor: 3, expected: 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handleMoveDown(tc.cursor, positions)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestCursorInExpandedRange(t *testing.T) {
	testCases := []struct {
		name           string
		linePos        int
		cursorPosition int
		extraLines     int
		found          bool
		endLinePos     int
	}{
		{
			name:           "cursor in range",
			linePos:        5,
			cursorPosition: 6,
			extraLines:     3,
			found:          true,
			endLinePos:     6,
		},
		{
			name:           "cursor at start of range",
			linePos:        5,
			cursorPosition: 5,
			extraLines:     3,
			found:          true,
			endLinePos:     5,
		},
		{
			name:           "cursor not in range",
			linePos:        5,
			cursorPosition: 10,
			extraLines:     3,
			found:          false,
			endLinePos:     8,
		},
		{
			name:           "zero extra lines",
			linePos:        5,
			cursorPosition: 5,
			extraLines:     0,
			found:          false,
			endLinePos:     5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			found, endPosition := cursorInExpandedRange(tc.linePos, tc.cursorPosition, tc.extraLines)
			if found != tc.found {
				t.Errorf("expected found=%v, got %v", tc.found, found)
			}
			if endPosition != tc.endLinePos {
				t.Errorf("expected endLinePos=%d, got %d", tc.endLinePos, endPosition)
			}
		})
	}
}
