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

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/security/security_domain"
)

const (
	// requestIDHeader is the HTTP header name used to get a request ID.
	requestIDHeader = "X-Request-Id"
)

// RealIPMiddleware extracts the client IP address and generates a request ID,
// storing both on the PikoRequestCtx carrier already present in the context.
//
// This middleware combines the functionality of chi's RealIP and RequestID
// middlewares while being more secure (only trusts forwarding headers from
// configured proxies) and zero-alloc (mutates the existing PikoRequestCtx
// pointer rather than creating new context values).
//
// The extracted data is:
//  1. Stored on PikoRequestCtx (accessible via
//     security_dto.ClientIPFromRequest
//     and security_dto.RequestIDFromRequest)
//  2. r.RemoteAddr is set to the client IP for compatibility
//
// This middleware should be placed early in the middleware chain so that
// downstream handlers and other middleware can access the correct client IP and
// request ID.
type RealIPMiddleware struct {
	// extractor extracts the client IP address from incoming requests.
	extractor security_domain.ClientIPExtractor
}

// NewRealIPMiddleware creates a new RealIP middleware with the given IP
// extractor.
//
// Takes extractor (security_domain.ClientIPExtractor) which extracts the
// client IP address from incoming requests.
//
// Returns *RealIPMiddleware which is ready for use as HTTP middleware.
func NewRealIPMiddleware(extractor security_domain.ClientIPExtractor) *RealIPMiddleware {
	return &RealIPMiddleware{
		extractor: extractor,
	}
}

// Handler returns the middleware handler function for use with chi or other
// routers.
//
// Takes next (http.Handler) which is the next handler in the chain.
//
// Returns http.Handler which wraps the next handler with IP and request ID
// extraction.
func (m *RealIPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pctx := daemon_dto.PikoRequestCtxFromContext(r.Context())
		if pctx == nil {
			pctx = &daemon_dto.PikoRequestCtx{}
			r = r.WithContext(daemon_dto.WithPikoRequestCtx(r.Context(), pctx))
		}

		remoteIP, _ := parseRemoteAddrWithIP(r.RemoteAddr)
		pctx.FromTrustedProxy = m.extractor.IsTrustedProxy(remoteIP)
		pctx.ClientIP = m.extractor.ExtractClientIP(r)

		if pctx.FromTrustedProxy {
			if fwd := r.Header.Get(requestIDHeader); fwd != "" {
				pctx.ForwardedRequestID = fwd
			}
		}

		if pctx.RequestIDCounter == 0 && pctx.ForwardedRequestID == "" {
			pctx.RequestIDCounter = daemon_dto.NextRequestIDCounter()
		}

		r.RemoteAddr = pctx.ClientIP

		next.ServeHTTP(w, r)
	})
}
