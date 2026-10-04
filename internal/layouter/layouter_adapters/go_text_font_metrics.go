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
	"slices"
	"sync"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/segmenter"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// variableWeightStep is the increment between weight stops when registering a variable
	// font's weight axis.
	variableWeightStep = 100

	// fallbackAdvanceFraction is the fraction of the font size used as the advance width
	// when no matching font is found.
	fallbackAdvanceFraction = 0.5

	// defaultAscentFraction is the fraction of font size used as ascent when real font
	// metrics are unavailable.
	defaultAscentFraction = 0.8

	// defaultDescentFraction is the fraction of font size used as descent when real font
	// metrics are unavailable.
	defaultDescentFraction = 0.2

	// defaultUnitsPerEm is the fallback units-per-em value when no real font metrics are
	// available.
	defaultUnitsPerEm = 1000

	// defaultFallbackWeight is the normal CSS font-weight used as a last resort when
	// resolving a font descriptor.
	defaultFallbackWeight = 400

	// fixedPointScale converts a floating-point ppem value to the fixed.Int26_6
	// representation used by go-text.
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

// fontRecord stores a parsed font together with its registration metadata. A record is
// immutable once registered and is shared by every goroutine; per-goroutine shaping state
// (faces and their glyph caches) lives in a shapingSession instead.
type fontRecord struct {
	// font is the parsed font file. Its methods are read-only, so it is safe to share across
	// goroutines.
	font *font.Font

	// family is the CSS font-family name.
	family string

	// data is the raw TTF or OTF font bytes.
	data []byte

	// coords holds the normalised variation coordinates of the instance, or nil for a static
	// font.
	coords []tables.Coord

	// style is the font style variant.
	style layouter_domain.FontStyle

	// weight is the CSS font-weight value.
	weight int

	// extents holds the unscaled horizontal font extents in font units.
	extents font.FontExtents

	// capHeight holds the unscaled capital letter height in font units.
	capHeight float32

	// xHeight holds the unscaled x-height in font units.
	xHeight float32

	// unitsPerEm holds the number of font units per em.
	unitsPerEm uint16

	// hasExtents reports whether the font defines its horizontal extents.
	hasExtents bool
}

// recordShaper pairs a goroutine-local face for one font record with a HarfBuzz shaper
// used only for that face, so the shaper's internal font cache (keyed by the shared font
// file) never mixes the variation coordinates of different instances.
type recordShaper struct {
	// face is the goroutine-local face for the record. Faces cache glyph data and are not
	// safe for concurrent use.
	face *font.Face

	// shaper is the HarfBuzz shaper dedicated to face.
	shaper shaping.HarfbuzzShaper
}

// shapingSession holds the mutable shaping state used by one goroutine at a time.
// Sessions are pooled so concurrent renders shape in parallel instead of serialising on a
// single shaper.
type shapingSession struct {
	// shapers maps each font record to its goroutine-local face and shaper, created on first
	// use.
	shapers map[*fontRecord]*recordShaper
}

// GoTextFontMetrics implements FontMetricsPort using the go-text/typesetting library for
// HarfBuzz-based text shaping with full GSUB/GPOS support.
//
// It is safe for concurrent use. The registered fonts are immutable after construction;
// shaping runs on pooled per-goroutine sessions, so concurrent renders never wait on one
// another.
type GoTextFontMetrics struct {
	// fonts maps font keys to their parsed records.
	fonts map[fontKey]*fontRecord

	// sessions pools the per-goroutine shaping state.
	sessions sync.Pool

	// fallback lists every font record in registration order. It is the deterministic order
	// used when resolving descriptors without an exact match and when searching for a font
	// that covers a character.
	fallback []*fontRecord
}

// NewGoTextFontMetrics creates a new GoTextFontMetrics from a slice of font registration
// entries.
//
// Takes entries ([]layouter_dto.FontEntry) which describes the fonts to register.
//
// Returns *GoTextFontMetrics which is the configured metrics adapter.
// Returns error which is non-nil if any font data fails to parse.
func NewGoTextFontMetrics(entries []layouter_dto.FontEntry) (*GoTextFontMetrics, error) {
	fonts := make(map[fontKey]*fontRecord, len(entries))
	fallback := make([]*fontRecord, 0, len(entries))

	for _, entry := range entries {
		records, err := newFontRecords(entry)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			fonts[newFontKey(record)] = record
			fallback = append(fallback, record)
		}
	}

	return &GoTextFontMetrics{
		fonts: fonts,
		sessions: sync.Pool{
			New: func() any { return newShapingSession() },
		},
		fallback: fallback,
	}, nil
}

// shape runs HarfBuzz over the whole of runes with the record's face at the given CSS
// pixel size.
//
// Takes record (*fontRecord) which identifies the font to shape with.
// Takes runes ([]rune) which is the text to shape; it must not be empty.
// Takes cssPixelSize (float64) which is the font size in CSS pixels.
// Takes direction (layouter_domain.DirectionType) which is the text direction.
//
// Returns shaping.Output which holds the shaped glyphs and total advance.
func (s *shapingSession) shape(
	record *fontRecord,
	runes []rune,
	cssPixelSize float64,
	direction layouter_domain.DirectionType,
) shaping.Output {
	state := s.shaperFor(record)
	script, lang := detectScriptAndLanguage(runes)
	return state.shaper.Shape(shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: mapDirection(direction),
		Face:      state.face,
		Size:      fixed.Int26_6(cssPixelSize * fixedPointScale),
		Script:    script,
		Language:  lang,
	})
}

// shaperFor returns the session's face and shaper for the record, creating them on first
// use.
//
// Takes record (*fontRecord) which identifies the font.
//
// Returns *recordShaper which is owned by this session.
func (s *shapingSession) shaperFor(record *fontRecord) *recordShaper {
	if state, exists := s.shapers[record]; exists {
		return state
	}
	state := &recordShaper{face: record.newFace(), shaper: shaping.HarfbuzzShaper{}}
	s.shapers[record] = state
	return state
}

// newFace creates a fresh face for the record with its variation coordinates applied. The
// face is owned by the caller and must not be shared between goroutines.
//
// Returns *font.Face which is ready for shaping or glyph queries.
func (r *fontRecord) newFace() *font.Face {
	face := font.NewFace(r.font)
	if len(r.coords) > 0 {
		face.SetCoords(r.coords)
	}
	return face
}

// covers reports whether the record's font maps the character to a glyph.
//
// Takes character (rune) which is the character to look up.
//
// Returns bool which is true when the font has a glyph for the character.
func (r *fontRecord) covers(character rune) bool {
	_, hasGlyph := r.font.Cmap.Lookup(character)
	return hasGlyph
}

// descriptor returns the font descriptor under which the record is registered.
//
// Returns layouter_domain.FontDescriptor which identifies the record.
func (r *fontRecord) descriptor() layouter_domain.FontDescriptor {
	return layouter_domain.FontDescriptor{
		Family: r.family,
		Weight: r.weight,
		Style:  r.style,
	}
}

// MeasureText returns the width in points of the given text string when rendered with the
// specified font and size, using HarfBuzz shaping for accurate GSUB/GPOS-aware
// measurement.
//
// The shaper is invoked at CSS pixel ppem rather than point ppem. go-text's HarfBuzz
// wrapper applies Ceil() to the ppem before computing the font scale, which distorts
// fractional ppem values. Since CSS pixel sizes are typically integers (e.g. font-size:
// 14px becomes 10.5pt, but 14px is integer), shaping at CSS pixels avoids this rounding
// and matches Chrome's HarfBuzz behaviour. The output is then converted back to points.
//
// Takes fontDescriptor (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
// Takes text (string) which is the text to measure.
// Takes direction (DirectionType) which is the text direction.
//
// Returns the total advance width in points.
//
// Each shaping call uses its own pooled session, allowing calls to run concurrently.
func (m *GoTextFontMetrics) MeasureText(
	fontDescriptor layouter_domain.FontDescriptor,
	size float64,
	text string,
	direction layouter_domain.DirectionType,
) float64 {
	_, width := m.ShapeAndMeasureText(fontDescriptor, size, text, direction)
	return width
}

// ShapeText produces positioned glyphs for the given text using HarfBuzz shaping,
// applying kerning, GSUB, and GPOS. Like MeasureText, the shaper is invoked at CSS pixel
// ppem to avoid go-text's Ceil() rounding on fractional ppem values, and the output is
// converted back to points.
//
// Takes fontDescriptor (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
// Takes text (string) which is the text to shape.
// Takes direction (DirectionType) which is the text direction.
//
// Returns a slice of glyph positions, one per output glyph.
//
// Each shaping call uses its own pooled session, allowing calls to run concurrently.
func (m *GoTextFontMetrics) ShapeText(
	fontDescriptor layouter_domain.FontDescriptor,
	size float64,
	text string,
	direction layouter_domain.DirectionType,
) []layouter_domain.GlyphPosition {
	glyphs, _ := m.ShapeAndMeasureText(fontDescriptor, size, text, direction)
	return glyphs
}

// ShapeAndMeasureText shapes the text once and returns both the positioned glyphs and the
// total advance width, giving the same results as ShapeText and MeasureText without
// shaping the text twice.
//
// Takes fontDescriptor (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
// Takes text (string) which is the text to shape.
// Takes direction (DirectionType) which is the text direction.
//
// Returns []layouter_domain.GlyphPosition which holds one position per output glyph.
// Returns float64 which is the total advance width in points.
//
// Each shaping call uses its own pooled session, allowing calls to run concurrently.
func (m *GoTextFontMetrics) ShapeAndMeasureText(
	fontDescriptor layouter_domain.FontDescriptor,
	size float64,
	text string,
	direction layouter_domain.DirectionType,
) ([]layouter_domain.GlyphPosition, float64) {
	record := m.resolveFont(fontDescriptor)
	if record == nil {
		return fallbackShapeText(size, text), float64(len([]rune(text))) * size * fallbackAdvanceFraction
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return nil, 0
	}

	session := m.acquireSession()
	output := session.shape(record, runes, size/layouter_domain.PixelsToPoints, direction)
	m.sessions.Put(session)

	return convertGlyphs(output.Glyphs), fixedToFloat(output.Advance) * layouter_domain.PixelsToPoints
}

// GetMetrics returns the vertical metrics (ascent, descent, line gap, cap height,
// x-height) for the specified font at the given size.
//
// Takes fontDescriptor (FontDescriptor) which identifies the typeface.
// Takes size (float64) which is the font size in points.
//
// Returns the vertical metrics for the font at the given size.
//
// Metric lookups read immutable values captured during registration and can run
// concurrently.
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
			LineGap:    0,
			CapHeight:  0,
			XHeight:    0,
		}
	}

	if !record.hasExtents {
		return layouter_domain.FontMetrics{
			Ascent:     size * defaultAscentFraction,
			Descent:    size * defaultDescentFraction,
			UnitsPerEm: int(record.unitsPerEm),
			LineGap:    0,
			CapHeight:  0,
			XHeight:    0,
		}
	}

	scale := size / float64(record.unitsPerEm)
	return layouter_domain.FontMetrics{
		Ascent:     float64(record.extents.Ascender) * scale,
		Descent:    -float64(record.extents.Descender) * scale,
		LineGap:    float64(record.extents.LineGap) * scale,
		CapHeight:  float64(record.capHeight) * scale,
		XHeight:    float64(record.xHeight) * scale,
		UnitsPerEm: int(record.unitsPerEm),
	}
}

// ResolveFallback returns a font descriptor for a font that contains the given character,
// walking the fallback chain if the primary font lacks coverage.
//
// The chain is searched in font registration order, so the result is deterministic. A
// covering font with the requested weight and style is preferred; otherwise the first
// registered font that covers the character is used.
//
// Takes fontDescriptor (FontDescriptor) which is the primary font.
// Takes character (rune) which is the character needing a fallback.
//
// Returns a FontDescriptor for a font containing the character, or the original if no
// fallback has coverage.
//
// Fallback lookups read immutable font coverage and can run concurrently.
func (m *GoTextFontMetrics) ResolveFallback(
	fontDescriptor layouter_domain.FontDescriptor,
	character rune,
) layouter_domain.FontDescriptor {
	if primaryRecord := m.resolveFont(fontDescriptor); primaryRecord != nil && primaryRecord.covers(character) {
		return fontDescriptor
	}

	firstCovering := slices.IndexFunc(m.fallback, func(record *fontRecord) bool {
		return record.covers(character)
	})
	if firstCovering < 0 {
		return fontDescriptor
	}

	for _, record := range m.fallback[firstCovering:] {
		if record.weight == fontDescriptor.Weight && record.style == fontDescriptor.Style && record.covers(character) {
			return record.descriptor()
		}
	}
	return m.fallback[firstCovering].descriptor()
}

// GetFontData returns the raw TTF bytes for the font matching the given descriptor. This
// is used by the PDF embedder for font subsetting.
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

// GetFontFace returns a go-text Face for the font matching the given descriptor, with the
// instance's variation coordinates applied. Used by the PDF pipeline to compute
// variation-aware glyph advance widths for variable fonts.
//
// Each call returns a new Face owned by the caller, because faces cache glyph data and
// are not safe for concurrent use.
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
	return record.newFace()
}

// SplitGraphemeClusters segments text into grapheme clusters using Unicode UAX #29 rules
// via the go-text/typesetting segmenter.
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

// acquireSession takes a shaping session from the pool, creating one when the pool is
// empty. The caller returns it with m.sessions.Put once shaping is complete.
//
// Returns *shapingSession which is owned by the caller until it is put back.
func (m *GoTextFontMetrics) acquireSession() *shapingSession {
	if session, ok := m.sessions.Get().(*shapingSession); ok {
		return session
	}
	return newShapingSession()
}

// resolveFont looks up the best matching fontRecord for the given descriptor, falling
// back through style, weight, and the registration-ordered fallback chain.
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

	for _, record := range m.fallback {
		if record.weight == fontDescriptor.Weight && record.style == fontDescriptor.Style {
			return record
		}
	}
	for _, record := range m.fallback {
		if record.weight == fontDescriptor.Weight && record.style == layouter_domain.FontStyleNormal {
			return record
		}
	}

	if len(m.fallback) > 0 {
		return m.fallback[0]
	}

	return nil
}

// newFontRecords parses a font entry into one record for a static font or one record per
// weight step for a variable font.
//
// Takes entry (layouter_dto.FontEntry) which describes the font to register.
//
// Returns []*fontRecord which holds the records in weight order.
// Returns error when the font data cannot be parsed.
func newFontRecords(entry layouter_dto.FontEntry) ([]*fontRecord, error) {
	face, parseError := font.ParseTTF(bytes.NewReader(entry.Data))
	if parseError != nil {
		if entry.IsVariable {
			return nil, fmt.Errorf("parse variable font %q: %w", entry.Family, parseError)
		}
		return nil, fmt.Errorf("parse font %q weight=%d style=%d: %w",
			entry.Family, entry.Weight, entry.Style, parseError)
	}

	if !entry.IsVariable {
		return []*fontRecord{newFontRecord(face, entry, entry.Weight)}, nil
	}

	var records []*fontRecord
	for weight := entry.WeightMin; weight <= entry.WeightMax; weight += variableWeightStep {
		instance := font.NewFace(face.Font)
		instance.SetVariations([]font.Variation{
			{Tag: mustTag("wght"), Value: float32(weight)},
		})
		records = append(records, newFontRecord(instance, entry, weight))
	}
	return records, nil
}

// newFontRecord creates a fontRecord for a parsed face, registered under the entry's
// family and style at the given weight. The face's variation coordinates and unscaled
// vertical metrics are captured so the record never needs the face again.
//
// Takes face (*font.Face) which is the parsed font face with any variations applied.
// Takes entry (layouter_dto.FontEntry) which supplies the family, style and raw font
// bytes.
// Takes weight (int) which is the CSS font-weight the face is registered at.
//
// Returns *fontRecord which holds the font and its registration metadata.
func newFontRecord(face *font.Face, entry layouter_dto.FontEntry, weight int) *fontRecord {
	extents, hasExtents := face.FontHExtents()
	return &fontRecord{
		font:       face.Font,
		family:     entry.Family,
		data:       entry.Data,
		coords:     slices.Clone(face.Coords()),
		extents:    extents,
		capHeight:  face.LineMetric(font.CapHeight),
		xHeight:    face.LineMetric(font.XHeight),
		unitsPerEm: face.Upem(),
		style:      layouter_domain.FontStyle(entry.Style),
		weight:     weight,
		hasExtents: hasExtents,
	}
}

// newFontKey creates the lookup key under which a fontRecord is registered.
//
// Takes record (*fontRecord) which is the record to key.
//
// Returns fontKey which identifies the record by family, weight and style.
func newFontKey(record *fontRecord) fontKey {
	return fontKey{
		family: record.family,
		weight: record.weight,
		style:  record.style,
	}
}

// newShapingSession creates an empty shaping session.
//
// Returns *shapingSession which creates faces and shapers lazily.
func newShapingSession() *shapingSession {
	return &shapingSession{shapers: make(map[*fontRecord]*recordShaper)}
}

// mustTag converts a 4-character string to an OpenType tag (uint32).
//
// Takes s (string) which is the 4-character tag string.
//
// Returns font.Tag which is the corresponding OpenType tag.
func mustTag(s string) font.Tag {
	return font.Tag(uint32(s[0])<<24 | uint32(s[1])<<16 | uint32(s[2])<<8 | uint32(s[3]))
}

// convertGlyphs converts shaped go-text glyphs into layouter glyph positions, scaling the
// CSS pixel output back to points.
//
// Takes glyphs ([]shaping.Glyph) which holds the shaper output.
//
// Returns []layouter_domain.GlyphPosition which holds one position per glyph.
func convertGlyphs(glyphs []shaping.Glyph) []layouter_domain.GlyphPosition {
	scale := layouter_domain.PixelsToPoints
	positions := make([]layouter_domain.GlyphPosition, len(glyphs))
	for index, glyph := range glyphs {
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

// fallbackShapeText produces synthetic glyph positions when no real font is available,
// assigning each rune a uniform advance width.
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
			GlyphID:      safeconv.RuneToUint16(character),
			XAdvance:     advance,
			ClusterIndex: index,
			RuneCount:    1,
			XOffset:      0,
			YOffset:      0,
		}
	}
	return positions
}

// detectScriptAndLanguage examines the runes in the text to find the dominant non-Common,
// non-Inherited script and returns the corresponding HarfBuzz script tag and a default
// language for that script. Falls back to Latin/EN when the text contains only common
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
