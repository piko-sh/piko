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

// MeasureContentExtent walks the box tree and returns the maximum Y
// coordinate reached by any box's margin-box bottom edge. This is
// the total content height needed for auto-height pages.
//
// Takes root (*LayoutBox) which is the root of the laid-out box tree.
//
// Returns float64 which is the bottom edge of the lowest box in
// points.
func MeasureContentExtent(root *LayoutBox) float64 {
	maxY := 0.0
	measureExtent(root, &maxY)
	return maxY
}

// measureExtent recursively updates maxY with the bottom edge of each box's
// margin box.
//
// Takes box (*LayoutBox) which is the current box to measure.
// Takes maxY (*float64) which is the running maximum Y coordinate.
func measureExtent(box *LayoutBox, maxY *float64) {
	bottom := box.ContentY + box.MarginBoxHeight()
	if bottom > *maxY {
		*maxY = bottom
	}
	for _, child := range box.Children {
		measureExtent(child, maxY)
	}
}
