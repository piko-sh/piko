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
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// DialogType represents the kind of JavaScript dialog shown in a browser.
type DialogType string

const (
	// DialogTypeAlert is the dialog type for alert() dialogues.
	DialogTypeAlert DialogType = "alert"

	// DialogTypeConfirm is a dialog type that asks the user to confirm an action.
	DialogTypeConfirm DialogType = "confirm"

	// DialogTypePrompt represents a prompt() dialog.
	DialogTypePrompt DialogType = "prompt"

	// DialogTypeBeforeunload represents a beforeunload dialog.
	DialogTypeBeforeunload DialogType = "beforeunload"

	// dialogChannelBufferSize is the buffer size for the dialog channel.
	dialogChannelBufferSize = 10
)

// DialogInfo holds details about a JavaScript dialog event.
type DialogInfo struct {
	// Type specifies the kind of dialog to display.
	Type DialogType

	// Message is the text content displayed in the dialog.
	Message string

	// URL is the web address for the dialog.
	URL string
}

// DialogHandler handles JavaScript dialog boxes in the browser.
type DialogHandler struct {
	// lastDialog stores the most recently recorded dialog; nil if none has
	// occurred or has been cleared.
	lastDialog *DialogInfo

	// dialogChan receives dialog events for WaitForDialog consumers.
	dialogChan chan DialogInfo

	// stopChan signals when to stop the dialog handler; closed by Disable.
	stopChan chan struct{}

	// autoText is the text to enter in JavaScript prompt dialogues.
	autoText string

	// mu guards concurrent access to the dialog handler state.
	mu sync.RWMutex

	// wg coordinates goroutine completion for dialog handling.
	wg sync.WaitGroup

	// enabled indicates whether dialog handling is active.
	enabled bool

	// autoAccept indicates whether dialogues should be accepted or dismissed.
	autoAccept bool

	// closed tracks whether the stop channel has been closed.
	closed bool
}

// NewDialogHandler creates a new dialog handler.
//
// Returns *DialogHandler which is ready to use but not yet enabled.
func NewDialogHandler() *DialogHandler {
	return &DialogHandler{
		lastDialog: nil,
		dialogChan: make(chan DialogInfo, dialogChannelBufferSize),
		stopChan:   make(chan struct{}),
		autoText:   "",
		mu:         sync.RWMutex{},
		wg:         sync.WaitGroup{},
		enabled:    false,
		autoAccept: false,
		closed:     false,
	}
}

// Enable enables automatic dialog handling.
//
// Takes ctx (*ActionContext) which provides the Chrome DevTools context.
// Takes autoAccept (bool) which when true accepts dialogues automatically;
// otherwise they are dismissed.
//
// Returns error when enabling fails.
//
// Safe for concurrent use. Registers a target listener that handles dialog
// events in the background until the context is cancelled.
func (dh *DialogHandler) Enable(ctx *ActionContext, autoAccept bool) error {
	dh.mu.Lock()
	defer dh.mu.Unlock()

	if dh.enabled {
		return nil
	}

	dh.autoAccept = autoAccept
	dh.enabled = true

	chromedp.ListenTarget(ctx.Ctx, func(ev any) {
		dh.handleTargetEvent(ctx, ev)
	})

	return nil
}

// Disable disables automatic dialog handling.
//
// Safe for concurrent use. Waits up to two seconds for any in-flight
// goroutines to complete before returning.
func (dh *DialogHandler) Disable() {
	dh.mu.Lock()
	if dh.enabled && !dh.closed {
		close(dh.stopChan)
		dh.closed = true
	}
	dh.enabled = false
	dh.mu.Unlock()

	done := make(chan struct{})
	go func() {
		dh.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

// SetPromptText sets the text to enter in prompt dialogues.
//
// Takes text (string) which specifies the response text for prompts.
//
// Safe for concurrent use.
func (dh *DialogHandler) SetPromptText(text string) {
	dh.mu.Lock()
	defer dh.mu.Unlock()
	dh.autoText = text
}

// GetLastDialog returns information about the last dialog that appeared.
//
// Returns *DialogInfo which contains details of the most recent dialog, or nil
// if no dialog has appeared.
//
// Safe for concurrent use.
func (dh *DialogHandler) GetLastDialog() *DialogInfo {
	dh.mu.RLock()
	defer dh.mu.RUnlock()
	return dh.lastDialog
}

// ClearLastDialog clears the last dialog information.
//
// Safe for concurrent use.
func (dh *DialogHandler) ClearLastDialog() {
	dh.mu.Lock()
	defer dh.mu.Unlock()
	dh.lastDialog = nil
}

// WaitForDialog waits for a dialog to appear within the timeout.
//
// Takes timeout (time.Duration) which specifies how long to wait for a dialog.
//
// Returns *DialogInfo which contains the dialog details when one appears.
// Returns error when the timeout is reached before a dialog appears.
func (dh *DialogHandler) WaitForDialog(timeout time.Duration) (*DialogInfo, error) {
	select {
	case info := <-dh.dialogChan:
		return &info, nil
	case <-time.After(timeout):
		return nil, errors.New("timeout waiting for dialog")
	}
}

// handleTargetEvent processes CDP events for dialog handling.
//
// Takes ctx (*ActionContext) which provides the execution context for actions.
// Takes ev (any) which is the CDP event to process.
func (dh *DialogHandler) handleTargetEvent(ctx *ActionContext, ev any) {
	if dh.isStopped() {
		return
	}

	dialog, ok := ev.(*page.EventJavascriptDialogOpening)
	if !ok {
		return
	}

	info := dh.recordDialog(dialog)
	dh.autoHandleDialog(ctx, info)
}

// isStopped checks if the handler has been stopped.
//
// Returns bool which is true if the stop channel is closed or the handler is
// disabled.
//
// Safe for concurrent use. Uses a read lock to check the enabled state.
func (dh *DialogHandler) isStopped() bool {
	select {
	case <-dh.stopChan:
		return true
	default:
	}

	dh.mu.RLock()
	enabled := dh.enabled
	dh.mu.RUnlock()

	return !enabled
}

// recordDialog records dialog info and sends to channel.
//
// Takes dialog (*page.EventJavascriptDialogOpening) which provides the dialog
// event data to record.
//
// Returns DialogInfo which contains the recorded dialog details.
//
// Safe for concurrent use. Sends to the dialog channel without blocking if the
// channel buffer is full.
func (dh *DialogHandler) recordDialog(dialog *page.EventJavascriptDialogOpening) DialogInfo {
	dh.mu.Lock()
	info := DialogInfo{
		Type:    DialogType(dialog.Type.String()),
		Message: dialog.Message,
		URL:     dialog.URL,
	}
	dh.lastDialog = &info
	closed := dh.closed
	dh.mu.Unlock()

	if !closed {
		select {
		case dh.dialogChan <- info:
		default:
		}
	}

	return info
}

// autoHandleDialog spawns a background task to handle the
// dialog, checking enabled state under the read lock first.
//
// Takes ctx (*ActionContext) which provides the browser context
// for dispatching the dialog response.
//
// Safe for concurrent use. Spawns a goroutine to dispatch the dialog response
// asynchronously, tracked by the handler's WaitGroup.
func (dh *DialogHandler) autoHandleDialog(ctx *ActionContext, _ DialogInfo) {
	dh.mu.RLock()
	if !dh.enabled || dh.closed {
		dh.mu.RUnlock()
		return
	}
	dh.wg.Add(1)
	dh.mu.RUnlock()

	go func() {
		defer dh.wg.Done()
		select {
		case <-ctx.Ctx.Done():
			return
		default:
		}
		timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("dialog autoHandleDialog exceeded %s timeout", DefaultActionTimeout))
		defer cancel()
		_ = chromedp.Run(timedCtx, chromedp.ActionFunc(func(ctx2 context.Context) error {
			return page.HandleJavaScriptDialog(dh.autoAccept).WithPromptText(dh.autoText).Do(ctx2)
		}))
	}()
}

// DialogAutoHandler provides a way to stop automatic dialog handling.
type DialogAutoHandler struct {
	// stopChan signals when to stop the dialog handling goroutine.
	stopChan chan struct{}

	// wg tracks goroutines handling dialog events.
	wg sync.WaitGroup

	// stopped indicates whether the handler has been shut down.
	stopped bool

	// mu guards access to the stopped field.
	mu sync.Mutex
}

// Stop stops the automatic dialog handling and waits for cleanup.
//
// Safe for concurrent use. Waits up to two seconds for in-flight goroutines
// to complete before returning.
func (h *DialogAutoHandler) Stop() {
	h.mu.Lock()
	if !h.stopped {
		close(h.stopChan)
		h.stopped = true
	}
	h.mu.Unlock()

	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

// HandleAlert accepts an alert dialog.
//
// This is a one-shot handler that waits for and accepts a single alert.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
//
// Returns error when the dialog cannot be accepted.
func HandleAlert(ctx *ActionContext) error {
	return handleDialog(ctx, true, "")
}

// HandleConfirm handles a confirm dialog.
// If accept is true, clicks OK; otherwise clicks Cancel.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes accept (bool) which determines whether to click OK or Cancel.
//
// Returns error when the dialog cannot be handled.
func HandleConfirm(ctx *ActionContext, accept bool) error {
	return handleDialog(ctx, accept, "")
}

// HandlePrompt handles a prompt dialog.
//
// If accept is true, enters the text and clicks OK; otherwise clicks Cancel.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes accept (bool) which controls whether to accept or cancel the dialog.
// Takes text (string) which specifies the text to enter when accepting.
//
// Returns error when handling the dialog fails.
func HandlePrompt(ctx *ActionContext, accept bool, text string) error {
	return handleDialog(ctx, accept, text)
}

// DismissDialog dismisses any open dialog by clicking Cancel or No.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
//
// Returns error when the dialog cannot be dismissed.
func DismissDialog(ctx *ActionContext) error {
	return handleDialog(ctx, false, "")
}

// AcceptDialog accepts any open dialog by clicking OK/Yes.
//
// Takes ctx (*ActionContext) which provides the browser action context.
//
// Returns error when the dialog cannot be accepted.
func AcceptDialog(ctx *ActionContext) error {
	return handleDialog(ctx, true, "")
}

// SetupDialogAutoAccept sets up automatic acceptance of all dialogues. This is
// useful when you expect dialogues but do not need to inspect them.
//
// Takes ctx (*ActionContext) which provides the browser context for listening.
// Takes promptText (string) which specifies the text to enter for prompt
// dialogues.
//
// Returns *DialogAutoHandler which can be used to stop the automatic handling.
func SetupDialogAutoAccept(ctx *ActionContext, promptText string) *DialogAutoHandler {
	handler := &DialogAutoHandler{
		stopChan: make(chan struct{}),
		wg:       sync.WaitGroup{},
		stopped:  false,
		mu:       sync.Mutex{},
	}

	chromedp.ListenTarget(ctx.Ctx, func(ev any) {
		select {
		case <-handler.stopChan:
			return
		default:
		}

		if _, ok := ev.(*page.EventJavascriptDialogOpening); ok {
			handler.wg.Go(func() {
				select {
				case <-ctx.Ctx.Done():
					return
				default:
				}
				timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("dialog SetupDialogAutoAccept exceeded %s timeout", DefaultActionTimeout))
				defer cancel()
				_ = chromedp.Run(timedCtx, chromedp.ActionFunc(func(ctx2 context.Context) error {
					return page.HandleJavaScriptDialog(true).WithPromptText(promptText).Do(ctx2)
				}))
			})
		}
	})
	return handler
}

// SetupDialogAutoDismiss sets up automatic dismissal of all dialogues.
//
// Takes ctx (*ActionContext) which provides the browser context for listening.
//
// Returns *DialogAutoHandler which can be used to stop the automatic handling.
func SetupDialogAutoDismiss(ctx *ActionContext) *DialogAutoHandler {
	handler := &DialogAutoHandler{
		stopChan: make(chan struct{}),
		wg:       sync.WaitGroup{},
		stopped:  false,
		mu:       sync.Mutex{},
	}

	chromedp.ListenTarget(ctx.Ctx, func(ev any) {
		select {
		case <-handler.stopChan:
			return
		default:
		}

		if _, ok := ev.(*page.EventJavascriptDialogOpening); ok {
			handler.wg.Go(func() {
				select {
				case <-ctx.Ctx.Done():
					return
				default:
				}
				timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("dialog SetupDialogAutoDismiss exceeded %s timeout", DefaultActionTimeout))
				defer cancel()
				_ = chromedp.Run(timedCtx, chromedp.ActionFunc(func(ctx2 context.Context) error {
					return page.HandleJavaScriptDialog(false).Do(ctx2)
				}))
			})
		}
	})
	return handler
}

// TriggerAlert triggers an alert dialog via JavaScript.
// Useful for testing dialog handling.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes message (string) which specifies the text to display in the alert.
//
// Returns error when the JavaScript execution fails.
func TriggerAlert(ctx *ActionContext, message string) error {
	js := scripts.MustExecute("trigger_alert.js.tmpl", map[string]any{
		"Message": message,
	})
	return chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
}

// TriggerConfirm triggers a confirm dialog via JavaScript.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes message (string) which specifies the message to display in the dialog.
//
// Returns bool which is true if confirmed, false if cancelled.
// Returns error when the JavaScript execution fails.
func TriggerConfirm(ctx *ActionContext, message string) (bool, error) {
	js := scripts.MustExecute("trigger_confirm.js.tmpl", map[string]any{
		"Message": message,
	})
	var result bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	return result, err
}

// TriggerPrompt triggers a prompt dialog via JavaScript.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes message (string) which specifies the prompt message to display.
// Takes defaultValue (string) which sets the initial value in the input field.
//
// Returns string which contains the entered text, or empty string if cancelled.
// Returns error when the JavaScript execution fails.
func TriggerPrompt(ctx *ActionContext, message, defaultValue string) (string, error) {
	js := scripts.MustExecute("trigger_prompt.js.tmpl", map[string]any{
		"Message":      message,
		"DefaultValue": defaultValue,
	})
	var result *string
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	return *result, nil
}

// handleDialog handles a JavaScript dialog by accepting or dismissing it.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes accept (bool) which indicates whether to accept or dismiss the dialog.
// Takes promptText (string) which specifies text to enter for prompt dialogues.
//
// Returns error when the dialog cannot be handled.
func handleDialog(ctx *ActionContext, accept bool, promptText string) error {
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return page.HandleJavaScriptDialog(accept).WithPromptText(promptText).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("handling dialog: %w", err)
	}
	return nil
}
