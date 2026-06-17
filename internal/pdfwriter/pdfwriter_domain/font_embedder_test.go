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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/fonts"
)

func TestNewFontEmbedder(t *testing.T) {
	t.Parallel()

	embedder := NewFontEmbedder()
	require.NotNil(t, embedder, "expected non-nil embedder")
	assert.False(t, embedder.HasFonts(), "new embedder should have no fonts")
}

func TestFontEmbedder_RegisterFont(t *testing.T) {
	t.Parallel()

	t.Run("first registration returns F1", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		assert.Equal(t, "F1", name)
		assert.True(t, embedder.HasFonts(), "expected HasFonts() to be true after registration")
	})

	t.Run("second registration returns F2", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		_ = embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		name := embedder.RegisterFont(fonts.NotoSansBoldTTF, "NotoSans:700:0")
		assert.Equal(t, "F2", name)
	})

	t.Run("duplicate instance key returns existing name", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name1 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		name2 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		assert.Equal(t, name1, name2, "expected same name for duplicate key")
	})

	t.Run("empty instance key uses data length", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name1 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "")

		name2 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "")
		assert.Equal(t, name1, name2, "expected same name for same data with empty key")
	})

	t.Run("different instance keys produce different names", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name1 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "key-a")
		name2 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "key-b")
		assert.NotEqual(t, name1, name2, "expected different names for different keys")
	})
}

func TestFontEmbedder_RecordGlyph(t *testing.T) {
	t.Parallel()

	t.Run("records glyph for known font", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")

		embedder.RecordGlyph(name, 36, "A")
		embedder.RecordGlyph(name, 68, "e")
	})

	t.Run("ignores unknown font name", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()

		embedder.RecordGlyph("F99", 36, "A")
	})

	t.Run("overwrites duplicate glyph ID", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		embedder.RecordGlyph(name, 36, "A")
		embedder.RecordGlyph(name, 36, "B")

	})
}

func TestFontEmbedder_RecordGlyphWidth(t *testing.T) {
	t.Parallel()

	t.Run("records width for known font", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")

		embedder.RecordGlyphWidth(name, 36, 600)
	})

	t.Run("ignores unknown font name", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()

		embedder.RecordGlyphWidth("F99", 36, 600)
	})

	t.Run("multiple width recordings", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		embedder.RecordGlyphWidth(name, 36, 600)
		embedder.RecordGlyphWidth(name, 68, 550)
		embedder.RecordGlyphWidth(name, 36, 610)
	})
}

func TestFontEmbedder_HasFonts(t *testing.T) {
	t.Parallel()

	t.Run("false when empty", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		assert.False(t, embedder.HasFonts(), "expected HasFonts() to be false")
	})

	t.Run("true after registration", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		assert.True(t, embedder.HasFonts(), "expected HasFonts() to be true")
	})
}

func TestFontEmbedder_WriteObjects(t *testing.T) {
	t.Parallel()

	t.Run("single font lifecycle", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		embedder.RecordGlyph(name, 36, "A")
		embedder.RecordGlyph(name, 68, "e")
		embedder.RecordGlyph(name, 79, "l")

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		entries := embedder.WriteObjects(writer)
		output := string(writer.Bytes())

		assert.Contains(t, entries, "/F1", "expected /F1 in entries")

		assert.Contains(t, output, "/Type /Font", "expected /Type /Font in output")
		assert.Contains(t, output, "/CIDFontType2", "expected /CIDFontType2 in output")
		assert.Contains(t, output, "/Type /FontDescriptor", "expected /Type /FontDescriptor in output")
		assert.Contains(t, output, "/FontFile2", "expected /FontFile2 reference in output")
		assert.Contains(t, output, "/ToUnicode", "expected /ToUnicode reference in output")
		assert.Contains(t, output, "/Type0", "expected /Type0 (composite font) in output")
		assert.Contains(t, output, "/Identity-H", "expected /Identity-H encoding in output")
	})

	t.Run("two fonts produce two resource entries", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name1 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		name2 := embedder.RegisterFont(fonts.NotoSansBoldTTF, "NotoSans:700:0")
		embedder.RecordGlyph(name1, 36, "A")
		embedder.RecordGlyph(name2, 36, "A")

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		entries := embedder.WriteObjects(writer)

		assert.Contains(t, entries, "/F1", "expected /F1 in entries")
		assert.Contains(t, entries, "/F2", "expected /F2 in entries")
	})

	t.Run("font with no recorded glyphs still writes objects", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		entries := embedder.WriteObjects(writer)

		assert.Contains(t, entries, "/F1", "expected /F1 in entries even with no glyphs")
	})

	t.Run("font with width overrides", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		embedder.RecordGlyph(name, 36, "A")
		embedder.RecordGlyphWidth(name, 36, 600)

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		entries := embedder.WriteObjects(writer)

		assert.Contains(t, entries, "/F1", "expected /F1 in entries")
		output := string(writer.Bytes())

		assert.Contains(t, output, "/W", "expected /W (width array) in output")
	})

	t.Run("empty embedder produces no entries", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		entries := embedder.WriteObjects(writer)

		assert.Empty(t, entries, "expected empty entries for empty embedder")
	})

	t.Run("output contains subset tag for static fonts", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		embedder.RecordGlyph(name, 36, "A")

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		embedder.WriteObjects(writer)
		output := string(writer.Bytes())

		assert.Contains(t, output, "+", "expected subset tag with '+' separator in PostScript name")
	})

	t.Run("FontDescriptor contains required fields", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		embedder.RecordGlyph(name, 36, "A")

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		embedder.WriteObjects(writer)
		output := string(writer.Bytes())

		requiredFields := []string{
			"/FontName",
			"/Flags",
			"/FontBBox",
			"/ItalicAngle",
			"/Ascent",
			"/Descent",
			"/CapHeight",
			"/StemV",
			"/FontFile2",
		}
		for _, field := range requiredFields {
			assert.Contains(t, output, field, "FontDescriptor missing required field %s", field)
		}
	})

	t.Run("ToUnicode CMap is present", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		embedder.RecordGlyph(name, 36, "A")
		embedder.RecordGlyph(name, 68, "e")

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		embedder.WriteObjects(writer)
		output := string(writer.Bytes())

		assert.Contains(t, output, "/ToUnicode", "expected ToUnicode reference in output")
	})
}

func TestFontEmbedder_FullLifecycle(t *testing.T) {
	t.Parallel()

	embedder := NewFontEmbedder()

	regularName := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
	boldName := embedder.RegisterFont(fonts.NotoSansBoldTTF, "NotoSans:700:0")

	require.NotEqual(t, regularName, boldName, "expected different resource names")

	embedder.RecordGlyph(regularName, 36, "A")
	embedder.RecordGlyph(regularName, 68, "e")
	embedder.RecordGlyph(regularName, 79, "l")
	embedder.RecordGlyph(regularName, 82, "o")

	embedder.RecordGlyph(boldName, 36, "A")
	embedder.RecordGlyph(boldName, 79, "l")

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	pagesNum := writer.AllocateObject()
	catNum := writer.AllocateObject()
	pageNum := writer.AllocateObject()
	contentNum := writer.AllocateObject()

	fontEntries := embedder.WriteObjects(writer)

	writer.WriteObject(catNum,
		"<< /Type /Catalog /Pages "+FormatReference(pagesNum)+" >>")
	writer.WriteObject(pagesNum,
		"<< /Type /Pages /Kids ["+FormatReference(pageNum)+"] /Count 1 >>")
	writer.WriteStreamObject(contentNum, "", []byte("BT /F1 12 Tf (Hello) Tj ET"))
	writer.WriteObject(pageNum,
		"<< /Type /Page /Parent "+FormatReference(pagesNum)+
			" /Contents "+FormatReference(contentNum)+
			" /Resources << /Font <<"+fontEntries+" >> >> >>")
	writer.WriteTrailer(catNum)

	output := string(writer.Bytes())

	assert.True(t, strings.HasPrefix(output, "%PDF-1.7"), "missing PDF header")
	assert.Contains(t, output, "%"+"%EOF", "missing EOF marker")
	assert.Contains(t, output, "/F1", "missing F1 font reference")
	assert.Contains(t, output, "/F2", "missing F2 font reference")
	assert.Contains(t, output, "/CIDFontType2", "missing CIDFontType2")
	assert.GreaterOrEqual(t, strings.Count(output, "/Type /FontDescriptor"), 2, "expected at least 2 FontDescriptor objects for 2 fonts")
}
