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

package capabilities_functions

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultPlaceholderWidth is the width in pixels for placeholder images.
	defaultPlaceholderWidth = 20

	// defaultPlaceholderHeight is the default height in pixels for placeholder images.
	defaultPlaceholderHeight = 0

	// defaultPlaceholderQuality is the image quality for placeholders.
	defaultPlaceholderQuality = 10

	// defaultPlaceholderBlur is the blur sigma value for placeholder images.
	defaultPlaceholderBlur = 5.0

	// maxQuality is the maximum image quality percentage allowed.
	maxQuality = 100

	// minQuality is the lowest allowed image quality value.
	minQuality = 1
)

// ImageTransform creates a capability function that performs image
// transformations. It depends on the image domain's Service to select the
// provider and execute the transformation via streaming.
//
// Takes imageService (image_domain.Service) which provides the image
// transformation backend.
//
// Returns capabilities_domain.CapabilityFunc which wraps the transformation
// logic for use within the capabilities system.
//
// Supported Parameters:
//
// Standard Transformation Fields:
//   - provider: Transformer to use ("vips", "imaging", "default")
//   - width: Target width in pixels (integer)
//   - height: Target height in pixels (integer)
//   - quality: Compression quality 1-100 (integer)
//   - format: Output format ("jpeg", "png", "webp", "avif", "gif")
//
// Fit & Sizing:
//   - fit: Resize mode ("cover", "contain", "fill", "inside", "outside")
//   - crop: Legacy boolean for backward compatibility
//     (true="cover", false="contain")
//   - aspectratio: Force aspect ratio (e.g., "16:9", "4:3", "1:1")
//   - withoutenlargement: Prevent upscaling ("true", "false")
//
// Visual Options:
//   - background: Background colour for letterboxing (hex, e.g., "#FFFFFF")
//
// Provider-Specific Modifiers:
// Any other parameters are passed as modifiers for advanced transformations:
// - greyscale: "true"
// - blur: "5.0" (sigma value)
// - sharpen: "2.0" (sigma value)
// - rotate: "90", "180", "270"
// - flip: "horizontal", "vertical"
// - brightness: "-100" to "100"
// - contrast: "-100" to "100"
// - saturation: "-100" to "100"
// - hue: "0" to "360" (vips only)
// - gravity: "centre", "entropy", "attention" (smart cropping)
// - focus: alias for gravity
// - tint: "#FF0000" (hex colour, vips only)
// Responsive image generation (sizes, densities) and placeholder
// generation are complex features that should use the dedicated service
// methods (GenerateResponsiveVariants, GeneratePlaceholder) rather than
// this capability.
func ImageTransform(imageService image_domain.Service) capabilities_domain.CapabilityFunc {
	return func(ctx context.Context, inputData io.Reader, params capabilities_domain.CapabilityParams) (io.Reader, error) {
		ctx, span, l := log.Span(ctx, "ImageTransformCapability")
		defer span.End()

		isPlaceholder := parseIsPlaceholder(params)
		spec := buildTransformSpec(params, isPlaceholder)

		validatedSpec, err := image_domain.ValidateTransformationSpec(spec, nil)
		if err != nil {
			l.ReportError(span, err, "Invalid transformation spec provided")
			return nil, fmt.Errorf("invalid transformation spec: %w", err)
		}

		if isPlaceholder {
			return executePlaceholderTransform(ctx, span, imageService, inputData, &validatedSpec)
		}

		return executeImageTransform(ctx, span, imageService, inputData, &validatedSpec)
	}
}

// parseIsPlaceholder checks if the params indicate a placeholder request.
//
// Takes params (CapabilityParams) which contains the parameters to check.
//
// Returns bool which is true when the placeholder key exists and has a truthy
// value (true, yes, or 1).
func parseIsPlaceholder(params capabilities_domain.CapabilityParams) bool {
	placeholderVal, ok := params["placeholder"]
	if !ok {
		return false
	}
	value := strings.ToLower(placeholderVal)
	return value == "true" || value == "yes" || value == "1"
}

// buildTransformSpec creates a TransformationSpec from the capability parameters.
//
// Takes params (capabilities_domain.CapabilityParams) which provides the
// transformation settings to apply.
// Takes isPlaceholder (bool) which indicates whether to build a placeholder spec.
//
// Returns image_dto.TransformationSpec which contains the configured
// transformation settings.
func buildTransformSpec(params capabilities_domain.CapabilityParams, isPlaceholder bool) image_dto.TransformationSpec {
	spec := image_dto.DefaultTransformationSpec()
	spec.Modifiers = map[string]string{}

	if isPlaceholder {
		spec.Placeholder = buildPlaceholderSpec(params)
	}

	parseTransformParams(params, &spec)
	return spec
}

// buildPlaceholderSpec creates a PlaceholderSpec with defaults and overrides
// from params.
//
// Takes params (capabilities_domain.CapabilityParams) which provides the
// capability parameters used to override default placeholder settings.
//
// Returns *image_dto.PlaceholderSpec which is the configured placeholder
// specification.
func buildPlaceholderSpec(params capabilities_domain.CapabilityParams) *image_dto.PlaceholderSpec {
	placeholder := &image_dto.PlaceholderSpec{
		Enabled:   true,
		Width:     defaultPlaceholderWidth,
		Height:    defaultPlaceholderHeight,
		Quality:   defaultPlaceholderQuality,
		BlurSigma: defaultPlaceholderBlur,
	}

	if w, ok := parsePlaceholderWidth(params); ok {
		placeholder.Width = w
	}
	if h, ok := parsePlaceholderHeight(params); ok {
		placeholder.Height = h
	}
	if q, ok := parsePlaceholderQuality(params); ok {
		placeholder.Quality = q
	}
	if b, ok := parsePlaceholderBlur(params); ok {
		placeholder.BlurSigma = b
	}

	return placeholder
}

// parsePlaceholderWidth extracts and validates the placeholder width from the
// given parameters.
//
// Takes params (CapabilityParams) which contains the capability parameters to
// search.
//
// Returns int which is the width value if valid, or 0 if not present or
// invalid.
// Returns bool which indicates whether a valid width was found.
func parsePlaceholderWidth(params capabilities_domain.CapabilityParams) (int, bool) {
	value, ok := params["placeholder-width"]
	if !ok {
		return 0, false
	}
	w, err := strconv.Atoi(value)
	if err != nil || w <= 0 {
		return 0, false
	}
	return w, true
}

// parsePlaceholderHeight extracts and checks the placeholder height from the
// given parameters.
//
// Takes params (CapabilityParams) which contains the parameters to search for
// a placeholder-height value.
//
// Returns int which is the height value if valid, or 0 if not found or invalid.
// Returns bool which shows whether a valid height was found.
func parsePlaceholderHeight(params capabilities_domain.CapabilityParams) (int, bool) {
	value, ok := params["placeholder-height"]
	if !ok {
		return 0, false
	}
	h, err := strconv.Atoi(value)
	if err != nil || h < 0 {
		return 0, false
	}
	return h, true
}

// parsePlaceholderQuality extracts and checks the placeholder quality from
// params.
//
// Takes params (CapabilityParams) which contains the parameters to search for
// placeholder quality.
//
// Returns int which is the quality value if valid (1-100), or 0 if not found
// or invalid.
// Returns bool which shows whether a valid quality was found.
func parsePlaceholderQuality(params capabilities_domain.CapabilityParams) (int, bool) {
	value, ok := params["placeholder-quality"]
	if !ok {
		return 0, false
	}
	q, err := strconv.Atoi(value)
	if err != nil || q < minQuality || q > maxQuality {
		return 0, false
	}
	return q, true
}

// parsePlaceholderBlur extracts and validates the placeholder blur sigma
// from the given parameters.
//
// Takes params (capabilities_domain.CapabilityParams) which contains the
// image transformation settings including the optional placeholder-blur key.
//
// Returns float64 which is the blur sigma value if valid.
// Returns bool which is true when a valid blur sigma was found (>= 0),
// or false if not present or invalid.
func parsePlaceholderBlur(params capabilities_domain.CapabilityParams) (float64, bool) {
	value, ok := params["placeholder-blur"]
	if !ok {
		return 0, false
	}
	b, err := strconv.ParseFloat(value, 64)
	if err != nil || b < 0 {
		return 0, false
	}
	return b, true
}

// transformParamParsers maps lowercased parameter names to setter
// functions that populate fields on a TransformationSpec.
var transformParamParsers = map[string]func(string, *image_dto.TransformationSpec){
	"provider": func(v string, s *image_dto.TransformationSpec) { s.Provider = v },
	"width": func(v string, s *image_dto.TransformationSpec) {
		if w, err := strconv.Atoi(v); err == nil && w >= 0 {
			s.Width = w
		}
	},
	"height": func(v string, s *image_dto.TransformationSpec) {
		if h, err := strconv.Atoi(v); err == nil && h >= 0 {
			s.Height = h
		}
	},
	"quality": func(v string, s *image_dto.TransformationSpec) {
		if q, err := strconv.Atoi(v); err == nil && q >= minQuality && q <= maxQuality {
			s.Quality = q
		}
	},
	"format": func(v string, s *image_dto.TransformationSpec) {
		if v != "original" {
			s.Format = v
		}
	},
	"fit":                 func(v string, s *image_dto.TransformationSpec) { s.Fit = image_dto.FitMode(v) },
	"aspectratio":         func(v string, s *image_dto.TransformationSpec) { s.AspectRatio = v },
	"aspect_ratio":        func(v string, s *image_dto.TransformationSpec) { s.AspectRatio = v },
	"withoutenlargement":  parseWithoutEnlargement,
	"without_enlargement": parseWithoutEnlargement,
	"background":          func(v string, s *image_dto.TransformationSpec) { s.Background = v },
	"bg":                  func(v string, s *image_dto.TransformationSpec) { s.Background = v },
	"placeholder":         func(string, *image_dto.TransformationSpec) {},
	"placeholder-width":   func(string, *image_dto.TransformationSpec) {},
	"placeholder-height":  func(string, *image_dto.TransformationSpec) {},
	"placeholder-quality": func(string, *image_dto.TransformationSpec) {},
	"placeholder-blur":    func(string, *image_dto.TransformationSpec) {},
}

// parseWithoutEnlargement parses a boolean value and sets the
// WithoutEnlargement field on the spec.
//
// Takes v (string) which is the raw boolean string to parse.
// Takes s (*image_dto.TransformationSpec) which receives the
// parsed WithoutEnlargement value.
func parseWithoutEnlargement(v string, s *image_dto.TransformationSpec) {
	if we, err := strconv.ParseBool(v); err == nil {
		s.WithoutEnlargement = we
	}
}

// parseTransformParams populates the spec from the generic params map.
//
// Dispatches through the transformParamParsers lookup table. Unrecognised keys
// are stored in the Modifiers map for provider-specific handling.
//
// Takes params (capabilities_domain.CapabilityParams) which contains the
// key-value pairs to parse.
// Takes spec (*image_dto.TransformationSpec) which receives the parsed values.
func parseTransformParams(params capabilities_domain.CapabilityParams, spec *image_dto.TransformationSpec) {
	for key, value := range params {
		if parser, ok := transformParamParsers[strings.ToLower(key)]; ok {
			parser(value, spec)
		} else {
			spec.Modifiers[key] = value
		}
	}
}

// executePlaceholderTransform creates a low quality image placeholder (LQIP).
//
// Takes span (trace.Span) which receives error reports on failure.
// Takes imageService (image_domain.Service) which creates the placeholder image.
// Takes inputData (io.Reader) which provides the source image data.
// Takes spec (*image_dto.TransformationSpec) which defines the placeholder
// settings.
//
// Returns io.Reader which contains the created data URL string.
// Returns error when the image service fails to create the placeholder.
func executePlaceholderTransform(
	ctx context.Context,
	span trace.Span,
	imageService image_domain.Service,
	inputData io.Reader,
	spec *image_dto.TransformationSpec,
) (io.Reader, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Generating image placeholder (LQIP)",
		logger_domain.String("provider", spec.Provider),
		logger_domain.Int("width", spec.Placeholder.Width),
		logger_domain.Int("quality", spec.Placeholder.Quality),
	)

	dataURL, err := imageService.GeneratePlaceholder(ctx, inputData, *spec)
	if err != nil {
		l.ReportError(span, err, "Image service failed to generate placeholder")
		return nil, fmt.Errorf("failed to generate placeholder: %w", err)
	}

	return strings.NewReader(dataURL), nil
}

// executeImageTransform runs a standard image transformation.
//
// Takes span (trace.Span) which receives error reports on failure.
// Takes imageService (image_domain.Service) which performs the transformation.
// Takes inputData (io.Reader) which provides the source image data.
// Takes spec (*image_dto.TransformationSpec) which defines the transformation.
//
// Returns io.Reader which provides the transformed image data.
// Returns error when the image service fails to start the transform stream.
func executeImageTransform(
	ctx context.Context,
	span trace.Span,
	imageService image_domain.Service,
	inputData io.Reader,
	spec *image_dto.TransformationSpec,
) (io.Reader, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Executing image transformation via streaming service",
		logger_domain.String("spec", spec.String()),
		logger_domain.String("provider", spec.Provider),
		logger_domain.String("fit", string(spec.Fit)),
	)

	result, err := imageService.TransformStream(ctx, inputData, *spec)
	if err != nil {
		l.ReportError(span, err, "Image service failed to initiate transform stream")
		return nil, fmt.Errorf("failed to initiate image transform stream: %w", err)
	}
	return result.Body, nil
}
