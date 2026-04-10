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
	"piko.sh/piko/internal/logger/logger_domain"
)

// AnalyticsMiddleware captures HTTP request/response data and fires
// page view events to the analytics service.
type AnalyticsMiddleware struct {
	// service distributes tracked events to all registered collectors.
	service *analytics_domain.Service

	// logger is used for diagnostic messages during event capture.
	logger logger_domain.Logger
}

// NewAnalyticsMiddleware creates middleware that fires events to the
// given analytics service.
//
// Takes service (*analytics_domain.Service) which receives tracked
// events.
// Takes logger (logger_domain.Logger) which is used for diagnostics.
//
// Returns *AnalyticsMiddleware which is the configured middleware.
func NewAnalyticsMiddleware(
	service *analytics_domain.Service,
	logger logger_domain.Logger,
) *AnalyticsMiddleware {
	return &AnalyticsMiddleware{
		service: service,
		logger:  logger,
	}
}

// Handler returns an http.Handler middleware that captures the status
// code and duration via PikoRequestCtx (zero allocations), then fires
// a page view event after the downstream handler returns.
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

		statusCode := pctx.ResponseStatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		ev := analytics_dto.AcquireEvent()
		ev.Request = r
		ev.Hostname = r.Host
		ev.URL = r.URL.String()
		ev.Path = r.URL.Path
		ev.Method = r.Method
		ev.UserAgent = r.UserAgent()
		ev.Referrer = r.Referer()
		ev.Timestamp = start
		ev.Duration = time.Since(start)
		ev.StatusCode = statusCode
		ev.Type = analytics_dto.EventPageView
		ev.ClientIP = pctx.ClientIP
		ev.Locale = pctx.Locale
		ev.MatchedPattern = pctx.MatchedPattern
		ev.Revenue = pctx.AnalyticsRevenue
		ev.Properties = pctx.AnalyticsProperties

		if pctx.AnalyticsEventName != "" {
			ev.EventName = pctx.AnalyticsEventName
			ev.Type = analytics_dto.EventCustom
		}

		if auth, ok := pctx.CachedAuth.(daemon_dto.AuthContext); ok && auth.IsAuthenticated() {
			ev.UserID = auth.UserID()
		}

		m.service.Track(ev)
	})
}
