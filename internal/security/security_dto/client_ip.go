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
	"net"
	"net/http"
	"strings"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

// ClientIPFromContext retrieves the client IP from the PikoRequestCtx carrier in the
// context.
//
// Returns string which is the client IP, or empty if not set.
func ClientIPFromContext(ctx context.Context) string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		return pctx.ClientIP
	}
	return ""
}

// ClientIPFromRequest retrieves the client IP from the request context. This is a
// convenience wrapper around ClientIPFromContext.
//
// Takes r (*http.Request) which provides the request context.
//
// Returns string which is the client IP, or empty if not set.
func ClientIPFromRequest(r *http.Request) string {
	return ClientIPFromContext(r.Context())
}

// FromTrustedProxyFromContext returns whether the request came from a trusted proxy CIDR
// range.
//
// Returns bool which is true if the original RemoteAddr was a trusted proxy.
func FromTrustedProxyFromContext(ctx context.Context) bool {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		return pctx.FromTrustedProxy
	}
	return false
}

// RequestIDFromContext retrieves the request ID from the PikoRequestCtx carrier in the
// context, formatting server-generated IDs lazily on first access.
//
// Returns string which is the request ID, or empty if not set.
func RequestIDFromContext(ctx context.Context) string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		return pctx.RequestID()
	}
	return ""
}

// RequestIDFromRequest retrieves the request ID from the request context. This is a
// convenience wrapper around RequestIDFromContext.
//
// Takes r (*http.Request) which provides the request context.
//
// Returns string which is the request ID, or empty if not set.
func RequestIDFromRequest(r *http.Request) string {
	return RequestIDFromContext(r.Context())
}

// RateLimitClientAddress finds the address used to count a request against a rate limit.
// It uses the resolved client IP or, when that is empty, the host from RemoteAddr without
// its port so source port changes do not create new buckets.
//
// An empty result means the address is unknown. A rate limiter must refuse the request
// because an empty key would put every unknown peer into one bucket.
//
// Takes r (*http.Request) which supplies the client IP and peer address.
//
// Returns string which is the address, or empty when it is unknown.
func RateLimitClientAddress(r *http.Request) string {
	if clientIP := ClientIPFromRequest(r); clientIP != "" {
		return clientIP
	}
	return remoteAddrHost(r.RemoteAddr)
}

// remoteAddrHost returns the host part of a peer address.
//
// Takes remoteAddr (string) which is a host:port pair or a bare host.
//
// Returns string which is the parsed host, or the trimmed input when it is not a
// host:port pair.
func remoteAddrHost(remoteAddr string) string {
	trimmed := strings.TrimSpace(remoteAddr)
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		return host
	}
	return trimmed
}
