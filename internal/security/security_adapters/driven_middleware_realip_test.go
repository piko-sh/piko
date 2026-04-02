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
	"piko.sh/piko/internal/security/security_dto"
)

func testRequestWithPikoCtx() *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := daemon_dto.WithPikoRequestCtx(r.Context(), &daemon_dto.PikoRequestCtx{})
	return r.WithContext(ctx)
}

func TestRealIPMiddleware_Handler_SetsClientIPInContext(t *testing.T) {
	t.Parallel()

	extractor := &mockClientIPExtractor{
		extractFunc: func(_ *http.Request) string { return "1.2.3.4" },
	}
	mw := NewRealIPMiddleware(extractor)

	var capturedIP string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedIP = security_dto.ClientIPFromRequest(r)
	})

	w := httptest.NewRecorder()
	r := testRequestWithPikoCtx()

	mw.Handler(next).ServeHTTP(w, r)

	assert.Equal(t, "1.2.3.4", capturedIP)
}

func TestRealIPMiddleware_Handler_SetsRemoteAddr(t *testing.T) {
	t.Parallel()

	extractor := &mockClientIPExtractor{
		extractFunc: func(_ *http.Request) string { return "1.2.3.4" },
	}
	mw := NewRealIPMiddleware(extractor)

	var capturedRemoteAddr string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedRemoteAddr = r.RemoteAddr
	})

	w := httptest.NewRecorder()
	r := testRequestWithPikoCtx()

	mw.Handler(next).ServeHTTP(w, r)

	assert.Equal(t, "1.2.3.4", capturedRemoteAddr)
}

func TestRealIPMiddleware_Handler_GeneratesRequestID(t *testing.T) {
	t.Parallel()

	extractor := &mockClientIPExtractor{
		extractFunc: func(_ *http.Request) string { return "1.2.3.4" },
	}
	mw := NewRealIPMiddleware(extractor)

	var capturedRequestID string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedRequestID = security_dto.RequestIDFromRequest(r)
	})

	w := httptest.NewRecorder()
	r := testRequestWithPikoCtx()

	mw.Handler(next).ServeHTTP(w, r)

	require.NotEmpty(t, capturedRequestID)
	assert.Contains(t, capturedRequestID, "/", "generated ID should contain hostname/prefix separator")
	assert.Contains(t, capturedRequestID, "-", "generated ID should contain prefix-counter separator")
}

func TestRealIPMiddleware_Handler_PreservesExistingRequestID_FromTrustedProxy(t *testing.T) {
	t.Parallel()

	extractor := &mockClientIPExtractor{
		extractFunc:   func(_ *http.Request) string { return "1.2.3.4" },
		isTrustedFunc: func(_ string) bool { return true },
	}
	mw := NewRealIPMiddleware(extractor)

	var capturedRequestID string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedRequestID = security_dto.RequestIDFromRequest(r)
	})

	w := httptest.NewRecorder()
	r := testRequestWithPikoCtx()
	r.Header.Set("X-Request-Id", "existing-request-id")

	mw.Handler(next).ServeHTTP(w, r)

	assert.Equal(t, "existing-request-id", capturedRequestID)
}

func TestRealIPMiddleware_Handler_IgnoresRequestID_FromUntrustedClient(t *testing.T) {
	t.Parallel()

	extractor := &mockClientIPExtractor{
		extractFunc:   func(_ *http.Request) string { return "1.2.3.4" },
		isTrustedFunc: func(_ string) bool { return false },
	}
	mw := NewRealIPMiddleware(extractor)

	var capturedRequestID string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedRequestID = security_dto.RequestIDFromRequest(r)
	})

	w := httptest.NewRecorder()
	r := testRequestWithPikoCtx()
	r.Header.Set("X-Request-Id", "injected-by-attacker")

	mw.Handler(next).ServeHTTP(w, r)

	assert.NotEqual(t, "injected-by-attacker", capturedRequestID)
	require.NotEmpty(t, capturedRequestID)
	assert.Contains(t, capturedRequestID, "/", "should be a server-generated ID")
}

func TestRealIPMiddleware_Handler_SetsFromTrustedProxy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		isTrusted       bool
		expectedTrusted bool
	}{
		{
			name:            "trusted proxy sets FromTrustedProxy true",
			isTrusted:       true,
			expectedTrusted: true,
		},
		{
			name:            "untrusted client sets FromTrustedProxy false",
			isTrusted:       false,
			expectedTrusted: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			extractor := &mockClientIPExtractor{
				extractFunc:   func(_ *http.Request) string { return "1.2.3.4" },
				isTrustedFunc: func(_ string) bool { return tc.isTrusted },
			}
			mw := NewRealIPMiddleware(extractor)

			var captured bool
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				captured = security_dto.FromTrustedProxyFromContext(r.Context())
			})

			w := httptest.NewRecorder()
			r := testRequestWithPikoCtx()
			mw.Handler(next).ServeHTTP(w, r)

			assert.Equal(t, tc.expectedTrusted, captured)
		})
	}
}

func TestRealIPMiddleware_Handler_RequestIDIncrements(t *testing.T) {
	t.Parallel()

	extractor := &mockClientIPExtractor{
		extractFunc: func(_ *http.Request) string { return "1.2.3.4" },
	}
	mw := NewRealIPMiddleware(extractor)

	var id1, id2 string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if id1 == "" {
			id1 = security_dto.RequestIDFromRequest(r)
		} else {
			id2 = security_dto.RequestIDFromRequest(r)
		}
	})

	handler := mw.Handler(next)

	r1 := testRequestWithPikoCtx()
	r2 := testRequestWithPikoCtx()
	handler.ServeHTTP(httptest.NewRecorder(), r1)
	handler.ServeHTTP(httptest.NewRecorder(), r2)

	require.NotEmpty(t, id1)
	require.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "sequential requests should have different request IDs")
}

func TestRealIPMiddleware_Handler_CallsNext(t *testing.T) {
	t.Parallel()

	extractor := &mockClientIPExtractor{}
	mw := NewRealIPMiddleware(extractor)

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})

	w := httptest.NewRecorder()
	r := testRequestWithPikoCtx()

	mw.Handler(next).ServeHTTP(w, r)
	assert.True(t, called)
}
