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

import (
	"bytes"
	"fmt"
	"math"
	"sync"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/segmenter"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

const (
	// variableWeightStep is the increment between weight stops when
	// registering a variable font's weight axis.
	variableWeightStep = 100

	// fallbackAdvanceFraction is the fraction of the font size used as
	// the advance width when no matching font is found.
	fallbackAdvanceFraction = 0.5

	// defaultAscentFraction is the fraction of font size used as ascent
	// when real font metrics are unavailable.
	defaultAscentFraction = 0.8

	// defaultDescentFraction is the fraction of font size used as
	// descent when real font metrics are unavailable.
	defaultDescentFraction = 0.2

	// defaultUnitsPerEm is the fallback units-per-em value when no real
	// font metrics are available.
	defaultUnitsPerEm = 1000

	// defaultFallbackWeight is the normal CSS font-weight used as a
	// last resort when resolving a font descriptor.
	defaultFallbackWeight = 400

	// fixedPointScale converts a floating-point ppem value to the
	// fixed.Int26_6 representation used by go-text.
	fixedPointScale = 64.0
)

// fontKey uniquely identifies a registered font by family, weight, and style.
type fontKey struct {
	// family is the CSS font-family name.
	family string

	// weight is the CSS font-weight value.
	weight int

	// style is the font style variant.
	style layouter_domain.FontStyle
}

// fontRecord stores a parsed font face together with its registration metadata.
type fontRecord struct {
	// face is the parsed go-text font face.
	face *font.Face

	// family is the CSS font-family name.
	family string

	// data is the raw TTF or OTF font bytes.
	data []byte

	// style is the font style variant.
	style layouter_domain.FontStyle

	// weight is the CSS font-weight value.
	weight int
}

// GoTextFontMetrics implements FontMetricsPort using the go-text/typesetting
// library for HarfBuzz-based text shaping with full GSUB/GPOS support.
type GoTextFontMetrics struct {
	// fonts maps font keys to their parsed records.
	fonts map[fontKey]*fontRecord

	// fallback is the ordered list of font records used when no exact match is found.
	fallback []*fontRecord

	// shaper is the HarfBuzz shaper instance.
	shaper shaping.HarfbuzzShaper

	// mutex guards concurrent access to the shaper and font faces.
	mutex sync.Mutex
}

// mustTag converts a 4-character string to an OpenType tag (uint32).
//
// Takes s (string) which is the 4-character tag string.
//
// Returns font.Tag which is the corresponding OpenType tag.
func mustTag(s string) font.Tag {
	return font.Tag(uint32(s[0])<<24 | uint32(s[1])<<16 | uint32(s[2])<<8 | uint32(s[3]))
}

// NewGoTextFontMetrics creates a new GoTextFontMetrics from a slice of font
// registration entries.
//
// Takes entries ([]layouter_dto.FontEntry) which describes the fonts to
// register.
//
// Returns *GoTextFontMetrics which is the configured metrics adapter.
// Returns error which is non-nil if any font data fails to parse.
func NewGoTextFontMetrics(entries []layouter_dto.FontEntry) (*GoTextFontMetrics, error) {
	fonts := make(map[fontKey]*fontRecord, len(entries))
	fallback := make([]*fontRecord, 0, len(entries))

	for _, entry := range entries {
		if entry.IsVariable {
			baseFace, parseError := font.ParseTTF(bytes.NewReader(entry.Data))
			if parseError != nil {
				return nil, fmt.Errorf("parse variable font %q: %w", entry.Family, parseError)
			}
			for weight := entry.WeightMin; weight <= entry.WeightMax; weight += variableWeightStep {
				face := font.NewFace(baseFace.Font)
				face.SetVariations([]font.Variation{
					{Tag: mustTag("wght"), Value: float32(weight)},
				})
				key := fontKey{
					family: entry.Family,
					weight: weight,
					style:  layouter_domain.FontStyle(entry.Style),
				}
				record := &fontRecord{
					face:   face,
					data:   entry.Data,
					family: entry.Family,
					weight: weight,
					style:  layouter_domain.FontStyle(entry.Style),
				}
				fonts[key] = record
				fallback = append(fallback, record)
			}
			continue
		}

		face, parseError := font.ParseTTF(bytes.NewReader(entry.Data))
		if parseError != nil {
			return nil, fmt.Errorf("parse font %q weight=%d style=%d: %w",
				entry.Family, entry.Weight, entry.Style, parseError)
		}

		key := fontKey{
			family: entry.Family,
			weight: entry.Weight,
			style:  layouter_domain.FontStyle(entry.Style),
		}

		record := &fontRecord{
			face:   face,
			data:   entry.Data,
			family: entry.Family,
			weight: entry.Weight,
			style:  layouter_domain.FontStyle(entry.Style),
		}
		fonts[key] = record
		fallback = append(fallback, record)
	}

	return &GoTextFontMetrics{
		fonts:    fonts,
		fallback: fallback,
	}, nil
}

// MeasureText returns the width in points of the given text string when
// rendered with the specified font and size, using HarfBuzz shaping for
// accurate GSUB/GPOS-aware measurement.
//
// The shaper is invoked at CSS pixel ppem rather than point ppem. go-text's
// HarfBuzz wrapper applies Ceil() to the ppem before computing the font
// scale, which distorts fractional ppem values. Since CSS pixel sizes are
// typically integers (e.g. font-size: 14px becomes 10.5pt, but 14px is
// integer), shaping at CSS pixels avoids this rounding and matches Chrome's
// HarfBuzz behaviour. The output is then converted back to points.
//
// Takes fontDescriptor (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
// Takes text (string) which is the text to measure.
// Takes direction (DirectionType) which is the text direction.
//
// Returns the total advance width in points.
//
// Safe for concurrent use; the shaper is guarded by a mutex.
func (m *GoTextFontMetrics) MeasureText(
	fontDescriptor layouter_domain.FontDescriptor,
	size float64,
	text string,
	direction layouter_domain.DirectionType,
) float64 {
	record := m.resolveFont(fontDescriptor)
	if record == nil {
		return float64(len([]rune(text))) * size * fallbackAdvanceFraction
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return 0
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	script, lang := detectScriptAndLanguage(runes)

	cssPixelSize := size / layouter_domain.PixelsToPoints

	output := m.shaper.Shape(shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: mapDirection(direction),
		Face:      record.face,
		Size:      fixed.Int26_6(cssPixelSize * fixedPointScale),
		Script:    script,
		Language:  lang,
	})

	return fixedToFloat(output.Advance) * layouter_domain.PixelsToPoints
}

// ShapeText produces positioned glyphs for the given text using HarfBuzz
// shaping, applying kerning, GSUB, and GPOS. Like MeasureText, the shaper
// is invoked at CSS pixel ppem to avoid go-text's Ceil() rounding on
// fractional ppem values, and the output is converted back to points.
//
// Takes fontDescriptor (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
// Takes text (string) which is the text to shape.
// Takes direction (DirectionType) which is the text direction.
//
// Returns a slice of glyph positions, one per output glyph.
//
// Safe for concurrent use; the shaper is guarded by a mutex.
func (m *GoTextFontMetrics) ShapeText(
	fontDescriptor layouter_domain.FontDescriptor,
	size float64,
	text string,
	direction layouter_domain.DirectionType,
) []layouter_domain.GlyphPosition {
	record := m.resolveFont(fontDescriptor)
	if record == nil {
		return fallbackShapeText(size, text)
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	script, lang := detectScriptAndLanguage(runes)

	cssPixelSize := size / layouter_domain.PixelsToPoints

	output := m.shaper.Shape(shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: mapDirection(direction),
		Face:      record.face,
		Size:      fixed.Int26_6(cssPixelSize * fixedPointScale),
		Script:    script,
		Language:  lang,
	})

	scale := layouter_domain.PixelsToPoints
	positions := make([]layouter_domain.GlyphPosition, len(output.Glyphs))
	for index, glyph := range output.Glyphs {
		var glyphID uint16
		if uint32(glyph.GlyphID) <= math.MaxUint16 {
			glyphID = uint16(glyph.GlyphID) //nolint:gosec // guarded by the bounds check above
		}
		positions[index] = layouter_domain.GlyphPosition{
			GlyphID:      glyphID,
			XOffset:      fixedToFloat(glyph.XOffset) * scale,
			YOffset:      fixedToFloat(glyph.YOffset) * scale,
			XAdvance:     fixedToFloat(glyph.Advance) * scale,
			ClusterIndex: glyph.TextIndex(),
			RuneCount:    glyph.RunesCount(),
		}
	}

	return positions
}

// GetMetrics returns the vertical metrics (ascent, descent, line gap,
// cap height, x-height) for the specified font at the given size.
//
// Takes fontDescriptor (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
//
// Returns the vertical metrics for the font at the given size.
//
// Safe for concurrent use; font face access is guarded by a mutex.
func (m *GoTextFontMetrics) GetMetrics(
	fontDescriptor layouter_domain.FontDescriptor,
	size float64,
) layouter_domain.FontMetrics {
	record := m.resolveFont(fontDescriptor)
	if record == nil {
		return layouter_domain.FontMetrics{
			Ascent:     size * defaultAscentFraction,
			Descent:    size * defaultDescentFraction,
			UnitsPerEm: defaultUnitsPerEm,
		}
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	unitsPerEm := record.face.Upem()
	scale := size / float64(unitsPerEm)

	extents, hasExtents := record.face.FontHExtents()

	if !hasExtents {
		return layouter_domain.FontMetrics{
			Ascent:     size * defaultAscentFraction,
			Descent:    size * defaultDescentFraction,
			UnitsPerEm: int(unitsPerEm),
		}
	}

	return layouter_domain.FontMetrics{
		Ascent:     float64(extents.Ascender) * scale,
		Descent:    -float64(extents.Descender) * scale,
		LineGap:    float64(extents.LineGap) * scale,
		CapHeight:  float64(record.face.LineMetric(font.CapHeight)) * scale,
		XHeight:    float64(record.face.LineMetric(font.XHeight)) * scale,
		UnitsPerEm: int(unitsPerEm),
	}
}

// ResolveFallback returns a font descriptor for a font that contains the
// given character, walking the fallback chain if the primary font lacks
// coverage.
//
// Takes fontDescriptor (FontDescriptor) which is the primary font.
// Takes character (rune) which is the character needing a fallback.
//
// Returns a FontDescriptor for a font containing the character, or the
// original if no fallback has coverage.
//
// Safe for concurrent use; font face access is guarded by a mutex.
func (m *GoTextFontMetrics) ResolveFallback(
	fontDescriptor layouter_domain.FontDescriptor,
	character rune,
) layouter_domain.FontDescriptor {
	primaryRecord := m.resolveFont(fontDescriptor)
	if primaryRecord != nil {
		m.mutex.Lock()
		_, hasGlyph := primaryRecord.face.NominalGlyph(character)
		m.mutex.Unlock()
		if hasGlyph {
			return fontDescriptor
		}
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, record := range m.fonts {
		_, hasGlyph := record.face.NominalGlyph(character)
		if hasGlyph {
			return layouter_domain.FontDescriptor{
				Family: key.family,
				Weight: key.weight,
				Style:  key.style,
			}
		}
	}

	return fontDescriptor
}

// GetFontData returns the raw TTF bytes for the font matching the given
// descriptor. This is used by the PDF embedder for font subsetting.
//
// Takes fontDescriptor (FontDescriptor) which identifies the font.
//
// Returns the raw font bytes and true if found, or nil and false otherwise.
func (m *GoTextFontMetrics) GetFontData(
	fontDescriptor layouter_domain.FontDescriptor,
) ([]byte, bool) {
	record := m.resolveFont(fontDescriptor)
	if record == nil {
		return nil, false
	}
	return record.data, true
}

// GetFontFace returns the go-text Face for the font matching the given
// descriptor. Used by the PDF pipeline to compute variation-aware glyph
// advance widths for variable fonts.
//
// Takes fontDescriptor (FontDescriptor) which identifies the font.
//
// Returns the *font.Face if found, or nil otherwise.
func (m *GoTextFontMetrics) GetFontFace(
	fontDescriptor layouter_domain.FontDescriptor,
) *font.Face {
	record := m.resolveFont(fontDescriptor)
	if record == nil {
		return nil
	}
	return record.face
}

// SplitGraphemeClusters segments text into grapheme clusters using
// Unicode UAX #29 rules via the go-text/typesetting segmenter.
//
// Takes text (string) which is the text to segment.
//
// Returns []string which is the list of grapheme clusters.
func (*GoTextFontMetrics) SplitGraphemeClusters(text string) []string {
	if text == "" {
		return nil
	}

	var seg segmenter.Segmenter
	seg.Init([]rune(text))
	iter := seg.GraphemeIterator()

	var clusters []string
	for iter.Next() {
		clusters = append(clusters, string(iter.Grapheme().Text))
	}
	return clusters
}

// mapDirection converts a layouter DirectionType to a go-text di.Direction.
//
// Takes d (DirectionType) which is the layouter direction.
//
// Returns di.Direction which is the go-text direction.
func mapDirection(d layouter_domain.DirectionType) di.Direction {
	if d == layouter_domain.DirectionRTL {
		return di.DirectionRTL
	}
	return di.DirectionLTR
}

// resolveFont looks up the best matching fontRecord for the given descriptor,
// falling back through style, weight, and the global fallback chain.
//
// Takes fontDescriptor (FontDescriptor) which identifies the desired font.
//
// Returns *fontRecord which is the matched record, or nil if no fonts are registered.
func (m *GoTextFontMetrics) resolveFont(
	fontDescriptor layouter_domain.FontDescriptor,
) *fontRecord {
	key := fontKey{
		family: fontDescriptor.Family,
		weight: fontDescriptor.Weight,
		style:  fontDescriptor.Style,
	}
	if record, exists := m.fonts[key]; exists {
		return record
	}

	key.style = layouter_domain.FontStyleNormal
	if record, exists := m.fonts[key]; exists {
		return record
	}

	key.weight = defaultFallbackWeight
	if record, exists := m.fonts[key]; exists {
		return record
	}

	for k, record := range m.fonts {
		if k.weight == fontDescriptor.Weight && k.style == fontDescriptor.Style {
			return record
		}
	}
	for k, record := range m.fonts {
		if k.weight == fontDescriptor.Weight && k.style == layouter_domain.FontStyleNormal {
			return record
		}
	}

	if len(m.fallback) > 0 {
		return m.fallback[0]
	}

	return nil
}

// fallbackShapeText produces synthetic glyph positions when no real font is
// available, assigning each rune a uniform advance width.
//
// Takes size (float64) which is the font size in points.
// Takes text (string) which is the text to shape.
//
// Returns []GlyphPosition which is a position per rune with uniform advance.
func fallbackShapeText(
	size float64,
	text string,
) []layouter_domain.GlyphPosition {
	runes := []rune(text)
	positions := make([]layouter_domain.GlyphPosition, len(runes))
	advance := size * fallbackAdvanceFraction
	for index, character := range runes {
		positions[index] = layouter_domain.GlyphPosition{
			GlyphID:      uint16(character),
			XAdvance:     advance,
			ClusterIndex: index,
			RuneCount:    1,
		}
	}
	return positions
}

// detectScriptAndLanguage examines the runes in the text to find the
// dominant non-Common, non-Inherited script and returns the
// corresponding HarfBuzz script tag and a default language for that
// script. Falls back to Latin/EN when the text contains only common
// characters (punctuation, digits, emoji).
//
// Takes runes ([]rune) which is the text to analyse.
//
// Returns language.Script which is the detected script tag.
// Returns language.Language which is the default language for the script.
func detectScriptAndLanguage(runes []rune) (language.Script, language.Language) {
	for _, r := range runes {
		script := language.LookupScript(r)
		if script == language.Common || script == language.Inherited || script == language.Unknown {
			continue
		}
		lang, ok := language.ScriptToLang[script]
		if ok {
			return script, lang.Language()
		}
		return script, language.NewLanguage("EN")
	}
	return language.Latin, language.NewLanguage("EN")
}

// fixedToFloat converts a fixed.Int26_6 value to float64.
//
// Takes value (fixed.Int26_6) which is the fixed-point value.
//
// Returns float64 which is the floating-point equivalent.
func fixedToFloat(value fixed.Int26_6) float64 {
	return float64(value) / fixedPointScale
}
