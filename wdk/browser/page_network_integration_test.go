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
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

const (
	fetchAPI = "fetch('/api').then(r => r.json()).then(j => j.agent + '|' + j.custom)"
)

func TestPageNetwork(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.Navigate("/")
	assert.Equal(t, "text/html", page.GetResponseHeaders()["contentType"])

	page.SetUserAgent("wdk-agent").SetCustomHeader("X-Custom", "one")
	assert.Equal(t, "wdk-agent|one", page.EvalReturn(fetchAPI))
	page.SetAuthorisationHeader("token").SetBasicAuthHeader("user:pass").SetAcceptLanguageHeader("en-GB")
	page.SetExtraHTTPHeaders(map[string]string{"X-Custom": "two"})
	assert.Equal(t, "wdk-agent|two", page.EvalReturn(fetchAPI))
	page.ClearExtraHTTPHeaders()
	assert.Equal(t, "wdk-agent|", page.EvalReturn(fetchAPI))

	page.InterceptRequest("/api", browser_provider_chromedp.MockResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"agent":"mock","custom":"body"}`,
		StatusCode: 200,
	})
	assert.Equal(t, "mock|body", page.EvalReturn(fetchAPI))
	page.RemoveRequestIntercept("/api")
	assert.Equal(t, "wdk-agent|", page.EvalReturn(fetchAPI))
	page.InterceptRequest("/api", browser_provider_chromedp.MockResponse{Body: `{"agent":"again","custom":""}`, StatusCode: 200})
	page.ClearRequestIntercepts()
	assert.Equal(t, "wdk-agent|", page.EvalReturn(fetchAPI))

	page.Eval(scripts.MustGet("network_tracking.js"))
	page.EvalReturn(fetchAPI)
	matcher := browser_provider_chromedp.RequestMatcher{URLContains: "/api"}
	assert.True(t, page.CheckRequestMade(matcher))
	assert.False(t, page.CheckRequestMade(browser_provider_chromedp.RequestMatcher{URLContains: "/never"}))
	request := page.WaitForRequest(matcher, WithTimeout(time.Second))
	require.NotNil(t, request)
	assert.Contains(t, request.URL, "/api")
	require.Empty(t, recorder.take())

	failures := recorder.attempt(func() {
		page.WaitForRequest(browser_provider_chromedp.RequestMatcher{URLContains: "/never"}, WithTimeout(50*time.Millisecond))
	})
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0], "WaitForRequest")
}

func TestPageDialogs(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	page.Navigate("/")

	page.SetupDialogAutoAccept("Ada").Click("#ask")
	assert.Equal(t, "Ada", page.GetAttribute("#ask", "data-answer"))
	page.SetupDialogAutoDismiss().Click("#confirm")
	assert.Equal(t, "false", page.GetAttribute("#confirm", "data-answer"))
	require.Empty(t, recorder.take())

	requireEachStepFails(t, recorder, []pageStep{
		{name: "HandleAlert", run: func() { page.HandleAlert() }},
		{name: "HandleConfirm", run: func() { page.HandleConfirm(true) }},
		{name: "HandlePrompt", run: func() { page.HandlePrompt(true, "x") }},
		{name: "DismissDialog", run: func() { page.DismissDialog() }},
		{name: "AcceptDialog", run: func() { page.AcceptDialog() }},
	})
}

func TestPageNetworkReportsFailures(t *testing.T) {
	page, recorder, _ := newFixturePage(t)
	cancelPage(t, page)
	matcher := browser_provider_chromedp.RequestMatcher{URLContains: "/api"}
	requireEachStepFails(t, recorder, []pageStep{
		{name: "WaitForNetworkIdle", run: func() { page.WaitForNetworkIdle(time.Millisecond) }},
		{name: "CheckRequestMade", run: func() { page.CheckRequestMade(matcher) }},
		{name: "WaitForRequest", run: func() { page.WaitForRequest(matcher) }},
		{name: "InterceptRequest", run: func() { page.InterceptRequest("/api", browser_provider_chromedp.MockResponse{}) }},
		{name: "RemoveRequestIntercept", run: func() { page.RemoveRequestIntercept("/api") }},
		{name: "ClearRequestIntercepts", run: func() { page.ClearRequestIntercepts() }},
		{name: "SetExtraHTTPHeaders", run: func() { page.SetExtraHTTPHeaders(map[string]string{"A": "b"}) }},
		{name: "SetAuthorisationHeader", run: func() { page.SetAuthorisationHeader("token") }},
		{name: "SetBasicAuthHeader", run: func() { page.SetBasicAuthHeader("a:b") }},
		{name: "SetUserAgent", run: func() { page.SetUserAgent("agent") }},
		{name: "ClearExtraHTTPHeaders", run: func() { page.ClearExtraHTTPHeaders() }},
		{name: "SetAcceptLanguageHeader", run: func() { page.SetAcceptLanguageHeader("en") }},
		{name: "SetCustomHeader", run: func() { page.SetCustomHeader("A", "b") }},
		{name: "GetResponseHeaders", run: func() { page.GetResponseHeaders() }},
	})
}
