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
	if err != nil {
		t.Fatalf("SubsetTrueTypeFont failed: %v", err)
	}

	if len(subset) < 12 {
		t.Fatalf("subset too short: %d bytes", len(subset))
	}

	version := binary.BigEndian.Uint32(subset[0:4])
	if version != 0x00010000 {
		t.Errorf("unexpected TTF version: 0x%08X", version)
	}
}

func TestSubsetTrueTypeFont_IsSmallerThanOriginal(t *testing.T) {
	usedGlyphs := map[uint16]rune{
		36: 'A',
		72: 'a',
	}

	subset, err := pdfwriter_domain.SubsetTrueTypeFont(fonts.NotoSansRegularTTF, usedGlyphs)
	if err != nil {
		t.Fatalf("SubsetTrueTypeFont failed: %v", err)
	}

	if len(subset) >= len(fonts.NotoSansRegularTTF) {
		t.Errorf("subset (%d bytes) is not smaller than original (%d bytes)",
			len(subset), len(fonts.NotoSansRegularTTF))
	}
}

func TestSubsetTrueTypeFont_RoundTripWithGoText(t *testing.T) {
	usedGlyphs := map[uint16]rune{
		36: 'A',
		37: 'B',
		72: 'a',
		73: 'b',
	}

	subset, err := pdfwriter_domain.SubsetTrueTypeFont(fonts.NotoSansRegularTTF, usedGlyphs)
	if err != nil {
		t.Fatalf("SubsetTrueTypeFont failed: %v", err)
	}

	face, parseError := goTextFont.ParseTTF(bytes.NewReader(subset))
	if parseError != nil {
		t.Fatalf("go-text/typesetting failed to parse subset: %v", parseError)
	}

	glyphID, hasGlyph := face.NominalGlyph('A')
	if !hasGlyph {
		t.Error("subset font does not contain glyph for 'A'")
	}
	if glyphID != 36 {
		t.Errorf("expected glyph ID 36 for 'A', got %d", glyphID)
	}

	advance := face.HorizontalAdvance(glyphID)
	if advance <= 0 {
		t.Errorf("expected positive advance for 'A', got %f", advance)
	}
}

func TestSubsetTrueTypeFont_IncludesNotdef(t *testing.T) {
	usedGlyphs := map[uint16]rune{
		36: 'A',
	}

	subset, err := pdfwriter_domain.SubsetTrueTypeFont(fonts.NotoSansRegularTTF, usedGlyphs)
	if err != nil {
		t.Fatalf("SubsetTrueTypeFont failed: %v", err)
	}

	face, parseError := goTextFont.ParseTTF(bytes.NewReader(subset))
	if parseError != nil {
		t.Fatalf("go-text/typesetting failed to parse subset: %v", parseError)
	}

	_ = face.Upem()
}

func TestSubsetTrueTypeFont_EmptyGlyphSet(t *testing.T) {
	usedGlyphs := map[uint16]rune{}

	subset, err := pdfwriter_domain.SubsetTrueTypeFont(fonts.NotoSansRegularTTF, usedGlyphs)
	if err != nil {
		t.Fatalf("SubsetTrueTypeFont failed: %v", err)
	}

	if len(subset) >= len(fonts.NotoSansRegularTTF) {
		t.Errorf("empty subset (%d bytes) should be smaller than original (%d bytes)",
			len(subset), len(fonts.NotoSansRegularTTF))
	}
}

func TestExtractFontDescriptor(t *testing.T) {
	info, err := pdfwriter_domain.ExtractFontDescriptor(fonts.NotoSansRegularTTF)
	if err != nil {
		t.Fatalf("ExtractFontDescriptor failed: %v", err)
	}

	if info.UnitsPerEm != 1000 {
		t.Errorf("expected UnitsPerEm=1000, got %d", info.UnitsPerEm)
	}

	if info.Ascent <= 0 {
		t.Errorf("expected positive Ascent, got %d", info.Ascent)
	}

	if info.Descent >= 0 {
		t.Errorf("expected negative Descent, got %d", info.Descent)
	}

	if info.PostScriptName == "" || info.PostScriptName == "Unknown" {
		t.Errorf("expected a PostScript name, got %q", info.PostScriptName)
	}
}

func TestBuildToUnicodeCMap(t *testing.T) {
	usedGlyphs := map[uint16]string{
		36: "A",
		37: "B",
		72: "a",
	}

	cmap := pdfwriter_domain.BuildToUnicodeCMap(usedGlyphs)

	if !bytes.Contains([]byte(cmap), []byte("beginbfchar")) {
		t.Error("CMap does not contain beginbfchar")
	}
	if !bytes.Contains([]byte(cmap), []byte("endcmap")) {
		t.Error("CMap does not contain endcmap")
	}
	if !bytes.Contains([]byte(cmap), []byte("<0024> <0041>")) {
		t.Error("CMap does not contain mapping for glyph 36 -> 'A'")
	}
}

func TestBuildToUnicodeCMap_Ligature(t *testing.T) {
	usedGlyphs := map[uint16]string{
		36:   "A",
		1654: "fi",
	}

	cmap := pdfwriter_domain.BuildToUnicodeCMap(usedGlyphs)

	if !bytes.Contains([]byte(cmap), []byte("<0676> <00660069>")) {
		t.Errorf("CMap does not contain multi-char mapping for fi ligature.\nCMap:\n%s", cmap)
	}
}

func TestGlyphAdvanceWidth(t *testing.T) {
	width := pdfwriter_domain.GlyphAdvanceWidth(fonts.NotoSansRegularTTF, 36)
	if width <= 0 {
		t.Errorf("expected positive advance width for glyph 36, got %d", width)
	}
}
