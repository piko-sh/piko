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

//go:build !integration

package pdfwriter_domain

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/fonts"
)

const (
	os2FSTypeEmbedding uint16 = 0x0008
)

func TestFontUnitsPerEm(t *testing.T) {
	t.Parallel()

	t.Run("NotoSans regular returns correct unitsPerEm", func(t *testing.T) {
		t.Parallel()
		got := FontUnitsPerEm(fonts.NotoSansRegularTTF)

		assert.EqualValues(t, 1000, got)
	})

	t.Run("empty data returns fallback", func(t *testing.T) {
		t.Parallel()
		got := FontUnitsPerEm(nil)
		assert.EqualValues(t, 1000, got, "expected fallback")
	})

	t.Run("truncated data returns fallback", func(t *testing.T) {
		t.Parallel()
		got := FontUnitsPerEm([]byte{0, 1, 0, 0})
		assert.EqualValues(t, 1000, got, "expected fallback")
	})
}

func TestHasFvarTable(t *testing.T) {
	t.Parallel()

	t.Run("NotoSans regular is not variable", func(t *testing.T) {
		t.Parallel()
		assert.False(t, HasFvarTable(fonts.NotoSansRegularTTF), "expected NotoSans regular to NOT have fvar table")
	})

	t.Run("empty data returns false", func(t *testing.T) {
		t.Parallel()
		assert.False(t, HasFvarTable(nil), "expected nil data to return false")
	})

	t.Run("truncated data returns false", func(t *testing.T) {
		t.Parallel()
		assert.False(t, HasFvarTable([]byte{0, 1, 0, 0, 0, 1}), "expected truncated data to return false")
	})
}

func TestSanitisePostScriptName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "already valid", input: "NotoSans-Regular", want: "NotoSans-Regular"},
		{name: "spaces become hyphens", input: "Noto Sans", want: "Noto-Sans"},
		{name: "brackets removed", input: "Font[1]", want: "Font1"},
		{name: "parentheses removed", input: "Font(Bold)", want: "FontBold"},
		{name: "braces removed", input: "Font{var}", want: "Fontvar"},
		{name: "angle brackets removed", input: "Font<name>", want: "Fontname"},
		{name: "slash removed", input: "Font/Sub", want: "FontSub"},
		{name: "percent removed", input: "Font%20", want: "Font20"},
		{name: "non-ASCII removed", input: "Fonte\u00e9", want: "Fonte"},
		{name: "empty string", input: "", want: ""},
		{name: "all special chars", input: "[](){}<>/%", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := SanitisePostScriptName(test.input)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGenerateSubsetTag(t *testing.T) {
	t.Parallel()

	t.Run("produces 6 uppercase letters", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]rune{
			0:  0,
			36: 'A',
			68: 'e',
		}
		tag := GenerateSubsetTag(glyphs)
		assert.Len(t, tag, 6, "expected 6-character tag")
		for _, c := range tag {
			if c < 'A' || c > 'Z' {
				assert.Failf(t, "non-uppercase character", "expected uppercase letters only, got %c in %q", c, tag)
				break
			}
		}
	})

	t.Run("deterministic for same input", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]rune{0: 0, 42: 'X'}
		tag1 := GenerateSubsetTag(glyphs)
		tag2 := GenerateSubsetTag(glyphs)
		assert.Equal(t, tag1, tag2, "expected deterministic tags")
	})

	t.Run("different glyphs produce different tags", func(t *testing.T) {
		t.Parallel()
		tag1 := GenerateSubsetTag(map[uint16]rune{0: 0, 1: 'A'})
		tag2 := GenerateSubsetTag(map[uint16]rune{0: 0, 999: 'Z'})
		assert.NotEqual(t, tag1, tag2, "expected different tags for different glyph sets")
	})

	t.Run("empty glyph map", func(t *testing.T) {
		t.Parallel()
		tag := GenerateSubsetTag(map[uint16]rune{})
		assert.Len(t, tag, 6, "expected 6-character tag even for empty map")
	})
}

func TestExtractFontDescriptor(t *testing.T) {
	t.Parallel()

	t.Run("NotoSans regular succeeds", func(t *testing.T) {
		t.Parallel()
		info, err := ExtractFontDescriptor(fonts.NotoSansRegularTTF)
		require.NoError(t, err)
		require.NotNil(t, info, "expected non-nil descriptor")
		assert.EqualValues(t, 1000, info.UnitsPerEm)
		assert.NotZero(t, info.Ascent, "expected non-zero Ascent")
		assert.NotZero(t, info.Descent, "expected non-zero Descent")
		assert.NotEqual(t, "", info.PostScriptName)
		assert.NotEqual(t, "Unknown", info.PostScriptName)
	})

	t.Run("nil data returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ExtractFontDescriptor(nil)
		assert.Error(t, err, "expected error for nil data")
	})

	t.Run("truncated data returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ExtractFontDescriptor([]byte{0, 1, 0, 0})
		assert.Error(t, err, "expected error for truncated data")
	})
}

func TestGlyphAdvanceWidth(t *testing.T) {
	t.Parallel()

	t.Run("valid glyph returns non-zero width", func(t *testing.T) {
		t.Parallel()

		width := GlyphAdvanceWidth(fonts.NotoSansRegularTTF, 0)
		assert.NotZero(t, width, "expected non-zero width for glyph 0")
	})

	t.Run("nil data returns zero", func(t *testing.T) {
		t.Parallel()
		width := GlyphAdvanceWidth(nil, 0)
		assert.Zero(t, width, "expected 0 for nil data")
	})
}

func TestBuildToUnicodeCMap(t *testing.T) {
	t.Parallel()

	t.Run("basic mapping", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]string{
			36: "A",
			68: "e",
		}
		cmap := BuildToUnicodeCMap(glyphs)

		assert.Contains(t, cmap, "beginbfchar")
		assert.Contains(t, cmap, "endbfchar")
		assert.Contains(t, cmap, "begincmap")
		assert.Contains(t, cmap, "endcmap")
		assert.Contains(t, cmap, "<0024>", "expected glyph 36 (0x0024) in CMap")
		assert.Contains(t, cmap, "<0044>", "expected glyph 68 (0x0044) in CMap")
	})

	t.Run("skips glyph 0 in bfchar entries", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]string{
			0:  ".notdef",
			36: "A",
		}
		cmap := BuildToUnicodeCMap(glyphs)

		assert.Contains(t, cmap, "1 beginbfchar", "expected exactly 1 bfchar entry (glyph 0 should be excluded)")
	})

	t.Run("empty glyph map", func(t *testing.T) {
		t.Parallel()
		cmap := BuildToUnicodeCMap(map[uint16]string{})
		assert.Contains(t, cmap, "begincmap", "expected begincmap even for empty map")

		assert.NotContains(t, cmap, "beginbfchar", "expected no bfchar entries for empty map")
	})

	t.Run("ligature mapping", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]string{
			100: "fi",
		}
		cmap := BuildToUnicodeCMap(glyphs)
		assert.Contains(t, cmap, "<0064>", "expected glyph 100 (0x0064) in CMap")
		assert.Contains(t, cmap, "<00660069>", "expected ligature decomposed to f+i (<00660069>) in CMap value")
	})

	t.Run("astral codepoint as surrogate pair", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]string{
			200: "\U0001F600",
		}
		cmap := BuildToUnicodeCMap(glyphs)
		assert.Contains(t, cmap, "<D83DDE00>", "expected emoji encoded as UTF-16 surrogate pair")
	})

	t.Run("mixed bmp and astral preserved", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]string{
			201: "x\U0001D400",
		}
		cmap := BuildToUnicodeCMap(glyphs)
		assert.Contains(t, cmap, "<0078D835DC00>", "expected x + astral preserved (<0078D835DC00>)")
	})
}

func TestMissingToUnicodeGlyphs(t *testing.T) {
	t.Parallel()

	t.Run("full coverage reports none", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]string{0: "", 36: "A", 68: "e", 100: "fi"}
		assert.Empty(t, missingToUnicodeGlyphs(glyphs), "expected no missing glyphs")
	})

	t.Run("reports drawn glyphs with no text", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]string{36: "A", 70: "", 50: ""}
		missing := missingToUnicodeGlyphs(glyphs)
		assert.Equal(t, []uint16{50, 70}, missing, "expected sorted [50 70]")
	})
}

func TestDecodeUTF16BE_CombinesSurrogatePairs(t *testing.T) {
	data := []byte{0xD8, 0x3D, 0xDE, 0x00}
	assert.Equal(t, "\U0001F600", decodeUTF16BE(data), "decodeUTF16BE surrogate pair")
	assert.Equal(t, "AB", decodeUTF16BE([]byte{0x00, 0x41, 0x00, 0x42}), "decodeUTF16BE BMP")
}

func TestDeriveFlags_EmbeddingBitDoesNotSetItalic(t *testing.T) {

	os2Data := make([]byte, 96)
	binary.BigEndian.PutUint16(os2Data[os2FSTypeOffset:os2FSTypeEnd], os2FSTypeEmbedding)

	flags := deriveFlags(os2Data)

	require.Zero(t, flags&pdfFlagItalic, "embedding fsType bit must not set the italic flag")
	require.NotZero(t, flags&pdfFlagNonSymbolic, "non-symbolic flag should always be set")
}

func TestDeriveFlags_FSSelectionItalicBitSetsItalic(t *testing.T) {

	os2Data := make([]byte, 96)
	os2Data[os2FSSelectionFieldLen-1] |= os2FSSelectionItalicBit

	flags := deriveFlags(os2Data)

	require.NotZero(t, flags&pdfFlagItalic, "fsSelection italic bit should set the italic flag")
}

func TestFontEmbedder_UnmappedGlyphCount(t *testing.T) {
	embedder := &FontEmbedder{fonts: map[string]*embeddedFontState{
		"F1": {usedGlyphs: map[uint16]string{1: "A", 2: ""}},
		"F2": {usedGlyphs: map[uint16]string{3: "", 0: ""}},
	}}
	assert.Equal(t, 2, embedder.UnmappedGlyphCount())
}
