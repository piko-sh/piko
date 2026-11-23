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
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimitHistory(t *testing.T) {
	for _, data := range [][]byte{nil, {0xff, 0, 1, 2, 3, 4, 8, 12, 128, 255}, {0x32, 11, 7, 14, 196, 64, 17, 0, 0, 0}} {
		require.NoError(t, RunRateLimit(t.Context(), data))
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, RunRateLimit(ctx, nil), context.Canceled)
}

func TestRateLimitObservationRequiresDecisionQuotaAndReset(t *testing.T) {
	end := time.Unix(1700000060, 0).UTC()
	for _, test := range []struct {
		name    string
		allowed bool
		change  func(*httptest.ResponseRecorder)
		want    string
	}{
		{name: "allow-status", allowed: true, change: func(recorder *httptest.ResponseRecorder) { recorder.Code = http.StatusTooManyRequests }, want: "rate-decision:"},
		{name: "deny-status", allowed: false, change: func(recorder *httptest.ResponseRecorder) { recorder.Code = http.StatusOK }, want: "rate-decision:"},
		{name: "wrong-capacity", allowed: true, change: func(recorder *httptest.ResponseRecorder) { recorder.Header().Set("X-RateLimit-Limit", "3") }, want: "rate-quota:"},
		{name: "wrong-remaining", allowed: false, change: func(recorder *httptest.ResponseRecorder) { recorder.Header().Set("X-RateLimit-Remaining", "1") }, want: "rate-quota:"},
		{name: "reset-drift", allowed: true, change: func(recorder *httptest.ResponseRecorder) { recorder.Header().Set("X-RateLimit-Reset", "1700000061") }, want: "rate-reset:"},
		{name: "missing-retry", allowed: false, change: func(recorder *httptest.ResponseRecorder) { recorder.Header().Del("Retry-After") }, want: "rate-reset:"},
		{name: "spurious-retry", allowed: true, change: func(recorder *httptest.ResponseRecorder) { recorder.Header().Set("Retry-After", "1") }, want: "rate-reset:"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			recorder.Header().Set("X-RateLimit-Limit", "2")
			recorder.Header().Set("X-RateLimit-Remaining", "0")
			recorder.Header().Set("X-RateLimit-Reset", "1700000060")
			window := rateWindow{count: 2, end: end}
			now := end.Add(-time.Second)
			if !test.allowed {
				window.count = 3
				recorder.Code = http.StatusTooManyRequests
				recorder.Header().Set("Retry-After", "1")
			}
			require.NoError(t, checkRateResponse(recorder, 2, window, now))
			test.change(recorder)
			err := checkRateResponse(recorder, 2, window, now)
			require.Error(t, err, "changed observation accepted")
			assert.True(t, strings.HasPrefix(err.Error(), test.want), err.Error())
		})
	}
}

func FuzzRateLimitContract(f *testing.F) { fuzzContract(f, RunRateLimit) }
