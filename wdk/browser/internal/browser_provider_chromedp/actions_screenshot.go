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
	"fmt"
	"time"

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

	// defaultScreenshotTimeout bounds a single screenshot capture when no timeout is
	// configured. A capture normally completes in well under a second; the limit only stops
	// a stalled capture from blocking its caller indefinitely.
	defaultScreenshotTimeout = 60 * time.Second
)

var (
	// errScreenshotTimedOut is the cancellation cause recorded when a screenshot capture
	// exceeds its timeout.
	errScreenshotTimedOut = errors.New("screenshot capture timed out")
)

// ScreenshotOptions configures how screenshots are taken.
type ScreenshotOptions struct {
	// Format specifies the image format (png, jpeg, or webp).
	Format ScreenshotFormat

	// Quality specifies the image quality from 0 to 100. Only applies to JPEG and WebP
	// formats.
	Quality int

	// FromSurface captures the screenshot from the surface rather than the view.
	FromSurface bool

	// CaptureBeyondViewport captures content outside the visible browser window.
	CaptureBeyondViewport bool

	// OptimiseForSpeed trades encoding efficiency for faster capture during high-frequency
	// screenshot sequences.
	OptimiseForSpeed bool

	// Timeout bounds the capture. Zero or a negative value uses the default of 60 seconds.
	Timeout time.Duration
}

// CaptureOption configures a region or element screenshot capture.
type CaptureOption func(*captureSettings)

// captureSettings holds the resolved configuration for a region or element capture.
type captureSettings struct {
	// timeout bounds the whole capture, including locating the element.
	timeout time.Duration
}

// DefaultScreenshotOptions returns sensible defaults for screenshots.
//
// Returns ScreenshotOptions which is configured with PNG format, maximum quality, surface
// capture enabled, and viewport-only capture.
func DefaultScreenshotOptions() ScreenshotOptions {
	return ScreenshotOptions{
		Format:                ScreenshotFormatPNG,
		Quality:               ScreenshotQualityMax,
		FromSurface:           true,
		CaptureBeyondViewport: false,
		OptimiseForSpeed:      false,
		Timeout:               defaultScreenshotTimeout,
	}
}

// WithCaptureTimeout bounds how long a region or element capture may take. Zero or a
// negative value keeps the default of 60 seconds.
//
// Takes timeout (time.Duration) which is the maximum duration of the capture.
//
// Returns CaptureOption which applies the timeout.
func WithCaptureTimeout(timeout time.Duration) CaptureOption {
	return func(settings *captureSettings) {
		if timeout > 0 {
			settings.timeout = timeout
		}
	}
}

// ScreenshotWithFormat captures a screenshot with a specified format and quality. Use
// this for JPEG or WebP formats that support quality settings.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes opts (ScreenshotOptions) which specifies format, quality, and capture settings.
//
// Returns []byte which contains the screenshot image data.
// Returns error when the screenshot capture fails.
func ScreenshotWithFormat(ctx *ActionContext, opts ScreenshotOptions) ([]byte, error) {
	format := page.CaptureScreenshotFormatPng
	switch opts.Format { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case ScreenshotFormatJPEG:
		format = page.CaptureScreenshotFormatJpeg
	case ScreenshotFormatWebP:
		format = page.CaptureScreenshotFormatWebp
	}

	buffer, err := captureScreenshot(ctx.Ctx, resolveCaptureTimeout(opts.Timeout), page.CaptureScreenshot().
		WithFormat(format).
		WithQuality(int64(opts.Quality)).
		WithFromSurface(opts.FromSurface).
		WithCaptureBeyondViewport(opts.CaptureBeyondViewport).
		WithOptimizeForSpeed(opts.OptimiseForSpeed))
	if err != nil {
		return nil, fmt.Errorf("capturing screenshot with options: %w", err)
	}

	return buffer, nil
}

// ScreenshotJPEG captures a JPEG screenshot with the specified quality.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
// Takes quality (int) which specifies the image quality from 0-100, where 100 is best
// quality but largest file size.
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
// Takes ctx (*ActionContext) which provides the browser context for the screenshot.
// Takes quality (int) which sets the image quality from 0-100, where 100 is best quality.
//
// Returns []byte which contains the WebP-encoded image data.
// Returns error when the screenshot cannot be captured.
func ScreenshotWebP(ctx *ActionContext, quality int) ([]byte, error) {
	opts := DefaultScreenshotOptions()
	opts.Format = ScreenshotFormatWebP
	opts.Quality = quality
	return ScreenshotWithFormat(ctx, opts)
}

// ScreenshotRegion captures a screenshot of a specific viewport region. The coordinates
// are relative to the viewport (not the document).
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes x (float64) which specifies the left edge of the region.
// Takes y (float64) which specifies the top edge of the region.
// Takes width (float64) which specifies the region width.
// Takes height (float64) which specifies the region height.
// Takes opts (...CaptureOption) which adjust the capture, such as its timeout.
//
// Returns []byte which contains the PNG-encoded screenshot data.
// Returns error when the screenshot capture fails or exceeds its timeout.
func ScreenshotRegion(ctx *ActionContext, x, y, width, height float64, opts ...CaptureOption) ([]byte, error) {
	settings := newCaptureSettings(opts)
	buffer, err := captureScreenshot(ctx.Ctx, settings.timeout, newRegionCapture(x, y, width, height))
	if err != nil {
		return nil, fmt.Errorf("capturing region screenshot: %w", err)
	}
	return buffer, nil
}

// ScreenshotElementWithPadding captures a screenshot of an element with extra padding
// around it.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes selector (string) which identifies the target element.
// Takes padding (float64) which specifies the extra space around the element.
// Takes opts (...CaptureOption) which adjust the capture, such as its timeout.
//
// Returns []byte which contains the screenshot image data.
// Returns error when the element cannot be found, has invalid bounds, or the capture
// exceeds its timeout.
func ScreenshotElementWithPadding(ctx *ActionContext, selector string, padding float64, opts ...CaptureOption) ([]byte, error) {
	settings := newCaptureSettings(opts)

	var buffer []byte
	err := runBoundedCapture(ctx.Ctx, settings.timeout, chromedp.ActionFunc(func(actionCtx context.Context) error {
		region, err := elementRegion(actionCtx, selector, padding)
		if err != nil {
			return err
		}
		buffer, err = region.Do(actionCtx)
		return err
	}))
	if err != nil {
		return nil, fmt.Errorf("capturing element screenshot of %s: %w", selector, err)
	}
	return buffer, nil
}

// CompareScreenshots compares two screenshots and reports the percentage of differing
// bytes.
//
// Yields 0.0 if identical, 1.0 if completely different. Only works with same-size images
// and compares raw bytes.
//
// Takes a ([]byte) which is the first screenshot as raw bytes.
// Takes b ([]byte) which is the second screenshot as raw bytes.
//
// Returns float64 which is the difference ratio from 0.0 to 1.0.
// Returns error which is always nil for the current implementation.
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

// newCaptureSettings applies the capture options over the defaults.
//
// Takes opts ([]CaptureOption) which are the options to apply in order.
//
// Returns captureSettings which holds the resolved configuration.
func newCaptureSettings(opts []CaptureOption) captureSettings {
	settings := captureSettings{timeout: defaultScreenshotTimeout}
	for _, opt := range opts {
		opt(&settings)
	}
	return settings
}

// resolveCaptureTimeout returns the configured timeout, or the default when it is not
// positive.
//
// Takes timeout (time.Duration) which is the configured timeout.
//
// Returns time.Duration which is the timeout to apply.
func resolveCaptureTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultScreenshotTimeout
	}
	return timeout
}

// newRegionCapture builds a PNG capture of a viewport region at a scale of one.
//
// Takes x (float64) which is the left edge of the region.
// Takes y (float64) which is the top edge of the region.
// Takes width (float64) which is the region width.
// Takes height (float64) which is the region height.
//
// Returns *page.CaptureScreenshotParams which is the clipped capture command.
func newRegionCapture(x, y, width, height float64) *page.CaptureScreenshotParams {
	return page.CaptureScreenshot().
		WithFormat(page.CaptureScreenshotFormatPng).
		WithClip(&page.Viewport{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
			Scale:  1,
		})
}

// elementRegion measures the element, including padding, and builds a clipped capture of
// the matching viewport region.
//
// Takes selector (string) which identifies the target element.
// Takes padding (float64) which specifies the extra space around the element.
//
// Returns *page.CaptureScreenshotParams which is the clipped capture command.
// Returns error when the element cannot be found or has invalid bounds.
func elementRegion(ctx context.Context, selector string, padding float64) (*page.CaptureScreenshotParams, error) {
	js := scripts.MustExecute("element_bounds_with_padding.js.tmpl", map[string]any{
		"Selector": selector,
		"Padding":  padding,
	})

	var bounds map[string]any
	if err := chromedp.Evaluate(js, &bounds).Do(ctx); err != nil {
		return nil, fmt.Errorf("getting element bounds: %w", err)
	}
	if bounds == nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}
	return regionFromBounds(bounds, selector)
}

// regionFromBounds converts element bounds reported by the page into a clipped capture,
// trimming any part of the region that lies above or left of the viewport.
//
// Takes bounds (map[string]any) which holds the x, y, width and height values.
// Takes selector (string) which identifies the element for error messages.
//
// Returns *page.CaptureScreenshotParams which is the clipped capture command.
// Returns error when a bound is missing or not a number.
func regionFromBounds(bounds map[string]any, selector string) (*page.CaptureScreenshotParams, error) {
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

	return newRegionCapture(x, y, width, height), nil
}

// captureScreenshot runs a screenshot command through runBoundedCapture and returns the
// encoded image.
//
// Takes timeout (time.Duration) which bounds the capture.
// Takes capture (*page.CaptureScreenshotParams) which is the screenshot command to run.
//
// Returns []byte which contains the encoded image data.
// Returns error when the capture fails or exceeds the timeout.
func captureScreenshot(ctx context.Context, timeout time.Duration, capture *page.CaptureScreenshotParams) ([]byte, error) {
	var buffer []byte
	err := runBoundedCapture(ctx, timeout, chromedp.ActionFunc(func(actionCtx context.Context) error {
		var captureErr error
		buffer, captureErr = capture.Do(actionCtx)
		return captureErr
	}))
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

// runBoundedCapture brings the page to the front of its window and then runs the capture
// action under a timeout. Chrome only produces compositor frames for the front tab of a
// window and Page.captureScreenshot waits for a fresh frame, so a capture of a background
// tab can otherwise stall indefinitely.
//
// Takes timeout (time.Duration) which bounds the whole capture.
// Takes capture (chromedp.Action) which performs the screenshot.
//
// Returns error when the capture fails, carrying the timeout cause when the deadline
// passed first.
func runBoundedCapture(ctx context.Context, timeout time.Duration, capture chromedp.Action) error {
	captureCtx, cancel := context.WithTimeoutCause(ctx, timeout,
		fmt.Errorf("%w after %s", errScreenshotTimedOut, timeout))
	defer cancel()

	if err := chromedp.Run(captureCtx, page.BringToFront(), capture); err != nil {
		return withCaptureCause(captureCtx, err)
	}
	return nil
}

// withCaptureCause attaches the context's cancellation cause to err when the context has
// ended, so a timed-out capture reports why it was cancelled.
//
// Takes err (error) which is the error returned by the capture.
//
// Returns error which wraps both err and the cancellation cause when there is one.
func withCaptureCause(ctx context.Context, err error) error {
	if ctx.Err() == nil {
		return err
	}
	cause := context.Cause(ctx)
	if cause == nil || errors.Is(err, cause) {
		return err
	}
	return fmt.Errorf("%w: %w", err, cause)
}
