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

import "fmt"

// FontStyle represents the style variant of a font.
type FontStyle int

const (
	// FontStyleNormal is the upright (roman) style.
	FontStyleNormal FontStyle = iota

	// FontStyleItalic is the italic or oblique style.
	FontStyleItalic
)

// String returns a human-readable name for the font style.
//
// Returns string which is the style name.
func (s FontStyle) String() string {
	switch s {
	case FontStyleNormal:
		return "normal"
	case FontStyleItalic:
		return "italic"
	default:
		return "unknown"
	}
}

// FontDescriptor identifies a specific font face by family, weight, and style.
type FontDescriptor struct {
	// Family is the CSS font-family name (e.g. "Helvetica", "serif").
	Family string

	// Weight is the CSS font-weight value (100-900). 400 is normal, 700 is
	// bold.
	Weight int

	// Style is the font style variant.
	Style FontStyle
}

// String returns a human-readable representation of the font
// descriptor.
//
// Returns string which is the formatted descriptor.
func (d FontDescriptor) String() string {
	return fmt.Sprintf("%s %d %s", d.Family, d.Weight, d.Style)
}

// FontMetrics holds the vertical metrics for a font at a given size, all in
// points.
type FontMetrics struct {
	// Ascent is the distance from the baseline to the top of the tallest
	// glyph.
	Ascent float64

	// Descent is the distance from the baseline to the bottom of the lowest
	// glyph (typically negative in some systems, but stored as a positive
	// value here).
	Descent float64

	// LineGap is the recommended extra spacing between lines.
	LineGap float64

	// CapHeight is the height of capital letters above the baseline.
	CapHeight float64

	// XHeight is the height of lowercase letters (e.g. 'x') above the
	// baseline.
	XHeight float64

	// UnitsPerEm is the font design units per em square.
	UnitsPerEm int
}

// GlyphPosition describes the placement of a single glyph within a run.
type GlyphPosition struct {
	// XOffset is the horizontal offset from the current pen position.
	XOffset float64

	// YOffset is the vertical offset from the current pen position.
	YOffset float64

	// XAdvance is the horizontal distance the pen advances after drawing
	// this glyph.
	XAdvance float64

	// GlyphID is the font-internal glyph identifier.
	GlyphID uint16

	// ClusterIndex is the index of the first input rune that this glyph
	// corresponds to. When HarfBuzz produces ligatures, multiple input
	// runes map to a single glyph; ClusterIndex identifies which runes.
	ClusterIndex int

	// RuneCount is the number of input runes consumed by this
	// glyph cluster.
	RuneCount int
}
