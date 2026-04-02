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
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
)

func TestParseResolution(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		resolution     string
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "should parse standard 1080p",
			resolution:     "1920x1080",
			expectedWidth:  1920,
			expectedHeight: 1080,
		},
		{
			name:           "should parse 720p",
			resolution:     "1280x720",
			expectedWidth:  1280,
			expectedHeight: 720,
		},
		{
			name:           "should parse 4k",
			resolution:     "3840x2160",
			expectedWidth:  3840,
			expectedHeight: 2160,
		},
		{
			name:           "should handle uppercase X",
			resolution:     "1920X1080",
			expectedWidth:  1920,
			expectedHeight: 1080,
		},
		{
			name:           "should handle whitespace around resolution",
			resolution:     " 1920x1080 ",
			expectedWidth:  1920,
			expectedHeight: 1080,
		},
		{
			name:           "should handle whitespace around parts",
			resolution:     " 1920 x 1080 ",
			expectedWidth:  1920,
			expectedHeight: 1080,
		},
		{
			name:           "should return 0,0 for empty string",
			resolution:     "",
			expectedWidth:  0,
			expectedHeight: 0,
		},
		{
			name:           "should return 0,0 for invalid format",
			resolution:     "1920",
			expectedWidth:  0,
			expectedHeight: 0,
		},
		{
			name:           "should return 0,0 for triple parts",
			resolution:     "1920x1080x30",
			expectedWidth:  0,
			expectedHeight: 0,
		},
		{
			name:           "should return 0,0 for non-numeric width",
			resolution:     "abcx1080",
			expectedWidth:  0,
			expectedHeight: 0,
		},
		{
			name:           "should return 0,0 for non-numeric height",
			resolution:     "1920xabc",
			expectedWidth:  0,
			expectedHeight: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			width, height := parseResolution(tc.resolution)
			assert.Equal(t, tc.expectedWidth, width)
			assert.Equal(t, tc.expectedHeight, height)
		})
	}
}

func TestCopyParamIfPresent(t *testing.T) {
	t.Parallel()

	t.Run("should copy existing non-empty param", func(t *testing.T) {
		t.Parallel()
		src := capabilities_domain.CapabilityParams{"key": "value"}
		dst := make(map[string]string)
		copyParamIfPresent(src, dst, "key")
		assert.Equal(t, "value", dst["key"])
	})

	t.Run("should not copy when key missing", func(t *testing.T) {
		t.Parallel()
		src := capabilities_domain.CapabilityParams{}
		dst := make(map[string]string)
		copyParamIfPresent(src, dst, "key")
		_, ok := dst["key"]
		assert.False(t, ok)
	})

	t.Run("should not copy empty value", func(t *testing.T) {
		t.Parallel()
		src := capabilities_domain.CapabilityParams{"key": ""}
		dst := make(map[string]string)
		copyParamIfPresent(src, dst, "key")
		_, ok := dst["key"]
		assert.False(t, ok)
	})

	t.Run("should not overwrite existing dest value when source empty", func(t *testing.T) {
		t.Parallel()
		src := capabilities_domain.CapabilityParams{"key": ""}
		dst := map[string]string{"key": "existing"}
		copyParamIfPresent(src, dst, "key")
		assert.Equal(t, "existing", dst["key"])
	})
}

func TestBuildVideoTranscodeParams(t *testing.T) {
	t.Parallel()

	t.Run("should set defaults when no params provided", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, defaultVideoCodec, result[paramCodec])
		assert.Equal(t, defaultVideoPreset, result["preset"])
	})

	t.Run("should parse resolution into width and height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"resolution": "1920x1080"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "1920", result["width"])
		assert.Equal(t, "1080", result["height"])
	})

	t.Run("should not set dimensions for invalid resolution", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"resolution": "invalid"}
		result := buildVideoTranscodeParams(params)
		_, hasWidth := result["width"]
		_, hasHeight := result["height"]
		assert.False(t, hasWidth)
		assert.False(t, hasHeight)
	})

	t.Run("should copy direct width and height over resolution", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"resolution": "1920x1080",
			"width":      "1280",
			"height":     "720",
		}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "1280", result["width"])
		assert.Equal(t, "720", result["height"])
	})

	t.Run("should copy codec param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{paramCodec: "h265"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "h265", result[paramCodec])
	})

	t.Run("should copy preset param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"preset": "fast"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "fast", result["preset"])
	})

	t.Run("should copy bitrate param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"bitrate": "5000k"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "5000k", result["bitrate"])
	})

	t.Run("should copy crf param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"crf": "23"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "23", result["crf"])
	})

	t.Run("should copy profile param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"profile": "main"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "main", result["profile"])
	})

	t.Run("should copy level param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"level": "4.0"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "4.0", result["level"])
	})

	t.Run("should copy format param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"format": "webm"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "webm", result["format"])
	})

	t.Run("should copy audio_codec param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"audio_codec": "opus"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "opus", result["audio_codec"])
	})

	t.Run("should copy audio_bitrate param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"audio_bitrate": "128000"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "128000", result["audio_bitrate"])
	})

	t.Run("should copy segment_duration param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"segment_duration": "6"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "6", result["segment_duration"])
	})

	t.Run("should copy framerate param", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"framerate": "30"}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "30", result["framerate"])
	})

	t.Run("should not copy unknown params", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"unknown_param": "value"}
		result := buildVideoTranscodeParams(params)
		_, ok := result["unknown_param"]
		assert.False(t, ok)
	})

	t.Run("should not copy empty params", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"bitrate": "",
			"crf":     "",
		}
		result := buildVideoTranscodeParams(params)
		_, hasBitrate := result["bitrate"]
		_, hasCRF := result["crf"]
		assert.False(t, hasBitrate)
		assert.False(t, hasCRF)
	})

	t.Run("should set all provided params", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"resolution":       "1280x720",
			"bitrate":          "2500k",
			"crf":              "28",
			"preset":           "slow",
			paramCodec:         "vp9",
			"profile":          "high",
			"level":            "3.0",
			"format":           "webm",
			"audio_codec":      "opus",
			"audio_bitrate":    "96000",
			"segment_duration": "4",
			"framerate":        "24",
		}
		result := buildVideoTranscodeParams(params)
		assert.Equal(t, "1280", result["width"])
		assert.Equal(t, "720", result["height"])
		assert.Equal(t, "2500k", result["bitrate"])
		assert.Equal(t, "28", result["crf"])
		assert.Equal(t, "slow", result["preset"])
		assert.Equal(t, "vp9", result[paramCodec])
		assert.Equal(t, "high", result["profile"])
		assert.Equal(t, "3.0", result["level"])
		assert.Equal(t, "webm", result["format"])
		assert.Equal(t, "opus", result["audio_codec"])
		assert.Equal(t, "96000", result["audio_bitrate"])
		assert.Equal(t, "4", result["segment_duration"])
		assert.Equal(t, "24", result["framerate"])
	})

	t.Run("should handle empty resolution", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"resolution": ""}
		result := buildVideoTranscodeParams(params)
		_, hasWidth := result["width"]
		_, hasHeight := result["height"]
		assert.False(t, hasWidth)
		assert.False(t, hasHeight)
	})
}

func TestVideoTranscodeCapability(t *testing.T) {
	t.Parallel()

	t.Run("should transcode successfully", func(t *testing.T) {
		t.Parallel()
		service := &mockVideoService{
			transcodeFunction: func(_ context.Context, _ io.Reader, _ map[string]string) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("transcoded data")), nil
			},
		}

		capabilityFunction := VideoTranscode(service)
		params := capabilities_domain.CapabilityParams{
			"resolution": "1920x1080",
			paramCodec:   "h265",
			"preset":     "fast",
		}
		result, err := capabilityFunction(context.Background(), strings.NewReader("video"), params)
		require.NoError(t, err)
		require.NotNil(t, result)

		output, err := io.ReadAll(result)
		require.NoError(t, err)
		assert.Equal(t, "transcoded data", string(output))
	})

	t.Run("should transcode with default params", func(t *testing.T) {
		t.Parallel()
		var capturedParams map[string]string
		service := &mockVideoService{
			transcodeFunction: func(_ context.Context, _ io.Reader, params map[string]string) (io.ReadCloser, error) {
				capturedParams = params
				return io.NopCloser(strings.NewReader("output")), nil
			},
		}

		capabilityFunction := VideoTranscode(service)
		result, err := capabilityFunction(context.Background(), strings.NewReader("video"), nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		_, _ = io.ReadAll(result)
		assert.Equal(t, defaultVideoCodec, capturedParams[paramCodec])
		assert.Equal(t, defaultVideoPreset, capturedParams["preset"])
	})

	t.Run("should return error when transcode fails", func(t *testing.T) {
		t.Parallel()
		service := &mockVideoService{
			transcodeFunction: func(_ context.Context, _ io.Reader, _ map[string]string) (io.ReadCloser, error) {
				return nil, errors.New("transcode failed")
			},
		}

		capabilityFunction := VideoTranscode(service)
		_, err := capabilityFunction(context.Background(), strings.NewReader("video"), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to transcode video")
	})

	t.Run("should pass all params to service", func(t *testing.T) {
		t.Parallel()
		var capturedParams map[string]string
		service := &mockVideoService{
			transcodeFunction: func(_ context.Context, _ io.Reader, params map[string]string) (io.ReadCloser, error) {
				capturedParams = params
				return io.NopCloser(strings.NewReader("ok")), nil
			},
		}

		capabilityFunction := VideoTranscode(service)
		params := capabilities_domain.CapabilityParams{
			"resolution":  "1280x720",
			"bitrate":     "2500k",
			paramCodec:    "vp9",
			"preset":      "slow",
			"audio_codec": "opus",
		}
		result, err := capabilityFunction(context.Background(), strings.NewReader("video"), params)
		require.NoError(t, err)
		_, _ = io.ReadAll(result)

		assert.Equal(t, "1280", capturedParams["width"])
		assert.Equal(t, "720", capturedParams["height"])
		assert.Equal(t, "2500k", capturedParams["bitrate"])
		assert.Equal(t, "vp9", capturedParams[paramCodec])
		assert.Equal(t, "slow", capturedParams["preset"])
		assert.Equal(t, "opus", capturedParams["audio_codec"])
	})
}
