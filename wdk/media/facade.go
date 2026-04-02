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

package media

import (
	"context"
	"fmt"
	"io"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/piko/internal/video/video_dto"
)

// ImageTransformerPort is the interface that image processing providers must
// implement. Providers receive image data, apply transformations, and write the
// result.
type ImageTransformerPort = image_domain.TransformerPort

// ImageService is the high-level service interface for image operations.
// Providers implement ImageTransformerPort; framework code uses this type.
type ImageService = image_domain.Service

// ImageServiceConfig holds settings for the image service.
type ImageServiceConfig = image_domain.ServiceConfig

// TransformationSpec holds the settings for an image transformation.
type TransformationSpec = image_dto.TransformationSpec

// FitMode determines how an image is resized when both width and height are
// set.
type FitMode = image_dto.FitMode

const (
	// FitCover fills dimensions, cropping excess.
	FitCover = image_dto.FitCover

	// FitContain fits within dimensions, letterboxing if needed.
	FitContain = image_dto.FitContain

	// FitFill stretches to exact dimensions, ignoring aspect ratio.
	FitFill = image_dto.FitFill

	// FitInside resizes to be <= dimensions, preserving aspect ratio.
	FitInside = image_dto.FitInside

	// FitOutside resizes to be >= dimensions, preserving aspect ratio.
	FitOutside = image_dto.FitOutside
)

// ResponsiveSpec holds settings for creating responsive image variants.
type ResponsiveSpec = image_dto.ResponsiveSpec

// ResponsiveVariant represents a single image variant generated for responsive
// images.
type ResponsiveVariant = image_dto.ResponsiveVariant

// PlaceholderSpec holds settings for creating LQIP placeholders.
type PlaceholderSpec = image_dto.PlaceholderSpec

// Logger is the logging interface used by media providers.
type Logger = logger.Logger

// Attr is a structured log attribute (key-value pair).
type Attr = logger.Attr

var (
	// DefaultTransformationSpec returns a TransformationSpec with sensible
	// defaults.
	DefaultTransformationSpec = image_dto.DefaultTransformationSpec

	// DefaultImageServiceConfig returns an ImageServiceConfig with sensible
	// defaults.
	DefaultImageServiceConfig = image_domain.DefaultServiceConfig

	// DefaultPredefinedVariants returns the default set of predefined variants.
	// These are included when using ImageConfigBuilder.FromDefaults().
	DefaultPredefinedVariants = image_domain.DefaultPredefinedVariants

	// NewThumbnailSpec creates a ThumbnailSpec with default values.
	NewThumbnailSpec = video_dto.NewThumbnailSpec

	// ParseThumbnailTime parses a time string into a Duration.
	ParseThumbnailTime = video_dto.ParseThumbnailTime

	// ParseTranscodeSpec parses capability parameters into a TranscodeSpec.
	ParseTranscodeSpec = video_dto.ParseTranscodeSpec

	// DefaultVideoServiceConfig returns a VideoServiceConfig with sensible
	// defaults.
	DefaultVideoServiceConfig = video_domain.DefaultServiceConfig

	// ErrUnsupportedCodec is returned when the requested output codec is not
	// supported.
	ErrUnsupportedCodec = video_domain.ErrUnsupportedCodec

	// ErrUnsupportedFormat is returned when the input or output format is not
	// supported.
	ErrUnsupportedFormat = video_domain.ErrUnsupportedFormat

	// ErrInvalidResolution is returned when the requested output resolution is
	// invalid.
	ErrInvalidResolution = video_domain.ErrInvalidResolution

	// ErrInvalidBitrate is returned when the requested bitrate is invalid or
	// exceeds limits.
	ErrInvalidBitrate = video_domain.ErrInvalidBitrate

	// ErrInvalidFramerate is returned when the video framerate is not valid.
	ErrInvalidFramerate = video_domain.ErrInvalidFramerate

	// ErrDurationExceedsLimit is returned when the input video duration exceeds
	// limits.
	ErrDurationExceedsLimit = video_domain.ErrDurationExceedsLimit

	// ErrFileSizeExceedsLimit is returned when the input file size exceeds limits.
	ErrFileSizeExceedsLimit = video_domain.ErrFileSizeExceedsLimit

	// ErrResolutionExceedsLimit is returned when the requested resolution
	// exceeds the limit.
	ErrResolutionExceedsLimit = video_domain.ErrResolutionExceedsLimit

	// ErrTranscodingFailed is returned when the transcoding operation fails.
	ErrTranscodingFailed = video_domain.ErrTranscodingFailed

	// ErrInvalidStream is returned when the input video has no valid video stream.
	ErrInvalidStream = video_domain.ErrInvalidStream

	// ErrContextCancelled is returned when the transcoding is cancelled via
	// context.
	ErrContextCancelled = video_domain.ErrContextCancelled

	// ErrTimeout is returned when a transcoding operation exceeds the timeout.
	ErrTimeout = video_domain.ErrTimeout

	// ErrResourceExhausted is returned when system resources are exhausted.
	ErrResourceExhausted = video_domain.ErrResourceExhausted

	// ErrInvalidHLSSpec is returned when the HLS specification is invalid.
	ErrInvalidHLSSpec = video_domain.ErrInvalidHLSSpec

	// ErrSegmentationFailed is returned when video segmentation fails.
	ErrSegmentationFailed = video_domain.ErrSegmentationFailed
)

// ImageConfig holds the settings for an image hexagon.
// Pass this to piko.WithImage() after building it with ImageConfigBuilder.
type ImageConfig = image_domain.ImageConfig

// ImageConfigBuilder provides a fluent API for setting up image options.
type ImageConfigBuilder = image_domain.ImageConfigBuilder

// VariantBuilder provides a fluent API for defining image variant settings.
type VariantBuilder = image_domain.VariantBuilder

// TransformBuilder provides a fluent API for building image changes.
type TransformBuilder = image_domain.TransformBuilder

// TransformedImageResult holds the output of an image transformation.
type TransformedImageResult = image_dto.TransformedImageResult

// VideoTranscoderPort is the interface that video transcoding providers must
// implement. Providers receive video data, transcode it according to spec, and
// return the result.
type VideoTranscoderPort = video_domain.TranscoderPort

// StreamingTranscoderPort extends VideoTranscoderPort with HLS streaming
// support.
type StreamingTranscoderPort = video_domain.StreamingTranscoderPort

// VideoService is the high-level service interface for video operations. It is
// used by the framework internally; providers implement VideoTranscoderPort.
type VideoService = video_domain.Service

// VideoServiceConfig holds settings for the video service.
type VideoServiceConfig = video_domain.ServiceConfig

// TranscodeSpec defines the settings for a video transcoding operation.
type TranscodeSpec = video_dto.TranscodeSpec

// ThumbnailSpec defines the parameters for extracting a thumbnail from video.
type ThumbnailSpec = video_dto.ThumbnailSpec

// VideoCapabilities holds the properties and supported output options for a
// video.
type VideoCapabilities = video_dto.VideoCapabilities

// HLSSpec defines the settings for HLS (HTTP Live Streaming) output.
type HLSSpec = video_dto.HLSSpec

// HLSResult represents the result of an HLS video generation process.
type HLSResult = video_dto.HLSResult

// HLSVariant represents a single bitrate variant in an HLS stream.
type HLSVariant = video_dto.HLSVariant

// HLSSegment represents a single video segment in an HLS stream.
type HLSSegment = video_dto.HLSSegment

// DASHSpec defines the settings for DASH streaming output.
type DASHSpec = video_dto.DASHSpec

// DASHResult holds the output from a DASH generation operation.
type DASHResult = video_dto.DASHResult

// TranscodeResult represents the outcome of a basic transcode operation.
type TranscodeResult = video_dto.TranscodeResult

// NewService creates a new image service instance.
//
// Takes defaultTransformerName (string) which specifies the transformer to use
// when none is specified.
//
// Returns ImageService which is the configured image service ready for use.
//
// Example:
//
//	service := media.NewService("sharp")
//	transformer, _ := image_transformer_sharp.NewTransformer(config)
//	service.RegisterTransformer("sharp", transformer)
func NewService(defaultTransformerName string) ImageService {
	return image_domain.NewServiceWithDefaultTransformer(defaultTransformerName)
}

// GetDefaultService returns the image service initialised by the framework.
//
// Returns ImageService which is the configured service instance.
// Returns error when the framework has not been bootstrapped.
func GetDefaultService() (ImageService, error) {
	service, err := bootstrap.GetImageService()
	if err != nil {
		return nil, fmt.Errorf("media: get default service: %w", err)
	}
	return service, nil
}

// GetImageDimensions extracts width and height from image data
// using the framework's image service, reading only the image
// header (first few KB) and supporting all formats registered
// with Go's image package (JPEG, PNG, GIF, WebP).
//
// Takes ctx (context.Context) which controls cancellation.
// Takes input (io.Reader) which provides the source image data. The caller
// retains ownership and is responsible for closing it afterwards.
//
// Returns width (int) in pixels.
// Returns height (int) in pixels.
// Returns error when the image service is not configured or the image
// cannot be decoded.
//
// Example:
//
//	width, height, err := media.GetImageDimensions(ctx, imageReader)
//	if err != nil {
//	    return fmt.Errorf("failed to detect dimensions: %w", err)
//	}
func GetImageDimensions(ctx context.Context, input io.Reader) (width int, height int, err error) {
	service, err := GetDefaultService()
	if err != nil {
		return 0, 0, fmt.Errorf("image service not available: %w", err)
	}
	return service.GetDimensions(ctx, input)
}

// NewTransformBuilder creates a transform builder for image transformations.
//
// Takes service (ImageService) which performs the transformations.
// Takes input (io.Reader) which provides the source image data.
//
// Returns *TransformBuilder which provides a fluent interface for configuring
// and executing transformations.
//
// Example:
//
//	service := media.NewService("sharp")
//	result, err := media.NewTransformBuilder(service, reader).
//	    Size(200, 200).
//	    Format("webp").
//	    Do(ctx)
func NewTransformBuilder(service ImageService, input io.Reader) *TransformBuilder {
	return service.Transform(input)
}

// NewTransformBuilderFromDefault creates a new transform builder using the
// framework's bootstrapped image service.
//
// Takes input (io.Reader) which provides the source image data.
//
// Returns *TransformBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	builder, err := media.NewTransformBuilderFromDefault(reader)
//	if err != nil {
//	    return err
//	}
//	result, err := builder.
//	    Size(200, 200).
//	    Format("webp").
//	    Do(ctx)
func NewTransformBuilderFromDefault(input io.Reader) (*TransformBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf("media: get default service for transform: %w", err)
	}
	return NewTransformBuilder(service, input), nil
}

// Image creates a new image configuration builder with sensible defaults. Use
// this to set up the image service at startup.
//
// Example:
//
//	app := piko.New(
//	    piko.WithImage(
//	        media.Image().
//	            Provider("vips", vipsProvider).
//	            MaxFileSizeMB(50).
//	            WithVariant("thumb", media.Variant().Size(200, 200).Cover().Build()).
//	            Build(),
//	    ),
//	)
//
// Returns *ImageConfigBuilder which provides a fluent interface for setting
// up the image service.
func Image() *ImageConfigBuilder {
	return image_domain.Image()
}

// Variant creates a new variant specification builder with sensible defaults.
// Use this to define reusable transformation specifications.
//
// Example:
//
//	spec := media.Variant().
//	    Size(200, 200).
//	    Format("webp").
//	    Quality(80).
//	    Cover().
//	    Build()
//
// Returns *VariantBuilder which provides a fluent interface for building
// a TransformationSpec.
func Variant() *VariantBuilder {
	return image_domain.Variant()
}

// GetPredefinedVariants returns all registered predefined image variants.
//
// Returns nil if no variants are set up or the framework is not initialised.
//
// Returns map[string]TransformationSpec which maps variant names to their
// transformation settings.
func GetPredefinedVariants() map[string]TransformationSpec {
	return bootstrap.GetImagePredefinedVariants()
}

// GetVariantSpec retrieves a predefined variant by name.
//
// Takes name (string) which identifies the variant to look up.
//
// Returns TransformationSpec which contains the variant settings.
// Returns bool which indicates whether the variant was found.
func GetVariantSpec(name string) (TransformationSpec, bool) {
	variants := GetPredefinedVariants()
	if variants == nil {
		return TransformationSpec{}, false
	}
	spec, ok := variants[name]
	return spec, ok
}
