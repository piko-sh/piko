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
	"bytes"
	"context"
	"strings"
	"testing"

	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

func newTestPainter(entries []layouter_dto.FontEntry) *PdfPainter {
	return NewPdfPainter(595, 842, entries, nil)
}

func TestResolveFontData_ExactMatch(t *testing.T) {
	bold := []byte("bold-font-data")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: []byte("regular")},
		{Family: "NotoSans", Weight: 700, Style: 0, Data: bold},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 0)

	if !resolved.found {
		t.Fatal("expected font to be found")
	}
	if string(resolved.data) != string(bold) {
		t.Errorf("expected bold font data, got %q", string(resolved.data))
	}
}

func TestResolveFontData_FallsBackToNormalStyle(t *testing.T) {
	regular := []byte("regular-data")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 700, Style: 0, Data: regular},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 1)

	if !resolved.found {
		t.Fatal("expected font to be found via style fallback")
	}
	if string(resolved.data) != string(regular) {
		t.Errorf("expected regular style fallback, got %q", string(resolved.data))
	}
}

func TestResolveFontData_FallsBackToWeight400(t *testing.T) {
	regular := []byte("regular-data")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: regular},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 0)

	if !resolved.found {
		t.Fatal("expected font to be found via weight fallback")
	}
	if string(resolved.data) != string(regular) {
		t.Errorf("expected weight 400 fallback, got %q", string(resolved.data))
	}
}

func TestResolveFontData_FallsBackToWeight400AndNormalStyle(t *testing.T) {
	regular := []byte("regular-data")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: regular},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 1)

	if !resolved.found {
		t.Fatal("expected font to be found via weight+style fallback")
	}
	if string(resolved.data) != string(regular) {
		t.Errorf("expected weight+style fallback, got %q", string(resolved.data))
	}
}

func TestResolveFontData_FallsBackToFirstRegistered(t *testing.T) {
	first := []byte("first-font")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "Roboto", Weight: 400, Style: 0, Data: first},
	})

	resolved := painter.resolveFontData("NotoSans", 400, 0)

	if !resolved.found {
		t.Fatal("expected font to be found via first-font fallback")
	}
	if string(resolved.data) != string(first) {
		t.Errorf("expected first registered font, got %q", string(resolved.data))
	}
}

func TestResolveFontData_NoFontsRegistered(t *testing.T) {
	painter := newTestPainter(nil)

	resolved := painter.resolveFontData("NotoSans", 400, 0)

	if resolved.found {
		t.Error("expected no font to be found when none registered")
	}
}

func TestResolveFontData_PrefersSameWeightOverSameStyle(t *testing.T) {
	bold_italic := []byte("bold-italic")
	regular := []byte("regular")
	painter := newTestPainter([]layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 700, Style: 1, Data: bold_italic},
		{Family: "NotoSans", Weight: 400, Style: 0, Data: regular},
	})

	resolved := painter.resolveFontData("NotoSans", 700, 0)

	if !resolved.found {
		t.Fatal("expected font to be found")
	}

	if string(resolved.data) != string(regular) {
		t.Errorf("expected regular fallback, got %q", string(resolved.data))
	}
}

func TestNeedsSyntheticBold(t *testing.T) {
	if !needsSyntheticBold(
		pdfFontKey{weight: 700, style: 0},
		pdfFontKey{weight: 400, style: 0},
	) {
		t.Error("expected synthetic bold when requesting 700 but resolved 400")
	}
	if needsSyntheticBold(
		pdfFontKey{weight: 700, style: 0},
		pdfFontKey{weight: 700, style: 0},
	) {
		t.Error("should not need synthetic bold when resolved weight matches")
	}
	if needsSyntheticBold(
		pdfFontKey{weight: 400, style: 0},
		pdfFontKey{weight: 400, style: 0},
	) {
		t.Error("should not need synthetic bold for normal weight")
	}
}

func TestNeedsSyntheticItalic(t *testing.T) {
	if !needsSyntheticItalic(
		pdfFontKey{weight: 400, style: int(layouter_domain.FontStyleItalic)},
		pdfFontKey{weight: 400, style: 0},
	) {
		t.Error("expected synthetic italic when requesting italic but resolved normal")
	}
	if needsSyntheticItalic(
		pdfFontKey{weight: 400, style: int(layouter_domain.FontStyleItalic)},
		pdfFontKey{weight: 400, style: int(layouter_domain.FontStyleItalic)},
	) {
		t.Error("should not need synthetic italic when resolved style matches")
	}
	if needsSyntheticItalic(
		pdfFontKey{weight: 400, style: 0},
		pdfFontKey{weight: 400, style: 0},
	) {
		t.Error("should not need synthetic italic for normal style")
	}
}

func TestNewPdfPainter_RegistersMultipleWeights(t *testing.T) {
	entries := []layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: []byte("regular")},
		{Family: "NotoSans", Weight: 700, Style: 0, Data: []byte("bold")},
		{Family: "NotoSans", Weight: 400, Style: 1, Data: []byte("italic")},
	}
	painter := newTestPainter(entries)

	if len(painter.fontDataMap) != 3 {
		t.Errorf("expected 3 font entries, got %d", len(painter.fontDataMap))
	}
}

func TestBuildInfoDictionary_DefaultProducerOnly(t *testing.T) {
	painter := newTestPainter(nil)

	result := painter.buildInfoDictionary()

	if result != "<< /Producer (Piko) >>" {
		t.Errorf("expected default info dict, got %q", result)
	}
}

func TestBuildInfoDictionary_WithAllMetadata(t *testing.T) {
	painter := newTestPainter(nil)
	painter.setMetadata(&PdfMetadata{
		Title:    "My Document",
		Author:   "Jane Doe",
		Subject:  "Test Subject",
		Keywords: "pdf, test",
		Creator:  "Test Suite",
	})

	result := painter.buildInfoDictionary()

	for _, expected := range []string{
		"/Producer (Piko)",
		"/Title (My Document)",
		"/Author (Jane Doe)",
		"/Subject (Test Subject)",
		"/Keywords (pdf, test)",
		"/Creator (Test Suite)",
	} {
		if !strings.Contains(result, expected) {
			t.Errorf("expected info dict to contain %q, got %q", expected, result)
		}
	}
}

func TestBuildInfoDictionary_EscapesParentheses(t *testing.T) {
	painter := newTestPainter(nil)
	painter.setMetadata(&PdfMetadata{
		Title: "Title (with parens)",
	})

	result := painter.buildInfoDictionary()

	if !strings.Contains(result, `/Title (Title \(with parens\))`) {
		t.Errorf("expected escaped parentheses, got %q", result)
	}
}

func TestBuildInfoDictionary_SkipsEmptyFields(t *testing.T) {
	painter := newTestPainter(nil)
	painter.setMetadata(&PdfMetadata{
		Title: "Only Title",
	})

	result := painter.buildInfoDictionary()

	if strings.Contains(result, "/Author") {
		t.Errorf("expected no /Author when empty, got %q", result)
	}
	if !strings.Contains(result, "/Title (Only Title)") {
		t.Errorf("expected /Title, got %q", result)
	}
}

func newTestBox() *layouter_domain.LayoutBox {
	return &layouter_domain.LayoutBox{
		ContentX:      20,
		ContentY:      30,
		ContentWidth:  100,
		ContentHeight: 50,
		Padding:       layouter_domain.BoxEdges{Top: 5, Right: 10, Bottom: 5, Left: 10},
		Border:        layouter_domain.BoxEdges{Top: 2, Right: 2, Bottom: 2, Left: 2},
	}
}

func TestBackgroundBox_BorderBox(t *testing.T) {
	box := newTestBox()
	x, y, w, h := backgroundBox(box, "border-box")

	if x != box.BorderBoxX() || y != box.BorderBoxY() {
		t.Errorf("border-box position: got (%v, %v), want (%v, %v)", x, y, box.BorderBoxX(), box.BorderBoxY())
	}
	if w != box.BorderBoxWidth() || h != box.BorderBoxHeight() {
		t.Errorf("border-box size: got (%v, %v), want (%v, %v)", w, h, box.BorderBoxWidth(), box.BorderBoxHeight())
	}
}

func TestBackgroundBox_PaddingBox(t *testing.T) {
	box := newTestBox()
	x, y, w, h := backgroundBox(box, "padding-box")

	expected_x := box.ContentX - box.Padding.Left
	expected_y := box.ContentY - box.Padding.Top
	expected_w := box.ContentWidth + box.Padding.Horizontal()
	expected_h := box.ContentHeight + box.Padding.Vertical()

	if x != expected_x || y != expected_y {
		t.Errorf("padding-box position: got (%v, %v), want (%v, %v)", x, y, expected_x, expected_y)
	}
	if w != expected_w || h != expected_h {
		t.Errorf("padding-box size: got (%v, %v), want (%v, %v)", w, h, expected_w, expected_h)
	}
}

func TestBackgroundBox_ContentBox(t *testing.T) {
	box := newTestBox()
	x, y, w, h := backgroundBox(box, "content-box")

	if x != 20 || y != 30 || w != 100 || h != 50 {
		t.Errorf("content-box: got (%v, %v, %v, %v), want (20, 30, 100, 50)", x, y, w, h)
	}
}

func TestBackgroundBox_DefaultIsBorderBox(t *testing.T) {
	box := newTestBox()
	x1, y1, w1, h1 := backgroundBox(box, "")
	x2, y2, w2, h2 := backgroundBox(box, "border-box")

	if x1 != x2 || y1 != y2 || w1 != w2 || h1 != h2 {
		t.Error("empty string should default to border-box")
	}
}

func TestPaint_SingleEmptyPage(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	rootBox := newLayoutBox().
		WithContentRect(0, 0, 595, 842).
		WithBoxType(layouter_domain.BoxBlock).
		Build()

	result := &layouter_dto.LayoutResult{
		RootBox: rootBox,
		Pages:   []layouter_dto.PageOutput{{Index: 0}},
	}

	var buf bytes.Buffer
	err := painter.Paint(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Paint failed: %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(output, "%PDF-1.7") {
		t.Errorf("output should start with %%PDF-1.7, got %q", output[:min(len(output), 20)])
	}
	if !strings.HasSuffix(strings.TrimSpace(output), "%%EOF") {
		t.Error("output should end with the PDF end-of-file marker")
	}
}

func TestPaint_SinglePageWithBackground(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	rootBox := newLayoutBox().
		WithContentRect(0, 0, 595, 842).
		WithBoxType(layouter_domain.BoxBlock).
		WithBackground(testColour(1.0, 0.0, 0.0, 1.0)).
		Build()

	result := &layouter_dto.LayoutResult{
		RootBox: rootBox,
		Pages:   []layouter_dto.PageOutput{{Index: 0}},
	}

	var buf bytes.Buffer
	err := painter.Paint(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Paint failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "/FlateDecode") {
		t.Error("expected /FlateDecode in output (content stream with background)")
	}
}

func TestPaint_MultiplePages(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	child0 := newLayoutBox().
		WithContentRect(0, 0, 595, 842).
		WithBoxType(layouter_domain.BoxBlock).
		WithPageIndex(0).
		Build()
	child1 := newLayoutBox().
		WithContentRect(0, 842, 595, 842).
		WithBoxType(layouter_domain.BoxBlock).
		WithPageIndex(1).
		Build()

	rootBox := newLayoutBox().
		WithContentRect(0, 0, 595, 1684).
		WithBoxType(layouter_domain.BoxBlock).
		WithChildren(child0, child1).
		Build()

	result := &layouter_dto.LayoutResult{
		RootBox: rootBox,
		Pages: []layouter_dto.PageOutput{
			{Index: 0},
			{Index: 1},
		},
	}

	var buf bytes.Buffer
	err := painter.Paint(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Paint failed: %v", err)
	}

	output := buf.String()
	typePageCount := strings.Count(output, "/Type /Page\n")

	pageCount := strings.Count(output, "/Type /Page ")
	if typePageCount == 0 && pageCount == 0 {

		allPageRefs := strings.Count(output, "/Type /Page")

		if allPageRefs < 3 {
			t.Errorf("expected at least 2 page objects, found %d /Type /Page references", allPageRefs)
		}
	}
}

func TestPaint_InvalidRootBox(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	result := &layouter_dto.LayoutResult{
		RootBox: "not a layout box",
		Pages:   []layouter_dto.PageOutput{{Index: 0}},
	}

	var buf bytes.Buffer
	err := painter.Paint(context.Background(), result, &buf)
	if err == nil {
		t.Fatal("expected error for non-LayoutBox root")
	}
	if !strings.Contains(err.Error(), "not *LayoutBox") {
		t.Errorf("expected error about LayoutBox type, got %q", err.Error())
	}
}

func TestPaint_WithMetadata(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.setMetadata(&PdfMetadata{
		Title:  "Test Document",
		Author: "Test Author",
	})

	rootBox := newLayoutBox().
		WithContentRect(0, 0, 595, 842).
		WithBoxType(layouter_domain.BoxBlock).
		Build()

	result := &layouter_dto.LayoutResult{
		RootBox: rootBox,
		Pages:   []layouter_dto.PageOutput{{Index: 0}},
	}

	var buf bytes.Buffer
	err := painter.Paint(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Paint failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/Producer") {
		t.Error("expected /Producer in output")
	}
	if !strings.Contains(output, "/Title") {
		t.Error("expected /Title in output")
	}
}

func TestPaint_NoPages_DefaultsToOnePage(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	rootBox := newLayoutBox().
		WithContentRect(0, 0, 595, 842).
		WithBoxType(layouter_domain.BoxBlock).
		Build()

	result := &layouter_dto.LayoutResult{
		RootBox: rootBox,
		Pages:   nil,
	}

	var buf bytes.Buffer
	err := painter.Paint(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Paint failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/Count 1") {
		t.Error("expected /Count 1 for single default page")
	}
}

func TestConfigurePainter_AllOptions(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	metadata := &PdfMetadata{
		Title:    "Test",
		Author:   "Author",
		Subject:  "Subject",
		Keywords: "kw1, kw2",
		Creator:  "Creator",
	}
	viewerPrefs := &ViewerPreferences{
		PageLayout:  "SinglePage",
		HideToolbar: true,
	}
	pageLabels := []PageLabelRange{
		{PageIndex: 0, Style: LabelDecimal, Start: 1},
	}
	watermark := &WatermarkConfig{
		Text:     "DRAFT",
		FontSize: 72,
	}
	pdfaConfig := &PdfAConfig{Level: PdfA2B}

	ConfigurePainter(painter, PainterConfig{
		Metadata:    metadata,
		ViewerPrefs: viewerPrefs,
		PageLabels:  pageLabels,
		Watermark:   watermark,
		PdfAConfig:  pdfaConfig,
		Tagged:      true,
	})

	if painter.metadata != metadata {
		t.Error("metadata not propagated")
	}
	if painter.viewerPrefs != viewerPrefs {
		t.Error("viewer preferences not propagated")
	}
	if len(painter.pageLabels) != 1 {
		t.Errorf("page labels: got %d, want 1", len(painter.pageLabels))
	}
	if painter.watermark == nil || painter.watermark.Text != "DRAFT" {
		t.Error("watermark not propagated")
	}
	if painter.pdfaConfig != pdfaConfig {
		t.Error("PDF/A config not propagated")
	}
	if painter.structTree == nil {
		t.Error("tagged PDF should enable struct tree")
	}
}

func TestConfigurePainter_Minimal(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	ConfigurePainter(painter, PainterConfig{})

	if painter.metadata != nil {
		t.Error("metadata should remain nil")
	}
	if painter.viewerPrefs != nil {
		t.Error("viewer prefs should remain nil")
	}
	if painter.watermark != nil {
		t.Error("watermark should remain nil")
	}
	if painter.structTree != nil {
		t.Error("struct tree should remain nil")
	}
	if painter.pdfaConfig != nil {
		t.Error("PDF/A config should remain nil")
	}
}

func TestConfigurePainter_PdfA2A_EnablesTagging(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	ConfigurePainter(painter, PainterConfig{
		PdfAConfig: &PdfAConfig{Level: PdfA2A},
	})

	if painter.structTree == nil {
		t.Error("PDF/A-2a should automatically enable tagged PDF")
	}
}

func TestConfigurePainter_GlyphWidthFunc(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	fn := func(_ string, _ int, _ int, _ uint16) int { return 600 }

	ConfigurePainter(painter, PainterConfig{
		GlyphWidthFunc: fn,
	})

	if painter.glyphWidthFunc == nil {
		t.Error("glyph width function should be set")
	}
	if painter.glyphWidthFunc("test", 400, 0, 0) != 600 {
		t.Error("glyph width function should return the expected value")
	}
}

func TestMarkVariableFont(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	painter.MarkVariableFont("NotoSans", 400, 0)
	painter.MarkVariableFont("NotoSans", 700, 0)

	if len(painter.variableFonts) != 2 {
		t.Errorf("expected 2 variable font entries, got %d", len(painter.variableFonts))
	}
	key := pdfFontKey{family: "NotoSans", weight: 400, style: 0}
	if !painter.variableFonts[key] {
		t.Error("expected NotoSans/400/0 to be marked as variable")
	}
}

func TestPaint_WithWatermark(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.setWatermarkConfig(WatermarkConfig{
		Text:     "CONFIDENTIAL",
		FontSize: 60,
	})

	rootBox := newLayoutBox().
		WithContentRect(0, 0, 595, 842).
		WithBoxType(layouter_domain.BoxBlock).
		Build()

	result := &layouter_dto.LayoutResult{
		RootBox: rootBox,
		Pages:   []layouter_dto.PageOutput{{Index: 0}},
	}

	var buf bytes.Buffer
	err := painter.Paint(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Paint failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "/BaseFont /Helvetica") {
		t.Error("expected Helvetica font object for watermark")
	}
}

func TestPaint_WithTaggedPDF(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.enableTaggedPDF()

	textChild := newLayoutBox().
		WithContentRect(10, 10, 100, 20).
		WithBoxType(layouter_domain.BoxTextRun).
		WithText("Hello").
		WithFontStyle("Helvetica", 400, 0, 12).
		WithPageIndex(0).
		WithSourceNode(testSourceNode("p")).
		Build()

	rootBox := newLayoutBox().
		WithContentRect(0, 0, 595, 842).
		WithBoxType(layouter_domain.BoxBlock).
		WithSourceNode(testSourceNode("div")).
		WithChildren(textChild).
		Build()

	result := &layouter_dto.LayoutResult{
		RootBox: rootBox,
		Pages:   []layouter_dto.PageOutput{{Index: 0}},
	}

	var buf bytes.Buffer
	err := painter.Paint(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Paint failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "/MarkInfo") {
		t.Error("expected /MarkInfo in tagged PDF output")
	}
}

type mockSVGWriter struct{}

func (*mockSVGWriter) RenderSVG(_ context.Context, _ string, _ SVGRenderContext, _, _, _, _ float64) error {
	return nil
}

type mockSVGData struct{}

func (m *mockSVGData) GetSVGData(_ context.Context, _ string) (string, bool) {
	return "<svg></svg>", true
}

func TestSetSVGWriter_SetsFields(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	writer := &mockSVGWriter{}
	data := &mockSVGData{}

	painter.setSVGWriter(writer, data)

	if painter.svgWriter == nil {
		t.Error("expected svgWriter to be set")
	}
	if painter.svgData == nil {
		t.Error("expected svgData to be set")
	}
}

func TestIsVariableFont_ReturnsFalseWhenNilMap(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	key := pdfFontKey{family: "NotoSans", weight: 400, style: 0}

	if painter.isVariableFont(key) {
		t.Error("expected false when variableFonts map is nil")
	}
}

func TestIsVariableFont_ReturnsTrueWhenMarked(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.MarkVariableFont("NotoSans", 400, 0)
	key := pdfFontKey{family: "NotoSans", weight: 400, style: 0}

	if !painter.isVariableFont(key) {
		t.Error("expected true for marked variable font")
	}
}

func TestIsVariableFont_ReturnsFalseForUnmarkedKey(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.MarkVariableFont("NotoSans", 400, 0)
	differentKey := pdfFontKey{family: "NotoSans", weight: 700, style: 0}

	if painter.isVariableFont(differentKey) {
		t.Error("expected false for unmarked key")
	}
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

	if width != 999 {
		t.Errorf("expected glyphWidthFunc result 999, got %d", width)
	}
}

func TestGlyphAdvanceWidth_FallsBackToGlyphAdvanceWidth(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	key := pdfFontKey{family: "TestFont", weight: 400, style: 0}

	width := painter.glyphAdvanceWidth(key, nil, 1)

	if width != 0 {
		t.Errorf("expected 0 for nil font data fallback, got %d", width)
	}
}

func TestFontInstanceKey_FormatsCorrectly(t *testing.T) {
	t.Parallel()

	key := pdfFontKey{family: "NotoSans", weight: 700, style: 1}
	result := fontInstanceKey(key)

	expected := "NotoSans:700:1"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSetFillColour_Grey(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	colour := layouter_domain.Colour{
		Red:   0.5,
		Alpha: 1,
		Space: layouter_domain.ColourSpaceGrey,
	}

	painter.setFillColour(stream, colour)
	output := stream.String()
	if !strings.Contains(output, "g") {
		t.Errorf("expected grey fill colour operator, got %q", output)
	}
}

func TestSetFillColour_CMYK(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	colour := layouter_domain.Colour{
		Cyan:    0.1,
		Magenta: 0.2,
		Yellow:  0.3,
		Key:     0.4,
		Alpha:   1,
		Space:   layouter_domain.ColourSpaceCMYK,
	}

	painter.setFillColour(stream, colour)
	output := stream.String()
	if !strings.Contains(output, "k") {
		t.Errorf("expected CMYK fill colour operator, got %q", output)
	}
}

func TestSetStrokeColour_Grey(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	colour := layouter_domain.Colour{
		Red:   0.7,
		Alpha: 1,
		Space: layouter_domain.ColourSpaceGrey,
	}

	painter.setStrokeColour(stream, colour)
	output := stream.String()
	if !strings.Contains(output, "G") {
		t.Errorf("expected grey stroke colour operator, got %q", output)
	}
}

func TestSetStrokeColour_CMYK(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	colour := layouter_domain.Colour{
		Cyan:    0.5,
		Magenta: 0.5,
		Yellow:  0.5,
		Key:     0.5,
		Alpha:   1,
		Space:   layouter_domain.ColourSpaceCMYK,
	}

	painter.setStrokeColour(stream, colour)
	output := stream.String()
	if !strings.Contains(output, "K") {
		t.Errorf("expected CMYK stroke colour operator, got %q", output)
	}
}

func TestSetStrokeColour_RGB(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	colour := layouter_domain.Colour{
		Red:   1,
		Green: 0,
		Blue:  0,
		Alpha: 1,
	}

	painter.setStrokeColour(stream, colour)
	output := stream.String()
	if !strings.Contains(output, "RG") {
		t.Errorf("expected RGB stroke colour operator, got %q", output)
	}
}

func TestApplyFilterOpacity_WithOpacityFilter(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		Build()
	box.Style.Filter = []layouter_domain.FilterValue{
		{Function: layouter_domain.FilterOpacity, Amount: 0.5},
	}

	result := painter.applyFilterOpacity(stream, box)
	if !result {
		t.Error("expected true for opacity filter with amount < 1.0")
	}

	output := stream.String()
	if !strings.Contains(output, "gs") {
		t.Error("expected ExtGState (gs) operator for filter opacity")
	}
}

func TestApplyFilterOpacity_NoFilter(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		Build()

	result := painter.applyFilterOpacity(stream, box)
	if result {
		t.Error("expected false for no filter")
	}
}

func TestApplyClipPath_Circle(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 100).
		WithPadding(0, 0, 0, 0).
		WithBorder(0, 0, 0, 0).
		Build()
	box.Style.ClipPath = "circle(50%)"

	result := painter.applyClipPath(stream, box)
	if !result {
		t.Error("expected true for circle clip path")
	}

	output := stream.String()
	if !strings.Contains(output, "W") {
		t.Error("expected clip operator (W) in output")
	}
}

func TestApplyClipPath_None(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 100).
		Build()
	box.Style.ClipPath = "none"

	result := painter.applyClipPath(stream, box)
	if result {
		t.Error("expected false for 'none' clip path")
	}
}

func TestApplyClipPath_Empty(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 100).
		Build()

	result := painter.applyClipPath(stream, box)
	if result {
		t.Error("expected false for empty clip path")
	}
}

func TestApplyTransform_WithTranslate(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithPadding(0, 0, 0, 0).
		WithBorder(0, 0, 0, 0).
		Build()
	box.Style.HasTransform = true
	box.Style.TransformValue = "translate(10, 20)"

	result := painter.applyTransform(stream, box)
	if !result {
		t.Error("expected true for translate transform")
	}

	output := stream.String()
	if !strings.Contains(output, "cm") {
		t.Error("expected ConcatMatrix (cm) operator for transform")
	}
}

func TestApplyTransform_NoTransform(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		Build()
	box.Style.HasTransform = false

	result := painter.applyTransform(stream, box)
	if result {
		t.Error("expected false for no transform")
	}
}

func TestApplyTransform_EmptyValue(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		Build()
	box.Style.HasTransform = true
	box.Style.TransformValue = ""

	result := painter.applyTransform(stream, box)
	if result {
		t.Error("expected false for empty transform value")
	}
}

func TestApplyBoxStates_WithOpacityAndBlendMode(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithOpacity(0.5).
		Build()
	box.Style.MixBlendMode = layouter_domain.BlendModeMultiply

	states := painter.applyBoxStates(stream, box)
	if !states.hasOpacity {
		t.Error("expected hasOpacity to be true")
	}
	if !states.hasBlendMode {
		t.Error("expected hasBlendMode to be true")
	}

	output := stream.String()
	if !strings.Contains(output, "q") {
		t.Error("expected SaveState for opacity/blend mode")
	}
}

func TestApplyBoxStates_WithOverflowHidden(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithOverflow(layouter_domain.OverflowHidden, layouter_domain.OverflowHidden).
		Build()

	states := painter.applyBoxStates(stream, box)
	if !states.hasOverflowClip {
		t.Error("expected hasOverflowClip to be true")
	}
}

func TestRestoreBoxStates_RestoresAllStates(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}

	stream.SaveState()
	stream.SaveState()
	stream.SaveState()
	stream.SaveState()
	stream.SaveState()

	states := boxRenderStates{
		hasOverflowClip: true,
		hasOpacity:      true,
		hasBlendMode:    true,
		hasClipPath:     true,
		hasTransform:    true,
	}

	painter.restoreBoxStates(stream, states)

	output := stream.String()

	qCount := strings.Count(output, "Q\n")
	if qCount < 5 {
		t.Errorf("expected at least 5 RestoreState operators, got %d in output: %q", qCount, output)
	}
}

func TestApplyStructTag_NilStructTree_ReturnsFalse(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithSourceNode(testSourceNode("div")).
		Build()

	result := painter.applyStructTag(stream, box)
	if result {
		t.Error("expected false when struct tree is nil")
	}
}

func TestApplyStructTag_NilSourceNode_ReturnsFalse(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.enableTaggedPDF()
	stream := &ContentStream{}
	box := newLayoutBox().Build()

	result := painter.applyStructTag(stream, box)
	if result {
		t.Error("expected false when source node is nil")
	}
}

func TestWriteAcroformObjects_NoFields_ReturnsZero(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	pageObjNumbers := []int{1}
	pageAnnotRefs := [][]string{{}}

	result := painter.writeAcroformObjects(writer, pageObjNumbers, pageAnnotRefs, 1)
	if result != 0 {
		t.Errorf("expected 0 for no acroform fields, got %d", result)
	}
}

func TestNewPdfPainter_VariableFontEntries_RegisteredMultipleTimes(t *testing.T) {
	t.Parallel()

	entries := []layouter_dto.FontEntry{
		{
			Family:     "TestFont",
			Weight:     400,
			Style:      0,
			Data:       []byte("variable-data"),
			IsVariable: true,
			WeightMin:  100,
			WeightMax:  300,
		},
	}

	painter := NewPdfPainter(595, 842, entries, nil)

	key100 := pdfFontKey{family: "TestFont", weight: 100, style: 0}
	key200 := pdfFontKey{family: "TestFont", weight: 200, style: 0}
	key300 := pdfFontKey{family: "TestFont", weight: 300, style: 0}

	if _, ok := painter.fontDataMap[key100]; !ok {
		t.Error("expected weight 100 to be registered")
	}
	if _, ok := painter.fontDataMap[key200]; !ok {
		t.Error("expected weight 200 to be registered")
	}
	if _, ok := painter.fontDataMap[key300]; !ok {
		t.Error("expected weight 300 to be registered")
	}
}

func TestConfigurePainter_WithSVGWriter(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	svgWriter := &mockSVGWriter{}
	svgData := &mockSVGData{}

	ConfigurePainter(painter, PainterConfig{
		SVGWriter: svgWriter,
		SVGData:   svgData,
	})

	if painter.svgWriter == nil {
		t.Error("expected svgWriter to be configured")
	}
	if painter.svgData == nil {
		t.Error("expected svgData to be configured")
	}
}

func TestConfigurePainter_WithGlyphWidthFunc(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	called := false
	ConfigurePainter(painter, PainterConfig{
		GlyphWidthFunc: func(_ string, _ int, _ int, _ uint16) int {
			called = true
			return 500
		},
	})

	if painter.glyphWidthFunc == nil {
		t.Error("expected glyphWidthFunc to be set")
	}

	result := painter.glyphWidthFunc("test", 400, 0, 1)
	if !called {
		t.Error("expected glyphWidthFunc to be called")
	}
	if result != 500 {
		t.Errorf("expected 500, got %d", result)
	}
}
