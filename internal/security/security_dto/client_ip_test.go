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
	"testing"

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
