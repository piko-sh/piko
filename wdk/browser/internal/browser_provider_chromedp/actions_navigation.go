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
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

var (
	// errNoHistoryEntry is returned when there is no history entry to move to.
	errNoHistoryEntry = errors.New("no history entry in that direction")
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

	js := scripts.MustGet(documentReadyStateScript)
	var readyState string
	err := chromedp.Run(timedCtx, chromedp.Evaluate(js, &readyState))
	if err != nil {
		return fmt.Errorf("page not responsive before navigation (likely CDP stuck): %w", err)
	}

	return navigateOnce(ctx, url)
}

// GoBack navigates back in browser history.
//
// Takes ctx (*ActionContext) which provides the browser context for the navigation.
//
// Returns error when there is no earlier entry, the navigation fails, the page does not
// finish loading in time or ctx ends.
func GoBack(ctx *ActionContext) error {
	if err := navigateHistory(ctx.Ctx, -1, "GoBack"); err != nil {
		return fmt.Errorf("navigating back: %w", err)
	}
	return nil
}

// GoForward navigates forward in browser history.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
//
// Returns error when there is no later entry, the navigation fails, the page does not
// finish loading in time or ctx ends.
func GoForward(ctx *ActionContext) error {
	if err := navigateHistory(ctx.Ctx, 1, "GoForward"); err != nil {
		return fmt.Errorf("navigating forward: %w", err)
	}
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
// Returns error when navigation fails, the page appears empty, or the page becomes
// unresponsive after loading.
func navigateOnce(ctx *ActionContext, url string) error {
	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, navigationTimeout, fmt.Errorf("navigation navigateOnce exceeded %s timeout", navigationTimeout))
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

	_ = WaitStable(timedCtx, stableCheckInterval)

	jsHealth := scripts.MustGet(documentReadyStateScript)
	var readyState string
	err = chromedp.Run(timedCtx, chromedp.Evaluate(jsHealth, &readyState))
	if err != nil {
		return fmt.Errorf("page became unresponsive after navigation: %w", err)
	}

	return nil
}

// navigateHistory moves offset entries through the session history and waits for the
// destination document to load.
//
// A page restored from the back/forward cache may skip the load event, so navigation
// checks the destination's readiness before waiting for the DOM to settle.
//
// Takes offset (int64) which is the signed distance to move, negative for back.
// Takes action (string) which names the action in the timeout cause.
//
// Returns error when there is no entry at the offset, the navigation fails, the page does
// not finish loading within historyNavigationTimeout or ctx ends.
func navigateHistory(ctx context.Context, offset int64, action string) error {
	timedCtx, cancel := context.WithTimeoutCause(ctx, historyNavigationTimeout,
		fmt.Errorf("navigation %s exceeded %s timeout", action, historyNavigationTimeout))
	defer cancel()

	var targetIndex int64
	err := chromedp.Run(timedCtx, chromedp.ActionFunc(func(runCtx context.Context) error {
		current, entries, err := page.GetNavigationHistory().Do(runCtx)
		if err != nil {
			return fmt.Errorf("reading navigation history: %w", err)
		}
		targetIndex = current + offset
		if targetIndex < 0 || targetIndex >= int64(len(entries)) {
			return errNoHistoryEntry
		}
		return page.NavigateToHistoryEntry(entries[targetIndex].ID).Do(runCtx)
	}))
	if err != nil {
		return withContextCause(timedCtx, err)
	}

	if err := waitForHistoryEntry(timedCtx, targetIndex); err != nil {
		return err
	}

	if err := WaitStable(ctx, stableCheckInterval); err != nil {
		return fmt.Errorf("waiting for page to settle: %w", err)
	}
	return nil
}

// waitForHistoryEntry polls until the session history index is targetIndex and the
// document has finished loading.
//
// Takes targetIndex (int64) which is the history index the navigation moves to.
//
// Returns error carrying the cause of ctx ending, and the last lookup error if there was
// one, when the entry is not loaded before ctx ends.
func waitForHistoryEntry(ctx context.Context, targetIndex int64) error {
	readyStateScript := scripts.MustGet(documentReadyStateScript)
	ticker := time.NewTicker(DefaultPollingInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		loaded, err := historyEntryLoaded(ctx, targetIndex, readyStateScript)
		if loaded {
			return nil
		}
		if err != nil {
			lastErr = err
		}

		if err := waitForTick(ctx, ticker); err != nil {
			if lastErr != nil {
				return fmt.Errorf("waiting for history entry %d to load: %w (last error: %w)", targetIndex, err, lastErr)
			}
			return fmt.Errorf("waiting for history entry %d to load: %w", targetIndex, err)
		}
	}
}

// historyEntryLoaded reports whether the current history entry is targetIndex and its
// document has finished loading.
//
// Takes targetIndex (int64) which is the history index expected to be current.
// Takes readyStateScript (string) which reads document.readyState.
//
// Returns bool which is true when the entry is current and its document is complete.
// Returns error when the history or the ready state cannot be read.
func historyEntryLoaded(ctx context.Context, targetIndex int64, readyStateScript string) (bool, error) {
	var readyState string
	var current int64
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		var historyErr error
		current, _, historyErr = page.GetNavigationHistory().Do(runCtx)
		return historyErr
	}))
	if err != nil {
		return false, fmt.Errorf("reading navigation history: %w", err)
	}
	if current != targetIndex {
		return false, nil
	}

	if err := chromedp.Run(ctx, chromedp.Evaluate(readyStateScript, &readyState)); err != nil {
		return false, fmt.Errorf("reading document ready state: %w", err)
	}
	return readyState == "complete", nil
}

// verifyPageHasContent checks that the page has a body element with content.
//
// Returns error when the page is unresponsive, the ready state is unexpected, no body
// element exists, or the body is empty.
func verifyPageHasContent(ctx context.Context) error {
	timedCtx, cancel := context.WithTimeoutCause(ctx, 2*time.Second, fmt.Errorf("navigation verifyPageHasContent exceeded %s timeout", 2*time.Second))
	defer cancel()

	js := scripts.MustGet(documentReadyStateScript)
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
	timedCtx, cancel := context.WithTimeoutCause(ctx, domQueryTimeout, fmt.Errorf("navigation getCurrentURL exceeded %s timeout", domQueryTimeout))
	defer cancel()

	var url string
	err := chromedp.Run(timedCtx, chromedp.Location(&url))
	if err != nil {
		return "<unknown>"
	}
	return url
}
