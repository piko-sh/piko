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

// resolveFlexAutoMargins distributes free space on the main
// axis to items with auto margins.
//
// Takes line (*flexLine) which is the flex line to process.
// Takes context (*flexPositionContext) which provides the
// container dimensions and direction.
//
// Returns bool which is true if any auto margins were found.
func resolveFlexAutoMargins(line *flexLine, context *flexPositionContext) bool {
	autoCount, usedSpace := countAutoMainMargins(line, context.isRowDirection, context.mainGap)
	if autoCount == 0 {
		return false
	}

	freeSpace := context.containerMainSize - usedSpace
	if freeSpace < 0 {
		freeSpace = 0
	}

	perMargin := freeSpace / float64(autoCount)
	applyAutoMainMargins(line, context.isRowDirection, perMargin)
	return true
}

// countAutoMainMargins counts the number of auto margins
// on the main axis across all items and computes the total
// used space (item main sizes plus gaps).
//
// Takes line (*flexLine) which is the flex line.
// Takes isRowDirection (bool) which selects horizontal or
// vertical margins.
// Takes mainGap (float64) which is the gap between items.
//
// Returns the auto margin count and total used space.
func countAutoMainMargins(line *flexLine, isRowDirection bool, mainGap float64) (int, float64) {
	autoCount := 0
	usedSpace := mainGap * float64(len(line.items)-1)
	for _, item := range line.items {
		usedSpace += item.mainSize
		autoCount += countItemAutoMainMargins(item, isRowDirection)
	}
	return autoCount, usedSpace
}

// countItemAutoMainMargins returns the number of auto
// margins on the main axis for a single flex item (0, 1,
// or 2).
//
// Takes item (*flexItem) which is the flex item to inspect.
//
// Returns int which is the number of auto margins (0, 1, or 2).
func countItemAutoMainMargins(item *flexItem, isRowDirection bool) int {
	count := 0
	if isRowDirection {
		if item.box.Style.MarginLeft.IsAuto() {
			count++
		}
		if item.box.Style.MarginRight.IsAuto() {
			count++
		}
	} else {
		if item.box.Style.MarginTop.IsAuto() {
			count++
		}
		if item.box.Style.MarginBottom.IsAuto() {
			count++
		}
	}
	return count
}

// applyAutoMainMargins assigns the per-margin value to
// each auto margin on the main axis.
//
// Takes line (*flexLine) which is the flex line to modify.
// Takes isRowDirection (bool) which selects horizontal or vertical margins.
// Takes perMargin (float64) which is the space to assign per auto margin.
func applyAutoMainMargins(line *flexLine, isRowDirection bool, perMargin float64) {
	for _, item := range line.items {
		if isRowDirection {
			if item.box.Style.MarginLeft.IsAuto() {
				item.autoMarginStart = perMargin
			}
			if item.box.Style.MarginRight.IsAuto() {
				item.autoMarginEnd = perMargin
			}
		} else {
			if item.box.Style.MarginTop.IsAuto() {
				item.autoMarginStart = perMargin
			}
			if item.box.Style.MarginBottom.IsAuto() {
				item.autoMarginEnd = perMargin
			}
		}
	}
}

// itemAutoMainMargins returns the resolved auto margin
// values for the start and end of the main axis. Items
// without auto margins return (0, 0).
//
// Takes item (*flexItem) which is the flex item to query.
//
// Returns the before and after auto margin values.
func itemAutoMainMargins(item *flexItem, _ bool) (before, after float64) {
	return item.autoMarginStart, item.autoMarginEnd
}
