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
	"bytes"
	"image/png"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testHTMLScreenshot = `<!DOCTYPE html>
	<html>
	<head><title>Screenshot Test</title></head>
	<body style="margin:0;padding:20px;background:#f0f0f0;">
	<div id="box" style="width:100px;height:100px;background:red;margin:10px;"></div>
	<div id="content" style="width:200px;height:50px;background:blue;margin:10px;"></div>
	</body>
	</html>`
	testHTMLTallPage = `<!DOCTYPE html>
	<html>
	<head><title>Tall Page</title></head>
	<body style="margin:0;">
	<div style="height:1600px;background:linear-gradient(red, blue);"></div>
	</body>
	</html>`
)

func TestScreenshotFormats(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLScreenshot)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("captures JPEG screenshot", func(t *testing.T) {
			data, err := ScreenshotJPEG(ctx, 80)
			require.NoError(t, err)
			assert.True(t, bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}), "expected JPEG data")
		})

		t.Run("captures WebP screenshot", func(t *testing.T) {
			data, err := ScreenshotWebP(ctx, 80)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(data), 12)
			assert.Equal(t, "RIFF", string(data[0:4]))
			assert.Equal(t, "WEBP", string(data[8:12]))
		})

		t.Run("captures screenshot with custom options", func(t *testing.T) {
			opts := DefaultScreenshotOptions()
			opts.Format = ScreenshotFormatJPEG
			opts.Quality = 50

			data, err := ScreenshotWithFormat(ctx, opts)
			require.NoError(t, err)
			assert.NotEmpty(t, data)
		})

		t.Run("reports the timeout cause when the capture runs out of time", func(t *testing.T) {
			opts := DefaultScreenshotOptions()
			opts.Timeout = time.Nanosecond

			_, err := ScreenshotWithFormat(ctx, opts)
			require.ErrorIs(t, err, errScreenshotTimedOut)
		})
	})
}

func TestScreenshotRegion(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLScreenshot)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("captures region screenshot", func(t *testing.T) {
			data, err := ScreenshotRegion(ctx, 10, 10, 100, 100)
			require.NoError(t, err)
			requirePNGSize(t, data, 100, 100)
		})

		t.Run("captures different region", func(t *testing.T) {
			data, err := ScreenshotRegion(ctx, 0, 0, 200, 150)
			require.NoError(t, err)
			requirePNGSize(t, data, 200, 150)
		})

		t.Run("captures the same region repeatedly on an idle page", func(t *testing.T) {
			for range 5 {
				data, err := ScreenshotRegion(ctx, 0, 0, 200, 150, WithCaptureTimeout(20*time.Second))
				require.NoError(t, err)
				requirePNGSize(t, data, 200, 150)
			}
		})

		t.Run("reports the timeout cause when the capture runs out of time", func(t *testing.T) {
			_, err := ScreenshotRegion(ctx, 0, 0, 200, 150, WithCaptureTimeout(time.Nanosecond))
			require.ErrorIs(t, err, errScreenshotTimedOut)
		})
	})
}

func TestScreenshotRegion_WithSeveralPagesInOneBrowser(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLScreenshot)
	defer server.Close()

	pool := requireExclusivePool(t)
	browser, err := pool.Acquire(t.Context())
	require.NoError(t, err)
	defer pool.Release(browser)

	first, err := browser.NewIncognitoPage()
	require.NoError(t, err)
	defer func() { assert.NoError(t, first.Close()) }()

	firstHelper := NewPageHelper(first.Ctx)
	defer firstHelper.Close()
	require.NoError(t, firstHelper.Navigate(server.URL))

	second, err := browser.NewIncognitoPage()
	require.NoError(t, err)
	defer func() { assert.NoError(t, second.Close()) }()

	secondHelper := NewPageHelper(second.Ctx)
	defer secondHelper.Close()
	require.NoError(t, secondHelper.Navigate(server.URL))

	for _, helper := range []*PageHelper{firstHelper, secondHelper, firstHelper, secondHelper} {
		data, err := ScreenshotRegion(newActionContext(helper), 0, 0, 120, 80, WithCaptureTimeout(20*time.Second))
		require.NoError(t, err)
		requirePNGSize(t, data, 120, 80)
	}
}

func TestScreenshotElementWithPadding(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLScreenshot)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("captures element with padding", func(t *testing.T) {
			data, err := ScreenshotElementWithPadding(ctx, "#box", 10)
			require.NoError(t, err)
			requirePNGSize(t, data, 120, 120)
		})

		t.Run("captures the element again on an idle page", func(t *testing.T) {
			data, err := ScreenshotElementWithPadding(ctx, "#content", 5, WithCaptureTimeout(20*time.Second))
			require.NoError(t, err)
			requirePNGSize(t, data, 210, 60)
		})

		t.Run("returns error for non-existent element", func(t *testing.T) {
			_, err := ScreenshotElementWithPadding(ctx, "#nonexistent", 10)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "element not found")
		})

		t.Run("reports the timeout cause when the capture runs out of time", func(t *testing.T) {
			_, err := ScreenshotElementWithPadding(ctx, "#box", 10, WithCaptureTimeout(time.Nanosecond))
			require.ErrorIs(t, err, errScreenshotTimedOut)
		})
	})
}

func TestFullPageScreenshots(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := newTestServer(testHTMLTallPage)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		t.Run("captures the whole page as PNG", func(t *testing.T) {
			data, err := FullPageScreenshot(page.Ctx())
			require.NoError(t, err)
			image, err := png.Decode(bytes.NewReader(data))
			require.NoError(t, err)
			assert.GreaterOrEqual(t, image.Bounds().Dy(), 1500)
		})

		t.Run("captures the whole page as JPEG", func(t *testing.T) {
			data, err := FullPageScreenshotWithFormat(page.Ctx(), ScreenshotFormatJPEG, 70)
			require.NoError(t, err)
			assert.True(t, bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}), "expected JPEG data")
		})

		t.Run("captures the page in viewport sized chunks", func(t *testing.T) {
			chunks, err := FullPageScreenshotChunks(page.Ctx(), 400, 500, ChunkScreenshotOptions{
				Format:  ScreenshotFormatPNG,
				Quality: ScreenshotQualityMax,
				Scale:   1,
			})
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(chunks), 3)
			for index, chunk := range chunks {
				assert.Equal(t, index, chunk.Index)
				image, decodeErr := png.Decode(bytes.NewReader(chunk.Data))
				require.NoError(t, decodeErr)
				assert.Equal(t, 400, image.Bounds().Dx())
				assert.LessOrEqual(t, image.Bounds().Dy(), 500)
			}
		})
	})
}

func TestCompareScreenshots(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testCases := []struct {
		name     string
		a        []byte
		b        []byte
		expected float64
	}{
		{name: "identical screenshots return 0", a: []byte{1, 2, 3, 4, 5}, b: []byte{1, 2, 3, 4, 5}, expected: 0},
		{name: "different sizes return 1", a: []byte{1, 2, 3}, b: []byte{1, 2, 3, 4, 5}, expected: 1},
		{name: "empty screenshots return 0", a: []byte{}, b: []byte{}, expected: 0},
		{name: "partially different returns fraction", a: []byte{1, 2, 3, 4}, b: []byte{1, 2, 0, 0}, expected: 0.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff, err := CompareScreenshots(tc.a, tc.b)
			require.NoError(t, err)
			assert.InDelta(t, tc.expected, diff, 0.001)
		})
	}
}

func requirePNGSize(t *testing.T, data []byte, width, height int) {
	t.Helper()

	image, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err, "expected PNG data")
	assert.Equal(t, width, image.Bounds().Dx())
	assert.Equal(t, height, image.Bounds().Dy())
}
