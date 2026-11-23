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
	"testing"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkObservationKeepsRequestIdentityAndCookieEligibility(t *testing.T) {
	started, ok := networkObservation(&network.EventRequestWillBeSent{RequestID: "step", Request: &network.Request{URL: "https://local.invalid/form", Method: "POST", Headers: network.Headers{"cOnTeNt-TyPe": "application/json"}}, Initiator: &network.Initiator{Type: network.InitiatorTypeScript}})
	require.True(t, ok)
	assert.Equal(t, RequestStarted, started.Kind)
	assert.Equal(t, RequestID("step"), started.ID)
	assert.Equal(t, "https://local.invalid/form", started.URL)
	assert.Equal(t, "POST", started.Method)
	assert.Equal(t, "application/json", started.ContentType)
	assert.Equal(t, "script", started.Initiator)

	cookies, ok := networkObservation(&network.EventRequestWillBeSentExtraInfo{RequestID: "step", AssociatedCookies: []*network.AssociatedCookie{
		nil,
		{Cookie: &network.Cookie{Name: "selected", Value: "ignored-value"}},
		{Cookie: &network.Cookie{Name: "withheld"}, BlockedReasons: []network.CookieBlockedReason{"SecureOnly"}},
	}})
	require.True(t, ok)
	assert.Equal(t, RequestCookies, cookies.Kind)
	assert.Equal(t, RequestID("step"), cookies.ID)
	assert.Equal(t, []RequestCookie{{Name: "selected", Blocked: false}, {Name: "withheld", Blocked: true}}, cookies.Cookies)

	response, ok := networkObservation(&network.EventResponseReceived{RequestID: "step", Response: &network.Response{URL: "https://local.invalid/form", Status: 429}})
	require.True(t, ok)
	assert.Equal(t, ResponseReceived, response.Kind)
	assert.Equal(t, RequestID("step"), response.ID)
	assert.Equal(t, 429, response.Status)
	assert.Equal(t, "https://local.invalid/form", response.URL)

	details := &runtime.ExceptionDetails{Text: "fixture exception"}
	exception, ok := networkObservation(&runtime.EventExceptionThrown{ExceptionDetails: details})
	require.True(t, ok)
	assert.Equal(t, JavaScriptException, exception.Kind)
	assert.Equal(t, details.Error(), exception.Exception)

	for _, raw := range []any{nil, &network.EventRequestWillBeSent{}, &network.EventResponseReceived{}, &runtime.EventExceptionThrown{}} {
		_, ok := networkObservation(raw)
		assert.False(t, ok, "unexpected notification for %T", raw)
	}
}

func TestObserveNetworkRequiresAnObserver(t *testing.T) {
	session := &Session{}

	_, err := session.ObserveNetwork(t.Context(), nil)

	require.ErrorIs(t, err, errMissingNetworkObserver)
}
