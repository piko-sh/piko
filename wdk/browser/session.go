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
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

const (
	// defaultWaitReadyTimeout is the maximum time WaitReady will wait for an element to
	// appear in the DOM before returning an error.
	defaultWaitReadyTimeout = 30 * time.Second
)

// Session drives a browser tab and reports failures as errors.
//
// Create a Session with OpenSession or Harness.NewSession. The tab lifetime and the
// caller's context both bound each browser operation.
type Session struct {
	// page is the browser page this session drives.
	page *browser_provider_chromedp.IncognitoPage

	// owner is the browser the session started, set only by OpenSession.
	owner *browser_provider_chromedp.Browser

	// closeErr is the result of the first Close.
	closeErr error

	// serverURL is prepended to relative paths in Navigate.
	serverURL string

	// closeOnce makes Close idempotent.
	closeOnce sync.Once
}

// operationContext is a tab context cancelled with the caller's cause, which also reports
// the caller's deadline so polling and waits see the same bound.
type operationContext struct {
	context.Context

	// deadline is the caller's deadline, meaningful when hasDeadline is true.
	deadline time.Time

	// hasDeadline reports whether the caller's context has a deadline.
	hasDeadline bool
}

// Deadline reports the earlier of the caller's and the tab's deadlines.
//
// Returns time.Time which is the deadline.
// Returns bool which is false when neither context has a deadline.
func (c operationContext) Deadline() (time.Time, bool) {
	tabDeadline, tabHasDeadline := c.Context.Deadline()
	switch {
	case !c.hasDeadline:
		return tabDeadline, tabHasDeadline
	case tabHasDeadline && tabDeadline.Before(c.deadline):
		return tabDeadline, true
	default:
		return c.deadline, true
	}
}

// Navigate loads a path on the configured server after checking tab responsiveness.
//
// Takes path (string) which is appended to the server URL.
//
// Returns error when the page is not responsive or navigation fails.
func (s *Session) Navigate(ctx context.Context, path string) error {
	action, release := s.actionCtx(ctx)
	defer release()
	return browser_provider_chromedp.Navigate(action, path)
}

// Stop halts page loading with a two-second timeout.
//
// Wait for the DOM elements you need before stopping background resource loads.
//
// Returns error when the stop command fails or times out.
func (s *Session) Stop(ctx context.Context) error {
	action, release := s.actionCtx(ctx)
	defer release()
	return browser_provider_chromedp.Stop(action)
}

// SetViewport sets the viewport dimensions.
//
// Takes width (int64) which is the viewport width in pixels.
// Takes height (int64) which is the viewport height in pixels.
//
// Returns error when the viewport cannot be set.
func (s *Session) SetViewport(ctx context.Context, width, height int64) error {
	action, release := s.actionCtx(ctx)
	defer release()
	return browser_provider_chromedp.SetViewport(action, width, height)
}

// SetViewportWithScale sets the viewport dimensions and device pixel ratio.
//
// A scale above one increases image resolution while preserving CSS dimensions.
//
// Takes width (int64) which is the CSS viewport width.
// Takes height (int64) which is the CSS viewport height.
// Takes scale (float64) which is the device pixel ratio.
//
// Returns error when the viewport cannot be set.
func (s *Session) SetViewportWithScale(ctx context.Context, width, height int64, scale float64) error {
	operation, release := s.operation(ctx)
	defer release()
	err := chromedp.Run(operation,
		chromedp.EmulateViewport(width, height, chromedp.EmulateScale(scale)),
	)
	if err != nil {
		return fmt.Errorf("setting viewport to %dx%d at scale %.1f: %w", width, height, scale, err)
	}
	return nil
}

// WaitReady waits up to 30 seconds for a matching element to appear in the DOM.
//
// Takes selector (string) which identifies the element to wait for.
//
// Returns error when the wait ends before the element is ready.
func (s *Session) WaitReady(ctx context.Context, selector string) error {
	return s.WaitReadyTimeout(ctx, selector, defaultWaitReadyTimeout)
}

// WaitReadyTimeout waits for an element with a custom timeout.
//
// Takes selector (string) which identifies the element.
// Takes timeout (time.Duration) which is the maximum wait time.
//
// Returns error when the wait ends before the element is ready.
func (s *Session) WaitReadyTimeout(ctx context.Context, selector string, timeout time.Duration) error {
	operation, release := s.operation(ctx)
	defer release()
	waitCtx, cancel := context.WithTimeoutCause(operation, timeout, errWaitReadyTimeout)
	defer cancel()
	return chromedp.Run(waitCtx, chromedp.WaitReady(selector))
}

// Eval executes JavaScript on the page without returning a value.
//
// Takes js (string) which contains the JavaScript code to run.
//
// Returns error when the script fails to execute.
func (s *Session) Eval(ctx context.Context, js string) error {
	action, release := s.actionCtx(ctx)
	defer release()
	return browser_provider_chromedp.Eval(action, "", js)
}

// EvalReturn executes JavaScript and returns the result.
//
// Takes js (string) which contains the JavaScript code to run.
//
// Returns any which is the result of the JavaScript execution.
// Returns error when the script fails to execute.
func (s *Session) EvalReturn(ctx context.Context, js string) (any, error) {
	operation, release := s.operation(ctx)
	defer release()
	var result any
	if err := chromedp.Run(operation, chromedp.Evaluate(js, &result)); err != nil {
		return nil, fmt.Errorf("evaluating script: %w", err)
	}
	return result, nil
}

// ScreenshotViewport captures a PNG screenshot of the current viewport with
// speed-optimised encoding.
//
// Returns []byte which contains the PNG image data.
// Returns error when the screenshot cannot be captured.
func (s *Session) ScreenshotViewport(ctx context.Context) ([]byte, error) {
	action, release := s.actionCtx(ctx)
	defer release()
	opts := browser_provider_chromedp.DefaultScreenshotOptions()
	opts.OptimiseForSpeed = true
	return browser_provider_chromedp.ScreenshotWithFormat(action, opts)
}

// Close releases the browser tab and its resources.
//
// A session created by OpenSession also closes the browser it owns. Repeated calls wait
// for the first call and return its result.
//
// Returns error when closing the tab fails.
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		if s.page != nil {
			s.closeErr = s.page.Close()
		}
		if s.owner != nil {
			s.owner.Close()
		}
	})
	return s.closeErr
}

// operation derives a browser operation context from the tab context.
//
// The result uses the earlier deadline and ends when either context is cancelled.
//
// Returns context.Context which runs the operation on this tab.
// Returns context.CancelFunc which releases the context once the operation ends.
func (s *Session) operation(ctx context.Context) (context.Context, context.CancelFunc) {
	merged, cancel := context.WithCancelCause(s.page.Ctx)
	if ctx.Err() != nil {
		cancel(context.Cause(ctx))
	}
	stop := context.AfterFunc(ctx, func() { cancel(context.Cause(ctx)) })
	deadline, hasDeadline := ctx.Deadline()
	operation := operationContext{Context: merged, deadline: deadline, hasDeadline: hasDeadline}
	return operation, func() {
		stop()
		cancel(nil)
	}
}

// actionCtx builds the ActionContext used by internal provider functions.
//
// Returns *browser_provider_chromedp.ActionContext which holds the operation context and
// server URL.
// Returns context.CancelFunc which releases the operation context.
func (s *Session) actionCtx(ctx context.Context) (*browser_provider_chromedp.ActionContext, context.CancelFunc) {
	operation, release := s.operation(ctx)
	return &browser_provider_chromedp.ActionContext{
		Ctx:            operation,
		ServerURL:      s.serverURL,
		SrcSandbox:     nil,
		SandboxFactory: nil,
		PageHelper:     nil,
	}, release
}
