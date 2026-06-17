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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/wdk/clock"
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

func TestNewPdfPainter_RegistersMultipleWeights(t *testing.T) {
	entries := []layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: []byte("regular")},
		{Family: "NotoSans", Weight: 700, Style: 0, Data: []byte("bold")},
		{Family: "NotoSans", Weight: 400, Style: 1, Data: []byte("italic")},
	}
	painter := newTestPainter(entries)

	assert.Len(t, painter.fontDataMap, 3)
}

func TestBuildInfoDictionary_DefaultProducerOnly(t *testing.T) {
	painter := newTestPainter(nil)

	result := painter.buildInfoDictionary(time.Time{})

	assert.Equal(t, "<< /Producer (Piko) >>", result, "expected default info dict")
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

	result := painter.buildInfoDictionary(time.Time{})

	for _, expected := range []string{
		"/Producer (Piko)",
		"/Title (My Document)",
		"/Author (Jane Doe)",
		"/Subject (Test Subject)",
		"/Keywords (pdf, test)",
		"/Creator (Test Suite)",
	} {
		assert.Contains(t, result, expected, "expected info dict to contain %q", expected)
	}
}

func TestBuildInfoDictionary_EscapesParentheses(t *testing.T) {
	painter := newTestPainter(nil)
	painter.setMetadata(&PdfMetadata{
		Title: "Title (with parens)",
	})

	result := painter.buildInfoDictionary(time.Time{})

	assert.Contains(t, result, `/Title (Title \(with parens\))`, "expected escaped parentheses")
}

func TestBuildInfoDictionary_SkipsEmptyFields(t *testing.T) {
	painter := newTestPainter(nil)
	painter.setMetadata(&PdfMetadata{
		Title: "Only Title",
	})

	result := painter.buildInfoDictionary(time.Time{})

	assert.NotContains(t, result, "/Author", "expected no /Author when empty")
	assert.Contains(t, result, "/Title (Only Title)", "expected /Title")
}

func TestBuildInfoDictionary_NoDatesWhenXMPDisabled(t *testing.T) {
	painter := newTestPainter(nil)

	result := painter.buildInfoDictionary(time.Date(2026, 6, 26, 17, 0, 0, 0, time.UTC))

	assert.NotContains(t, result, "/CreationDate", "expected no dates when XMP disabled")
	assert.NotContains(t, result, "/ModDate", "expected no dates when XMP disabled")
}

func TestBuildInfoDictionary_DatesWhenXMPEnabled(t *testing.T) {
	painter := newTestPainter(nil)
	painter.emitXMP = true

	result := painter.buildInfoDictionary(time.Date(2026, 6, 26, 17, 0, 0, 0, time.UTC))

	for _, expected := range []string{
		"/CreationDate (D:20260626170000+00'00')",
		"/ModDate (D:20260626170000+00'00')",
	} {
		assert.Contains(t, result, expected, "expected info dict to contain %q", expected)
	}
}

func TestResolveCreationTime_UsesInjectedClock(t *testing.T) {
	fixed := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	painter := newTestPainter(nil)
	painter.clock = clock.NewMockClock(fixed)

	got := painter.resolveCreationTime()
	assert.True(t, got.Equal(fixed), "resolveCreationTime = %v, want injected clock time %v", got, fixed)
}

func TestResolveCreationTime_MetadataOverridesClock(t *testing.T) {
	metaTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	painter := newTestPainter(nil)
	painter.clock = clock.NewMockClock(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	painter.setMetadata(&PdfMetadata{CreatedAt: metaTime})

	got := painter.resolveCreationTime()
	assert.True(t, got.Equal(metaTime), "resolveCreationTime = %v, want metadata time %v", got, metaTime)
}

func TestConfigurePainter_SetsClock(t *testing.T) {
	painter := newTestPainter(nil)
	mock := clock.NewMockClock(time.Unix(0, 0))

	ConfigurePainter(painter, PainterConfig{Clock: mock})

	assert.Same(t, mock, painter.clock, "ConfigurePainter should set the injected clock")
}

func TestBuildInfoDictionary_EncodesUnicodeTitle(t *testing.T) {
	painter := newTestPainter(nil)
	painter.setMetadata(&PdfMetadata{Title: "Michael Haddon — CV"})

	result := painter.buildInfoDictionary(time.Time{})

	assert.NotContains(t, result, "\xe2\x80\x94", "expected em dash to be UTF-16BE encoded, not raw UTF-8")
	assert.Contains(t, result, "/Title <FEFF", "expected UTF-16BE title token")
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

	assert.Equal(t, box.BorderBoxX(), x, "border-box position x")
	assert.Equal(t, box.BorderBoxY(), y, "border-box position y")
	assert.Equal(t, box.BorderBoxWidth(), w, "border-box size width")
	assert.Equal(t, box.BorderBoxHeight(), h, "border-box size height")
}

func TestBackgroundBox_PaddingBox(t *testing.T) {
	box := newTestBox()
	x, y, w, h := backgroundBox(box, "padding-box")

	expected_x := box.ContentX - box.Padding.Left
	expected_y := box.ContentY - box.Padding.Top
	expected_w := box.ContentWidth + box.Padding.Horizontal()
	expected_h := box.ContentHeight + box.Padding.Vertical()

	assert.Equal(t, expected_x, x, "padding-box position x")
	assert.Equal(t, expected_y, y, "padding-box position y")
	assert.Equal(t, expected_w, w, "padding-box size width")
	assert.Equal(t, expected_h, h, "padding-box size height")
}

func TestBackgroundBox_ContentBox(t *testing.T) {
	box := newTestBox()
	x, y, w, h := backgroundBox(box, "content-box")

	assert.Equal(t, 20.0, x, "content-box x")
	assert.Equal(t, 30.0, y, "content-box y")
	assert.Equal(t, 100.0, w, "content-box width")
	assert.Equal(t, 50.0, h, "content-box height")
}

func TestBackgroundBox_DefaultIsBorderBox(t *testing.T) {
	box := newTestBox()
	x1, y1, w1, h1 := backgroundBox(box, "")
	x2, y2, w2, h2 := backgroundBox(box, "border-box")

	assert.Equal(t, x2, x1, "empty string should default to border-box (x)")
	assert.Equal(t, y2, y1, "empty string should default to border-box (y)")
	assert.Equal(t, w2, w1, "empty string should default to border-box (w)")
	assert.Equal(t, h2, h1, "empty string should default to border-box (h)")
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
	require.NoError(t, err, "Paint failed")

	output := buf.String()
	assert.True(t, strings.HasPrefix(output, "%PDF-1.7"), "output should start with %%PDF-1.7, got %q", output[:min(len(output), 20)])
	assert.True(t, strings.HasSuffix(strings.TrimSpace(output), "%%EOF"), "output should end with the PDF end-of-file marker")
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
	require.NoError(t, err, "Paint failed")

	output := buf.String()

	assert.Contains(t, output, "/FlateDecode", "expected /FlateDecode in output (content stream with background)")
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
	require.NoError(t, err, "Paint failed")

	output := buf.String()
	typePageCount := strings.Count(output, "/Type /Page\n")

	pageCount := strings.Count(output, "/Type /Page ")
	if typePageCount == 0 && pageCount == 0 {

		allPageRefs := strings.Count(output, "/Type /Page")

		assert.GreaterOrEqual(t, allPageRefs, 3, "expected at least 2 page objects")
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
	require.Error(t, err, "expected error for non-LayoutBox root")
	assert.Contains(t, err.Error(), "not *LayoutBox", "expected error about LayoutBox type")
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
	require.NoError(t, err, "Paint failed")

	output := buf.String()
	assert.Contains(t, output, "/Producer", "expected /Producer in output")
	assert.Contains(t, output, "/Title", "expected /Title in output")
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
	require.NoError(t, err, "Paint failed")

	output := buf.String()
	assert.Contains(t, output, "/Count 1", "expected /Count 1 for single default page")
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

	assert.Same(t, metadata, painter.metadata, "metadata not propagated")
	assert.Same(t, viewerPrefs, painter.viewerPrefs, "viewer preferences not propagated")
	assert.Len(t, painter.pageLabels, 1, "page labels")
	require.NotNil(t, painter.watermark, "watermark not propagated")
	assert.Equal(t, "DRAFT", painter.watermark.Text, "watermark not propagated")
	assert.Same(t, pdfaConfig, painter.pdfaConfig, "PDF/A config not propagated")
	assert.NotNil(t, painter.structTree, "tagged PDF should enable struct tree")
}

func TestConfigurePainter_Minimal(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	ConfigurePainter(painter, PainterConfig{})

	assert.Nil(t, painter.metadata, "metadata should remain nil")
	assert.Nil(t, painter.viewerPrefs, "viewer prefs should remain nil")
	assert.Nil(t, painter.watermark, "watermark should remain nil")
	assert.Nil(t, painter.structTree, "struct tree should remain nil")
	assert.Nil(t, painter.pdfaConfig, "PDF/A config should remain nil")
}

func TestConfigurePainter_PdfA2A_EnablesTagging(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	ConfigurePainter(painter, PainterConfig{
		PdfAConfig: &PdfAConfig{Level: PdfA2A},
	})

	assert.NotNil(t, painter.structTree, "PDF/A-2a should automatically enable tagged PDF")
}

func TestConfigurePainter_GlyphWidthFunc(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	fn := func(_ string, _ int, _ int, _ uint16) int { return 600 }

	ConfigurePainter(painter, PainterConfig{
		GlyphWidthFunc: fn,
	})

	require.NotNil(t, painter.glyphWidthFunc, "glyph width function should be set")
	assert.Equal(t, 600, painter.glyphWidthFunc("test", 400, 0, 0), "glyph width function should return the expected value")
}

func TestMarkVariableFont(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	painter.MarkVariableFont("NotoSans", 400, 0)
	painter.MarkVariableFont("NotoSans", 700, 0)

	assert.Len(t, painter.variableFonts, 2)
	key := pdfFontKey{family: "NotoSans", weight: 400, style: 0}
	assert.True(t, painter.variableFonts[key], "expected NotoSans/400/0 to be marked as variable")
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
	require.NoError(t, err, "Paint failed")

	output := buf.String()

	assert.Contains(t, output, "/BaseFont /Helvetica", "expected Helvetica font object for watermark")
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
	require.NoError(t, err, "Paint failed")

	output := buf.String()

	assert.Contains(t, output, "/MarkInfo", "expected /MarkInfo in tagged PDF output")
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

	assert.NotNil(t, painter.svgWriter, "expected svgWriter to be set")
	assert.NotNil(t, painter.svgData, "expected svgData to be set")
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
	assert.Contains(t, output, "g", "expected grey fill colour operator")
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
	assert.Contains(t, output, "k", "expected CMYK fill colour operator")
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
	assert.Contains(t, output, "G", "expected grey stroke colour operator")
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
	assert.Contains(t, output, "K", "expected CMYK stroke colour operator")
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
	assert.Contains(t, output, "RG", "expected RGB stroke colour operator")
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
	assert.True(t, result, "expected true for opacity filter with amount < 1.0")

	output := stream.String()
	assert.Contains(t, output, "gs", "expected ExtGState (gs) operator for filter opacity")
}

func TestApplyFilterOpacity_NoFilter(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		Build()

	result := painter.applyFilterOpacity(stream, box)
	assert.False(t, result, "expected false for no filter")
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
	assert.True(t, result, "expected true for circle clip path")

	output := stream.String()
	assert.Contains(t, output, "W", "expected clip operator (W) in output")
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
	assert.False(t, result, "expected false for 'none' clip path")
}

func TestApplyClipPath_Empty(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 100).
		Build()

	result := painter.applyClipPath(stream, box)
	assert.False(t, result, "expected false for empty clip path")
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
	assert.True(t, result, "expected true for translate transform")

	output := stream.String()
	assert.Contains(t, output, "cm", "expected ConcatMatrix (cm) operator for transform")
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
	assert.False(t, result, "expected false for no transform")
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
	assert.False(t, result, "expected false for empty transform value")
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
	assert.True(t, states.hasOpacity, "expected hasOpacity to be true")
	assert.True(t, states.hasBlendMode, "expected hasBlendMode to be true")

	output := stream.String()
	assert.Contains(t, output, "q", "expected SaveState for opacity/blend mode")
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
	assert.True(t, states.hasOverflowClip, "expected hasOverflowClip to be true")
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
	assert.GreaterOrEqual(t, qCount, 5, "expected at least 5 RestoreState operators in output: %q", output)
}

func TestApplyStructTag_NilStructTree_ReturnsFalse(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stream := &ContentStream{}
	box := newLayoutBox().
		WithSourceNode(testSourceNode("div")).
		Build()

	result := painter.applyStructTag(stream, box)
	assert.False(t, result, "expected false when struct tree is nil")
}

func TestApplyStructTag_NilSourceNode_ReturnsFalse(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	painter.enableTaggedPDF()
	stream := &ContentStream{}
	box := newLayoutBox().Build()

	result := painter.applyStructTag(stream, box)
	assert.False(t, result, "expected false when source node is nil")
}

func TestWriteAcroformObjects_NoFields_ReturnsZero(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	pageObjNumbers := []int{1}
	pageAnnotRefs := [][]string{{}}

	result, err := painter.writeAcroformObjects(writer, pageObjNumbers, pageAnnotRefs, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, result, "expected 0 for no acroform fields")
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

	assert.Contains(t, painter.fontDataMap, key100, "expected weight 100 to be registered")
	assert.Contains(t, painter.fontDataMap, key200, "expected weight 200 to be registered")
	assert.Contains(t, painter.fontDataMap, key300, "expected weight 300 to be registered")
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

	assert.NotNil(t, painter.svgWriter, "expected svgWriter to be configured")
	assert.NotNil(t, painter.svgData, "expected svgData to be configured")
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

	require.NotNil(t, painter.glyphWidthFunc, "expected glyphWidthFunc to be set")

	result := painter.glyphWidthFunc("test", 400, 0, 1)
	assert.True(t, called, "expected glyphWidthFunc to be called")
	assert.Equal(t, 500, result)
}
