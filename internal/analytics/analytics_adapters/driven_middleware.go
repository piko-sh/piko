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

package analytics_adapters

import (
	"net/http"
	"time"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/runeutil"
)

const (
	// maxFieldLength is the upper bound (in runes) for attacker-influenced string fields
	// (URL, User-Agent, Referer) to prevent oversized events.
	maxFieldLength = 2048
)

// AnalyticsMiddleware captures HTTP request/response data and fires page view events to
// the analytics service.
type AnalyticsMiddleware struct {
	// service distributes tracked events to all registered collectors.
	service *analytics_domain.Service
}

// NewAnalyticsMiddleware creates middleware that fires events to the given analytics
// service.
//
// Takes service (*analytics_domain.Service) which receives tracked events.
//
// Returns *AnalyticsMiddleware which is the configured middleware.
func NewAnalyticsMiddleware(
	service *analytics_domain.Service,
) *AnalyticsMiddleware {
	return &AnalyticsMiddleware{
		service: service,
	}
}

// Handler wraps the downstream handler and records a page view after it returns. The
// event includes the response status and duration from PikoRequestCtx.
//
// The event is built from copies and never holds the live *http.Request, and its
// properties and revenue are copied from the carrier, so the asynchronous collectors
// never share state with the request or its pooled carrier.
//
// Takes next (http.Handler) which is the downstream handler.
//
// Returns http.Handler which wraps next with analytics tracking.
func (m *AnalyticsMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pctx := daemon_dto.PikoRequestCtxFromContext(r.Context())
		if pctx == nil {
			next.ServeHTTP(w, r)
			return
		}

		pctx.ResponseWriter = w
		pctx.Hostname = r.Host
		start := time.Now()

		next.ServeHTTP(pctx, r)

		ev := newPageEvent(r, pctx, start)
		m.service.Track(r.Context(), ev)
	})
}

// newPageEvent builds the analytics event for a finished request from copies of the
// request and carrier values.
//
// Takes r (*http.Request) which is the finished request.
// Takes pctx (*daemon_dto.PikoRequestCtx) which carries the response status and the
// values stashed by handlers.
// Takes start (time.Time) which is when the request started.
//
// Returns *analytics_dto.Event which is a pooled event owned by the caller.
func newPageEvent(r *http.Request, pctx *daemon_dto.PikoRequestCtx, start time.Time) *analytics_dto.Event {
	statusCode := pctx.ResponseStatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	stashed := pctx.Analytics()

	ev := analytics_dto.AcquireEvent()
	ev.Hostname = r.Host
	ev.URL = runeutil.Truncate(r.RequestURI, maxFieldLength)
	ev.Path = r.URL.Path
	ev.Method = r.Method
	ev.UserAgent = runeutil.Truncate(r.UserAgent(), maxFieldLength)
	ev.Referrer = runeutil.Truncate(r.Referer(), maxFieldLength)
	ev.Timestamp = start
	ev.Duration = time.Since(start)
	ev.StatusCode = statusCode
	ev.Type = analytics_dto.EventPageView
	ev.ClientIP = pctx.ClientIP
	ev.Locale = pctx.Locale
	ev.MatchedPattern = pctx.MatchedPattern
	ev.Revenue = stashed.Revenue
	ev.Properties = stashed.Properties

	if stashed.ActionName != "" {
		ev.ActionName = stashed.ActionName
		ev.Type = analytics_dto.EventAction
	}

	if stashed.EventName != "" {
		ev.EventName = stashed.EventName
		ev.Type = analytics_dto.EventCustom
	}

	if auth, ok := pctx.CachedAuth.(daemon_dto.AuthContext); ok && auth.IsAuthenticated() {
		ev.UserID = auth.UserID()
	}

	return ev
}
