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
	"testing"
)

const testHTMLScreenshot = `<!DOCTYPE html>
<html>
<head><title>Screenshot Test</title></head>
<body style="margin:0;padding:20px;background:#f0f0f0;">
<div id="box" style="width:100px;height:100px;background:red;margin:10px;"></div>
<div id="content" style="width:200px;height:50px;background:blue;margin:10px;"></div>
</body>
</html>`

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
			if err != nil {
				t.Fatalf("ScreenshotJPEG() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("ScreenshotJPEG() returned empty data")
			}

			if !bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
				t.Error("ScreenshotJPEG() did not return valid JPEG data")
			}
		})

		t.Run("captures WebP screenshot", func(t *testing.T) {
			data, err := ScreenshotWebP(ctx, 80)
			if err != nil {
				t.Fatalf("ScreenshotWebP() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("ScreenshotWebP() returned empty data")
			}

			if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
				t.Error("ScreenshotWebP() did not return valid WebP data")
			}
		})

		t.Run("captures screenshot with custom options", func(t *testing.T) {
			opts := DefaultScreenshotOptions()
			opts.Format = ScreenshotFormatJPEG
			opts.Quality = 50

			data, err := ScreenshotWithFormat(ctx, opts)
			if err != nil {
				t.Fatalf("ScreenshotWithFormat() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("ScreenshotWithFormat() returned empty data")
			}
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
			if err != nil {
				t.Fatalf("ScreenshotRegion() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("ScreenshotRegion() returned empty data")
			}

			if !bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) {
				t.Error("ScreenshotRegion() did not return valid PNG data")
			}
		})

		t.Run("captures different region", func(t *testing.T) {
			data, err := ScreenshotRegion(ctx, 0, 0, 200, 150)
			if err != nil {
				t.Fatalf("ScreenshotRegion() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("ScreenshotRegion() returned empty data")
			}
		})
	})
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
			if err != nil {
				t.Fatalf("ScreenshotElementWithPadding() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("ScreenshotElementWithPadding() returned empty data")
			}

			if !bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) {
				t.Error("ScreenshotElementWithPadding() did not return valid PNG data")
			}
		})

		t.Run("returns error for non-existent element", func(t *testing.T) {
			_, err := ScreenshotElementWithPadding(ctx, "#nonexistent", 10)
			if err == nil {
				t.Error("ScreenshotElementWithPadding() expected error for non-existent element")
			}
		})
	})
}

func TestCompareScreenshots(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("identical screenshots return 0", func(t *testing.T) {
		data := []byte{1, 2, 3, 4, 5}
		diff, err := CompareScreenshots(data, data)
		if err != nil {
			t.Fatalf("CompareScreenshots() error = %v", err)
		}
		if diff != 0 {
			t.Errorf("CompareScreenshots() = %v, want 0 for identical", diff)
		}
	})

	t.Run("different sizes return 1", func(t *testing.T) {
		a := []byte{1, 2, 3}
		b := []byte{1, 2, 3, 4, 5}
		diff, err := CompareScreenshots(a, b)
		if err != nil {
			t.Fatalf("CompareScreenshots() error = %v", err)
		}
		if diff != 1.0 {
			t.Errorf("CompareScreenshots() = %v, want 1.0 for different sizes", diff)
		}
	})

	t.Run("empty screenshots return 0", func(t *testing.T) {
		diff, err := CompareScreenshots([]byte{}, []byte{})
		if err != nil {
			t.Fatalf("CompareScreenshots() error = %v", err)
		}
		if diff != 0 {
			t.Errorf("CompareScreenshots() = %v, want 0 for empty", diff)
		}
	})

	t.Run("partially different returns fraction", func(t *testing.T) {
		a := []byte{1, 2, 3, 4}
		b := []byte{1, 2, 0, 0}
		diff, err := CompareScreenshots(a, b)
		if err != nil {
			t.Fatalf("CompareScreenshots() error = %v", err)
		}

		if diff != 0.5 {
			t.Errorf("CompareScreenshots() = %v, want 0.5", diff)
		}
	})
}
