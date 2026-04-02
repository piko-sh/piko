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
	"context"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/go-text/typesetting/font"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

func TestRenderBuilder_FluentChaining(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	result := builder.
		Template("pdfs/test.pk").
		Props(map[string]string{"key": "value"}).
		Metadata(PdfMetadata{Title: "Test"}).
		ViewerPreferences(ViewerPreferences{}).
		PageLabels(PageLabelRange{PageIndex: 0, Style: LabelDecimal, Start: 1}).
		Watermark("DRAFT").
		TaggedPDF().
		PdfA(PdfA2B).
		Stylesheet("body { color: red; }").
		Stylesheet("h1 { font-size: 24pt; }").
		Page(layouter_dto.PageA4).
		FontSize(14.0).
		LineHeight(1.5)

	if result != builder {
		t.Error("fluent methods should return the same builder")
	}
}

func TestRenderBuilder_Do_MissingTemplate(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	_, err := builder.Do(context.Background())
	if err == nil {
		t.Fatal("expected error when template path is not set")
	}
}

func TestRenderBuilder_WatermarkConfig(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	wm := WatermarkConfig{Text: "CONFIDENTIAL", FontSize: 48, Angle: 30}
	result := builder.WatermarkConfig(wm)

	if result != builder {
		t.Error("WatermarkConfig should return the same builder")
	}
	if builder.watermark == nil || builder.watermark.Text != "CONFIDENTIAL" {
		t.Error("WatermarkConfig should set watermark")
	}
}

func TestRenderBuilder_PdfA_A2A_EnablesTagged(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	builder.PdfA(PdfA2A)

	if !builder.tagged {
		t.Error("PdfA(PdfA2A) should automatically enable tagged PDF")
	}
}

func TestRenderBuilder_PdfA_A2B_DoesNotEnableTagged(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	builder.PdfA(PdfA2B)

	if builder.tagged {
		t.Error("PdfA(PdfA2B) should not automatically enable tagged PDF")
	}
}

func TestRenderBuilder_MultipleStylesheets(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	builder.Stylesheet("css1").Stylesheet("css2").Stylesheet("css3")

	if len(builder.stylesheets) != 3 {
		t.Errorf("expected 3 stylesheets, got %d", len(builder.stylesheets))
	}
}

func TestBuildLayoutConfig_Defaults(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()

	config := builder.buildLayoutConfig()

	if config.Page != layouter_dto.PageA4 {
		t.Error("default page should be A4")
	}
	if config.DefaultFontSize != builderDefaultFontSize {
		t.Errorf("default font size: got %v, want %v", config.DefaultFontSize, builderDefaultFontSize)
	}
	if config.DefaultLineHeight != 0 {
		t.Errorf("default line height: got %v, want 0", config.DefaultLineHeight)
	}
}

func TestBuildLayoutConfig_CustomPage(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.Page(layouter_dto.PageConfig{Width: 800, Height: 600})

	config := builder.buildLayoutConfig()

	if config.Page.Width != 800 || config.Page.Height != 600 {
		t.Errorf("page: got (%v, %v), want (800, 600)", config.Page.Width, config.Page.Height)
	}
}

func TestBuildLayoutConfig_CustomFontSize(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.FontSize(16.0)

	config := builder.buildLayoutConfig()

	if config.DefaultFontSize != 16.0 {
		t.Errorf("font size: got %v, want 16.0", config.DefaultFontSize)
	}
}

func TestBuildLayoutConfig_CustomLineHeight(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.LineHeight(1.5)

	config := builder.buildLayoutConfig()

	if config.DefaultLineHeight != 1.5 {
		t.Errorf("line height: got %v, want 1.5", config.DefaultLineHeight)
	}
}

func TestBuildLayoutConfig_Stylesheets(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.Stylesheet("body { margin: 0; }")
	builder.Stylesheet("h1 { colour: red; }")

	config := builder.buildLayoutConfig()

	if len(config.Stylesheets) != 2 {
		t.Errorf("expected 2 stylesheets, got %d", len(config.Stylesheets))
	}
}

func TestSegmentsToContours_MoveToLineTo_SingleContour(t *testing.T) {
	t.Parallel()

	segments := []font.Segment{
		{Op: 0, Args: [3]font.SegmentPoint{{X: 10, Y: 20}, {}, {}}},
		{Op: 1, Args: [3]font.SegmentPoint{{X: 100, Y: 20}, {}, {}}},
		{Op: 1, Args: [3]font.SegmentPoint{{X: 100, Y: 200}, {}, {}}},
	}

	contours := segmentsToContours(segments)

	if len(contours) != 1 {
		t.Fatalf("expected 1 contour, got %d", len(contours))
	}
	if len(contours[0]) != 3 {
		t.Errorf("expected 3 points, got %d", len(contours[0]))
	}

	for i, pt := range contours[0] {
		if !pt.OnCurve {
			t.Errorf("point %d should be on-curve", i)
		}
	}
}

func TestSegmentsToContours_MultipleMoveToCreatesMultipleContours(t *testing.T) {
	t.Parallel()

	segments := []font.Segment{
		{Op: 0, Args: [3]font.SegmentPoint{{X: 0, Y: 0}, {}, {}}},
		{Op: 1, Args: [3]font.SegmentPoint{{X: 100, Y: 0}, {}, {}}},
		{Op: 0, Args: [3]font.SegmentPoint{{X: 200, Y: 0}, {}, {}}},
		{Op: 1, Args: [3]font.SegmentPoint{{X: 300, Y: 0}, {}, {}}},
	}

	contours := segmentsToContours(segments)

	if len(contours) != 2 {
		t.Fatalf("expected 2 contours, got %d", len(contours))
	}
	if len(contours[0]) != 2 {
		t.Errorf("contour 0: expected 2 points, got %d", len(contours[0]))
	}
	if len(contours[1]) != 2 {
		t.Errorf("contour 1: expected 2 points, got %d", len(contours[1]))
	}
}

func TestSegmentsToContours_QuadToAddsOffCurveAndOnCurve(t *testing.T) {
	t.Parallel()

	segments := []font.Segment{
		{Op: 0, Args: [3]font.SegmentPoint{{X: 0, Y: 0}, {}, {}}},
		{Op: 2, Args: [3]font.SegmentPoint{{X: 50, Y: 100}, {X: 100, Y: 0}}},
	}

	contours := segmentsToContours(segments)

	if len(contours) != 1 {
		t.Fatalf("expected 1 contour, got %d", len(contours))
	}

	if len(contours[0]) != 3 {
		t.Fatalf("expected 3 points, got %d", len(contours[0]))
	}

	if !contours[0][0].OnCurve {
		t.Error("point 0 (MoveTo) should be on-curve")
	}

	if contours[0][1].OnCurve {
		t.Error("point 1 (QuadTo control) should be off-curve")
	}
	if contours[0][1].X != 50 || contours[0][1].Y != 100 {
		t.Errorf("point 1: got (%v, %v), want (50, 100)", contours[0][1].X, contours[0][1].Y)
	}

	if !contours[0][2].OnCurve {
		t.Error("point 2 (QuadTo end) should be on-curve")
	}
	if contours[0][2].X != 100 || contours[0][2].Y != 0 {
		t.Errorf("point 2: got (%v, %v), want (100, 0)", contours[0][2].X, contours[0][2].Y)
	}
}

func TestSegmentsToContours_CubeToApproximation(t *testing.T) {
	t.Parallel()

	segments := []font.Segment{
		{Op: 0, Args: [3]font.SegmentPoint{{X: 0, Y: 0}, {}, {}}},
		{Op: 3, Args: [3]font.SegmentPoint{{X: 30, Y: 100}, {X: 70, Y: 100}, {X: 100, Y: 0}}},
	}

	contours := segmentsToContours(segments)

	if len(contours) != 1 {
		t.Fatalf("expected 1 contour, got %d", len(contours))
	}

	if len(contours[0]) != 3 {
		t.Fatalf("expected 3 points, got %d", len(contours[0]))
	}

	offCurve := contours[0][1]
	if offCurve.OnCurve {
		t.Error("CubeTo midpoint should be off-curve")
	}
	if offCurve.X != 50 || offCurve.Y != 100 {
		t.Errorf("CubeTo midpoint: got (%v, %v), want (50, 100)", offCurve.X, offCurve.Y)
	}

	onCurve := contours[0][2]
	if !onCurve.OnCurve {
		t.Error("CubeTo end should be on-curve")
	}
	if onCurve.X != 100 || onCurve.Y != 0 {
		t.Errorf("CubeTo end: got (%v, %v), want (100, 0)", onCurve.X, onCurve.Y)
	}
}

func TestSegmentsToContours_EmptySegments(t *testing.T) {
	t.Parallel()

	contours := segmentsToContours(nil)
	if len(contours) != 0 {
		t.Errorf("expected 0 contours for nil segments, got %d", len(contours))
	}
}

func TestRemoveClosingPoint_DuplicateRemoved(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10, Y: 20, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	if len(result) != 2 {
		t.Errorf("expected 2 points after removing closing duplicate, got %d", len(result))
	}
}

func TestRemoveClosingPoint_DuplicateWithinTolerance(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10.4, Y: 20.3, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	if len(result) != 2 {
		t.Errorf("expected 2 points (within tolerance), got %d", len(result))
	}
}

func TestRemoveClosingPoint_DifferentPointKept(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 50, Y: 100, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	if len(result) != 3 {
		t.Errorf("expected 3 points (different closing point), got %d", len(result))
	}
}

func TestRemoveClosingPoint_SinglePointUnchanged(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	if len(result) != 1 {
		t.Errorf("expected 1 point unchanged, got %d", len(result))
	}
}

func TestRemoveClosingPoint_OffCurveLastNotRemoved(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10, Y: 20, OnCurve: false},
	}

	result := removeClosingPoint(contour)

	if len(result) != 3 {
		t.Errorf("expected 3 points (off-curve last not removed), got %d", len(result))
	}
}

func TestRemoveClosingPoint_OffCurveFirstNotRemoved(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: false},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10, Y: 20, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	if len(result) != 3 {
		t.Errorf("expected 3 points (off-curve first not removed), got %d", len(result))
	}
}

func TestRemoveClosingPoint_BeyondTolerance(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10.6, Y: 20, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	if len(result) != 3 {
		t.Errorf("expected 3 points (beyond tolerance), got %d", len(result))
	}
}

func TestRenderBuilder_Request_SetsFieldAndReturnsBuilder(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	result := builder.Request(req)

	if result != builder {
		t.Error("Request should return the same builder")
	}
	if builder.request != req {
		t.Error("Request should set the request field")
	}
}

func TestRenderBuilder_Transformations_SetsFieldsAndReturnsBuilder(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	registry := NewPdfTransformerRegistry()
	config := pdfwriter_dto.TransformConfig{}

	result := builder.Transformations(registry, config)

	if result != builder {
		t.Error("Transformations should return the same builder")
	}
	if builder.transformRegistry != registry {
		t.Error("Transformations should set the registry field")
	}
	if builder.transformConfig == nil {
		t.Error("Transformations should set the config field")
	}
}

func TestRenderBuilder_SVGWriter_SetsFieldsAndReturnsBuilder(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()

	result := builder.SVGWriter(nil, nil)

	if result != builder {
		t.Error("SVGWriter should return the same builder")
	}
}

func TestApplyTransforms_NilRegistryPassthrough(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()

	input := []byte("fake PDF content")
	output, err := builder.applyTransforms(context.Background(), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(output) != string(input) {
		t.Error("nil registry should pass through unchanged")
	}
}

func TestApplyTransforms_EmptyChainPassthrough(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.transformRegistry = NewPdfTransformerRegistry()
	builder.transformConfig = &pdfwriter_dto.TransformConfig{}

	input := []byte("fake PDF content")
	output, err := builder.applyTransforms(context.Background(), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(output) != string(input) {
		t.Error("empty chain should pass through unchanged")
	}
}

func TestSegmentsToContours_ClosingPointRemoved(t *testing.T) {
	t.Parallel()

	segments := []font.Segment{
		{Op: 0, Args: [3]font.SegmentPoint{{X: 0, Y: 0}, {}, {}}},
		{Op: 1, Args: [3]font.SegmentPoint{{X: 100, Y: 200}, {}, {}}},
		{Op: 1, Args: [3]font.SegmentPoint{{X: 200, Y: 0}, {}, {}}},
		{Op: 1, Args: [3]font.SegmentPoint{{X: 0, Y: 0}, {}, {}}},
	}

	contours := segmentsToContours(segments)

	if len(contours) != 1 {
		t.Fatalf("expected 1 contour, got %d", len(contours))
	}

	if len(contours[0]) != 3 {
		t.Errorf("expected 3 points (closing removed), got %d", len(contours[0]))
	}
}

func TestBuildLayoutConfig_AllOverrides(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.Page(layouter_dto.PageConfig{Width: 400, Height: 300})
	builder.FontSize(20.0)
	builder.LineHeight(1.8)
	builder.Stylesheet("body { margin: 0; }")

	config := builder.buildLayoutConfig()

	if config.Page.Width != 400 || config.Page.Height != 300 {
		t.Errorf("page: got (%v, %v), want (400, 300)", config.Page.Width, config.Page.Height)
	}
	if config.DefaultFontSize != 20.0 {
		t.Errorf("font size: got %v, want 20.0", config.DefaultFontSize)
	}
	if math.Abs(config.DefaultLineHeight-1.8) > 1e-9 {
		t.Errorf("line height: got %v, want 1.8", config.DefaultLineHeight)
	}
	if len(config.Stylesheets) != 1 {
		t.Errorf("stylesheets: got %d, want 1", len(config.Stylesheets))
	}
}

func TestSubstitutePageNumbers_ReturnsCorrectPageCount(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()

	rootBox := newLayoutBox().
		WithContentRect(0, 0, 595, 842).
		Build()

	layoutResult := &layouter_dto.LayoutResult{
		RootBox: rootBox,
		Pages: []layouter_dto.PageOutput{
			{Height: 842},
			{Height: 842},
			{Height: 842},
		},
	}

	pageCount := builder.substitutePageNumbers(layoutResult)
	if pageCount != 3 {
		t.Errorf("expected page count 3, got %d", pageCount)
	}
}

func TestSubstitutePageNumbers_ZeroPages_ReturnsOne(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()

	rootBox := newLayoutBox().Build()
	layoutResult := &layouter_dto.LayoutResult{
		RootBox: rootBox,
		Pages:   nil,
	}

	pageCount := builder.substitutePageNumbers(layoutResult)
	if pageCount != 1 {
		t.Errorf("expected page count 1 for zero pages, got %d", pageCount)
	}
}

func TestInstanceVariableFonts_StaticFontsPassThrough(t *testing.T) {
	t.Parallel()

	entries := []layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: []byte("static-data"), IsVariable: false},
		{Family: "NotoSans", Weight: 700, Style: 0, Data: []byte("bold-data"), IsVariable: false},
	}

	result, err := instanceVariableFonts(entries, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 static font entries to pass through, got %d", len(result))
	}
	if string(result[0].Data) != "static-data" {
		t.Errorf("expected first entry data to be 'static-data', got %q", string(result[0].Data))
	}
}

func TestInstanceVariableFonts_VariableSkippedWithNilFontMetrics(t *testing.T) {
	t.Parallel()

	entries := []layouter_dto.FontEntry{
		{
			Family:     "NotoSans",
			Weight:     400,
			Style:      0,
			Data:       []byte("variable-data"),
			IsVariable: true,
			WeightMin:  100,
			WeightMax:  900,
		},
	}

	result, err := instanceVariableFonts(entries, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries when fontMetrics is nil for variable fonts, got %d", len(result))
	}
}

func TestGetFontFace_NilFontMetrics_ReturnsNil(t *testing.T) {
	t.Parallel()

	face := getFontFace(nil, layouter_domain.FontDescriptor{
		Family: "NotoSans",
		Weight: 400,
	})
	if face != nil {
		t.Error("expected nil face for nil fontMetrics")
	}
}

type stubTemplateRunner struct {
	ast     *ast_domain.TemplateAST
	styling string
	err     error
}

func (s *stubTemplateRunner) RunPdfWithProps(
	_ context.Context,
	_ string,
	_ *http.Request,
	_ any,
) (*ast_domain.TemplateAST, string, error) {
	return s.ast, s.styling, s.err
}

func TestBuilderDo_NilAST(t *testing.T) {
	t.Parallel()

	mockRunner := &stubTemplateRunner{
		ast:     nil,
		styling: "",
		err:     nil,
	}
	service := &pdfWriterService{templateRunner: mockRunner}
	builder := service.NewRender()
	builder.Template("test.pk")

	_, err := builder.Do(context.Background())
	if err == nil {
		t.Fatal("expected error for nil AST")
	}
	if !strings.Contains(err.Error(), "nil AST") {
		t.Errorf("expected 'nil AST' in error, got %q", err.Error())
	}
}

func TestNewPdfWriterService_ReturnsService(t *testing.T) {
	t.Parallel()

	service := NewPdfWriterService(nil, nil, nil, nil, nil)
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewPdfWriterService_NewRender_ReturnsBuilder(t *testing.T) {
	t.Parallel()

	service := NewPdfWriterService(nil, nil, nil, nil, nil)
	builder := service.NewRender()
	if builder == nil {
		t.Fatal("expected non-nil builder")
	}
}
