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
	"net/http"

	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

// pdfWriterService implements the PdfWriterService interface.
type pdfWriterService struct {
	// templateRunner executes compiled PDF templates.
	templateRunner TemplateRunnerPort

	// layouter resolves CSS, builds box trees, and performs layout.
	layouter LayoutPort

	// imageData provides image bytes for embedding. May be nil to skip image rendering.
	imageData ImageDataPort

	// fontMetrics provides font measurement for page number substitution. May be nil when
	// page number substitution is not needed.
	fontMetrics layouter_domain.FontMetricsPort

	// svgWriter renders SVGs as native PDF vector commands. May be nil.
	svgWriter SVGWriterPort

	// svgData provides raw SVG markup for src-based SVG sources. May be nil.
	svgData SVGDataPort

	// fontEntries holds the fonts available for embedding in PDF output.
	fontEntries []layouter_dto.FontEntry

	// layoutLimits bounds the work of every layout this service performs. Zero fields use
	// the built-in defaults; a render can override individual fields with
	// RenderBuilder.WithLayoutLimits.
	layoutLimits layouter_dto.LayoutLimits

	// maxImagePixels caps the pixel area (width times height) of any image a render embeds.
	// Zero uses the built-in default.
	maxImagePixels int
}

var (
	_ PdfWriterService = (*pdfWriterService)(nil)
)

// PdfServiceOption configures optional behaviour on a pdfWriterService.
type PdfServiceOption func(*pdfWriterService)

// WithSVGRenderer enables native SVG-to-PDF vector rendering.
//
// The writer paints SVG markup; the data port resolves src-based SVG sources (e.g. data:
// URIs).
//
// Takes writer (SVGWriterPort) which paints SVG documents as PDF vectors.
// Takes data (SVGDataPort) which provides raw SVG markup for src-based sources.
//
// Returns PdfServiceOption which applies the configuration.
func WithSVGRenderer(writer SVGWriterPort, data SVGDataPort) PdfServiceOption {
	return func(s *pdfWriterService) {
		s.svgWriter = writer
		s.svgData = data
	}
}

// WithLayoutLimits sets the layout limits applied to every render made by the service.
//
// Unset (non-positive) fields keep their built-in defaults, and a render may override
// individual fields with RenderBuilder.WithLayoutLimits.
//
// Takes limits (layouter_dto.LayoutLimits) which holds the service-wide limits.
//
// Returns PdfServiceOption which applies the configuration.
func WithLayoutLimits(limits layouter_dto.LayoutLimits) PdfServiceOption {
	return func(s *pdfWriterService) {
		s.layoutLimits = limits
	}
}

// WithMaxImagePixels caps the pixel area (width times height) of any image embedded by
// the service's renders.
//
// Images above the cap fail the render rather than allocating a huge pixel buffer.
// Non-positive values keep the built-in default.
//
// Takes pixels (int) which is the maximum pixel area of one image.
//
// Returns PdfServiceOption which applies the configuration.
func WithMaxImagePixels(pixels int) PdfServiceOption {
	return func(s *pdfWriterService) {
		s.maxImagePixels = pixels
	}
}

// NewPdfWriterService creates a new PDF writer service.
//
// Takes templateRunner (TemplateRunnerPort) which executes compiled PDF templates.
// Takes layouter (LayoutPort) which provides CSS resolution, box tree construction, and
// layout.
// Takes fontEntries ([]layouter_dto.FontEntry) which are the fonts available for
// embedding. May be nil for Helvetica fallback.
// Takes imageData (ImageDataPort) which provides image bytes for embedding. May be nil to
// skip image rendering.
// Takes fontMetrics (layouter_domain.FontMetricsPort) which provides font measurement for
// page number substitution. May be nil.
// Takes opts (...PdfServiceOption) which configure PDF generation and resource limits.
//
// Returns PdfWriterService which is configured and ready for use.
func NewPdfWriterService(
	templateRunner TemplateRunnerPort,
	layouter LayoutPort,
	fontEntries []layouter_dto.FontEntry,
	imageData ImageDataPort,
	fontMetrics layouter_domain.FontMetricsPort,
	opts ...PdfServiceOption,
) PdfWriterService {
	service := &pdfWriterService{
		templateRunner: templateRunner,
		layouter:       layouter,
		fontEntries:    fontEntries,
		imageData:      imageData,
		fontMetrics:    fontMetrics,
		svgWriter:      nil,
		svgData:        nil,
		layoutLimits:   layouter_dto.LayoutLimits{},
		maxImagePixels: 0,
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

// NewRender creates a RenderBuilder for composing a PDF render operation using a fluent
// interface.
//
// Returns *RenderBuilder which provides methods for configuring the render and executing
// it via Do(ctx).
func (s *pdfWriterService) NewRender() *RenderBuilder {
	builder := RenderBuilder{}
	builder.service = s
	builder.svgWriter = s.svgWriter
	builder.svgData = s.svgData
	builder.tagged = true
	return &builder
}

// Render executes the full PDF pipeline for a single PDF template.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes request (*http.Request) which provides the HTTP context for template rendering.
// Takes templatePath (string) which is the path to the PDF template.
// Takes props (any) which contains the data to pass to the template.
// Takes config (pdfwriter_dto.PdfConfig) which specifies page dimensions, font size, and
// other layout settings.
//
// Returns result (*pdfwriter_dto.PdfResult) which contains the rendered PDF bytes and
// page count.
// Returns err (error) when any stage of the pipeline fails, including a recovered panic.
func (s *pdfWriterService) Render(
	ctx context.Context,
	request *http.Request,
	templatePath string,
	props any,
	config pdfwriter_dto.PdfConfig,
) (result *pdfwriter_dto.PdfResult, err error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "PdfWriterService.Render")
	defer span.End()
	defer func() { StorePanicAsError(ctx, "PDF render", recover(), &err) }()

	templateAST, styling, err := s.templateRunner.RunPdfWithProps(ctx, templatePath, request, props)
	if err != nil {
		l.ReportError(span, err, "Failed to run PDF template via manifest runner")
		return nil, fmt.Errorf("failed to run PDF template '%s': %w", templatePath, err)
	}
	if templateAST == nil {
		nilASTErr := fmt.Errorf("manifest runner returned a nil AST for PDF template '%s'", templatePath)
		l.ReportError(span, nilASTErr, "Cannot render nil AST")
		return nil, nilASTErr
	}

	layoutConfig := layouter_dto.LayoutConfig{
		Page:              config.Page,
		DefaultFontSize:   config.DefaultFontSize,
		DefaultLineHeight: config.DefaultLineHeight,
		Stylesheets:       config.Stylesheets,
		DefaultFontFamily: "",
		Limits:            s.layoutLimits,
	}

	layoutConfig.Page = applyPageCSS(styling, layoutConfig.Page)

	layoutResult, err := s.layouter.Layout(ctx, templateAST, styling, layoutConfig)
	if err != nil {
		l.ReportError(span, err, "Layout failed for PDF template")
		return nil, fmt.Errorf("layout failed for PDF template '%s': %w", templatePath, err)
	}

	pdfBytes, err := s.paintLayout(ctx, layoutConfig.Page, layoutResult)
	if err != nil {
		l.ReportError(span, err, "PDF painting failed")
		return nil, fmt.Errorf("PDF painting failed for template '%s': %w", templatePath, err)
	}

	l.Trace("Successfully rendered PDF template",
		logger_domain.String("templatePath", templatePath),
		logger_domain.Int("pdfSizeBytes", len(pdfBytes)),
		logger_domain.Int("pageCount", len(layoutResult.Pages)),
	)

	return &pdfwriter_dto.PdfResult{
		Content:    pdfBytes,
		PageCount:  max(len(layoutResult.Pages), 1),
		LayoutDump: "",
	}, nil
}

// paintLayout paints a laid-out document to PDF bytes with the service's fonts, image
// data, SVG renderer and image pixel cap.
//
// Takes page (layouter_dto.PageConfig) which holds the page size and margins.
// Takes layoutResult (*layouter_dto.LayoutResult) which holds the laid-out document.
//
// Returns []byte which is the painted PDF document.
// Returns error when painting fails.
func (s *pdfWriterService) paintLayout(
	ctx context.Context, page layouter_dto.PageConfig, layoutResult *layouter_dto.LayoutResult,
) ([]byte, error) {
	painter := NewPdfPainter(page.Width, page.Height, s.fontEntries, s.imageData)
	painter.imageEmbedder.SetMaxPixels(s.maxImagePixels)
	if s.svgWriter != nil {
		painter.setSVGWriter(s.svgWriter, s.svgData)
	}
	painter.setPageMargins(page.MarginLeft, page.MarginTop, page.ContentAreaHeight())

	var buffer bytes.Buffer
	if err := painter.Paint(ctx, layoutResult, &buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
