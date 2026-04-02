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

package image_provider_imaging

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/HugoSmits86/nativewebp"
	"github.com/disintegration/imaging"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/media"
)

const (
	// rotationAngle90 is a 90-degree clockwise rotation.
	rotationAngle90 = 90

	// rotationAngle180 is the rotation angle in degrees for a half turn.
	rotationAngle180 = 180

	// rotationAngle270 is 270 degrees clockwise.
	rotationAngle270 = 270

	// maxConcurrentTransforms is the upper limit for concurrent
	// image transformations, capped at runtime.NumCPU() to
	// prevent memory exhaustion from the pure Go WebP encoder's
	// large temporary buffers.
	maxConcurrentTransforms = 2
)

var (
	// log is the package-level logger for image_provider_imaging.
	log = logger.GetLogger("piko/media/image_provider_imaging")

	// warnOnce guards single logging of the development-only warning.
	warnOnce sync.Once

	_ media.ImageTransformerPort = (*Provider)(nil)
)

// Config holds configuration options for the imaging provider.
type Config struct {
	// ImageServiceConfig holds security settings for input validation and resource
	// limits. If nil, default configuration will be used.
	media.ImageServiceConfig
}

// Provider implements the ImageTransformerPort interface using pure Go
// libraries. WebP encoding is lossless-only; use VipsTransformer for lossy
// WebP and better performance.
type Provider struct {
	// semaphore limits how many transforms run at the same time to prevent
	// memory exhaustion. The nativewebp encoder uses large temporary buffers
	// (100-500MB per image), so concurrent transforms must be limited.
	semaphore chan struct{}

	// config holds settings for image processing and size limits.
	config media.ImageServiceConfig
}

// NewProvider creates a pure Go image transformer for development and testing.
// Unlike VipsTransformer, this requires no external dependencies or setup.
//
// Takes config (Config) which specifies the image processing settings.
//
// Returns *Provider which is the configured transformer ready for use.
func NewProvider(config Config) *Provider {
	warnOnce.Do(func() {
		_, l := logger.From(context.Background(), log)
		l.Warn("Using pure Go image transformer (image_provider_imaging). " +
			"This is intended for development/testing only. " +
			"For production, use image_provider_vips which provides 10-20x lower memory usage, " +
			"lossy WebP (smaller files), and AVIF support.")
	})

	serviceConfig := config.ImageServiceConfig
	if serviceConfig.MaxFileSizeBytes == 0 {
		serviceConfig = media.DefaultImageServiceConfig()
	}

	return &Provider{
		config:    serviceConfig,
		semaphore: make(chan struct{}, min(maxConcurrentTransforms, runtime.NumCPU())),
	}
}

// Transform applies the specified transformations to the input image stream. It
// decodes the image, performs resizing/cropping and modifiers, then encodes to
// the target format.
//
// Takes input (io.Reader) which provides the source image data.
// Takes output (io.Writer) which receives the transformed image data.
// Takes spec (media.TransformationSpec) which defines the transformations to
// apply.
//
// Returns string which is the resulting image format after transformation.
// Returns error when the spec is invalid, the format is disallowed, or
// decoding/encoding fails.
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

	srcImage, err := p.decodeImage(ctx, input)
	if err != nil {
		return "", err
	}

	if err := p.validateImageSecurity(ctx, srcImage); err != nil {
		return "", err
	}

	transformedImage := srcImage
	if spec.Width > 0 || spec.Height > 0 {
		transformedImage = p.resizeOrCrop(srcImage, spec)
	}

	if err := p.applyModifiers(&transformedImage, spec.Modifiers); err != nil {
		return "", err
	}

	return p.encode(transformedImage, output, spec)
}

// GetSupportedFormats returns the list of output formats that the pure Go
// imaging library can generate.
//
// Returns []string which contains the supported format names. WebP is
// lossless-only (VP8L); for lossy WebP, use VipsTransformer. AVIF is not
// supported by the pure Go stack.
func (*Provider) GetSupportedFormats() []string {
	return []string{"jpeg", "jpg", "png", "webp", "gif"}
}

// GetSupportedModifiers returns the list of transformation modifiers
// supported by the pure Go imaging library. Advanced modifiers like hue,
// tint, gravity, and radius are not supported.
//
// Returns []string which contains the names of supported modifiers.
func (*Provider) GetSupportedModifiers() []string {
	return []string{
		"greyscale", "blur", "sharpen", "rotate", "flip",
		"brightness", "contrast", "saturation",
	}
}

// GetDimensions extracts width and height from image data using lightweight
// header decoding. Uses Go's standard library image.DecodeConfig() for
// minimal overhead.
//
// Takes input (io.Reader) which provides the source image data.
//
// Returns width (int) in pixels.
// Returns height (int) in pixels.
// Returns error when the image cannot be decoded or format is
// unsupported.
func (*Provider) GetDimensions(_ context.Context, input io.Reader) (width int, height int, err error) {
	config, _, err := image.DecodeConfig(input)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image header: %w", err)
	}

	return config.Width, config.Height, nil
}

// decodeImage decodes the input stream into an image.Image, applying size
// limits.
//
// Takes ctx (context.Context) which provides the request-scoped logger.
// Takes input (io.Reader) which provides the image data to decode.
//
// Returns image.Image which is the decoded image with auto-orientation applied.
// Returns error when the input exceeds the size limit or cannot be decoded.
func (p *Provider) decodeImage(ctx context.Context, input io.Reader) (image.Image, error) {
	limitedInput := image_domain.NewLimitedReader(input, p.config.MaxFileSizeBytes)
	srcImage, err := imaging.Decode(limitedInput, imaging.AutoOrientation(true))
	if err != nil {
		if errors.Is(err, image_domain.ErrSizeLimitExceeded) {
			_, l := logger.From(ctx, log)
			l.Warn("Rejected transformation due to file size limit",
				logger.Int64("max_bytes", p.config.MaxFileSizeBytes),
				logger.Error(err))
		}
		return nil, fmt.Errorf("failed to decode input image: %w", err)
	}
	return srcImage, nil
}

// validateImageSecurity validates image dimensions and pixel count against
// security limits.
//
// Takes img (image.Image) which is the image to validate.
//
// Returns error when the image exceeds dimension or pixel count limits.
func (p *Provider) validateImageSecurity(ctx context.Context, img image.Image) error {
	ctx, l := logger.From(ctx, log)
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

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
// Takes img (image.Image) which is the source image to transform.
// Takes spec (TransformationSpec) which defines the target dimensions and fit
// mode.
//
// Returns image.Image which is the transformed image, or the original if no
// resize is needed.
func (*Provider) resizeOrCrop(img image.Image, spec media.TransformationSpec) image.Image {
	targetWidth, targetHeight := calculateTargetDimensions(spec.Width, spec.Height, spec.AspectRatio)

	if targetWidth == 0 && targetHeight == 0 {
		return img
	}

	bounds := img.Bounds()
	currentWidth := bounds.Dx()
	currentHeight := bounds.Dy()

	if spec.WithoutEnlargement {
		targetWidth = applyWithoutEnlargement(targetWidth, currentWidth)
		targetHeight = applyWithoutEnlargement(targetHeight, currentHeight)
		if targetWidth == 0 && targetHeight == 0 {
			return img
		}
	}

	switch spec.Fit {
	case media.FitCover:
		return imaging.Fill(img, targetWidth, targetHeight, imaging.Center, imaging.Lanczos)
	case media.FitContain, media.FitInside:
		return resizeContain(img, targetWidth, targetHeight)
	case media.FitFill:
		return resizeFill(img, targetWidth, targetHeight, currentWidth, currentHeight)
	case media.FitOutside:
		return resizeOutside(img, targetWidth, targetHeight, currentWidth, currentHeight)
	default:
		return imaging.Fit(img, targetWidth, targetHeight, imaging.Lanczos)
	}
}

// applyModifiers applies transformations to an image using the modifiers map.
//
// Takes img (*image.Image) which is the image to transform.
// Takes modifiers (map[string]string) which specifies transformations such as
// greyscale, blur, sharpen, rotation, flip, and colour adjustments.
//
// Returns error when a transformation fails.
//
// Hue rotation is not available in the pure Go imaging library. Use
// provider="vips" for hue rotation support.
func (*Provider) applyModifiers(img *image.Image, modifiers map[string]string) error {
	applyGreyscale(img, modifiers)
	applyBlur(img, modifiers)
	applySharpen(img, modifiers)
	applyRotation(img, modifiers)
	applyFlip(img, modifiers)
	applyColourAdjustments(img, modifiers)
	return nil
}

// encode writes the transformed image to the output in the specified format.
//
// Takes img (image.Image) which is the transformed image to encode.
// Takes output (io.Writer) which receives the encoded image data.
// Takes spec (media.TransformationSpec) which specifies the output format
// and quality settings.
//
// Returns string which is the MIME type of the encoded image.
// Returns error when encoding fails or the format is unsupported.
func (*Provider) encode(img image.Image, output io.Writer, spec media.TransformationSpec) (string, error) {
	outputFormat := strings.ToLower(spec.Format)

	switch outputFormat {
	case "jpeg", "jpg":
		if err := jpeg.Encode(output, img, &jpeg.Options{Quality: spec.Quality}); err != nil {
			return "", fmt.Errorf("failed to encode JPEG: %w", err)
		}
		return "image/jpeg", nil

	case "png":
		encoder := png.Encoder{
			CompressionLevel: png.DefaultCompression,
			BufferPool:       nil,
		}
		if err := encoder.Encode(output, img); err != nil {
			return "", fmt.Errorf("failed to encode PNG: %w", err)
		}
		return "image/png", nil

	case "webp":
		if err := nativewebp.Encode(output, img, &nativewebp.Options{UseExtendedFormat: true}); err != nil {
			return "", fmt.Errorf("failed to encode WebP: %w", err)
		}
		return "image/webp", nil

	default:
		return "", fmt.Errorf("unsupported output format for pure Go imaging provider: '%s'", outputFormat)
	}
}

// parseAspectRatio parses an aspect ratio string like "16:9" and returns the
// ratio as width divided by height.
//
// Takes ar (string) which specifies the aspect ratio in "width:height" format.
//
// Returns float64 which is the aspect ratio as a decimal value.
// Returns error when the format is invalid or the values are not positive.
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

// calculateTargetDimensions computes the final target dimensions based on
// aspect ratio.
//
// When aspectRatio is empty or invalid, returns the original width and height
// unchanged. When only width is provided, calculates height from the ratio.
// When only height is provided, calculates width from the ratio.
//
// Takes width (int) which specifies the input width, or zero if unknown.
// Takes height (int) which specifies the input height, or zero if unknown.
// Takes aspectRatio (string) which defines the desired ratio (e.g. "16:9").
//
// Returns targetWidth (int) which is the calculated or original width.
// Returns targetHeight (int) which is the calculated or original height.
func calculateTargetDimensions(width, height int, aspectRatio string) (targetWidth int, targetHeight int) {
	if aspectRatio == "" {
		return width, height
	}

	ratio, err := parseAspectRatio(aspectRatio)
	if err != nil {
		return width, height
	}

	if width > 0 && height == 0 {
		return width, int(float64(width) / ratio)
	}
	if height > 0 && width == 0 {
		return int(float64(height) * ratio), height
	}
	return width, height
}

// applyWithoutEnlargement clamps a target dimension to not exceed the current
// dimension.
//
// Takes target (int) which is the desired dimension value.
// Takes current (int) which is the existing dimension to compare against.
//
// Returns int which is the clamped value, either target or current if target
// would cause enlargement.
func applyWithoutEnlargement(target, current int) int {
	if target > 0 && target > current {
		return current
	}
	return target
}

// resizeOutside calculates dimensions for "outside" fit mode and resizes the
// image to the smallest size that is greater than or equal to the target.
//
// Takes img (image.Image) which is the source image to resize.
// Takes targetW (int) which is the desired minimum width.
// Takes targetH (int) which is the desired minimum height.
// Takes currentW (int) which is the current width of the image.
// Takes currentH (int) which is the current height of the image.
//
// Returns image.Image which is the resized image using Lanczos resampling.
func resizeOutside(img image.Image, targetW, targetH, currentW, currentH int) image.Image {
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
	return imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
}

// resizeContain fits an image within the given bounds for contain/inside modes.
//
// Takes img (image.Image) which is the source image to resize.
// Takes targetW (int) which is the target width in pixels.
// Takes targetH (int) which is the target height in pixels.
//
// Returns image.Image which is the resized image that fits within the target
// dimensions while preserving aspect ratio.
func resizeContain(img image.Image, targetW, targetH int) image.Image {
	if targetW == 0 || targetH == 0 {
		return imaging.Resize(img, targetW, targetH, imaging.Lanczos)
	}
	return imaging.Fit(img, targetW, targetH, imaging.Lanczos)
}

// resizeFill resizes an image to the exact target size by stretching.
//
// Takes img (image.Image) which is the source image to resize.
// Takes targetW (int) which is the target width, or zero to keep the current
// width.
// Takes targetH (int) which is the target height, or zero to keep the current
// height.
// Takes currentW (int) which is the current width of the image.
// Takes currentH (int) which is the current height of the image.
//
// Returns image.Image which is the resized image at the target size.
func resizeFill(img image.Image, targetW, targetH, currentW, currentH int) image.Image {
	if targetW == 0 {
		targetW = currentW
	}
	if targetH == 0 {
		targetH = currentH
	}
	return imaging.Resize(img, targetW, targetH, imaging.Lanczos)
}

// applyGreyscale converts an image to greyscale if the filter is enabled.
//
// Takes img (*image.Image) which is the image to modify in place.
// Takes modifiers (map[string]string) which holds filter settings; the
// "greyscale" key enables the filter when set to "true" or left empty.
func applyGreyscale(img *image.Image, modifiers map[string]string) {
	greyscale, ok := modifiers["greyscale"]
	if ok && (greyscale == "true" || greyscale == "") {
		*img = imaging.Grayscale(*img)
	}
}

// applyBlur applies a Gaussian blur to an image using the given sigma value.
//
// Takes img (*image.Image) which is the image to blur, changed in place.
// Takes modifiers (map[string]string) which holds the blur sigma value under
// the "blur" key.
func applyBlur(img *image.Image, modifiers map[string]string) {
	blurString, ok := modifiers["blur"]
	if !ok {
		return
	}
	sigma, err := strconv.ParseFloat(blurString, 64)
	if err == nil && sigma > 0 {
		*img = imaging.Blur(*img, sigma)
	}
}

// applySharpen applies a sharpening filter to an image.
//
// Takes img (*image.Image) which is the image to sharpen.
// Takes modifiers (map[string]string) which contains the sharpen sigma value.
func applySharpen(img *image.Image, modifiers map[string]string) {
	sharpenString, ok := modifiers["sharpen"]
	if !ok {
		return
	}
	sigma, err := strconv.ParseFloat(sharpenString, 64)
	if err == nil && sigma > 0 {
		*img = imaging.Sharpen(*img, sigma)
	}
}

// applyRotation rotates an image by 90, 180, or 270 degrees.
//
// Takes img (*image.Image) which is the image to rotate.
// Takes modifiers (map[string]string) which holds the rotation angle under
// the "rotate" key.
func applyRotation(img *image.Image, modifiers map[string]string) {
	rotateString, ok := modifiers["rotate"]
	if !ok {
		return
	}
	angle, err := strconv.Atoi(rotateString)
	if err != nil {
		return
	}
	switch angle {
	case rotationAngle90:
		*img = imaging.Rotate90(*img)
	case rotationAngle180:
		*img = imaging.Rotate180(*img)
	case rotationAngle270:
		*img = imaging.Rotate270(*img)
	}
}

// applyFlip flips an image horizontally or vertically based on the modifier.
//
// Takes img (*image.Image) which is the image to flip in place.
// Takes modifiers (map[string]string) which contains the flip direction
// ("horizontal", "h", "vertical", or "v").
func applyFlip(img *image.Image, modifiers map[string]string) {
	flip, ok := modifiers["flip"]
	if !ok {
		return
	}
	switch flip {
	case "horizontal", "h":
		*img = imaging.FlipH(*img)
	case "vertical", "v":
		*img = imaging.FlipV(*img)
	}
}

// applyColourAdjustments changes the brightness, contrast, and saturation of
// an image.
//
// Takes img (*image.Image) which is changed in place with the new values.
// Takes modifiers (map[string]string) which holds the adjustment values keyed
// by "brightness", "contrast", and "saturation".
func applyColourAdjustments(img *image.Image, modifiers map[string]string) {
	if brightnessString, ok := modifiers["brightness"]; ok {
		if brightness, err := strconv.ParseFloat(brightnessString, 64); err == nil {
			*img = imaging.AdjustBrightness(*img, brightness)
		}
	}
	if contrastString, ok := modifiers["contrast"]; ok {
		if contrast, err := strconv.ParseFloat(contrastString, 64); err == nil {
			*img = imaging.AdjustContrast(*img, contrast)
		}
	}
	if saturationString, ok := modifiers["saturation"]; ok {
		if saturation, err := strconv.ParseFloat(saturationString, 64); err == nil {
			*img = imaging.AdjustSaturation(*img, saturation)
		}
	}
}
