//go:build vips

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

package image_provider_vips

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"runtime"
	"strconv"
	"strings"
	"sync"

	// Register standard image decoders for image.DecodeConfig.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/davidbyttow/govips/v2/vips"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/media"

	// Register WebP decoder for image.DecodeConfig.
	_ "golang.org/x/image/webp"
)

const (
	// rotationAngle90 is the angle in degrees for a 90-degree rotation.
	rotationAngle90 = 90

	// rotationAngle180 is the angle in degrees for a 180-degree image rotation.
	rotationAngle180 = 180

	// rotationAngle270 is a 270 degree clockwise rotation.
	rotationAngle270 = 270

	// sharpenM1 is the flat parameter for the VIPS sharpen function.
	sharpenM1 = 1.0

	// sharpenM2 is the second slope value for vips sharpen operations.
	sharpenM2 = 2.0

	// tintColourLength is the expected length of a hex colour string (e.g. "#FF0000").
	tintColourLength = 7

	// percentBase is the divisor used to convert a percentage into a multiplier.
	percentBase = 100.0

	// colourMidpoint is the colour channel midpoint for contrast calculation.
	colourMidpoint = 128.0

	// pngMaxCompression is the highest PNG compression level.
	pngMaxCompression = 9

	// pngCompressionBase is the divisor used to convert quality to compression level.
	pngCompressionBase = 10

	// avifEncodingSpeed controls AVIF encoding speed (0-10, higher is faster
	// but produces larger files).
	avifEncodingSpeed = 8
)

// Config holds configuration options for the vips provider.
type Config struct {
	// VipsConfig provides settings for the govips library; nil uses defaults.
	VipsConfig *vips.Config

	// ImageServiceConfig holds security settings for input validation and resource limits.
	// If nil, default configuration will be used.
	media.ImageServiceConfig

	// ConcurrencyLevel limits how many image tasks can run at the same time.
	// If set to 0, it defaults to runtime.NumCPU().
	ConcurrencyLevel int
}

// Provider implements the ImageTransformerPort interface using the govips
// library.
type Provider struct {
	// semaphore limits the number of concurrent transforms.
	semaphore chan struct{}

	// config holds security settings for input checking and resource limits.
	config media.ImageServiceConfig

	// shutdownOnce guards single execution of the vips shutdown.
	shutdownOnce sync.Once
}

var _ media.ImageTransformerPort = (*Provider)(nil)

// NewProvider creates a new transformer and initialises the underlying vips
// library.
//
// It is critical to call the returned Close method on application shutdown
// to free resources.
//
// Takes config (Config) which specifies the provider configuration.
//
// Returns *Provider which is ready to perform image transformations.
// Returns error when the transformer cannot be created.
func NewProvider(config Config) (*Provider, error) {
	vips.LoggingSettings(vipsLogger, vips.LogLevelWarning)
	if err := vips.Startup(config.VipsConfig); err != nil {
		return nil, fmt.Errorf("vips startup: %w", err)
	}

	serviceConfig := config.ImageServiceConfig
	if serviceConfig.MaxFileSizeBytes == 0 {
		serviceConfig = media.DefaultImageServiceConfig()
	}

	concurrency := config.ConcurrencyLevel
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	return &Provider{
		config:       serviceConfig,
		shutdownOnce: sync.Once{},
		semaphore:    make(chan struct{}, concurrency),
	}, nil
}

// Transform applies the given changes to an image from the input stream.
//
// Takes input (io.Reader) which provides the source image data.
// Takes output (io.Writer) which receives the changed image data.
// Takes spec (media.TransformationSpec) which defines the changes to apply.
//
// Returns string which is the MIME type of the output image.
// Returns error when validation fails, the context is cancelled, or image
// processing fails.
func (p *Provider) Transform(
	ctx context.Context,
	input io.Reader,
	output io.Writer,
	spec media.TransformationSpec,
) (string, error) {
	ctx, l := logger.From(ctx, log)

	validatedSpec, err := image_domain.ValidateTransformationSpec(spec, nil)
	if err != nil {
		return "", fmt.Errorf("invalid transformation spec: %w", err)
	}
	spec = validatedSpec

	if err := image_domain.ValidateImageFormat(ctx, spec.Format, p.config); err != nil {
		l.Warn("Rejected transformation due to disallowed format",
			logger.String("format", spec.Format),
			logger.Error(err))
		return "", fmt.Errorf("format validation failed: %w", err)
	}

	select {
	case p.semaphore <- struct{}{}:
		defer func() { <-p.semaphore }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	img, err := p.decodeImage(ctx, input)
	if err != nil {
		return "", err
	}
	defer img.Close()

	if err := p.validateImageSecurity(ctx, img); err != nil {
		return "", err
	}

	if spec.Width > 0 || spec.Height > 0 {
		if err := p.resizeOrCrop(img, spec, spec.Modifiers); err != nil {
			return "", err
		}
	}

	if err := p.applyModifiers(img, spec.Modifiers); err != nil {
		return "", err
	}

	buffer, mimeType, err := p.export(img, spec)
	if err != nil {
		return "", err
	}
	if _, err := output.Write(buffer); err != nil {
		return "", fmt.Errorf("failed to write transformed image to output: %w", err)
	}
	return mimeType, nil
}

// GetSupportedFormats returns the list of output formats that libvips can
// generate.
//
// Returns []string which contains the supported format identifiers.
func (*Provider) GetSupportedFormats() []string {
	return []string{"jpeg", "jpg", "png", "webp", "avif", "gif"}
}

// GetSupportedModifiers returns the list of transformation modifiers
// supported by libvips.
//
// Returns []string which contains the names of all supported image
// transformation modifiers.
func (*Provider) GetSupportedModifiers() []string {
	return []string{
		"greyscale", "blur", "sharpen", "rotate", "flip",
		"brightness", "contrast", "saturation",
		"hue", "tint", "gravity", "focus", "radius",
	}
}

// GetDimensions extracts width and height from image data using
// lightweight header decoding, falling back to a full vips load
// for formats the standard library cannot decode (e.g. AVIF).
//
// Takes input (io.Reader) which provides the source image data.
//
// Returns width (int) in pixels.
// Returns height (int) in pixels.
// Returns error when the image cannot be decoded.
func (*Provider) GetDimensions(_ context.Context, input io.Reader) (width int, height int, err error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read image data: %w", err)
	}

	config, _, decodeErr := image.DecodeConfig(bytes.NewReader(data))
	if decodeErr == nil {
		return config.Width, config.Height, nil
	}

	img, err := vips.NewImageFromBuffer(data)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image header: %w", decodeErr)
	}
	defer img.Close()

	return img.Width(), img.Height(), nil
}

// Close releases resources held by the underlying libvips library.
//
// Returns error which is always nil as shutdown cannot fail.
func (p *Provider) Close() error {
	p.shutdownOnce.Do(func() {
		vips.Shutdown()
	})
	return nil
}

// decodeImage decodes the input stream into a vips.ImageRef, applying size
// limits.
//
// Takes ctx (context.Context) which provides the request-scoped logger.
// Takes input (io.Reader) which provides the image data to decode.
//
// Returns *vips.ImageRef which is the decoded image ready for processing.
// Returns error when the image cannot be decoded or exceeds the size limit.
func (p *Provider) decodeImage(ctx context.Context, input io.Reader) (*vips.ImageRef, error) {
	limitedInput := image_domain.NewLimitedReader(input, p.config.MaxFileSizeBytes)
	img, err := vips.NewImageFromReader(limitedInput)
	if err != nil {
		if errors.Is(err, image_domain.ErrSizeLimitExceeded) {
			_, l := logger.From(ctx, log)
			l.Warn("Rejected transformation due to file size limit",
				logger.Int64("max_bytes", p.config.MaxFileSizeBytes),
				logger.Error(err))
		}
		return nil, fmt.Errorf("failed to decode input image with vips: %w", err)
	}
	return img, nil
}

// validateImageSecurity validates image dimensions and pixel count against
// security limits.
//
// Takes img (*vips.ImageRef) which is the image to validate.
//
// Returns error when dimensions or pixel count exceed configured limits.
func (p *Provider) validateImageSecurity(ctx context.Context, img *vips.ImageRef) error {
	ctx, l := logger.From(ctx, log)
	width := img.Width()
	height := img.Height()

	if err := image_domain.ValidateImageDimensions(ctx, width, height, p.config); err != nil {
		l.Warn("Rejected transformation due to dimension limits",
			logger.Int("width", width),
			logger.Int("height", height),
			logger.Error(err))
		return fmt.Errorf("dimension validation failed: %w", err)
	}

	if err := image_domain.ValidateImagePixelCount(ctx, width, height, p.config); err != nil {
		l.Warn("Rejected transformation due to pixel count limit",
			logger.Int("width", width),
			logger.Int("height", height),
			logger.Int64("pixels", int64(width)*int64(height)),
			logger.Error(err))
		return fmt.Errorf("pixel count validation failed: %w", err)
	}

	l.Trace("Image validation passed",
		logger.Int("width", width),
		logger.Int("height", height),
		logger.Int64("pixels", int64(width)*int64(height)))

	return nil
}

// resizeOrCrop handles the resizing and cropping logic based on the fit mode.
//
// Takes img (*vips.ImageRef) which is the image to resize or crop.
// Takes spec (media.TransformationSpec) which defines the target dimensions
// and fit mode.
// Takes modifiers (map[string]string) which provides additional processing
// options.
//
// Returns error when dimension calculation or resize execution fails.
func (*Provider) resizeOrCrop(img *vips.ImageRef, spec media.TransformationSpec, modifiers map[string]string) error {
	targetWidth, targetHeight, err := calculateVipsTargetDimensions(spec)
	if err != nil {
		return err
	}

	if targetWidth == 0 && targetHeight == 0 {
		return nil
	}

	currentWidth := img.Width()
	currentHeight := img.Height()

	targetWidth, targetHeight = applyVipsWithoutEnlargement(spec.WithoutEnlargement, targetWidth, targetHeight, currentWidth, currentHeight)

	return executeVipsResize(img, spec.Fit, targetWidth, targetHeight, currentWidth, currentHeight, modifiers)
}

// applyModifiers applies additional transformations based on the modifiers map.
//
// Takes img (*vips.ImageRef) which is the image to transform.
// Takes modifiers (map[string]string) which specifies the transformations to
// apply.
//
// Returns error when any transformation fails.
func (*Provider) applyModifiers(img *vips.ImageRef, modifiers map[string]string) error {
	if err := applyVipsGreyscale(img, modifiers); err != nil {
		return err
	}
	if err := applyVipsBlur(img, modifiers); err != nil {
		return err
	}
	if err := applyVipsSharpen(img, modifiers); err != nil {
		return err
	}
	if err := applyVipsRotation(img, modifiers); err != nil {
		return err
	}
	if err := applyVipsFlip(img, modifiers); err != nil {
		return err
	}
	if err := applyVipsTint(img, modifiers); err != nil {
		return err
	}
	return applyVipsColourAdjustments(img, modifiers)
}

// export encodes the vips.ImageRef into the specified output format.
//
// Takes img (*vips.ImageRef) which is the image to encode.
// Takes spec (media.TransformationSpec) which defines the output format and
// quality settings.
//
// Returns []byte which contains the encoded image data.
// Returns string which is the MIME type of the encoded image.
// Returns error when the format is unsupported or encoding fails.
func (*Provider) export(img *vips.ImageRef, spec media.TransformationSpec) ([]byte, string, error) {
	switch strings.ToLower(spec.Format) {
	case "jpeg", "jpg":
		params := vips.NewJpegExportParams()
		params.Quality = spec.Quality
		params.StripMetadata = true
		params.Interlace = true

		outBuf, _, err := img.ExportJpeg(params)
		if err != nil {
			return nil, "", fmt.Errorf("vips jpeg export failed: %w", err)
		}
		return outBuf, "image/jpeg", nil

	case "png":
		params := vips.NewPngExportParams()
		compression := (spec.Quality / pngCompressionBase) - 1
		params.Compression = max(0, min(compression, pngMaxCompression))
		params.StripMetadata = true

		outBuf, _, err := img.ExportPng(params)
		if err != nil {
			return nil, "", fmt.Errorf("vips png export failed: %w", err)
		}
		return outBuf, "image/png", nil

	case "webp":
		params := vips.NewWebpExportParams()
		params.Quality = spec.Quality
		params.StripMetadata = true

		outBuf, _, err := img.ExportWebp(params)
		if err != nil {
			return nil, "", fmt.Errorf("vips webp export failed: %w", err)
		}
		return outBuf, "image/webp", nil

	case "avif":
		params := vips.NewAvifExportParams()
		params.Quality = spec.Quality
		params.Speed = avifEncodingSpeed
		params.StripMetadata = true

		outBuf, _, err := img.ExportAvif(params)
		if err != nil {
			return nil, "", fmt.Errorf("vips avif export failed: %w", err)
		}
		return outBuf, "image/avif", nil
	}

	return nil, "", fmt.Errorf("unhandled export format: %s", spec.Format)
}

// vipsLogger handles log messages from the govips library.
//
// Takes messageDomain (string) which identifies where the log message came
// from.
// Takes verbosity (vips.LogLevel) which sets the severity level.
// Takes message (string) which contains the log text.
func vipsLogger(messageDomain string, verbosity vips.LogLevel, message string) {
	_, l := logger.From(context.Background(), log)
	fullMessage := fmt.Sprintf("govips/%s: %s", messageDomain, message)
	switch verbosity {
	case vips.LogLevelError, vips.LogLevelCritical:
		l.Error(fullMessage)
	case vips.LogLevelWarning:
		l.Warn(fullMessage)
	case vips.LogLevelMessage, vips.LogLevelInfo, vips.LogLevelDebug:
		l.Debug(fullMessage)
	}
}

// parseAspectRatio parses an aspect ratio string like "16:9" and returns
// the ratio as a decimal value.
//
// Takes ar (string) which specifies the aspect ratio in "width:height" format.
//
// Returns float64 which is the calculated ratio (width divided by height).
// Returns error when the format is invalid, the values cannot be parsed, or
// the dimensions are not positive.
func parseAspectRatio(ar string) (float64, error) {
	parts := strings.Split(ar, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid aspect ratio format: %s (expected format: '16:9')", ar)
	}

	w, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid aspect ratio width: %w", err)
	}

	h, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid aspect ratio height: %w", err)
	}

	if w <= 0 || h <= 0 {
		return 0, errors.New("aspect ratio dimensions must be positive")
	}

	return w / h, nil
}

// parseGravity determines the vips Interesting value from gravity or focus
// modifiers.
//
// Takes modifiers (map[string]string) which contains the image processing
// modifiers to check for gravity or focus settings.
//
// Returns vips.Interesting which indicates the cropping strategy to use,
// defaulting to centre if no recognised modifier is found.
func parseGravity(modifiers map[string]string) vips.Interesting {
	interesting := vips.InterestingCentre

	if gravity, ok := modifiers["gravity"]; ok {
		switch strings.ToLower(gravity) {
		case "entropy":
			return vips.InterestingEntropy
		case "attention", "auto":
			return vips.InterestingAttention
		case "none", "low":
			return vips.InterestingNone
		}
	}

	if focus, ok := modifiers["focus"]; ok {
		switch strings.ToLower(focus) {
		case "entropy":
			return vips.InterestingEntropy
		case "attention", "auto":
			return vips.InterestingAttention
		}
	}

	return interesting
}

// resizeOutsideVips resizes an image for outside fit mode.
//
// Takes img (*vips.ImageRef) which is the image to resize.
// Takes targetW (int) which is the target width; zero uses targetH.
// Takes targetH (int) which is the target height; zero uses targetW.
// Takes currentW (int) which is the current image width.
// Takes currentH (int) which is the current image height.
//
// Returns error when the thumbnail operation fails.
func resizeOutsideVips(img *vips.ImageRef, targetW, targetH, currentW, currentH int) error {
	if targetW == 0 {
		targetW = targetH
	}
	if targetH == 0 {
		targetH = targetW
	}

	widthScale := float64(targetW) / float64(currentW)
	heightScale := float64(targetH) / float64(currentH)
	scale := max(widthScale, heightScale)

	newWidth := int(float64(currentW) * scale)
	newHeight := int(float64(currentH) * scale)
	return img.ThumbnailWithSize(newWidth, newHeight, vips.InterestingNone, vips.SizeBoth)
}

// resizeFillVips handles fill fit mode by stretching to exact dimensions.
//
// Takes img (*vips.ImageRef) which is the image to resize.
// Takes targetW (int) which is the target width, or 0 to keep current.
// Takes targetH (int) which is the target height, or 0 to keep current.
// Takes currentW (int) which is the current width of the image.
// Takes currentH (int) which is the current height of the image.
//
// Returns error when the thumbnail operation fails.
func resizeFillVips(img *vips.ImageRef, targetW, targetH, currentW, currentH int) error {
	if targetW == 0 {
		targetW = currentW
	}
	if targetH == 0 {
		targetH = currentH
	}
	return img.ThumbnailWithSize(targetW, targetH, vips.InterestingNone, vips.SizeForce)
}

// calculateVipsTargetDimensions computes target dimensions, accounting for
// aspect ratio.
//
// Takes spec (media.TransformationSpec) which provides the requested width,
// height, and optional aspect ratio.
//
// Returns targetWidth (int) which is the calculated target width.
// Returns targetHeight (int) which is the calculated target height.
// Returns err (error) when the aspect ratio cannot be parsed.
func calculateVipsTargetDimensions(spec media.TransformationSpec) (targetWidth int, targetHeight int, err error) {
	targetWidth = spec.Width
	targetHeight = spec.Height

	if spec.AspectRatio == "" {
		return targetWidth, targetHeight, nil
	}

	var ratio float64
	ratio, err = parseAspectRatio(spec.AspectRatio)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse aspect ratio: %w", err)
	}

	if targetWidth > 0 && targetHeight == 0 {
		targetHeight = int(float64(targetWidth) / ratio)
	} else if targetHeight > 0 && targetWidth == 0 {
		targetWidth = int(float64(targetHeight) * ratio)
	}

	return targetWidth, targetHeight, nil
}

// applyVipsWithoutEnlargement clamps target dimensions to not exceed current
// dimensions.
//
// Takes enabled (bool) which controls whether clamping is applied.
// Takes targetW (int) which specifies the desired width.
// Takes targetH (int) which specifies the desired height.
// Takes currentW (int) which specifies the current width.
// Takes currentH (int) which specifies the current height.
//
// Returns width (int) which is the clamped width.
// Returns height (int) which is the clamped height.
func applyVipsWithoutEnlargement(enabled bool, targetW, targetH, currentW, currentH int) (width int, height int) {
	if !enabled {
		return targetW, targetH
	}
	if targetW > 0 && targetW > currentW {
		targetW = currentW
	}
	if targetH > 0 && targetH > currentH {
		targetH = currentH
	}
	return targetW, targetH
}

// inferMissingDimension computes the missing target dimension from the current
// aspect ratio when only one of width or height is specified.
//
// When targetW is 0, it is calculated from targetH and the aspect ratio.
// When targetH is 0, it is calculated from targetW and the aspect ratio.
// When both are non-zero, the values are returned unchanged.
//
// This matches the behaviour of Go's imaging.Resize which treats a zero
// dimension as "calculate from aspect ratio".
//
// Takes targetW (int) which is the desired width, or 0 to infer from height.
// Takes targetH (int) which is the desired height, or 0 to infer from width.
// Takes currentW (int) which is the current image width in pixels.
// Takes currentH (int) which is the current image height in pixels.
//
// Returns int which is the resolved width, guaranteed non-zero when a target
// dimension was provided.
// Returns int which is the resolved height, guaranteed non-zero when a target
// dimension was provided.
func inferMissingDimension(targetW, targetH, currentW, currentH int) (resolvedW int, resolvedH int) {
	if targetW > 0 && targetH > 0 {
		return targetW, targetH
	}
	if currentW == 0 || currentH == 0 {
		return targetW, targetH
	}
	if targetW == 0 && targetH > 0 {
		targetW = targetH * currentW / currentH
		if targetW == 0 {
			targetW = 1
		}
	}
	if targetH == 0 && targetW > 0 {
		targetH = targetW * currentH / currentW
		if targetH == 0 {
			targetH = 1
		}
	}
	return targetW, targetH
}

// executeVipsResize performs the actual resize operation based on fit mode.
//
// Takes img (*vips.ImageRef) which is the image to resize.
// Takes fit (media.FitMode) which specifies the resize mode.
// Takes targetW (int) which is the target width in pixels.
// Takes targetH (int) which is the target height in pixels.
// Takes currentW (int) which is the current image width in pixels.
// Takes currentH (int) which is the current image height in pixels.
// Takes modifiers (map[string]string) which provides extra options such as
// gravity for cover mode.
//
// Returns error when the fit mode is unsupported or the resize fails.
func executeVipsResize(img *vips.ImageRef, fit media.FitMode, targetW, targetH, currentW, currentH int, modifiers map[string]string) error {
	switch fit {
	case media.FitCover:
		targetW, targetH = inferMissingDimension(targetW, targetH, currentW, currentH)
		return img.Thumbnail(targetW, targetH, parseGravity(modifiers))
	case media.FitContain, media.FitInside:
		targetW, targetH = inferMissingDimension(targetW, targetH, currentW, currentH)
		return img.Thumbnail(targetW, targetH, vips.InterestingNone)
	case media.FitFill:
		return resizeFillVips(img, targetW, targetH, currentW, currentH)
	case media.FitOutside:
		return resizeOutsideVips(img, targetW, targetH, currentW, currentH)
	default:
		return fmt.Errorf("unsupported fit mode: %s", fit)
	}
}

// applyVipsGreyscale applies a greyscale filter if enabled in the modifiers.
//
// Takes img (*vips.ImageRef) which is the image to change.
// Takes modifiers (map[string]string) which contains the filter settings.
//
// Returns error when the colour space change fails.
func applyVipsGreyscale(img *vips.ImageRef, modifiers map[string]string) error {
	greyscale, ok := modifiers["greyscale"]
	if !ok || (greyscale != "true" && greyscale != "") {
		return nil
	}
	if err := img.ToColorSpace(vips.InterpretationBW); err != nil {
		return fmt.Errorf("vips greyscale transformation failed: %w", err)
	}
	return nil
}

// applyVipsBlur applies a Gaussian blur to an image using the given sigma.
//
// Takes img (*vips.ImageRef) which is the image to blur.
// Takes modifiers (map[string]string) which contains the blur sigma value.
//
// Returns error when the blur operation fails.
func applyVipsBlur(img *vips.ImageRef, modifiers map[string]string) error {
	blurString, ok := modifiers["blur"]
	if !ok {
		return nil
	}
	sigma, err := strconv.ParseFloat(blurString, 64)
	if err != nil || sigma <= 0 {
		return nil
	}
	if err := img.GaussianBlur(sigma); err != nil {
		return fmt.Errorf("vips blur transformation failed: %w", err)
	}
	return nil
}

// applyVipsSharpen applies a sharpening filter to an image using the given
// sigma value.
//
// Takes img (*vips.ImageRef) which is the image to sharpen.
// Takes modifiers (map[string]string) which contains the sharpen sigma value.
//
// Returns error when the sharpen operation fails.
func applyVipsSharpen(img *vips.ImageRef, modifiers map[string]string) error {
	sharpenString, ok := modifiers["sharpen"]
	if !ok {
		return nil
	}
	sigma, err := strconv.ParseFloat(sharpenString, 64)
	if err != nil || sigma <= 0 {
		return nil
	}
	if err := img.Sharpen(sigma, sharpenM1, sharpenM2); err != nil {
		return fmt.Errorf("vips sharpen transformation failed: %w", err)
	}
	return nil
}

// applyVipsRotation applies rotation transformation (90, 180, or 270 degrees).
//
// Takes img (*vips.ImageRef) which is the image to rotate.
// Takes modifiers (map[string]string) which contains the rotation angle under
// the "rotate" key.
//
// Returns error when the vips rotation operation fails.
func applyVipsRotation(img *vips.ImageRef, modifiers map[string]string) error {
	rotateString, ok := modifiers["rotate"]
	if !ok {
		return nil
	}
	angle, err := strconv.Atoi(rotateString)
	if err != nil {
		return nil
	}
	var vipsAngle vips.Angle
	switch angle {
	case rotationAngle90:
		vipsAngle = vips.Angle90
	case rotationAngle180:
		vipsAngle = vips.Angle180
	case rotationAngle270:
		vipsAngle = vips.Angle270
	default:
		return nil
	}
	if err := img.Rotate(vipsAngle); err != nil {
		return fmt.Errorf("vips rotate transformation failed: %w", err)
	}
	return nil
}

// applyVipsFlip flips an image horizontally or vertically based on a modifier.
//
// Takes img (*vips.ImageRef) which is the image to flip.
// Takes modifiers (map[string]string) which contains the flip direction.
//
// Returns error when the vips flip operation fails.
func applyVipsFlip(img *vips.ImageRef, modifiers map[string]string) error {
	flip, ok := modifiers["flip"]
	if !ok {
		return nil
	}
	switch flip {
	case "horizontal", "h":
		if err := img.Flip(vips.DirectionHorizontal); err != nil {
			return fmt.Errorf("vips horizontal flip failed: %w", err)
		}
	case "vertical", "v":
		if err := img.Flip(vips.DirectionVertical); err != nil {
			return fmt.Errorf("vips vertical flip failed: %w", err)
		}
	}
	return nil
}

// applyVipsTint applies a tint colour change to an image.
//
// Takes img (*vips.ImageRef) which is the image to change.
// Takes modifiers (map[string]string) which contains the tint colour as a hex
// value in the "tint" key.
//
// Returns error when the vips modulate operation fails.
func applyVipsTint(img *vips.ImageRef, modifiers map[string]string) error {
	tint, ok := modifiers["tint"]
	if !ok || tint == "" {
		return nil
	}
	if len(tint) != tintColourLength || tint[0] != '#' {
		return nil
	}
	var r, g, b int
	if _, err := fmt.Sscanf(tint, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return nil
	}
	if err := img.Modulate(1.0, 1.0, float64(r)/255.0); err != nil {
		return fmt.Errorf("vips tint transformation failed: %w", err)
	}
	return nil
}

// applyVipsColourAdjustments applies brightness, contrast, saturation, and hue
// changes to an image.
//
// Takes img (*vips.ImageRef) which is the image to change.
// Takes modifiers (map[string]string) which contains the adjustment values.
//
// Returns error when any adjustment fails to apply.
func applyVipsColourAdjustments(img *vips.ImageRef, modifiers map[string]string) error {
	if err := applyBrightness(img, modifiers); err != nil {
		return err
	}
	if err := applyContrast(img, modifiers); err != nil {
		return err
	}
	if err := applySaturation(img, modifiers); err != nil {
		return err
	}
	return applyHue(img, modifiers)
}

// applyBrightness adjusts the brightness of the given image.
//
// Takes img (*vips.ImageRef) which is the image to adjust.
// Takes modifiers (map[string]string) which contains the brightness value.
//
// Returns error when the vips brightness change fails.
func applyBrightness(img *vips.ImageRef, modifiers map[string]string) error {
	brightnessString, ok := modifiers["brightness"]
	if !ok {
		return nil
	}
	brightness, err := strconv.ParseFloat(brightnessString, 64)
	if err != nil {
		return nil
	}
	multiplier := 1.0 + (brightness / percentBase)
	if err := img.Linear([]float64{multiplier, multiplier, multiplier}, []float64{0, 0, 0}); err != nil {
		return fmt.Errorf("vips brightness adjustment failed: %w", err)
	}
	return nil
}

// applyContrast applies contrast adjustment to the image.
//
// Takes img (*vips.ImageRef) which is the image to adjust.
// Takes modifiers (map[string]string) which contains the contrast value.
//
// Returns error when the vips contrast adjustment fails.
func applyContrast(img *vips.ImageRef, modifiers map[string]string) error {
	contrastString, ok := modifiers["contrast"]
	if !ok {
		return nil
	}
	contrast, err := strconv.ParseFloat(contrastString, 64)
	if err != nil {
		return nil
	}
	multiplier := 1.0 + (contrast / percentBase)
	offset := colourMidpoint * (1.0 - multiplier)
	if err := img.Linear([]float64{multiplier, multiplier, multiplier}, []float64{offset, offset, offset}); err != nil {
		return fmt.Errorf("vips contrast adjustment failed: %w", err)
	}
	return nil
}

// applySaturation adjusts the colour saturation of an image.
//
// Takes img (*vips.ImageRef) which is the image to modify.
// Takes modifiers (map[string]string) which contains the saturation value.
//
// Returns error when the image saturation change fails.
func applySaturation(img *vips.ImageRef, modifiers map[string]string) error {
	saturationString, ok := modifiers["saturation"]
	if !ok {
		return nil
	}
	saturation, err := strconv.ParseFloat(saturationString, 64)
	if err != nil {
		return nil
	}
	satMultiplier := max(0, 1.0+(saturation/percentBase))
	if err := img.Modulate(1.0, satMultiplier, 0); err != nil {
		return fmt.Errorf("vips saturation adjustment failed: %w", err)
	}
	return nil
}

// applyHue changes the hue of the image by a given amount.
//
// Takes img (*vips.ImageRef) which is the image to change.
// Takes modifiers (map[string]string) which holds the hue value to apply.
//
// Returns error when the image hue change fails.
func applyHue(img *vips.ImageRef, modifiers map[string]string) error {
	hueString, ok := modifiers["hue"]
	if !ok {
		return nil
	}
	hue, err := strconv.ParseFloat(hueString, 64)
	if err != nil {
		return nil
	}
	if err := img.Modulate(1.0, 1.0, hue); err != nil {
		return fmt.Errorf("vips hue rotation failed: %w", err)
	}
	return nil
}
