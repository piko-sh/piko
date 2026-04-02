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
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/video/video_dto"
)

func TestDefaultServiceConfig(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()

	assert.Equal(t, 3840, config.MaxVideoWidth)
	assert.Equal(t, 2160, config.MaxVideoHeight)
	assert.Equal(t, int64(8_294_400), config.MaxVideoPixels)
	assert.Equal(t, int64(500*1024*1024), config.MaxFileSizeBytes)
	assert.Equal(t, 5*time.Minute, config.TranscodeTimeout)
	assert.Equal(t, 20_000_000, config.MaxBitrate)
	assert.Equal(t, 60.0, config.MaxFramerate)
}

func TestValidateServiceInputs(t *testing.T) {
	t.Parallel()

	transcoders := map[string]TranscoderPort{
		"ffmpeg": TranscoderPort(nil),
	}

	t.Run("no transcoders", func(t *testing.T) {
		t.Parallel()

		err := validateServiceInputs(nil, "ffmpeg")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one video transcoder")
	})

	t.Run("empty provider", func(t *testing.T) {
		t.Parallel()

		err := validateServiceInputs(transcoders, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "default video provider cannot be empty")
	})

	t.Run("provider not in map", func(t *testing.T) {
		t.Parallel()

		err := validateServiceInputs(transcoders, "unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		err := validateServiceInputs(transcoders, "ffmpeg")
		assert.NoError(t, err)
	})
}

func TestApplyConfigDefaults(t *testing.T) {
	t.Parallel()

	t.Run("all-zero config fills defaults", func(t *testing.T) {
		t.Parallel()

		result := applyConfigDefaults(ServiceConfig{})
		defaults := DefaultServiceConfig()

		assert.Equal(t, defaults.MaxVideoWidth, result.MaxVideoWidth)
		assert.Equal(t, defaults.MaxVideoHeight, result.MaxVideoHeight)
		assert.Equal(t, defaults.MaxVideoPixels, result.MaxVideoPixels)
		assert.Equal(t, defaults.MaxFileSizeBytes, result.MaxFileSizeBytes)
		assert.Equal(t, defaults.TranscodeTimeout, result.TranscodeTimeout)
		assert.Equal(t, defaults.MaxBitrate, result.MaxBitrate)
		assert.Equal(t, defaults.MaxFramerate, result.MaxFramerate)
	})

	t.Run("partial config preserves set values", func(t *testing.T) {
		t.Parallel()

		custom := ServiceConfig{
			MaxVideoWidth: 1920,
			MaxBitrate:    10_000_000,
		}
		result := applyConfigDefaults(custom)

		assert.Equal(t, 1920, result.MaxVideoWidth)
		assert.Equal(t, 10_000_000, result.MaxBitrate)

		assert.Equal(t, 2160, result.MaxVideoHeight)
		assert.Equal(t, int64(8_294_400), result.MaxVideoPixels)
	})
}

func TestValidateSpec(t *testing.T) {
	t.Parallel()

	service := &service{
		config: DefaultServiceConfig(),
	}

	tests := []struct {
		name    string
		wantErr string
		spec    video_dto.TranscodeSpec
	}{
		{
			name:    "valid spec",
			spec:    video_dto.TranscodeSpec{Width: 1920, Height: 1080, Bitrate: 5_000_000, Framerate: 30},
			wantErr: "",
		},
		{
			name:    "width too large",
			spec:    video_dto.TranscodeSpec{Width: 5000, Height: 1080},
			wantErr: "width 5000 exceeds maximum",
		},
		{
			name:    "height too large",
			spec:    video_dto.TranscodeSpec{Width: 1920, Height: 3000},
			wantErr: "height 3000 exceeds maximum",
		},
		{
			name:    "pixels exactly at max are valid",
			spec:    video_dto.TranscodeSpec{Width: 3840, Height: 2160},
			wantErr: "",
		},
		{
			name:    "bitrate too high",
			spec:    video_dto.TranscodeSpec{Bitrate: 25_000_000},
			wantErr: "bitrate 25000000 exceeds maximum",
		},
		{
			name:    "framerate too high",
			spec:    video_dto.TranscodeSpec{Framerate: 120},
			wantErr: "framerate 120.00 exceeds maximum",
		},
		{
			name:    "zero dimensions are valid",
			spec:    video_dto.TranscodeSpec{Width: 0, Height: 0},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := service.validateSpec(tt.spec)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateSpec_PixelsExceedMax(t *testing.T) {
	t.Parallel()

	service := &service{
		config: ServiceConfig{
			MaxVideoWidth:  4000,
			MaxVideoHeight: 4000,
			MaxVideoPixels: 1_000_000,
			MaxBitrate:     20_000_000,
			MaxFramerate:   60.0,
		},
	}

	err := service.validateSpec(video_dto.TranscodeSpec{Width: 1500, Height: 1500})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "total pixels")
}

func TestServiceCheck_WithTranscoders(t *testing.T) {
	t.Parallel()

	service := &service{
		transcoders: map[string]TranscoderPort{
			"ffmpeg": TranscoderPort(nil),
		},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	assert.Equal(t, "VideoService", status.Name)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Contains(t, status.Message, "1 transcoder(s)")
}

func TestServiceCheck_NoTranscoders(t *testing.T) {
	t.Parallel()

	service := &service{
		transcoders: map[string]TranscoderPort{},
		config:      DefaultServiceConfig(),
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	assert.Contains(t, status.Message, "No video transcoders registered")
}

func TestServiceName(t *testing.T) {
	t.Parallel()

	service := &service{}
	assert.Equal(t, "VideoService", service.Name())
}

type mockTranscoder struct {
	transcodeFunction        func(ctx context.Context, input io.Reader, spec video_dto.TranscodeSpec) (io.ReadCloser, error)
	extractCapsFunction      func(ctx context.Context, input io.Reader) (video_dto.VideoCapabilities, error)
	supportsCodecFunction    func(codec string) bool
	extractThumbnailFunction func(ctx context.Context, input io.Reader, spec video_dto.ThumbnailSpec) (io.ReadCloser, error)
}

func (m *mockTranscoder) Transcode(ctx context.Context, input io.Reader, spec video_dto.TranscodeSpec) (io.ReadCloser, error) {
	if m.transcodeFunction != nil {
		return m.transcodeFunction(ctx, input, spec)
	}
	return io.NopCloser(bytes.NewReader([]byte("transcoded"))), nil
}

func (m *mockTranscoder) ExtractCapabilities(ctx context.Context, input io.Reader) (video_dto.VideoCapabilities, error) {
	if m.extractCapsFunction != nil {
		return m.extractCapsFunction(ctx, input)
	}
	return video_dto.VideoCapabilities{Width: 1920, Height: 1080, VideoCodec: "h264", Duration: 30 * time.Second}, nil
}

func (m *mockTranscoder) SupportsCodec(codec string) bool {
	if m.supportsCodecFunction != nil {
		return m.supportsCodecFunction(codec)
	}
	return true
}

func (m *mockTranscoder) ExtractThumbnail(ctx context.Context, input io.Reader, spec video_dto.ThumbnailSpec) (io.ReadCloser, error) {
	if m.extractThumbnailFunction != nil {
		return m.extractThumbnailFunction(ctx, input, spec)
	}
	return io.NopCloser(bytes.NewReader([]byte("thumbnail"))), nil
}

type mockStreamingTranscoder struct {
	mockTranscoder
	transcodeHLSFn func(ctx context.Context, input io.Reader, spec video_dto.HLSSpec) (video_dto.HLSResult, error)
}

func (m *mockStreamingTranscoder) TranscodeHLS(ctx context.Context, input io.Reader, spec video_dto.HLSSpec) (video_dto.HLSResult, error) {
	if m.transcodeHLSFn != nil {
		return m.transcodeHLSFn(ctx, input, spec)
	}
	return video_dto.HLSResult{TotalSegments: 10, Variants: []video_dto.HLSVariant{{Bitrate: 1000000}}}, nil
}

func TestNewService_Valid(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	transcoders := map[string]TranscoderPort{"ffmpeg": tc}

	service, err := NewService(transcoders, "ffmpeg", DefaultServiceConfig())
	require.NoError(t, err)
	assert.NotNil(t, service)
}

func TestNewService_WithStreamingTranscoder(t *testing.T) {
	t.Parallel()

	stc := &mockStreamingTranscoder{}
	transcoders := map[string]TranscoderPort{"ffmpeg": stc}

	videoService, err := NewService(transcoders, "ffmpeg", DefaultServiceConfig())
	require.NoError(t, err)
	assert.NotNil(t, videoService)

	s, ok := videoService.(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	assert.NotNil(t, s.streamingTranscoder)
}

func TestNewService_WithoutStreamingTranscoder(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	transcoders := map[string]TranscoderPort{"ffmpeg": tc}

	videoService, err := NewService(transcoders, "ffmpeg", DefaultServiceConfig())
	require.NoError(t, err)

	s, ok := videoService.(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	assert.Nil(t, s.streamingTranscoder)
}

func TestNewService_NoTranscoders(t *testing.T) {
	t.Parallel()

	_, err := NewService(nil, "ffmpeg", DefaultServiceConfig())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validating video service inputs")
}

func TestNewService_EmptyDefaultProvider(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	transcoders := map[string]TranscoderPort{"ffmpeg": tc}

	_, err := NewService(transcoders, "", DefaultServiceConfig())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validating video service inputs")
}

func TestNewService_DefaultProviderNotRegistered(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	transcoders := map[string]TranscoderPort{"ffmpeg": tc}

	_, err := NewService(transcoders, "gstreamer", DefaultServiceConfig())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validating video service inputs")
}

func TestNewService_CustomConfig(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	transcoders := map[string]TranscoderPort{"ffmpeg": tc}
	config := ServiceConfig{
		MaxVideoWidth:  1920,
		MaxVideoHeight: 1080,
	}

	videoService, err := NewService(transcoders, "ffmpeg", config)
	require.NoError(t, err)

	s, ok := videoService.(*service)
	if !ok {
		t.Fatal("expected *service")
	}
	assert.Equal(t, 1920, s.config.MaxVideoWidth)
	assert.Equal(t, 1080, s.config.MaxVideoHeight)

	assert.Equal(t, int64(8_294_400), s.config.MaxVideoPixels)
	assert.Equal(t, int64(500*1024*1024), s.config.MaxFileSizeBytes)
}

func TestSelectTranscoder_Found(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
	}

	result, name, err := service.selectTranscoder()
	require.NoError(t, err)
	assert.Equal(t, tc, result)
	assert.Equal(t, "ffmpeg", name)
}

func TestSelectTranscoder_NotFound(t *testing.T) {
	t.Parallel()

	service := &service{
		transcoders:     map[string]TranscoderPort{},
		defaultProvider: "missing",
	}

	_, _, err := service.selectTranscoder()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestValidateSpec_AllFieldsValid(t *testing.T) {
	t.Parallel()

	service := &service{config: DefaultServiceConfig()}

	err := service.validateSpec(video_dto.TranscodeSpec{
		Width:     1920,
		Height:    1080,
		Bitrate:   5_000_000,
		Framerate: 30,
	})
	assert.NoError(t, err)
}

func TestValidateSpec_ZeroValues(t *testing.T) {
	t.Parallel()

	service := &service{config: DefaultServiceConfig()}

	err := service.validateSpec(video_dto.TranscodeSpec{})
	assert.NoError(t, err)
}

func TestValidateSpec_MaxWidthBoundary(t *testing.T) {
	t.Parallel()

	service := &service{config: DefaultServiceConfig()}

	err := service.validateSpec(video_dto.TranscodeSpec{Width: 3840})
	assert.NoError(t, err)

	err = service.validateSpec(video_dto.TranscodeSpec{Width: 3841})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "width 3841 exceeds maximum")
}

func TestValidateSpec_MaxHeightBoundary(t *testing.T) {
	t.Parallel()

	service := &service{config: DefaultServiceConfig()}

	err := service.validateSpec(video_dto.TranscodeSpec{Height: 2160})
	assert.NoError(t, err)

	err = service.validateSpec(video_dto.TranscodeSpec{Height: 2161})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "height 2161 exceeds maximum")
}

func TestValidateSpec_MaxBitrateBoundary(t *testing.T) {
	t.Parallel()

	service := &service{config: DefaultServiceConfig()}

	err := service.validateSpec(video_dto.TranscodeSpec{Bitrate: 20_000_000})
	assert.NoError(t, err)

	err = service.validateSpec(video_dto.TranscodeSpec{Bitrate: 20_000_001})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bitrate 20000001 exceeds maximum")
}

func TestValidateSpec_MaxFramerateBoundary(t *testing.T) {
	t.Parallel()

	service := &service{config: DefaultServiceConfig()}

	err := service.validateSpec(video_dto.TranscodeSpec{Framerate: 60.0})
	assert.NoError(t, err)

	err = service.validateSpec(video_dto.TranscodeSpec{Framerate: 60.1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "framerate 60.10 exceeds maximum")
}

func TestValidateSpec_PixelsExactlyAtMax(t *testing.T) {
	t.Parallel()

	service := &service{config: DefaultServiceConfig()}

	err := service.validateSpec(video_dto.TranscodeSpec{Width: 3840, Height: 2160})
	assert.NoError(t, err)
}

func TestValidateSpec_PixelsOneOverMax(t *testing.T) {
	t.Parallel()

	service := &service{
		config: ServiceConfig{
			MaxVideoWidth:  4000,
			MaxVideoHeight: 4000,
			MaxVideoPixels: 1_000_000,
			MaxBitrate:     20_000_000,
			MaxFramerate:   60.0,
		},
	}

	err := service.validateSpec(video_dto.TranscodeSpec{Width: 1001, Height: 1000})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "total pixels")
}

func TestValidateSpec_OneWidthZero(t *testing.T) {
	t.Parallel()

	service := &service{config: DefaultServiceConfig()}

	err := service.validateSpec(video_dto.TranscodeSpec{Width: 0, Height: 2160})
	assert.NoError(t, err)
}

func TestValidateSpec_OneHeightZero(t *testing.T) {
	t.Parallel()

	service := &service{config: DefaultServiceConfig()}

	err := service.validateSpec(video_dto.TranscodeSpec{Width: 3840, Height: 0})
	assert.NoError(t, err)
}

func TestApplyConfigDefaults_NegativeValues(t *testing.T) {
	t.Parallel()

	result := applyConfigDefaults(ServiceConfig{
		MaxVideoWidth:    -1,
		MaxVideoHeight:   -1,
		MaxVideoPixels:   -1,
		MaxFileSizeBytes: -1,
		TranscodeTimeout: -1,
		MaxBitrate:       -1,
		MaxFramerate:     -1,
	})

	defaults := DefaultServiceConfig()
	assert.Equal(t, defaults.MaxVideoWidth, result.MaxVideoWidth)
	assert.Equal(t, defaults.MaxVideoHeight, result.MaxVideoHeight)
	assert.Equal(t, defaults.MaxVideoPixels, result.MaxVideoPixels)
	assert.Equal(t, defaults.MaxFileSizeBytes, result.MaxFileSizeBytes)
	assert.Equal(t, defaults.TranscodeTimeout, result.TranscodeTimeout)
	assert.Equal(t, defaults.MaxBitrate, result.MaxBitrate)
	assert.Equal(t, defaults.MaxFramerate, result.MaxFramerate)
}

func TestApplyConfigDefaults_AllCustomValues(t *testing.T) {
	t.Parallel()

	custom := ServiceConfig{
		MaxVideoWidth:    1920,
		MaxVideoHeight:   1080,
		MaxVideoPixels:   2_073_600,
		MaxFileSizeBytes: 100 * 1024 * 1024,
		TranscodeTimeout: 2 * time.Minute,
		MaxBitrate:       10_000_000,
		MaxFramerate:     30.0,
	}

	result := applyConfigDefaults(custom)

	assert.Equal(t, 1920, result.MaxVideoWidth)
	assert.Equal(t, 1080, result.MaxVideoHeight)
	assert.Equal(t, int64(2_073_600), result.MaxVideoPixels)
	assert.Equal(t, int64(100*1024*1024), result.MaxFileSizeBytes)
	assert.Equal(t, 2*time.Minute, result.TranscodeTimeout)
	assert.Equal(t, 10_000_000, result.MaxBitrate)
	assert.Equal(t, 30.0, result.MaxFramerate)
}

func TestServiceCheck_HLSSupported(t *testing.T) {
	t.Parallel()

	stc := &mockStreamingTranscoder{}
	service := &service{
		transcoders:         map[string]TranscoderPort{"ffmpeg": stc},
		streamingTranscoder: stc,
		defaultProvider:     "ffmpeg",
		config:              DefaultServiceConfig(),
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Contains(t, status.Message, "HLS supported")
}

func TestServiceCheck_MultipleTranscoders(t *testing.T) {
	t.Parallel()

	service := &service{
		transcoders: map[string]TranscoderPort{
			"ffmpeg":    &mockTranscoder{},
			"gstreamer": &mockTranscoder{},
		},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Contains(t, status.Message, "2 transcoder(s)")
}

func TestServiceCheck_TimestampAndDuration(t *testing.T) {
	t.Parallel()

	service := &service{
		transcoders: map[string]TranscoderPort{
			"ffmpeg": &mockTranscoder{},
		},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	before := time.Now()
	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	assert.False(t, status.Timestamp.Before(before))
	assert.NotEmpty(t, status.Duration)
	assert.Nil(t, status.Dependencies)
}

func TestTranscodeHLS_NoStreamingTranscoder(t *testing.T) {
	t.Parallel()

	service := &service{
		transcoders:         map[string]TranscoderPort{"ffmpeg": &mockTranscoder{}},
		streamingTranscoder: nil,
		defaultProvider:     "ffmpeg",
		config:              DefaultServiceConfig(),
	}

	_, err := service.TranscodeHLS(context.Background(), bytes.NewReader([]byte("video")), video_dto.HLSSpec{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support HLS transcoding")
}

func TestTranscodeHLS_Success(t *testing.T) {
	t.Parallel()

	stc := &mockStreamingTranscoder{
		transcodeHLSFn: func(_ context.Context, _ io.Reader, _ video_dto.HLSSpec) (video_dto.HLSResult, error) {
			return video_dto.HLSResult{
				TotalSegments: 5,
				Variants:      []video_dto.HLSVariant{{Bitrate: 500000}},
			}, nil
		},
	}

	service := &service{
		transcoders:         map[string]TranscoderPort{"ffmpeg": stc},
		streamingTranscoder: stc,
		defaultProvider:     "ffmpeg",
		config:              DefaultServiceConfig(),
	}

	result, err := service.TranscodeHLS(context.Background(), bytes.NewReader([]byte("video")), video_dto.HLSSpec{})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalSegments)
	assert.Len(t, result.Variants, 1)
}

func TestTranscodeHLS_Error(t *testing.T) {
	t.Parallel()

	stc := &mockStreamingTranscoder{
		transcodeHLSFn: func(_ context.Context, _ io.Reader, _ video_dto.HLSSpec) (video_dto.HLSResult, error) {
			return video_dto.HLSResult{}, errors.New("hls failed")
		},
	}

	service := &service{
		transcoders:         map[string]TranscoderPort{"ffmpeg": stc},
		streamingTranscoder: stc,
		defaultProvider:     "ffmpeg",
		config:              DefaultServiceConfig(),
	}

	_, err := service.TranscodeHLS(context.Background(), bytes.NewReader([]byte("video")), video_dto.HLSSpec{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HLS transcoding failed")
}

func TestExtractCapabilities_Success(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{
		extractCapsFunction: func(_ context.Context, _ io.Reader) (video_dto.VideoCapabilities, error) {
			return video_dto.VideoCapabilities{
				Width:      1920,
				Height:     1080,
				VideoCodec: "h264",
				Duration:   60 * time.Second,
			}, nil
		},
	}

	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	caps, err := service.ExtractCapabilities(context.Background(), bytes.NewReader([]byte("video")))
	require.NoError(t, err)
	assert.Equal(t, 1920, caps.Width)
	assert.Equal(t, 1080, caps.Height)
	assert.Equal(t, "h264", caps.VideoCodec)
}

func TestExtractCapabilities_Error(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{
		extractCapsFunction: func(_ context.Context, _ io.Reader) (video_dto.VideoCapabilities, error) {
			return video_dto.VideoCapabilities{}, errors.New("cannot read")
		},
	}

	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	_, err := service.ExtractCapabilities(context.Background(), bytes.NewReader([]byte("bad")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract capabilities")
}

func TestExtractCapabilities_TranscoderNotFound(t *testing.T) {
	t.Parallel()

	service := &service{
		transcoders:     map[string]TranscoderPort{},
		defaultProvider: "missing",
		config:          DefaultServiceConfig(),
	}

	_, err := service.ExtractCapabilities(context.Background(), bytes.NewReader([]byte("video")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestValidateServiceInputs_EmptyMap(t *testing.T) {
	t.Parallel()

	err := validateServiceInputs(map[string]TranscoderPort{}, "ffmpeg")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one video transcoder")
}

func TestValidateServiceInputs_MultipleProvidersValid(t *testing.T) {
	t.Parallel()

	transcoders := map[string]TranscoderPort{
		"ffmpeg":    &mockTranscoder{},
		"gstreamer": &mockTranscoder{},
	}

	err := validateServiceInputs(transcoders, "gstreamer")
	assert.NoError(t, err)
}

func TestErrorSentinels(t *testing.T) {
	t.Parallel()

	assert.Error(t, ErrUnsupportedCodec)
	assert.Error(t, ErrUnsupportedFormat)
	assert.Error(t, ErrInvalidResolution)
	assert.Error(t, ErrInvalidBitrate)
	assert.Error(t, ErrInvalidFramerate)
	assert.Error(t, ErrDurationExceedsLimit)
	assert.Error(t, ErrFileSizeExceedsLimit)
	assert.Error(t, ErrResolutionExceedsLimit)
	assert.Error(t, ErrTranscodingFailed)
	assert.Error(t, ErrInvalidStream)
	assert.Error(t, ErrContextCancelled)
	assert.Error(t, ErrTimeout)
	assert.Error(t, ErrResourceExhausted)
	assert.Error(t, ErrInvalidHLSSpec)
	assert.Error(t, ErrSegmentationFailed)
}

func TestErrorMessages(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "unsupported video codec", ErrUnsupportedCodec.Error())
	assert.Equal(t, "unsupported video format", ErrUnsupportedFormat.Error())
	assert.Equal(t, "invalid video resolution", ErrInvalidResolution.Error())
	assert.Equal(t, "invalid video bitrate", ErrInvalidBitrate.Error())
	assert.Equal(t, "invalid video framerate", ErrInvalidFramerate.Error())
	assert.Equal(t, "video duration exceeds maximum limit", ErrDurationExceedsLimit.Error())
	assert.Equal(t, "video file size exceeds maximum limit", ErrFileSizeExceedsLimit.Error())
	assert.Equal(t, "video resolution exceeds maximum limit", ErrResolutionExceedsLimit.Error())
	assert.Equal(t, "video transcoding failed", ErrTranscodingFailed.Error())
	assert.Equal(t, "invalid or missing video stream", ErrInvalidStream.Error())
	assert.Equal(t, "video transcoding cancelled", ErrContextCancelled.Error())
	assert.Equal(t, "video transcoding timeout exceeded", ErrTimeout.Error())
	assert.Equal(t, "system resources exhausted", ErrResourceExhausted.Error())
	assert.Equal(t, "invalid HLS specification", ErrInvalidHLSSpec.Error())
	assert.Equal(t, "video segmentation failed", ErrSegmentationFailed.Error())
}

func TestDefaultConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3840, defaultMaxVideoWidth)
	assert.Equal(t, 2160, defaultMaxVideoHeight)
	assert.Equal(t, int64(8_294_400), int64(defaultMaxVideoPixels))
	assert.Equal(t, int64(500*1024*1024), int64(defaultMaxFileSizeBytes))
	assert.Equal(t, 5*time.Minute, defaultTranscodeTimeout)
	assert.Equal(t, 20_000_000, defaultMaxBitrate)
	assert.Equal(t, 60.0, defaultMaxFramerate)
}

func TestCodecProfiles(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "baseline", h264ProfileBaseline.Name)
	assert.Equal(t, "3.0", h264ProfileBaseline.Level)
	assert.Contains(t, h264ProfileBaseline.Description, "Baseline")

	assert.Equal(t, "main", h264ProfileMain.Name)
	assert.Equal(t, "4.0", h264ProfileMain.Level)

	assert.Equal(t, "high", h264ProfileHigh.Name)
	assert.Equal(t, "4.2", h264ProfileHigh.Level)

	assert.Equal(t, "main", h265ProfileMain.Name)
	assert.Equal(t, "4.0", h265ProfileMain.Level)

	assert.Equal(t, "main10", h265ProfileMain10.Name)
	assert.Equal(t, "5.0", h265ProfileMain10.Level)
}

func TestQualityPresets(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ultrafast", presetUltrafast.Name)
	assert.Equal(t, 100, presetUltrafast.EncodingSpeed)
	assert.Equal(t, 10, presetUltrafast.QualityScore)

	assert.Equal(t, "fast", presetFast.Name)
	assert.Equal(t, 75, presetFast.EncodingSpeed)
	assert.Equal(t, 40, presetFast.QualityScore)

	assert.Equal(t, "medium", presetMedium.Name)
	assert.Equal(t, 50, presetMedium.EncodingSpeed)
	assert.Equal(t, 65, presetMedium.QualityScore)

	assert.Equal(t, "slow", presetSlow.Name)
	assert.Equal(t, 25, presetSlow.EncodingSpeed)
	assert.Equal(t, 85, presetSlow.QualityScore)

	assert.Equal(t, "veryslow", presetVeryslow.Name)
	assert.Equal(t, 10, presetVeryslow.EncodingSpeed)
	assert.Equal(t, 95, presetVeryslow.QualityScore)
}

func TestQualityPresetOrdering(t *testing.T) {
	t.Parallel()

	assert.Greater(t, presetUltrafast.EncodingSpeed, presetFast.EncodingSpeed)
	assert.Greater(t, presetFast.EncodingSpeed, presetMedium.EncodingSpeed)
	assert.Greater(t, presetMedium.EncodingSpeed, presetSlow.EncodingSpeed)
	assert.Greater(t, presetSlow.EncodingSpeed, presetVeryslow.EncodingSpeed)

	assert.Less(t, presetUltrafast.QualityScore, presetFast.QualityScore)
	assert.Less(t, presetFast.QualityScore, presetMedium.QualityScore)
	assert.Less(t, presetMedium.QualityScore, presetSlow.QualityScore)
	assert.Less(t, presetSlow.QualityScore, presetVeryslow.QualityScore)
}

func TestMetricAttributeConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "provider", metricAttributeProvider)
	assert.Equal(t, "codec", metricAttributeCodec)
}

func TestServiceInterfaceCompliance(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	transcoders := map[string]TranscoderPort{"ffmpeg": tc}

	service, err := NewService(transcoders, "ffmpeg", DefaultServiceConfig())
	require.NoError(t, err)

	var _ Service = service
}

func TestTranscode_InvalidParams(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	_, err := service.Transcode(context.Background(), bytes.NewReader([]byte("video")), map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transcode parameters")
}

func TestTranscode_SpecExceedsLimits(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	params := map[string]string{
		"codec":  "h264",
		"format": "mp4",
		"width":  "5000",
	}
	_, err := service.Transcode(context.Background(), bytes.NewReader([]byte("video")), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcode spec validation failed")
}

func TestTranscode_TranscoderNotFound(t *testing.T) {
	t.Parallel()

	service := &service{
		transcoders:     map[string]TranscoderPort{},
		defaultProvider: "missing",
		config:          DefaultServiceConfig(),
	}

	params := map[string]string{
		"codec":  "h264",
		"format": "mp4",
	}
	_, err := service.Transcode(context.Background(), bytes.NewReader([]byte("video")), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selecting transcoder")
}

func TestTranscode_Success(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{
		transcodeFunction: func(_ context.Context, _ io.Reader, _ video_dto.TranscodeSpec) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("output"))), nil
		},
	}
	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	params := map[string]string{
		"codec":  "h264",
		"format": "mp4",
	}
	result, err := service.Transcode(context.Background(), bytes.NewReader([]byte("video")), params)
	require.NoError(t, err)
	require.NotNil(t, result)
	defer func() { _ = result.Close() }()

	data, err := io.ReadAll(result)
	require.NoError(t, err)
	assert.Equal(t, "output", string(data))
}

func TestTranscode_TranscoderFailure(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{
		transcodeFunction: func(_ context.Context, _ io.Reader, _ video_dto.TranscodeSpec) (io.ReadCloser, error) {
			return nil, errors.New("encoding failed")
		},
	}
	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	params := map[string]string{
		"codec":  "h264",
		"format": "mp4",
	}
	_, err := service.Transcode(context.Background(), bytes.NewReader([]byte("video")), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcoding failed")
}

func TestExtractThumbnail_InvalidSpec(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{}
	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	spec := video_dto.ThumbnailSpec{
		Width: -1,
	}

	_, err := service.ExtractThumbnail(context.Background(), bytes.NewReader([]byte("video")), spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid thumbnail spec")
}

func TestExtractThumbnail_TranscoderNotFound(t *testing.T) {
	t.Parallel()

	service := &service{
		transcoders:     map[string]TranscoderPort{},
		defaultProvider: "missing",
		config:          DefaultServiceConfig(),
	}

	spec := video_dto.ThumbnailSpec{Format: "jpeg", Quality: 85}

	_, err := service.ExtractThumbnail(context.Background(), bytes.NewReader([]byte("video")), spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selecting transcoder for thumbnail extraction")
}

func TestExtractThumbnail_Success(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{
		extractThumbnailFunction: func(_ context.Context, _ io.Reader, _ video_dto.ThumbnailSpec) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("thumbnail_data"))), nil
		},
	}
	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	spec := video_dto.ThumbnailSpec{Format: "jpeg", Quality: 85}
	result, err := service.ExtractThumbnail(context.Background(), bytes.NewReader([]byte("video")), spec)
	require.NoError(t, err)
	require.NotNil(t, result)
	defer func() { _ = result.Close() }()

	data, err := io.ReadAll(result)
	require.NoError(t, err)
	assert.Equal(t, "thumbnail_data", string(data))
}

func TestExtractThumbnail_Error(t *testing.T) {
	t.Parallel()

	tc := &mockTranscoder{
		extractThumbnailFunction: func(_ context.Context, _ io.Reader, _ video_dto.ThumbnailSpec) (io.ReadCloser, error) {
			return nil, errors.New("decode failed")
		},
	}
	service := &service{
		transcoders:     map[string]TranscoderPort{"ffmpeg": tc},
		defaultProvider: "ffmpeg",
		config:          DefaultServiceConfig(),
	}

	spec := video_dto.ThumbnailSpec{Format: "png"}
	_, err := service.ExtractThumbnail(context.Background(), bytes.NewReader([]byte("video")), spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract thumbnail")
}

func TestRecordTranscodeMetrics(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spec := video_dto.TranscodeSpec{Codec: "h264", Format: "mp4"}

	assert.NotPanics(t, func() {
		recordTranscodeMetrics(ctx, "ffmpeg", spec, true)
	})
	assert.NotPanics(t, func() {
		recordTranscodeMetrics(ctx, "ffmpeg", spec, false)
	})
}

func TestRecordTranscodeDuration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spec := video_dto.TranscodeSpec{Codec: "h264", Format: "mp4"}

	assert.NotPanics(t, func() {
		recordTranscodeDuration(ctx, "ffmpeg", spec, 1500)
	})
}

func TestRecordTranscodeError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	assert.NotPanics(t, func() {
		recordTranscodeError(ctx, "ffmpeg", "test_error")
	})
}
