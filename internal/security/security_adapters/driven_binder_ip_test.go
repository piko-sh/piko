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

package security_adapters

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

func TestIPBinderAdapter_GetBindingIdentifier(t *testing.T) {
	t.Parallel()

	adapter := NewIPBinderAdapter()

	testCases := []struct {
		name           string
		request        func() *http.Request
		expectedResult string
	}{
		{
			name:           "nil request returns context_no_request",
			request:        func() *http.Request { return nil },
			expectedResult: "context_no_request",
		},
		{
			name: "reads resolved IP from context",
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				ctx := daemon_dto.WithPikoRequestCtx(r.Context(), &daemon_dto.PikoRequestCtx{
					ClientIP: "203.0.113.50",
				})
				return r.WithContext(ctx)
			},
			expectedResult: normaliseIP("203.0.113.50"),
		},
		{
			name: "IPv6 from context",
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				ctx := daemon_dto.WithPikoRequestCtx(r.Context(), &daemon_dto.PikoRequestCtx{
					ClientIP: "2001:db8::1",
				})
				return r.WithContext(ctx)
			},
			expectedResult: "2001:db8::1",
		},
		{
			name: "fallback to RemoteAddr with port when context empty",
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = "192.168.1.1:8080"
				return r
			},
			expectedResult: normaliseIP("192.168.1.1"),
		},
		{
			name: "fallback to RemoteAddr without port when context empty",
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = "192.168.1.1"
				return r
			},
			expectedResult: normaliseIP("192.168.1.1"),
		},
		{
			name: "fallback IPv6 RemoteAddr when context empty",
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = "[::1]:8080"
				return r
			},
			expectedResult: "::1",
		},
		{
			name: "invalid IP in RemoteAddr passthrough when context empty",
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = "not-an-ip"
				return r
			},
			expectedResult: "not-an-ip",
		},
		{
			name: "ignores raw headers uses context instead",
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("CF-Connecting-IP", "1.1.1.1")
				r.Header.Set("X-Real-IP", "2.2.2.2")
				ctx := daemon_dto.WithPikoRequestCtx(r.Context(), &daemon_dto.PikoRequestCtx{
					ClientIP: "203.0.113.50",
				})
				return r.WithContext(ctx)
			},
			expectedResult: normaliseIP("203.0.113.50"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := adapter.GetBindingIdentifier(tc.request())
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestNormaliseIP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "IPv4 returns uint32 string",
			input:    "192.168.1.1",
			expected: "3232235777",
		},
		{
			name:     "IPv4 loopback",
			input:    "127.0.0.1",
			expected: "2130706433",
		},
		{
			name:     "IPv6 loopback returns canonical",
			input:    "::1",
			expected: "::1",
		},
		{
			name:     "IPv6 full address returns canonical",
			input:    "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "invalid IP returns passthrough",
			input:    "not-valid",
			expected: "not-valid",
		},
		{
			name:     "empty string returns passthrough",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := normaliseIP(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
