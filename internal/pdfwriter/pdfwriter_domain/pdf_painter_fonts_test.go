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

package pdfwriter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

func TestResolveFontData_ExactMatch(t *testing.T) {
	bold := []byte("bold-font-data")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: []byte("regular")},
		{Family: "NotoSans", Weight: 700, Style: 0, Data: bold},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 0)

	require.True(t, resolved.found, "expected font to be found")
	assert.Equal(t, string(bold), string(resolved.data), "expected bold font data")
}

func TestResolveFontData_FallsBackToNormalStyle(t *testing.T) {
	regular := []byte("regular-data")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 700, Style: 0, Data: regular},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 1)

	require.True(t, resolved.found, "expected font to be found via style fallback")
	assert.Equal(t, string(regular), string(resolved.data), "expected regular style fallback")
}

func TestResolveFontData_FallsBackToWeight400(t *testing.T) {
	regular := []byte("regular-data")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: regular},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 0)

	require.True(t, resolved.found, "expected font to be found via weight fallback")
	assert.Equal(t, string(regular), string(resolved.data), "expected weight 400 fallback")
}

func TestResolveFontData_FallsBackToWeight400AndNormalStyle(t *testing.T) {
	regular := []byte("regular-data")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: regular},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 1)

	require.True(t, resolved.found, "expected font to be found via weight+style fallback")
	assert.Equal(t, string(regular), string(resolved.data), "expected weight+style fallback")
}

func TestResolveFontData_FallsBackToFirstRegistered(t *testing.T) {
	first := []byte("first-font")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "Roboto", Weight: 400, Style: 0, Data: first},
	})

	resolved := painter.resolveFontData("NotoSans", 400, 0)

	require.True(t, resolved.found, "expected font to be found via first-font fallback")
	assert.Equal(t, string(first), string(resolved.data), "expected first registered font")
}

func TestResolveFontData_NoFontsRegistered(t *testing.T) {
	painter := newTestPainter(nil)

	resolved := painter.resolveFontData("NotoSans", 400, 0)

	assert.False(t, resolved.found, "expected no font to be found when none registered")
}

func TestResolveFontData_PrefersSameWeightOverSameStyle(t *testing.T) {
	bold_italic := []byte("bold-italic")
	regular := []byte("regular")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 700, Style: 1, Data: bold_italic},
		{Family: "NotoSans", Weight: 400, Style: 0, Data: regular},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 0)

	require.True(t, resolved.found, "expected font to be found")

	assert.Equal(t, string(regular), string(resolved.data), "expected regular fallback")
}

func TestNeedsSyntheticBold(t *testing.T) {
	assert.True(t, needsSyntheticBold(
		pdfFontKey{weight: 700, style: 0},
		pdfFontKey{weight: 400, style: 0},
	), "expected synthetic bold when requesting 700 but resolved 400")
	assert.False(t, needsSyntheticBold(
		pdfFontKey{weight: 700, style: 0},
		pdfFontKey{weight: 700, style: 0},
	), "should not need synthetic bold when resolved weight matches")
	assert.False(t, needsSyntheticBold(
		pdfFontKey{weight: 400, style: 0},
		pdfFontKey{weight: 400, style: 0},
	), "should not need synthetic bold for normal weight")
}

func TestNeedsSyntheticItalic(t *testing.T) {
	assert.True(t, needsSyntheticItalic(
		pdfFontKey{weight: 400, style: int(layouter_domain.FontStyleItalic)},
		pdfFontKey{weight: 400, style: 0},
	), "expected synthetic italic when requesting italic but resolved normal")
	assert.False(t, needsSyntheticItalic(
		pdfFontKey{weight: 400, style: int(layouter_domain.FontStyleItalic)},
		pdfFontKey{weight: 400, style: int(layouter_domain.FontStyleItalic)},
	), "should not need synthetic italic when resolved style matches")
	assert.False(t, needsSyntheticItalic(
		pdfFontKey{weight: 400, style: 0},
		pdfFontKey{weight: 400, style: 0},
	), "should not need synthetic italic for normal style")
}

func TestIsVariableFont_ReturnsFalseWhenNilMap(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	key := pdfFontKey{family: "NotoSans", weight: 400, style: 0}

	assert.False(t, painter.isVariableFont(key), "expected false when variableFonts map is nil")
}

func TestIsVariableFont_ReturnsTrueWhenMarked(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.MarkVariableFont("NotoSans", 400, 0)
	key := pdfFontKey{family: "NotoSans", weight: 400, style: 0}

	assert.True(t, painter.isVariableFont(key), "expected true for marked variable font")
}

func TestIsVariableFont_ReturnsFalseForUnmarkedKey(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.MarkVariableFont("NotoSans", 400, 0)
	differentKey := pdfFontKey{family: "NotoSans", weight: 700, style: 0}

	assert.False(t, painter.isVariableFont(differentKey), "expected false for unmarked key")
}

func TestGlyphAdvanceWidth_DelegatesToGlyphWidthFunc(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.MarkVariableFont("TestFont", 400, 0)
	painter.setGlyphWidthFunc(func(family string, weight int, style int, glyphID uint16) int {
		return 999
	})

	key := pdfFontKey{family: "TestFont", weight: 400, style: 0}
	width := painter.glyphAdvanceWidth(key, nil, 1)

	assert.Equal(t, 999, width, "expected glyphWidthFunc result 999")
}

func TestGlyphAdvanceWidth_FallsBackToGlyphAdvanceWidth(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	key := pdfFontKey{family: "TestFont", weight: 400, style: 0}

	width := painter.glyphAdvanceWidth(key, nil, 1)

	assert.Equal(t, 0, width, "expected 0 for nil font data fallback")
}

func TestFontInstanceKey_FormatsCorrectly(t *testing.T) {
	t.Parallel()

	key := pdfFontKey{family: "NotoSans", weight: 700, style: 1}
	result := fontInstanceKey(key)

	expected := "NotoSans:700:1"
	assert.Equal(t, expected, result)
}
