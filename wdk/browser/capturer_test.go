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

package browser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

func TestParseScreenshotFormat(t *testing.T) {
	testCases := map[string]browser_provider_chromedp.ScreenshotFormat{
		"jpeg": browser_provider_chromedp.ScreenshotFormatJPEG,
		"jpg":  browser_provider_chromedp.ScreenshotFormatJPEG,
		"webp": browser_provider_chromedp.ScreenshotFormatWebP,
		"png":  browser_provider_chromedp.ScreenshotFormatPNG,
		"":     browser_provider_chromedp.ScreenshotFormatPNG,
		"gif":  browser_provider_chromedp.ScreenshotFormatPNG,
	}
	for input, want := range testCases {
		assert.Equal(t, want, parseScreenshotFormat(input), "format %q", input)
	}
}

func TestDefaultCaptureOptions(t *testing.T) {
	options := DefaultCaptureOptions()
	assert.True(t, options.Headless)
	assert.Equal(t, "jpeg", options.ScreenshotFormat)
	assert.Equal(t, defaultScreenshotQuality, options.ScreenshotQuality)
	assert.Less(t, options.ScreenshotQuality, screenshotQualityLossless)
	assert.Equal(t, 2*time.Second, options.NetworkIdleDuration)
	assert.Positive(t, options.ViewportWidth)
	assert.Positive(t, options.ViewportHeight)
	assert.Positive(t, options.MaxScrollPasses)
	assert.InDelta(t, 1.0, options.ScreenshotScale, 0)
	assert.False(t, options.ChunkScreenshots)
	assert.False(t, options.IncludeShadowDOM)
}
