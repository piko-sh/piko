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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

const (
	// sessionMargin is the browser start-up and shutdown allowance added to every session
	// budget.
	sessionMargin = 15 * time.Second

	// privateFileMode is the permission of the receipts and coverage files the worker
	// writes.
	privateFileMode = 0o600
)

var (
	// errSessionBudget marks a browser-session deadline as an infrastructure failure.
	errSessionBudget = fmt.Errorf("%s: browser session: %w", casefile.BrowserBudgetMarker, context.DeadlineExceeded)

	// errRuntimeCoverage marks a failure to start or collect runtime coverage.
	errRuntimeCoverage = errors.New("runtime coverage")

	// errStepBudget is the cause recorded when one browser step runs out of its own time.
	errStepBudget = errors.New("browser step budget exceeded")
)

// browserAction performs one browser operation. A generated step can require several
// ordered operations before its observation.
type browserAction func(context.Context, *browser.Session) error

// browserActions runs ordered browser operations as one step.
type browserActions []browserAction

// run performs the operations in order and stops at the first failure.
//
// Takes session (*browser.Session) which is the browser tab to drive.
//
// Returns error which reports the first failed operation, or nil on success.
func (actions browserActions) run(ctx context.Context, session *browser.Session) error {
	for _, action := range actions {
		if err := action(ctx, session); err != nil {
			return err
		}
	}
	return nil
}

// openBrowser starts a private Chromium session whose start-up and every later step share
// one deadline.
//
// Takes budget (time.Duration) which bounds the whole session, including start-up.
//
// Returns context.Context which carries the session deadline for every later step.
// Returns *browser.Session which is the opened browser tab.
// Returns func() which closes the browser and releases the deadline.
// Returns error which reports a browser start-up failure, or nil on success.
func openBrowser(ctx context.Context, budget time.Duration) (context.Context, *browser.Session, func(), error) {
	ctx, cancel := context.WithTimeoutCause(ctx, budget, errSessionBudget)
	options := browser.DefaultBrowserOptions()
	options.SocketDir = os.Getenv(envBrowserTemporary)
	session, err := browser.OpenSession(ctx, options)
	if err != nil {
		cancel()
		if cause := context.Cause(ctx); errors.Is(cause, errSessionBudget) {
			return nil, nil, nil, fmt.Errorf("%w: %w", cause, err)
		}
		return nil, nil, nil, err
	}
	closeBrowser := func() {
		defer cancel()
		if err := session.Close(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "closing browser:", err)
		}
	}
	return ctx, session, closeBrowser, nil
}

// stepContext bounds one browser step by its own budget within the session's.
//
// Takes budget (time.Duration) which bounds the step.
//
// Returns context.Context which carries the step deadline.
// Returns context.CancelFunc which releases the step deadline.
func stepContext(ctx context.Context, budget time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeoutCause(ctx, budget, errStepBudget)
}

// sessionFailure describes a failed browser step and names the session budget when that
// budget ended the step.
//
// Takes step (string) which names the failed step, or is empty when the receipt records
// the step separately.
// Takes err (error) which is the step's failure.
//
// Returns string which is the receipt's driver error text.
func sessionFailure(ctx context.Context, step string, err error) string {
	detail := err.Error()
	if step != "" {
		detail = step + ": " + detail
	}
	if cause := context.Cause(ctx); errors.Is(cause, errSessionBudget) {
		return cause.Error() + ": " + detail
	}
	return detail
}

// runBrowser performs ordered browser operations as one step.
//
// Takes session (*browser.Session) which is the browser tab to drive.
// Takes actions (...browserAction) which lists the operations in order.
//
// Returns error which reports the first failed operation, or nil on success.
func runBrowser(ctx context.Context, session *browser.Session, actions ...browserAction) error {
	return browserActions(actions).run(ctx, session)
}

// evaluate builds an action that runs a script and decodes its result.
//
// Takes script (string) which is the JavaScript expression.
// Takes out (any) which receives the decoded result, or is nil.
//
// Returns browserAction which runs the script.
func evaluate(script string, out any) browserAction {
	return func(ctx context.Context, session *browser.Session) error { return session.Evaluate(ctx, script, out) }
}

// await builds an action that runs a promise-returning script and decodes its result.
//
// Takes script (string) which is the JavaScript expression.
// Takes out (any) which receives the decoded result, or is nil.
//
// Returns browserAction which awaits the script.
func await(script string, out any) browserAction {
	return func(ctx context.Context, session *browser.Session) error { return session.Await(ctx, script, out) }
}

// navigate builds an action that loads an absolute URL.
//
// Takes address (string) which is the absolute URL.
//
// Returns browserAction which performs the navigation.
func navigate(address string) browserAction {
	return func(ctx context.Context, session *browser.Session) error { return session.NavigateURL(ctx, address) }
}

// waitExpression builds an action that waits for a script expression to become truthy.
//
// Takes expression (string) which is the JavaScript condition.
// Takes timeout (time.Duration) which bounds the wait.
//
// Returns browserAction which performs the wait.
func waitExpression(expression string, timeout time.Duration) browserAction {
	return func(ctx context.Context, session *browser.Session) error {
		return session.WaitForExpression(ctx, expression, timeout)
	}
}

// pressKey builds an action that presses a named key.
//
// Takes key (string) which names the key.
// Takes modifiers (...string) which lists held modifier keys.
//
// Returns browserAction which presses the key.
func pressKey(key string, modifiers ...string) browserAction {
	return func(ctx context.Context, session *browser.Session) error {
		return session.PressKey(ctx, key, modifiers...)
	}
}

// typeKeys builds an action that types text through native key events.
//
// Takes text (string) which is the text to type.
//
// Returns browserAction which types the text.
func typeKeys(text string) browserAction {
	return func(ctx context.Context, session *browser.Session) error { return session.Key(ctx, text) }
}

// clickSelector builds an action that clicks the first element matching a selector.
//
// Takes selector (string) which selects the element.
//
// Returns browserAction which performs the click.
func clickSelector(selector string) browserAction {
	return func(ctx context.Context, session *browser.Session) error { return session.Click(ctx, selector) }
}

// viewport builds an action that resizes the browser viewport.
//
// Takes width (int64) which is the viewport width in CSS pixels.
// Takes height (int64) which is the viewport height in CSS pixels.
//
// Returns browserAction which resizes the viewport.
func viewport(width, height int64) browserAction {
	return func(ctx context.Context, session *browser.Session) error {
		return session.SetViewport(ctx, width, height)
	}
}

// scriptString encodes text as a JavaScript string literal.
//
// Takes text (string) which is the value to encode.
//
// Returns string which is the quoted literal.
// Returns error which reports an encoding failure, or nil on success.
func scriptString(text string) (string, error) {
	encoded, err := json.Marshal(text)
	if err != nil {
		return "", fmt.Errorf("encoding script string: %w", err)
	}
	return string(encoded), nil
}

// writePrivateFile writes data to a harness-chosen file through a root opened on the
// file's directory, so the write cannot leave that directory.
//
// Takes path (string) which is the output file the harness created the directory for.
// Takes data ([]byte) which is the file content.
//
// Returns error which reports an unopenable directory, a write failure or a close
// failure, or nil on success.
func writePrivateFile(path string, data []byte) (err error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, root.Close())
	}()
	return root.WriteFile(filepath.Base(path), data, privateFileMode)
}
