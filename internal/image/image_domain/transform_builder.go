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
	"errors"
	"fmt"
	"io"

	"piko.sh/piko/internal/image/image_dto"
)

// TransformBuilder provides a fluent API for performing image transformations.
// It wraps the image service and provides a clean interface for runtime
// operations, similar to storage's UploadBuilder.
//
// Usage:
// result, err := service.Transform(reader).
//
//	Size(200, 200).
//	Format("webp").
//	Cover().
//	Do(ctx)
type TransformBuilder struct {
	// service provides the image transformation operations.
	service Service

	// input is the source image data stream.
	input io.Reader

	// predefinedVariants stores named transform presets for UseVariant lookups.
	predefinedVariants map[string]image_dto.TransformationSpec

	// spec holds the transformation settings being built.
	spec image_dto.TransformationSpec
}

// Transform creates a new transform builder for the given input.
// This is the entry point for performing image transformations.
//
// Takes input (io.Reader) which provides the source image data.
//
// Returns *TransformBuilder which provides a fluent interface for
// configuring and executing the transformation.
func (s *service) Transform(input io.Reader) *TransformBuilder {
	return &TransformBuilder{
		service:            s,
		input:              input,
		spec:               image_dto.DefaultTransformationSpec(),
		predefinedVariants: nil,
	}
}

// WithPredefinedVariants sets the available predefined variants for UseVariant
// lookups.
//
// Takes variants (map[string]image_dto.TransformationSpec) which provides the
// available variants.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) WithPredefinedVariants(variants map[string]image_dto.TransformationSpec) *TransformBuilder {
	b.predefinedVariants = variants
	return b
}

// Width sets the target width in pixels.
// Set to 0 to preserve aspect ratio based on height.
//
// Takes px (int) which specifies the target width.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Width(px int) *TransformBuilder {
	b.spec.Width = px
	return b
}

// Height sets the target height in pixels.
// Set to 0 to preserve aspect ratio based on width.
//
// Takes px (int) which specifies the target height.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Height(px int) *TransformBuilder {
	b.spec.Height = px
	return b
}

// Size sets both width and height in pixels.
// If both are specified with a fit mode, the image will be resized
// according to the fit mode's behaviour.
//
// Takes width (int) which specifies the target width.
// Takes height (int) which specifies the target height.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Size(width, height int) *TransformBuilder {
	b.spec.Width = width
	b.spec.Height = height
	return b
}

// MaxWidth sets the width while leaving height at 0 to preserve aspect ratio.
// The image will be scaled to fit within the specified width.
//
// Takes px (int) which specifies the maximum width.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) MaxWidth(px int) *TransformBuilder {
	b.spec.Width = px
	b.spec.Height = 0
	return b
}

// MaxHeight sets the height while leaving width at 0 to preserve aspect ratio.
// The image will be scaled to fit within the specified height.
//
// Takes px (int) which specifies the maximum height.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) MaxHeight(px int) *TransformBuilder {
	b.spec.Width = 0
	b.spec.Height = px
	return b
}

// Format sets the output format for the image.
// Supported formats: "jpeg", "jpg", "png", "webp", "avif", "gif".
//
// Takes format (string) which specifies the output format.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Format(format string) *TransformBuilder {
	b.spec.Format = format
	return b
}

// Quality sets the compression quality for the output image.
//
// Values range from 1 to 100, where higher values produce larger files with
// better quality. For JPEG, WebP, and AVIF formats, 80 is a good default.
//
// Takes q (int) which specifies the quality value between 1 and 100.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Quality(q int) *TransformBuilder {
	b.spec.Quality = q
	return b
}

// Fit sets the resize fitting mode.
//
// Takes fit (image_dto.FitMode) which specifies the fit mode.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Fit(fit image_dto.FitMode) *TransformBuilder {
	b.spec.Fit = fit
	return b
}

// Cover is a shorthand for Fit(image_dto.FitCover).
// The image will be resized to fill the dimensions, cropping any excess.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Cover() *TransformBuilder {
	b.spec.Fit = image_dto.FitCover
	return b
}

// Contain is a shorthand for Fit(image_dto.FitContain).
// The image will be resized to fit within the dimensions.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Contain() *TransformBuilder {
	b.spec.Fit = image_dto.FitContain
	return b
}

// Fill is a shorthand for Fit(image_dto.FitFill).
// The image will be stretched to exact dimensions.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Fill() *TransformBuilder {
	b.spec.Fit = image_dto.FitFill
	return b
}

// Inside is a shorthand for Fit(image_dto.FitInside).
// The image will be resized to be at most the specified dimensions.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Inside() *TransformBuilder {
	b.spec.Fit = image_dto.FitInside
	return b
}

// Outside is a shorthand for Fit(image_dto.FitOutside).
// The image will be resized to be at least the specified dimensions.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Outside() *TransformBuilder {
	b.spec.Fit = image_dto.FitOutside
	return b
}

// WithoutEnlargement prevents images from being scaled up beyond their
// original size.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) WithoutEnlargement() *TransformBuilder {
	b.spec.WithoutEnlargement = true
	return b
}

// Background sets the hex colour for letterboxing or transparency fill.
// Format: "#RRGGBB" (e.g., "#FFFFFF" for white).
//
// Takes hex (string) which specifies the background colour.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Background(hex string) *TransformBuilder {
	b.spec.Background = hex
	return b
}

// AspectRatio forces a specific aspect ratio.
// Format: "width:height" (e.g., "16:9", "4:3", "1:1").
//
// Takes ratio (string) which specifies the aspect ratio.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) AspectRatio(ratio string) *TransformBuilder {
	b.spec.AspectRatio = ratio
	return b
}

// Provider sets the image processing provider to use.
// If not set, the service's default provider will be used.
//
// Takes name (string) which identifies the provider.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Provider(name string) *TransformBuilder {
	b.spec.Provider = name
	return b
}

// WithModifier adds a provider-specific modifier.
//
// Takes key (string) which identifies the modifier.
// Takes value (string) which specifies the modifier value.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) WithModifier(key, value string) *TransformBuilder {
	if b.spec.Modifiers == nil {
		b.spec.Modifiers = make(map[string]string)
	}
	b.spec.Modifiers[key] = value
	return b
}

// Blur adds a blur modifier with the specified sigma value.
//
// Takes sigma (float64) which specifies the blur intensity.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Blur(sigma float64) *TransformBuilder {
	if b.spec.Modifiers == nil {
		b.spec.Modifiers = make(map[string]string)
	}
	b.spec.Modifiers["blur"] = formatFloat(sigma)
	return b
}

// Greyscale applies a greyscale filter to the image.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) Greyscale() *TransformBuilder {
	if b.spec.Modifiers == nil {
		b.spec.Modifiers = make(map[string]string)
	}
	b.spec.Modifiers["greyscale"] = "true"
	return b
}

// UseVariant applies a predefined variant's settings to this transformation.
// The variant must have been registered with WithPredefinedVariants or set up
// in the image service.
//
// Takes name (string) which identifies the predefined variant to apply.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) UseVariant(name string) *TransformBuilder {
	if b.predefinedVariants == nil {
		return b
	}
	if spec, ok := b.predefinedVariants[name]; ok {
		b.spec = spec
	}
	return b
}

// FromSpec copies all settings from an existing TransformationSpec. Use it to
// build on an existing specification.
//
// Takes spec (image_dto.TransformationSpec) which provides the base settings.
//
// Returns *TransformBuilder which allows method chaining.
func (b *TransformBuilder) FromSpec(spec image_dto.TransformationSpec) *TransformBuilder {
	b.spec = spec
	if b.spec.Modifiers == nil {
		b.spec.Modifiers = make(map[string]string)
	}
	return b
}

// Spec returns the current TransformationSpec without executing.
// Use this to inspect or pass the spec to other functions.
//
// Returns image_dto.TransformationSpec which contains the current settings.
func (b *TransformBuilder) Spec() image_dto.TransformationSpec {
	return b.spec
}

// Do performs the transformation and returns the result.
// This is the terminal method that executes the configured transformation.
//
// Returns *image_dto.TransformedImageResult which contains the transformed
// image stream and metadata.
// Returns error when the transformation fails.
func (b *TransformBuilder) Do(ctx context.Context) (*image_dto.TransformedImageResult, error) {
	if b.input == nil {
		return nil, errors.New("no input provided for transformation")
	}
	return b.service.TransformStream(ctx, b.input, b.spec)
}

// DoToWriter performs the transformation and writes the result to the
// provided writer. Use it to write directly to a file or buffer.
//
// Takes w (io.Writer) which receives the transformed image data.
//
// Returns error when the transformation fails or writing fails.
func (b *TransformBuilder) DoToWriter(ctx context.Context, w io.Writer) error {
	result, err := b.Do(ctx)
	if err != nil {
		return fmt.Errorf("executing transformation: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, result.Body)
		_ = result.Body.Close()
	}()

	if _, err = io.Copy(w, result.Body); err != nil {
		return fmt.Errorf("writing transformation result: %w", err)
	}
	return nil
}
