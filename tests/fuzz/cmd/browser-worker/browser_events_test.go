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
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/textlimit"
	"piko.sh/piko/wdk/browser"
)

func TestEventDiagnosticsBoundAndTruncateMessages(t *testing.T) {
	diagnostics := &eventDiagnostics{host: "app.invalid"}
	assert.Empty(t, diagnostics.summary())
	diagnostics.observe(browser.NetworkEvent{Kind: browser.ResponseReceived, URL: "http://other.invalid/x", Status: http.StatusNotFound})
	diagnostics.observe(browser.NetworkEvent{Kind: browser.ResponseReceived, URL: "http://app.invalid/ok", Status: http.StatusOK})
	assert.Empty(t, diagnostics.summary())
	diagnostics.observe(browser.NetworkEvent{Kind: browser.ResponseReceived, URL: "http://app.invalid/missing", Status: http.StatusNotFound})
	diagnostics.observe(browser.NetworkEvent{Kind: browser.JavaScriptException, Exception: strings.Repeat("é", eventsDiagnosticBytes)})
	for range eventsDiagnosticLimit {
		diagnostics.observe(browser.NetworkEvent{Kind: browser.JavaScriptException, Exception: "extra"})
	}
	assert.Len(t, diagnostics.messages, eventsDiagnosticLimit)
	assert.Equal(t, "HTTP 404 /missing", diagnostics.messages[0])
	assert.True(t, strings.HasSuffix(diagnostics.messages[1], textlimit.Marker))
	assert.LessOrEqual(t, len(diagnostics.messages[1]), eventsDiagnosticBytes+len(textlimit.Marker))
	assert.True(t, strings.HasPrefix(diagnostics.summary(), " (HTTP 404 /missing, "))
}

func TestBrowserEventsRecordsUnparseableAddress(t *testing.T) {
	trace := browserEvents(t.Context(), nil, "://missing-scheme", casefile.EventsInput{Rotate: 2})
	assert.Contains(t, trace.DriverError, "missing protocol scheme")
	assert.Equal(t, 2, trace.Input.Rotate)
	assert.Empty(t, trace.States)
}
