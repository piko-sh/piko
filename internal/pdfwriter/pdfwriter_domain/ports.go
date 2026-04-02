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

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

// PdfWriterService is the primary driving port for PDF generation. It
// orchestrates the full pipeline from template execution to PDF output.
//
// Use NewRender to create a fluent builder for configuring and executing
// a render operation with full control over metadata, watermarks, PDF/A,
// accessibility tagging, page labels, SVG rendering, and post-processing
// transformations. The older Render method is kept for backwards
// compatibility with existing daemon handler code.
type PdfWriterService interface {
	// Render executes the full PDF pipeline for a single PDF template:
	// runs the template through the manifest runner, resolves CSS, builds
	// the box tree, lays out, and paints to PDF.
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
	Render(
		ctx context.Context,
		request *http.Request,
		templatePath string,
		props any,
		config pdfwriter_dto.PdfConfig,
	) (*pdfwriter_dto.PdfResult, error)

	// NewRender creates a RenderBuilder for composing a PDF render
	// operation using a fluent interface. The builder provides full
	// control over metadata, viewer preferences, watermarks, PDF/A
	// conformance, accessibility tagging, page labels, SVG rendering,
	// stylesheets, and post-processing transformations.
	//
	// Returns *RenderBuilder which provides a fluent interface for
	// configuring and executing the render.
	NewRender() *RenderBuilder
}

// LayoutPort provides layout capabilities to the PDF writer. It resolves
// CSS, builds the box tree, and performs layout to produce positioned boxes.
type LayoutPort interface {
	// Layout resolves CSS, builds the box tree, performs layout, and
	// returns the result containing the positioned box tree.
	//
	// Takes ctx (context.Context) which carries cancellation and tracing.
	// Takes tree (*ast_domain.TemplateAST) which is the template AST to
	// lay out.
	// Takes styling (string) which is the CSS from the template's style
	// block.
	// Takes config (layouter_dto.LayoutConfig) which specifies page
	// dimensions and font settings.
	//
	// Returns *layouter_dto.LayoutResult which contains the positioned
	// box tree.
	// Returns error when CSS resolution, box tree construction, or layout
	// fails.
	Layout(
		ctx context.Context,
		tree *ast_domain.TemplateAST,
		styling string,
		config layouter_dto.LayoutConfig,
	) (*layouter_dto.LayoutResult, error)
}

// ImageDataPort provides image bytes for embedding in PDF output.
// This is separate from the layouter's ImageResolverPort (which only
// provides dimensions for layout) because painting needs the actual
// image bytes, not just dimensions.
type ImageDataPort interface {
	// GetImageData fetches the raw image bytes for the given source path.
	//
	// Takes ctx (context.Context) which carries cancellation and tracing.
	// Takes source (string) which is the image source path or URL.
	//
	// Returns []byte which contains the raw image data (JPEG or PNG).
	// Returns string which is the detected format ("jpeg" or "png").
	// Returns error when the image cannot be fetched or decoded.
	GetImageData(ctx context.Context, source string) ([]byte, string, error)
}

// SVGDataPort provides raw SVG markup for a given source. Used in
// combination with SVGWriterPort to enable native vector rendering.
type SVGDataPort interface {
	// GetSVGData returns the raw SVG XML string for the given source.
	//
	// Takes ctx (context.Context) which carries cancellation and tracing.
	// Takes source (string) which is the image source path or URL.
	//
	// Returns string which is the raw SVG XML markup.
	// Returns bool which is true if the source is an SVG.
	GetSVGData(ctx context.Context, source string) (string, bool)
}

// SVGWriterPort renders an SVG document as native PDF vector drawing
// commands into a content stream. When this port is not provided, the
// painter falls back to the raster image path for SVG elements.
type SVGWriterPort interface {
	// RenderSVG parses an SVG string and emits PDF drawing operators
	// into the provided render context at the given position and size.
	//
	// Takes ctx (context.Context) which carries cancellation and tracing.
	// Takes svgData (string) which is the raw SVG XML markup.
	// Takes renderContext (SVGRenderContext) which provides access to PDF
	// drawing infrastructure.
	// Takes x, y, w, h (float64) which define the render rectangle
	// in PDF coordinates (bottom-left origin).
	//
	// Returns error when the SVG cannot be parsed or rendered.
	RenderSVG(ctx context.Context, svgData string, renderContext SVGRenderContext, x, y, w, h float64) error
}

// SVGRenderContext provides the PDF infrastructure needed by an SVG
// writer to emit drawing commands.
type SVGRenderContext struct {
	// Stream holds the PDF content stream for emitting drawing operators.
	Stream *ContentStream

	// ShadingManager holds the shading resource manager for gradient fills.
	ShadingManager *ShadingManager

	// ExtGStateManager holds the graphics state manager for opacity and blend modes.
	ExtGStateManager *ExtGStateManager

	// FontEmbedder holds the font embedder for registering and embedding fonts.
	FontEmbedder *FontEmbedder

	// ImageEmbedder holds the image embedder for registering raster images.
	ImageEmbedder *ImageEmbedder

	// RegisterFont registers a font for use in SVG text and returns
	// the PDF resource name. When nil, text elements are skipped.
	RegisterFont func(family string, weight int, style int, size float64) string

	// MeasureText returns the width of the given text in points at
	// the specified font settings.
	//
	// When nil, text-anchor adjustments are skipped.
	MeasureText func(family string, weight int, style int, size float64, text string) float64

	// GetImageData fetches raw image bytes for embedding raster images
	// inside SVG <image> elements. When nil, <image> elements are skipped.
	GetImageData func(ctx context.Context, source string) ([]byte, string, error)

	// PageHeight holds the page height in points for coordinate conversion.
	PageHeight float64
}

// TemplateRunnerPort provides access to compiled PDF templates.
type TemplateRunnerPort interface {
	// RunPdfWithProps executes a PDF template with the given props and
	// returns the AST and styling.
	//
	// Takes ctx (context.Context) which carries cancellation and tracing.
	// Takes templatePath (string) which is the path to the PDF template.
	// Takes request (*http.Request) which provides the HTTP context.
	// Takes props (any) which contains the data to pass to the template.
	//
	// Returns *ast_domain.TemplateAST which is the compiled template tree.
	// Returns string which is the CSS styling from the template.
	// Returns error when the template cannot be found or executed.
	RunPdfWithProps(
		ctx context.Context,
		templatePath string,
		request *http.Request,
		props any,
	) (*ast_domain.TemplateAST, string, error)
}
