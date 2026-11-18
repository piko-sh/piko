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
	"context"
	"net/http"
	"sync/atomic"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/wdk/useragent"
)

const (
	// requestIDHeader is the HTTP header name used to get a request ID.
	requestIDHeader = "X-Request-Id"
)

// RealIPMiddleware resolves the client IP and stores it with a request ID in the request
// context. It trusts forwarding headers only from configured proxies and sets RemoteAddr
// to the resolved IP.
//
// Place RealIPMiddleware before handlers that use the client IP or request ID.
type RealIPMiddleware struct {
	// extractor extracts the client IP address from incoming requests.
	extractor security_domain.ClientIPExtractor

	// nextCounter returns the next request ID counter value.
	nextCounter func() (uint64, error)

	// counterFailureLogged records that a failure of nextCounter has been logged, so the
	// warning is written once rather than on every request.
	counterFailureLogged atomic.Bool
}

// NewRealIPMiddleware creates a new RealIP middleware with the given IP extractor.
//
// Takes extractor (security_domain.ClientIPExtractor) which extracts the client IP
// address from incoming requests.
//
// Returns *RealIPMiddleware which is ready for use as HTTP middleware.
func NewRealIPMiddleware(extractor security_domain.ClientIPExtractor) *RealIPMiddleware {
	return &RealIPMiddleware{
		extractor:            extractor,
		nextCounter:          daemon_dto.NextRequestIDCounter,
		counterFailureLogged: atomic.Bool{},
	}
}

// Handler returns the middleware handler function for use with chi or other routers.
//
// Takes next (http.Handler) which is the next handler in the chain.
//
// Returns http.Handler which wraps the next handler with IP and request ID extraction.
func (m *RealIPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pctx := daemon_dto.PikoRequestCtxFromContext(r.Context())
		if pctx == nil {
			pctx = &daemon_dto.PikoRequestCtx{}
			r = r.WithContext(daemon_dto.WithPikoRequestCtx(r.Context(), pctx))
		}

		pctx.ClientIP, pctx.FromTrustedProxy = m.extractor.ResolveClient(r)

		clampedAgent, wasClamped := useragent.Clamp(r.UserAgent())
		pctx.UserAgent = clampedAgent
		if wasClamped {
			userAgentClampedCount.Add(r.Context(), 1)
		}

		if pctx.FromTrustedProxy {
			forwarded := r.Header.Get(requestIDHeader)
			accepted, ok := daemon_dto.AcceptForwardedRequestID(forwarded)
			switch {
			case ok:
				pctx.ForwardedRequestID = accepted
			case forwarded != "":

				forwardedRequestIDRejectedCount.Add(r.Context(), 1)
			}
		}

		if pctx.RequestIDCounter == 0 && pctx.ForwardedRequestID == "" {
			pctx.RequestIDCounter = m.nextRequestIDCounter(r.Context())
		}

		r.RemoteAddr = pctx.ClientIP

		next.ServeHTTP(w, r)
	})
}

// nextRequestIDCounter returns the counter value for a server-generated request ID.
//
// When the sequence fails, which should not happen in practice, it logs one warning for
// the life of the middleware and falls back to a random counter value so that requests
// keep getting IDs in the usual format.
//
// Returns uint64 which is never zero.
func (m *RealIPMiddleware) nextRequestIDCounter(ctx context.Context) uint64 {
	counter, err := m.nextCounter()
	if err == nil {
		return counter
	}
	if m.counterFailureLogged.CompareAndSwap(false, true) {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Request ID sequence failed, using random request IDs",
			logger_domain.Error(err))
	}
	return daemon_dto.FallbackRequestIDCounter()
}
