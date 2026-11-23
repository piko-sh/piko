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

package runner

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestBrowserIntakeCrossChecks(t *testing.T) {
	want := generate.New("form-shim-contracts", 7, 0).Expected.FormShim
	last := want.Steps[len(want.Steps)-1]
	server := &FormShimReceipt{}
	for _, step := range want.Steps {
		server.Steps = append(server.Steps, FormShimRead{Name: step.Name, Status: step.Status, After: step.After})
	}
	client := &casefile.ClientTransition{FormShim: &casefile.FormShimBrowser{
		Page: want.Page, Module: formActionModule, FormAction: "/_piko/actions/intake.Submit", Label: last.Label, Ticket: last.Ticket,
		Before: &last.After, After: &last.After, Rejected: true, Status: http.StatusTooManyRequests, ErrorKind: "http",
		Requests: []casefile.FormShimBrowserRequest{
			{Method: http.MethodPost, Path: "/_piko/actions/intake.Read", Status: http.StatusOK, BindingCookie: true},
			{Method: http.MethodPost, Path: "/_piko/actions/intake.Submit", Status: http.StatusTooManyRequests, BindingCookie: true},
			{Method: http.MethodPost, Path: "/_piko/actions/intake.Read", Status: http.StatusOK, BindingCookie: true},
		},
	}}
	require.NoError(t, checkBrowserIntake(want, server, client))
	testCase := casefile.Case{Expected: casefile.Expected{FormShim: want}}
	observation := Observation{FormShim: server, Client: client}
	for _, selected := range []bool{true, false} {
		routes, err := browserWorkflowRoutes(&testCase, &observation, selected)
		require.NoError(t, err)
		route := routes["/intake"]
		assert.Equal(t, selected, route.Selected, "%+v", route)
		assert.Equal(t, selected, route.Visited, "%+v", route)
		assert.Equal(t, selected, route.Verified, "%+v", route)
		assert.Equal(t, "workflow", route.Role)
	}
	client.FormShim.Page = "/main"
	routes, err := browserWorkflowRoutes(&testCase, &observation, true)
	require.Error(t, err, "wrong URL credited")
	assert.False(t, routes["/intake"].Visited, "wrong URL credited: %+v", routes)
	assert.False(t, routes["/intake"].Verified, "wrong URL credited: %+v", routes)
	client.FormShim.Page = "/intake"
	client.FormShim.Rejected = false
	routes, err = browserWorkflowRoutes(&testCase, &observation, true)
	require.NoError(t, err)
	assert.True(t, routes["/intake"].Visited, "wrong form response credited: %+v", routes)
	assert.False(t, routes["/intake"].Verified, "wrong form response credited: %+v", routes)
	client.FormShim.Rejected = true
	require.NoError(t, checkBrowserIntake(want, server, nil), "browser-off")
	legacy := *want
	legacy.BrowserSubmission = false
	require.NoError(t, checkBrowserIntake(&legacy, server, &casefile.ClientTransition{}), "legacy receipt")
	data, err := json.Marshal(client)
	require.NoError(t, err)
	for _, mutation := range []struct {
		mutate  func(*casefile.FormShimBrowser)
		subject string
	}{
		{subject: "module", mutate: func(browser *casefile.FormShimBrowser) { browser.Module = "/different.js" }},
		{subject: "label", mutate: func(browser *casefile.FormShimBrowser) { browser.Label = "different label" }},
		{subject: "quota-error", mutate: func(browser *casefile.FormShimBrowser) { browser.Rejected = false }},
		{subject: "quota-error", mutate: func(browser *casefile.FormShimBrowser) { browser.ErrorKind = "network" }},
		{subject: "initial-model/writes", mutate: func(browser *casefile.FormShimBrowser) { browser.Before.Writes++ }},
		{subject: "after/checks", mutate: func(browser *casefile.FormShimBrowser) { browser.After.Checks++ }},
		{subject: "after/events", mutate: func(browser *casefile.FormShimBrowser) { browser.After.Events[0] = "extra" }},
		{subject: "requests", mutate: func(browser *casefile.FormShimBrowser) { browser.Requests = browser.Requests[:2] }},
		{subject: "request-2/http", mutate: func(browser *casefile.FormShimBrowser) { browser.Requests[1].Status = http.StatusOK }},
		{subject: "request-2/cookie", mutate: func(browser *casefile.FormShimBrowser) { browser.Requests[1].BindingCookie = false }},
	} {
		var changed casefile.ClientTransition
		require.NoError(t, json.Unmarshal(data, &changed))
		mutation.mutate(changed.FormShim)
		err := checkBrowserIntake(want, server, &changed)
		failure, ok := errors.AsType[*responseMismatch](err)
		if assert.True(t, ok, "%s: got %v", mutation.subject, err) {
			assert.Equal(t, "form-shim/browser/"+mutation.subject, failure.FailureSubject())
		}
	}
	server.Steps[len(server.Steps)-1].After.Checks++
	err = checkBrowserIntake(want, server, client)
	failure, ok := errors.AsType[*responseMismatch](err)
	require.True(t, ok, "HTTP divergence: %v", err)
	assert.Equal(t, "form-shim/browser/http-ledger/checks", failure.FailureSubject())
}
