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

	"piko.sh/piko/internal/layouter/layouter_domain"
)

func TestPaintText_FallbackFont(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("Hello World").
		WithFontStyle("Helvetica", 400, 0, 12).
		Build()

	painter.paintText(&stream, box)

	requireStreamContains(t, &stream, "/F1 12 Tf")
	requireStreamContains(t, &stream, "BT")
	requireStreamContains(t, &stream, "ET")
	requireStreamContains(t, &stream, "Tj")
	requireStreamContains(t, &stream, "rg")
}

func TestPaintText_EmptyTextSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("").
		WithFontStyle("Helvetica", 400, 0, 12).
		Build()

	painter.paintText(&stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream for empty text, got %q", got)
	}
}

func TestPaintText_NonTextBoxSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithBoxType(layouter_domain.BoxBlock).
		Build()
	box.Text = "Should not be painted"

	painter.paintText(&stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream for non-text box, got %q", got)
	}
}

func TestPaintText_ListMarkerIsPainted(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 20, 20).
		WithBoxType(layouter_domain.BoxListMarker).
		WithFontStyle("Helvetica", 400, 0, 12).
		Build()
	box.Text = "1."

	painter.paintText(&stream, box)

	requireStreamContains(t, &stream, "BT")
	requireStreamContains(t, &stream, "ET")
}

func TestPaintText_SyntheticBold(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("Bold Text").
		WithFontStyle("Helvetica", 700, 0, 12).
		Build()

	painter.paintText(&stream, box)

	requireStreamContains(t, &stream, "2 Tr")
	requireStreamContains(t, &stream, "0.30 w")
	requireStreamContains(t, &stream, "RG")
}

func TestPaintText_SyntheticItalic(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("Italic Text").
		WithFontStyle("Helvetica", 400, int(layouter_domain.FontStyleItalic), 12).
		Build()

	painter.paintText(&stream, box)

	requireStreamContains(t, &stream, "Tm")

	requireStreamContains(t, &stream, "0.21")
}

func TestPaintTextDecorations_Underline(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("Underlined").
		WithFontStyle("Helvetica", 400, 0, 14).
		Build()
	box.Style.TextDecoration = layouter_domain.TextDecorationUnderline

	painter.paintTextDecorations(&stream, box)

	requireStreamContains(t, &stream, "m")
	requireStreamContains(t, &stream, "l")
	requireStreamContains(t, &stream, "S")
	requireStreamContains(t, &stream, "1 w")
}

func TestPaintTextDecorations_LineThrough(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("Struck Through").
		WithFontStyle("Helvetica", 400, 0, 14).
		Build()
	box.Style.TextDecoration = layouter_domain.TextDecorationLineThrough

	painter.paintTextDecorations(&stream, box)

	requireStreamContains(t, &stream, "m")
	requireStreamContains(t, &stream, "l")
	requireStreamContains(t, &stream, "S")
}

func TestPaintTextDecorations_Overline(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("Overlined").
		WithFontStyle("Helvetica", 400, 0, 14).
		Build()
	box.Style.TextDecoration = layouter_domain.TextDecorationOverline

	painter.paintTextDecorations(&stream, box)

	requireStreamContains(t, &stream, "m")
	requireStreamContains(t, &stream, "l")
	requireStreamContains(t, &stream, "S")
}

func TestPaintTextDecorations_NoDecorationSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("No decoration").
		WithFontStyle("Helvetica", 400, 0, 14).
		Build()

	painter.paintTextDecorations(&stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream with no decoration, got %q", got)
	}
}

func TestPaintTextDecorations_EmptyTextSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("").
		WithFontStyle("Helvetica", 400, 0, 14).
		Build()
	box.Style.TextDecoration = layouter_domain.TextDecorationUnderline

	painter.paintTextDecorations(&stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream for empty text, got %q", got)
	}
}

func TestPaintTextDecorations_CustomDecorationColour(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithText("Coloured underline").
		WithFontStyle("Helvetica", 400, 0, 14).
		Build()
	box.Style.TextDecoration = layouter_domain.TextDecorationUnderline
	box.Style.TextDecorationColour = testColour(1, 0, 0, 1)
	box.Style.TextDecorationColourSet = true

	painter.paintTextDecorations(&stream, box)

	requireStreamContains(t, &stream, "1 0 0 RG")
}

func TestDrawDecorationLine_Solid(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream

	painter.drawDecorationLine(&stream, 10, 100, 50, 1, layouter_domain.TextDecorationStyleSolid)

	got := stream.String()
	requireStreamContains(t, &stream, "m")
	requireStreamContains(t, &stream, "l")
	requireStreamContains(t, &stream, "S")

	if strings.Contains(got, " d") {
		t.Error("solid decoration should not set dash pattern")
	}
}

func TestDrawDecorationLine_Dashed(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream

	painter.drawDecorationLine(&stream, 10, 100, 50, 1, layouter_domain.TextDecorationStyleDashed)

	requireStreamContains(t, &stream, "d")
	requireStreamContains(t, &stream, "S")
}

func TestDrawDecorationLine_Dotted(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream

	painter.drawDecorationLine(&stream, 10, 100, 50, 1, layouter_domain.TextDecorationStyleDotted)

	requireStreamContains(t, &stream, "1 J")
	requireStreamContains(t, &stream, "d")
	requireStreamContains(t, &stream, "S")
}

func TestDrawDecorationLine_Double(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream

	painter.drawDecorationLine(&stream, 10, 100, 50, 2, layouter_domain.TextDecorationStyleDouble)

	got := stream.String()

	strokeCount := strings.Count(got, "\nS\n") + strings.Count(got, " S\n")
	if strokeCount < 2 {
		t.Errorf("double decoration expected at least 2 strokes, got %d", strokeCount)
	}

	requireStreamContains(t, &stream, "1 w")
}

func TestDrawDecorationLine_Wavy(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream

	painter.drawDecorationLine(&stream, 10, 100, 50, 1, layouter_domain.TextDecorationStyleWavy)

	requireStreamContains(t, &stream, "c")
	requireStreamContains(t, &stream, "S")
}

func TestExtractSrcAttribute_Found(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBoxType(layouter_domain.BoxReplaced).
		WithSourceNode(testSourceNode("img", "src", "photo.jpg", "alt", "Photo")).
		Build()

	got := extractSrcAttribute(box)
	if got != "photo.jpg" {
		t.Errorf("expected photo.jpg, got %q", got)
	}
}

func TestExtractSrcAttribute_NoSrc(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBoxType(layouter_domain.BoxReplaced).
		WithSourceNode(testSourceNode("img", "alt", "Photo")).
		Build()

	got := extractSrcAttribute(box)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestExtractSrcAttribute_NilSourceNode(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBoxType(layouter_domain.BoxReplaced).
		Build()

	got := extractSrcAttribute(box)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestPaintImage_NonReplacedBoxSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithBoxType(layouter_domain.BoxBlock).
		Build()

	painter.paintImage(nil, &stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream for non-replaced box, got %q", got)
	}
}

func TestPaintImage_NilSourceNodeSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithBoxType(layouter_domain.BoxReplaced).
		Build()

	painter.paintImage(nil, &stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream for nil source node, got %q", got)
	}
}

func TestPaintImage_NoSrcAttributeSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithBoxType(layouter_domain.BoxReplaced).
		WithSourceNode(testSourceNode("img", "alt", "no-src")).
		Build()

	painter.paintImage(nil, &stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream without src attribute, got %q", got)
	}
}

func TestEmitWavyLine_ProducesCurves(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	emitWavyLine(&stream, 0, 100, 50, 2, 10)

	requireStreamContains(t, &stream, "m")
	requireStreamContains(t, &stream, "c")
	requireStreamContains(t, &stream, "S")
}

func TestEmitWavyLine_ShortSpan(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	emitWavyLine(&stream, 0, 5, 50, 2, 10)

	requireStreamContains(t, &stream, "m")
	requireStreamContains(t, &stream, "S")
}
