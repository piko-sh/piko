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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

const (
	// expressionPollBackstop lets the context deadline expire first so the wait can report
	// its cause if the in-page poll outlives the connection.
	expressionPollBackstop = 5 * time.Second
)

var (
	// errWaitTimeout is the cause when WaitForExpression runs out of time.
	errWaitTimeout = errors.New("browser expression wait timed out")

	// errWaitReadyTimeout is the cause when WaitReadyTimeout runs out of time.
	errWaitReadyTimeout = errors.New("browser element wait timed out")

	// errSessionHasNoPage is returned when an operation needs a page the session lacks.
	errSessionHasNoPage = errors.New("browser session has no page")
)

// NewTab opens a separate rendering window that shares the browser's storage.
//
// The tab closes independently of its opener. A tab opened from a pooled page waits for
// its own slot and releases it on close.
//
// Returns *Session which is the new tab, closed by the caller.
// Returns error when the session has no page, ctx ends first or the window cannot be
// opened.
func (s *Session) NewTab(ctx context.Context) (*Session, error) {
	if s.page == nil {
		return nil, errSessionHasNoPage
	}
	tabPage, err := s.page.NewSiblingPage(ctx)
	if err != nil {
		return nil, fmt.Errorf("opening browser tab: %w", err)
	}
	return &Session{
		page:      tabPage,
		serverURL: s.serverURL,
		owner:     nil,
		closeErr:  nil,
		closeOnce: sync.Once{},
	}, nil
}

// NavigateURL navigates directly to an absolute URL.
//
// Takes raw (string) which is the absolute URL.
//
// Returns error when navigation fails.
func (s *Session) NavigateURL(ctx context.Context, raw string) error {
	operation, release := s.operation(ctx)
	defer release()
	if err := chromedp.Run(operation, chromedp.Navigate(raw)); err != nil {
		return fmt.Errorf("navigating to %s: %w", raw, err)
	}
	return nil
}

// NavigateResponse loads a URL and reports the destination after redirects.
//
// Takes raw (string) which is the absolute URL.
//
// Returns int which is the final HTTP status, or zero without an HTTP response.
// Returns string which is the location the page landed on.
// Returns error when navigation or reading the location fails.
func (s *Session) NavigateResponse(ctx context.Context, raw string) (int, string, error) {
	operation, release := s.operation(ctx)
	defer release()
	response, err := chromedp.RunResponse(operation, chromedp.Navigate(raw))
	if err != nil {
		return 0, "", fmt.Errorf("navigating to %s: %w", raw, err)
	}
	var landed string
	if err := chromedp.Run(operation, chromedp.Location(&landed)); err != nil {
		return 0, "", fmt.Errorf("reading location after navigating to %s: %w", raw, err)
	}
	if response != nil && (strings.HasPrefix(response.URL, "http://") || strings.HasPrefix(response.URL, "https://")) {
		return int(response.Status), landed, nil
	}
	return 0, landed, nil
}

// Evaluate runs a JavaScript expression and decodes its value into out. Pass nil when the
// value is unneeded.
//
// Takes script (string) which is the expression.
// Takes out (any) which receives the decoded value, or nil.
//
// Returns error when evaluation or decoding fails.
func (s *Session) Evaluate(ctx context.Context, script string, out any) error {
	operation, release := s.operation(ctx)
	defer release()
	if err := chromedp.Run(operation, chromedp.Evaluate(script, out)); err != nil {
		return fmt.Errorf("evaluating script: %w", err)
	}
	return nil
}

// Await evaluates an expression and decodes its resolved promise value into out.
//
// Takes script (string) which is the expression producing a promise.
// Takes out (any) which receives the decoded value, or nil.
//
// Returns error when evaluation, the promise or decoding fails.
func (s *Session) Await(ctx context.Context, script string, out any) error {
	operation, release := s.operation(ctx)
	defer release()
	err := chromedp.Run(operation, chromedp.Evaluate(script, out, func(parameters *runtime.EvaluateParams) *runtime.EvaluateParams {
		return parameters.WithAwaitPromise(true)
	}))
	if err != nil {
		return fmt.Errorf("awaiting script: %w", err)
	}
	return nil
}

// WaitForExpression waits for a JavaScript expression to become truthy.
//
// Takes expr (string) which is the expression to poll.
// Takes timeout (time.Duration) which bounds the wait, or zero for 30 seconds.
//
// Returns error when the wait times out, is cancelled or polling fails.
func (s *Session) WaitForExpression(ctx context.Context, expr string, timeout time.Duration) error {
	if timeout == 0 {
		timeout = defaultWaitReadyTimeout
	}
	operation, release := s.operation(ctx)
	defer release()
	waitCtx, cancel := context.WithTimeoutCause(operation, timeout, errWaitTimeout)
	defer cancel()
	if err := waitCtx.Err(); err != nil {
		return fmt.Errorf("waiting for expression: %w", context.Cause(waitCtx))
	}
	deadline, _ := waitCtx.Deadline()
	pollTimeout := time.Until(deadline) + expressionPollBackstop
	if err := chromedp.Run(waitCtx, chromedp.Poll(expr, nil, chromedp.WithPollingTimeout(pollTimeout))); err != nil {
		return fmt.Errorf("waiting for expression: %w", errors.Join(err, context.Cause(waitCtx)))
	}
	return nil
}

// Click dispatches a pointer click to the matching element.
//
// Takes selector (string) which selects the element.
//
// Returns error when the element cannot be found or clicked.
func (s *Session) Click(ctx context.Context, selector string) error {
	action, release := s.actionCtx(ctx)
	defer release()
	return browser_provider_chromedp.Click(action, selector)
}

// Key types literal UTF-8 text at the current focus, one rune at a time. Printable ASCII
// is sent as key events; any other rune is inserted as text, because Chrome's key map
// assigns special keys to some Unicode letters.
//
// Takes text (string) which is the text to type.
//
// Returns error when a key event or text insertion fails.
func (s *Session) Key(ctx context.Context, text string) error {
	operation, release := s.operation(ctx)
	defer release()
	return chromedp.Run(operation, chromedp.ActionFunc(func(runCtx context.Context) error {
		for _, character := range text {
			var action chromedp.Action = input.InsertText(string(character))
			if character >= ' ' && character <= '~' {
				action = chromedp.KeyEvent(string(character))
			}
			if err := action.Do(runCtx); err != nil {
				return fmt.Errorf("typing %q: %w", character, err)
			}
		}
		return nil
	}))
}

// PressKey presses a named key such as Home or ArrowDown, with optional modifier names
// such as Shift, Control, Alt or Meta.
//
// Takes key (string) which names the key.
// Takes modifiers (...string) which name modifier keys held during the press.
//
// Returns error when the key press fails.
func (s *Session) PressKey(ctx context.Context, key string, modifiers ...string) error {
	if len(modifiers) != 0 {
		key = strings.Join(modifiers, "+") + "+" + key
	}
	action, release := s.actionCtx(ctx)
	defer release()
	return browser_provider_chromedp.Press(action, key)
}

// ScriptOnNewDocument installs a script that runs before application scripts in each new
// document loaded in this tab.
//
// Takes script (string) which is the script source.
//
// Returns error when the script cannot be installed.
func (s *Session) ScriptOnNewDocument(ctx context.Context, script string) error {
	operation, release := s.operation(ctx)
	defer release()
	return chromedp.Run(operation, chromedp.ActionFunc(func(runCtx context.Context) error {
		if _, err := page.AddScriptToEvaluateOnNewDocument(script).Do(runCtx); err != nil {
			return fmt.Errorf("installing document script: %w", err)
		}
		return nil
	}))
}

// OpenSession opens a private browser and page for an already running application.
//
// The caller's context bounds startup and the browser's lifetime. Close also releases the
// page, process and temporary profile while the context is active.
//
// Takes opts (BrowserOptions) which configures the browser.
//
// Returns *Session which owns the browser.
// Returns error when the browser or its page cannot be started.
func OpenSession(ctx context.Context, opts BrowserOptions) (*Session, error) {
	owner, err := browser_provider_chromedp.NewBrowserContext(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("starting browser: %w", err)
	}
	sessionPage, err := owner.NewIncognitoPage()
	if err != nil {
		owner.Close()
		return nil, fmt.Errorf("opening session page: %w", err)
	}
	session := &Session{
		page:      sessionPage,
		serverURL: "",
		owner:     owner,
		closeErr:  nil,
		closeOnce: sync.Once{},
	}
	context.AfterFunc(owner.BrowserCtx(), func() { _ = session.Close() })
	return session, nil
}
