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

package image_domain

import (
	"context"
	"io"

	"piko.sh/piko/internal/image/image_dto"
)

// Service is the primary (driving) port for the image hexagon. It defines
// the high-level image processing operations available to the application,
// orchestrating transformation and storage.
//
// Service implements image_domain.Service and media.ImageService.
type Service interface {
	// Transform creates a new transform builder for the given input.
	// This is the fluent API entry point for performing image transformations.
	//
	// Takes input (io.Reader) which provides the source image data.
	//
	// Returns *TransformBuilder which provides a fluent interface for
	// configuring and executing the transformation.
	Transform(input io.Reader) *TransformBuilder

	// TransformStream performs a direct streaming transform from input to result
	// stream.
	//
	// Takes input (io.Reader) which provides the source image data.
	// Takes spec (image_dto.TransformationSpec) which defines the transformation.
	//
	// Returns *image_dto.TransformedImageResult which contains the transformed
	// image data.
	// Returns error when the transformation fails.
	TransformStream(ctx context.Context, input io.Reader, spec image_dto.TransformationSpec) (*image_dto.TransformedImageResult, error)

	// GenerateResponsiveVariants creates multiple image variants for responsive
	// images. It parses sizes and densities from the spec's ResponsiveSpec and
	// generates all necessary variants.
	//
	// Takes input (io.Reader) which provides the source image data.
	// Takes baseSpec (image_dto.TransformationSpec) which defines the base
	// transformation settings including responsive configuration.
	//
	// Returns []image_dto.ResponsiveVariant which contains metadata for building
	// srcset attributes.
	// Returns error when variant generation fails.
	GenerateResponsiveVariants(ctx context.Context, input io.Reader, baseSpec image_dto.TransformationSpec) ([]image_dto.ResponsiveVariant, error)

	// GeneratePlaceholder creates a tiny, low-quality, heavily blurred
	// placeholder image (LQIP).
	//
	// Takes input (io.Reader) which provides the source image data.
	// Takes baseSpec (image_dto.TransformationSpec) which defines the base
	// transformation settings.
	//
	// Returns string which is the placeholder as a base64-encoded data URL
	// suitable for inline embedding.
	// Returns error when placeholder generation fails.
	GeneratePlaceholder(ctx context.Context, input io.Reader, baseSpec image_dto.TransformationSpec) (string, error)

	// GetDimensions extracts width and height from image data without transforming.
	//
	// Uses lightweight header decoding (image.DecodeConfig) for minimal overhead.
	// Supports all formats registered with the standard library (JPEG, PNG, GIF,
	// WebP).
	//
	// Takes input (io.Reader) which provides the source image data.
	//
	// Returns width (int) in pixels.
	// Returns height (int) in pixels.
	// Returns error when the image cannot be decoded or is not a valid image.
	GetDimensions(ctx context.Context, input io.Reader) (width int, height int, err error)
}

// TransformerPort is the driven port for image transformation adapters.
// It defines the contract that a concrete image processing library (such as
// govips) must fulfil.
type TransformerPort interface {
	// Transform reads from input, applies the given specification, and writes to
	// output.
	//
	// Takes input (io.Reader) which supplies the source data.
	// Takes output (io.Writer) which receives the transformed data.
	// Takes spec (TransformationSpec) which defines the transformation to apply.
	//
	// Returns mimeType (string) which is the MIME type of the output.
	// Returns error when the transformation fails.
	Transform(ctx context.Context, input io.Reader, output io.Writer, spec image_dto.TransformationSpec) (mimeType string, err error)

	// GetSupportedFormats returns the list of output formats this provider can
	// generate. Format names should be lowercase (e.g., "jpeg", "png", "webp",
	// "avif").
	//
	// Returns []string which contains the supported format names.
	GetSupportedFormats() []string

	// GetDimensions extracts width and height from image data without
	// transformation. Providers should use lightweight header decoding
	// when possible.
	//
	// Takes input (io.Reader) which provides the source image data.
	//
	// Returns width (int) in pixels.
	// Returns height (int) in pixels.
	// Returns error when the image cannot be decoded or is not supported
	// by this provider.
	GetDimensions(ctx context.Context, input io.Reader) (width int, height int, err error)

	// GetSupportedModifiers returns the list of transformation modifiers this
	// provider supports. Modifier names should be lowercase (e.g., "greyscale",
	// "blur", "hue").
	//
	// Returns []string which contains the supported modifier names.
	GetSupportedModifiers() []string
}
