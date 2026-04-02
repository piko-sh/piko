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

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

// TriggerPartialReload triggers a Piko partial reload.
//
// Takes name (string) which identifies the component to reload.
// Takes data (map[string]any) which provides the reload payload.
//
// Returns *Page which allows method chaining.
func (p *Page) TriggerPartialReload(name string, data map[string]any) *Page {
	p.beforeAction("TriggerPartialReload", name)
	start := time.Now()
	err := browser_provider_chromedp.TriggerPartialReload(p.actionCtx(), name, data, 0)
	p.afterAction("TriggerPartialReload", name, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("TriggerPartialReload(%q) failed: %v", name, err)
	}
	return p
}

// WaitForPartialReload waits for a Piko partial to finish loading.
//
// Takes name (string) which identifies the partial to wait for.
//
// Returns *Page which allows method chaining.
func (p *Page) WaitForPartialReload(name string) *Page {
	p.beforeAction("WaitForPartialReload", name)
	start := time.Now()
	err := browser_provider_chromedp.WaitForPartialReload(p.actionCtx(), name, 5*time.Second)
	p.afterAction("WaitForPartialReload", name, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("WaitForPartialReload(%q) failed: %v", name, err)
	}
	return p
}

// TriggerBusEvent triggers a Piko event bus event.
//
// Takes event (string) which specifies the event name to trigger.
// Takes payload (any) which provides the event data to send.
//
// Returns *Page which allows method chaining.
func (p *Page) TriggerBusEvent(event string, payload any) *Page {
	p.beforeAction("TriggerBusEvent", event)
	start := time.Now()
	detail := make(map[string]any)
	if payload != nil {
		if m, ok := payload.(map[string]any); ok {
			detail = m
		} else {
			detail["payload"] = payload
		}
	}
	err := browser_provider_chromedp.TriggerBusEvent(p.actionCtx(), event, detail)
	p.afterAction("TriggerBusEvent", event, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("TriggerBusEvent(%q) failed: %v", event, err)
	}
	return p
}

// PikoBusEmit emits a Piko bus event. This is an alias for TriggerBusEvent
// with explicit naming.
//
// Takes eventName (string) which specifies the event to emit.
// Takes detail (map[string]any) which provides event payload data.
//
// Returns *Page which allows method chaining.
func (p *Page) PikoBusEmit(eventName string, detail map[string]any) *Page {
	p.beforeAction("PikoBusEmit", eventName)
	start := time.Now()
	err := browser_provider_chromedp.PikoBusEmit(p.actionCtx(), eventName, detail)
	p.afterAction("PikoBusEmit", eventName, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PikoBusEmit(%q) failed: %v", eventName, err)
	}
	return p
}

// PikoBusWaitForEvent waits for a Piko bus event to be emitted and returns
// its detail.
//
// Takes eventName (string) which specifies the event name to wait for.
// Takes opts (...WaitOption) which provides optional wait behaviour controls.
//
// Returns map[string]any which contains the event detail payload.
func (p *Page) PikoBusWaitForEvent(eventName string, opts ...WaitOption) map[string]any {
	config := defaultWaitConfig()
	for _, opt := range opts {
		opt(&config)
	}

	p.beforeAction("PikoBusWaitForEvent", eventName)
	start := time.Now()
	detail, err := browser_provider_chromedp.PikoBusWaitForEvent(p.actionCtx(), eventName, config.timeout)
	p.afterAction("PikoBusWaitForEvent", eventName, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PikoBusWaitForEvent(%q) failed: %v", eventName, err)
	}
	return detail
}

// PikoPartialReload triggers a Piko partial reload.
//
// Takes partialName (string) which identifies the partial to reload.
// Takes data (map[string]any) which provides the data for the partial.
//
// Returns *Page which allows method chaining.
func (p *Page) PikoPartialReload(partialName string, data map[string]any) *Page {
	p.beforeAction("PikoPartialReload", partialName)
	start := time.Now()
	err := browser_provider_chromedp.PikoPartialReload(p.actionCtx(), partialName, data)
	p.afterAction("PikoPartialReload", partialName, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PikoPartialReload(%q) failed: %v", partialName, err)
	}
	return p
}

// PikoPartialReloadWithLevel triggers a Piko partial reload with a specific
// refresh level.
//
// Takes partialName (string) which identifies the partial to reload.
// Takes data (map[string]any) which provides the data to pass to the partial.
// Takes refreshLevel (int) which specifies the refresh level for the reload.
//
// Returns *Page which enables method chaining for further actions.
func (p *Page) PikoPartialReloadWithLevel(partialName string, data map[string]any, refreshLevel int) *Page {
	detail := fmt.Sprintf("%s (level %d)", partialName, refreshLevel)
	p.beforeAction("PikoPartialReloadWithLevel", detail)
	start := time.Now()
	err := browser_provider_chromedp.PikoPartialReloadWithLevel(p.actionCtx(), partialName, data, refreshLevel)
	p.afterAction("PikoPartialReloadWithLevel", detail, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PikoPartialReloadWithLevel(%q, %d) failed: %v", partialName, refreshLevel, err)
	}
	return p
}

// PikoWaitForPartialReload waits for a Piko partial to finish reloading.
//
// Takes partialName (string) which identifies the partial to wait for.
// Takes opts (...WaitOption) which provides optional wait behaviour controls.
//
// Returns *Page which enables method chaining.
func (p *Page) PikoWaitForPartialReload(partialName string, opts ...WaitOption) *Page {
	config := defaultWaitConfig()
	for _, opt := range opts {
		opt(&config)
	}

	p.beforeAction("PikoWaitForPartialReload", partialName)
	start := time.Now()
	err := browser_provider_chromedp.PikoWaitForPartialReload(p.actionCtx(), partialName, config.timeout)
	p.afterAction("PikoWaitForPartialReload", partialName, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PikoWaitForPartialReload(%q) failed: %v", partialName, err)
	}
	return p
}

// PikoGetPartialState returns the current state of a Piko partial.
//
// Takes partialName (string) which specifies the name of the partial to query.
//
// Returns map[string]any which contains the current state of the partial.
func (p *Page) PikoGetPartialState(partialName string) map[string]any {
	state, err := browser_provider_chromedp.PikoGetPartialState(p.actionCtx(), partialName)
	if err != nil {
		p.t.Fatalf("PikoGetPartialState(%q) failed: %v", partialName, err)
	}
	return state
}

// PikoSetupEventLog sets up Piko event bus logging for testing.
//
// Returns *Page which allows method chaining.
func (p *Page) PikoSetupEventLog() *Page {
	p.beforeAction("PikoSetupEventLog", "")
	start := time.Now()
	err := browser_provider_chromedp.PikoSetupEventLog(p.actionCtx())
	p.afterAction("PikoSetupEventLog", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PikoSetupEventLog() failed: %v", err)
	}
	return p
}

// PikoGetEventLog returns the captured Piko bus events.
//
// Returns []map[string]any which contains the recorded event log entries.
func (p *Page) PikoGetEventLog() []map[string]any {
	log, err := browser_provider_chromedp.PikoGetEventLog(p.actionCtx())
	if err != nil {
		p.t.Fatalf("PikoGetEventLog() failed: %v", err)
	}
	return log
}

// PikoClearEventLog clears the Piko event log.
//
// Returns *Page which allows method chaining.
func (p *Page) PikoClearEventLog() *Page {
	p.beforeAction("PikoClearEventLog", "")
	start := time.Now()
	err := browser_provider_chromedp.PikoClearEventLog(p.actionCtx())
	p.afterAction("PikoClearEventLog", "", err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PikoClearEventLog() failed: %v", err)
	}
	return p
}

// PikoCheckBusEventReceived checks if a specific Piko bus event was received.
//
// Takes eventName (string) which specifies the event name to check for.
//
// Returns bool which indicates whether the event was received.
func (p *Page) PikoCheckBusEventReceived(eventName string) bool {
	received, err := browser_provider_chromedp.PikoCheckBusEventReceived(p.actionCtx(), eventName)
	if err != nil {
		p.t.Fatalf("PikoCheckBusEventReceived(%q) failed: %v", eventName, err)
	}
	return received
}

// PikoDispatchFragmentMorph dispatches a Piko fragment morph to update DOM
// content.
//
// Takes selector (string) which identifies the target element in the DOM.
// Takes newHTML (string) which provides the new HTML content to morph into.
//
// Returns *Page which allows method chaining for further actions.
func (p *Page) PikoDispatchFragmentMorph(selector, newHTML string) *Page {
	p.beforeAction("PikoDispatchFragmentMorph", selector)
	start := time.Now()
	err := browser_provider_chromedp.PikoDispatchFragmentMorph(p.actionCtx(), selector, newHTML)
	p.afterAction("PikoDispatchFragmentMorph", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PikoDispatchFragmentMorph(%q) failed: %v", selector, err)
	}
	return p
}

// PikoDebugGetPartialInfo returns debug information about a partial element.
// This provides read-only access to internal lifecycle state for E2E testing.
//
// Takes selector (string) which identifies the partial element to inspect.
//
// Returns map[string]any which contains the internal lifecycle state.
func (p *Page) PikoDebugGetPartialInfo(selector string) map[string]any {
	p.beforeAction("PikoDebugGetPartialInfo", selector)
	start := time.Now()
	info, err := browser_provider_chromedp.PikoDebugGetPartialInfo(p.actionCtx(), selector)
	p.afterAction("PikoDebugGetPartialInfo", selector, err != nil, time.Since(start))
	if err != nil {
		p.t.Fatalf("PikoDebugGetPartialInfo(%q) failed: %v", selector, err)
	}
	return info
}

// PikoDebugIsConnected checks if a partial element is currently connected.
//
// Takes selector (string) which identifies the element to check.
//
// Returns bool which indicates whether the element is connected.
func (p *Page) PikoDebugIsConnected(selector string) bool {
	connected, err := browser_provider_chromedp.PikoDebugIsConnected(p.actionCtx(), selector)
	if err != nil {
		p.t.Fatalf("PikoDebugIsConnected(%q) failed: %v", selector, err)
	}
	return connected
}

// PikoDebugGetCleanupCount returns the number of registered cleanup functions.
//
// Takes selector (string) which identifies the cleanup functions to count.
//
// Returns int which is the count of matching cleanup functions.
func (p *Page) PikoDebugGetCleanupCount(selector string) int {
	count, err := browser_provider_chromedp.PikoDebugGetCleanupCount(p.actionCtx(), selector)
	if err != nil {
		p.t.Fatalf("PikoDebugGetCleanupCount(%q) failed: %v", selector, err)
	}
	return count
}

// PikoDebugGetRegisteredCallbacks returns the names of registered lifecycle
// callbacks.
//
// Takes selector (string) which identifies the element to query.
//
// Returns []string which contains the callback names registered for the
// element.
func (p *Page) PikoDebugGetRegisteredCallbacks(selector string) []string {
	callbacks, err := browser_provider_chromedp.PikoDebugGetRegisteredCallbacks(p.actionCtx(), selector)
	if err != nil {
		p.t.Fatalf("PikoDebugGetRegisteredCallbacks(%q) failed: %v", selector, err)
	}
	return callbacks
}

// PikoDebugGetAllConnectedPartials returns selectors for all connected
// partial elements.
//
// Returns []string which contains the CSS selectors for connected partials.
func (p *Page) PikoDebugGetAllConnectedPartials() []string {
	selectors, err := browser_provider_chromedp.PikoDebugGetAllConnectedPartials(p.actionCtx())
	if err != nil {
		p.t.Fatalf("PikoDebugGetAllConnectedPartials() failed: %v", err)
	}
	return selectors
}

// PikoDebugIsAvailable checks if the debug API is available in the browser.
//
// Returns bool which is true when the debug API can be accessed.
func (p *Page) PikoDebugIsAvailable() bool {
	available, err := browser_provider_chromedp.PikoDebugIsAvailable(p.actionCtx())
	if err != nil {
		p.t.Fatalf("PikoDebugIsAvailable() failed: %v", err)
	}
	return available
}
