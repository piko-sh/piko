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

	// imageData provides image bytes for embedding. May be nil to skip
	// image rendering.
	imageData ImageDataPort

	// fontMetrics provides font measurement for page number substitution.
	// May be nil when page number substitution is not needed.
	fontMetrics layouter_domain.FontMetricsPort

	// fontEntries holds the fonts available for embedding in PDF output.
	fontEntries []layouter_dto.FontEntry
}

var _ PdfWriterService = (*pdfWriterService)(nil)

// NewPdfWriterService creates a new PDF writer service.
//
// Takes templateRunner (TemplateRunnerPort) which executes compiled
// PDF templates.
// Takes layouter (LayoutPort) which provides CSS resolution, box tree
// construction, and layout.
// Takes fontEntries ([]layouter_dto.FontEntry) which are the fonts
// available for embedding. May be nil for Helvetica fallback.
// Takes imageData (ImageDataPort) which provides image bytes for
// embedding. May be nil to skip image rendering.
// Takes fontMetrics (layouter_domain.FontMetricsPort) which provides
// font measurement for page number substitution. May be nil.
//
// Returns PdfWriterService which is configured and ready for use.
func NewPdfWriterService(
	templateRunner TemplateRunnerPort,
	layouter LayoutPort,
	fontEntries []layouter_dto.FontEntry,
	imageData ImageDataPort,
	fontMetrics layouter_domain.FontMetricsPort,
) PdfWriterService {
	return &pdfWriterService{
		templateRunner: templateRunner,
		layouter:       layouter,
		fontEntries:    fontEntries,
		imageData:      imageData,
		fontMetrics:    fontMetrics,
	}
}

// NewRender creates a RenderBuilder for composing a PDF render operation
// using a fluent interface.
//
// Returns *RenderBuilder which provides methods for configuring the
// render and executing it via Do(ctx).
func (s *pdfWriterService) NewRender() *RenderBuilder {
	return &RenderBuilder{
		service: s,
	}
}

// Render executes the full PDF pipeline for a single PDF template.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes request (*http.Request) which provides the HTTP context for
// template rendering.
// Takes templatePath (string) which is the path to the PDF template.
// Takes props (any) which contains the data to pass to the template.
// Takes config (pdfwriter_dto.PdfConfig) which specifies page
// dimensions, font size, and other layout settings.
//
// Returns *pdfwriter_dto.PdfResult which contains the rendered PDF
// bytes and page count.
// Returns error when any stage of the pipeline fails.
func (s *pdfWriterService) Render(
	ctx context.Context,
	request *http.Request,
	templatePath string,
	props any,
	config pdfwriter_dto.PdfConfig,
) (*pdfwriter_dto.PdfResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "PdfWriterService.Render")
	defer span.End()

	templateAST, styling, err := s.templateRunner.RunPdfWithProps(ctx, templatePath, request, props)
	if err != nil {
		l.ReportError(span, err, "Failed to run PDF template via manifest runner")
		return nil, fmt.Errorf("failed to run PDF template '%s': %w", templatePath, err)
	}
	if templateAST == nil {
		err := fmt.Errorf("manifest runner returned a nil AST for PDF template '%s'", templatePath)
		l.ReportError(span, err, "Cannot render nil AST")
		return nil, err
	}

	layoutConfig := layouter_dto.LayoutConfig{
		Page:              config.Page,
		DefaultFontSize:   config.DefaultFontSize,
		DefaultLineHeight: config.DefaultLineHeight,
		Stylesheets:       config.Stylesheets,
	}

	layoutResult, err := s.layouter.Layout(ctx, templateAST, styling, layoutConfig)
	if err != nil {
		l.ReportError(span, err, "Layout failed for PDF template")
		return nil, fmt.Errorf("layout failed for PDF template '%s': %w", templatePath, err)
	}

	painter := NewPdfPainter(config.Page.Width, config.Page.Height, s.fontEntries, s.imageData)

	var buffer bytes.Buffer
	if err := painter.Paint(ctx, layoutResult, &buffer); err != nil {
		l.ReportError(span, err, "PDF painting failed")
		return nil, fmt.Errorf("PDF painting failed for template '%s': %w", templatePath, err)
	}

	l.Trace("Successfully rendered PDF template",
		logger_domain.String("templatePath", templatePath),
		logger_domain.Int("pdfSizeBytes", buffer.Len()),
		logger_domain.Int("pageCount", len(layoutResult.Pages)),
	)

	pageCount := len(layoutResult.Pages)
	if pageCount == 0 {
		pageCount = 1
	}

	return &pdfwriter_dto.PdfResult{
		Content:   buffer.Bytes(),
		PageCount: pageCount,
	}, nil
}
