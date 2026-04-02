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
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

const (
	// defaultViewportWidth is the default browser viewport width in pixels.
	defaultViewportWidth = 1920

	// defaultViewportHeight is the default browser viewport height in pixels.
	defaultViewportHeight = 1080

	// defaultScrollStepDelayMs is the default pause between scroll steps in
	// milliseconds.
	defaultScrollStepDelayMs = 500

	// defaultNetworkIdleTimeoutSec is the default network idle timeout in
	// seconds.
	defaultNetworkIdleTimeoutSec = 10

	// defaultMaxScrollPasses is the default number of top-to-bottom scroll
	// passes.
	defaultMaxScrollPasses = 3

	// defaultNavigationTimeoutSec is the default navigation timeout in seconds.
	defaultNavigationTimeoutSec = 30

	// defaultScreenshotQuality is the default JPEG/WebP quality (0-100).
	defaultScreenshotQuality = 90

	// screenshotQualityLossless is the quality threshold at or above which PNG
	// format is used without lossy compression.
	screenshotQualityLossless = 100
)

// ScreenshotChunk represents one tile of a chunked full-page screenshot.
type ScreenshotChunk = browser_provider_chromedp.ScreenshotChunk

// CaptureResult holds the output of capturing a single URL.
type CaptureResult struct {
	// URL is the original URL that was captured.
	URL string

	// HTML is the complete page HTML source.
	HTML string

	// Screenshot is the full-page screenshot data, populated when
	// ChunkScreenshots is false and nil when chunked.
	Screenshot []byte

	// Screenshots holds viewport-sized image chunks, populated when
	// ChunkScreenshots is true and nil when not chunked.
	Screenshots []ScreenshotChunk
}

// CaptureOptions configures how page capture behaves.
type CaptureOptions struct {
	// ScreenshotFormat specifies the image format: "png", "jpeg", or "webp".
	// Default: "jpeg".
	ScreenshotFormat string

	// ScrollStepDelay is the pause between scroll steps, giving lazy-loaded
	// images time to swap from blurry placeholders to sharp versions.
	// Default: 500ms.
	ScrollStepDelay time.Duration

	// NetworkIdleDuration is how long the network must be quiet before
	// considering it idle. Default: 2s.
	NetworkIdleDuration time.Duration

	// NetworkIdleTimeout is the maximum time to wait for network idle after
	// scrolling. Default: 10s.
	NetworkIdleTimeout time.Duration

	// NavigationTimeout is how long to wait for a page to load. Default: 30s.
	NavigationTimeout time.Duration

	// ScreenshotScale is the output scale factor for screenshots, where
	// 1.0 is original resolution and 0.5 is half size.
	// Default: 1.0.
	ScreenshotScale float64

	// ViewportWidth sets the browser viewport width in pixels. Default: 1920.
	ViewportWidth int64

	// ViewportHeight sets the browser viewport height in pixels. Default: 1080.
	ViewportHeight int64

	// MaxScrollPasses is the maximum number of top-to-bottom scroll passes.
	// Default: 3.
	MaxScrollPasses int

	// ScreenshotQuality is the image quality for JPEG/WebP (0-100).
	// Default: 90.
	ScreenshotQuality int

	// Headless controls whether Chrome runs without a visible window.
	// Default: true.
	Headless bool

	// ChunkScreenshots splits the full page into viewport-sized tiles instead
	// of capturing one large image.
	ChunkScreenshots bool

	// IncludeShadowDOM serialises shadow DOM content in the captured HTML
	// using getHTML({serializableShadowRoots: true}).
	IncludeShadowDOM bool
}

// Capturer drives a headless browser to capture page screenshots and HTML.
// It manages a single browser instance and creates isolated pages for each
// URL capture.
type Capturer struct {
	// browser is the headless Chrome instance used for captures.
	browser *browser_provider_chromedp.Browser

	// opts holds the capture configuration for this instance.
	opts CaptureOptions
}

// NewCapturer creates a Capturer and launches a browser instance.
//
// Takes opts (CaptureOptions) which specifies the capture settings.
//
// Returns *Capturer which is ready to capture URLs.
// Returns error when the browser fails to start.
func NewCapturer(opts CaptureOptions) (*Capturer, error) {
	br, err := browser_provider_chromedp.NewBrowser(browser_provider_chromedp.BrowserOptions{
		Headless: opts.Headless,
	})
	if err != nil {
		return nil, fmt.Errorf("creating browser: %w", err)
	}
	return &Capturer{browser: br, opts: opts}, nil
}

// CaptureURL navigates to a URL, scrolls slowly to trigger lazy-loaded
// content, then captures a full-page screenshot and the complete HTML source.
//
// Takes url (string) which is the absolute URL to capture.
//
// Returns *CaptureResult which contains the screenshot and HTML data.
// Returns error when any step of the capture process fails.
func (c *Capturer) CaptureURL(url string) (*CaptureResult, error) {
	pg, err := c.browser.NewIncognitoPage()
	if err != nil {
		return nil, fmt.Errorf("creating page: %w", err)
	}
	defer func() { _ = pg.Close() }()

	if err := c.preparePage(pg, url); err != nil {
		return nil, err
	}

	result := &CaptureResult{URL: url}

	if err := c.captureScreenshot(pg, result); err != nil {
		return nil, err
	}

	if err := c.captureHTML(pg, result); err != nil {
		return nil, err
	}

	return result, nil
}

// Close shuts down the browser and cleans up resources.
func (c *Capturer) Close() {
	if c.browser != nil {
		c.browser.Close()
	}
}

// preparePage sets up the viewport, injects overrides, navigates,
// and scrolls the page to trigger lazy-loaded content.
//
// Takes pg (*browser_provider_chromedp.IncognitoPage) which is the
// browser page to prepare.
// Takes url (string) which is the absolute URL to navigate to.
//
// Returns error when any preparation step fails.
func (c *Capturer) preparePage(
	pg *browser_provider_chromedp.IncognitoPage,
	url string,
) error {
	err := chromedp.Run(pg.Ctx,
		chromedp.EmulateViewport(c.opts.ViewportWidth, c.opts.ViewportHeight),
	)
	if err != nil {
		return fmt.Errorf("setting viewport: %w", err)
	}

	if err := browser_provider_chromedp.InjectObserverOverride(pg.Ctx); err != nil {
		return fmt.Errorf("injecting observer override: %w", err)
	}

	if err := browser_provider_chromedp.NavigateToURL(pg.Ctx, url, c.opts.NavigationTimeout); err != nil {
		return fmt.Errorf("navigating to %s: %w", url, err)
	}

	browser_provider_chromedp.DismissCookieConsent(pg.Ctx)

	err = browser_provider_chromedp.ScrollForLazyContent(pg.Ctx, browser_provider_chromedp.ScrollCaptureOptions{
		ScrollStepDelay:     c.opts.ScrollStepDelay,
		NetworkIdleDuration: c.opts.NetworkIdleDuration,
		NetworkIdleTimeout:  c.opts.NetworkIdleTimeout,
		MaxScrollPasses:     c.opts.MaxScrollPasses,
	})
	if err != nil {
		return fmt.Errorf("scrolling for lazy content: %w", err)
	}

	if err := browser_provider_chromedp.UnwrapScrollContainer(pg.Ctx); err != nil {
		return fmt.Errorf("unwrapping scroll container: %w", err)
	}

	_ = browser_provider_chromedp.ForceLoadAllImages(pg.Ctx, defaultNavigationTimeoutSec*time.Second)

	return nil
}

// captureScreenshot takes a screenshot (chunked or whole) and
// stores it in the result.
//
// Takes pg (*browser_provider_chromedp.IncognitoPage) which is the
// browser page to screenshot.
// Takes result (*CaptureResult) which receives the screenshot
// data.
//
// Returns error when the screenshot cannot be captured.
func (c *Capturer) captureScreenshot(
	pg *browser_provider_chromedp.IncognitoPage,
	result *CaptureResult,
) error {
	format := parseScreenshotFormat(c.opts.ScreenshotFormat)

	if c.opts.ChunkScreenshots {
		chunks, chunkErr := browser_provider_chromedp.FullPageScreenshotChunks(
			pg.Ctx,
			c.opts.ViewportWidth,
			c.opts.ViewportHeight,
			browser_provider_chromedp.ChunkScreenshotOptions{
				Format:  format,
				Quality: c.opts.ScreenshotQuality,
				Scale:   c.opts.ScreenshotScale,
			},
		)
		if chunkErr != nil {
			return fmt.Errorf("capturing screenshot chunks: %w", chunkErr)
		}
		result.Screenshots = chunks
		return nil
	}

	needsFormat := format != browser_provider_chromedp.ScreenshotFormatPNG ||
		c.opts.ScreenshotQuality < screenshotQualityLossless
	if needsFormat {
		screenshot, fmtErr := browser_provider_chromedp.FullPageScreenshotWithFormat(
			pg.Ctx, format, c.opts.ScreenshotQuality,
		)
		if fmtErr != nil {
			return fmt.Errorf("capturing screenshot: %w", fmtErr)
		}
		result.Screenshot = screenshot
		return nil
	}

	screenshot, ssErr := browser_provider_chromedp.FullPageScreenshot(pg.Ctx)
	if ssErr != nil {
		return fmt.Errorf("capturing screenshot: %w", ssErr)
	}
	result.Screenshot = screenshot
	return nil
}

// captureHTML retrieves the page HTML and stores it in the
// result.
//
// Takes pg (*browser_provider_chromedp.IncognitoPage) which is the
// browser page to capture HTML from.
// Takes result (*CaptureResult) which receives the HTML source.
//
// Returns error when the HTML cannot be captured.
func (c *Capturer) captureHTML(
	pg *browser_provider_chromedp.IncognitoPage,
	result *CaptureResult,
) error {
	var err error
	if c.opts.IncludeShadowDOM {
		result.HTML, err = browser_provider_chromedp.GetFullPageHTMLWithShadow(pg.Ctx)
	} else {
		result.HTML, err = browser_provider_chromedp.GetFullPageHTML(pg.Ctx)
	}
	if err != nil {
		return fmt.Errorf("capturing HTML: %w", err)
	}
	return nil
}

// DefaultCaptureOptions returns sensible defaults for page capture.
//
// Returns CaptureOptions which is configured with 1920x1080 viewport, 500ms
// scroll delay, 30s navigation timeout, and headless mode enabled.
func DefaultCaptureOptions() CaptureOptions {
	return CaptureOptions{
		ViewportWidth:       defaultViewportWidth,
		ViewportHeight:      defaultViewportHeight,
		ScrollStepDelay:     defaultScrollStepDelayMs * time.Millisecond,
		NetworkIdleDuration: 2 * time.Second,
		NetworkIdleTimeout:  defaultNetworkIdleTimeoutSec * time.Second,
		MaxScrollPasses:     defaultMaxScrollPasses,
		NavigationTimeout:   defaultNavigationTimeoutSec * time.Second,
		Headless:            true,
		ScreenshotFormat:    "jpeg",
		ScreenshotQuality:   defaultScreenshotQuality,
		ScreenshotScale:     1.0,
	}
}

// parseScreenshotFormat converts a format string to the internal format type.
//
// Takes format (string) which is the image format name such as "jpeg", "jpg",
// "webp", or "png".
//
// Returns browser_provider_chromedp.ScreenshotFormat which is the
// corresponding internal format constant, defaulting to PNG for unrecognised
// values.
func parseScreenshotFormat(format string) browser_provider_chromedp.ScreenshotFormat {
	switch format {
	case "jpeg", "jpg":
		return browser_provider_chromedp.ScreenshotFormatJPEG
	case "webp":
		return browser_provider_chromedp.ScreenshotFormatWebP
	default:
		return browser_provider_chromedp.ScreenshotFormatPNG
	}
}
