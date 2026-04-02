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

package video_domain

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/video/video_dto"
)

// service provides video transcoding by implementing the video Service and
// health Probe interfaces.
type service struct {
	// transcoders maps provider names to their transcoder implementations.
	transcoders map[string]TranscoderPort

	// streamingTranscoder handles HLS transcoding; nil means HLS is not supported.
	streamingTranscoder StreamingTranscoderPort

	// defaultProvider is the name of the provider to use when none is given.
	defaultProvider string

	// config holds settings for validation and service behaviour.
	config ServiceConfig
}

var _ Service = (*service)(nil)

// ServiceConfig holds settings for creating a new video service.
type ServiceConfig struct {
	// MaxVideoPixels is the maximum allowed total pixel count
	// (width * height), preventing memory exhaustion from
	// extremely large videos (default: 8,294,400 or roughly 4K).
	MaxVideoPixels int64

	// MaxFileSizeBytes is the maximum allowed input file size in bytes.
	// Default: 500 MB.
	MaxFileSizeBytes int64

	// TranscodeTimeout is the maximum time allowed for a single transcode
	// operation. Default is 5 minutes.
	TranscodeTimeout time.Duration

	// MaxVideoWidth is the maximum allowed width for a video in pixels.
	// Default: 3840 (4K).
	MaxVideoWidth int

	// MaxVideoHeight is the maximum allowed height for a video in pixels.
	// Defaults to 2160 (4K) if not set or if set to zero or below.
	MaxVideoHeight int

	// MaxBitrate is the maximum allowed bitrate in bits per second.
	// Values less than or equal to zero use the default of 20000000 (20 Mbps).
	MaxBitrate int

	// MaxFramerate is the maximum allowed framerate for transcoding.
	// Default: 60.
	MaxFramerate float64
}

// Transcode performs a streaming transcode using the selected provider.
//
// Takes input (io.Reader) which provides the source video data to transcode.
// Takes params (map[string]string) which specifies the transcode settings.
//
// Returns io.ReadCloser which provides the transcoded video stream.
// Returns error when parameters are invalid, spec validation fails, or
// transcoding fails.
func (s *service) Transcode(ctx context.Context, input io.Reader, params map[string]string) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)

	spec, err := video_dto.ParseTranscodeSpec(params)
	if err != nil {
		return nil, fmt.Errorf("invalid transcode parameters: %w", err)
	}

	if err := s.validateSpec(spec); err != nil {
		return nil, fmt.Errorf("transcode spec validation failed: %w", err)
	}

	tc, providerName, err := s.selectTranscoder()
	if err != nil {
		return nil, fmt.Errorf("selecting transcoder for video transcode: %w", err)
	}

	recordTranscodeMetrics(ctx, providerName, spec, true)

	l.Trace("Starting video transcode",
		logger_domain.String(metricAttributeProvider, providerName),
		logger_domain.String(metricAttributeCodec, spec.Codec),
		logger_domain.Int("width", spec.Width),
		logger_domain.Int("height", spec.Height),
	)

	startTime := time.Now()
	result, err := tc.Transcode(ctx, input, spec)
	if err != nil {
		recordTranscodeError(ctx, providerName, "transcode_failed")
		return nil, fmt.Errorf("transcoding failed: %w", err)
	}

	duration := time.Since(startTime).Milliseconds()
	recordTranscodeDuration(ctx, providerName, spec, duration)

	l.Trace("Transcode stream started successfully",
		logger_domain.String(metricAttributeProvider, providerName),
		logger_domain.Duration("initDuration", time.Duration(duration)*time.Millisecond),
	)

	return result, nil
}

// ExtractCapabilities analyses input video and returns metadata.
//
// Takes input (io.Reader) which provides the video data to analyse.
//
// Returns video_dto.VideoCapabilities which contains the extracted metadata.
// Returns error when transcoder selection fails or capability extraction fails.
func (s *service) ExtractCapabilities(ctx context.Context, input io.Reader) (video_dto.VideoCapabilities, error) {
	ctx, l := logger_domain.From(ctx, log)

	tc, providerName, err := s.selectTranscoder()
	if err != nil {
		return video_dto.VideoCapabilities{}, err
	}

	l.Trace("Extracting video capabilities",
		logger_domain.String(metricAttributeProvider, providerName),
	)

	caps, err := tc.ExtractCapabilities(ctx, input)
	if err != nil {
		recordTranscodeError(ctx, providerName, "extract_capabilities_failed")
		return video_dto.VideoCapabilities{}, fmt.Errorf("failed to extract capabilities: %w", err)
	}

	l.Trace("Video capabilities extracted",
		logger_domain.Int("width", caps.Width),
		logger_domain.Int("height", caps.Height),
		logger_domain.String(metricAttributeCodec, caps.VideoCodec),
		logger_domain.Duration("duration", caps.Duration),
	)

	return caps, nil
}

// TranscodeHLS generates HLS manifest and segments.
//
// Takes input (io.Reader) which provides the video data to transcode.
// Takes spec (video_dto.HLSSpec) which defines the HLS output settings.
//
// Returns video_dto.HLSResult which contains the manifest and segment data.
// Returns error when the streaming transcoder is nil or transcoding fails.
func (s *service) TranscodeHLS(ctx context.Context, input io.Reader, spec video_dto.HLSSpec) (video_dto.HLSResult, error) {
	ctx, l := logger_domain.From(ctx, log)

	if s.streamingTranscoder == nil {
		return video_dto.HLSResult{}, ErrHLSNotSupported
	}

	l.Trace("Starting HLS transcode",
		logger_domain.Int("variantBitratesCount", len(spec.VariantBitrates)),
	)

	result, err := s.streamingTranscoder.TranscodeHLS(ctx, input, spec)
	if err != nil {
		recordTranscodeError(ctx, s.defaultProvider, "hls_transcode_failed")
		return video_dto.HLSResult{}, fmt.Errorf("HLS transcoding failed: %w", err)
	}

	l.Trace("HLS transcode completed",
		logger_domain.Int("totalSegments", result.TotalSegments),
		logger_domain.Int("variantCount", len(result.Variants)),
	)

	return result, nil
}

// ExtractThumbnail extracts a frame from a video and returns it as an image.
//
// Takes input (io.Reader) which provides the video data to extract from.
// Takes spec (video_dto.ThumbnailSpec) which defines the extraction parameters.
//
// Returns io.ReadCloser which streams the encoded image data.
// Returns error when extraction fails or the video cannot be decoded.
func (s *service) ExtractThumbnail(ctx context.Context, input io.Reader, spec video_dto.ThumbnailSpec) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)

	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("invalid thumbnail spec: %w", err)
	}

	spec = spec.WithDefaults()

	tc, providerName, err := s.selectTranscoder()
	if err != nil {
		return nil, fmt.Errorf("selecting transcoder for thumbnail extraction: %w", err)
	}

	l.Trace("Extracting video thumbnail",
		logger_domain.String(metricAttributeProvider, providerName),
		logger_domain.Duration("timestamp", spec.Timestamp),
		logger_domain.String("format", spec.Format),
	)

	startTime := time.Now()
	result, err := tc.ExtractThumbnail(ctx, input, spec)
	if err != nil {
		recordTranscodeError(ctx, providerName, "extract_thumbnail_failed")
		return nil, fmt.Errorf("failed to extract thumbnail: %w", err)
	}

	l.Trace("Thumbnail extracted",
		logger_domain.String(metricAttributeProvider, providerName),
		logger_domain.Duration("duration", time.Since(startTime)),
	)

	return result, nil
}

// Name returns the service identifier and implements the
// healthprobe_domain.Probe interface.
//
// Returns string which is the service name "VideoService".
func (*service) Name() string {
	return "VideoService"
}

// Check implements the healthprobe_domain.Probe interface.
// It verifies the video transcoding pipeline is functional.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type
// of health check to perform.
//
// Returns healthprobe_dto.Status which contains the health check result
// including transcoder count and HLS support status.
func (s *service) Check(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	transcoderCount := len(s.transcoders)

	state := healthprobe_dto.StateHealthy
	message := fmt.Sprintf("Video service operational with %d transcoder(s)", transcoderCount)

	if transcoderCount == 0 {
		state = healthprobe_dto.StateUnhealthy
		message = "No video transcoders registered"
	}

	hlsSupported := s.streamingTranscoder != nil
	if hlsSupported {
		message += " (HLS supported)"
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

// selectTranscoder resolves the transcoder to use. Currently uses the default
// provider; future enhancement could support provider selection.
//
// Returns TranscoderPort which is the resolved transcoder instance.
// Returns string which is the name of the selected provider.
// Returns error when the default provider is not found.
func (s *service) selectTranscoder() (TranscoderPort, string, error) {
	tc, ok := s.transcoders[s.defaultProvider]
	if !ok {
		return nil, "", fmt.Errorf("video provider '%s' not found", s.defaultProvider)
	}
	return tc, s.defaultProvider, nil
}

// validateSpec checks a transcode spec against the service limits.
//
// Takes spec (video_dto.TranscodeSpec) which contains the video parameters to
// validate.
//
// Returns error when any dimension, bitrate, or framerate exceeds the
// configured maximum limits.
func (s *service) validateSpec(spec video_dto.TranscodeSpec) error {
	if spec.Width > s.config.MaxVideoWidth {
		return fmt.Errorf("width %d exceeds maximum %d", spec.Width, s.config.MaxVideoWidth)
	}
	if spec.Height > s.config.MaxVideoHeight {
		return fmt.Errorf("height %d exceeds maximum %d", spec.Height, s.config.MaxVideoHeight)
	}

	if spec.Width > 0 && spec.Height > 0 {
		pixels := int64(spec.Width) * int64(spec.Height)
		if pixels > s.config.MaxVideoPixels {
			return fmt.Errorf("total pixels %d exceeds maximum %d", pixels, s.config.MaxVideoPixels)
		}
	}

	if spec.Bitrate > s.config.MaxBitrate {
		return fmt.Errorf("bitrate %d exceeds maximum %d", spec.Bitrate, s.config.MaxBitrate)
	}

	if spec.Framerate > s.config.MaxFramerate {
		return fmt.Errorf("framerate %.2f exceeds maximum %.2f", spec.Framerate, s.config.MaxFramerate)
	}

	return nil
}

// DefaultServiceConfig returns a ServiceConfig with sensible production
// defaults.
//
// Returns ServiceConfig which contains default values for video processing
// limits, including dimensions, file size, timeout, bitrate, and frame rate.
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		MaxVideoWidth:    defaultMaxVideoWidth,
		MaxVideoHeight:   defaultMaxVideoHeight,
		MaxVideoPixels:   defaultMaxVideoPixels,
		MaxFileSizeBytes: defaultMaxFileSizeBytes,
		TranscodeTimeout: defaultTranscodeTimeout,
		MaxBitrate:       defaultMaxBitrate,
		MaxFramerate:     defaultMaxFramerate,
	}
}

// NewService creates a new video service.
//
// It requires a registry of transcoders for video processing and a default
// provider.
//
// Takes transcoders (map[string]TranscoderPort) which provides the registry
// of available transcoders for video processing.
// Takes defaultProvider (string) which specifies the default transcoder to
// use.
// Takes config (ServiceConfig) which specifies the service configuration
// settings.
//
// Returns Service which is the configured video service ready for use.
// Returns error when the transcoders or default provider are invalid.
func NewService(transcoders map[string]TranscoderPort, defaultProvider string, config ServiceConfig) (Service, error) {
	if err := validateServiceInputs(transcoders, defaultProvider); err != nil {
		return nil, fmt.Errorf("validating video service inputs: %w", err)
	}

	config = applyConfigDefaults(config)

	var streamingTranscoder StreamingTranscoderPort
	if tc, ok := transcoders[defaultProvider]; ok {
		if stc, ok := tc.(StreamingTranscoderPort); ok {
			streamingTranscoder = stc
		}
	}

	return &service{
		transcoders:         transcoders,
		streamingTranscoder: streamingTranscoder,
		config:              config,
		defaultProvider:     defaultProvider,
	}, nil
}

// validateServiceInputs checks that the required service inputs are valid.
//
// Takes transcoders (map[string]TranscoderPort) which provides the available
// video transcoders keyed by provider name.
// Takes defaultProvider (string) which specifies the provider to use by
// default.
//
// Returns error when no transcoders are provided, the default provider is
// empty, or the default provider is not found in the transcoders map.
func validateServiceInputs(transcoders map[string]TranscoderPort, defaultProvider string) error {
	if len(transcoders) == 0 {
		return errNoTranscoders
	}
	if defaultProvider == "" {
		return errDefaultProviderEmpty
	}
	if _, ok := transcoders[defaultProvider]; !ok {
		return fmt.Errorf("default video provider '%s' is not registered", defaultProvider)
	}
	return nil
}

// applyConfigDefaults fills in missing settings with default values.
//
// Takes config (ServiceConfig) which holds the settings to check and fill.
//
// Returns ServiceConfig which is a copy with defaults set for any empty fields.
func applyConfigDefaults(config ServiceConfig) ServiceConfig {
	defaults := DefaultServiceConfig()

	if config.MaxVideoWidth <= 0 {
		config.MaxVideoWidth = defaults.MaxVideoWidth
	}
	if config.MaxVideoHeight <= 0 {
		config.MaxVideoHeight = defaults.MaxVideoHeight
	}
	if config.MaxVideoPixels <= 0 {
		config.MaxVideoPixels = defaults.MaxVideoPixels
	}
	if config.MaxFileSizeBytes <= 0 {
		config.MaxFileSizeBytes = defaults.MaxFileSizeBytes
	}
	if config.TranscodeTimeout <= 0 {
		config.TranscodeTimeout = defaults.TranscodeTimeout
	}
	if config.MaxBitrate <= 0 {
		config.MaxBitrate = defaults.MaxBitrate
	}
	if config.MaxFramerate <= 0 {
		config.MaxFramerate = defaults.MaxFramerate
	}

	return config
}

// recordTranscodeMetrics records the start of a transcode operation.
//
// Takes provider (string) which identifies the transcoding service.
// Takes spec (video_dto.TranscodeSpec) which contains codec and format details.
// Takes incrementCount (bool) which controls whether to increment the counter.
func recordTranscodeMetrics(ctx context.Context, provider string, spec video_dto.TranscodeSpec, incrementCount bool) {
	attrs := metric.WithAttributes(
		attribute.String(metricAttributeProvider, provider),
		attribute.String(metricAttributeCodec, spec.Codec),
		attribute.String("format", spec.Format),
	)

	if incrementCount {
		transcodeCount.Add(ctx, 1, attrs)
	}
}

// recordTranscodeDuration records how long a transcode operation took.
//
// Takes provider (string) which identifies the transcoding service used.
// Takes spec (video_dto.TranscodeSpec) which describes the transcode settings.
// Takes durationMs (int64) which is the transcode duration in milliseconds.
func recordTranscodeDuration(ctx context.Context, provider string, spec video_dto.TranscodeSpec, durationMs int64) {
	transcodeDuration.Record(ctx, float64(durationMs),
		metric.WithAttributes(
			attribute.String(metricAttributeProvider, provider),
			attribute.String(metricAttributeCodec, spec.Codec),
			attribute.String("format", spec.Format),
		),
	)
}

// recordTranscodeError records a transcode error metric with provider and error
// type attributes.
//
// Takes provider (string) which identifies the transcoding provider.
// Takes errorType (string) which specifies the type of error that occurred.
func recordTranscodeError(ctx context.Context, provider, errorType string) {
	transcodeErrorCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(metricAttributeProvider, provider),
			attribute.String("error_type", errorType),
		),
	)
}
