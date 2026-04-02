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
	"errors"
	"fmt"
	"strings"
	"time"

	"piko.sh/piko/internal/image/image_dto"
)

// ImageConfig holds the settings produced by ImageConfigBuilder.
// Pass this to piko.WithImage to set up the image hexagon.
type ImageConfig struct {
	// Providers maps provider names to their transformer implementations.
	Providers map[string]TransformerPort

	// PredefinedVariants maps variant names to their transformation settings.
	PredefinedVariants map[string]image_dto.TransformationSpec

	// DefaultProvider is the name of the provider to use when none is specified.
	DefaultProvider string

	// ServiceConfig holds security and behaviour settings.
	ServiceConfig ServiceConfig
}

// GetVariant retrieves a predefined variant by name.
//
// Takes name (string) which identifies the variant.
//
// Returns image_dto.TransformationSpec which contains the variant settings.
// Returns bool which indicates whether the variant was found.
func (c *ImageConfig) GetVariant(name string) (image_dto.TransformationSpec, bool) {
	if c.PredefinedVariants == nil {
		return image_dto.TransformationSpec{}, false
	}
	spec, ok := c.PredefinedVariants[name]
	return spec, ok
}

// ImageConfigBuilder provides a fluent interface for setting up image service
// options. It creates an ImageConfig that can be passed to piko.WithImage().
//
// Usage:
// config := image_domain.Image().
//
//	Provider("vips", vipsProvider).
//	MaxFileSizeMB(50).
//	DefaultQuality(85).
//	WithVariant("thumb", image_domain.Variant().Size(200, 200).Cover().Build()).
//	Build()
type ImageConfigBuilder struct {
	// providers maps provider names to their transformer implementations.
	providers map[string]TransformerPort

	// predefinedVariants holds named transformation specs that define image variants.
	predefinedVariants map[string]image_dto.TransformationSpec

	// defaultProvider is the name of the provider to use when none is specified.
	defaultProvider string

	// errs collects validation errors during builder method calls.
	errs []error

	// config holds the image configuration being built.
	config ServiceConfig
}

// Provider registers an image transformer with a name.
// The first provider registered becomes the default unless DefaultProvider
// is called.
//
// Takes name (string) which identifies the provider.
// Takes transformer (TransformerPort) which provides the image processing.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) Provider(name string, transformer TransformerPort) *ImageConfigBuilder {
	if name == "" {
		b.errs = append(b.errs, errProviderNameEmpty)
		return b
	}
	if transformer == nil {
		b.errs = append(b.errs, errTransformerNil)
		return b
	}
	b.providers[name] = transformer
	if b.defaultProvider == "" {
		b.defaultProvider = name
	}
	return b
}

// DefaultProvider sets which provider to use when none is specified.
// This overrides the automatic selection of the first registered provider.
//
// Takes name (string) which identifies the default provider.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) DefaultProvider(name string) *ImageConfigBuilder {
	b.defaultProvider = name
	return b
}

// MaxDimensions sets the maximum allowed width and height for images.
//
// Takes width (int) which specifies the maximum width in pixels.
// Takes height (int) which specifies the maximum height in pixels.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) MaxDimensions(width, height int) *ImageConfigBuilder {
	b.config.MaxImageWidth = width
	b.config.MaxImageHeight = height
	return b
}

// MaxPixels sets the maximum allowed total pixel count (width * height).
// This prevents memory exhaustion from extremely large images.
//
// Takes pixels (int64) which specifies the maximum pixel count.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) MaxPixels(pixels int64) *ImageConfigBuilder {
	b.config.MaxImagePixels = pixels
	return b
}

// MaxFileSizeMB sets the maximum allowed input file size in megabytes.
//
// Takes mb (int) which specifies the maximum size in MB.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) MaxFileSizeMB(mb int) *ImageConfigBuilder {
	b.config.MaxFileSizeBytes = int64(mb) * bytesPerMB
	return b
}

// MaxFileSizeBytes sets the maximum allowed input file size in bytes.
//
// Takes bytes (int64) which specifies the maximum size in bytes.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) MaxFileSizeBytes(bytes int64) *ImageConfigBuilder {
	b.config.MaxFileSizeBytes = bytes
	return b
}

// TransformTimeout sets the maximum time allowed for a single transformation.
//
// Takes d (time.Duration) which specifies the timeout duration.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) TransformTimeout(d time.Duration) *ImageConfigBuilder {
	b.config.TransformTimeout = d
	return b
}

// AllowedFormats sets the list of permitted output formats.
// If not called, all standard formats are allowed.
//
// Takes formats (...string) which specifies the allowed formats.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) AllowedFormats(formats ...string) *ImageConfigBuilder {
	b.config.AllowedFormats = formats
	return b
}

// DefaultQuality sets the default compression quality (1-100).
// This is used when a transformation doesn't specify quality.
//
// Takes q (int) which specifies the quality value.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) DefaultQuality(q int) *ImageConfigBuilder {
	if q < minQuality || q > maxQuality {
		b.errs = append(b.errs, errQualityOutOfRange)
		return b
	}
	return b
}

// WithVariant registers a predefined variant specification.
// These variants can be referenced by name during transformations.
//
// Takes name (string) which identifies the variant.
// Takes spec (image_dto.TransformationSpec) which defines the variant.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) WithVariant(name string, spec image_dto.TransformationSpec) *ImageConfigBuilder {
	if name == "" {
		b.errs = append(b.errs, errVariantNameEmpty)
		return b
	}
	b.predefinedVariants[name] = spec
	return b
}

// WithVariantBuilder registers a predefined variant using a VariantBuilder.
// This is a convenience method that calls Build() on the builder.
//
// Takes name (string) which identifies the variant.
// Takes builder (*VariantBuilder) which provides the variant configuration.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) WithVariantBuilder(name string, builder *VariantBuilder) *ImageConfigBuilder {
	if builder == nil {
		b.errs = append(b.errs, errors.New("variant builder cannot be nil"))
		return b
	}
	return b.WithVariant(name, builder.Build())
}

// WithFallbackIcon sets a fallback icon for non-image MIME types.
// When a transformation is requested for a file that isn't an image,
// the fallback icon is returned instead.
//
// Takes mimePrefix (string) which specifies the MIME type prefix to match.
// Takes iconPath (string) which specifies the path to the icon file.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) WithFallbackIcon(mimePrefix, iconPath string) *ImageConfigBuilder {
	if b.config.FallbackIconPaths == nil {
		b.config.FallbackIconPaths = make(map[string]string)
	}
	b.config.FallbackIconPaths[mimePrefix] = iconPath
	return b
}

// FromConfig copies settings from an existing ServiceConfig. Use it to extend
// or modify an existing configuration.
//
// Takes config (ServiceConfig) which provides the base settings.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) FromConfig(config ServiceConfig) *ImageConfigBuilder {
	b.config = config
	return b
}

// FromDefaults resets the configuration to default values and includes
// default predefined variants.
//
// Returns *ImageConfigBuilder which allows method chaining.
func (b *ImageConfigBuilder) FromDefaults() *ImageConfigBuilder {
	b.config = DefaultServiceConfig()
	b.predefinedVariants = DefaultPredefinedVariants()
	return b
}

// Build validates and returns the ImageConfig.
// Returns an error if validation fails.
//
// Returns *ImageConfig which contains the complete configuration.
// Returns error when validation fails.
func (b *ImageConfigBuilder) Build() (*ImageConfig, error) {
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("building image config: %w", err)
	}

	return &ImageConfig{
		Providers:          b.providers,
		DefaultProvider:    b.defaultProvider,
		ServiceConfig:      b.config,
		PredefinedVariants: b.predefinedVariants,
	}, nil
}

// MustBuild validates and returns the ImageConfig, panicking on error.
// Use it in init() functions or when configuration errors should halt
// startup.
//
// Returns *ImageConfig which contains the complete configuration.
//
// Panics if the configuration is invalid.
func (b *ImageConfigBuilder) MustBuild() *ImageConfig {
	config, err := b.Build()
	if err != nil {
		panic("image config build failed: " + err.Error())
	}
	return config
}

// validate checks the configuration for errors.
//
// Returns error when any builder error exists, no providers are registered,
// default provider is not set or not registered, or validation of config or
// variants fails.
func (b *ImageConfigBuilder) validate() error {
	if len(b.errs) > 0 {
		return b.errs[0]
	}

	if len(b.providers) == 0 {
		return errNoProviders
	}

	if b.defaultProvider == "" {
		return errDefaultProviderNotSet
	}

	if _, ok := b.providers[b.defaultProvider]; !ok {
		return errors.New("default provider '" + b.defaultProvider + "' is not registered")
	}

	if err := b.validateConfig(); err != nil {
		return fmt.Errorf("validating service config: %w", err)
	}

	if err := b.validateVariants(); err != nil {
		return fmt.Errorf("validating predefined variants: %w", err)
	}

	return nil
}

// validateConfig checks the ServiceConfig values.
//
// Returns error when any configuration value is negative or an invalid format
// is specified.
func (b *ImageConfigBuilder) validateConfig() error {
	if b.config.MaxImageWidth < 0 {
		return errMaxWidthNegative
	}
	if b.config.MaxImageHeight < 0 {
		return errMaxHeightNegative
	}
	if b.config.MaxImagePixels < 0 {
		return errMaxPixelsNegative
	}
	if b.config.MaxFileSizeBytes < 0 {
		return errMaxFileSizeNegative
	}
	if b.config.TransformTimeout < 0 {
		return errTimeoutNegative
	}

	for _, format := range b.config.AllowedFormats {
		if !isValidFormat(format) {
			return errors.New("invalid format in allowed formats: " + format)
		}
	}

	return nil
}

// validateVariants checks all predefined variants.
//
// Returns error when a variant has invalid dimensions, quality, or format.
func (b *ImageConfigBuilder) validateVariants() error {
	for name := range b.predefinedVariants {
		spec := b.predefinedVariants[name]
		if spec.Width < 0 {
			return errors.New("variant '" + name + "' has negative width")
		}
		if spec.Height < 0 {
			return errors.New("variant '" + name + "' has negative height")
		}
		if spec.Quality < 0 || spec.Quality > maxQuality {
			return errors.New("variant '" + name + "' has invalid quality")
		}
		if spec.Format != "" && !isValidFormat(spec.Format) {
			return errors.New("variant '" + name + "' has invalid format: " + spec.Format)
		}
	}
	return nil
}

// Image creates a new image configuration builder with sensible defaults.
//
// Returns *ImageConfigBuilder which provides a fluent interface for
// setting up the image service.
func Image() *ImageConfigBuilder {
	return &ImageConfigBuilder{
		providers:          make(map[string]TransformerPort),
		defaultProvider:    "",
		config:             DefaultServiceConfig(),
		predefinedVariants: make(map[string]image_dto.TransformationSpec),
		errs:               nil,
	}
}

// DefaultPredefinedVariants returns a map of sensible default variants.
// These are included when FromDefaults() is called.
//
// Returns map[string]image_dto.TransformationSpec which contains the
// default variants.
func DefaultPredefinedVariants() map[string]image_dto.TransformationSpec {
	return map[string]image_dto.TransformationSpec{
		"thumb_100": {
			Width:   variantSizeThumb100,
			Height:  variantSizeThumb100,
			Format:  variantFormatWebP,
			Quality: variantQualityLow,
			Fit:     variantFitCover,
		},
		"thumb_200": {
			Width:   variantSizeThumb200,
			Height:  variantSizeThumb200,
			Format:  variantFormatWebP,
			Quality: variantQualityMedium,
			Fit:     variantFitCover,
		},
		"thumb_400": {
			Width:   variantSizeThumb400,
			Height:  variantSizeThumb400,
			Format:  variantFormatWebP,
			Quality: variantQualityHigh,
			Fit:     variantFitCover,
		},
		"preview_800": {
			Width:   variantSizePreview800,
			Height:  0,
			Format:  variantFormatWebP,
			Quality: variantQualityHigh,
			Fit:     variantFitContain,
		},
		"lqip": {
			Width:   variantSizeLQIP,
			Height:  variantSizeLQIP,
			Format:  variantFormatWebP,
			Quality: variantQualityLQIP,
			Fit:     variantFitCover,
		},
	}
}

// isValidFormat checks if a format string is a valid image format.
//
// Takes format (string) which is the format name to validate.
//
// Returns bool which is true if the format is a supported image format.
func isValidFormat(format string) bool {
	switch strings.ToLower(format) {
	case "jpeg", "jpg", "png", "webp", "avif", "gif":
		return true
	default:
		return false
	}
}
