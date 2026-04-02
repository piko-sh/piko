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

package pdf

import (
	"errors"
	"fmt"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_encrypt"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_pades"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

// Service orchestrates the PDF render pipeline from template execution to
// PDF output.
type Service = pdfwriter_domain.PdfWriterService

// RenderBuilder configures and executes a single PDF render operation using
// a fluent interface.
type RenderBuilder = pdfwriter_domain.RenderBuilder

// Result contains the rendered PDF bytes and metadata.
type Result = pdfwriter_dto.PdfResult

// Config holds configuration for a PDF rendering operation.
type Config = pdfwriter_dto.PdfConfig

// Metadata holds optional PDF info dictionary fields.
type Metadata = pdfwriter_domain.PdfMetadata

// ViewerPreferences configures how PDF viewers display the document.
type ViewerPreferences = pdfwriter_domain.ViewerPreferences

// WatermarkConfig holds watermark text and styling settings.
type WatermarkConfig = pdfwriter_domain.WatermarkConfig

// PdfALevel represents a PDF/A conformance level.
type PdfALevel = pdfwriter_domain.PdfALevel

// PdfAConfig holds PDF/A conformance settings.
type PdfAConfig = pdfwriter_domain.PdfAConfig

// PageLabelRange defines a page label range for custom page numbering.
type PageLabelRange = pdfwriter_domain.PageLabelRange

// PageLabelStyle identifies a page label numbering style.
type PageLabelStyle = pdfwriter_domain.PageLabelStyle

// PageConfig defines page dimensions and margins.
type PageConfig = layouter_dto.PageConfig

// FontEntry describes a font available for embedding.
type FontEntry = layouter_dto.FontEntry

// PainterConfig holds optional configuration applied to a PdfPainter.
type PainterConfig = pdfwriter_domain.PainterConfig

// SVGWriterPort renders SVG markup as native PDF vector drawing commands.
type SVGWriterPort = pdfwriter_domain.SVGWriterPort

// SVGDataPort provides raw SVG markup for a given source.
type SVGDataPort = pdfwriter_domain.SVGDataPort

// TransformerRegistry holds available PDF post-processing transformers.
type TransformerRegistry = pdfwriter_domain.PdfTransformerRegistry

// TransformerPort is the interface implemented by PDF post-processing
// transformers.
type TransformerPort = pdfwriter_domain.PdfTransformerPort

// TransformerType categorises a transformer by its role in the pipeline.
type TransformerType = pdfwriter_dto.TransformerType

// TransformConfig specifies which post-processing transformers to apply.
type TransformConfig = pdfwriter_dto.TransformConfig

// WatermarkOptions configures the watermark transformer.
type WatermarkOptions = pdfwriter_dto.WatermarkOptions

// EncryptionOptions configures the encryption transformer.
type EncryptionOptions = pdfwriter_dto.EncryptionOptions

// PadesSignOptions configures the PAdES signing transformer.
type PadesSignOptions = pdfwriter_dto.PadesSignOptions

// PdfUAOptions configures the PDF/UA transformer.
type PdfUAOptions = pdfwriter_dto.PdfUAOptions

// PdfAOptions configures the PDF/A transformer.
type PdfAOptions = pdfwriter_dto.PdfAOptions

// LineariseOptions configures the linearisation transformer.
type LineariseOptions = pdfwriter_dto.LineariseOptions

// ObjStmOptions configures the object stream transformer.
type ObjStmOptions = pdfwriter_dto.ObjStmOptions

// FlattenOptions configures the flattening transformer.
type FlattenOptions = pdfwriter_dto.FlattenOptions

// RedactionOptions configures the redaction transformer.
type RedactionOptions = pdfwriter_dto.RedactionOptions

// CompressOptions configures the compression transformer.
type CompressOptions = pdfwriter_dto.CompressOptions

// RedactionRegion defines a rectangular region to redact.
type RedactionRegion = pdfwriter_dto.RedactionRegion

// CompressAlgorithm identifies a compression algorithm.
type CompressAlgorithm = pdfwriter_dto.CompressAlgorithm

const (
	// PdfA2B is PDF/A-2b conformance (basic).
	PdfA2B = pdfwriter_domain.PdfA2B

	// PdfA2U is PDF/A-2u conformance (Unicode).
	PdfA2U = pdfwriter_domain.PdfA2U

	// PdfA2A is PDF/A-2a conformance (accessible, auto-enables tagged PDF).
	PdfA2A = pdfwriter_domain.PdfA2A

	// LabelDecimal uses decimal page numbers (1, 2, 3, ...).
	LabelDecimal = pdfwriter_domain.LabelDecimal

	// LabelRomanUpper uses uppercase Roman numerals (I, II, III, ...).
	LabelRomanUpper = pdfwriter_domain.LabelRomanUpper

	// LabelRomanLower uses lowercase Roman numerals (i, ii, iii, ...).
	LabelRomanLower = pdfwriter_domain.LabelRomanLower

	// LabelAlphaUpper uses uppercase letters (A, B, C, ...).
	LabelAlphaUpper = pdfwriter_domain.LabelAlphaUpper

	// LabelAlphaLower uses lowercase letters (a, b, c, ...).
	LabelAlphaLower = pdfwriter_domain.LabelAlphaLower

	// LabelNone suppresses page number display for the range.
	LabelNone = pdfwriter_domain.LabelNone

	// TransformerContent is the transformer type for content-level changes.
	TransformerContent = pdfwriter_dto.TransformerContent

	// TransformerCompliance is the transformer type for compliance changes.
	TransformerCompliance = pdfwriter_dto.TransformerCompliance

	// TransformerDelivery is the transformer type for delivery-level changes.
	TransformerDelivery = pdfwriter_dto.TransformerDelivery

	// TransformerSecurity is the transformer type for security changes.
	TransformerSecurity = pdfwriter_dto.TransformerSecurity

	// TransformerCompression is the transformer type for compression.
	TransformerCompression = pdfwriter_dto.TransformerCompression

	// CompressZstd selects Zstandard compression.
	CompressZstd = pdfwriter_dto.CompressZstd
)

// Standard page sizes.
var (
	// PageA4 is the ISO A4 page size (210mm x 297mm).
	PageA4 = layouter_dto.PageA4

	// PageA3 is the ISO A3 page size (297mm x 420mm).
	PageA3 = layouter_dto.PageA3

	// PageLetter is the US Letter page size (8.5in x 11in).
	PageLetter = layouter_dto.PageLetter
)

// GetDefaultService returns the PDF writer service initialised by the
// framework during bootstrap.
//
// Returns Service which is the service instance ready for use.
// Returns error when the framework has not been bootstrapped.
func GetDefaultService() (Service, error) {
	service, err := bootstrap.GetPdfWriterService()
	if err != nil {
		return nil, fmt.Errorf("pdf: get default service: %w", err)
	}
	return service, nil
}

// NewRenderBuilder creates a new render builder for composing a PDF render
// operation.
//
// Takes service (Service) which is the PDF service to use for rendering.
//
// Returns *RenderBuilder which provides a fluent interface for building
// the render.
// Returns error when service is nil.
func NewRenderBuilder(service Service) (*RenderBuilder, error) {
	if service == nil {
		return nil, errors.New("pdf: service must not be nil")
	}
	return service.NewRender(), nil
}

// NewRenderBuilderFromDefault creates a new render builder using the
// framework's bootstrapped service.
//
// Returns *RenderBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped.
func NewRenderBuilderFromDefault() (*RenderBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf("pdf: get default service: %w", err)
	}
	return NewRenderBuilder(service)
}

// NewTransformerRegistry creates a new empty transformer registry for
// registering PDF post-processing transformers.
//
// Returns *TransformerRegistry which is ready to accept transformer
// registrations.
func NewTransformerRegistry() *TransformerRegistry {
	return pdfwriter_domain.NewPdfTransformerRegistry()
}

// NewEncryptTransformer creates an AES-256 PDF encryption transformer.
// Configure it via EncryptionOptions when building the TransformConfig.
//
// Returns TransformerPort which can be registered with a TransformerRegistry.
func NewEncryptTransformer() TransformerPort {
	return driven_transform_encrypt.New()
}

// NewPadesSignTransformer creates a PAdES digital signature transformer.
// Configure it via PadesSignOptions when building the TransformConfig.
//
// Returns TransformerPort which can be registered with a TransformerRegistry.
func NewPadesSignTransformer() TransformerPort {
	return driven_transform_pades.New()
}
