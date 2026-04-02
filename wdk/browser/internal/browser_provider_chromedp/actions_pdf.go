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

package browser_provider_chromedp

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const (
	// PDFPaperWidthLetter is the width of US Letter paper in inches.
	PDFPaperWidthLetter = 8.5

	// PDFPaperHeightLetter is the height of US Letter paper in inches (11).
	PDFPaperHeightLetter = 11

	// PDFPaperWidthA4 is the width of A4 paper in inches (210 mm).
	PDFPaperWidthA4 = 8.27

	// PDFPaperHeightA4 is the height of A4 paper in inches (297mm).
	PDFPaperHeightA4 = 11.69

	// PDFMarginDefault is the default margin size in inches for all PDF edges.
	PDFMarginDefault = 0.4
)

// PDFOptions configures PDF generation.
type PDFOptions struct {
	// PageRanges specifies paper ranges to print, e.g., '1-5, 8, 11-13'.
	// Empty string means all pages.
	PageRanges string

	// HeaderTemplate is the HTML template for the page header.
	// Should be valid HTML markup with the following CSS classes:
	// date (formatted print date), title (document title),
	// url (document location), pageNumber (current page number),
	// totalPages (total pages in the document).
	HeaderTemplate string

	// FooterTemplate is the HTML template for the page footer.
	// Uses the same format as HeaderTemplate.
	FooterTemplate string

	// Scale is the zoom level for page rendering; default is 1.0.
	Scale float64

	// PaperWidth in inches. Default is 8.5.
	PaperWidth float64

	// PaperHeight is the paper height in inches. The default is 11.
	PaperHeight float64

	// MarginTop is the top margin in inches. Default is 0.4.
	MarginTop float64

	// MarginBottom is the bottom margin in inches. Default is 0.4.
	MarginBottom float64

	// MarginLeft is the left margin in inches. Default is 0.4.
	MarginLeft float64

	// MarginRight is the right margin in inches. Default is 0.4.
	MarginRight float64

	// Landscape sets the page orientation to landscape when true.
	Landscape bool

	// PrintBackground enables printing of background graphics.
	PrintBackground bool

	// PreferCSSPageSize uses CSS @page size instead of paper size settings.
	PreferCSSPageSize bool
}

// DefaultPDFOptions returns sensible defaults for PDF generation.
//
// Returns PDFOptions which contains letter-sized page settings with default
// margins and background printing enabled.
func DefaultPDFOptions() PDFOptions {
	return PDFOptions{
		PageRanges:        "",
		HeaderTemplate:    "",
		FooterTemplate:    "",
		Scale:             1.0,
		PaperWidth:        PDFPaperWidthLetter,
		PaperHeight:       PDFPaperHeightLetter,
		MarginTop:         PDFMarginDefault,
		MarginBottom:      PDFMarginDefault,
		MarginLeft:        PDFMarginDefault,
		MarginRight:       PDFMarginDefault,
		Landscape:         false,
		PrintBackground:   true,
		PreferCSSPageSize: false,
	}
}

// A4PDFOptions returns options for A4 paper size (210mm x 297mm).
//
// Returns PDFOptions which is configured for A4 dimensions.
func A4PDFOptions() PDFOptions {
	opts := DefaultPDFOptions()
	opts.PaperWidth = PDFPaperWidthA4
	opts.PaperHeight = PDFPaperHeightA4
	return opts
}

// PrintToPDF generates a PDF of the current page using default options.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// operation.
//
// Returns []byte which contains the PDF data.
// Returns error when the PDF generation fails.
func PrintToPDF(ctx *ActionContext) ([]byte, error) {
	return PrintToPDFWithOptions(ctx, DefaultPDFOptions())
}

// PrintToPDFWithOptions generates a PDF with custom options.
//
// Takes ctx (*ActionContext) which provides the browser context for rendering.
// Takes opts (PDFOptions) which specifies the PDF output settings.
//
// Returns []byte which contains the raw PDF data.
// Returns error when the PDF cannot be generated.
func PrintToPDFWithOptions(ctx *ActionContext, opts PDFOptions) ([]byte, error) {
	var buffer []byte

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		printCmd := page.PrintToPDF().
			WithLandscape(opts.Landscape).
			WithPrintBackground(opts.PrintBackground).
			WithScale(opts.Scale).
			WithPaperWidth(opts.PaperWidth).
			WithPaperHeight(opts.PaperHeight).
			WithMarginTop(opts.MarginTop).
			WithMarginBottom(opts.MarginBottom).
			WithMarginLeft(opts.MarginLeft).
			WithMarginRight(opts.MarginRight).
			WithPreferCSSPageSize(opts.PreferCSSPageSize)

		if opts.PageRanges != "" {
			printCmd = printCmd.WithPageRanges(opts.PageRanges)
		}
		if opts.HeaderTemplate != "" {
			printCmd = printCmd.WithDisplayHeaderFooter(true).WithHeaderTemplate(opts.HeaderTemplate)
		}
		if opts.FooterTemplate != "" {
			printCmd = printCmd.WithDisplayHeaderFooter(true).WithFooterTemplate(opts.FooterTemplate)
		}

		var err error
		buffer, _, err = printCmd.Do(ctx2)
		return err
	}))
	if err != nil {
		return nil, fmt.Errorf("printing to PDF: %w", err)
	}

	return buffer, nil
}

// PrintToPDFLandscape generates a landscape PDF of the current page.
//
// Takes ctx (*ActionContext) which provides the browser context for the page.
//
// Returns []byte which contains the PDF document data.
// Returns error when the PDF generation fails.
func PrintToPDFLandscape(ctx *ActionContext) ([]byte, error) {
	opts := DefaultPDFOptions()
	opts.Landscape = true
	return PrintToPDFWithOptions(ctx, opts)
}

// PrintToPDFA4 generates an A4-sized PDF of the current page.
//
// Takes ctx (*ActionContext) which provides the browser context for printing.
//
// Returns []byte which contains the generated PDF data.
// Returns error when the PDF generation fails.
func PrintToPDFA4(ctx *ActionContext) ([]byte, error) {
	return PrintToPDFWithOptions(ctx, A4PDFOptions())
}

// PrintToPDFNoBackground generates a PDF without background graphics.
//
// Takes ctx (*ActionContext) which provides the browser action context.
//
// Returns []byte which contains the PDF document data.
// Returns error when the PDF generation fails.
func PrintToPDFNoBackground(ctx *ActionContext) ([]byte, error) {
	opts := DefaultPDFOptions()
	opts.PrintBackground = false
	return PrintToPDFWithOptions(ctx, opts)
}

// PrintToPDFWithHeaderFooter generates a PDF with custom header and footer.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes header (string) which specifies the HTML template for the page header.
// Takes footer (string) which specifies the HTML template for the page footer.
//
// Returns []byte which contains the generated PDF data.
// Returns error when the PDF generation fails.
func PrintToPDFWithHeaderFooter(ctx *ActionContext, header, footer string) ([]byte, error) {
	opts := DefaultPDFOptions()
	opts.HeaderTemplate = header
	opts.FooterTemplate = footer
	return PrintToPDFWithOptions(ctx, opts)
}

// PrintToPDFPageRange generates a PDF of specific pages.
// Use pageRanges like "1-5, 8, 11-13" for specific pages.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes pageRanges (string) which specifies pages to include in the PDF.
//
// Returns []byte which contains the generated PDF data.
// Returns error when PDF generation fails.
func PrintToPDFPageRange(ctx *ActionContext, pageRanges string) ([]byte, error) {
	opts := DefaultPDFOptions()
	opts.PageRanges = pageRanges
	return PrintToPDFWithOptions(ctx, opts)
}
