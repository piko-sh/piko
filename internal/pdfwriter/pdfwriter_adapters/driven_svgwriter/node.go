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

package driven_svgwriter

// SVG is a parsed SVG document ready to be rendered to PDF.
type SVG struct {
	// Root holds the top-level SVG element node.
	Root *Node

	// Defs holds the map of reusable element definitions keyed by their ID.
	Defs map[string]*Node

	// PreserveAspectRatio holds the parsed aspect ratio preservation settings.
	PreserveAspectRatio AspectRatio

	// VBox holds the parsed viewBox attribute defining the SVG coordinate system.
	VBox ViewBox

	// Width holds the explicit width of the SVG element in user units.
	Width float64

	// Height holds the explicit height of the SVG element in user units.
	Height float64
}

// IntrinsicWidth returns the SVG intrinsic width. If an explicit width is
// set it is returned; otherwise the viewBox width is used.
//
// Returns float64 which holds the effective width in user
// units, or zero if neither is available.
func (s *SVG) IntrinsicWidth() float64 {
	if s.Width > 0 {
		return s.Width
	}
	if s.VBox.Valid {
		return s.VBox.Width
	}
	return 0
}

// IntrinsicHeight returns the SVG intrinsic height. If an explicit height
// is set it is returned; otherwise the viewBox height is used.
//
// Returns float64 which holds the effective height in user
// units, or zero if neither is available.
func (s *SVG) IntrinsicHeight() float64 {
	if s.Height > 0 {
		return s.Height
	}
	if s.VBox.Valid {
		return s.VBox.Height
	}
	return 0
}

// Node represents a parsed SVG element.
type Node struct {
	// Attrs holds the element attributes keyed by attribute name.
	Attrs map[string]string

	// Tag holds the SVG element tag name.
	Tag string

	// Text holds the direct text content of the element.
	Text string

	// Children holds the child element nodes.
	Children []*Node

	// Transform holds the parsed transform matrix for this element.
	Transform Matrix
}

// ViewBox defines the SVG coordinate system.
type ViewBox struct {
	// MinX holds the minimum x coordinate of the viewBox.
	MinX float64

	// MinY holds the minimum y coordinate of the viewBox.
	MinY float64

	// Width holds the width of the viewBox in user units.
	Width float64

	// Height holds the height of the viewBox in user units.
	Height float64

	// Valid holds whether the viewBox was successfully parsed.
	Valid bool
}

// AspectRatio holds the parsed preserveAspectRatio attribute.
//
// Align is one of "none", "xMinYMin", "xMidYMin", "xMaxYMin",
// "xMinYMid", "xMidYMid", "xMaxYMid", "xMinYMax", "xMidYMax",
// "xMaxYMax". MeetOrSlice is "meet" or "slice".
type AspectRatio struct {
	// Align holds the alignment value controlling how the
	// viewBox is positioned within the viewport.
	Align string

	// MeetOrSlice holds either "meet" or "slice" controlling
	// whether the viewBox fits within or fills the viewport.
	MeetOrSlice string
}

// DefaultAspectRatio returns the SVG default: "xMidYMid meet".
//
// Returns AspectRatio which holds the default centred-meet preservation settings.
func DefaultAspectRatio() AspectRatio {
	return AspectRatio{Align: "xMidYMid", MeetOrSlice: "meet"}
}
