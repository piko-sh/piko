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
	"fmt"
	"math"
	"net/http"

	"github.com/go-text/typesetting/font"

	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// builderDefaultFontSize is the root font size in points when none is specified.
	builderDefaultFontSize = 12.0

	// fontWeightStepSize is the interval between font weight instances
	// when expanding a variable font.
	fontWeightStepSize = 100

	// segmentOpCubeTo is the cubic Bezier segment operation type.
	segmentOpCubeTo = 3

	// closingPointTolerance is the maximum coordinate difference for
	// treating a closing point as a duplicate of the starting point.
	closingPointTolerance = 0.5

	// initialContourCapacity is the initial allocation size for contour
	// point slices (enough for most simple glyph segments).
	initialContourCapacity = 6
)

// RenderBuilder constructs a PDF render operation using a fluent interface.
// Create one via PdfWriterService.NewRender(), configure it with the fluent
// methods, then call Do(ctx) to execute the render pipeline.
type RenderBuilder struct {
	// props holds the template data to pass during rendering.
	props any

	// svgData provides raw SVG markup for image sources.
	svgData SVGDataPort

	// svgWriter renders SVG to PDF drawing commands.
	svgWriter SVGWriterPort

	// pageConfig holds the page dimensions, or nil for A4 default.
	pageConfig *layouter_dto.PageConfig

	// service holds the PDF writer service with shared dependencies.
	service *pdfWriterService

	// viewerPrefs holds PDF viewer preference settings, or nil for defaults.
	viewerPrefs *ViewerPreferences

	// watermark holds the watermark configuration, or nil for no watermark.
	watermark *WatermarkConfig

	// pdfaConfig holds the PDF/A conformance configuration, or nil to skip.
	pdfaConfig *PdfAConfig

	// transformConfig holds the post-processing transform settings, or nil.
	transformConfig *pdfwriter_dto.TransformConfig

	// transformRegistry holds the available PDF transformers, or nil.
	transformRegistry *PdfTransformerRegistry

	// metadata holds the PDF document metadata fields, or nil.
	metadata *PdfMetadata

	// request holds the HTTP request context for template rendering, or nil.
	request *http.Request

	// templatePath is the manifest key or path to the PDF template.
	templatePath string

	// stylesheets holds additional CSS stylesheets to apply during layout.
	stylesheets []string

	// pageLabels holds the page label ranges for the document.
	pageLabels []PageLabelRange

	// fontSize is the root font size in points (0 means use default).
	fontSize float64

	// lineHeight is the root line-height multiplier (0 means use default).
	lineHeight float64

	// tagged enables PDF structure tagging for accessibility.
	tagged bool
}

// Template sets the path to the PDF template to render.
//
// Takes templatePath (string) which is the manifest key or path to the
// PDF template.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) Template(templatePath string) *RenderBuilder {
	b.templatePath = templatePath
	return b
}

// Request sets the HTTP request context for template rendering. In daemon
// mode this provides the incoming request; in standalone mode it may be nil.
//
// Takes r (*http.Request) which provides the HTTP context.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) Request(r *http.Request) *RenderBuilder {
	b.request = r
	return b
}

// Props sets the data to pass to the template during rendering.
//
// Takes props (any) which contains the template data.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) Props(props any) *RenderBuilder {
	b.props = props
	return b
}

// Metadata sets PDF document metadata fields (title, author, subject,
// keywords, creator) that appear in the PDF info dictionary.
//
// Takes m (PdfMetadata) which holds the metadata fields.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) Metadata(m PdfMetadata) *RenderBuilder {
	b.metadata = &m
	return b
}

// ViewerPreferences configures how PDF viewers display the document
// (page layout, toolbar visibility, initial panel, etc.).
//
// Takes vp (ViewerPreferences) which holds the viewer preference fields.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) ViewerPreferences(vp ViewerPreferences) *RenderBuilder {
	b.viewerPrefs = &vp
	return b
}

// PageLabels configures page label ranges for the document.
//
// Each range applies from its PageIndex until the next range starts. For
// example, to use lowercase roman numerals for the first 4 pages then
// decimal from page 5:
//
//	builder.PageLabels(
//	    PageLabelRange{PageIndex: 0, Style: LabelRomanLower, Start: 1},
//	    PageLabelRange{PageIndex: 4, Style: LabelDecimal, Start: 1},
//	)
//
// Takes ranges (...PageLabelRange) which define the label ranges.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) PageLabels(ranges ...PageLabelRange) *RenderBuilder {
	b.pageLabels = ranges
	return b
}

// Watermark sets a diagonal text watermark rendered behind content on
// every page using default styling (60pt, light grey, 45 degrees).
//
// Takes text (string) which is the watermark text.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) Watermark(text string) *RenderBuilder {
	b.watermark = &WatermarkConfig{Text: text}
	return b
}

// WatermarkConfig sets a watermark with full control over styling
// (font size, colour, angle, opacity).
//
// Takes wm (WatermarkConfig) which holds the watermark settings.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) WatermarkConfig(wm WatermarkConfig) *RenderBuilder {
	b.watermark = &wm
	return b
}

// TaggedPDF enables PDF structure tagging for accessibility (PDF/UA).
// When enabled, the painter wraps painted elements in marked content
// sequences and builds a StructTreeRoot with semantic structure tags
// derived from HTML elements.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) TaggedPDF() *RenderBuilder {
	b.tagged = true
	return b
}

// PdfA enables PDF/A conformance output at the specified level.
//
// This adds an sRGB ICC output intent, XMP metadata, and PDF/A identification.
// For PdfA2A, tagged PDF is automatically enabled.
//
// Takes level (PdfALevel) which specifies the conformance level.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) PdfA(level PdfALevel) *RenderBuilder {
	b.pdfaConfig = &PdfAConfig{Level: level}
	if level == PdfA2A {
		b.tagged = true
	}
	return b
}

// Stylesheet adds an additional CSS stylesheet to apply during layout,
// after the user-agent stylesheet and before inline styles. Can be
// called multiple times; stylesheets are applied in order.
//
// Takes css (string) which is the raw CSS text to apply.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) Stylesheet(css string) *RenderBuilder {
	b.stylesheets = append(b.stylesheets, css)
	return b
}

// Transformations configures post-processing transformations to apply
// to the rendered PDF bytes. Transformations execute in priority order
// after the paint step completes.
//
// Takes registry (*PdfTransformerRegistry) which holds the available
// transformers.
// Takes config (pdfwriter_dto.TransformConfig) which specifies which
// transformers to enable and their options.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) Transformations(registry *PdfTransformerRegistry, config pdfwriter_dto.TransformConfig) *RenderBuilder {
	b.transformRegistry = registry
	b.transformConfig = &config
	return b
}

// SVGWriter enables native SVG-to-PDF vector rendering with custom
// writer and data ports. When set, SVG images are rendered as crisp
// vector paths instead of rasterised images.
//
// Takes writer (SVGWriterPort) which renders SVG to PDF drawing commands.
// Takes data (SVGDataPort) which provides raw SVG markup for sources.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) SVGWriter(writer SVGWriterPort, data SVGDataPort) *RenderBuilder {
	b.svgWriter = writer
	b.svgData = data
	return b
}

// Page sets the page dimensions for the render. Defaults to A4 if not set.
//
// Takes page (layouter_dto.PageConfig) which specifies the page geometry.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) Page(page layouter_dto.PageConfig) *RenderBuilder {
	b.pageConfig = &page
	return b
}

// FontSize sets the root font size in points. Defaults to 12.0 if not set.
//
// Takes size (float64) which is the font size in points.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) FontSize(size float64) *RenderBuilder {
	b.fontSize = size
	return b
}

// LineHeight sets the root line-height multiplier. Defaults to 0 (use
// the layouter's default) if not set.
//
// Takes height (float64) which is the unitless line-height multiplier.
//
// Returns *RenderBuilder for method chaining.
func (b *RenderBuilder) LineHeight(height float64) *RenderBuilder {
	b.lineHeight = height
	return b
}

// Do executes the full PDF render pipeline: runs the template, resolves
// CSS, builds the box tree, lays out, substitutes page numbers, instances
// variable fonts, paints to PDF, and applies any post-processing
// transformations.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
//
// Returns *pdfwriter_dto.PdfResult which contains the rendered PDF bytes,
// page count, and optional layout dump.
// Returns error when any stage of the pipeline fails.
func (b *RenderBuilder) Do(ctx context.Context) (*pdfwriter_dto.PdfResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RenderBuilder.Do")
	defer span.End()

	if b.templatePath == "" {
		l.ReportError(span, ErrTemplatePath, "Missing template path")
		return nil, ErrTemplatePath
	}

	templateAST, styling, err := b.service.templateRunner.RunPdfWithProps(ctx, b.templatePath, b.request, b.props)
	if err != nil {
		l.ReportError(span, err, "Failed to run PDF template")
		return nil, fmt.Errorf("failed to run PDF template '%s': %w", b.templatePath, err)
	}
	if templateAST == nil {
		return nil, fmt.Errorf("template runner returned a nil AST for PDF template '%s'", b.templatePath)
	}

	layoutConfig := b.buildLayoutConfig()
	layoutResult, err := b.service.layouter.Layout(ctx, templateAST, styling, layoutConfig)
	if err != nil {
		l.ReportError(span, err, "Layout failed for PDF template")
		return nil, fmt.Errorf("layout failed for PDF template '%s': %w", b.templatePath, err)
	}

	pageCount := b.substitutePageNumbers(layoutResult)

	pdfBytes, err := b.paintPDF(ctx, layoutConfig.Page, layoutResult)
	if err != nil {
		return nil, err
	}

	pdfBytes, err = b.applyTransforms(ctx, pdfBytes)
	if err != nil {
		l.ReportError(span, err, "PDF post-processing failed")
		return nil, fmt.Errorf("PDF post-processing failed for template '%s': %w", b.templatePath, err)
	}

	var layoutDump string
	if rootBox, ok := layoutResult.RootBox.(*layouter_domain.LayoutBox); ok && rootBox != nil {
		layoutDump = layouter_domain.SerialiseLayoutBoxToGoFileContent(rootBox, "test")
	}

	l.Trace("Successfully rendered PDF template via builder",
		logger_domain.String("templatePath", b.templatePath),
		logger_domain.Int("pdfSizeBytes", len(pdfBytes)),
		logger_domain.Int("pageCount", pageCount),
	)

	return &pdfwriter_dto.PdfResult{
		Content:    pdfBytes,
		PageCount:  pageCount,
		LayoutDump: layoutDump,
	}, nil
}

// substitutePageNumbers replaces page number placeholders in the layout
// tree and returns the total page count.
//
// Takes layoutResult (*layouter_dto.LayoutResult) which holds the layout
// tree to update.
//
// Returns int which is the total page count.
func (b *RenderBuilder) substitutePageNumbers(layoutResult *layouter_dto.LayoutResult) int {
	pageCount := len(layoutResult.Pages)
	if pageCount == 0 {
		pageCount = 1
	}
	if rootBox, ok := layoutResult.RootBox.(*layouter_domain.LayoutBox); ok && rootBox != nil {
		layouter_domain.SubstitutePageNumbers(rootBox, pageCount, b.service.fontMetrics)
	}
	return pageCount
}

// buildLayoutConfig constructs the layout configuration from the builder
// settings, applying defaults for page size and font size.
//
// Returns layouter_dto.LayoutConfig which holds the resolved layout parameters.
func (b *RenderBuilder) buildLayoutConfig() layouter_dto.LayoutConfig {
	pageConfig := layouter_dto.PageA4
	if b.pageConfig != nil {
		pageConfig = *b.pageConfig
	}

	fontSize := b.fontSize
	if fontSize == 0 {
		fontSize = builderDefaultFontSize
	}

	return layouter_dto.LayoutConfig{
		Page:              pageConfig,
		DefaultFontSize:   fontSize,
		DefaultLineHeight: b.lineHeight,
		Stylesheets:       b.stylesheets,
	}
}

// paintPDF instances variable fonts, creates and configures the painter,
// marks variable font instances, and paints the layout to PDF bytes.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes pageConfig (layouter_dto.PageConfig) which specifies the page
// dimensions.
//
// Returns []byte which is the rendered PDF content.
// Returns error when font instancing, context cancellation, or painting fails.
func (b *RenderBuilder) paintPDF(
	ctx context.Context,
	pageConfig layouter_dto.PageConfig,
	layoutResult *layouter_dto.LayoutResult,
) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	fontEntries := b.service.fontEntries
	painterFontEntries, err := instanceVariableFonts(fontEntries, b.service.fontMetrics)
	if err != nil {
		return nil, fmt.Errorf("variable font instancing failed: %w", err)
	}

	painterWidth := pageConfig.Width
	painterHeight := pageConfig.Height
	if pageConfig.AutoHeight && len(layoutResult.Pages) > 0 {
		painterHeight = layoutResult.Pages[0].Height
	}
	painter := NewPdfPainter(painterWidth, painterHeight, painterFontEntries, b.service.imageData)

	ConfigurePainter(painter, PainterConfig{
		Metadata:    b.metadata,
		ViewerPrefs: b.viewerPrefs,
		PageLabels:  b.pageLabels,
		Watermark:   b.watermark,
		PdfAConfig:  b.pdfaConfig,
		SVGWriter:   b.svgWriter,
		SVGData:     b.svgData,
		Tagged:      b.tagged,
	})

	for _, entry := range painterFontEntries {
		if entry.IsVariable {
			for weight := entry.WeightMin; weight <= entry.WeightMax; weight += fontWeightStepSize {
				painter.MarkVariableFont(entry.Family, weight, entry.Style)
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var buffer bytes.Buffer
	if err := painter.Paint(ctx, layoutResult, &buffer); err != nil {
		return nil, fmt.Errorf("PDF painting failed for template '%s': %w", b.templatePath, err)
	}

	return buffer.Bytes(), nil
}

// applyTransforms runs post-processing transform chains if configured.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
//
// Returns []byte which is the transformed PDF content.
// Returns error when chain creation or transformation fails.
func (b *RenderBuilder) applyTransforms(ctx context.Context, pdfBytes []byte) ([]byte, error) {
	if b.transformRegistry == nil || b.transformConfig == nil {
		return pdfBytes, nil
	}
	chain, err := NewPdfTransformerChain(b.transformRegistry, b.transformConfig)
	if err != nil {
		return nil, fmt.Errorf("creating PDF transform chain: %w", err)
	}
	if !chain.IsEmpty() {
		pdfBytes, err = chain.Transform(ctx, pdfBytes)
		if err != nil {
			return nil, fmt.Errorf("PDF post-processing failed: %w", err)
		}
	}
	return pdfBytes, nil
}

// instanceVariableFonts converts variable fonts into per-weight static
// instances suitable for PDF CIDFontType2 embedding.
//
// Takes fontEntries ([]layouter_dto.FontEntry) which holds the font
// entries to process.
// Takes fontMetrics (layouter_domain.FontMetricsPort) which provides
// font face access for glyph instancing.
//
// Returns []layouter_dto.FontEntry which holds the expanded font entries
// with variable fonts replaced by static per-weight instances.
// Returns error when font instancing fails.
func instanceVariableFonts(fontEntries []layouter_dto.FontEntry, fontMetrics layouter_domain.FontMetricsPort) ([]layouter_dto.FontEntry, error) {
	painterFontEntries := make([]layouter_dto.FontEntry, 0, len(fontEntries))
	for _, entry := range fontEntries {
		if !entry.IsVariable {
			painterFontEntries = append(painterFontEntries, entry)
			continue
		}
		for weight := entry.WeightMin; weight <= entry.WeightMax; weight += fontWeightStepSize {
			face := getFontFace(fontMetrics, layouter_domain.FontDescriptor{
				Family: entry.Family,
				Weight: weight,
				Style:  layouter_domain.FontStyle(entry.Style),
			})
			if face == nil {
				continue
			}
			instancedData, instanceError := InstanceVariableFont(entry.Data, func(gid uint16) InstancedGlyphData {
				return instanceGlyphFromFace(face, gid)
			})
			if instanceError != nil {
				return nil, fmt.Errorf("instancing variable font %q at weight %d: %w", entry.Family, weight, instanceError)
			}
			painterFontEntries = append(painterFontEntries, layouter_dto.FontEntry{
				Family: entry.Family,
				Weight: weight,
				Style:  entry.Style,
				Data:   instancedData,
			})
		}
	}
	return painterFontEntries, nil
}

// getFontFace retrieves the font face from font metrics.
//
// Takes fontMetrics (layouter_domain.FontMetricsPort) which provides
// font face access if it implements the fontFaceProvider interface.
// Takes desc (layouter_domain.FontDescriptor) which identifies the
// desired font family, weight, and style.
//
// Returns *font.Face which is the resolved font face, or nil if the
// metrics port does not support face retrieval.
func getFontFace(fontMetrics layouter_domain.FontMetricsPort, desc layouter_domain.FontDescriptor) *font.Face {
	type fontFaceProvider interface {
		GetFontFace(desc layouter_domain.FontDescriptor) *font.Face
	}
	if provider, ok := fontMetrics.(fontFaceProvider); ok {
		return provider.GetFontFace(desc)
	}
	return nil
}

// instanceGlyphFromFace extracts the variation-instanced outline and advance
// width for a glyph from a go-text Face.
//
// The Face must have its variations already set to the desired instance
// (e.g. wght=700).
//
// Takes face (*font.Face) which is the variation-instanced font face.
// Takes gid (uint16) which is the glyph ID to extract.
//
// Returns InstancedGlyphData which holds the instanced outline contours
// and advance width.
func instanceGlyphFromFace(face *font.Face, gid uint16) InstancedGlyphData {
	advance := uint16(math.Round(float64(face.HorizontalAdvance(font.GID(gid)))))

	data := face.GlyphData(font.GID(gid))
	outline, ok := data.(font.GlyphOutline)
	if !ok || len(outline.Segments) == 0 {
		return InstancedGlyphData{AdvanceWidth: advance}
	}

	contours := segmentsToContours(outline.Segments)
	return InstancedGlyphData{
		Contours:     contours,
		AdvanceWidth: advance,
	}
}

// segmentsToContours converts go-text outline segments into TrueType-style
// contour point lists.
//
// Each contour is a closed sequence of on-curve and off-curve points. Closing
// points that duplicate the first point are removed since TrueType contours
// close implicitly.
//
// Takes segments ([]font.Segment) which holds the go-text outline segments
// (MoveTo/LineTo/QuadTo/CubeTo).
//
// Returns [][]GlyphOutlinePoint which holds the converted contour point lists.
func segmentsToContours(segments []font.Segment) [][]GlyphOutlinePoint {
	var contours [][]GlyphOutlinePoint
	var current []GlyphOutlinePoint

	for _, seg := range segments {
		switch seg.Op {
		case 0:
			if len(current) > 0 {
				current = removeClosingPoint(current)
				contours = append(contours, current)
			}
			current = make([]GlyphOutlinePoint, 0, initialContourCapacity)
			current = append(current, GlyphOutlinePoint{
				X: seg.Args[0].X, Y: seg.Args[0].Y, OnCurve: true,
			})
		case 1:
			current = append(current, GlyphOutlinePoint{
				X: seg.Args[0].X, Y: seg.Args[0].Y, OnCurve: true,
			})
		case 2:
			current = append(current,
				GlyphOutlinePoint{
					X: seg.Args[0].X, Y: seg.Args[0].Y, OnCurve: false,
				},
				GlyphOutlinePoint{
					X: seg.Args[1].X, Y: seg.Args[1].Y, OnCurve: true,
				},
			)
		case segmentOpCubeTo:
			midX := (seg.Args[0].X + seg.Args[1].X) / 2
			midY := (seg.Args[0].Y + seg.Args[1].Y) / 2
			current = append(current,
				GlyphOutlinePoint{
					X: midX, Y: midY, OnCurve: false,
				},
				GlyphOutlinePoint{
					X: seg.Args[2].X, Y: seg.Args[2].Y, OnCurve: true,
				},
			)
		}
	}
	if len(current) > 0 {
		current = removeClosingPoint(current)
		contours = append(contours, current)
	}

	return contours
}

// removeClosingPoint strips the last on-curve point if it duplicates the
// first on-curve point, since TrueType contours close implicitly.
//
// Takes contour ([]GlyphOutlinePoint) which is the contour to trim.
//
// Returns []GlyphOutlinePoint which is the contour with the duplicate
// closing point removed if applicable.
func removeClosingPoint(contour []GlyphOutlinePoint) []GlyphOutlinePoint {
	if len(contour) < 2 {
		return contour
	}
	first := contour[0]
	last := contour[len(contour)-1]
	if last.OnCurve && first.OnCurve &&
		math.Abs(float64(last.X-first.X)) < closingPointTolerance &&
		math.Abs(float64(last.Y-first.Y)) < closingPointTolerance {
		return contour[:len(contour)-1]
	}
	return contour
}
