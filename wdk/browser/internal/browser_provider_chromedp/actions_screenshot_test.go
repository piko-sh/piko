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

package browser_provider_chromedp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultScreenshotOptions(t *testing.T) {
	opts := DefaultScreenshotOptions()

	assert.Equal(t, ScreenshotFormatPNG, opts.Format)
	assert.Equal(t, ScreenshotQualityMax, opts.Quality)
	assert.True(t, opts.FromSurface)
	assert.False(t, opts.CaptureBeyondViewport)
	assert.False(t, opts.OptimiseForSpeed)
	assert.Equal(t, defaultScreenshotTimeout, opts.Timeout)
}

func TestNewCaptureSettings(t *testing.T) {
	testCases := []struct {
		name     string
		opts     []CaptureOption
		expected time.Duration
	}{
		{name: "no options uses the default timeout", opts: nil, expected: defaultScreenshotTimeout},
		{name: "positive timeout is applied", opts: []CaptureOption{WithCaptureTimeout(5 * time.Second)}, expected: 5 * time.Second},
		{name: "zero timeout keeps the default", opts: []CaptureOption{WithCaptureTimeout(0)}, expected: defaultScreenshotTimeout},
		{name: "negative timeout keeps the default", opts: []CaptureOption{WithCaptureTimeout(-time.Second)}, expected: defaultScreenshotTimeout},
		{
			name:     "last positive timeout wins",
			opts:     []CaptureOption{WithCaptureTimeout(time.Second), WithCaptureTimeout(2 * time.Second)},
			expected: 2 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, newCaptureSettings(tc.opts).timeout)
		})
	}
}

func TestResolveCaptureTimeout(t *testing.T) {
	testCases := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{name: "positive timeout is kept", timeout: 3 * time.Second, expected: 3 * time.Second},
		{name: "zero uses the default", timeout: 0, expected: defaultScreenshotTimeout},
		{name: "negative uses the default", timeout: -time.Second, expected: defaultScreenshotTimeout},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, resolveCaptureTimeout(tc.timeout))
		})
	}
}

func TestRegionFromBounds(t *testing.T) {
	testCases := []struct {
		bounds         map[string]any
		name           string
		errorContains  string
		expectedX      float64
		expectedY      float64
		expectedWidth  float64
		expectedHeight float64
	}{
		{
			name:           "bounds inside the viewport are used as given",
			bounds:         map[string]any{"x": 10.0, "y": 20.0, "width": 120.0, "height": 80.0},
			expectedX:      10,
			expectedY:      20,
			expectedWidth:  120,
			expectedHeight: 80,
		},
		{
			name:           "region above and left of the viewport is trimmed",
			bounds:         map[string]any{"x": -5.0, "y": -10.0, "width": 120.0, "height": 80.0},
			expectedX:      0,
			expectedY:      0,
			expectedWidth:  115,
			expectedHeight: 70,
		},
		{name: "missing x is rejected", bounds: map[string]any{"y": 0.0, "width": 1.0, "height": 1.0}, errorContains: "invalid x"},
		{name: "missing y is rejected", bounds: map[string]any{"x": 0.0, "width": 1.0, "height": 1.0}, errorContains: "invalid y"},
		{name: "non-numeric width is rejected", bounds: map[string]any{"x": 0.0, "y": 0.0, "width": "wide", "height": 1.0}, errorContains: "invalid width"},
		{name: "missing height is rejected", bounds: map[string]any{"x": 0.0, "y": 0.0, "width": 1.0}, errorContains: "invalid height"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capture, err := regionFromBounds(tc.bounds, "#target")
			if tc.errorContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Contains(t, err.Error(), "#target")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, capture.Clip)
			assert.InDelta(t, tc.expectedX, capture.Clip.X, 0.001)
			assert.InDelta(t, tc.expectedY, capture.Clip.Y, 0.001)
			assert.InDelta(t, tc.expectedWidth, capture.Clip.Width, 0.001)
			assert.InDelta(t, tc.expectedHeight, capture.Clip.Height, 0.001)
			assert.InDelta(t, 1.0, capture.Clip.Scale, 0.001)
		})
	}
}

func TestWithCaptureCause(t *testing.T) {
	captureErr := errors.New("capture failed")

	t.Run("returns the error unchanged while the context is live", func(t *testing.T) {
		err := withCaptureCause(context.Background(), captureErr)
		assert.Same(t, captureErr, err)
	})

	t.Run("adds the cancellation cause once the context has ended", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(errScreenshotTimedOut)

		err := withCaptureCause(ctx, captureErr)
		require.ErrorIs(t, err, captureErr)
		require.ErrorIs(t, err, errScreenshotTimedOut)
	})

	t.Run("does not repeat a cause the error already carries", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(errScreenshotTimedOut)

		err := withCaptureCause(ctx, errScreenshotTimedOut)
		assert.Same(t, errScreenshotTimedOut, err)
	})
}

func TestCompareScreenshots_Unit(t *testing.T) {
	testCases := []struct {
		name     string
		a        []byte
		b        []byte
		expected float64
	}{
		{
			name:     "identical bytes",
			a:        []byte{0x00, 0x01, 0x02, 0x03},
			b:        []byte{0x00, 0x01, 0x02, 0x03},
			expected: 0.0,
		},
		{
			name:     "different lengths",
			a:        []byte{0x00, 0x01},
			b:        []byte{0x00, 0x01, 0x02},
			expected: 1.0,
		},
		{
			name:     "both empty",
			a:        []byte{},
			b:        []byte{},
			expected: 0.0,
		},
		{
			name:     "one of four bytes differs",
			a:        []byte{0x00, 0x01, 0x02, 0x03},
			b:        []byte{0x00, 0x01, 0xFF, 0x03},
			expected: 0.25,
		},
		{
			name:     "completely different",
			a:        []byte{0x00, 0x00, 0x00, 0x00},
			b:        []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expected: 1.0,
		},
		{
			name:     "half different",
			a:        []byte{0x00, 0x01, 0x02, 0x03},
			b:        []byte{0xFF, 0xFF, 0x02, 0x03},
			expected: 0.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := CompareScreenshots(tc.a, tc.b)
			require.NoError(t, err)
			assert.InDelta(t, tc.expected, result, 0.001)
		})
	}
}

func TestCompareScreenshots_Unit_ErrorIsAlwaysNil(t *testing.T) {
	_, err := CompareScreenshots([]byte{0x00}, []byte{0xFF, 0xFF})
	require.NoError(t, err)
}
