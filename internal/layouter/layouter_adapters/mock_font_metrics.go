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

package layouter_adapters

// Implements a mock FontMetricsPort for testing with deterministic, fixed-width
// character metrics. Uses the function-field pattern for test customisation.

import "piko.sh/piko/internal/layouter/layouter_domain"

const (
	// mockCharacterWidthRatio is the fraction of the font size used
	// as the advance width of each character.
	mockCharacterWidthRatio = 0.5

	// mockAscentRatio is the fraction of the font size above the
	// baseline.
	mockAscentRatio = 0.8

	// mockDescentRatio is the fraction of the font size below the
	// baseline.
	mockDescentRatio = 0.2

	// mockCapHeightRatio is the fraction of the font size used as
	// the capital letter height.
	mockCapHeightRatio = 0.7

	// mockXHeightRatio is the fraction of the font size used as
	// the x-height.
	mockXHeightRatio = 0.5

	// mockUnitsPerEm is the units-per-em value returned by the
	// mock metrics.
	mockUnitsPerEm = 1000
)

// MockFontMetrics is a test double for FontMetricsPort that returns
// deterministic fixed-width metrics. Each function field can be overridden;
// nil fields use sensible defaults.
type MockFontMetrics struct {
	// MeasureTextFunc overrides the default MeasureText behaviour
	// when non-nil.
	MeasureTextFunc func(font layouter_domain.FontDescriptor, size float64, text string) float64

	// ShapeTextFunc overrides the default ShapeText behaviour
	// when non-nil.
	ShapeTextFunc func(font layouter_domain.FontDescriptor, size float64, text string) []layouter_domain.GlyphPosition

	// GetMetricsFunc overrides the default GetMetrics behaviour
	// when non-nil.
	GetMetricsFunc func(font layouter_domain.FontDescriptor, size float64) layouter_domain.FontMetrics

	// ResolveFallbackFunc overrides the default ResolveFallback
	// behaviour when non-nil.
	ResolveFallbackFunc func(font layouter_domain.FontDescriptor, character rune) layouter_domain.FontDescriptor
}

// MeasureText returns the width of the given text string.
//
// Default behaviour assigns each character a width of 0.5em.
//
// Takes font (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
// Takes text (string) which is the text to measure.
//
// Returns the total advance width in points.
func (m *MockFontMetrics) MeasureText(font layouter_domain.FontDescriptor, size float64, text string, _ layouter_domain.DirectionType) float64 {
	if m.MeasureTextFunc != nil {
		return m.MeasureTextFunc(font, size, text)
	}
	return float64(len([]rune(text))) * size * mockCharacterWidthRatio
}

// ShapeText returns one GlyphPosition per rune with uniform advance.
//
// Default behaviour assigns each glyph an advance of 0.5em.
//
// Takes font (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
// Takes text (string) which is the text to shape.
//
// Returns a slice of glyph positions, one per rune.
func (m *MockFontMetrics) ShapeText(font layouter_domain.FontDescriptor, size float64, text string, _ layouter_domain.DirectionType) []layouter_domain.GlyphPosition {
	if m.ShapeTextFunc != nil {
		return m.ShapeTextFunc(font, size, text)
	}

	runes := []rune(text)
	glyphs := make([]layouter_domain.GlyphPosition, len(runes))

	advance := size * mockCharacterWidthRatio
	for index, character := range runes {
		glyphs[index] = layouter_domain.GlyphPosition{
			GlyphID:      uint16(character),
			XAdvance:     advance,
			ClusterIndex: index,
			RuneCount:    1,
		}
	}

	return glyphs
}

// GetMetrics returns fixed vertical font metrics.
//
// Default behaviour uses 0.8em ascent, 0.2em descent, and no
// line gap.
//
// Takes font (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
//
// Returns the vertical metrics for the font at the given size.
func (m *MockFontMetrics) GetMetrics(font layouter_domain.FontDescriptor, size float64) layouter_domain.FontMetrics {
	if m.GetMetricsFunc != nil {
		return m.GetMetricsFunc(font, size)
	}
	return layouter_domain.FontMetrics{
		Ascent:     size * mockAscentRatio,
		Descent:    size * mockDescentRatio,
		LineGap:    0,
		CapHeight:  size * mockCapHeightRatio,
		XHeight:    size * mockXHeightRatio,
		UnitsPerEm: mockUnitsPerEm,
	}
}

// ResolveFallback returns the original font unchanged.
//
// Takes font (FontDescriptor) which is the primary font.
// Takes character (rune) which is the character needing a
// fallback.
//
// Returns the original font descriptor without modification.
func (m *MockFontMetrics) ResolveFallback(font layouter_domain.FontDescriptor, character rune) layouter_domain.FontDescriptor {
	if m.ResolveFallbackFunc != nil {
		return m.ResolveFallbackFunc(font, character)
	}
	return font
}

// SplitGraphemeClusters splits text into individual runes as a
// simple approximation of grapheme cluster segmentation. The mock
// does not implement full UAX #29 rules.
//
// Takes text (string) which is the text to segment.
//
// Returns []string which is the list of single-rune strings.
func (*MockFontMetrics) SplitGraphemeClusters(text string) []string {
	clusters := make([]string, 0, len(text))
	for _, r := range text {
		clusters = append(clusters, string(r))
	}
	return clusters
}
