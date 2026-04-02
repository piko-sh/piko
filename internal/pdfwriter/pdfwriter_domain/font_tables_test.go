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
	"strings"
	"testing"

	"piko.sh/piko/internal/fonts"
)

func TestFontUnitsPerEm(t *testing.T) {
	t.Parallel()

	t.Run("NotoSans regular returns correct unitsPerEm", func(t *testing.T) {
		t.Parallel()
		got := FontUnitsPerEm(fonts.NotoSansRegularTTF)

		if got != 1000 {
			t.Errorf("FontUnitsPerEm(NotoSans) = %d, want 1000", got)
		}
	})

	t.Run("empty data returns fallback", func(t *testing.T) {
		t.Parallel()
		got := FontUnitsPerEm(nil)
		if got != 1000 {
			t.Errorf("FontUnitsPerEm(nil) = %d, want 1000 (fallback)", got)
		}
	})

	t.Run("truncated data returns fallback", func(t *testing.T) {
		t.Parallel()
		got := FontUnitsPerEm([]byte{0, 1, 0, 0})
		if got != 1000 {
			t.Errorf("FontUnitsPerEm(truncated) = %d, want 1000 (fallback)", got)
		}
	})
}

func TestHasFvarTable(t *testing.T) {
	t.Parallel()

	t.Run("NotoSans regular is not variable", func(t *testing.T) {
		t.Parallel()
		if HasFvarTable(fonts.NotoSansRegularTTF) {
			t.Error("expected NotoSans regular to NOT have fvar table")
		}
	})

	t.Run("empty data returns false", func(t *testing.T) {
		t.Parallel()
		if HasFvarTable(nil) {
			t.Error("expected nil data to return false")
		}
	})

	t.Run("truncated data returns false", func(t *testing.T) {
		t.Parallel()
		if HasFvarTable([]byte{0, 1, 0, 0, 0, 1}) {
			t.Error("expected truncated data to return false")
		}
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
			if got != test.want {
				t.Errorf("SanitisePostScriptName(%q) = %q, want %q", test.input, got, test.want)
			}
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
		if len(tag) != 6 {
			t.Errorf("expected 6-character tag, got %d characters: %q", len(tag), tag)
		}
		for _, c := range tag {
			if c < 'A' || c > 'Z' {
				t.Errorf("expected uppercase letters only, got %c in %q", c, tag)
				break
			}
		}
	})

	t.Run("deterministic for same input", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]rune{0: 0, 42: 'X'}
		tag1 := GenerateSubsetTag(glyphs)
		tag2 := GenerateSubsetTag(glyphs)
		if tag1 != tag2 {
			t.Errorf("expected deterministic tags, got %q and %q", tag1, tag2)
		}
	})

	t.Run("different glyphs produce different tags", func(t *testing.T) {
		t.Parallel()
		tag1 := GenerateSubsetTag(map[uint16]rune{0: 0, 1: 'A'})
		tag2 := GenerateSubsetTag(map[uint16]rune{0: 0, 999: 'Z'})
		if tag1 == tag2 {
			t.Errorf("expected different tags for different glyph sets, both got %q", tag1)
		}
	})

	t.Run("empty glyph map", func(t *testing.T) {
		t.Parallel()
		tag := GenerateSubsetTag(map[uint16]rune{})
		if len(tag) != 6 {
			t.Errorf("expected 6-character tag even for empty map, got %q", tag)
		}
	})
}

func TestExtractFontDescriptor(t *testing.T) {
	t.Parallel()

	t.Run("NotoSans regular succeeds", func(t *testing.T) {
		t.Parallel()
		info, err := ExtractFontDescriptor(fonts.NotoSansRegularTTF)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info == nil {
			t.Fatal("expected non-nil descriptor")
		}
		if info.UnitsPerEm != 1000 {
			t.Errorf("UnitsPerEm = %d, want 1000", info.UnitsPerEm)
		}
		if info.Ascent == 0 {
			t.Error("expected non-zero Ascent")
		}
		if info.Descent == 0 {
			t.Error("expected non-zero Descent")
		}
		if info.PostScriptName == "" || info.PostScriptName == "Unknown" {
			t.Errorf("expected a PostScript name, got %q", info.PostScriptName)
		}
	})

	t.Run("nil data returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ExtractFontDescriptor(nil)
		if err == nil {
			t.Error("expected error for nil data")
		}
	})

	t.Run("truncated data returns error", func(t *testing.T) {
		t.Parallel()
		_, err := ExtractFontDescriptor([]byte{0, 1, 0, 0})
		if err == nil {
			t.Error("expected error for truncated data")
		}
	})
}

func TestGlyphAdvanceWidth(t *testing.T) {
	t.Parallel()

	t.Run("valid glyph returns non-zero width", func(t *testing.T) {
		t.Parallel()

		width := GlyphAdvanceWidth(fonts.NotoSansRegularTTF, 0)
		if width == 0 {
			t.Error("expected non-zero width for glyph 0")
		}
	})

	t.Run("nil data returns zero", func(t *testing.T) {
		t.Parallel()
		width := GlyphAdvanceWidth(nil, 0)
		if width != 0 {
			t.Errorf("expected 0 for nil data, got %d", width)
		}
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

		if !strings.Contains(cmap, "beginbfchar") {
			t.Error("expected beginbfchar in CMap")
		}
		if !strings.Contains(cmap, "endbfchar") {
			t.Error("expected endbfchar in CMap")
		}
		if !strings.Contains(cmap, "begincmap") {
			t.Error("expected begincmap in CMap")
		}
		if !strings.Contains(cmap, "endcmap") {
			t.Error("expected endcmap in CMap")
		}
		if !strings.Contains(cmap, "<0024>") {
			t.Error("expected glyph 36 (0x0024) in CMap")
		}
		if !strings.Contains(cmap, "<0044>") {
			t.Error("expected glyph 68 (0x0044) in CMap")
		}
	})

	t.Run("skips glyph 0 in bfchar entries", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]string{
			0:  ".notdef",
			36: "A",
		}
		cmap := BuildToUnicodeCMap(glyphs)

		if !strings.Contains(cmap, "1 beginbfchar") {
			t.Error("expected exactly 1 bfchar entry (glyph 0 should be excluded)")
		}
	})

	t.Run("empty glyph map", func(t *testing.T) {
		t.Parallel()
		cmap := BuildToUnicodeCMap(map[uint16]string{})
		if !strings.Contains(cmap, "begincmap") {
			t.Error("expected begincmap even for empty map")
		}

		if strings.Contains(cmap, "beginbfchar") {
			t.Error("expected no bfchar entries for empty map")
		}
	})

	t.Run("ligature mapping", func(t *testing.T) {
		t.Parallel()
		glyphs := map[uint16]string{
			100: "fi",
		}
		cmap := BuildToUnicodeCMap(glyphs)
		if !strings.Contains(cmap, "<0064>") {
			t.Error("expected glyph 100 (0x0064) in CMap")
		}
	})
}
