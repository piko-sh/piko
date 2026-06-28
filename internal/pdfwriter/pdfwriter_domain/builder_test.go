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
	"net/http"
	"testing"

	"github.com/go-text/typesetting/font"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	assert.Same(t, builder, result, "fluent methods should return the same builder")
}

func TestRenderBuilder_Do_MissingTemplate(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	_, err := builder.Do(context.Background())
	require.Error(t, err, "expected error when template path is not set")
}

func TestRenderBuilder_WatermarkConfig(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	wm := WatermarkConfig{Text: "CONFIDENTIAL", FontSize: 48, Angle: 30}
	result := builder.WatermarkConfig(wm)

	assert.Same(t, builder, result, "WatermarkConfig should return the same builder")
	require.NotNil(t, builder.watermark, "WatermarkConfig should set watermark")
	assert.Equal(t, "CONFIDENTIAL", builder.watermark.Text, "WatermarkConfig should set watermark")
}

func TestRenderBuilder_PdfA_A2A_EnablesTagged(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	builder.PdfA(PdfA2A)

	assert.True(t, builder.tagged, "PdfA(PdfA2A) should automatically enable tagged PDF")
}

func TestRenderBuilder_DefaultIsTagged(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	assert.True(t, builder.tagged, "NewRender should produce a tagged builder by default")
}

func TestRenderBuilder_UntaggedDisablesTagging(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	builder.Untagged()

	assert.False(t, builder.tagged, "Untagged() should disable tagging")
}

func TestRenderBuilder_PdfA_A2B_DoesNotForceTagged(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	builder.Untagged().PdfA(PdfA2B)

	assert.False(t, builder.tagged, "PdfA(PdfA2B) must not re-enable tagging after Untagged()")
}

func TestRenderBuilder_MultipleStylesheets(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()

	builder.Stylesheet("css1").Stylesheet("css2").Stylesheet("css3")

	assert.Len(t, builder.stylesheets, 3, "expected 3 stylesheets")
}

func TestBuildLayoutConfig_Defaults(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()

	config := builder.buildLayoutConfig()

	assert.Equal(t, layouter_dto.PageA4, config.Page, "default page should be A4")
	assert.Equal(t, builderDefaultFontSize, config.DefaultFontSize, "default font size")
	assert.Equal(t, 0.0, config.DefaultLineHeight, "default line height")
}

func TestBuildLayoutConfig_CustomPage(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.Page(layouter_dto.PageConfig{Width: 800, Height: 600})

	config := builder.buildLayoutConfig()

	assert.Equal(t, 800.0, config.Page.Width, "page width")
	assert.Equal(t, 600.0, config.Page.Height, "page height")
}

func TestBuildLayoutConfig_CustomFontSize(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.FontSize(16.0)

	config := builder.buildLayoutConfig()

	assert.Equal(t, 16.0, config.DefaultFontSize, "font size")
}

func TestBuildLayoutConfig_CustomLineHeight(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.LineHeight(1.5)

	config := builder.buildLayoutConfig()

	assert.Equal(t, 1.5, config.DefaultLineHeight, "line height")
}

func TestBuildLayoutConfig_Stylesheets(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.Stylesheet("body { margin: 0; }")
	builder.Stylesheet("h1 { colour: red; }")

	config := builder.buildLayoutConfig()

	assert.Len(t, config.Stylesheets, 2, "expected 2 stylesheets")
}

func TestSegmentsToContours_MoveToLineTo_SingleContour(t *testing.T) {
	t.Parallel()

	segments := []font.Segment{
		{Op: 0, Args: [3]font.SegmentPoint{{X: 10, Y: 20}, {}, {}}},
		{Op: 1, Args: [3]font.SegmentPoint{{X: 100, Y: 20}, {}, {}}},
		{Op: 1, Args: [3]font.SegmentPoint{{X: 100, Y: 200}, {}, {}}},
	}

	contours := segmentsToContours(segments)

	require.Len(t, contours, 1, "expected 1 contour")
	assert.Len(t, contours[0], 3, "expected 3 points")

	for i, pt := range contours[0] {
		assert.True(t, pt.OnCurve, "point %d should be on-curve", i)
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

	require.Len(t, contours, 2, "expected 2 contours")
	assert.Len(t, contours[0], 2, "contour 0: expected 2 points")
	assert.Len(t, contours[1], 2, "contour 1: expected 2 points")
}

func TestSegmentsToContours_QuadToAddsOffCurveAndOnCurve(t *testing.T) {
	t.Parallel()

	segments := []font.Segment{
		{Op: 0, Args: [3]font.SegmentPoint{{X: 0, Y: 0}, {}, {}}},
		{Op: 2, Args: [3]font.SegmentPoint{{X: 50, Y: 100}, {X: 100, Y: 0}}},
	}

	contours := segmentsToContours(segments)

	require.Len(t, contours, 1, "expected 1 contour")

	require.Len(t, contours[0], 3, "expected 3 points")

	assert.True(t, contours[0][0].OnCurve, "point 0 (MoveTo) should be on-curve")

	assert.False(t, contours[0][1].OnCurve, "point 1 (QuadTo control) should be off-curve")
	assert.Equal(t, float32(50), contours[0][1].X, "point 1 X")
	assert.Equal(t, float32(100), contours[0][1].Y, "point 1 Y")

	assert.True(t, contours[0][2].OnCurve, "point 2 (QuadTo end) should be on-curve")
	assert.Equal(t, float32(100), contours[0][2].X, "point 2 X")
	assert.Equal(t, float32(0), contours[0][2].Y, "point 2 Y")
}

func TestSegmentsToContours_CubeToApproximation(t *testing.T) {
	t.Parallel()

	segments := []font.Segment{
		{Op: 0, Args: [3]font.SegmentPoint{{X: 0, Y: 0}, {}, {}}},
		{Op: 3, Args: [3]font.SegmentPoint{{X: 30, Y: 100}, {X: 70, Y: 100}, {X: 100, Y: 0}}},
	}

	contours := segmentsToContours(segments)

	require.Len(t, contours, 1, "expected 1 contour")

	require.Len(t, contours[0], 3, "expected 3 points")

	offCurve := contours[0][1]
	assert.False(t, offCurve.OnCurve, "CubeTo midpoint should be off-curve")
	assert.Equal(t, float32(50), offCurve.X, "CubeTo midpoint X")
	assert.Equal(t, float32(100), offCurve.Y, "CubeTo midpoint Y")

	onCurve := contours[0][2]
	assert.True(t, onCurve.OnCurve, "CubeTo end should be on-curve")
	assert.Equal(t, float32(100), onCurve.X, "CubeTo end X")
	assert.Equal(t, float32(0), onCurve.Y, "CubeTo end Y")
}

func TestSegmentsToContours_EmptySegments(t *testing.T) {
	t.Parallel()

	contours := segmentsToContours(nil)
	assert.Empty(t, contours, "expected 0 contours for nil segments")
}

func TestRemoveClosingPoint_DuplicateRemoved(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10, Y: 20, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	assert.Len(t, result, 2, "expected 2 points after removing closing duplicate")
}

func TestRemoveClosingPoint_DuplicateWithinTolerance(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10.4, Y: 20.3, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	assert.Len(t, result, 2, "expected 2 points (within tolerance)")
}

func TestRemoveClosingPoint_DifferentPointKept(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 50, Y: 100, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	assert.Len(t, result, 3, "expected 3 points (different closing point)")
}

func TestRemoveClosingPoint_SinglePointUnchanged(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	assert.Len(t, result, 1, "expected 1 point unchanged")
}

func TestRemoveClosingPoint_OffCurveLastNotRemoved(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10, Y: 20, OnCurve: false},
	}

	result := removeClosingPoint(contour)

	assert.Len(t, result, 3, "expected 3 points (off-curve last not removed)")
}

func TestRemoveClosingPoint_OffCurveFirstNotRemoved(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: false},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10, Y: 20, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	assert.Len(t, result, 3, "expected 3 points (off-curve first not removed)")
}

func TestRemoveClosingPoint_BeyondTolerance(t *testing.T) {
	t.Parallel()

	contour := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 100, Y: 200, OnCurve: true},
		{X: 10.6, Y: 20, OnCurve: true},
	}

	result := removeClosingPoint(contour)

	assert.Len(t, result, 3, "expected 3 points (beyond tolerance)")
}

func TestRenderBuilder_Request_SetsFieldAndReturnsBuilder(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	result := builder.Request(req)

	assert.Same(t, builder, result, "Request should return the same builder")
	assert.Same(t, req, builder.request, "Request should set the request field")
}

func TestRenderBuilder_Transformations_SetsFieldsAndReturnsBuilder(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	registry := NewPdfTransformerRegistry()
	config := pdfwriter_dto.TransformConfig{}

	result := builder.Transformations(registry, config)

	assert.Same(t, builder, result, "Transformations should return the same builder")
	assert.Same(t, registry, builder.transformRegistry, "Transformations should set the registry field")
	assert.NotNil(t, builder.transformConfig, "Transformations should set the config field")
}

func TestRenderBuilder_SVGWriter_SetsFieldsAndReturnsBuilder(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()

	result := builder.SVGWriter(nil, nil)

	assert.Same(t, builder, result, "SVGWriter should return the same builder")
}

func TestApplyTransforms_NilRegistryPassthrough(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()

	input := []byte("fake PDF content")
	output, err := builder.applyTransforms(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, string(input), string(output), "nil registry should pass through unchanged")
}

func TestApplyTransforms_EmptyChainPassthrough(t *testing.T) {
	t.Parallel()

	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.transformRegistry = NewPdfTransformerRegistry()
	builder.transformConfig = &pdfwriter_dto.TransformConfig{}

	input := []byte("fake PDF content")
	output, err := builder.applyTransforms(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, string(input), string(output), "empty chain should pass through unchanged")
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

	require.Len(t, contours, 1, "expected 1 contour")

	assert.Len(t, contours[0], 3, "expected 3 points (closing removed)")
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

	assert.Equal(t, 400.0, config.Page.Width, "page width")
	assert.Equal(t, 300.0, config.Page.Height, "page height")
	assert.Equal(t, 20.0, config.DefaultFontSize, "font size")
	assert.InDelta(t, 1.8, config.DefaultLineHeight, 1e-9, "line height")
	assert.Len(t, config.Stylesheets, 1, "stylesheets")
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
	assert.Equal(t, 3, pageCount, "expected page count 3")
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
	assert.Equal(t, 1, pageCount, "expected page count 1 for zero pages")
}

func TestInstanceVariableFonts_StaticFontsPassThrough(t *testing.T) {
	t.Parallel()

	entries := []layouter_dto.FontEntry{
		{Family: "NotoSans", Weight: 400, Style: 0, Data: []byte("static-data"), IsVariable: false},
		{Family: "NotoSans", Weight: 700, Style: 0, Data: []byte("bold-data"), IsVariable: false},
	}

	result, err := instanceVariableFonts(entries, nil)
	require.NoError(t, err)
	require.Len(t, result, 2, "expected 2 static font entries to pass through")
	assert.Equal(t, "static-data", string(result[0].Data), "expected first entry data to be 'static-data'")
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
	require.NoError(t, err)
	assert.Empty(t, result, "expected 0 entries when fontMetrics is nil for variable fonts")
}

func TestGetFontFace_NilFontMetrics_ReturnsNil(t *testing.T) {
	t.Parallel()

	face := getFontFace(nil, layouter_domain.FontDescriptor{
		Family: "NotoSans",
		Weight: 400,
	})
	assert.Nil(t, face, "expected nil face for nil fontMetrics")
}

type stubTemplateRunner struct {
	err     error
	ast     *ast_domain.TemplateAST
	styling string
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
	require.Error(t, err, "expected error for nil AST")
	assert.Contains(t, err.Error(), "nil AST", "expected 'nil AST' in error")
}

func TestNewPdfWriterService_ReturnsService(t *testing.T) {
	t.Parallel()

	service := NewPdfWriterService(nil, nil, nil, nil, nil)
	require.NotNil(t, service, "expected non-nil service")
}

func TestNewPdfWriterService_NewRender_ReturnsBuilder(t *testing.T) {
	t.Parallel()

	service := NewPdfWriterService(nil, nil, nil, nil, nil)
	builder := service.NewRender()
	require.NotNil(t, builder, "expected non-nil builder")
}

func TestRenderBuilder_LangSetsMetadata(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.Lang("en-GB")
	require.NotNil(t, builder.metadata, "expected metadata to be set")
	assert.Equal(t, "en-GB", builder.metadata.Lang, "expected Lang en-GB")
}

func TestRenderBuilder_EmbedDataAppends(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.EmbedData(EmbeddedFile{Name: "a.json"}).EmbedData(EmbeddedFile{Name: "b.json"})
	assert.Len(t, builder.embeddedFiles, 2, "expected 2 embedded files")
}

func TestRenderBuilder_WithEmbeddedDataLimits(t *testing.T) {
	service := &pdfWriterService{}
	builder := service.NewRender()
	builder.WithEmbeddedDataLimits(EmbeddedDataLimits{MaxFiles: 3})
	require.NotNil(t, builder.embeddedLimits, "expected embedded limits to be set")
	assert.Equal(t, 3, builder.embeddedLimits.MaxFiles, "expected MaxFiles 3")
}
