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

package piko

import (
	"context"

	"piko.sh/piko/internal/analytics/analytics_adapters"
	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/daemon/daemon_dto"
)

// AnalyticsCollector is the interface that backend analytics adapters
// implement.
type AnalyticsCollector = analytics_domain.Collector

// AnalyticsEvent carries the data for a single backend analytics event.
type AnalyticsEvent = analytics_dto.Event

// AnalyticsEventType classifies a backend analytics event.
type AnalyticsEventType = analytics_dto.EventType

const (
	// EventPageView is fired automatically for each page request.
	EventPageView = analytics_dto.EventPageView

	// EventAction is fired when a server action executes.
	EventAction = analytics_dto.EventAction

	// EventCustom is a user-defined event fired manually from action
	// handlers.
	EventCustom = analytics_dto.EventCustom
)

// WebhookCollectorOption configures a [WebhookCollector].
type WebhookCollectorOption = analytics_adapters.WebhookOption

// NewWebhookCollector creates a backend analytics collector that
// batches events and POSTs them as JSON to the given URL. This is
// provided as a built-in adapter to demonstrate the pattern; for
// production use, dedicated adapters for specific analytics backends
// are recommended.
//
// Takes url (string) which is the webhook endpoint.
// Takes opts (...WebhookCollectorOption) which configure the collector.
//
// Returns AnalyticsCollector which posts JSON batches to the URL.
func NewWebhookCollector(url string, opts ...WebhookCollectorOption) AnalyticsCollector {
	return analytics_adapters.NewWebhookCollector(url, opts...)
}

// WithWebhookHeaders sets custom HTTP headers sent with each batch
// POST (e.g. Authorization).
var WithWebhookHeaders = analytics_adapters.WithWebhookHeaders

// WithWebhookBatchSize sets the maximum number of events per batch.
var WithWebhookBatchSize = analytics_adapters.WithWebhookBatchSize

// WithWebhookFlushInterval sets the time between automatic batch
// flushes.
var WithWebhookFlushInterval = analytics_adapters.WithWebhookFlushInterval

// WithWebhookTimeout sets the HTTP client timeout for batch POSTs.
var WithWebhookTimeout = analytics_adapters.WithWebhookTimeout

// TrackAnalyticsEvent sends a custom analytics event to all registered
// collectors. The event is enriched with request context (ClientIP,
// Locale, UserID, etc.) from the PikoRequestCtx if available.
//
// When no collectors are registered, this is a no-op.
//
// Takes event (*AnalyticsEvent) which is the event to send.
func TrackAnalyticsEvent(ctx context.Context, event *AnalyticsEvent) {
	svc := bootstrap.GetGlobalAnalyticsService()
	if svc == nil {
		return
	}

	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		if event.ClientIP == "" {
			event.ClientIP = pctx.ClientIP
		}
		if event.Locale == "" {
			event.Locale = pctx.Locale
		}
		if event.MatchedPattern == "" {
			event.MatchedPattern = pctx.MatchedPattern
		}
		if event.UserID == "" {
			if auth, ok := pctx.CachedAuth.(daemon_dto.AuthContext); ok && auth.IsAuthenticated() {
				event.UserID = auth.UserID()
			}
		}
	}

	svc.Track(event)
}
