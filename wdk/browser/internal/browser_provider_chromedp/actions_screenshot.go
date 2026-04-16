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
	"fmt"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// ScreenshotFormat specifies the image format for screenshots.
type ScreenshotFormat string

const (
	// ScreenshotFormatPNG creates a PNG image (default, lossless).
	ScreenshotFormatPNG ScreenshotFormat = "png"

	// ScreenshotFormatJPEG creates a JPEG image (lossy, smaller files).
	ScreenshotFormatJPEG ScreenshotFormat = "jpeg"

	// ScreenshotFormatWebP specifies the WebP image format for screenshots.
	ScreenshotFormatWebP ScreenshotFormat = "webp"

	// ScreenshotQualityMax is the maximum quality for lossy formats (100%).
	ScreenshotQualityMax = 100
)

// ScreenshotOptions configures how screenshots are taken.
type ScreenshotOptions struct {
	// Format specifies the image format (png, jpeg, or webp).
	Format ScreenshotFormat

	// Quality specifies the image quality from 0 to 100. Only applies to
	// JPEG and WebP formats.
	Quality int

	// FromSurface captures the screenshot from the surface rather than the view.
	FromSurface bool

	// CaptureBeyondViewport captures content outside the visible browser window.
	CaptureBeyondViewport bool

	// OptimiseForSpeed trades encoding efficiency for faster capture during
	// high-frequency screenshot sequences.
	OptimiseForSpeed bool
}

// DefaultScreenshotOptions returns sensible defaults for screenshots.
//
// Returns ScreenshotOptions which is configured with PNG format, maximum
// quality, surface capture enabled, and viewport-only capture.
func DefaultScreenshotOptions() ScreenshotOptions {
	return ScreenshotOptions{
		Format:                ScreenshotFormatPNG,
		Quality:               ScreenshotQualityMax,
		FromSurface:           true,
		CaptureBeyondViewport: false,
	}
}

// ScreenshotWithFormat captures a screenshot with a specified format and quality.
// Use this for JPEG or WebP formats that support quality settings.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes opts (ScreenshotOptions) which specifies format, quality, and capture
// settings.
//
// Returns []byte which contains the screenshot image data.
// Returns error when the screenshot capture fails.
func ScreenshotWithFormat(ctx *ActionContext, opts ScreenshotOptions) ([]byte, error) {
	var buffer []byte

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		format := page.CaptureScreenshotFormatPng
		switch opts.Format {
		case ScreenshotFormatJPEG:
			format = page.CaptureScreenshotFormatJpeg
		case ScreenshotFormatWebP:
			format = page.CaptureScreenshotFormatWebp
		}

		var err error
		buffer, err = page.CaptureScreenshot().
			WithFormat(format).
			WithQuality(int64(opts.Quality)).
			WithFromSurface(opts.FromSurface).
			WithCaptureBeyondViewport(opts.CaptureBeyondViewport).
			WithOptimizeForSpeed(opts.OptimiseForSpeed).
			Do(ctx2)
		return err
	}))
	if err != nil {
		return nil, fmt.Errorf("capturing screenshot with options: %w", err)
	}

	return buffer, nil
}

// ScreenshotJPEG captures a JPEG screenshot with the specified quality.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes quality (int) which specifies the image quality from 0-100, where 100
// is best quality but largest file size.
//
// Returns []byte which contains the JPEG image data.
// Returns error when the screenshot cannot be captured.
func ScreenshotJPEG(ctx *ActionContext, quality int) ([]byte, error) {
	opts := DefaultScreenshotOptions()
	opts.Format = ScreenshotFormatJPEG
	opts.Quality = quality
	return ScreenshotWithFormat(ctx, opts)
}

// ScreenshotWebP captures a WebP screenshot with the specified quality.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// screenshot.
// Takes quality (int) which sets the image quality from 0-100, where 100 is
// best quality.
//
// Returns []byte which contains the WebP-encoded image data.
// Returns error when the screenshot cannot be captured.
func ScreenshotWebP(ctx *ActionContext, quality int) ([]byte, error) {
	opts := DefaultScreenshotOptions()
	opts.Format = ScreenshotFormatWebP
	opts.Quality = quality
	return ScreenshotWithFormat(ctx, opts)
}

// ScreenshotRegion captures a screenshot of a specific viewport region.
// The coordinates are relative to the viewport (not the document).
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes x (float64) which specifies the left edge of the region.
// Takes y (float64) which specifies the top edge of the region.
// Takes width (float64) which specifies the region width.
// Takes height (float64) which specifies the region height.
//
// Returns []byte which contains the PNG-encoded screenshot data.
// Returns error when the screenshot capture fails.
func ScreenshotRegion(ctx *ActionContext, x, y, width, height float64) ([]byte, error) {
	var buffer []byte

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		var err error
		buffer, err = page.CaptureScreenshot().
			WithFormat(page.CaptureScreenshotFormatPng).
			WithClip(&page.Viewport{
				X:      x,
				Y:      y,
				Width:  width,
				Height: height,
				Scale:  1,
			}).
			Do(ctx2)
		return err
	}))
	if err != nil {
		return nil, fmt.Errorf("capturing region screenshot: %w", err)
	}

	return buffer, nil
}

// ScreenshotElementWithPadding captures a screenshot of an element with extra
// padding around it.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which identifies the target element.
// Takes padding (float64) which specifies the extra space around the element.
//
// Returns []byte which contains the screenshot image data.
// Returns error when the element cannot be found or has invalid bounds.
func ScreenshotElementWithPadding(ctx *ActionContext, selector string, padding float64) ([]byte, error) {
	js := scripts.MustExecute("element_bounds_with_padding.js.tmpl", map[string]any{
		"Selector": selector,
		"Padding":  padding,
	})

	var bounds map[string]any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &bounds))
	if err != nil {
		return nil, fmt.Errorf("getting element bounds: %w", err)
	}
	if bounds == nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	x, ok := bounds["x"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid x coordinate type for element: %s", selector)
	}
	y, ok := bounds["y"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid y coordinate type for element: %s", selector)
	}
	width, ok := bounds["width"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid width type for element: %s", selector)
	}
	height, ok := bounds["height"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid height type for element: %s", selector)
	}

	if x < 0 {
		width += x
		x = 0
	}
	if y < 0 {
		height += y
		y = 0
	}

	return ScreenshotRegion(ctx, x, y, width, height)
}

// CompareScreenshots compares two screenshots and returns the percentage of
// differing bytes.
//
// This is a simple comparison that returns 0.0 if identical, 1.0 if completely
// different. This function only works with same-size images and compares raw
// bytes.
//
// Takes a ([]byte) which is the first screenshot as raw bytes.
// Takes b ([]byte) which is the second screenshot as raw bytes.
//
// Returns float64 which is the difference ratio from 0.0 to 1.0.
// Returns error which is always nil for this implementation.
func CompareScreenshots(a, b []byte) (float64, error) {
	if len(a) != len(b) {
		return 1.0, nil
	}

	if len(a) == 0 {
		return 0.0, nil
	}

	var diffCount int
	for i := range a {
		if a[i] != b[i] {
			diffCount++
		}
	}

	return float64(diffCount) / float64(len(a)), nil
}
