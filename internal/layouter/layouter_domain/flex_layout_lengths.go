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
	"math"
)

// resolveFlexibleLengths distributes free space among flex items on a line using
// flex-grow or flex-shrink factors.
//
// Following CSS Flexbox spec section 9.7, the algorithm determines whether the line is
// growing or shrinking, freezes inflexible items, then iteratively redistributes free
// space until no min/max violations remain.
//
// Takes line (*flexLine) which is the flex line whose items receive distributed space.
// Takes containerMainSize (float64) which is the main axis size of the container.
// Takes mainGap (float64) which is the gap between items along the main axis.
// Takes isRowDirection (bool) which is true for row flex.
func resolveFlexibleLengths(line *flexLine, containerMainSize, mainGap float64, isRowDirection bool) {
	totalGaps := mainGap * float64(len(line.items)-1)

	growing := isFlexLineGrowing(line, totalGaps, isRowDirection, containerMainSize)
	frozen := freezeInflexibleItems(line, growing, isRowDirection, containerMainSize)

	flexFreezeLoop(line, frozen, growing, totalGaps, containerMainSize, isRowDirection)
}

// isFlexLineGrowing returns true when the sum of hypothetical main sizes (base sizes
// clamped to min/max) plus gaps does not exceed the container main size, meaning items
// should grow rather than shrink.
//
// Takes line (*flexLine) which is the flex line to check.
// Takes totalGaps (float64) which is the total gap space between items.
// Takes isRowDirection (bool) which selects horizontal or vertical clamping.
// Takes containerMainSize (float64) which is the main axis size of the container.
//
// Returns bool which is true when items should grow.
func isFlexLineGrowing(line *flexLine, totalGaps float64, isRowDirection bool, containerMainSize float64) bool {
	used := totalGaps
	for _, item := range line.items {
		used += clampFlexMainSize(item.flexBaseSize, item.box, isRowDirection, containerMainSize)
	}
	return used <= containerMainSize
}

// freezeInflexibleItems freezes items that have a zero flex factor for the current
// grow/shrink mode, setting their main size to the clamped base size.
//
// Takes line (*flexLine) which is the flex line to process.
// Takes growing (bool) which indicates the grow/shrink mode.
// Takes isRowDirection (bool) which selects horizontal or vertical clamping.
// Takes containerMainSize (float64) which is the main axis size for percentage
// resolution.
//
// Returns []bool which holds the frozen flag for each item.
func freezeInflexibleItems(line *flexLine, growing, isRowDirection bool, containerMainSize float64) []bool {
	frozen := make([]bool, len(line.items))
	for i, item := range line.items {
		if (growing && item.flexGrow == 0) || (!growing && item.flexShrink == 0) {
			item.mainSize = clampFlexMainSize(
				item.flexBaseSize, item.box, isRowDirection, containerMainSize,
			)
			item.targetMainSize = item.mainSize
			frozen[i] = true
		}
	}
	return frozen
}

// flexFreezeLoop performs the iterative freeze-and-redistribute loop per CSS Flexbox spec
// section 9.7, distributing free space to unfrozen items and freezing any that violate
// their min/max constraints until convergence.
//
// Takes line (*flexLine) which is the flex line to process.
// Takes frozen ([]bool) which tracks which items are frozen.
// Takes growing (bool) which indicates the grow/shrink mode.
// Takes totalGaps (float64) which is the total gap space.
// Takes containerMainSize (float64) which is the main axis size.
// Takes isRowDirection (bool) which selects horizontal or vertical clamping.
func flexFreezeLoop(
	line *flexLine, frozen []bool, growing bool,
	totalGaps, containerMainSize float64, isRowDirection bool,
) {
	for {
		freeSpace, totalFactor := computeFlexFreeSpace(line, frozen, growing, totalGaps, containerMainSize)
		distributeFlexFreeSpace(line, frozen, growing, freeSpace, totalFactor)

		if !freezeFlexViolators(line, frozen, isRowDirection, containerMainSize) {
			applyUnfrozenTargets(line, frozen)
			break
		}
	}
}

// computeFlexFreeSpace calculates the remaining free space and the total flex factor
// among unfrozen items.
//
// Takes line (*flexLine) which is the flex line to measure.
// Takes frozen ([]bool) which tracks which items are frozen.
// Takes growing (bool) which indicates the grow/shrink mode.
// Takes totalGaps (float64) which is the total gap space.
// Takes containerMainSize (float64) which is the main axis size.
//
// Returns freeSpace (float64) which is the remaining free space in points.
// Returns totalFactor (float64) which is the flex-grow sum when growing or weighted
// flex-shrink sum when shrinking.
func computeFlexFreeSpace(
	line *flexLine, frozen []bool, growing bool,
	totalGaps, containerMainSize float64,
) (freeSpace float64, totalFactor float64) {
	usedByFrozen := totalGaps
	for i, item := range line.items {
		if frozen[i] {
			usedByFrozen += item.mainSize
		} else {
			usedByFrozen += item.flexBaseSize
			if growing {
				totalFactor += item.flexGrow
			} else {
				totalFactor += item.flexShrink * item.flexBaseSize
			}
		}
	}
	return containerMainSize - usedByFrozen, totalFactor
}

// distributeFlexFreeSpace assigns target main sizes to unfrozen items by distributing the
// available free space according to their flex factors.
//
// Takes line (*flexLine) which is the flex line to update.
// Takes frozen ([]bool) which tracks which items are frozen.
// Takes growing (bool) which indicates the grow/shrink mode.
// Takes freeSpace (float64) which is the available free space.
// Takes totalFactor (float64) which is the total flex factor.
func distributeFlexFreeSpace(line *flexLine, frozen []bool, growing bool, freeSpace, totalFactor float64) {
	for i, item := range line.items {
		if frozen[i] {
			continue
		}
		if growing && totalFactor > 0 {
			item.targetMainSize = item.flexBaseSize +
				freeSpace*(item.flexGrow/totalFactor)
		} else if !growing && totalFactor > 0 {
			ratio := (item.flexShrink * item.flexBaseSize) / totalFactor
			item.targetMainSize = math.Max(0, item.flexBaseSize+freeSpace*ratio)
		} else {
			item.targetMainSize = item.flexBaseSize
		}
	}
}

// freezeFlexViolators clamps each unfrozen item's target main size to its min/max
// constraints and freezes any item that was clamped.
//
// Takes line (*flexLine) which is the flex line to check.
// Takes frozen ([]bool) which tracks which items are frozen.
// Takes isRowDirection (bool) which selects horizontal or vertical clamping.
// Takes containerMainSize (float64) which is the main axis size for percentage
// resolution.
//
// Returns bool which is true if at least one item was frozen.
func freezeFlexViolators(line *flexLine, frozen []bool, isRowDirection bool, containerMainSize float64) bool {
	anyFrozen := false
	for i, item := range line.items {
		if frozen[i] {
			continue
		}
		clamped := clampFlexMainSize(
			item.targetMainSize, item.box, isRowDirection, containerMainSize,
		)
		if clamped != item.targetMainSize {
			item.mainSize = clamped
			item.targetMainSize = clamped
			frozen[i] = true
			anyFrozen = true
		}
	}
	return anyFrozen
}

// applyUnfrozenTargets copies the target main size to the resolved main size for all
// items that were never frozen.
//
// Takes line (*flexLine) which is the flex line to update.
// Takes frozen ([]bool) which tracks which items are frozen.
func applyUnfrozenTargets(line *flexLine, frozen []bool) {
	for i, item := range line.items {
		if !frozen[i] {
			item.mainSize = item.targetMainSize
		}
	}
}

// clampFlexMainSize applies min/max main-axis constraints to a flex item's border-box
// size.
//
// Takes size (float64) which is the border-box main size.
// Takes box (*LayoutBox) which is the flex item.
// Takes isRowDirection (bool) which is true for row flex.
// Takes containerMainSize (float64) for percentage resolution.
//
// Returns the clamped border-box size.
func clampFlexMainSize(size float64, box *LayoutBox, isRowDirection bool, containerMainSize float64) float64 {
	if isRowDirection {
		if !box.Style.MinWidth.IsAuto() && !box.Style.MinWidth.IsFitContent() {
			minWidth := resolveFlexDimension(box, box.Style.MinWidth, containerMainSize, true)
			size = math.Max(size, minWidth)
		}
		if !box.Style.MaxWidth.IsAuto() && !box.Style.MaxWidth.IsFitContent() {
			maxWidth := resolveFlexDimension(box, box.Style.MaxWidth, containerMainSize, true)
			size = math.Min(size, maxWidth)
		}
	} else {
		if !box.Style.MinHeight.IsAuto() && !box.Style.MinHeight.IsFitContent() {
			minHeight := resolveFlexDimension(box, box.Style.MinHeight, 0, false)
			size = math.Max(size, minHeight)
		}
		if !box.Style.MaxHeight.IsAuto() && !box.Style.MaxHeight.IsFitContent() {
			maxHeight := resolveFlexDimension(box, box.Style.MaxHeight, 0, false)
			size = math.Min(size, maxHeight)
		}
	}
	return size
}
