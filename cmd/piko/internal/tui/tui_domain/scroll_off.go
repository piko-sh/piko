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

// DefaultScrollOff is the default vim-style scroll-off margin in lines. The
// cursor is kept this many lines away from viewport edges where possible so
// the user always has a few rows of context above and below the selection.
const DefaultScrollOff = 3

// ApplyScrollOff returns a scroll offset adjusted so the cursor sits at
// least margin lines from the viewport edges, falling back to clamping
// behaviour when the viewport is too small to accommodate the margin.
//
// The function is total: it returns a sane scroll offset for any inputs,
// including negative or out-of-range cursor positions.
//
// Takes scrollOffset (int) which is the current scroll offset.
// Takes cursor (int) which is the current cursor row index.
// Takes visibleHeight (int) which is the number of rows visible in the
// viewport.
// Takes lineCount (int) which is the total number of rows in the content.
// Takes margin (int) which is the desired number of context rows around the
// cursor.
//
// Returns int which is the adjusted scroll offset.
func ApplyScrollOff(scrollOffset, cursor, visibleHeight, lineCount, margin int) int {
	if visibleHeight <= 0 {
		return scrollOffset
	}
	if margin < 0 {
		margin = 0
	}

	effectiveMargin := margin
	if effectiveMargin*2 >= visibleHeight {
		effectiveMargin = max(0, (visibleHeight-1)/2)
	}

	if cursor < scrollOffset+effectiveMargin {
		scrollOffset = cursor - effectiveMargin
	}

	if cursor >= scrollOffset+visibleHeight-effectiveMargin {
		scrollOffset = cursor - visibleHeight + 1 + effectiveMargin
	}

	maxScroll := max(0, lineCount-visibleHeight)
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	return scrollOffset
}

// AdjustScrollForCursorWithMargin is the margin-aware companion to
// AdjustScrollForCursor. Calling this with margin=0 yields identical
// behaviour to AdjustScrollForCursor; positive margins keep the cursor away
// from the viewport edges.
//
// Takes cursor (int) which is the current cursor line.
// Takes scrollOffset (int) which is the current scroll position.
// Takes visibleHeight (int) which is the number of visible lines.
// Takes lineCount (int) which is the total number of lines.
// Takes margin (int) which is the scroll-off margin.
//
// Returns int which is the adjusted scroll offset.
func AdjustScrollForCursorWithMargin(cursor, scrollOffset, visibleHeight, lineCount, margin int) int {
	if margin <= 0 {
		return AdjustScrollForCursor(cursor, scrollOffset, visibleHeight, lineCount)
	}
	return ApplyScrollOff(scrollOffset, cursor, visibleHeight, lineCount, margin)
}
