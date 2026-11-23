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

package shimcontracts

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSRFHistory(t *testing.T) {
	for _, data := range [][]byte{nil, {4, 8, 9, 10, 11, 4, 11, 9, 11}, {0xf0, 0x91, 0x52, 0x33, 0x7c, 0x2d, 0xab, 0x89, 0x1b}} {
		require.NoError(t, RunCSRF(t.Context(), data))
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, RunCSRF(ctx, nil), context.Canceled)
}

func TestCSRFFormObservations(t *testing.T) {
	for _, test := range []struct {
		name    string
		expired bool
		change  func(*httptest.ResponseRecorder, *csrfScenario)
		want    string
	}{
		{name: "returned-label", expired: false, change: func(recorder *httptest.ResponseRecorder, _ *csrfScenario) {
			recorder.Body = bytes.NewBufferString(`{"saved":"other"}`)
		}, want: "csrf-accept:"},
		{name: "business-arguments", expired: false, change: func(_ *httptest.ResponseRecorder, scenario *csrfScenario) { scenario.argsOK = false }, want: "csrf-accept:"},
		{name: "duplicate-call", expired: false, change: func(_ *httptest.ResponseRecorder, scenario *csrfScenario) { scenario.calls++ }, want: "csrf-accept:"},
		{name: "wrong-recovery", expired: true, change: func(recorder *httptest.ResponseRecorder, _ *csrfScenario) {
			recorder.Body = bytes.NewBufferString(`{"status":403,"error":"csrf_missing"}`)
		}, want: "csrf-refuse:"},
		{name: "refused-but-called", expired: true, change: func(_ *httptest.ResponseRecorder, scenario *csrfScenario) { scenario.calls++ }, want: "csrf-refuse:"},
		{name: "oversized-reply", expired: false, change: func(recorder *httptest.ResponseRecorder, _ *csrfScenario) {
			recorder.Body = bytes.NewBufferString(strings.Repeat(" ", csrfReplyLimit+1))
		}, want: "csrf-reply:"},
	} {
		t.Run(test.name, func(t *testing.T) {
			scenario, err := newCSRFScenario()
			require.NoError(t, err)
			scenario.label = "saved note"
			require.NoError(t, scenario.issue(t.Context(), 0))
			if test.expired {
				require.NoError(t, scenario.logout())
			}
			require.NoError(t, scenario.submit(t.Context(), 0, true, false, forgeNone))
			mounted := scenario.handler
			scenario.handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				recorder := httptest.NewRecorder()
				mounted.ServeHTTP(recorder, request)
				test.change(recorder, scenario)
				writer.WriteHeader(recorder.Code)
				_, _ = writer.Write(recorder.Body.Bytes())
			})
			err = scenario.submit(t.Context(), 0, true, false, forgeNone)
			require.Error(t, err, "changed observation accepted")
			assert.True(t, strings.HasPrefix(err.Error(), test.want), err.Error())
		})
	}
}

func TestCSRFForgeriesAreRefused(t *testing.T) {
	for _, forgery := range []csrfForgery{forgeSwappedPair, forgeTamperedAction, forgeTamperedEphemeral, forgeCrossSiteWithoutTokens} {
		t.Run(fmt.Sprintf("forgery-%d", forgery), func(t *testing.T) {
			scenario, err := newCSRFScenario()
			require.NoError(t, err)
			scenario.label = "saved note"
			require.NoError(t, scenario.issue(t.Context(), 0))
			require.NoError(t, scenario.issue(t.Context(), 1))
			require.NoError(t, scenario.submit(t.Context(), 1, false, false, forgery), "the service must refuse the forged submission")

			scenario.handler = http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				scenario.calls++
				scenario.argsOK = true
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte(`{"saved":"saved note"}`))
			})
			err = scenario.submit(t.Context(), 1, false, false, forgery)
			require.Error(t, err, "a handler that accepts the forgery must be reported")
			assert.True(t, strings.HasPrefix(err.Error(), "csrf-refuse:"), err.Error())
		})
	}
}

func FuzzCSRFContract(f *testing.F) { fuzzContract(f, RunCSRF) }
