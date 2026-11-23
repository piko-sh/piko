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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

func TestIntakeRequestsCorrelateCookieAndWireEvents(t *testing.T) {
	done := make(chan struct{})
	calls := &intakeRequests{origin: "http://loopback.invalid", cookies: make(map[browser.RequestID]bool), complete: done}
	binding := browser.RequestCookie{Name: "_piko_csrf_token"}
	blocked := binding
	blocked.Blocked = true
	id := browser.RequestID("first")
	extra := func(id browser.RequestID, cookie browser.RequestCookie) {
		calls.observe(browser.NetworkEvent{Kind: browser.RequestCookies, ID: id, Cookies: []browser.RequestCookie{cookie}})
	}
	request := func(id browser.RequestID, path string) {
		calls.observe(browser.NetworkEvent{Kind: browser.RequestStarted, ID: id, URL: calls.origin + path, Method: http.MethodPost})
	}
	response := func(id browser.RequestID, status int) {
		calls.observe(browser.NetworkEvent{Kind: browser.ResponseReceived, ID: id, Status: status})
	}
	extra(id, binding)
	request(id, "/_piko/actions/intake.Read")
	response(id, http.StatusOK)
	id = "second"
	request(id, "/_piko/actions/intake.Submit")
	response(id, http.StatusTooManyRequests)
	extra(id, blocked)
	id = "third"
	request(id, "/_piko/actions/intake.Read")
	response(id, http.StatusOK)
	select {
	case <-done:
		require.FailNow(t, "completed before third extraInfo")
	default:
	}
	extra(id, binding)
	select {
	case <-done:
	default:
		require.FailNow(t, "missing completion")
	}
	assert.False(t, calls.cookies["second"], "blocked binding cookie must not count")
	assert.True(t, calls.cookies["first"])
	assert.True(t, calls.cookies["third"])
	require.Len(t, calls.rows, 3)
	assert.Equal(t, http.StatusTooManyRequests, calls.rows[1].Status)
	var out casefile.FormShimBrowser
	calls.receipt(&out)
	require.Len(t, out.Requests, 3)
	assert.Equal(t, []bool{true, false, true}, []bool{out.Requests[0].BindingCookie, out.Requests[1].BindingCookie, out.Requests[2].BindingCookie})
	assert.Empty(t, out.DriverError)
}

func TestIntakeRequestsReportOverflow(t *testing.T) {
	calls := &intakeRequests{origin: "http://loopback.invalid", cookies: make(map[browser.RequestID]bool)}
	for index := range intakeRequestLimit + 1 {
		calls.observe(browser.NetworkEvent{Kind: browser.RequestStarted, ID: browser.RequestID(strconv.Itoa(index)), URL: calls.origin + "/_piko/actions/intake.Read", Method: http.MethodPost})
	}
	var out casefile.FormShimBrowser
	calls.receipt(&out)
	assert.Len(t, out.Requests, intakeRequestLimit)
	assert.Contains(t, out.DriverError, "intake action count exceeded")
}
