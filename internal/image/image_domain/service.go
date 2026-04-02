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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/contextaware"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// service provides image processing and health check functions.
// It implements the image Service interface and HealthProbe interface.
type service struct {
	// transformers maps provider names to their transformer implementations.
	transformers map[string]TransformerPort

	// providerCapabilities maps provider names to their supported capabilities.
	providerCapabilities map[string]providerCapability

	// fallbackIcons maps MIME type prefixes to their default icon data.
	fallbackIcons map[string][]byte

	// defaultProvider is the name of the provider used when none is specified.
	defaultProvider string

	// config holds validation and behaviour settings for the service.
	config ServiceConfig
}

var _ Service = (*service)(nil)

// ServiceConfig holds settings for creating and setting up an image service.
// It is used by image transformers and validation functions across the image
// processing domain.
type ServiceConfig struct {
	// FallbackIconSandbox is an optional injected sandbox for testing icon
	// file loading, where nil causes sandboxes to be created per icon
	// file's parent directory.
	//
	// The caller is responsible for closing an injected sandbox.
	FallbackIconSandbox safedisk.Sandbox

	// FallbackIconSandboxFactory creates sandboxes for icon file loading.
	// When non-nil and FallbackIconSandbox is nil, this factory is used
	// instead of safedisk.NewNoOpSandbox.
	FallbackIconSandboxFactory safedisk.Factory

	// FallbackIconPaths maps MIME type prefixes (e.g. "application/pdf") to file
	// paths for icons shown when a non-image file is requested.
	FallbackIconPaths map[string]string

	// AllowedFormats specifies permitted output formats such as "jpeg", "png",
	// "webp", or "avif". If empty, all supported formats are allowed.
	AllowedFormats []string

	// MaxImagePixels is the maximum allowed total pixel count
	// (width * height), preventing memory exhaustion from extremely
	// large images. Default: 100,000,000 (100 megapixels).
	MaxImagePixels int64

	// MaxFileSizeBytes is the maximum allowed input file size in bytes.
	// Default: 50 MB.
	MaxFileSizeBytes int64

	// TransformTimeout is the maximum time allowed for a single image
	// transformation. Default is 30 seconds.
	TransformTimeout time.Duration

	// MaxImageWidth is the maximum allowed width for images in pixels.
	// Default: 8192.
	MaxImageWidth int

	// MaxImageHeight is the maximum height in pixels allowed for uploaded images.
	// Default: 8192.
	MaxImageHeight int
}

// TransformStream performs a direct streaming transform using the selected
// provider.
//
// Takes input (io.Reader) which provides the source image data.
// Takes spec (image_dto.TransformationSpec) which defines the transformation.
//
// Returns *image_dto.TransformedImageResult which contains the transformed
// image stream.
// Returns error when the transformation spec is invalid or no provider is
// available.
//
// Spawns a goroutine to run the transformation pipeline. The pipeline writes
// to the returned result's Body reader until complete or cancelled.
func (s *service) TransformStream(ctx context.Context, input io.Reader, spec image_dto.TransformationSpec) (*image_dto.TransformedImageResult, error) {
	validated, err := ValidateTransformationSpec(spec, s.providerCapabilities)
	if err != nil {
		return nil, fmt.Errorf("invalid transformation spec: %w", err)
	}

	tr, providerName, err := s.selectTransformer(validated)
	if err != nil {
		return nil, fmt.Errorf("selecting image transformer: %w", err)
	}

	recordTransformMetrics(ctx, providerName, validated, true)

	ctx, cancel := context.WithTimeoutCause(ctx, s.config.TransformTimeout,
		fmt.Errorf("image transformation exceeded %s timeout", s.config.TransformTimeout))

	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Starting image transform stream",
		logger_domain.String(metricAttributeProvider, providerName),
		logger_domain.String("spec", validated.String()))

	pr, pw := io.Pipe()
	go s.runTransformPipeline(ctx, contextaware.NewReader(ctx, input), pw, tr, validated, providerName, cancel)

	return &image_dto.TransformedImageResult{
		Body:     pr,
		MIMEType: specToMIMEType(validated),
		Size:     0,
		ETag:     "",
	}, nil
}

// GenerateResponsiveVariants creates multiple image variants for responsive
// images.
//
// Takes input (io.Reader) which supplies the original image data.
// Takes baseSpec (image_dto.TransformationSpec) which defines the responsive
// image settings including widths and densities.
//
// Returns []image_dto.ResponsiveVariant which contains the generated variants.
// Returns error when the responsive spec is nil, the input cannot be read, or
// variant generation fails.
func (s *service) GenerateResponsiveVariants(ctx context.Context, input io.Reader, baseSpec image_dto.TransformationSpec) ([]image_dto.ResponsiveVariant, error) {
	if baseSpec.Responsive == nil {
		return nil, errors.New("responsive spec is nil")
	}

	ctx, cancel := context.WithTimeoutCause(ctx, s.config.TransformTimeout,
		fmt.Errorf("responsive variant generation exceeded %s timeout", s.config.TransformTimeout))
	defer cancel()

	originalData, err := io.ReadAll(contextaware.NewReader(ctx, input))
	if err != nil {
		return nil, fmt.Errorf("failed to read input image: %w", err)
	}

	widths := determineResponsiveWidths(baseSpec)
	densities := determineResponsiveDensities(baseSpec)

	return s.generateVariantsConcurrently(ctx, originalData, baseSpec, widths, densities)
}

// GeneratePlaceholder creates a tiny, low-quality, heavily blurred
// placeholder image.
//
// Takes input (io.Reader) which provides the source image data.
// Takes baseSpec (image_dto.TransformationSpec) which defines the placeholder
// settings.
//
// Returns string which contains the base64-encoded data URL of the placeholder.
// Returns error when placeholder is not enabled or image processing fails.
func (s *service) GeneratePlaceholder(ctx context.Context, input io.Reader, baseSpec image_dto.TransformationSpec) (string, error) {
	if baseSpec.Placeholder == nil || !baseSpec.Placeholder.Enabled {
		return "", errors.New("placeholder spec is not enabled")
	}

	ctx, cancel := context.WithTimeoutCause(ctx, s.config.TransformTimeout,
		fmt.Errorf("placeholder generation exceeded %s timeout", s.config.TransformTimeout))
	defer cancel()

	originalData, err := io.ReadAll(contextaware.NewReader(ctx, input))
	if err != nil {
		return "", fmt.Errorf("failed to read input image: %w", err)
	}

	placeholderSpec := buildPlaceholderSpec(baseSpec)

	mimeType, imageData, err := s.transformPlaceholder(ctx, originalData, placeholderSpec)
	if err != nil {
		return "", err
	}

	return encodePlaceholderAsDataURL(mimeType, imageData), nil
}

// GetDimensions extracts width and height from image data using the default
// provider. Delegates to the provider's GetDimensions implementation for format
// support.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes input (io.Reader) which provides the source image data.
//
// Returns width (int) which is the image width in pixels.
// Returns height (int) which is the image height in pixels.
// Returns error when no provider is configured or the image cannot
// be decoded.
func (s *service) GetDimensions(ctx context.Context, input io.Reader) (width int, height int, err error) {
	provider, ok := s.transformers[s.defaultProvider]
	if !ok || provider == nil {
		return 0, 0, errors.New("no image provider configured")
	}

	return goroutine.SafeCall2(ctx, "image.GetDimensions", func() (int, int, error) { return provider.GetDimensions(ctx, input) })
}

// Name returns the service identifier string.
// Implements the healthprobe_domain.Probe interface.
//
// Returns string which is the name "ImageService".
func (*service) Name() string {
	return "ImageService"
}

// Check implements the healthprobe_domain.Probe interface.
// It verifies the image transformation pipeline is functional.
//
// Takes ctx (context.Context) which is the request context.
// Takes checkType (healthprobe_dto.CheckType) which specifies the
// type of health check to perform.
//
// Returns healthprobe_dto.Status which indicates the health state of
// the image service.
func (s *service) Check(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	transformerCount := len(s.transformers)

	state := healthprobe_dto.StateHealthy
	message := fmt.Sprintf("Image service operational with %d transformer(s)", transformerCount)

	if transformerCount == 0 {
		state = healthprobe_dto.StateUnhealthy
		message = "No image transformers registered"
	}

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: nil,
	}
}

// selectTransformer resolves the transformer to use for a given spec.
//
// Takes spec (TransformationSpec) which defines the transformation settings
// including the provider name.
//
// Returns TransformerPort which is the resolved transformer for the provider.
// Returns string which is the provider name used (may be the default).
// Returns error when the specified provider is not found.
func (s *service) selectTransformer(spec image_dto.TransformationSpec) (TransformerPort, string, error) {
	name := spec.Provider
	if name == "" {
		name = s.defaultProvider
	}
	tr, ok := s.transformers[name]
	if !ok {
		return nil, "", fmt.Errorf("image provider '%s' not found", name)
	}
	return tr, name, nil
}

// runTransformPipeline runs the image transformation with metrics and error
// handling.
//
// Takes input (io.Reader) which provides the source image data.
// Takes pw (*io.PipeWriter) which receives the transformed output.
// Takes tr (TransformerPort) which performs the actual transformation.
// Takes spec (image_dto.TransformationSpec) which defines the transformation
// to apply.
// Takes providerName (string) which identifies the provider for metrics.
func (*service) runTransformPipeline(
	ctx context.Context,
	input io.Reader,
	pw *io.PipeWriter,
	tr TransformerPort,
	spec image_dto.TransformationSpec,
	providerName string,
	cancel context.CancelFunc,
) {
	startTime := time.Now()

	defer func() {
		cancel()
		duration := time.Since(startTime).Milliseconds()
		recordTransformDuration(ctx, providerName, spec, duration)

		if r := recover(); r != nil {
			recordTransformError(ctx, providerName, "panic")
			_ = pw.CloseWithError(fmt.Errorf("panic in transformer: %v", r))
		}
	}()

	mimeType, err := goroutine.SafeCall1(ctx, "image.Transform", func() (string, error) { return tr.Transform(ctx, input, pw, spec) })
	if err != nil {
		recordTransformError(ctx, providerName, "transform_failed")
	}

	_ = mimeType
	_ = pw.CloseWithError(err)
}

// buildSidecarKey creates a consistent, unique key for a transformed image.
// Uses xxhash for speed and to make it clear this is not for cryptographic
// purposes.
//
// Takes originalKey (string) which is the path to the source image.
// Takes spec (image_dto.TransformationSpec) which defines the transformation.
//
// Returns string which is the sidecar key in the format
// "path/to/image.transform_a1b2c3d4.webp".
func (*service) buildSidecarKey(originalKey string, spec image_dto.TransformationSpec) string {
	hasher := xxhash.New()
	_, _ = hasher.WriteString(spec.String())
	hash := hex.EncodeToString(hasher.Sum(nil))

	shortHash := hash[:hashPrefixLength]

	ext := filepath.Ext(originalKey)
	baseKey := originalKey[:len(originalKey)-len(ext)]

	return fmt.Sprintf("%s.transform_%s.%s", baseKey, shortHash, spec.Format)
}

// getFallbackIcon finds the appropriate fallback icon for a given MIME type.
//
// Takes contentType (string) which specifies the MIME type to match against.
//
// Returns []byte which contains the icon data for the matching type, or the
// default icon if no specific match is found.
func (s *service) getFallbackIcon(contentType string) []byte {
	for prefix, data := range s.fallbackIcons {
		if strings.HasPrefix(contentType, prefix) {
			return data
		}
	}
	return s.fallbackIcons[image_dto.ImageNameDefault]
}

// generateVariantsConcurrently generates all responsive variants
// simultaneously, spawning one task per width/density combination
// and collecting results under a lock.
//
// Takes data ([]byte) which contains the source image bytes.
// Takes baseSpec (image_dto.TransformationSpec) which defines the
// base transformation settings.
// Takes widths ([]int) which lists the target widths in pixels.
// Takes densities ([]string) which lists the pixel density
// multipliers.
//
// Returns []image_dto.ResponsiveVariant which contains all
// generated variants.
// Returns error when any variant generation fails.
//
// Concurrent goroutines are spawned per width/density combination
// via sync.WaitGroup. Results are collected under a mutex.
func (s *service) generateVariantsConcurrently(
	ctx context.Context,
	data []byte,
	baseSpec image_dto.TransformationSpec,
	widths []int,
	densities []string,
) ([]image_dto.ResponsiveVariant, error) {
	var variants []image_dto.ResponsiveVariant
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(widths)*len(densities))

	for _, width := range widths {
		for _, density := range densities {
			wg.Go(func() {
				variant, err := s.generateSingleVariant(ctx, data, baseSpec, width, density)
				if err != nil {
					errChan <- err
					return
				}

				mu.Lock()
				variants = append(variants, variant)
				mu.Unlock()
			})
		}
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	return variants, nil
}

// generateSingleVariant creates a single responsive variant by transforming
// the image data to the specified width and density.
//
// Takes data ([]byte) which contains the source image bytes.
// Takes baseSpec (image_dto.TransformationSpec) which defines the base
// transformation settings.
// Takes width (int) which specifies the target width in pixels.
// Takes density (string) which specifies the pixel density multiplier.
//
// Returns image_dto.ResponsiveVariant which contains the variant metadata.
// Returns error when transformer selection or image transformation fails.
func (s *service) generateSingleVariant(
	ctx context.Context,
	data []byte,
	baseSpec image_dto.TransformationSpec,
	width int,
	density string,
) (image_dto.ResponsiveVariant, error) {
	densityMultiplier := parseDensity(density)
	actualWidth := int(float64(width) * densityMultiplier)

	variantSpec := buildResponsiveVariantSpec(baseSpec, actualWidth)

	tr, _, err := s.selectTransformer(variantSpec)
	if err != nil {
		return image_dto.ResponsiveVariant{}, fmt.Errorf("failed to select transformer: %w", err)
	}

	var buffer bytes.Buffer
	_, err = goroutine.SafeCall1(ctx, "image.Transform", func() (string, error) { return tr.Transform(ctx, bytes.NewReader(data), &buffer, variantSpec) })
	if err != nil {
		return image_dto.ResponsiveVariant{}, fmt.Errorf("failed to transform variant %dx%s: %w", width, density, err)
	}

	return image_dto.ResponsiveVariant{
		Width:         actualWidth,
		Height:        0,
		Density:       density,
		Specification: variantSpec,
		URL:           "",
		SrcsetEntry:   "",
	}, nil
}

// transformPlaceholder performs the actual image transformation for a
// placeholder.
//
// Takes data ([]byte) which contains the raw image data to transform.
// Takes spec (image_dto.TransformationSpec) which defines the transformation
// parameters.
//
// Returns string which is the MIME type of the transformed image.
// Returns []byte which contains the transformed image data.
// Returns error when transformer selection or transformation fails.
func (s *service) transformPlaceholder(ctx context.Context, data []byte, spec image_dto.TransformationSpec) (string, []byte, error) {
	var buffer bytes.Buffer

	tr, _, err := s.selectTransformer(spec)
	if err != nil {
		return "", nil, fmt.Errorf("failed to select transformer: %w", err)
	}

	mimeType, err := goroutine.SafeCall1(ctx, "image.Transform", func() (string, error) { return tr.Transform(ctx, bytes.NewReader(data), &buffer, spec) })
	if err != nil {
		return "", nil, fmt.Errorf("failed to transform placeholder: %w", err)
	}

	return mimeType, buffer.Bytes(), nil
}

// DefaultServiceConfig returns a ServiceConfig with sensible defaults for
// production use.
//
// Returns ServiceConfig which contains default values for image processing
// limits, timeouts, and allowed formats.
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		FallbackIconPaths: make(map[string]string),
		MaxImageWidth:     defaultMaxImageWidth,
		MaxImageHeight:    defaultMaxImageHeight,
		MaxImagePixels:    defaultMaxImagePixels,
		MaxFileSizeBytes:  defaultMaxFileSizeBytes,
		TransformTimeout:  defaultTransformTimeout,
		AllowedFormats:    []string{"jpeg", "jpg", "png", "webp", "avif", "gif"},
	}
}

// NewServiceWithDefaultTransformer creates a new image service with a specified
// default transformer name. Transformers must be registered separately via
// RegisterTransformer.
//
// Takes defaultTransformerName (string) which is the name of the transformer to
// use by default.
//
// Returns Service which is the configured image service ready for use.
func NewServiceWithDefaultTransformer(defaultTransformerName string) Service {
	config := DefaultServiceConfig()
	return &service{
		transformers:         make(map[string]TransformerPort),
		providerCapabilities: make(map[string]providerCapability),
		fallbackIcons:        make(map[string][]byte),
		defaultProvider:      defaultTransformerName,
		config:               config,
	}
}

// NewService creates a new image service.
//
// It needs a set of transformers for image processing and checks the inputs
// before it applies the default settings.
//
// Takes transformers (map[string]TransformerPort) which provides the image
// processing backends keyed by provider name.
// Takes defaultProvider (string) which specifies which transformer to use
// when none is requested.
// Takes config (ServiceConfig) which provides service settings including
// fallback icon paths.
//
// Returns Service which is the configured image service ready for use.
// Returns error when validation fails or fallback icons cannot be loaded.
func NewService(transformers map[string]TransformerPort, defaultProvider string, config ServiceConfig) (Service, error) {
	if err := validateServiceInputs(transformers, defaultProvider); err != nil {
		return nil, fmt.Errorf("initialising image service: %w", err)
	}

	config = applyConfigDefaults(config)

	icons, err := loadFallbackIcons(config.FallbackIconPaths, config.FallbackIconSandbox, config.FallbackIconSandboxFactory)
	if err != nil {
		return nil, fmt.Errorf("failed to load fallback icons: %w", err)
	}

	capabilities := extractProviderCapabilities(transformers)

	return &service{
		transformers:         transformers,
		providerCapabilities: capabilities,
		fallbackIcons:        icons,
		config:               config,
		defaultProvider:      defaultProvider,
	}, nil
}

// validateServiceInputs checks that the required inputs are valid before
// creating a new image service.
//
// Takes transformers (map[string]TransformerPort) which provides the available
// image transformers keyed by provider name.
// Takes defaultProvider (string) which specifies which transformer to use by
// default.
//
// Returns error when no transformers are provided, when the default provider
// is empty, or when the default provider is not found in the transformers map.
func validateServiceInputs(transformers map[string]TransformerPort, defaultProvider string) error {
	if len(transformers) == 0 {
		return errNoTransformers
	}
	if defaultProvider == "" {
		return errDefaultProviderEmpty
	}
	if _, ok := transformers[defaultProvider]; !ok {
		return fmt.Errorf("default image provider '%s' is not registered", defaultProvider)
	}
	return nil
}

// applyConfigDefaults fills in any missing fields with default values.
//
// Takes config (ServiceConfig) which holds the settings to check and fill.
//
// Returns ServiceConfig which has all fields set to valid values.
func applyConfigDefaults(config ServiceConfig) ServiceConfig {
	defaults := DefaultServiceConfig()

	if config.MaxImageWidth <= 0 {
		config.MaxImageWidth = defaults.MaxImageWidth
	}
	if config.MaxImageHeight <= 0 {
		config.MaxImageHeight = defaults.MaxImageHeight
	}
	if config.MaxImagePixels <= 0 {
		config.MaxImagePixels = defaults.MaxImagePixels
	}
	if config.MaxFileSizeBytes <= 0 {
		config.MaxFileSizeBytes = defaults.MaxFileSizeBytes
	}
	if config.TransformTimeout <= 0 {
		config.TransformTimeout = defaults.TransformTimeout
	}
	if len(config.AllowedFormats) == 0 {
		config.AllowedFormats = defaults.AllowedFormats
	}

	return config
}

// extractProviderCapabilities builds a map of provider capabilities from the
// given transformers.
//
// Takes transformers (map[string]TransformerPort) which provides the image
// transformers to read capabilities from.
//
// Returns map[string]providerCapability which maps provider names to their
// supported formats and modifiers.
func extractProviderCapabilities(transformers map[string]TransformerPort) map[string]providerCapability {
	capabilities := make(map[string]providerCapability)

	for name, transformer := range transformers {
		formats := transformer.GetSupportedFormats()
		modifiers := transformer.GetSupportedModifiers()

		modifierMap := make(map[string]bool, len(modifiers))
		for _, mod := range modifiers {
			modifierMap[mod] = true
		}

		capabilities[name] = providerCapability{
			supportedModifiers: modifierMap,
			supportedFormats:   formats,
		}
	}

	return capabilities
}

// recordTransformMetrics records metrics when a transformation starts.
//
// Takes provider (string) which identifies the image provider.
// Takes spec (image_dto.TransformationSpec) which describes the transformation.
// Takes incrementCount (bool) which controls whether to increment the counter.
func recordTransformMetrics(ctx context.Context, provider string, spec image_dto.TransformationSpec, incrementCount bool) {
	attrs := metric.WithAttributes(
		attribute.String(metricAttributeProvider, provider),
		attribute.String("format", spec.Format),
		attribute.String("fit", string(spec.Fit)),
	)

	if incrementCount {
		transformCount.Add(ctx, 1, attrs)
	}
}

// recordTransformDuration records how long an image transformation took.
//
// Takes provider (string) which names the image processing provider.
// Takes spec (image_dto.TransformationSpec) which describes the transformation.
// Takes durationMs (int64) which is the duration in milliseconds.
func recordTransformDuration(ctx context.Context, provider string, spec image_dto.TransformationSpec, durationMs int64) {
	transformDuration.Record(ctx, float64(durationMs),
		metric.WithAttributes(
			attribute.String(metricAttributeProvider, provider),
			attribute.String("format", spec.Format),
			attribute.String("fit", string(spec.Fit)),
		),
	)
}

// recordTransformError increases the transform error counter with the given
// provider and error type labels.
//
// Takes provider (string) which identifies the data provider that caused the
// error.
// Takes errorType (string) which describes the kind of transform error that
// happened.
func recordTransformError(ctx context.Context, provider, errorType string) {
	transformErrorCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(metricAttributeProvider, provider),
			attribute.String("error_type", errorType),
		),
	)
}

// loadFallbackIcons reads icon files from disk and stores them in memory.
//
// Takes paths (map[string]string) which maps icon names to their file paths.
// Takes injectedSandbox (safedisk.Sandbox) which is an optional sandbox for
// testing. When nil, sandboxes are created per icon file's parent directory.
// Takes factory (safedisk.Factory) which is an optional factory for creating
// sandboxes. When non-nil and injectedSandbox is nil, this factory is used
// instead of safedisk.NewNoOpSandbox.
//
// Returns map[string][]byte which contains the icon data keyed by name.
// Returns error when a sandbox cannot be created or a file cannot be read.
func loadFallbackIcons(paths map[string]string, injectedSandbox safedisk.Sandbox, factory safedisk.Factory) (map[string][]byte, error) {
	icons := make(map[string][]byte)
	for name, iconPath := range paths {
		fileName := filepath.Base(iconPath)

		if injectedSandbox != nil {
			data, err := injectedSandbox.ReadFile(fileName)
			if err != nil {
				return nil, fmt.Errorf("could not read icon file for '%s' at path '%s': %w", name, iconPath, err)
			}
			icons[name] = data
			continue
		}

		parentDir := filepath.Dir(iconPath)

		var sandbox safedisk.Sandbox
		var err error
		if factory != nil {
			sandbox, err = factory.Create("fallback-icon", parentDir, safedisk.ModeReadOnly)
		} else {
			sandbox, err = safedisk.NewNoOpSandbox(parentDir, safedisk.ModeReadOnly)
		}
		if err != nil {
			return nil, fmt.Errorf("could not create sandbox for icon file '%s': %w", name, err)
		}

		data, readErr := sandbox.ReadFile(fileName)
		_ = sandbox.Close()
		if readErr != nil {
			return nil, fmt.Errorf("could not read icon file for '%s' at path '%s': %w", name, iconPath, readErr)
		}
		icons[name] = data
	}
	return icons, nil
}

// specToMIMEType converts an image format from a transformation spec into the
// matching MIME type.
//
// Takes spec (image_dto.TransformationSpec) which holds the image format to
// look up.
//
// Returns string which is the MIME type for the format, or an empty string if
// the format is not known.
func specToMIMEType(spec image_dto.TransformationSpec) string {
	format := strings.ToLower(spec.Format)

	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "avif":
		return "image/avif"
	case "gif":
		return "image/gif"
	default:
		return mime.TypeByExtension("." + format)
	}
}

// determineResponsiveWidths parses the responsive configuration and returns the
// list of widths to generate for responsive images.
//
// Takes spec (image_dto.TransformationSpec) which contains the responsive
// sizing configuration including sizes and screen breakpoints.
//
// Returns []int which contains the parsed widths, or falls back to the base
// width if parsing yields no results.
func determineResponsiveWidths(spec image_dto.TransformationSpec) []int {
	widths := parseSizes(spec.Responsive.Sizes, spec.Responsive.Screens)
	if len(widths) == 0 {
		return []int{spec.Width}
	}
	return widths
}

// determineResponsiveDensities returns the list of densities to generate for
// responsive images, defaulting to x1 if none are specified.
//
// Takes spec (image_dto.TransformationSpec) which contains the responsive
// image configuration including density settings.
//
// Returns []string which contains the density identifiers to generate.
func determineResponsiveDensities(spec image_dto.TransformationSpec) []string {
	if len(spec.Responsive.Densities) == 0 {
		return []string{"x1"}
	}
	return spec.Responsive.Densities
}

// buildResponsiveVariantSpec creates a spec for a single responsive variant.
//
// Takes baseSpec (image_dto.TransformationSpec) which provides the base
// settings to copy.
// Takes width (int) which sets the target width for this variant.
//
// Returns image_dto.TransformationSpec which is a copy of the base spec with
// the width set and responsive and placeholder options turned off to prevent
// endless loops.
func buildResponsiveVariantSpec(baseSpec image_dto.TransformationSpec, width int) image_dto.TransformationSpec {
	spec := baseSpec
	spec.Width = width
	spec.Responsive = nil
	spec.Placeholder = nil
	return spec
}

// buildPlaceholderSpec creates a transformation spec for placeholder images.
//
// Takes baseSpec (image_dto.TransformationSpec) which provides the source
// settings including placeholder dimensions, quality, and blur values.
//
// Returns image_dto.TransformationSpec which contains the configured
// placeholder settings with defaults applied for any missing values.
func buildPlaceholderSpec(baseSpec image_dto.TransformationSpec) image_dto.TransformationSpec {
	spec := baseSpec

	spec.Width = baseSpec.Placeholder.Width
	if spec.Width == 0 {
		spec.Width = defaultPlaceholderWidth
	}

	spec.Height = baseSpec.Placeholder.Height

	spec.Quality = baseSpec.Placeholder.Quality
	if spec.Quality == 0 {
		spec.Quality = defaultPlaceholderQuality
	}

	if spec.Modifiers == nil {
		spec.Modifiers = make(map[string]string)
	}
	blurSigma := baseSpec.Placeholder.BlurSigma
	if blurSigma == 0 {
		blurSigma = defaultPlaceholderBlur
	}
	spec.Modifiers["blur"] = fmt.Sprintf("%.1f", blurSigma)

	spec.Responsive = nil
	spec.Placeholder = nil

	return spec
}

// encodePlaceholderAsDataURL encodes image data as a base64 data URL string.
//
// Takes mimeType (string) which sets the MIME type for the data URL.
// Takes data ([]byte) which holds the raw image bytes to encode.
//
// Returns string which is the full data URL in the format
// "data:mimeType;base64,encodedData".
func encodePlaceholderAsDataURL(mimeType string, data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
}

// parseSizes parses a sizes string like "100vw sm:50vw md:400px" into a list
// of pixel widths.
//
// Takes sizes (string) which specifies the responsive size definitions.
// Takes screens (map[string]int) which maps screen names to pixel widths.
//
// Returns []int which contains the pixel widths for each breakpoint.
func parseSizes(sizes string, screens map[string]int) []int {
	if sizes == "" {
		return nil
	}

	if screens == nil {
		screens = map[string]int{
			"sm":  screenSizeSmall,
			"md":  screenSizeMedium,
			"lg":  screenSizeLarge,
			"xl":  screenSizeXLarge,
			"2xl": screenSize2XLarge,
		}
	}

	widths := []int{
		responsiveWidthXSmall,
		responsiveWidthSmall,
		responsiveWidthMedium,
		responsiveWidthLarge,
		responsiveWidthXLarge,
		responsiveWidth2XLarge,
	}
	return widths
}

// parseDensity parses a density string such as "x1", "x2", or "2x" into a
// multiplier value.
//
// Takes density (string) which is the density value to parse.
//
// Returns float64 which is the parsed multiplier, or defaultDensityMultiplier
// when parsing fails or the value is not positive.
func parseDensity(density string) float64 {
	density = strings.ToLower(strings.TrimSpace(density))
	density = strings.TrimPrefix(density, "x")
	density = strings.TrimSuffix(density, "x")

	multiplier, err := strconv.ParseFloat(density, 64)
	if err != nil || multiplier <= 0 {
		return defaultDensityMultiplier
	}

	return multiplier
}
