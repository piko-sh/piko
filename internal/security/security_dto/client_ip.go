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

	"piko.sh/piko/internal/daemon/daemon_dto"
)

// ClientIPFromContext retrieves the client IP from the PikoRequestCtx carrier
// in the context.
//
// Returns string which is the client IP, or empty if not set.
func ClientIPFromContext(ctx context.Context) string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		return pctx.ClientIP
	}
	return ""
}

// ClientIPFromRequest retrieves the client IP from the request context.
// This is a convenience wrapper around ClientIPFromContext.
//
// Takes r (*http.Request) which provides the request context.
//
// Returns string which is the client IP, or empty if not set.
func ClientIPFromRequest(r *http.Request) string {
	return ClientIPFromContext(r.Context())
}

// FromTrustedProxyFromContext returns whether the request came from a trusted
// proxy CIDR range.
//
// Returns bool which is true if the original RemoteAddr was a trusted proxy.
func FromTrustedProxyFromContext(ctx context.Context) bool {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		return pctx.FromTrustedProxy
	}
	return false
}

// RequestIDFromContext retrieves the request ID from the PikoRequestCtx
// carrier in the context, formatting server-generated IDs lazily on first
// access.
//
// Returns string which is the request ID, or empty if not set.
func RequestIDFromContext(ctx context.Context) string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		return pctx.RequestID()
	}
	return ""
}

// RequestIDFromRequest retrieves the request ID from the request context.
// This is a convenience wrapper around RequestIDFromContext.
//
// Takes r (*http.Request) which provides the request context.
//
// Returns string which is the request ID, or empty if not set.
func RequestIDFromRequest(r *http.Request) string {
	return RequestIDFromContext(r.Context())
}
