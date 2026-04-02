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
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// Navigate navigates to a path, prepending the server URL.
//
// Takes ctx (*ActionContext) which provides the server URL and context.
// Takes path (string) which specifies the path to navigate to.
//
// Returns error when the page is not responsive or navigation fails.
func Navigate(ctx *ActionContext, path string) error {
	url := ctx.ServerURL + path

	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, 2*time.Second, fmt.Errorf("navigation Navigate exceeded %s timeout", 2*time.Second))
	defer cancel()

	js := scripts.MustGet("document_ready_state.js")
	var readyState string
	err := chromedp.Run(timedCtx, chromedp.Evaluate(js, &readyState))
	if err != nil {
		return fmt.Errorf("page not responsive before navigation (likely CDP stuck): %w", err)
	}

	return navigateOnce(ctx, url)
}

// GoBack navigates back in browser history.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// navigation.
//
// Returns error when the navigation fails.
func GoBack(ctx *ActionContext) error {
	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, 5*time.Second, fmt.Errorf("navigation GoBack exceeded %s timeout", 5*time.Second))
	defer cancel()

	err := chromedp.Run(timedCtx, chromedp.NavigateBack())
	if err != nil {
		return fmt.Errorf("navigating back: %w", err)
	}

	_ = WaitStable(timedCtx, 500*time.Millisecond)
	return nil
}

// GoForward navigates forward in browser history.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
//
// Returns error when navigation fails.
func GoForward(ctx *ActionContext) error {
	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, 5*time.Second, fmt.Errorf("navigation GoForward exceeded %s timeout", 5*time.Second))
	defer cancel()

	err := chromedp.Run(timedCtx, chromedp.NavigateForward())
	if err != nil {
		return fmt.Errorf("navigating forward: %w", err)
	}

	_ = WaitStable(timedCtx, 500*time.Millisecond)
	return nil
}

// GetTitle returns the current page title.
//
// Takes ctx (*ActionContext) which provides the browser context for the query.
//
// Returns string which is the current page title.
// Returns error when the title cannot be retrieved.
func GetTitle(ctx *ActionContext) (string, error) {
	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, 2*time.Second, fmt.Errorf("navigation GetTitle exceeded %s timeout", 2*time.Second))
	defer cancel()

	var title string
	err := chromedp.Run(timedCtx, chromedp.Title(&title))
	if err != nil {
		return "", fmt.Errorf("getting page title: %w", err)
	}
	return title, nil
}

// GetURL returns the current page URL.
//
// Takes ctx (*ActionContext) which provides the browser context for the query.
//
// Returns string which is the current URL of the page.
// Returns error when the URL cannot be retrieved.
func GetURL(ctx *ActionContext) (string, error) {
	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, 2*time.Second, fmt.Errorf("navigation GetURL exceeded %s timeout", 2*time.Second))
	defer cancel()

	var url string
	err := chromedp.Run(timedCtx, chromedp.Location(&url))
	if err != nil {
		return "", fmt.Errorf("getting page URL: %w", err)
	}
	return url, nil
}

// Stop stops the current page from loading.
//
// Takes ctx (*ActionContext) which provides the browser context and timeout.
//
// Returns error when the page load cannot be stopped.
func Stop(ctx *ActionContext) error {
	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, 2*time.Second, fmt.Errorf("navigation Stop exceeded %s timeout", 2*time.Second))
	defer cancel()

	err := chromedp.Run(timedCtx, chromedp.Stop())
	if err != nil {
		return fmt.Errorf("stopping page load: %w", err)
	}
	return nil
}

// navigateOnce performs a single navigation attempt with verification.
//
// Takes ctx (*ActionContext) which provides the browser context and settings.
// Takes url (string) which specifies the target URL to navigate to.
//
// Returns error when navigation fails, the page appears empty, or the page
// becomes unresponsive after loading.
func navigateOnce(ctx *ActionContext, url string) error {
	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, 15*time.Second, fmt.Errorf("navigation navigateOnce exceeded %s timeout", 15*time.Second))
	defer cancel()

	err := chromedp.Run(timedCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		return fmt.Errorf("navigating to %s: %w", url, err)
	}

	if err := verifyPageHasContent(timedCtx); err != nil {
		return fmt.Errorf("page loaded but appears empty at %s: %w", url, err)
	}

	_ = WaitStable(timedCtx, 500*time.Millisecond)

	jsHealth := scripts.MustGet("document_ready_state.js")
	var readyState string
	err = chromedp.Run(timedCtx, chromedp.Evaluate(jsHealth, &readyState))
	if err != nil {
		return fmt.Errorf("page became unresponsive after navigation: %w", err)
	}

	return nil
}

// verifyPageHasContent checks that the page has a body element with content.
//
// Returns error when the page is unresponsive, the ready state is unexpected,
// no body element exists, or the body is empty.
func verifyPageHasContent(ctx context.Context) error {
	timedCtx, cancel := context.WithTimeoutCause(ctx, 2*time.Second, fmt.Errorf("navigation verifyPageHasContent exceeded %s timeout", 2*time.Second))
	defer cancel()

	js := scripts.MustGet("document_ready_state.js")
	var readyState string
	err := chromedp.Run(timedCtx, chromedp.Evaluate(js, &readyState))
	if err != nil {
		return fmt.Errorf("page unresponsive after navigation: %w", err)
	}
	if readyState != "complete" && readyState != "interactive" {
		return fmt.Errorf("unexpected ready state after navigation: %s", readyState)
	}

	var html string
	err = chromedp.Run(timedCtx,
		chromedp.OuterHTML("body", &html, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("no body element: %w", err)
	}
	if len(strings.TrimSpace(html)) < MinBodyLength {
		return fmt.Errorf("body is empty or nearly empty: %q", html)
	}
	return nil
}

// getCurrentURL safely retrieves the current URL of the page.
//
// Returns string which is the page URL, or "<unknown>" if retrieval fails.
func getCurrentURL(ctx context.Context) string {
	timedCtx, cancel := context.WithTimeoutCause(ctx, 500*time.Millisecond, fmt.Errorf("navigation getCurrentURL exceeded %s timeout", 500*time.Millisecond))
	defer cancel()

	var url string
	err := chromedp.Run(timedCtx, chromedp.Location(&url))
	if err != nil {
		return "<unknown>"
	}
	return url
}
