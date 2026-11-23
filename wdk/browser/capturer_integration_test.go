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

//go:build integration

package browser

import (
	"bytes"
	"image"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fastCaptureOptions() CaptureOptions {
	options := DefaultCaptureOptions()
	options.ViewportWidth = 400
	options.ViewportHeight = 300
	options.ScrollStepDelay = time.Millisecond
	options.NetworkIdleDuration = 10 * time.Millisecond
	options.NetworkIdleTimeout = 500 * time.Millisecond
	options.NavigationTimeout = 10 * time.Second
	options.MaxScrollPasses = 1
	return options
}

func TestCapturerCaptureURL(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("no local Chrome or Chromium executable")
	}
	server := httptest.NewServer(fixtureHandler())
	t.Cleanup(server.Close)

	testCases := []struct {
		configure func(*CaptureOptions)
		name      string
		format    string
		chunked   bool
	}{
		{name: "jpeg", format: "jpeg", configure: func(*CaptureOptions) {}},
		{name: "png", format: "png", configure: func(options *CaptureOptions) {
			options.ScreenshotFormat = "png"
			options.ScreenshotQuality = 100
			options.IncludeShadowDOM = true
		}},
		{name: "chunks", chunked: true, configure: func(options *CaptureOptions) {
			options.ChunkScreenshots = true
			options.ScreenshotFormat = "webp"
		}},
	}
	options := fastCaptureOptions()
	capturer, err := NewCapturer(options)
	require.NoError(t, err)
	defer capturer.Close()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			configured := options
			testCase.configure(&configured)
			capturer.opts = configured
			result, err := capturer.CaptureURL(server.URL + "/")
			require.NoError(t, err)
			assert.Equal(t, server.URL+"/", result.URL)
			assert.Contains(t, result.HTML, "Hello")
			if testCase.chunked {
				require.NotEmpty(t, result.Screenshots)
				return
			}
			_, format, err := image.DecodeConfig(bytes.NewReader(result.Screenshot))
			require.NoError(t, err)
			assert.Equal(t, testCase.format, format)
		})
	}

	_, err = capturer.CaptureURL("http://127.0.0.1:1/")
	require.ErrorContains(t, err, "navigating to")

	capturer.Close()
	_, err = capturer.CaptureURL(server.URL)
	require.ErrorContains(t, err, "creating page")
}
