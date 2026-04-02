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

func TestNewFontEmbedder(t *testing.T) {
	t.Parallel()

	embedder := NewFontEmbedder()
	if embedder == nil {
		t.Fatal("expected non-nil embedder")
	}
	if embedder.HasFonts() {
		t.Error("new embedder should have no fonts")
	}
}

func TestFontEmbedder_RegisterFont(t *testing.T) {
	t.Parallel()

	t.Run("first registration returns F1", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		if name != "F1" {
			t.Errorf("expected F1, got %s", name)
		}
		if !embedder.HasFonts() {
			t.Error("expected HasFonts() to be true after registration")
		}
	})

	t.Run("second registration returns F2", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		_ = embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		name := embedder.RegisterFont(fonts.NotoSansBoldTTF, "NotoSans:700:0")
		if name != "F2" {
			t.Errorf("expected F2, got %s", name)
		}
	})

	t.Run("duplicate instance key returns existing name", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name1 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		name2 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		if name1 != name2 {
			t.Errorf("expected same name for duplicate key, got %s and %s", name1, name2)
		}
	})

	t.Run("empty instance key uses data length", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name1 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "")

		name2 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "")
		if name1 != name2 {
			t.Errorf("expected same name for same data with empty key, got %s and %s", name1, name2)
		}
	})

	t.Run("different instance keys produce different names", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		name1 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "key-a")
		name2 := embedder.RegisterFont(fonts.NotoSansRegularTTF, "key-b")
		if name1 == name2 {
			t.Errorf("expected different names for different keys, both got %s", name1)
		}
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
		if embedder.HasFonts() {
			t.Error("expected HasFonts() to be false")
		}
	})

	t.Run("true after registration", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
		if !embedder.HasFonts() {
			t.Error("expected HasFonts() to be true")
		}
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

		if !strings.Contains(entries, "/F1") {
			t.Errorf("expected /F1 in entries, got %q", entries)
		}

		if !strings.Contains(output, "/Type /Font") {
			t.Error("expected /Type /Font in output")
		}
		if !strings.Contains(output, "/CIDFontType2") {
			t.Error("expected /CIDFontType2 in output")
		}
		if !strings.Contains(output, "/Type /FontDescriptor") {
			t.Error("expected /Type /FontDescriptor in output")
		}
		if !strings.Contains(output, "/FontFile2") {
			t.Error("expected /FontFile2 reference in output")
		}
		if !strings.Contains(output, "/ToUnicode") {
			t.Error("expected /ToUnicode reference in output")
		}
		if !strings.Contains(output, "/Type0") {
			t.Error("expected /Type0 (composite font) in output")
		}
		if !strings.Contains(output, "/Identity-H") {
			t.Error("expected /Identity-H encoding in output")
		}
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

		if !strings.Contains(entries, "/F1") {
			t.Error("expected /F1 in entries")
		}
		if !strings.Contains(entries, "/F2") {
			t.Error("expected /F2 in entries")
		}
	})

	t.Run("font with no recorded glyphs still writes objects", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()
		embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		entries := embedder.WriteObjects(writer)

		if !strings.Contains(entries, "/F1") {
			t.Errorf("expected /F1 in entries even with no glyphs, got %q", entries)
		}
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

		if !strings.Contains(entries, "/F1") {
			t.Errorf("expected /F1 in entries, got %q", entries)
		}
		output := string(writer.Bytes())

		if !strings.Contains(output, "/W") {
			t.Error("expected /W (width array) in output")
		}
	})

	t.Run("empty embedder produces no entries", func(t *testing.T) {
		t.Parallel()
		embedder := NewFontEmbedder()

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		entries := embedder.WriteObjects(writer)

		if entries != "" {
			t.Errorf("expected empty entries for empty embedder, got %q", entries)
		}
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

		if !strings.Contains(output, "+") {
			t.Error("expected subset tag with '+' separator in PostScript name")
		}
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
			if !strings.Contains(output, field) {
				t.Errorf("FontDescriptor missing required field %s", field)
			}
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

		if !strings.Contains(output, "/ToUnicode") {
			t.Error("expected ToUnicode reference in output")
		}
	})
}

func TestFontEmbedder_FullLifecycle(t *testing.T) {
	t.Parallel()

	embedder := NewFontEmbedder()

	regularName := embedder.RegisterFont(fonts.NotoSansRegularTTF, "NotoSans:400:0")
	boldName := embedder.RegisterFont(fonts.NotoSansBoldTTF, "NotoSans:700:0")

	if regularName == boldName {
		t.Fatalf("expected different resource names, both got %s", regularName)
	}

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

	if !strings.HasPrefix(output, "%PDF-1.7") {
		t.Error("missing PDF header")
	}
	if !strings.Contains(output, "%"+"%EOF") {
		t.Error("missing EOF marker")
	}
	if !strings.Contains(output, "/F1") {
		t.Error("missing F1 font reference")
	}
	if !strings.Contains(output, "/F2") {
		t.Error("missing F2 font reference")
	}
	if !strings.Contains(output, "/CIDFontType2") {
		t.Error("missing CIDFontType2")
	}
	if strings.Count(output, "/Type /FontDescriptor") < 2 {
		t.Error("expected at least 2 FontDescriptor objects for 2 fonts")
	}
}
