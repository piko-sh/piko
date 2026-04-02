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

import tea "charm.land/bubbletea/v2"

// NavigationMode controls how the cursor moves within an AssetViewer.
type NavigationMode int

const (
	// NavigationSimple allows the cursor to move to every line, including expanded
	// details.
	NavigationSimple NavigationMode = iota

	// NavigationSkipLine moves the cursor only to item headers, skipping expanded
	// detail lines. Use this when only top-level items should be selectable.
	NavigationSkipLine
)

// NavigablePositions returns the line positions of all items that can be
// navigated to.
//
// For NavigationSimple, this returns positions for all lines (0, 1, 2, ...,
// totalLines-1). For NavigationSkipLine, this returns only the positions of
// item header lines, skipping expanded detail lines.
//
// Takes itemCount (int) which specifies the number of items in the list.
// Takes isExpanded (func(...)) which returns true if the item at the given
// index is expanded.
// Takes expandedLineCount (func(...)) which returns the number of detail lines
// for an expanded item.
// Takes mode (NavigationMode) which specifies the navigation mode.
//
// Returns []int which contains the line positions of items that can be
// navigated to.
func NavigablePositions(
	itemCount int,
	isExpanded func(index int) bool,
	expandedLineCount func(index int) int,
	mode NavigationMode,
) []int {
	if itemCount == 0 {
		return nil
	}

	positions := make([]int, 0, itemCount)
	linePos := 0

	for i := range itemCount {
		positions = append(positions, linePos)
		linePos++

		if isExpanded(i) {
			extraLines := expandedLineCount(i)
			if mode == NavigationSimple {
				for j := range extraLines {
					positions = append(positions, linePos+j)
				}
			}
			linePos += extraLines
		}
	}

	return positions
}

// NextNavigablePosition finds the next position in the list after the cursor.
//
// Takes cursorPosition (int) which is the current cursor position.
// Takes positions ([]int) which holds the positions to move to.
//
// Returns int which is the next position after cursorPosition, or -1 if there is
// none.
func NextNavigablePosition(cursorPosition int, positions []int) int {
	for _, position := range positions {
		if position > cursorPosition {
			return position
		}
	}
	return -1
}

// PreviousNavigablePosition finds the previous position that can
// be navigated to before the cursor.
//
// Takes cursorPosition (int) which is the current cursor position.
// Takes positions ([]int) which contains the positions that can be navigated
// to.
//
// Returns int which is the previous position, or -1 if there is no previous
// position.
func PreviousNavigablePosition(cursorPosition int, positions []int) int {
	for i := len(positions) - 1; i >= 0; i-- {
		if positions[i] < cursorPosition {
			return positions[i]
		}
	}
	return -1
}

// CursorToItemIndex converts a cursor position to the matching item index.
//
// When mode is NavigationSkipLine, this only matches item header positions.
// When mode is NavigationSimple, this returns the item that contains the
// cursor position, including any expanded content lines.
//
// Takes cursorPosition (int) which is the cursor position to convert.
// Takes itemCount (int) which is the total number of items.
// Takes isExpanded (func(index int) bool) which checks if an item is expanded.
// Takes expandedLineCount (func(index int) int) which returns the number of
// extra lines for an expanded item.
// Takes mode (NavigationMode) which sets how cursor positions map to items.
//
// Returns int which is the item index at the cursor position, or -1 if the
// cursor does not match any item.
func CursorToItemIndex(
	cursorPosition int,
	itemCount int,
	isExpanded func(index int) bool,
	expandedLineCount func(index int) int,
	mode NavigationMode,
) int {
	linePos := 0

	for i := range itemCount {
		if linePos == cursorPosition {
			return i
		}
		linePos++

		if !isExpanded(i) {
			continue
		}

		extraLines := expandedLineCount(i)
		if mode != NavigationSimple {
			linePos += extraLines
			continue
		}

		found, newPos := cursorInExpandedRange(linePos, cursorPosition, extraLines)
		if found {
			return i
		}
		linePos = newPos
	}

	return -1
}

// AdjustScrollForCursor adjusts the scroll position to keep the cursor visible
// within the scroll window.
//
// Takes cursor (int) which is the current cursor line position.
// Takes scrollOffset (int) which is the current scroll position.
// Takes visibleHeight (int) which is the number of lines visible in the window.
// Takes lineCount (int) which is the total number of lines in the content.
//
// Returns int which is the adjusted scroll offset.
func AdjustScrollForCursor(cursor, scrollOffset, visibleHeight, lineCount int) int {
	if visibleHeight <= 0 {
		return scrollOffset
	}

	if cursor < scrollOffset {
		scrollOffset = cursor
	}

	if cursor >= scrollOffset+visibleHeight {
		scrollOffset = cursor - visibleHeight + 1
	}

	maxScroll := max(lineCount-visibleHeight, 0)
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	return scrollOffset
}

// CalculateLineCount returns the total number of lines including expanded
// content.
//
// Takes itemCount (int) which specifies how many items to count.
// Takes isExpanded (func(index int) bool) which checks if an item at the given
// index is expanded.
// Takes expandedLineCount (func(index int) int) which returns how many lines an
// expanded item uses.
//
// Returns int which is the total line count across all items.
func CalculateLineCount(
	itemCount int,
	isExpanded func(index int) bool,
	expandedLineCount func(index int) int,
) int {
	lineCount := 0
	for i := range itemCount {
		lineCount++
		if isExpanded(i) {
			lineCount += expandedLineCount(i)
		}
	}
	return lineCount
}

// HandleNavigationKey handles keyboard input for list navigation.
//
// Supported keys:
//   - up/k: Move cursor up
//   - down/j: Move cursor down
//   - g: Go to top
//   - G: Go to bottom
//   - pgup/ctrl+u: Page up
//   - pgdown/ctrl+d: Page down
//
// When positions is empty, returns the original values unchanged.
//
// Takes message (tea.KeyPressMsg) which is the key press event to handle.
// Takes cursor (int) which is the current cursor position.
// Takes scrollOffset (int) which is the current scroll offset.
// Takes visibleHeight (int) which is the number of visible lines.
// Takes positions ([]int) which holds the valid cursor positions.
// Takes lineCount (int) which is the total number of lines.
//
// Returns newCursor (int) which is the new cursor position.
// Returns newScrollOffset (int) which is the new scroll offset.
// Returns handled (bool) which is true if the key was handled.
func HandleNavigationKey(
	message tea.KeyPressMsg,
	cursor, scrollOffset, visibleHeight int,
	positions []int,
	lineCount int,
) (newCursor, newScrollOffset int, handled bool) {
	if len(positions) == 0 {
		return cursor, scrollOffset, false
	}

	newCursor, handled = handleNavKeyDispatch(message, cursor, visibleHeight, lineCount, positions)
	if !handled {
		return cursor, scrollOffset, false
	}

	if message.String() == "g" {
		scrollOffset = 0
	}

	newScrollOffset = AdjustScrollForCursor(newCursor, scrollOffset, visibleHeight, lineCount)
	return newCursor, newScrollOffset, true
}

// cursorInExpandedRange checks if a cursor position falls within a range of
// lines starting at a given position.
//
// Takes linePos (int) which is the starting line position.
// Takes cursorPosition (int) which is the cursor position to check.
// Takes extraLines (int) which is the number of lines in the range.
//
// Returns bool which is true if the cursor is within the range.
// Returns int which is the line position after checking the range.
func cursorInExpandedRange(linePos, cursorPosition, extraLines int) (bool, int) {
	for range extraLines {
		if linePos == cursorPosition {
			return true, linePos
		}
		linePos++
	}
	return false, linePos
}

// handleNavKeyDispatch routes a navigation key to the correct handler.
//
// Takes message (tea.KeyPressMsg) which is the key message to process.
// Takes cursor (int) which is the current cursor position.
// Takes visibleHeight (int) which is the visible height of the viewport.
// Takes lineCount (int) which is the total number of lines.
// Takes positions ([]int) which contains valid cursor positions.
//
// Returns int which is the new cursor position after navigation.
// Returns bool which indicates whether the key was handled.
func handleNavKeyDispatch(message tea.KeyPressMsg, cursor, visibleHeight, lineCount int, positions []int) (int, bool) {
	keyString := message.String()

	if isKeyUp(keyString) {
		return handleMoveUp(cursor, positions), true
	}
	if isKeyDown(keyString) {
		return handleMoveDown(cursor, positions), true
	}
	if keyString == "g" {
		return positions[0], true
	}
	if keyString == "G" {
		return positions[len(positions)-1], true
	}
	if isKeyPageUp(keyString) {
		return handlePageUp(cursor, visibleHeight, positions), true
	}
	if isKeyPageDown(keyString) {
		return handlePageDown(cursor, visibleHeight, lineCount, positions), true
	}

	return cursor, false
}

// isKeyUp checks whether the given key string is an up navigation key.
//
// Takes keyString (string) which is the string form of the key.
//
// Returns bool which is true if the key is "up" or "k".
func isKeyUp(keyString string) bool {
	return keyString == "up" || keyString == "k"
}

// isKeyDown checks whether the given key string represents a key for moving
// down.
//
// Takes keyString (string) which is the string form of the key.
//
// Returns bool which is true if the key is for moving down.
func isKeyDown(keyString string) bool {
	return keyString == "down" || keyString == "j"
}

// isKeyPageUp reports whether the given key string is a page up key.
//
// Takes keyString (string) which is the string form of the key.
//
// Returns bool which is true if the key is pgup or ctrl+u.
func isKeyPageUp(keyString string) bool {
	return keyString == "pgup" || keyString == "ctrl+u"
}

// isKeyPageDown checks whether the key string represents a page down action.
//
// Takes keyString (string) which is the string form of the key.
//
// Returns bool which is true if the key is pgdown or ctrl+d.
func isKeyPageDown(keyString string) bool {
	return keyString == "pgdown" || keyString == "ctrl+d"
}

// handleMoveUp moves the cursor to the previous navigable position.
//
// Takes cursor (int) which is the current cursor position.
// Takes positions ([]int) which contains the navigable positions.
//
// Returns int which is the new cursor position, or the current position if
// there is no previous navigable position.
func handleMoveUp(cursor int, positions []int) int {
	previousPos := PreviousNavigablePosition(cursor, positions)
	if previousPos >= 0 {
		return previousPos
	}
	return cursor
}

// handleMoveDown moves the cursor down to the next valid position.
//
// Takes cursor (int) which is the current cursor position.
// Takes positions ([]int) which contains the valid positions for navigation.
//
// Returns int which is the next position below the cursor, or the current
// position if there is no valid position below.
func handleMoveDown(cursor int, positions []int) int {
	nextPos := NextNavigablePosition(cursor, positions)
	if nextPos >= 0 {
		return nextPos
	}
	return cursor
}

// handlePageUp moves the cursor up by one page of visible lines.
//
// Takes cursor (int) which is the current cursor position.
// Takes visibleHeight (int) which is the number of visible lines per page.
// Takes positions ([]int) which contains the valid cursor positions.
//
// Returns int which is the new cursor position after moving up one page.
func handlePageUp(cursor, visibleHeight int, positions []int) int {
	targetPos := max(cursor-visibleHeight, 0)
	for i := len(positions) - 1; i >= 0; i-- {
		if positions[i] <= targetPos || i == 0 {
			return positions[i]
		}
	}
	return cursor
}

// handlePageDown moves the cursor down by one page in the viewer.
//
// Takes cursor (int) which is the current cursor position.
// Takes visibleHeight (int) which is the number of lines shown on screen.
// Takes lineCount (int) which is the total number of lines.
// Takes positions ([]int) which holds the allowed cursor positions.
//
// Returns int which is the new cursor position after moving down one page.
func handlePageDown(cursor, visibleHeight, lineCount int, positions []int) int {
	targetPos := min(cursor+visibleHeight, lineCount-1)
	newCursor := cursor
	for _, position := range positions {
		if position >= targetPos {
			return position
		}
		newCursor = position
	}
	return newCursor
}
