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
	"strconv"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

const (
	// defaultScrollStepDelay is the default pause between scroll steps.
	defaultScrollStepDelay = 500 * time.Millisecond

	// defaultNetworkIdleTimeout is the default maximum time to wait for
	// network idle after scrolling.
	defaultNetworkIdleTimeout = 10 * time.Second

	// defaultMaxScrollPasses is the default number of top-to-bottom scroll
	// passes.
	defaultMaxScrollPasses = 3
)

// ScrollCaptureOptions configures the scroll-and-capture behaviour for
// triggering lazy-loaded content on external pages.
type ScrollCaptureOptions struct {
	// ScrollStepDelay is the pause between scroll increments, giving
	// lazy-loaded images time to swap from blurry placeholders to sharp
	// versions. Default: 500ms.
	ScrollStepDelay time.Duration

	// NetworkIdleDuration is how long the network must be quiet before
	// considering it idle. Default: 2s.
	NetworkIdleDuration time.Duration

	// NetworkIdleTimeout is the maximum time to wait for network idle after
	// scrolling. Default: 10s.
	NetworkIdleTimeout time.Duration

	// MaxScrollPasses is the maximum number of top-to-bottom scroll passes,
	// where pages with infinite scroll may grow after the first pass.
	// Default: 3.
	MaxScrollPasses int
}

// scrollContainerInfo holds information about the page's scroll container.
type scrollContainerInfo struct {
	// Selector is the CSS selector for the scroll container, or empty if the
	// window is the scroll container.
	Selector string

	// ScrollHeight is the total scrollable height of the container.
	ScrollHeight float64

	// ClientHeight is the visible height of the container.
	ClientHeight float64
}

// pageDimensions holds the scroll height and viewport dimensions of a page.
type pageDimensions struct {
	// ScrollHeight is the total scrollable height in pixels.
	ScrollHeight float64

	// ViewportWidth is the visible viewport width in pixels.
	ViewportWidth float64

	// ViewportHeight is the visible viewport height in pixels.
	ViewportHeight float64
}

// ScreenshotChunk represents one tile of a chunked full-page screenshot.
type ScreenshotChunk struct {
	// Data is the image data for this chunk.
	Data []byte

	// Index is the zero-based position of this chunk (top to bottom).
	Index int
}

// ChunkScreenshotOptions configures chunked screenshot capture.
type ChunkScreenshotOptions struct {
	// Format specifies the image format (png, jpeg, or webp).
	Format ScreenshotFormat

	// Quality specifies the image quality for lossy formats (0-100).
	Quality int

	// Scale is the output scale factor. 1.0 = original, 0.5 = half size.
	Scale float64
}

// chunkCaptureParams bundles the parameters for capturing a
// single screenshot chunk, keeping the function signature under
// the argument limit.
type chunkCaptureParams struct {
	// cdpFormat is the CDP screenshot format enum.
	cdpFormat page.CaptureScreenshotFormat

	// quality is the image quality for lossy formats (0-100).
	quality int

	// scale is the output scale factor (1.0 = original).
	scale float64

	// chunkW is the chunk width in pixels before scaling.
	chunkW float64

	// chunkH is the chunk height in pixels before scaling.
	chunkH float64

	// totalHeight is the full page scroll height in pixels.
	totalHeight float64
}

// DefaultScrollCaptureOptions returns sensible defaults for scroll capture.
//
// Returns ScrollCaptureOptions which is configured with 500ms scroll delay,
// 2s network idle duration, 10s network idle timeout, and 3 max scroll passes.
func DefaultScrollCaptureOptions() ScrollCaptureOptions {
	return ScrollCaptureOptions{
		ScrollStepDelay:     defaultScrollStepDelay,
		NetworkIdleDuration: 2 * time.Second,
		NetworkIdleTimeout:  defaultNetworkIdleTimeout,
		MaxScrollPasses:     defaultMaxScrollPasses,
	}
}

// InjectObserverOverride injects a script that runs before any page JavaScript
// and overrides IntersectionObserver so that every observed element immediately
// triggers its callback with isIntersecting=true. This forces frameworks like
// Angular and React that use IntersectionObserver for lazy-loading to load all
// content eagerly rather than waiting for elements to enter the viewport.
//
// Must be called before navigation so the override is in place when the page's
// JavaScript initialises.
//
// Takes ctx (context.Context) which is the chromedp browser context.
//
// Returns error when the script injection fails.
func InjectObserverOverride(ctx context.Context) error {
	js := scripts.MustGet("intercept_intersection_observer.js")

	return chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(js).Do(ctx)
		return err
	}))
}

// DismissCookieConsent attempts to find and click a cookie consent accept
// button, checking known consent libraries (CookieBot, OneTrust, etc.) and
// falling back to text-matching heuristics for generic accept buttons on a
// best-effort basis where errors are ignored since not all pages have banners.
//
// Takes ctx (context.Context) which is the chromedp browser context.
func DismissCookieConsent(ctx context.Context) {
	js := scripts.MustGet("dismiss_cookie_consent.js")

	var result bool
	_ = chromedp.Run(ctx, chromedp.Evaluate(js, &result))

	if result {
		time.Sleep(defaultScrollStepDelay)
	}
}

// NavigateToURL navigates to an absolute URL and waits for the page to load
// and stabilise. Unlike the test-oriented Navigate, this does not prepend a
// server URL.
//
// Takes ctx (context.Context) which is the chromedp browser context.
// Takes url (string) which is the absolute URL to navigate to.
// Takes timeout (time.Duration) which is the maximum time to wait for
// navigation.
//
// Returns error when navigation fails or the page does not become ready.
func NavigateToURL(ctx context.Context, url string, timeout time.Duration) error {
	timedCtx, cancel := context.WithTimeoutCause(
		ctx, timeout,
		fmt.Errorf("capture NavigateToURL exceeded %s timeout", timeout),
	)
	defer cancel()

	err := chromedp.Run(timedCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		return fmt.Errorf("navigating to %s: %w", url, err)
	}

	_ = WaitStable(timedCtx, defaultScrollStepDelay)

	return nil
}

// ScrollForLazyContent scrolls through a page incrementally to trigger
// IntersectionObserver-based lazy loading. It detects whether the page uses
// a nested scroll container (common in Angular/React SPAs) or the normal
// window scroll, and scrolls the correct element.
//
// Takes ctx (context.Context) which is the chromedp browser context.
// Takes opts (ScrollCaptureOptions) which configures scroll behaviour.
//
// Returns error when a critical scroll operation fails.
func ScrollForLazyContent(ctx context.Context, opts ScrollCaptureOptions) error {
	container, err := findScrollContainer(ctx)
	if err != nil {
		return fmt.Errorf("detecting scroll container: %w", err)
	}

	selector := container.Selector

	for pass := range opts.MaxScrollPasses {
		grown, passErr := runScrollPass(ctx, selector, pass, opts)
		if passErr != nil {
			return passErr
		}
		if !grown {
			break
		}
	}

	_ = scrollTo(ctx, selector, 0)
	time.Sleep(100 * time.Millisecond)

	return nil
}

// UnwrapScrollContainer removes overflow/height constraints from the page so
// that chromedp.FullScreenshot can capture the entire page content. This is
// necessary for SPA sites that use a nested scroll container with
// html,body{height:100%;overflow:hidden}.
//
// Takes ctx (context.Context) which is the chromedp browser context.
//
// Returns error when the JavaScript evaluation fails.
func UnwrapScrollContainer(ctx context.Context) error {
	container, err := findScrollContainer(ctx)
	if err != nil {
		return fmt.Errorf("finding scroll container to unwrap: %w", err)
	}

	selector := container.Selector
	if selector == "" {
		return nil
	}

	js := scripts.MustExecute("unwrap_scroll_container.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var result bool
	err = chromedp.Run(ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return fmt.Errorf("unwrapping scroll container: %w", err)
	}

	time.Sleep(200 * time.Millisecond)

	return nil
}

// ForceLoadAllImages removes loading="lazy" from all images, forces the
// browser to start fetching them, and waits for all images to complete.
// Call this after UnwrapScrollContainer and before taking a screenshot.
//
// Takes ctx (context.Context) which is the chromedp browser context.
// Takes timeout (time.Duration) which is the maximum time to wait for all
// images to load.
//
// Returns error when the JavaScript evaluation fails.
func ForceLoadAllImages(ctx context.Context, timeout time.Duration) error {
	js := scripts.MustExecute("force_load_all_images.js.tmpl", map[string]any{
		"Timeout": timeout.Milliseconds(),
	})

	var result bool
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &result, chromedp.EvalAsValue))
	if err != nil {
		return fmt.Errorf("forcing image load: %w", err)
	}
	return nil
}

// GetFullPageHTML captures the complete outer HTML of the document.
//
// Takes ctx (context.Context) which is the chromedp browser context.
//
// Returns string which contains the full HTML source of the page.
// Returns error when the HTML cannot be captured.
func GetFullPageHTML(ctx context.Context) (string, error) {
	var html string
	err := chromedp.Run(ctx,
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return "", fmt.Errorf("capturing full page HTML: %w", err)
	}
	return html, nil
}

// FullPageScreenshot captures a PNG screenshot of the entire page, including
// content below the viewport.
//
// Takes ctx (context.Context) which is the chromedp browser context.
//
// Returns []byte which contains the PNG image data.
// Returns error when the screenshot cannot be captured.
func FullPageScreenshot(ctx context.Context) ([]byte, error) {
	var buffer []byte
	err := chromedp.Run(ctx,
		chromedp.FullScreenshot(&buffer, ScreenshotQualityFull),
	)
	if err != nil {
		return nil, fmt.Errorf("capturing full page screenshot: %w", err)
	}
	return buffer, nil
}

// FullPageScreenshotWithFormat captures a screenshot of the entire page in the
// specified format and quality. Unlike FullPageScreenshot which always produces
// PNG, this supports JPEG and WebP output.
//
// Takes ctx (context.Context) which is the chromedp browser context.
// Takes format (ScreenshotFormat) which specifies the image format.
// Takes quality (int) which specifies the quality for lossy formats (0-100).
//
// Returns []byte which contains the image data.
// Returns error when the screenshot cannot be captured.
func FullPageScreenshotWithFormat(
	ctx context.Context, format ScreenshotFormat, quality int,
) ([]byte, error) {
	dims, err := getPageDimensions(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting page dimensions for screenshot: %w", err)
	}

	cdpFormat := toCDPFormat(format)

	var buffer []byte
	err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		var captureErr error
		buffer, captureErr = page.CaptureScreenshot().
			WithFormat(cdpFormat).
			WithQuality(int64(quality)).
			WithCaptureBeyondViewport(true).
			WithClip(&page.Viewport{
				X:      0,
				Y:      0,
				Width:  dims.ViewportWidth,
				Height: dims.ScrollHeight,
				Scale:  1,
			}).
			Do(ctx2)
		return captureErr
	}))
	if err != nil {
		return nil, fmt.Errorf("capturing full page screenshot with format: %w", err)
	}
	return buffer, nil
}

// FullPageScreenshotChunks captures the full page as viewport-sized chunks.
// Each chunk is at most viewportWidth x viewportHeight pixels (before scaling),
// making them suitable for APIs with per-image dimension limits.
//
// Takes ctx (context.Context) which is the chromedp browser context.
// Takes viewportWidth (int64) which is the configured viewport width.
// Takes viewportHeight (int64) which is the configured viewport height.
// Takes opts (ChunkScreenshotOptions) which configures format, quality, and
// scale.
//
// Returns []ScreenshotChunk which contains the ordered image chunks.
// Returns error when any chunk capture fails.
func FullPageScreenshotChunks(
	ctx context.Context,
	viewportWidth, viewportHeight int64,
	opts ChunkScreenshotOptions,
) ([]ScreenshotChunk, error) {
	dims, err := getPageDimensions(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting page dimensions for chunks: %w", err)
	}

	scale := opts.Scale
	if scale <= 0 {
		scale = 1.0
	}

	chunkH := float64(viewportHeight)
	totalHeight := dims.ScrollHeight
	if totalHeight <= 0 {
		totalHeight = chunkH
	}

	params := chunkCaptureParams{
		cdpFormat:   toCDPFormat(opts.Format),
		quality:     opts.Quality,
		scale:       scale,
		chunkW:      float64(viewportWidth),
		chunkH:      chunkH,
		totalHeight: totalHeight,
	}

	var chunks []ScreenshotChunk
	index := 0

	for y := 0.0; y < totalHeight; y += chunkH {
		buffer, captureErr := captureChunk(ctx, params, y)
		if captureErr != nil {
			return nil, fmt.Errorf("capturing chunk %d at y=%.0f: %w", index, y, captureErr)
		}

		chunks = append(chunks, ScreenshotChunk{
			Data:  buffer,
			Index: index,
		})
		index++
	}

	return chunks, nil
}

// GetFullPageHTMLWithShadow captures the complete page HTML including
// serialised shadow DOM content. Shadow roots are emitted as
// <template shadowrootmode="open"> elements within the captured HTML.
//
// Takes ctx (context.Context) which is the chromedp browser context.
//
// Returns string which contains the full HTML source with shadow roots.
// Returns error when the HTML cannot be captured.
func GetFullPageHTMLWithShadow(ctx context.Context) (string, error) {
	js := scripts.MustExecute("capture_dom_with_shadow.js.tmpl", map[string]any{
		"Selector": "html",
	})

	var html string
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &html))
	if err != nil {
		return "", fmt.Errorf("capturing full page HTML with shadow roots: %w", err)
	}
	return html, nil
}

// findScrollContainer detects whether the page scrolls via the window or via
// a nested container (common in Angular/React SPAs that set
// html,body{height:100%;overflow:hidden}).
//
// Takes ctx (context.Context) which is the chromedp browser context.
//
// Returns scrollContainerInfo which identifies the scroll container.
// Returns error when the JavaScript evaluation fails.
func findScrollContainer(ctx context.Context) (scrollContainerInfo, error) {
	js := scripts.MustGet("find_scroll_container.js")

	var result map[string]any
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return scrollContainerInfo{}, fmt.Errorf("finding scroll container: %w", err)
	}

	return scrollContainerInfo{
		Selector:     mapGetString(result, "selector"),
		ScrollHeight: mapGetFloat64(result, "scrollHeight"),
		ClientHeight: mapGetFloat64(result, "clientHeight"),
	}, nil
}

// mapGetString safely extracts a string value from a map,
// returning the zero value when the key is absent or the value
// is not a string.
//
// Takes m (map[string]any) which is the map to read from.
// Takes key (string) which is the key to look up.
//
// Returns string which is the value, or empty when absent or
// not a string.
func mapGetString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// mapGetFloat64 safely extracts a float64 value from a map,
// returning 0 when the key is absent or the value is not a
// float64.
//
// Takes m (map[string]any) which is the map to read from.
// Takes key (string) which is the key to look up.
//
// Returns float64 which is the value, or 0 when absent or not
// a float64.
func mapGetFloat64(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

// getPageDimensions returns the current page scroll height and viewport height.
//
// Takes ctx (context.Context) which is the chromedp browser context.
//
// Returns pageDimensions which contains the scroll and viewport heights.
// Returns error when the JavaScript evaluation fails.
func getPageDimensions(ctx context.Context) (pageDimensions, error) {
	js := scripts.MustGet("get_page_dimensions.js")

	var result map[string]any
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return pageDimensions{}, fmt.Errorf("getting page dimensions: %w", err)
	}

	return pageDimensions{
		ScrollHeight:   mapGetFloat64(result, "scrollHeight"),
		ViewportWidth:  mapGetFloat64(result, "viewportWidth"),
		ViewportHeight: mapGetFloat64(result, "viewportHeight"),
	}, nil
}

// scrollTo scrolls either the window or a specific container element to the
// given Y position.
//
// Takes ctx (context.Context) which is the chromedp browser context.
// Takes selector (string) which is the CSS selector of the scroll container,
// or empty string to scroll the window.
// Takes y (float64) which is the target scroll position in pixels.
//
// Returns error when the scroll operation fails.
func scrollTo(ctx context.Context, selector string, y float64) error {
	position := strconv.FormatFloat(y, 'f', 0, 64)

	var js string
	if selector == "" {
		js = scripts.MustExecute("window_scroll_to.js.tmpl", map[string]any{
			"Position": position,
		})
	} else {
		js = scripts.MustExecute("scroll_element_to.js.tmpl", map[string]any{
			"Selector": selector,
			"Position": position,
		})
	}

	err := chromedp.Run(ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("scrolling to position %s: %w", position, err)
	}
	return nil
}

// getScrollDimensions returns the scroll height and client height of either
// the window or a specific container element.
//
// Takes ctx (context.Context) which is the chromedp browser context.
// Takes selector (string) which is the CSS selector of the scroll container,
// or empty string for the window.
//
// Returns scrollHeight (float64) which is the total scrollable
// height.
// Returns clientHeight (float64) which is the visible viewport
// height.
// Returns err (error) when the JavaScript evaluation fails.
func getScrollDimensions(
	ctx context.Context, selector string,
) (scrollHeight, clientHeight float64, err error) {
	var js string
	if selector == "" {
		js = scripts.MustGet("get_scroll_dimensions_window.js")
	} else {
		js = scripts.MustExecute("get_scroll_dimensions_element.js.tmpl", map[string]any{
			"Selector": selector,
		})
	}

	var result map[string]any
	if runErr := chromedp.Run(ctx, chromedp.Evaluate(js, &result)); runErr != nil {
		return 0, 0, fmt.Errorf("getting scroll dimensions: %w", runErr)
	}

	return mapGetFloat64(result, "scrollHeight"),
		mapGetFloat64(result, "clientHeight"),
		nil
}

// runScrollPass performs a single top-to-bottom scroll pass and
// reports whether the page grew.
//
// Takes ctx (context.Context) which is the chromedp browser
// context.
// Takes selector (string) which is the CSS selector of the
// scroll container, or empty string for the window.
// Takes pass (int) which is the zero-based pass index.
// Takes opts (ScrollCaptureOptions) which configures scroll
// timing and limits.
//
// Returns grown (bool) which is true when the page height
// increased during this pass.
// Returns err (error) when a scroll operation fails.
func runScrollPass(
	ctx context.Context,
	selector string,
	pass int,
	opts ScrollCaptureOptions,
) (grown bool, err error) {
	sh, ch, err := getScrollDimensions(ctx, selector)
	if err != nil {
		return false, fmt.Errorf("scroll pass %d: %w", pass, err)
	}

	if ch <= 0 {
		return false, fmt.Errorf("scroll pass %d: invalid client height: %f", pass, ch)
	}

	previousScrollHeight := sh

	currentY := 0.0
	for currentY < sh {
		if scrollErr := scrollTo(ctx, selector, currentY); scrollErr != nil {
			return false, scrollErr
		}
		time.Sleep(opts.ScrollStepDelay)
		waitForVisibleImages(ctx, 5*time.Second)
		currentY += ch
	}

	if scrollErr := scrollTo(ctx, selector, sh); scrollErr != nil {
		return false, scrollErr
	}
	time.Sleep(opts.ScrollStepDelay)
	waitForVisibleImages(ctx, 5*time.Second)

	waitNetworkIdle(ctx, opts.NetworkIdleDuration, opts.NetworkIdleTimeout)

	newSH, _, err := getScrollDimensions(ctx, selector)
	if err != nil {
		return false, fmt.Errorf("re-checking dimensions on pass %d: %w", pass, err)
	}

	return newSH > previousScrollHeight, nil
}

// waitNetworkIdle waits for network activity to cease. This is best-effort;
// timeout errors are ignored since some pages have persistent connections.
//
// Takes ctx (context.Context) which is the chromedp browser context.
// Takes idleDuration (time.Duration) which is how long the network must be
// quiet.
// Takes timeout (time.Duration) which is the maximum time to wait.
func waitNetworkIdle(ctx context.Context, idleDuration, timeout time.Duration) {
	js, err := scripts.Execute("wait_network_idle.js.tmpl", map[string]any{
		"IdleDuration": idleDuration.Milliseconds(),
		"Timeout":      timeout.Milliseconds(),
	})
	if err != nil {
		return
	}

	var result bool
	_ = chromedp.Run(ctx, chromedp.Evaluate(js, &result, chromedp.EvalAsValue))
}

// waitForVisibleImages waits for images currently in or near the viewport to
// finish loading. This is best-effort; timeout errors are ignored.
//
// Takes ctx (context.Context) which is the chromedp browser context.
// Takes timeout (time.Duration) which is the maximum time to wait.
func waitForVisibleImages(ctx context.Context, timeout time.Duration) {
	js, err := scripts.Execute("wait_images_loaded.js.tmpl", map[string]any{
		"Timeout": timeout.Milliseconds(),
	})
	if err != nil {
		return
	}

	var result bool
	_ = chromedp.Run(ctx, chromedp.Evaluate(js, &result, chromedp.EvalAsValue))
}

// captureChunk captures a single screenshot chunk at the given
// Y offset.
//
// Takes ctx (context.Context) which is the chromedp browser
// context.
// Takes p (chunkCaptureParams) which holds the format, quality,
// scale, and dimension settings.
// Takes y (float64) which is the vertical offset in pixels for
// this chunk.
//
// Returns []byte which contains the image data for this chunk.
// Returns error when the screenshot capture fails.
func captureChunk(
	ctx context.Context,
	p chunkCaptureParams,
	y float64,
) ([]byte, error) {
	h := p.chunkH
	if y+h > p.totalHeight {
		h = p.totalHeight - y
	}

	var buffer []byte
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		var captureErr error
		buffer, captureErr = page.CaptureScreenshot().
			WithFormat(p.cdpFormat).
			WithQuality(int64(p.quality)).
			WithCaptureBeyondViewport(true).
			WithClip(&page.Viewport{
				X:      0,
				Y:      y,
				Width:  p.chunkW,
				Height: h,
				Scale:  p.scale,
			}).
			Do(ctx2)
		return captureErr
	}))
	return buffer, err
}

// toCDPFormat converts a ScreenshotFormat to the CDP protocol format enum.
//
// Takes format (ScreenshotFormat) which is the internal screenshot format to
// convert.
//
// Returns page.CaptureScreenshotFormat which is the corresponding CDP protocol
// format, defaulting to PNG for unrecognised values.
func toCDPFormat(format ScreenshotFormat) page.CaptureScreenshotFormat {
	switch format {
	case ScreenshotFormatJPEG:
		return page.CaptureScreenshotFormatJpeg
	case ScreenshotFormatWebP:
		return page.CaptureScreenshotFormatWebp
	default:
		return page.CaptureScreenshotFormatPng
	}
}
