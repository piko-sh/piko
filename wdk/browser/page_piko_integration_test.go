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

//go:build integration

package browser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

func TestPagePikoReportsFailures(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	cancelPage(t, page)
	requireEachStepFails(t, recorder, []pageStep{
		{name: "TriggerPartialReload", run: func() { page.TriggerPartialReload("cart", map[string]any{"id": 1}) }},
		{name: "TriggerBusEvent", run: func() { page.TriggerBusEvent("saved", map[string]any{"id": 1}) }},
		{name: "TriggerBusEvent", run: func() { page.TriggerBusEvent("saved", 42) }},
		{name: "TriggerBusEvent", run: func() { page.TriggerBusEvent("saved", nil) }},
		{name: "PikoBusEmit", run: func() { page.PikoBusEmit("saved", nil) }},
		{name: "PikoBusWaitForEvent", run: func() { page.PikoBusWaitForEvent("saved") }},
		{name: "PikoPartialReloadWithOptions", run: func() {
			page.PikoPartialReloadWithOptions("cart", nil, &browser_provider_chromedp.PartialReloadOptions{Mode: "replace"})
		}},
		{name: "PikoPartialReload", run: func() { page.PikoPartialReload("cart", nil) }},
		{name: "PikoGetPartialState", run: func() { page.PikoGetPartialState("cart") }},
		{name: "PikoSetupEventLog", run: func() { page.PikoSetupEventLog() }},
		{name: "PikoGetEventLog", run: func() { page.PikoGetEventLog() }},
		{name: "PikoClearEventLog", run: func() { page.PikoClearEventLog() }},
		{name: "PikoCheckBusEventReceived", run: func() { page.PikoCheckBusEventReceived("saved") }},
		{name: "PikoDispatchFragmentMorph", run: func() { page.PikoDispatchFragmentMorph("#title", "<h1>x</h1>") }},
		{name: "PikoDebugGetPartialInfo", run: func() { page.PikoDebugGetPartialInfo("#title") }},
		{name: "PikoDebugIsConnected", run: func() { page.PikoDebugIsConnected("#title") }},
		{name: "PikoDebugGetCleanupCount", run: func() { page.PikoDebugGetCleanupCount("#title") }},
		{name: "PikoDebugGetRegisteredCallbacks", run: func() { page.PikoDebugGetRegisteredCallbacks("#title") }},
		{name: "PikoDebugGetAllConnectedPartials", run: func() { page.PikoDebugGetAllConnectedPartials() }},
		{name: "PikoDebugIsAvailable", run: func() { page.PikoDebugIsAvailable() }},
	})
}

func TestPagePikoWithoutRuntime(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.Navigate("/")
	assert.False(t, page.PikoDebugIsAvailable())
	require.Empty(t, recorder.take())

	failures := recorder.attempt(func() { page.PikoWaitForPartialReload("cart", WithTimeout(100*time.Millisecond)) })
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0], "PikoWaitForPartialReload")
}
