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
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/json"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// PikoFrameworkURL is the URL where the Piko framework JavaScript bundle is
// served.
const PikoFrameworkURL = "/_piko/dist/ppframework.core.es.js"

// PikoBusEmit sends an event to the Piko event bus in the browser.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes eventName (string) which is the name of the event to send.
// Takes detail (map[string]any) which holds optional data for the event.
//
// Returns error when the detail cannot be converted to JSON or when the
// browser fails to send the event.
func PikoBusEmit(ctx *ActionContext, eventName string, detail map[string]any) error {
	detailJSON, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("marshalling event detail: %w", err)
	}

	js := scripts.MustExecute("bus_emit.js.tmpl", map[string]any{
		"EventName":  eventName,
		"DetailJSON": string(detailJSON),
	})

	err = chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("emitting Piko bus event: %w", err)
	}
	return nil
}

// PikoBusWaitForEvent waits for an event on the Piko event bus.
// It sets up a temporary listener and waits for the named event.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes eventName (string) which names the event to wait for.
// Takes timeout (time.Duration) which sets the maximum wait time.
//
// Returns map[string]any which holds the event detail received.
// Returns error when the event does not arrive within the timeout.
func PikoBusWaitForEvent(ctx *ActionContext, eventName string, timeout time.Duration) (map[string]any, error) {
	jsSetup := scripts.MustExecute("bus_wait_for_event.js.tmpl", map[string]any{
		"EventName": eventName,
		"TimeoutMS": timeout.Milliseconds(),
	})

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(jsSetup, nil))
	if err != nil {
		return nil, fmt.Errorf("setting up event listener: %w", err)
	}

	var result any
	jsWait := scripts.MustGet("piko_event_promise.js")

	err = chromedp.Run(ctx.Ctx,
		chromedp.Evaluate(jsWait, &result, chromedp.EvalAsValue),
	)
	if err != nil {
		return nil, fmt.Errorf("waiting for Piko bus event '%s': %w", eventName, err)
	}

	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, nil
}

// PikoPartialReload triggers a reload of a Piko partial element. The partial
// is found by its name, which is the value of the pk-partial attribute.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes partialName (string) which identifies the partial to reload.
// Takes data (map[string]any) which provides optional reload data.
//
// Returns error when the partial cannot be reloaded.
func PikoPartialReload(ctx *ActionContext, partialName string, data map[string]any) error {
	return TriggerPartialReload(ctx, partialName, data, 0)
}

// PikoPartialReloadWithLevel reloads a Piko partial with a given refresh level.
//
// The refresh level controls how the reload is performed:
//   - 0: Default reload
//   - 1: Soft reload (morph)
//   - 2: Replace reload
//   - 3: Hard reload (full page)
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes partialName (string) which identifies the partial to reload.
// Takes data (map[string]any) which provides optional reload data.
// Takes refreshLevel (int) which specifies the reload behaviour.
//
// Returns error when the partial cannot be reloaded.
func PikoPartialReloadWithLevel(ctx *ActionContext, partialName string, data map[string]any, refreshLevel int) error {
	return TriggerPartialReload(ctx, partialName, data, refreshLevel)
}

// PikoWaitForPartialReload waits for a Piko partial to finish reloading.
// Call it after triggering a reload to ensure the DOM has been updated.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes partialName (string) which identifies the partial to wait for.
// Takes timeout (time.Duration) which specifies the maximum wait time.
//
// Returns error when the partial does not finish reloading within the timeout.
func PikoWaitForPartialReload(ctx *ActionContext, partialName string, timeout time.Duration) error {
	return WaitForPartialReload(ctx, partialName, timeout)
}

// PikoGetPartialState retrieves the current state of a Piko partial element.
// This includes whether it's loading, its last reload timestamp, etc.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes partialName (string) which identifies the partial.
//
// Returns map[string]any which contains the partial state data.
// Returns error when the state cannot be retrieved.
func PikoGetPartialState(ctx *ActionContext, partialName string) (map[string]any, error) {
	js := scripts.MustExecute("piko_get_partial_state.js.tmpl", map[string]any{
		"PartialName": partialName,
	})

	var result any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result, chromedp.EvalAsValue))
	if err != nil {
		return nil, fmt.Errorf("getting Piko partial state: %w", err)
	}

	if result == nil {
		return map[string]any{"exists": false, "name": partialName}, nil
	}

	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, fmt.Errorf("unexpected partial state type: %T", result)
}

// PikoCheckBusEventReceived checks if a specific event was received on the bus.
// Use it in test assertions to verify event communication.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes eventName (string) which identifies the event to check.
//
// Returns bool which is true if the event was received.
// Returns error when the check cannot be performed.
func PikoCheckBusEventReceived(ctx *ActionContext, eventName string) (bool, error) {
	js := scripts.MustExecute("piko_check_bus_event.js.tmpl", map[string]any{
		"EventName": eventName,
	})

	var received bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &received))
	if err != nil {
		return false, fmt.Errorf("checking Piko bus event: %w", err)
	}
	return received, nil
}

// PikoSetupEventLog initialises event logging on the page for testing.
// This must be called before events are emitted to capture them.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns error when setup fails.
func PikoSetupEventLog(ctx *ActionContext) error {
	js := scripts.MustGet("piko_setup_event_log.js")

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("setting up Piko event log: %w", err)
	}
	return nil
}

// PikoGetEventLog retrieves the captured event log from the page.
// This requires PikoSetupEventLog to have been called first.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns []map[string]any which contains the captured events.
// Returns error when retrieval fails.
func PikoGetEventLog(ctx *ActionContext) ([]map[string]any, error) {
	var result any
	jsGet := scripts.MustGet("piko_get_event_log.js")

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(jsGet, &result))
	if err != nil {
		return nil, fmt.Errorf("getting Piko event log: %w", err)
	}

	events, ok := result.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected event log type: %T", result)
	}

	log := make([]map[string]any, 0, len(events))
	for _, e := range events {
		if m, ok := e.(map[string]any); ok {
			log = append(log, m)
		}
	}
	return log, nil
}

// PikoClearEventLog clears the captured event log.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns error when clearing fails.
func PikoClearEventLog(ctx *ActionContext) error {
	js := scripts.MustGet("piko_clear_event_log.js")
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("clearing Piko event log: %w", err)
	}
	return nil
}

// PikoDispatchFragmentMorph triggers a fragment morph operation on an element.
// Fragment morphing is used for efficiently updating DOM fragments.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the target element.
// Takes newHTML (string) which contains the new HTML content.
//
// Returns error when the morph operation fails.
func PikoDispatchFragmentMorph(ctx *ActionContext, selector, newHTML string) error {
	js := scripts.MustExecute("piko_fragment_morph.js.tmpl", map[string]any{
		"Selector": selector,
		"NewHTML":  newHTML,
	})

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("dispatching Piko fragment morph: %w", err)
	}
	return nil
}

// PikoDebugGetPartialInfo returns debug information about a partial element.
// This provides read-only access to internal lifecycle state for E2E testing.
//
// The returned map contains:
//   - exists: bool - whether the partial has lifecycle state registered
//   - partialName: string|null - the data-partial-name attribute
//   - partialId: string|null - the partial or data-partial attribute
//   - isConnected: bool - whether the partial is currently connected
//   - connectedOnce: bool - whether onConnected has fired
//   - registeredCallbacks: []string - names of registered lifecycle callbacks
//   - cleanupCount: int - number of registered cleanup functions
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the partial element.
//
// Returns map[string]any which contains the debug information described above.
// Returns error when the element is not found or debug API is unavailable.
func PikoDebugGetPartialInfo(ctx *ActionContext, selector string) (map[string]any, error) {
	js := scripts.MustExecute("piko_debug_get_partial_info.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var result any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting partial debug info: %w", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected debug info type: %T", result)
	}

	if errMessage, hasErr := m["error"]; hasErr {
		return nil, fmt.Errorf("debug API error: %v", errMessage)
	}
	return m, nil
}

// PikoDebugIsConnected checks if a partial element is currently connected.
// A connected partial is one that is in the DOM and has had onConnected fired.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the partial element.
//
// Returns bool which is true if the partial is connected.
// Returns error when the check cannot be performed.
func PikoDebugIsConnected(ctx *ActionContext, selector string) (bool, error) {
	js := scripts.MustExecute("piko_debug_is_connected.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var result bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return false, fmt.Errorf("checking partial connected state: %w", err)
	}
	return result, nil
}

// PikoDebugGetCleanupCount returns the number of registered cleanup functions
// for a partial element. This includes both element-scoped and lifecycle
// cleanups.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the partial element.
//
// Returns int which is the count of registered cleanup functions.
// Returns error when the count cannot be retrieved.
func PikoDebugGetCleanupCount(ctx *ActionContext, selector string) (int, error) {
	js := scripts.MustExecute("piko_debug_get_cleanup_count.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var result float64
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return 0, fmt.Errorf("getting cleanup count: %w", err)
	}
	return int(result), nil
}

// PikoDebugGetRegisteredCallbacks returns the names of registered lifecycle
// callbacks for a partial element.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes selector (string) which identifies the partial element.
//
// Returns []string which contains callback names such as "onConnected",
// "onDisconnected", and others.
// Returns error when the callbacks cannot be retrieved.
func PikoDebugGetRegisteredCallbacks(ctx *ActionContext, selector string) ([]string, error) {
	js := scripts.MustExecute("piko_debug_get_callbacks.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var result []any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting registered callbacks: %w", err)
	}

	callbacks := make([]string, 0, len(result))
	for _, v := range result {
		if s, ok := v.(string); ok {
			callbacks = append(callbacks, s)
		}
	}
	return callbacks, nil
}

// PikoDebugGetAllConnectedPartials returns selectors for all currently
// connected partial elements in the DOM.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns []string which contains CSS selectors for connected partials.
// Returns error when the partials cannot be retrieved.
func PikoDebugGetAllConnectedPartials(ctx *ActionContext) ([]string, error) {
	js := scripts.MustGet("piko_debug_get_connected_partials.js")

	var result []any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting connected partials: %w", err)
	}

	selectors := make([]string, 0, len(result))
	for _, v := range result {
		if s, ok := v.(string); ok {
			selectors = append(selectors, s)
		}
	}
	return selectors, nil
}

// PikoDebugIsAvailable checks if the debug API is available in the browser.
// The debug API is exposed on window.__pikoDebug when the Piko framework loads.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns bool which is true if the debug API is available.
// Returns error when the check cannot be performed.
func PikoDebugIsAvailable(ctx *ActionContext) (bool, error) {
	js := scripts.MustGet("check_piko_debug_available.js")

	var result bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return false, fmt.Errorf("checking debug API availability: %w", err)
	}
	return result, nil
}
