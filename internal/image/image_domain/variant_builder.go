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
	"fmt"

	"piko.sh/piko/internal/image/image_dto"
)

// VariantBuilder provides a fluent API for defining image variant specifications.
// It creates reusable TransformationSpec objects that can be registered as
// predefined variants or used directly for transformations.
type VariantBuilder struct {
	// spec holds the transformation settings being built.
	spec image_dto.TransformationSpec
}

// Width sets the target width in pixels.
// Set to 0 to preserve aspect ratio based on height.
//
// Takes px (int) which specifies the target width.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Width(px int) *VariantBuilder {
	b.spec.Width = px
	return b
}

// Height sets the target height in pixels.
// Set to 0 to preserve aspect ratio based on width.
//
// Takes px (int) which specifies the target height.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Height(px int) *VariantBuilder {
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
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Size(width, height int) *VariantBuilder {
	b.spec.Width = width
	b.spec.Height = height
	return b
}

// MaxWidth sets the width while leaving height at 0 to preserve aspect ratio.
// The image will be scaled to fit within the specified width.
//
// Takes px (int) which specifies the maximum width.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) MaxWidth(px int) *VariantBuilder {
	b.spec.Width = px
	b.spec.Height = 0
	return b
}

// MaxHeight sets the height while leaving width at 0 to preserve aspect ratio.
// The image will be scaled to fit within the specified height.
//
// Takes px (int) which specifies the maximum height.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) MaxHeight(px int) *VariantBuilder {
	b.spec.Width = 0
	b.spec.Height = px
	return b
}

// Format sets the output format for the image.
// Supported formats: "jpeg", "jpg", "png", "webp", "avif", "gif".
//
// Takes format (string) which specifies the output format.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Format(format string) *VariantBuilder {
	b.spec.Format = format
	return b
}

// Quality sets the compression quality (1-100). Higher values produce larger
// files with better quality, and 80 is a good default for JPEG, WebP, and
// AVIF formats.
//
// Takes q (int) which specifies the quality value.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Quality(q int) *VariantBuilder {
	b.spec.Quality = q
	return b
}

// Fit sets the resize fitting mode.
// Valid values: "cover", "contain", "fill", "inside", "outside".
//
//   - "cover": Resize to fill dimensions, cropping excess
//   - "contain": Resize to fit within dimensions, letterboxing if needed
//   - "fill": Stretch to exact dimensions, ignoring aspect ratio
//   - "inside": Resize to be <= dimensions, preserving aspect ratio
//   - "outside": Resize to be >= dimensions, preserving aspect ratio
//
// Takes fit (image_dto.FitMode) which specifies the fit mode.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Fit(fit image_dto.FitMode) *VariantBuilder {
	b.spec.Fit = fit
	return b
}

// Cover is a shorthand for Fit(image_dto.FitCover).
// The image will be resized to fill the dimensions, cropping any excess.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Cover() *VariantBuilder {
	b.spec.Fit = image_dto.FitCover
	return b
}

// Contain is a shorthand for Fit(image_dto.FitContain).
// The image will be resized to fit within the dimensions, with
// letterboxing if the aspect ratio doesn't match.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Contain() *VariantBuilder {
	b.spec.Fit = image_dto.FitContain
	return b
}

// Fill is a shorthand for Fit(image_dto.FitFill).
// The image will be stretched to exact dimensions, potentially distorting
// the aspect ratio.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Fill() *VariantBuilder {
	b.spec.Fit = image_dto.FitFill
	return b
}

// Inside is a shorthand for Fit(image_dto.FitInside).
// The image will be resized to be at most the specified dimensions,
// preserving aspect ratio.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Inside() *VariantBuilder {
	b.spec.Fit = image_dto.FitInside
	return b
}

// Outside is a shorthand for Fit(image_dto.FitOutside).
// The image will be resized to be at least the specified dimensions,
// preserving aspect ratio.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Outside() *VariantBuilder {
	b.spec.Fit = image_dto.FitOutside
	return b
}

// WithoutEnlargement prevents images from being scaled up beyond their
// original size. When enabled, images smaller than the target size
// keep their original dimensions.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) WithoutEnlargement() *VariantBuilder {
	b.spec.WithoutEnlargement = true
	return b
}

// Background sets the hex colour for letterboxing or transparency fill.
// Format: "#RRGGBB" (e.g., "#FFFFFF" for white).
//
// Takes hex (string) which specifies the background colour.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Background(hex string) *VariantBuilder {
	b.spec.Background = hex
	return b
}

// AspectRatio sets a specific aspect ratio for the variant, where one dimension
// can be omitted and will be calculated automatically.
// Format: "width:height" (e.g., "16:9", "4:3", "1:1").
//
// Takes ratio (string) which specifies the aspect ratio.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) AspectRatio(ratio string) *VariantBuilder {
	b.spec.AspectRatio = ratio
	return b
}

// Provider sets the image processing provider to use.
// If not set, the service's default provider will be used.
//
// Takes name (string) which identifies the provider (e.g., "vips", "imaging").
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Provider(name string) *VariantBuilder {
	b.spec.Provider = name
	return b
}

// WithModifier adds a provider-specific modifier.
// Modifiers are passed through to the underlying image library.
//
// Takes key (string) which identifies the modifier.
// Takes value (string) which specifies the modifier value.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) WithModifier(key, value string) *VariantBuilder {
	if b.spec.Modifiers == nil {
		b.spec.Modifiers = make(map[string]string)
	}
	b.spec.Modifiers[key] = value
	return b
}

// Blur adds a blur modifier with the specified sigma value.
// This is a convenience method for common blur operations.
//
// Takes sigma (float64) which specifies the blur intensity.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Blur(sigma float64) *VariantBuilder {
	if b.spec.Modifiers == nil {
		b.spec.Modifiers = make(map[string]string)
	}
	b.spec.Modifiers["blur"] = formatFloat(sigma)
	return b
}

// Greyscale applies a greyscale filter to the image.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) Greyscale() *VariantBuilder {
	if b.spec.Modifiers == nil {
		b.spec.Modifiers = make(map[string]string)
	}
	b.spec.Modifiers["greyscale"] = "true"
	return b
}

// FromSpec copies values from an existing TransformationSpec. Use it to extend
// or modify an existing specification.
//
// Takes spec (image_dto.TransformationSpec) which provides the base values.
//
// Returns *VariantBuilder which allows method chaining.
func (b *VariantBuilder) FromSpec(spec image_dto.TransformationSpec) *VariantBuilder {
	b.spec = spec
	if b.spec.Modifiers == nil {
		b.spec.Modifiers = make(map[string]string)
	}
	return b
}

// Build returns the configured TransformationSpec.
// This is the terminal method that produces the final specification.
//
// Returns image_dto.TransformationSpec which contains the configured settings.
func (b *VariantBuilder) Build() image_dto.TransformationSpec {
	return b.spec
}

// Variant creates a new variant builder with default values.
//
// Returns *VariantBuilder which provides a fluent interface for building
// a TransformationSpec.
func Variant() *VariantBuilder {
	return &VariantBuilder{
		spec: image_dto.DefaultTransformationSpec(),
	}
}

// formatFloat formats a float with one decimal place.
//
// Takes f (float64) which is the value to format.
//
// Returns string which is the formatted number with one decimal place.
func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}
