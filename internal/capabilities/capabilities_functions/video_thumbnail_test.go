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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/piko/internal/video/video_dto"
)

type mockVideoService struct {
	transcodeFunction        func(ctx context.Context, input io.Reader, params map[string]string) (io.ReadCloser, error)
	extractThumbnailFunction func(ctx context.Context, input io.Reader, spec video_dto.ThumbnailSpec) (io.ReadCloser, error)
}

func (m *mockVideoService) Transcode(ctx context.Context, input io.Reader, params map[string]string) (io.ReadCloser, error) {
	if m.transcodeFunction != nil {
		return m.transcodeFunction(ctx, input, params)
	}
	return nil, errors.New("no mock configured")
}

func (m *mockVideoService) ExtractCapabilities(_ context.Context, _ io.Reader) (video_dto.VideoCapabilities, error) {
	return video_dto.VideoCapabilities{}, errors.New("not implemented")
}

func (m *mockVideoService) TranscodeHLS(_ context.Context, _ io.Reader, _ video_dto.HLSSpec) (video_dto.HLSResult, error) {
	return video_dto.HLSResult{}, errors.New("not implemented")
}

func (m *mockVideoService) ExtractThumbnail(ctx context.Context, input io.Reader, spec video_dto.ThumbnailSpec) (io.ReadCloser, error) {
	if m.extractThumbnailFunction != nil {
		return m.extractThumbnailFunction(ctx, input, spec)
	}
	return nil, errors.New("no mock configured")
}

var _ video_domain.Service = (*mockVideoService)(nil)

func TestGetFirstNonEmpty(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		params   capabilities_domain.CapabilityParams
		name     string
		expected string
		keys     []string
	}{
		{
			name:     "should return empty when no keys provided",
			params:   capabilities_domain.CapabilityParams{"a": "1"},
			keys:     []string{},
			expected: "",
		},
		{
			name:     "should return empty when no matching keys",
			params:   capabilities_domain.CapabilityParams{"a": "1"},
			keys:     []string{"b", "c"},
			expected: "",
		},
		{
			name:     "should return empty when nil params",
			params:   nil,
			keys:     []string{"a"},
			expected: "",
		},
		{
			name:     "should return first non-empty value",
			params:   capabilities_domain.CapabilityParams{"a": "1", "b": "2"},
			keys:     []string{"a", "b"},
			expected: "1",
		},
		{
			name:     "should skip empty values",
			params:   capabilities_domain.CapabilityParams{"a": "", "b": "2"},
			keys:     []string{"a", "b"},
			expected: "2",
		},
		{
			name:     "should return empty when all values are empty",
			params:   capabilities_domain.CapabilityParams{"a": "", "b": ""},
			keys:     []string{"a", "b"},
			expected: "",
		},
		{
			name:     "should return first matching key",
			params:   capabilities_domain.CapabilityParams{"timestamp": "5s", "time": "10s"},
			keys:     []string{"timestamp", "time"},
			expected: "5s",
		},
		{
			name:     "should fall back to second key if first missing",
			params:   capabilities_domain.CapabilityParams{"time": "10s"},
			keys:     []string{"timestamp", "time"},
			expected: "10s",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getFirstNonEmpty(tc.params, tc.keys...)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseThumbnailTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("should return nil error when no timestamp provided", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailTimestamp(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), spec.Timestamp)
	})

	t.Run("should parse duration format timestamp", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"timestamp": "5s"}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailTimestamp(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, 5*time.Second, spec.Timestamp)
	})

	t.Run("should use time alias", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"time": "10s"}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailTimestamp(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, 10*time.Second, spec.Timestamp)
	})

	t.Run("should prefer timestamp over time", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"timestamp": "5s", "time": "10s"}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailTimestamp(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, 5*time.Second, spec.Timestamp)
	})

	t.Run("should return error for invalid timestamp", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"timestamp": "notavalidtime"}
		err := parseThumbnailTimestamp(params, new(video_dto.NewThumbnailSpec()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing timestamp")
	})
}

func TestParseThumbnailDimensions(t *testing.T) {
	t.Parallel()

	t.Run("should leave defaults when no dimensions provided", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailDimensions(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, 0, spec.Width)
		assert.Equal(t, 0, spec.Height)
	})

	t.Run("should parse width", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"width": "640"}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailDimensions(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, 640, spec.Width)
	})

	t.Run("should parse height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"height": "480"}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailDimensions(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, 480, spec.Height)
	})

	t.Run("should parse both width and height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"width": "640", "height": "480"}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailDimensions(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, 640, spec.Width)
		assert.Equal(t, 480, spec.Height)
	})

	t.Run("should return error for invalid width", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"width": "abc"}
		err := parseThumbnailDimensions(params, new(video_dto.NewThumbnailSpec()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing width")
	})

	t.Run("should return error for invalid height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"height": "xyz"}
		err := parseThumbnailDimensions(params, new(video_dto.NewThumbnailSpec()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing height")
	})

	t.Run("should skip empty width", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"width": ""}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailDimensions(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, 0, spec.Width)
	})

	t.Run("should skip empty height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"height": ""}
		spec := video_dto.NewThumbnailSpec()
		err := parseThumbnailDimensions(params, &spec)
		require.NoError(t, err)
		assert.Equal(t, 0, spec.Height)
	})
}

func TestParseThumbnailOutputSettings(t *testing.T) {
	t.Parallel()

	t.Run("should leave defaults when no settings provided", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{}
		spec := video_dto.NewThumbnailSpec()
		defaultFormat := spec.Format
		defaultQuality := spec.Quality
		parseThumbnailOutputSettings(params, &spec)
		assert.Equal(t, defaultFormat, spec.Format)
		assert.Equal(t, defaultQuality, spec.Quality)
	})

	t.Run("should parse format", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"format": "png"}
		spec := video_dto.NewThumbnailSpec()
		parseThumbnailOutputSettings(params, &spec)
		assert.Equal(t, "png", spec.Format)
	})

	t.Run("should parse quality", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"quality": "90"}
		spec := video_dto.NewThumbnailSpec()
		parseThumbnailOutputSettings(params, &spec)
		assert.Equal(t, 90, spec.Quality)
	})

	t.Run("should ignore invalid quality", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"quality": "abc"}
		spec := video_dto.NewThumbnailSpec()
		defaultQuality := spec.Quality
		parseThumbnailOutputSettings(params, &spec)
		assert.Equal(t, defaultQuality, spec.Quality)
	})

	t.Run("should skip empty format", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"format": ""}
		spec := video_dto.NewThumbnailSpec()
		defaultFormat := spec.Format
		parseThumbnailOutputSettings(params, &spec)
		assert.Equal(t, defaultFormat, spec.Format)
	})

	t.Run("should skip empty quality", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"quality": ""}
		spec := video_dto.NewThumbnailSpec()
		defaultQuality := spec.Quality
		parseThumbnailOutputSettings(params, &spec)
		assert.Equal(t, defaultQuality, spec.Quality)
	})

	t.Run("should parse both format and quality", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"format": "webp", "quality": "75"}
		spec := video_dto.NewThumbnailSpec()
		parseThumbnailOutputSettings(params, &spec)
		assert.Equal(t, "webp", spec.Format)
		assert.Equal(t, 75, spec.Quality)
	})
}

func TestBuildThumbnailSpec(t *testing.T) {
	t.Parallel()

	t.Run("should return default spec with no params", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{}
		spec, err := buildThumbnailSpec(params)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), spec.Timestamp)
		assert.Equal(t, 0, spec.Width)
		assert.Equal(t, 0, spec.Height)
	})

	t.Run("should parse all fields", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{
			"timestamp": "5s",
			"width":     "640",
			"height":    "480",
			"format":    "png",
			"quality":   "90",
		}
		spec, err := buildThumbnailSpec(params)
		require.NoError(t, err)
		assert.Equal(t, 5*time.Second, spec.Timestamp)
		assert.Equal(t, 640, spec.Width)
		assert.Equal(t, 480, spec.Height)
		assert.Equal(t, "png", spec.Format)
		assert.Equal(t, 90, spec.Quality)
	})

	t.Run("should return error for invalid timestamp", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"timestamp": "invalid"}
		_, err := buildThumbnailSpec(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing timestamp")
	})

	t.Run("should return error for invalid width", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"width": "abc"}
		_, err := buildThumbnailSpec(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing width")
	})

	t.Run("should return error for invalid height", func(t *testing.T) {
		t.Parallel()
		params := capabilities_domain.CapabilityParams{"height": "abc"}
		_, err := buildThumbnailSpec(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing height")
	})
}

func TestVideoThumbnailCapability(t *testing.T) {
	t.Parallel()

	t.Run("should extract thumbnail successfully", func(t *testing.T) {
		t.Parallel()
		service := &mockVideoService{
			extractThumbnailFunction: func(_ context.Context, _ io.Reader, _ video_dto.ThumbnailSpec) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("thumbnail data")), nil
			},
		}

		capabilityFunction := VideoThumbnail(service)
		params := capabilities_domain.CapabilityParams{
			"timestamp": "5s",
			"width":     "320",
			"height":    "240",
			"format":    "jpeg",
			"quality":   "85",
		}
		result, err := capabilityFunction(context.Background(), strings.NewReader("video data"), params)
		require.NoError(t, err)
		require.NotNil(t, result)

		output, err := io.ReadAll(result)
		require.NoError(t, err)
		assert.Equal(t, "thumbnail data", string(output))
	})

	t.Run("should extract thumbnail with default params", func(t *testing.T) {
		t.Parallel()
		service := &mockVideoService{
			extractThumbnailFunction: func(_ context.Context, _ io.Reader, _ video_dto.ThumbnailSpec) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("thumb")), nil
			},
		}

		capabilityFunction := VideoThumbnail(service)
		result, err := capabilityFunction(context.Background(), strings.NewReader("video"), nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		output, err := io.ReadAll(result)
		require.NoError(t, err)
		assert.Equal(t, "thumb", string(output))
	})

	t.Run("should return error for invalid params", func(t *testing.T) {
		t.Parallel()
		service := &mockVideoService{}

		capabilityFunction := VideoThumbnail(service)
		params := capabilities_domain.CapabilityParams{"timestamp": "invalid_time"}
		_, err := capabilityFunction(context.Background(), strings.NewReader("video"), params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid thumbnail parameters")
	})

	t.Run("should return error when video service fails", func(t *testing.T) {
		t.Parallel()
		service := &mockVideoService{
			extractThumbnailFunction: func(_ context.Context, _ io.Reader, _ video_dto.ThumbnailSpec) (io.ReadCloser, error) {
				return nil, errors.New("extraction failed")
			},
		}

		capabilityFunction := VideoThumbnail(service)
		_, err := capabilityFunction(context.Background(), strings.NewReader("video"), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to extract thumbnail")
	})
}
