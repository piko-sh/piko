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
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/image/image_dto"
)

// providerCapability describes what features a specific image provider
// supports.
type providerCapability struct {
	// supportedModifiers maps lowercase modifier names to their availability.
	supportedModifiers map[string]bool

	// supportedFormats lists the output formats this provider can produce.
	supportedFormats []string
}

// ValidateTransformationSpec checks a transformation spec DTO for valid values
// and returns a normalised version.
//
// If capabilities is provided, it validates provider-specific formats and
// modifiers.
//
// Takes spec (image_dto.TransformationSpec) which is the transformation
// specification to validate.
// Takes capabilities (map[string]providerCapability) which provides
// provider-specific validation rules, or nil to skip provider validation.
//
// Returns image_dto.TransformationSpec which is the normalised specification.
// Returns error when dimensions, quality, or optional fields are invalid.
func ValidateTransformationSpec(spec image_dto.TransformationSpec, capabilities map[string]providerCapability) (image_dto.TransformationSpec, error) {
	if err := validateDimensions(spec); err != nil {
		return spec, err
	}

	if err := validateQuality(spec); err != nil {
		return spec, err
	}

	normalisedSpec, err := normaliseSpecFields(spec, capabilities)
	if err != nil {
		return spec, err
	}

	if err := validateOptionalFields(normalisedSpec, capabilities); err != nil {
		return spec, err
	}

	return normalisedSpec, nil
}

// normaliseSpecFields validates and normalises format, fit, and background
// fields.
//
// Takes spec (image_dto.TransformationSpec) which contains the fields to
// validate and normalise.
// Takes capabilities (map[string]providerCapability) which defines the
// supported capabilities for each provider.
//
// Returns image_dto.TransformationSpec which contains the normalised fields.
// Returns error when any field validation fails.
func normaliseSpecFields(spec image_dto.TransformationSpec, capabilities map[string]providerCapability) (image_dto.TransformationSpec, error) {
	normalisedFormat, err := validateAndNormaliseFormat(spec.Format, spec.Provider, capabilities)
	if err != nil {
		return spec, err
	}
	spec.Format = normalisedFormat

	normalisedFit, err := validateAndNormaliseFit(spec.Fit)
	if err != nil {
		return spec, err
	}
	spec.Fit = image_dto.FitMode(normalisedFit)

	if spec.Background != "" {
		normalisedBg, err := validateAndNormaliseBackground(spec.Background)
		if err != nil {
			return spec, err
		}
		spec.Background = normalisedBg
	}

	return spec, nil
}

// validateOptionalFields checks optional fields in a transformation spec.
//
// Takes spec (image_dto.TransformationSpec) which contains the fields to
// check.
// Takes capabilities (map[string]providerCapability) which defines provider
// validation rules.
//
// Returns error when any optional field fails validation.
func validateOptionalFields(spec image_dto.TransformationSpec, capabilities map[string]providerCapability) error {
	if spec.AspectRatio != "" {
		if err := validateAspectRatio(spec.AspectRatio); err != nil {
			return fmt.Errorf("validating aspect ratio: %w", err)
		}
	}

	if shouldValidateModifiers(spec, capabilities) {
		if err := validateModifiersForProvider(spec.Modifiers, spec.Provider, capabilities); err != nil {
			return fmt.Errorf("validating modifiers for provider: %w", err)
		}
	}

	if spec.Placeholder != nil {
		if err := validatePlaceholderSpec(spec.Placeholder); err != nil {
			return fmt.Errorf("validating placeholder spec: %w", err)
		}
	}

	return nil
}

// validateDimensions checks that width and height are not negative.
//
// Takes spec (image_dto.TransformationSpec) which holds the dimensions to
// check.
//
// Returns error when width or height is negative.
func validateDimensions(spec image_dto.TransformationSpec) error {
	if spec.Width < 0 {
		return fmt.Errorf("width cannot be negative, got %d", spec.Width)
	}
	if spec.Height < 0 {
		return fmt.Errorf("height cannot be negative, got %d", spec.Height)
	}
	return nil
}

// validateQuality checks that quality is within the valid range.
//
// Takes spec (image_dto.TransformationSpec) which contains the quality value
// to validate.
//
// Returns error when quality is less than minQuality or greater than
// maxQuality.
func validateQuality(spec image_dto.TransformationSpec) error {
	if spec.Quality < minQuality || spec.Quality > maxQuality {
		return fmt.Errorf("quality must be between %d and %d, got %d", minQuality, maxQuality, spec.Quality)
	}
	return nil
}

// validateAndNormaliseFormat checks the output format and returns the
// lowercase version.
//
// Takes format (string) which is the output format to check.
// Takes provider (string) which names the provider to check against.
// Takes capabilities (map[string]providerCapability) which maps provider names
// to their supported features.
//
// Returns string which is the format in lowercase.
// Returns error when the format is not supported or not valid for the provider.
func validateAndNormaliseFormat(format, provider string, capabilities map[string]providerCapability) (string, error) {
	normalisedFormat := strings.ToLower(format)

	if !isFormatSupported(normalisedFormat) {
		return "", fmt.Errorf("unsupported output format: '%s'", format)
	}

	if provider != "" && capabilities != nil {
		if err := validateFormatForProvider(normalisedFormat, provider, capabilities); err != nil {
			return "", err
		}
	}

	return normalisedFormat, nil
}

// isFormatSupported checks if a format is in the standard supported list.
//
// Takes format (string) which is the image format name to check.
//
// Returns bool which is true if the format is supported.
func isFormatSupported(format string) bool {
	switch format {
	case "jpeg", "jpg", "png", "webp", "avif", "gif":
		return true
	default:
		return false
	}
}

// validateAndNormaliseFit checks and normalises the fit mode.
//
// Takes fit (image_dto.FitMode) which specifies the desired fit mode.
//
// Returns string which is the normalised fit mode in lowercase.
// Returns error when the fit mode is not valid.
func validateAndNormaliseFit(fit image_dto.FitMode) (string, error) {
	normalisedFit := strings.ToLower(string(fit))

	if normalisedFit == "" {
		return "contain", nil
	}

	if !isFitModeValid(normalisedFit) {
		return "", fmt.Errorf("unsupported fit mode: '%s' (valid: cover, contain, fill, inside, outside)", fit)
	}

	return normalisedFit, nil
}

// isFitModeValid checks whether a fit mode string is valid.
//
// Takes fit (string) which is the fit mode to check.
//
// Returns bool which is true if fit is one of the valid options (cover,
// contain, fill, inside, outside), false otherwise.
func isFitModeValid(fit string) bool {
	switch fit {
	case "cover", "contain", "fill", "inside", "outside":
		return true
	default:
		return false
	}
}

// validateAndNormaliseBackground checks and normalises a background colour
// string. It validates that the value is a valid hex colour format starting
// with #.
//
// Takes background (string) which is the colour value to check.
//
// Returns string which is the normalised lowercase colour value.
// Returns error when the background does not start with #.
func validateAndNormaliseBackground(background string) (string, error) {
	normalised := strings.ToLower(background)

	if !strings.HasPrefix(normalised, "#") {
		return "", fmt.Errorf("background colour must be a hex colour starting with # (got: '%s')", background)
	}

	return normalised, nil
}

// shouldValidateModifiers checks if modifier validation should run.
// It returns true when the spec has modifiers, a provider is set, and
// capabilities are available.
//
// Takes spec (image_dto.TransformationSpec) which holds the transformation
// details including modifiers and provider.
// Takes capabilities (map[string]providerCapability) which maps provider names
// to their capabilities.
//
// Returns bool which is true when all conditions for validation are met.
func shouldValidateModifiers(spec image_dto.TransformationSpec, capabilities map[string]providerCapability) bool {
	return len(spec.Modifiers) > 0 && spec.Provider != "" && capabilities != nil
}

// validateFormatForProvider checks if a format is supported by a given
// provider.
//
// When the provider is not found in capabilities, returns nil to skip
// validation.
//
// Takes format (string) which specifies the image format to check.
// Takes provider (string) which identifies the provider to check against.
// Takes capabilities (map[string]providerCapability) which holds the supported
// formats for each provider.
//
// Returns error when the format is not supported by the given provider.
func validateFormatForProvider(format, provider string, capabilities map[string]providerCapability) error {
	capability, ok := capabilities[provider]
	if !ok {
		return nil
	}

	format = strings.ToLower(format)
	if slices.Contains(capability.supportedFormats, format) {
		return nil
	}

	return fmt.Errorf("format '%s' is not supported by provider '%s' (supported: %s)",
		format, provider, strings.Join(capability.supportedFormats, ", "))
}

// validateModifiersForProvider checks if all modifiers are supported by the
// selected provider.
//
// When the provider is not found in capabilities, returns nil without
// validation. The unknown provider will fail at execution time.
//
// Takes modifiers (map[string]string) which contains the modifier key-value
// pairs to validate.
// Takes provider (string) which identifies the image processing provider.
// Takes capabilities (map[string]providerCapability) which maps provider names
// to their supported modifiers.
//
// Returns error when any modifier is not supported by the provider.
func validateModifiersForProvider(modifiers map[string]string, provider string, capabilities map[string]providerCapability) error {
	capability, ok := capabilities[provider]
	if !ok {
		return nil
	}

	var unsupported []string
	for modifier := range modifiers {
		modifierLower := strings.ToLower(modifier)
		if !capability.supportedModifiers[modifierLower] {
			unsupported = append(unsupported, modifier)
		}
	}

	if len(unsupported) > 0 {
		return fmt.Errorf("modifiers not supported by provider '%s': %s (try provider='vips' for full feature support)",
			provider, strings.Join(unsupported, ", "))
	}

	return nil
}

// validatePlaceholderSpec checks the placeholder settings are valid.
//
// When the spec is disabled, returns nil without any checks.
//
// Takes spec (*image_dto.PlaceholderSpec) which holds the placeholder
// settings to check.
//
// Returns error when width, height, or blur sigma is negative, or when
// quality is outside the range 0 to maxQuality.
func validatePlaceholderSpec(spec *image_dto.PlaceholderSpec) error {
	if !spec.Enabled {
		return nil
	}

	if spec.Width < 0 {
		return fmt.Errorf("placeholder width cannot be negative, got %d", spec.Width)
	}
	if spec.Height < 0 {
		return fmt.Errorf("placeholder height cannot be negative, got %d", spec.Height)
	}

	if spec.Quality < 0 || spec.Quality > maxQuality {
		return fmt.Errorf("placeholder quality must be between 0 and %d, got %d", maxQuality, spec.Quality)
	}

	if spec.BlurSigma < 0 {
		return fmt.Errorf("placeholder blur sigma cannot be negative, got %.2f", spec.BlurSigma)
	}

	return nil
}

// validateAspectRatio validates the aspect ratio format.
//
// Accepts formats such as "16:9", "4:3", or "1:1" where both width and height
// are positive numbers.
//
// Takes ar (string) which specifies the aspect ratio to validate.
//
// Returns error when the format is invalid, values cannot be parsed as
// numbers, or either dimension is not positive.
func validateAspectRatio(ar string) error {
	parts := strings.Split(ar, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid aspect ratio format: '%s' (expected format: '16:9')", ar)
	}

	width, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Errorf("invalid aspect ratio width: %w", err)
	}

	height, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return fmt.Errorf("invalid aspect ratio height: %w", err)
	}

	if width <= 0 || height <= 0 {
		return fmt.Errorf("aspect ratio dimensions must be positive, got %s", ar)
	}

	return nil
}
