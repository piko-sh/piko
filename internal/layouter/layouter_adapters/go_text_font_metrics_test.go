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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/shaping"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

const (
	testVariableFamily = "NotoVariable"
	devanagariKa       = 'क'
)

func staticNotoEntries() []layouter_dto.FontEntry {
	return []layouter_dto.FontEntry{
		{Family: fonts.NotoSansFamilyName, Weight: 400, Data: fonts.NotoSansRegularTTF},
		{Family: fonts.NotoSansFamilyName, Weight: 700, Data: fonts.NotoSansBoldTTF},
	}
}

func variableNotoEntry() layouter_dto.FontEntry {
	return layouter_dto.FontEntry{
		Family:     testVariableFamily,
		Data:       fonts.NotoSansVariableTTF,
		IsVariable: true,
		WeightMin:  100,
		WeightMax:  900,
	}
}

func newTestMetrics(t *testing.T, entries ...layouter_dto.FontEntry) *GoTextFontMetrics {
	t.Helper()
	metrics, err := NewGoTextFontMetrics(entries)
	require.NoError(t, err)
	return metrics
}

func parseTestFace(t *testing.T, data []byte) *font.Face {
	t.Helper()
	face, err := font.ParseTTF(bytes.NewReader(data))
	require.NoError(t, err)
	return face
}

func TestNewGoTextFontMetrics_RegistersFonts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		entries       []layouter_dto.FontEntry
		wantRegisters []fontKey
		wantFirstKey  fontKey
		wantRecords   int
	}{
		{
			name:         "static fonts are registered once each in order",
			entries:      staticNotoEntries(),
			wantRecords:  2,
			wantFirstKey: fontKey{family: fonts.NotoSansFamilyName, weight: 400, style: layouter_domain.FontStyleNormal},
			wantRegisters: []fontKey{
				{family: fonts.NotoSansFamilyName, weight: 400, style: layouter_domain.FontStyleNormal},
				{family: fonts.NotoSansFamilyName, weight: 700, style: layouter_domain.FontStyleNormal},
			},
		},
		{
			name:         "variable font is registered at every weight step",
			entries:      []layouter_dto.FontEntry{variableNotoEntry()},
			wantRecords:  9,
			wantFirstKey: fontKey{family: testVariableFamily, weight: 100, style: layouter_domain.FontStyleNormal},
			wantRegisters: []fontKey{
				{family: testVariableFamily, weight: 100, style: layouter_domain.FontStyleNormal},
				{family: testVariableFamily, weight: 500, style: layouter_domain.FontStyleNormal},
				{family: testVariableFamily, weight: 900, style: layouter_domain.FontStyleNormal},
			},
		},
		{
			name:        "no entries registers nothing",
			entries:     nil,
			wantRecords: 0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			metrics := newTestMetrics(t, testCase.entries...)

			assert.Len(t, metrics.fallback, testCase.wantRecords)
			assert.Len(t, metrics.fonts, testCase.wantRecords)
			if testCase.wantRecords > 0 {
				assert.Equal(t, testCase.wantFirstKey, newFontKey(metrics.fallback[0]))
			}
			for _, key := range testCase.wantRegisters {
				assert.Contains(t, metrics.fonts, key)
			}
		})
	}
}

func TestNewGoTextFontMetrics_RejectsInvalidFontData(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		wantMessage string
		entry       layouter_dto.FontEntry
	}{
		{
			name:        "static font",
			entry:       layouter_dto.FontEntry{Family: "Broken", Weight: 400, Data: []byte("not a font")},
			wantMessage: `parse font "Broken" weight=400 style=0`,
		},
		{
			name:        "variable font",
			entry:       layouter_dto.FontEntry{Family: "Broken", Data: []byte("not a font"), IsVariable: true, WeightMin: 100, WeightMax: 200},
			wantMessage: `parse variable font "Broken"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			metrics, err := NewGoTextFontMetrics([]layouter_dto.FontEntry{testCase.entry})

			require.Error(t, err)
			assert.Nil(t, metrics)
			assert.Contains(t, err.Error(), testCase.wantMessage)
		})
	}
}

func TestNewFontRecord(t *testing.T) {
	t.Parallel()

	t.Run("static face captures metadata and metrics", func(t *testing.T) {
		t.Parallel()

		face := parseTestFace(t, fonts.NotoSansRegularTTF)
		entry := layouter_dto.FontEntry{
			Family: fonts.NotoSansFamilyName,
			Data:   fonts.NotoSansRegularTTF,
			Style:  int(layouter_domain.FontStyleItalic),
		}

		record := newFontRecord(face, entry, 450)

		assert.Same(t, face.Font, record.font)
		assert.Equal(t, fonts.NotoSansFamilyName, record.family)
		assert.Equal(t, 450, record.weight)
		assert.Equal(t, layouter_domain.FontStyleItalic, record.style)
		assert.Equal(t, face.Upem(), record.unitsPerEm)
		assert.True(t, record.hasExtents)
		assert.Positive(t, record.extents.Ascender)
		assert.Negative(t, record.extents.Descender)
		assert.Positive(t, record.capHeight)
		assert.Positive(t, record.xHeight)
		assert.Empty(t, record.coords)
		assert.Equal(t, fonts.NotoSansRegularTTF, record.data)
	})

	t.Run("variable face captures its variation coordinates", func(t *testing.T) {
		t.Parallel()

		face := parseTestFace(t, fonts.NotoSansVariableTTF)
		face.SetVariations([]font.Variation{{Tag: mustTag("wght"), Value: 700}})

		record := newFontRecord(face, variableNotoEntry(), 700)

		require.NotEmpty(t, record.coords)
		assert.Equal(t, face.Coords(), record.coords)

		face.SetVariations([]font.Variation{{Tag: mustTag("wght"), Value: 100}})
		assert.NotEqual(t, face.Coords(), record.coords, "the record must keep its own copy of the coordinates")
	})
}

func TestNewFontKey(t *testing.T) {
	t.Parallel()

	record := &fontRecord{family: "Serif", weight: 300, style: layouter_domain.FontStyleItalic}

	key := newFontKey(record)

	assert.Equal(t, fontKey{family: "Serif", weight: 300, style: layouter_domain.FontStyleItalic}, key)
	assert.Equal(t, layouter_domain.FontDescriptor{Family: "Serif", Weight: 300, Style: layouter_domain.FontStyleItalic}, record.descriptor())
}

func TestMustTag(t *testing.T) {
	t.Parallel()

	assert.Equal(t, font.Tag(0x77676874), mustTag("wght"))
}

func TestGoTextFontMetrics_ShapeAndMeasureText_MatchesSeparateCalls(t *testing.T) {
	t.Parallel()

	metrics := newTestMetrics(t, append(staticNotoEntries(), variableNotoEntry())...)

	testCases := []struct {
		name       string
		text       string
		descriptor layouter_domain.FontDescriptor
		direction  layouter_domain.DirectionType
	}{
		{name: "latin regular", descriptor: layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 400}, text: "Hello, office!"},
		{name: "latin bold", descriptor: layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 700}, text: "AVATAR Wa To"},
		{name: "variable weight", descriptor: layouter_domain.FontDescriptor{Family: testVariableFamily, Weight: 300}, text: "Variable"},
		{name: "right to left", descriptor: layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 400}, text: "abc def", direction: layouter_domain.DirectionRTL},
		{name: "unknown family falls back", descriptor: layouter_domain.FontDescriptor{Family: "Missing", Weight: 400}, text: "fallback"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			glyphs, width := metrics.ShapeAndMeasureText(testCase.descriptor, 12, testCase.text, testCase.direction)

			assert.Equal(t, metrics.ShapeText(testCase.descriptor, 12, testCase.text, testCase.direction), glyphs)
			assert.Equal(t, metrics.MeasureText(testCase.descriptor, 12, testCase.text, testCase.direction), width)

			advances := 0.0
			for _, glyph := range glyphs {
				advances += glyph.XAdvance
			}
			assert.InDelta(t, width, advances, 1e-9)
			assert.Positive(t, width)
		})
	}
}

func TestGoTextFontMetrics_ShapeAndMeasureText_EmptyText(t *testing.T) {
	t.Parallel()

	metrics := newTestMetrics(t, staticNotoEntries()...)

	glyphs, width := metrics.ShapeAndMeasureText(layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 400}, 12, "", layouter_domain.DirectionLTR)

	assert.Nil(t, glyphs)
	assert.Zero(t, width)
}

func TestGoTextFontMetrics_ShapeAndMeasureText_NoFonts(t *testing.T) {
	t.Parallel()

	metrics := newTestMetrics(t)

	glyphs, width := metrics.ShapeAndMeasureText(layouter_domain.FontDescriptor{Family: "Any"}, 10, "ab", layouter_domain.DirectionLTR)

	require.Len(t, glyphs, 2)
	assert.Equal(t, uint16('a'), glyphs[0].GlyphID)
	assert.InDelta(t, 2*10*fallbackAdvanceFraction, width, 1e-9)
}

func TestGoTextFontMetrics_VariableWeightsShapeIndependentlyOfOrder(t *testing.T) {
	t.Parallel()

	thin := layouter_domain.FontDescriptor{Family: testVariableFamily, Weight: 100}
	black := layouter_domain.FontDescriptor{Family: testVariableFamily, Weight: 900}
	text := "Weights differ"

	thinFirst := newTestMetrics(t, variableNotoEntry())
	thinWidth := thinFirst.MeasureText(thin, 12, text, layouter_domain.DirectionLTR)
	blackWidth := thinFirst.MeasureText(black, 12, text, layouter_domain.DirectionLTR)

	blackFirst := newTestMetrics(t, variableNotoEntry())
	blackWidthAgain := blackFirst.MeasureText(black, 12, text, layouter_domain.DirectionLTR)
	thinWidthAgain := blackFirst.MeasureText(thin, 12, text, layouter_domain.DirectionLTR)

	assert.Greater(t, blackWidth, thinWidth, "a heavier instance should be wider")
	assert.Equal(t, thinWidth, thinWidthAgain)
	assert.Equal(t, blackWidth, blackWidthAgain)
}

func TestGoTextFontMetrics_ConcurrentShapingMatchesSequential(t *testing.T) {
	t.Parallel()

	metrics := newTestMetrics(t, append(staticNotoEntries(), variableNotoEntry())...)

	type request struct {
		text       string
		descriptor layouter_domain.FontDescriptor
	}
	requests := []request{
		{descriptor: layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 400}, text: "The quick brown fox"},
		{descriptor: layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 700}, text: "jumps over"},
		{descriptor: layouter_domain.FontDescriptor{Family: testVariableFamily, Weight: 200}, text: "the lazy dog"},
		{descriptor: layouter_domain.FontDescriptor{Family: testVariableFamily, Weight: 800}, text: "office ffi fl"},
	}

	type result struct {
		glyphs []layouter_domain.GlyphPosition
		width  float64
	}
	expected := make([]result, len(requests))
	for index, req := range requests {
		glyphs, width := metrics.ShapeAndMeasureText(req.descriptor, 11, req.text, layouter_domain.DirectionLTR)
		expected[index] = result{glyphs: glyphs, width: width}
	}

	const workers = 16
	const iterations = 25
	var mismatches atomic.Int64
	var waitGroup sync.WaitGroup
	for worker := range workers {
		waitGroup.Go(func() {
			for iteration := range iterations {
				index := (worker + iteration) % len(requests)
				req := requests[index]
				glyphs, width := metrics.ShapeAndMeasureText(req.descriptor, 11, req.text, layouter_domain.DirectionLTR)
				if width != expected[index].width || !assert.ObjectsAreEqual(expected[index].glyphs, glyphs) {
					mismatches.Add(1)
				}
				metrics.GetMetrics(req.descriptor, 11)
				metrics.ResolveFallback(req.descriptor, devanagariKa)
				if face := metrics.GetFontFace(req.descriptor); face != nil {
					face.HorizontalAdvance(1)
				}
			}
		})
	}
	waitGroup.Wait()

	assert.Zero(t, mismatches.Load(), "concurrent shaping must match sequential shaping")
}

func TestGoTextFontMetrics_GetMetrics(t *testing.T) {
	t.Parallel()

	metrics := newTestMetrics(t, staticNotoEntries()...)
	descriptor := layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 400}

	small := metrics.GetMetrics(descriptor, 10)
	large := metrics.GetMetrics(descriptor, 20)

	assert.Equal(t, 1000, small.UnitsPerEm)
	assert.Positive(t, small.Ascent)
	assert.Positive(t, small.Descent)
	assert.Positive(t, small.CapHeight)
	assert.Positive(t, small.XHeight)
	assert.InDelta(t, 2*small.Ascent, large.Ascent, 1e-9)
	assert.InDelta(t, 2*small.Descent, large.Descent, 1e-9)
	assert.InDelta(t, 2*small.XHeight, large.XHeight, 1e-9)
}

func TestGoTextFontMetrics_GetMetrics_WithoutExtents(t *testing.T) {
	t.Parallel()

	metrics := &GoTextFontMetrics{
		fonts: map[fontKey]*fontRecord{
			{family: "Bare", weight: 400}: {family: "Bare", weight: 400, unitsPerEm: 2048, hasExtents: false},
		},
		sessions: sync.Pool{},
		fallback: nil,
	}

	result := metrics.GetMetrics(layouter_domain.FontDescriptor{Family: "Bare", Weight: 400}, 10)

	assert.InDelta(t, 10*defaultAscentFraction, result.Ascent, 1e-9)
	assert.InDelta(t, 10*defaultDescentFraction, result.Descent, 1e-9)
	assert.Equal(t, 2048, result.UnitsPerEm)
}

func TestGoTextFontMetrics_ResolveFallback(t *testing.T) {
	t.Parallel()

	metrics := newTestMetrics(t, append(staticNotoEntries(), variableNotoEntry())...)

	testCases := []struct {
		name      string
		requested layouter_domain.FontDescriptor
		want      layouter_domain.FontDescriptor
		character rune
	}{
		{
			name:      "primary font covers the character",
			requested: layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 700},
			character: 'a',
			want:      layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 700},
		},
		{
			name:      "covering font with the requested weight is preferred",
			requested: layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 700},
			character: devanagariKa,
			want:      layouter_domain.FontDescriptor{Family: testVariableFamily, Weight: 700},
		},
		{
			name:      "first covering font in registration order otherwise",
			requested: layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 400, Style: layouter_domain.FontStyleItalic},
			character: devanagariKa,
			want:      layouter_domain.FontDescriptor{Family: testVariableFamily, Weight: 100},
		},
		{
			name:      "no covering font returns the request",
			requested: layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 400},
			character: '日',
			want:      layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 400},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			for range 5 {
				assert.Equal(t, testCase.want, metrics.ResolveFallback(testCase.requested, testCase.character))
			}
		})
	}
}

func TestGoTextFontMetrics_ResolveFontIsDeterministic(t *testing.T) {
	t.Parallel()

	entries := []layouter_dto.FontEntry{
		{Family: "Second", Weight: 700, Data: fonts.NotoSansBoldTTF},
		{Family: "First", Weight: 700, Data: fonts.NotoSansBoldTTF},
		{Family: "First", Weight: 400, Data: fonts.NotoSansRegularTTF},
	}
	metrics := newTestMetrics(t, entries...)

	testCases := []struct {
		name      string
		requested layouter_domain.FontDescriptor
		want      fontKey
	}{
		{
			name:      "exact match",
			requested: layouter_domain.FontDescriptor{Family: "First", Weight: 700},
			want:      fontKey{family: "First", weight: 700},
		},
		{
			name:      "style falls back to normal",
			requested: layouter_domain.FontDescriptor{Family: "First", Weight: 700, Style: layouter_domain.FontStyleItalic},
			want:      fontKey{family: "First", weight: 700},
		},
		{
			name:      "weight falls back to regular",
			requested: layouter_domain.FontDescriptor{Family: "First", Weight: 300},
			want:      fontKey{family: "First", weight: 400},
		},
		{
			name:      "unknown family takes the first registered font of that weight",
			requested: layouter_domain.FontDescriptor{Family: "Unknown", Weight: 700},
			want:      fontKey{family: "Second", weight: 700},
		},
		{
			name:      "unknown family and style takes the first normal font of that weight",
			requested: layouter_domain.FontDescriptor{Family: "Unknown", Weight: 400, Style: layouter_domain.FontStyleItalic},
			want:      fontKey{family: "First", weight: 400},
		},
		{
			name:      "nothing similar takes the first registered font",
			requested: layouter_domain.FontDescriptor{Family: "Unknown", Weight: 100},
			want:      fontKey{family: "Second", weight: 700},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			for range 5 {
				record := metrics.resolveFont(testCase.requested)
				require.NotNil(t, record)
				assert.Equal(t, testCase.want, newFontKey(record))
			}
		})
	}
}

func TestGoTextFontMetrics_GetFontFaceAndData(t *testing.T) {
	t.Parallel()

	metrics := newTestMetrics(t, append(staticNotoEntries(), variableNotoEntry())...)
	bold := layouter_domain.FontDescriptor{Family: testVariableFamily, Weight: 700}

	first := metrics.GetFontFace(bold)
	second := metrics.GetFontFace(bold)

	require.NotNil(t, first)
	require.NotNil(t, second)
	assert.NotSame(t, first, second, "each caller receives its own face")
	assert.Equal(t, first.Coords(), second.Coords())
	assert.NotEmpty(t, first.Coords())

	data, found := metrics.GetFontData(layouter_domain.FontDescriptor{Family: fonts.NotoSansFamilyName, Weight: 400})
	require.True(t, found)
	assert.Equal(t, fonts.NotoSansRegularTTF, data)
}

func TestGoTextFontMetrics_AcquireSession(t *testing.T) {
	t.Parallel()

	metrics := newTestMetrics(t, staticNotoEntries()...)
	metrics.sessions.Put("not a session")

	for range 3 {
		session := metrics.acquireSession()
		require.NotNil(t, session)
		assert.NotNil(t, session.shapers)
	}

	empty := &GoTextFontMetrics{fonts: nil, sessions: sync.Pool{}, fallback: nil}
	assert.NotNil(t, empty.acquireSession())
}

func TestShapingSession_ReusesRecordState(t *testing.T) {
	t.Parallel()

	metrics := newTestMetrics(t, staticNotoEntries()...)
	record := metrics.fallback[0]
	session := newShapingSession()

	first := session.shaperFor(record)
	second := session.shaperFor(record)

	assert.Same(t, first, second)
	assert.Len(t, session.shapers, 1)
}

func TestConvertGlyphs(t *testing.T) {
	t.Parallel()

	glyphs := []shaping.Glyph{
		{GlyphID: 42, XAdvance: 128, Advance: 128, XOffset: 64, YOffset: -64},
		{GlyphID: font.GID(70000)},
	}

	positions := convertGlyphs(glyphs)

	require.Len(t, positions, 2)
	assert.Equal(t, uint16(42), positions[0].GlyphID)
	assert.InDelta(t, 2*layouter_domain.PixelsToPoints, positions[0].XAdvance, 1e-9)
	assert.InDelta(t, layouter_domain.PixelsToPoints, positions[0].XOffset, 1e-9)
	assert.InDelta(t, -layouter_domain.PixelsToPoints, positions[0].YOffset, 1e-9)
	assert.Zero(t, positions[1].GlyphID, "glyph IDs beyond 16 bits are not representable")
}

func TestDetectScriptAndLanguage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		text       string
		wantScript string
	}{
		{name: "latin text", text: "hello", wantScript: "Latn"},
		{name: "common characters default to latin", text: "123 !?", wantScript: "Latn"},
		{name: "arabic text", text: "مرحبا", wantScript: "Arab"},
		{name: "leading punctuation is skipped", text: "«שלום", wantScript: "Hebr"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			script, language := detectScriptAndLanguage([]rune(testCase.text))

			assert.Equal(t, testCase.wantScript, script.String())
			assert.NotEmpty(t, string(language))
		})
	}
}
