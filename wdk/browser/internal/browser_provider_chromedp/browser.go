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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// maxPageCreationRetries is the number of times to retry creating an incognito
// page if the page fails health checks.
const maxPageCreationRetries = 3

// Browser wraps a chromedp browser instance and manages its lifecycle.
// It implements io.Closer.
type Browser struct {
	// allocatorCtx is the context for the browser allocator.
	allocatorCtx context.Context

	// allocatorCancel stops the browser allocator when called; nil if not set.
	allocatorCancel context.CancelFunc

	// browserCtx is the parent context for all browser operations.
	browserCtx context.Context

	// browserCancel cancels the browser context when Close is called.
	browserCancel context.CancelFunc

	// userDataDir is the temporary directory for the browser profile. Managed
	// explicitly because chromedp's context-based cleanup is unreliable.
	userDataDir string

	// headless indicates whether the browser runs without a visible window.
	headless bool

	// mu guards access to browser state during page creation and cleanup.
	mu sync.Mutex
}

// ChromeFlag is a key-value pair passed as a Chrome command-line flag.
type ChromeFlag struct {
	// Value is the flag value passed to Chrome.
	Value any

	// Name is the flag name without the leading dashes.
	Name string
}

// BrowserOptions holds settings for creating a browser instance.
type BrowserOptions struct {
	// ChromeFlags overrides the default Chrome flags when non-nil, with each
	// flag passed to Chrome via --name=value; when nil, DefaultChromeFlags is
	// used.
	ChromeFlags []ChromeFlag

	// Headless controls whether the browser runs without a visible window.
	Headless bool

	// IgnoreCertErrors makes the browser accept any TLS certificate, including
	// self-signed ones. Use this for testing servers with self-signed certs.
	IgnoreCertErrors bool
}

// DefaultChromeFlags returns the Chrome command-line flags applied on top of
// chromedp.DefaultExecAllocatorOptions. Only flags NOT already covered by the
// chromedp defaults are listed here.
//
// Returns []ChromeFlag which contains the default set of extra Chrome flags.
func DefaultChromeFlags() []ChromeFlag {
	return []ChromeFlag{
		{Name: "disable-gpu", Value: true},
		{Name: "no-sandbox", Value: true},
		{Name: "mute-audio", Value: true},
		{Name: "disable-component-update", Value: true},
		{Name: "disable-domain-reliability", Value: true},
	}
}

// IncognitoPage wraps a page with its incognito browser context for proper
// cleanup. It implements io.Closer.
type IncognitoPage struct {
	// Ctx is the browser context for this incognito page.
	Ctx context.Context

	// Cancel stops the page context when called; nil means no cleanup is needed.
	Cancel context.CancelFunc

	// browser references the parent Browser instance for context cleanup.
	browser *Browser

	// releaseFunction releases the pool semaphore slot. Set by BrowserPool when
	// MaxConcurrentPages is configured; nil for pages created directly.
	releaseFunction func()

	// browserContextID is the CDP context identifier; empty after CloseContext.
	browserContextID cdp.BrowserContextID

	// releaseOnce ensures the pool semaphore slot is released exactly once.
	releaseOnce sync.Once
}

// Close closes both the page and its incognito context. The pool
// semaphore slot is released exactly once via sync.Once when created
// by a BrowserPool with MaxPages.
//
// Returns error when the context cannot be closed.
func (ip *IncognitoPage) Close() error {
	var errs []error

	if ip.Cancel != nil {
		ip.Cancel()
	}

	if err := ip.CloseContext(); err != nil {
		errs = append(errs, err)
	}

	if ip.releaseFunction != nil {
		ip.releaseOnce.Do(ip.releaseFunction)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// CloseContext closes just the incognito browser context (not the page).
// Use this when the page is closed separately (e.g., via PageHelper.Close).
//
// Returns error when disposing the browser context fails.
func (ip *IncognitoPage) CloseContext() error {
	if ip.browserContextID != "" && ip.browser != nil {
		timedCtx, cancel := context.WithTimeoutCause(
			ip.browser.browserCtx, DefaultActionTimeout,
			fmt.Errorf("browser CloseContext exceeded %s timeout", DefaultActionTimeout),
		)
		defer cancel()

		err := chromedp.Run(timedCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return target.DisposeBrowserContext(ip.browserContextID).Do(ctx)
			}),
		)
		if err != nil {
			return fmt.Errorf("closing incognito context: %w", err)
		}
		ip.browserContextID = ""
	}
	return nil
}

// NewBrowser creates a new browser instance.
//
// Takes opts (BrowserOptions) which specifies the browser settings.
//
// Returns *Browser which is the configured browser ready for use.
// Returns error when the browser fails to start.
func NewBrowser(opts BrowserOptions) (*Browser, error) {
	userDataDir, err := os.MkdirTemp("", "piko-chromedp-*")
	if err != nil {
		return nil, fmt.Errorf("creating browser temp directory: %w", err)
	}

	headlessValue := func() any {
		if opts.Headless {
			return "new"
		}
		return false
	}()

	chromeFlags := opts.ChromeFlags
	if chromeFlags == nil {
		chromeFlags = DefaultChromeFlags()
	}

	allocatorOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(userDataDir),
		chromedp.Flag("headless", headlessValue),
	)
	for _, flag := range chromeFlags {
		allocatorOpts = append(allocatorOpts, chromedp.Flag(flag.Name, flag.Value))
	}

	if opts.IgnoreCertErrors {
		allocatorOpts = append(allocatorOpts, chromedp.Flag("ignore-certificate-errors", true))
	}

	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(context.Background(), allocatorOpts...)

	browserCtx, browserCancel := chromedp.NewContext(allocatorCtx)

	if err := chromedp.Run(browserCtx); err != nil {
		allocatorCancel()
		_ = os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("starting browser: %w", err)
	}

	return &Browser{
		allocatorCtx:    allocatorCtx,
		allocatorCancel: allocatorCancel,
		browserCtx:      browserCtx,
		browserCancel:   browserCancel,
		userDataDir:     userDataDir,
		headless:        opts.Headless,
		mu:              sync.Mutex{},
	}, nil
}

// NewIncognitoPage creates a new isolated browser page in an incognito context.
// Each page has its own cookies, storage, and cache, making it safe for
// parallel tests.
//
// Returns *IncognitoPage which provides an isolated browsing context.
// Returns error when page creation fails after retrying with backoff.
//
// Safe for concurrent use. Call IncognitoPage.Close() to clean up both the
// page and the browser context.
func (b *Browser) NewIncognitoPage() (*IncognitoPage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var lastErr error
	for attempt := range maxPageCreationRetries {
		ip, err := b.createIncognitoPageOnce()
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100*(attempt+1)) * time.Millisecond)
			continue
		}

		if err := b.verifyPageHealthy(ip.Ctx); err != nil {
			ip.Cancel()
			lastErr = fmt.Errorf("page health check failed (attempt %d): %w", attempt+1, err)
			time.Sleep(time.Duration(100*(attempt+1)) * time.Millisecond)
			continue
		}

		return ip, nil
	}

	return nil, fmt.Errorf("failed to create healthy page after %d attempts: %w", maxPageCreationRetries, lastErr)
}

// Close closes the browser and cleans up resources, including
// the temporary user-data directory on disk.
//
// Concurrency: safe for concurrent use. Repeated calls are
// no-ops because the mutex serialises access and nil-guarded
// cancels are idempotent.
func (b *Browser) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.browserCancel != nil {
		b.browserCancel()
	}
	if b.allocatorCancel != nil {
		b.allocatorCancel()
	}
	if b.userDataDir != "" {
		_ = os.RemoveAll(b.userDataDir)
		b.userDataDir = ""
	}
}

// UserDataDir returns the path to the browser's temporary user-data directory.
// Callers can use this to identify (and exclude) the directory during cleanup
// of orphaned temp dirs left by killed subprocesses.
//
// Returns string which is the path to the temporary user-data directory.
func (b *Browser) UserDataDir() string {
	return b.userDataDir
}

// BrowserCtx returns the underlying chromedp browser context.
// Use with caution - prefer using the wrapped methods.
//
// Returns context.Context which is the chromedp browser context.
func (b *Browser) BrowserCtx() context.Context {
	return b.browserCtx
}

// createIncognitoPageOnce creates a single page (tab) without retries.
// This creates a new tab context, not a true incognito context, but provides
// sufficient isolation for parallel test execution.
//
// Returns *IncognitoPage which is the new tab ready for use.
// Returns error when the page fails to initialise.
func (b *Browser) createIncognitoPageOnce() (*IncognitoPage, error) {
	pageCtx, pageCancel := chromedp.NewContext(b.browserCtx)

	err := chromedp.Run(pageCtx,
		chromedp.Navigate("about:blank"),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetFocusEmulationEnabled(true).Do(ctx)
		}),
	)
	if err != nil {
		pageCancel()
		return nil, fmt.Errorf("initialising page: %w", err)
	}

	time.Sleep(200 * time.Millisecond)

	return &IncognitoPage{
		Ctx:              pageCtx,
		Cancel:           pageCancel,
		browserContextID: "",
		browser:          b,
	}, nil
}

// verifyPageHealthy checks that the page is responsive to CDP commands.
//
// Returns error when the page fails to respond within the timeout or the
// document ready state is neither complete nor interactive.
func (*Browser) verifyPageHealthy(ctx context.Context) error {
	timedCtx, cancel := context.WithTimeoutCause(
		ctx, 2*time.Second,
		fmt.Errorf("browser verifyPageHealthy exceeded %s timeout", 2*time.Second),
	)
	defer cancel()

	js := scripts.MustGet("document_ready_state.js")
	var readyState string
	err := chromedp.Run(timedCtx,
		chromedp.Evaluate(js, &readyState),
	)
	if err != nil {
		return fmt.Errorf("page eval check failed: %w", err)
	}

	if readyState != "complete" && readyState != "interactive" {
		return fmt.Errorf("unexpected ready state: %s", readyState)
	}

	return nil
}

// ConsoleLog represents a browser console message with its severity level.
type ConsoleLog struct {
	// Time is when the log entry was created.
	Time time.Time

	// Level is the log severity: "log", "warn", "error", "info", "debug", or
	// "trace".
	Level string

	// Message is the text content of the console log entry.
	Message string
}

// PageHelper wraps a chromedp context with additional helper methods for
// E2E testing. It implements io.Closer.
type PageHelper struct {
	// ctx is the chromedp context for browser automation.
	ctx context.Context

	// cancel stops all page operations when called, providing a cause for
	// cancellation diagnostics.
	cancel context.CancelCauseFunc

	// stopChan signals when to stop console capture.
	stopChan chan struct{}

	// consoleLogs stores console messages as strings; protected by consoleMutex.
	consoleLogs []string

	// consoleLogsV2 stores console logs with their severity levels.
	consoleLogsV2 []ConsoleLog

	// consoleMutex guards access to consoleLogs and consoleLogsV2.
	consoleMutex sync.Mutex

	// stopped tracks whether stopChan has been closed.
	stopped bool
}

// NewPageHelper creates a new page helper that sets up console log capture.
//
// Returns *PageHelper which is the configured helper ready for use.
func NewPageHelper(ctx context.Context) *PageHelper {
	helperCtx, cancel := context.WithCancelCause(ctx)
	ph := &PageHelper{
		ctx:           helperCtx,
		cancel:        cancel,
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
		consoleMutex:  sync.Mutex{},
		stopped:       false,
	}
	ph.setupConsoleCapture()
	return ph
}

// Ctx returns the underlying chromedp context.
//
// Returns context.Context which is the chromedp browser context.
func (ph *PageHelper) Ctx() context.Context {
	return ph.ctx
}

// ConsoleLogs returns a copy of the captured console logs.
//
// Returns []string which contains a copy of all logged messages.
//
// Safe for concurrent use.
func (ph *PageHelper) ConsoleLogs() []string {
	ph.consoleMutex.Lock()
	defer ph.consoleMutex.Unlock()

	logs := make([]string, len(ph.consoleLogs))
	copy(logs, ph.consoleLogs)
	return logs
}

// ClearConsoleLogs clears the console log buffer.
//
// Safe for concurrent use. Uses mutex to protect internal state.
func (ph *PageHelper) ClearConsoleLogs() {
	js := scripts.MustGet("console_clear.js")
	clearContext, cancel := context.WithTimeout(ph.ctx, 2*time.Second)
	defer cancel()
	_ = chromedp.Run(clearContext, chromedp.Evaluate(js, nil))

	time.Sleep(50 * time.Millisecond)

	ph.consoleMutex.Lock()
	ph.consoleLogs = []string{}
	ph.consoleLogsV2 = []ConsoleLog{}
	ph.consoleMutex.Unlock()
}

// ConsoleLogsWithLevel returns a copy of the captured console logs with their
// levels.
//
// Returns []ConsoleLog which contains a snapshot of all captured logs.
//
// Safe for concurrent use. The returned slice is a copy, so callers may
// modify it without affecting the internal state.
func (ph *PageHelper) ConsoleLogsWithLevel() []ConsoleLog {
	ph.consoleMutex.Lock()
	defer ph.consoleMutex.Unlock()

	logs := make([]ConsoleLog, len(ph.consoleLogsV2))
	copy(logs, ph.consoleLogsV2)
	return logs
}

// ConsoleLogsByLevel returns console logs filtered by level.
//
// Takes level (string) which specifies the log level to filter by.
//
// Returns []ConsoleLog which contains all logs matching the given level.
//
// Safe for concurrent use. Guards access with a mutex.
func (ph *PageHelper) ConsoleLogsByLevel(level string) []ConsoleLog {
	ph.consoleMutex.Lock()
	defer ph.consoleMutex.Unlock()

	var filtered []ConsoleLog
	for _, log := range ph.consoleLogsV2 {
		if log.Level == level {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// HasConsoleErrors returns true if any error-level messages were logged.
//
// Returns bool which is true when at least one console log has error level.
//
// Safe for concurrent use; protects access with a mutex.
func (ph *PageHelper) HasConsoleErrors() bool {
	ph.consoleMutex.Lock()
	defer ph.consoleMutex.Unlock()

	for _, log := range ph.consoleLogsV2 {
		if log.Level == "error" {
			return true
		}
	}
	return false
}

// ConsoleErrors returns all error-level console messages.
//
// Returns []ConsoleLog which contains all console entries logged at error
// level.
func (ph *PageHelper) ConsoleErrors() []ConsoleLog {
	return ph.ConsoleLogsByLevel("error")
}

// Close closes the page, handling beforeunload dialogues and
// signalling the console capture task to stop.
//
// Concurrency: safe for concurrent use. The stop channel
// is closed under consoleMutex to prevent double-close.
func (ph *PageHelper) Close() {
	ph.consoleMutex.Lock()
	if !ph.stopped {
		close(ph.stopChan)
		ph.stopped = true
	}
	ph.consoleMutex.Unlock()

	ph.cancel(errors.New("page helper closed"))
}

// Navigate navigates to a URL and waits for the page to stabilise.
//
// Takes url (string) which specifies the target URL to navigate to.
//
// Returns error when navigation fails or the page does not become ready.
func (ph *PageHelper) Navigate(url string) error {
	timedCtx, cancel := context.WithTimeoutCause(ph.ctx, 15*time.Second, fmt.Errorf("browser Navigate exceeded %s timeout", 15*time.Second))
	defer cancel()

	err := chromedp.Run(timedCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		return fmt.Errorf("navigating to %s: %w", url, err)
	}

	_ = WaitStable(timedCtx, 500*time.Millisecond)

	return nil
}

// setupConsoleCapture begins capturing console logs from the page.
func (ph *PageHelper) setupConsoleCapture() {
	chromedp.ListenTarget(ph.ctx, func(ev any) {
		ph.handleConsoleEvent(ev)
	})
}

// handleConsoleEvent processes a console API event.
//
// Takes ev (any) which is the raw console event to process.
func (ph *PageHelper) handleConsoleEvent(ev any) {
	select {
	case <-ph.stopChan:
		return
	default:
	}

	e, ok := ev.(*runtime.EventConsoleAPICalled)
	if !ok {
		return
	}

	message := buildConsoleMessage(e.Args)
	level := mapConsoleLevel(e.Type)
	ph.recordConsoleLog(message, level)
}

// recordConsoleLog records a console log entry under lock.
//
// Takes message (string) which is the log message to record.
// Takes level (string) which is the log severity level.
//
// Safe for concurrent use. Uses consoleMutex to protect access to the log
// slices. Does nothing if the helper has been stopped.
func (ph *PageHelper) recordConsoleLog(message, level string) {
	ph.consoleMutex.Lock()
	defer ph.consoleMutex.Unlock()

	if ph.stopped {
		return
	}

	ph.consoleLogs = append(ph.consoleLogs, message)
	ph.consoleLogsV2 = append(ph.consoleLogsV2, ConsoleLog{
		Time:    time.Now(),
		Level:   level,
		Message: message,
	})
}

// DefaultBrowserOptions returns the default browser options.
//
// Returns BrowserOptions which is configured with headless mode enabled.
func DefaultBrowserOptions() BrowserOptions {
	return BrowserOptions{
		Headless: true,
	}
}

// WaitStable waits for the page DOM to stabilise, meaning no changes occur
// for the given interval.
//
// Takes interval (time.Duration) which sets how long to wait between checks.
//
// Returns error when the context is cancelled or the DOM cannot be read.
func WaitStable(ctx context.Context, interval time.Duration) error {
	var previousHTML string
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadline := time.Now().Add(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var html string
			err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html, chromedp.ByQuery))
			if err != nil {
				return err
			}
			if html == previousHTML {
				return nil
			}
			previousHTML = html

			if time.Now().After(deadline) {
				return nil
			}
		}
	}
}

// buildConsoleMessage converts console arguments to a single message string.
//
// Takes arguments ([]*runtime.RemoteObject) which are the console arguments to
// convert.
//
// Returns string which is the space-separated message built from all arguments.
func buildConsoleMessage(arguments []*runtime.RemoteObject) string {
	parts := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		parts = append(parts, remoteObjectToString(argument))
	}
	return strings.Join(parts, " ")
}

// remoteObjectToString converts a CDP RemoteObject to its string
// representation.
//
// Takes argument (*runtime.RemoteObject) which is the remote object to convert.
//
// Returns string which is the value, description, or formatted representation.
func remoteObjectToString(argument *runtime.RemoteObject) string {
	if len([]byte(argument.Value)) > 0 {
		s := string([]byte(argument.Value))
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
		return s
	}
	if argument.Description != "" {
		return argument.Description
	}
	return fmt.Sprintf("%v", argument)
}

// mapConsoleLevel maps a runtime.APIType to a simple level string.
//
// Takes t (runtime.APIType) which specifies the console API type to map.
//
// Returns string which is the corresponding level name such as "debug",
// "info", "error", "warn", "trace", or "log" for unhandled types.
func mapConsoleLevel(t runtime.APIType) string {
	switch t {
	case runtime.APITypeDebug:
		return "debug"
	case runtime.APITypeInfo:
		return "info"
	case runtime.APITypeError:
		return "error"
	case runtime.APITypeWarning:
		return "warn"
	case runtime.APITypeTrace:
		return "trace"
	default:
		return "log"
	}
}
