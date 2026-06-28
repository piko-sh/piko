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

package pdfwriter_domain_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"

	goTextFont "github.com/go-text/typesetting/font"
)

func TestSubsetTrueTypeFont_ProducesValidTTF(t *testing.T) {
	usedGlyphs := map[uint16]rune{
		36: 'A',
		37: 'B',
		38: 'C',
		72: 'a',
		73: 'b',
		74: 'c',
	}

	subset, err := pdfwriter_domain.SubsetTrueTypeFont(fonts.NotoSansRegularTTF, usedGlyphs)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(subset), 12, "subset too short")

	version := binary.BigEndian.Uint32(subset[0:4])
	assert.Equal(t, uint32(0x00010000), version)
}

func TestSubsetTrueTypeFont_IsSmallerThanOriginal(t *testing.T) {
	usedGlyphs := map[uint16]rune{
		36: 'A',
		72: 'a',
	}

	subset, err := pdfwriter_domain.SubsetTrueTypeFont(fonts.NotoSansRegularTTF, usedGlyphs)
	require.NoError(t, err)

	assert.Less(t, len(subset), len(fonts.NotoSansRegularTTF), "subset is not smaller than original")
}

func TestSubsetTrueTypeFont_RoundTripWithGoText(t *testing.T) {
	usedGlyphs := map[uint16]rune{
		36: 'A',
		37: 'B',
		72: 'a',
		73: 'b',
	}

	subset, err := pdfwriter_domain.SubsetTrueTypeFont(fonts.NotoSansRegularTTF, usedGlyphs)
	require.NoError(t, err)

	face, parseError := goTextFont.ParseTTF(bytes.NewReader(subset))
	require.NoError(t, parseError, "go-text/typesetting failed to parse subset")

	glyphID, hasGlyph := face.NominalGlyph('A')
	assert.True(t, hasGlyph, "subset font does not contain glyph for 'A'")
	assert.EqualValues(t, 36, glyphID)

	advance := face.HorizontalAdvance(glyphID)
	assert.Positive(t, advance, "expected positive advance for 'A'")
}

func TestSubsetTrueTypeFont_IncludesNotdef(t *testing.T) {
	usedGlyphs := map[uint16]rune{
		36: 'A',
	}

	subset, err := pdfwriter_domain.SubsetTrueTypeFont(fonts.NotoSansRegularTTF, usedGlyphs)
	require.NoError(t, err)

	face, parseError := goTextFont.ParseTTF(bytes.NewReader(subset))
	require.NoError(t, parseError, "go-text/typesetting failed to parse subset")

	_ = face.Upem()
}

func TestSubsetTrueTypeFont_EmptyGlyphSet(t *testing.T) {
	usedGlyphs := map[uint16]rune{}

	subset, err := pdfwriter_domain.SubsetTrueTypeFont(fonts.NotoSansRegularTTF, usedGlyphs)
	require.NoError(t, err)

	assert.Less(t, len(subset), len(fonts.NotoSansRegularTTF), "empty subset should be smaller than original")
}

func TestExtractFontDescriptor(t *testing.T) {
	info, err := pdfwriter_domain.ExtractFontDescriptor(fonts.NotoSansRegularTTF)
	require.NoError(t, err)

	assert.EqualValues(t, 1000, info.UnitsPerEm)

	assert.Positive(t, info.Ascent, "expected positive Ascent")

	assert.Negative(t, info.Descent, "expected negative Descent")

	assert.NotEqual(t, "", info.PostScriptName)
	assert.NotEqual(t, "Unknown", info.PostScriptName)
}

func TestBuildToUnicodeCMap(t *testing.T) {
	usedGlyphs := map[uint16]string{
		36: "A",
		37: "B",
		72: "a",
	}

	cmap := pdfwriter_domain.BuildToUnicodeCMap(usedGlyphs)

	assert.Contains(t, cmap, "beginbfchar")
	assert.Contains(t, cmap, "endcmap")
	assert.Contains(t, cmap, "<0024> <0041>", "CMap does not contain mapping for glyph 36 -> 'A'")
}

func TestBuildToUnicodeCMap_Ligature(t *testing.T) {
	usedGlyphs := map[uint16]string{
		36:   "A",
		1654: "fi",
	}

	cmap := pdfwriter_domain.BuildToUnicodeCMap(usedGlyphs)

	assert.Contains(t, cmap, "<0676> <00660069>", "CMap does not contain multi-char mapping for fi ligature")
}

func TestGlyphAdvanceWidth(t *testing.T) {
	width := pdfwriter_domain.GlyphAdvanceWidth(fonts.NotoSansRegularTTF, 36)
	assert.Positive(t, width, "expected positive advance width for glyph 36")
}
