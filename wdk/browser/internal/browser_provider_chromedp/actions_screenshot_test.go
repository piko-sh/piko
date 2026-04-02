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
	"math"
	"testing"
)

func TestDefaultScreenshotOptions(t *testing.T) {
	opts := DefaultScreenshotOptions()

	if opts.Format != ScreenshotFormatPNG {
		t.Errorf("Format = %q, expected %q", opts.Format, ScreenshotFormatPNG)
	}
	if opts.Quality != ScreenshotQualityMax {
		t.Errorf("Quality = %d, expected %d", opts.Quality, ScreenshotQualityMax)
	}
	if !opts.FromSurface {
		t.Error("FromSurface should be true")
	}
	if opts.CaptureBeyondViewport {
		t.Error("CaptureBeyondViewport should be false")
	}
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(result-tc.expected) > 0.001 {
				t.Errorf("CompareScreenshots = %f, expected %f", result, tc.expected)
			}
		})
	}
}

func TestCompareScreenshots_Unit_ErrorIsAlwaysNil(t *testing.T) {
	_, err := CompareScreenshots([]byte{0x00}, []byte{0xFF, 0xFF})
	if err != nil {
		t.Fatalf("CompareScreenshots error should always be nil, got: %v", err)
	}
}
