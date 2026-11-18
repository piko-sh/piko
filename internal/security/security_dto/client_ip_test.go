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

package security_dto

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/daemon/daemon_dto"
)

func TestPikoRequestCtxRoundTrip_Forwarded(t *testing.T) {
	t.Parallel()

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
		ClientIP:           "192.168.1.1",
		ForwardedRequestID: "request-123",
		FromTrustedProxy:   true,
	})

	if got := ClientIPFromContext(ctx); got != "192.168.1.1" {
		t.Errorf("ClientIPFromContext = %q, want %q", got, "192.168.1.1")
	}
	if got := RequestIDFromContext(ctx); got != "request-123" {
		t.Errorf("RequestIDFromContext = %q, want %q", got, "request-123")
	}
	if !FromTrustedProxyFromContext(ctx) {
		t.Error("FromTrustedProxyFromContext should be true for forwarded context")
	}
}

func TestPikoRequestCtxRoundTrip_Generated(t *testing.T) {
	t.Parallel()

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
		ClientIP:         "10.0.0.1",
		RequestIDCounter: 42,
	})

	if got := ClientIPFromContext(ctx); got != "10.0.0.1" {
		t.Errorf("ClientIPFromContext = %q, want %q", got, "10.0.0.1")
	}

	id := RequestIDFromContext(ctx)
	if id == "" {
		t.Error("RequestIDFromContext should not be empty for a generated ID")
	}
	if FromTrustedProxyFromContext(ctx) {
		t.Error("FromTrustedProxyFromContext should be false when not set")
	}
}

func TestClientIPFromContext_Missing(t *testing.T) {
	t.Parallel()

	if got := ClientIPFromContext(context.Background()); got != "" {
		t.Errorf("ClientIPFromContext(empty) = %q, want empty", got)
	}
}

func TestClientIPFromContext(t *testing.T) {
	t.Parallel()

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
	})
	if got := ClientIPFromContext(ctx); got != "10.0.0.1" {
		t.Errorf("ClientIPFromContext = %q, want %q", got, "10.0.0.1")
	}

	if got := ClientIPFromContext(context.Background()); got != "" {
		t.Errorf("ClientIPFromContext(empty) = %q, want empty", got)
	}
}

func TestClientIPFromRequest(t *testing.T) {
	t.Parallel()

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
		ClientIP: "172.16.0.1",
	})
	request, _ := http.NewRequestWithContext(ctx, "GET", "/", nil)
	if got := ClientIPFromRequest(request); got != "172.16.0.1" {
		t.Errorf("ClientIPFromRequest = %q, want %q", got, "172.16.0.1")
	}
}

func TestFromTrustedProxyFromContext(t *testing.T) {
	t.Parallel()

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
		FromTrustedProxy: true,
	})
	if !FromTrustedProxyFromContext(ctx) {
		t.Error("should be true")
	}

	if FromTrustedProxyFromContext(context.Background()) {
		t.Error("should be false for empty context")
	}
}

func TestRequestIDFromContext_Forwarded(t *testing.T) {
	t.Parallel()

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
		ForwardedRequestID: "abc-definition",
	})
	if got := RequestIDFromContext(ctx); got != "abc-definition" {
		t.Errorf("RequestIDFromContext = %q, want %q", got, "abc-definition")
	}

	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("RequestIDFromContext(empty) = %q, want empty", got)
	}
}

func TestRequestIDFromContext_Generated(t *testing.T) {
	t.Parallel()

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
		RequestIDCounter: 7,
	})
	got := RequestIDFromContext(ctx)
	if got == "" {
		t.Error("RequestIDFromContext should not be empty for a generated ID")
	}
}

func TestRequestIDFromRequest(t *testing.T) {
	t.Parallel()

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
		ForwardedRequestID: "xyz-789",
	})
	request, _ := http.NewRequestWithContext(ctx, "GET", "/", nil)
	if got := RequestIDFromRequest(request); got != "xyz-789" {
		t.Errorf("RequestIDFromRequest = %q, want %q", got, "xyz-789")
	}
}

func TestRateLimitClientAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		clientIP   string
		remoteAddr string
		expected   string
	}{
		{name: "resolved client IP wins", clientIP: "198.51.100.4", remoteAddr: "10.0.0.1:4321", expected: "198.51.100.4"},
		{name: "IPv4 peer loses its port", remoteAddr: "203.0.113.5:4321", expected: "203.0.113.5"},
		{name: "IPv6 peer loses its port", remoteAddr: "[2001:db8::1]:443", expected: "2001:db8::1"},
		{name: "peer without a port is kept", remoteAddr: "203.0.113.5", expected: "203.0.113.5"},
		{name: "padded peer is trimmed", remoteAddr: "  203.0.113.5:80  ", expected: "203.0.113.5"},
		{name: "port without a host is unknown", remoteAddr: ":8080", expected: ""},
		{name: "blank peer is unknown", remoteAddr: "   ", expected: ""},
		{name: "no peer is unknown", remoteAddr: "", expected: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = testCase.remoteAddr
			if testCase.clientIP != "" {
				request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
					ClientIP: testCase.clientIP,
				}))
			}

			assert.Equal(t, testCase.expected, RateLimitClientAddress(request))
		})
	}
}

func TestRateLimitClientAddressGivesEachPortOfOnePeerOneKey(t *testing.T) {
	t.Parallel()

	first := httptest.NewRequest(http.MethodGet, "/", nil)
	first.RemoteAddr = "203.0.113.5:1111"
	second := httptest.NewRequest(http.MethodGet, "/", nil)
	second.RemoteAddr = "203.0.113.5:2222"

	assert.Equal(t, SanitiseRateLimitKey(RateLimitClientAddress(first)), SanitiseRateLimitKey(RateLimitClientAddress(second)),
		"one peer must not get a fresh bucket for every source port")
}
