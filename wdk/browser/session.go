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
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

// defaultWaitReadyTimeout is the maximum time WaitReady will wait for an
// element to appear in the DOM before returning an error.
const defaultWaitReadyTimeout = 30 * time.Second

// Session is a browser tab for standalone (non-test) automation. Unlike Page,
// it returns errors instead of calling testing.TB.Fatalf, making it usable
// from main() and CLI tools.
//
// Create a Session via Harness.NewSession.
type Session struct {
	// page is the underlying incognito browser page.
	page *browser_provider_chromedp.IncognitoPage

	// serverURL is prepended to relative paths in Navigate.
	serverURL string
}

// Navigate loads a path on the server. The path is appended to the server URL
// configured on the harness. A responsiveness pre-flight check is performed
// before navigation.
//
// Takes path (string) which is appended to the server URL.
//
// Returns error when the page is not responsive or navigation fails.
func (s *Session) Navigate(path string) error {
	return browser_provider_chromedp.Navigate(s.actionCtx(), path)
}

// Stop halts any in-progress page loading with a 2-second timeout. Call this
// after the DOM elements you need are ready to prevent background resource
// fetches from blocking subsequent CDP calls.
//
// Returns error when the stop command fails or times out.
func (s *Session) Stop() error {
	return browser_provider_chromedp.Stop(s.actionCtx())
}

// SetViewport sets the viewport dimensions.
//
// Takes width (int64) which is the viewport width in pixels.
// Takes height (int64) which is the viewport height in pixels.
//
// Returns error when the viewport cannot be set.
func (s *Session) SetViewport(width, height int64) error {
	return browser_provider_chromedp.SetViewport(s.actionCtx(), width, height)
}

// SetViewportWithScale sets the viewport dimensions and device pixel ratio.
// Use scale > 1 to render at higher resolution while keeping CSS layout at
// the given width/height.
//
// Takes width (int64) which is the CSS viewport width.
// Takes height (int64) which is the CSS viewport height.
// Takes scale (float64) which is the device pixel ratio.
//
// Returns error when the viewport cannot be set.
func (s *Session) SetViewportWithScale(width, height int64, scale float64) error {
	err := chromedp.Run(s.actionCtx().Ctx,
		chromedp.EmulateViewport(width, height, chromedp.EmulateScale(scale)),
	)
	if err != nil {
		return fmt.Errorf("setting viewport to %dx%d at scale %.1f: %w", width, height, scale, err)
	}
	return nil
}

// WaitReady waits for an element matching the selector to be present in the
// DOM, with a default timeout of 30 seconds.
//
// Takes selector (string) which identifies the element to wait for.
//
// Returns error when the element is not found within the timeout.
func (s *Session) WaitReady(selector string) error {
	return s.WaitReadyTimeout(selector, defaultWaitReadyTimeout)
}

// WaitReadyTimeout waits for an element with a custom timeout.
//
// Takes selector (string) which identifies the element.
// Takes timeout (time.Duration) which is the maximum wait time.
//
// Returns error when the element is not found within the timeout.
func (s *Session) WaitReadyTimeout(selector string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(s.actionCtx().Ctx, timeout)
	defer cancel()
	return chromedp.Run(ctx, chromedp.WaitReady(selector))
}

// Eval executes JavaScript on the page without returning a value.
//
// Takes js (string) which contains the JavaScript code to run.
//
// Returns error when the script fails to execute.
func (s *Session) Eval(js string) error {
	return browser_provider_chromedp.Eval(s.actionCtx(), "", js)
}

// EvalReturn executes JavaScript and returns the result.
//
// Takes js (string) which contains the JavaScript code to run.
//
// Returns any which is the result of the JavaScript execution.
// Returns error when the script fails to execute.
func (s *Session) EvalReturn(js string) (any, error) {
	var result any
	err := chromedp.Run(s.actionCtx().Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("evaluating script: %w", err)
	}
	return result, nil
}

// ScreenshotViewport captures a PNG screenshot of the current viewport with
// speed-optimised encoding.
//
// Returns []byte which contains the PNG image data.
// Returns error when the screenshot cannot be captured.
func (s *Session) ScreenshotViewport() ([]byte, error) {
	opts := browser_provider_chromedp.DefaultScreenshotOptions()
	opts.OptimiseForSpeed = true
	return browser_provider_chromedp.ScreenshotWithFormat(s.actionCtx(), opts)
}

// Close releases the browser tab and its resources.
//
// Returns error when the browser context cannot be disposed.
func (s *Session) Close() error {
	if s.page != nil {
		return s.page.Close()
	}
	return nil
}

// actionCtx builds the ActionContext used by internal provider functions.
//
// Returns *ActionContext with only Ctx and ServerURL populated; sandbox and
// PageHelper fields are left nil.
func (s *Session) actionCtx() *browser_provider_chromedp.ActionContext {
	return &browser_provider_chromedp.ActionContext{
		Ctx:       s.page.Ctx,
		ServerURL: s.serverURL,
	}
}
