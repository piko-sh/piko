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

package bootstrap

import (
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

// SetPdfWriterService sets the PDF writer service. Called by the daemon builder to set
// the service built using the selected manifest runner and layouter.
//
// Takes s (PdfWriterService) which provides PDF rendering operations.
func (c *Container) SetPdfWriterService(s pdfwriter_domain.PdfWriterService) {
	c.pdfWriterService = s
}

// GetPdfWriterService returns the PdfWriterService previously registered by the daemon
// builder.
//
// Returns pdfwriter_domain.PdfWriterService which provides PDF rendering operations, or
// nil if not yet initialised.
func (c *Container) GetPdfWriterService() pdfwriter_domain.PdfWriterService {
	return c.pdfWriterService
}

// WithPdfLayoutLimits bounds each PDF layout by limiting raw HTML size, nesting depth,
// box count, table and grid sizes, repeat() counts and page count. Unset (non-positive)
// fields keep their configured or built-in defaults.
//
// Takes limits (layouter_dto.LayoutLimits) which holds the limits to apply.
//
// Returns Option which the bootstrap consumes when applied.
func WithPdfLayoutLimits(limits layouter_dto.LayoutLimits) Option {
	return func(c *Container) {
		pdf := &c.ensureOverrides().Pdf
		setPositiveOverride(&pdf.MaxRawHTMLBytes, limits.MaxRawHTMLBytes)
		setPositiveOverride(&pdf.MaxNestingDepth, limits.MaxNestingDepth)
		setPositiveOverride(&pdf.MaxBoxNodes, limits.MaxBoxNodes)
		setPositiveOverride(&pdf.MaxColspan, limits.MaxColspan)
		setPositiveOverride(&pdf.MaxRowspan, limits.MaxRowspan)
		setPositiveOverride(&pdf.MaxTableColumns, limits.MaxTableColumns)
		setPositiveOverride(&pdf.MaxGridTracks, limits.MaxGridTracks)
		setPositiveOverride(&pdf.MaxGridCells, limits.MaxGridCells)
		setPositiveOverride(&pdf.MaxRepeatCount, limits.MaxRepeatCount)
		setPositiveOverride(&pdf.MaxPages, limits.MaxPages)
	}
}

// WithPdfMaxImagePixels caps the pixel area (width times height) of any image a PDF
// render embeds. Non-positive values keep the configured or built-in default.
//
// Takes pixels (int) which is the maximum pixel area of one image.
//
// Returns Option which the bootstrap consumes when applied.
func WithPdfMaxImagePixels(pixels int) Option {
	return func(c *Container) {
		setPositiveOverride(&c.ensureOverrides().Pdf.MaxImagePixels, pixels)
	}
}

// pdfServiceLimitOptions converts the resolved PDF configuration into the PDF writer
// service options carrying its layout limits and image pixel cap. Unset fields become
// zero, which the PDF pipeline treats as its built-in default.
//
// Takes pdf (config.PdfConfig) which holds the resolved PDF limits.
//
// Returns []pdfwriter_domain.PdfServiceOption which applies the limits to a service.
func pdfServiceLimitOptions(pdf config.PdfConfig) []pdfwriter_domain.PdfServiceOption {
	return []pdfwriter_domain.PdfServiceOption{
		pdfwriter_domain.WithLayoutLimits(pdfLayoutLimits(pdf)),
		pdfwriter_domain.WithMaxImagePixels(deref(pdf.MaxImagePixels, 0)),
	}
}

// pdfLayoutLimits converts the resolved PDF configuration into layout limits. Unset
// fields become zero, which the layouter treats as its built-in default.
//
// Takes pdf (config.PdfConfig) which holds the resolved PDF limits.
//
// Returns layouter_dto.LayoutLimits which holds the configured layout limits.
func pdfLayoutLimits(pdf config.PdfConfig) layouter_dto.LayoutLimits {
	return layouter_dto.LayoutLimits{
		MaxRawHTMLBytes: deref(pdf.MaxRawHTMLBytes, 0),
		MaxNestingDepth: deref(pdf.MaxNestingDepth, 0),
		MaxBoxNodes:     deref(pdf.MaxBoxNodes, 0),
		MaxColspan:      deref(pdf.MaxColspan, 0),
		MaxRowspan:      deref(pdf.MaxRowspan, 0),
		MaxTableColumns: deref(pdf.MaxTableColumns, 0),
		MaxGridTracks:   deref(pdf.MaxGridTracks, 0),
		MaxGridCells:    deref(pdf.MaxGridCells, 0),
		MaxRepeatCount:  deref(pdf.MaxRepeatCount, 0),
		MaxPages:        deref(pdf.MaxPages, 0),
	}
}

// setPositiveOverride stores value in target when it is positive, leaving an unset
// override untouched otherwise.
//
// Takes target (**int) which is the override field to set.
// Takes value (int) which is the candidate value.
func setPositiveOverride(target **int, value int) {
	if value > 0 {
		*target = new(value)
	}
}
