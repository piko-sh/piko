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

// floatEntry records the position and dimensions of a single floated
// element within its containing block.
type floatEntry struct {
	// x is the horizontal offset of the float from the container origin.
	x float64

	// y is the vertical offset of the float from the container origin.
	y float64

	// width is the rendered width of the floated element.
	width float64

	// height is the rendered height of the floated element.
	height float64
}

// floatContext tracks all active left and right floats within a
// containing block, used for positioning new floats and clearing.
type floatContext struct {
	// leftFloats holds entries for elements floated to the left.
	leftFloats []floatEntry

	// rightFloats holds entries for elements floated to the right.
	rightFloats []floatEntry
}

// placeLeftFloat positions a left-floated element, stacking it beside
// existing left floats when space permits and wrapping below them when
// the container width is exceeded.
//
// Takes cursorY (float64) which is the current vertical cursor
// position.
// Takes width (float64) which is the width of the element to float.
// Takes height (float64) which is the height of the element to float.
// Takes containerX (float64) which is the left edge of the container.
// Takes containerWidth (float64) which is the total width of the
// container.
//
// Returns floatX (float64) which is the computed horizontal position.
// Returns floatY (float64) which is the computed vertical position.
func (context *floatContext) placeLeftFloat(
	cursorY, width, height, containerX, containerWidth float64,
) (floatX float64, floatY float64) {
	x := containerX
	y := cursorY

	for _, existing := range context.leftFloats {
		if y < existing.y+existing.height && y+height > existing.y {
			if x < existing.x+existing.width {
				x = existing.x + existing.width
			}
		}
	}

	if x+width > containerX+containerWidth {
		x = containerX
		y = context.clearLeftY()
	}

	context.leftFloats = append(context.leftFloats, floatEntry{
		x: x, y: y, width: width, height: height,
	})

	return x, y
}

// placeRightFloat positions a right-floated element, stacking it beside
// existing right floats when space permits and wrapping below them when
// the element would extend past the container's left edge.
//
// Takes cursorY (float64) which is the current vertical cursor
// position.
// Takes width (float64) which is the width of the element to float.
// Takes height (float64) which is the height of the element to float.
// Takes containerX (float64) which is the left edge of the container.
// Takes containerWidth (float64) which is the total width of the
// container.
//
// Returns floatX (float64) which is the computed horizontal position.
// Returns floatY (float64) which is the computed vertical position.
func (context *floatContext) placeRightFloat(
	cursorY, width, height, containerX, containerWidth float64,
) (floatX float64, floatY float64) {
	x := containerX + containerWidth - width
	y := cursorY

	for _, existing := range context.rightFloats {
		if y < existing.y+existing.height && y+height > existing.y {
			if x+width > existing.x {
				x = existing.x - width
			}
		}
	}

	if x < containerX {
		x = containerX + containerWidth - width
		y = context.clearRightY()
	}

	context.rightFloats = append(context.rightFloats, floatEntry{
		x: x, y: y, width: width, height: height,
	})

	return x, y
}

// clearLeftY returns the Y coordinate just below all left floats.
//
// Returns float64 which is the bottom edge of the lowest left float,
// or zero when no left floats exist.
func (context *floatContext) clearLeftY() float64 {
	maxY := 0.0
	for _, entry := range context.leftFloats {
		bottom := entry.y + entry.height
		if bottom > maxY {
			maxY = bottom
		}
	}
	return maxY
}

// clearRightY returns the Y coordinate just below all right floats.
//
// Returns float64 which is the bottom edge of the lowest right float,
// or zero when no right floats exist.
func (context *floatContext) clearRightY() float64 {
	maxY := 0.0
	for _, entry := range context.rightFloats {
		bottom := entry.y + entry.height
		if bottom > maxY {
			maxY = bottom
		}
	}
	return maxY
}

// clearBothY returns the Y coordinate just below all floats on both
// sides, taking the maximum of clearLeftY and clearRightY.
//
// Returns float64 which is the bottom edge of the lowest float on
// either side, or zero when no floats exist.
func (context *floatContext) clearBothY() float64 {
	leftY := context.clearLeftY()
	rightY := context.clearRightY()
	if leftY > rightY {
		return leftY
	}
	return rightY
}

// availableWidthAtY returns the width available for content at the
// given vertical position, narrowed by any floats that overlap the
// range [y, y+height].
//
// Takes y (float64) which is the top of the vertical range.
// Takes height (float64) which is the height of the vertical range.
// Takes containerX (float64) which is the left edge of the container
// content area.
// Takes containerWidth (float64) which is the full container content
// width.
//
// Returns float64 which is the available width between floats.
func (context *floatContext) availableWidthAtY(
	y, height, containerX, containerWidth float64,
) float64 {
	leftEdge := containerX
	rightEdge := containerX + containerWidth

	for _, entry := range context.leftFloats {
		if y < entry.y+entry.height && y+height >= entry.y {
			edge := entry.x + entry.width
			if edge > leftEdge {
				leftEdge = edge
			}
		}
	}

	for _, entry := range context.rightFloats {
		if y < entry.y+entry.height && y+height >= entry.y {
			if entry.x < rightEdge {
				rightEdge = entry.x
			}
		}
	}

	available := rightEdge - leftEdge
	if available < 0 {
		return 0
	}
	return available
}

// leftOffsetAtY returns the X position of the right edge of the
// rightmost left float overlapping the range [y, y+height], or
// containerX if no left floats overlap.
//
// Takes y (float64) which is the top of the vertical range.
// Takes height (float64) which is the height of the vertical range.
// Takes containerX (float64) which is the left edge of the container
// content area.
//
// Returns float64 which is the left offset for content.
func (context *floatContext) leftOffsetAtY(y, height, containerX float64) float64 {
	leftEdge := containerX
	for _, entry := range context.leftFloats {
		if y < entry.y+entry.height && y+height >= entry.y {
			edge := entry.x + entry.width
			if edge > leftEdge {
				leftEdge = edge
			}
		}
	}
	return leftEdge
}

// clearY returns the Y coordinate past the floats indicated by the
// given clear type, dispatching to clearLeftY, clearRightY, or
// clearBothY accordingly.
//
// Takes clearType (ClearType) which selects which side to clear.
//
// Returns float64 which is the Y coordinate below the cleared floats,
// or zero for an unrecognised clear type.
func (context *floatContext) clearY(clearType ClearType) float64 {
	switch clearType {
	case ClearLeft:
		return context.clearLeftY()
	case ClearRight:
		return context.clearRightY()
	case ClearBoth:
		return context.clearBothY()
	default:
		return 0
	}
}
