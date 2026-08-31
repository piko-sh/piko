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
	"cmp"
	"context"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/maths"
)

// AnalyticsCollector is the interface that backend analytics adapters implement.
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

	// EventCustom is a user-defined event fired manually from action handlers.
	EventCustom = analytics_dto.EventCustom

	// maxAnalyticsProperties is the maximum number of custom properties per event to prevent
	// unbounded map growth.
	maxAnalyticsProperties = 64
)

// TrackAnalyticsEvent sends a custom analytics event to all registered collectors. The
// event is enriched with request context (ClientIP, Locale, UserID, etc.) from the
// PikoRequestCtx if available.
//
// When no collectors are registered, this is a no-op.
//
// Takes event (*AnalyticsEvent) which is the event to send.
func TrackAnalyticsEvent(ctx context.Context, event *AnalyticsEvent) {
	if event == nil {
		return
	}

	service := bootstrap.GetGlobalAnalyticsService()
	if service == nil {
		return
	}

	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		enrichEventFromRequestCtx(event, pctx)
	}

	service.Track(ctx, event)
}

// SetAnalyticsRevenue attaches revenue data to the automatic analytics event for the
// current request.
//
// The middleware copies the value into the pageview event after the handler returns.
// No-op when called outside a request context.
//
// Takes revenue (maths.Money) which is the monetary value to record.
func SetAnalyticsRevenue(ctx context.Context, revenue maths.Money) {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		pctx.SetAnalyticsRevenue(&revenue)
	}
}

// AddAnalyticsProperty attaches a key-value property to the automatic analytics event for
// the current request.
//
// The middleware merges all properties into the event after the handler returns. No-op
// when called outside a request context.
//
// Takes key (string) which is the property name.
// Takes value (string) which is the property value.
func AddAnalyticsProperty(ctx context.Context, key, value string) {
	pctx := daemon_dto.PikoRequestCtxFromContext(ctx)
	if pctx == nil {
		return
	}
	if !pctx.SetAnalyticsProperty(key, value, maxAnalyticsProperties) {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Analytics property dropped; the per-request property limit is full",
			logger_domain.String("property", key),
			logger_domain.Int("limit", maxAnalyticsProperties),
		)
	}
}

// SetAnalyticsEventName changes the automatic analytics event from a page view to a named
// custom event.
//
// When set, the middleware promotes the event type from pageview to custom. No-op when
// called outside a request context.
//
// Takes name (string) which is the event name.
func SetAnalyticsEventName(ctx context.Context, name string) {
	pctx := daemon_dto.PikoRequestCtxFromContext(ctx)
	if pctx == nil {
		return
	}

	clamped, wasClamped := daemon_dto.ClampAnalyticsName(name)
	if wasClamped {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Analytics event name shortened to fit the name limit",
			logger_domain.Int("supplied_bytes", len(name)),
			logger_domain.String("event_name", clamped),
		)
	}
	pctx.SetAnalyticsEvent(clamped)
}

// SetAnalyticsActionName attributes the request's automatic analytics event to a named
// action, promoting it from a page view to an action event.
//
// Takes name (string) which is the action to attribute the event to.
func SetAnalyticsActionName(ctx context.Context, name string) {
	pctx := daemon_dto.PikoRequestCtxFromContext(ctx)
	if pctx == nil {
		return
	}

	clamped, wasClamped := daemon_dto.ClampAnalyticsName(name)
	if wasClamped {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Analytics action name shortened to fit the name limit",
			logger_domain.Int("supplied_bytes", len(name)),
			logger_domain.String("action_name", clamped),
		)
	}
	pctx.SetAnalyticsAction(clamped)
}

// enrichEventFromRequestCtx fills in empty event fields from the per-request carrier so
// that custom events fired from action handlers inherit request-level context
// automatically.
//
// Takes event (*AnalyticsEvent) which is the event to enrich.
// Takes pctx (*daemon_dto.PikoRequestCtx) which provides the request-level values.
func enrichEventFromRequestCtx(event *AnalyticsEvent, pctx *daemon_dto.PikoRequestCtx) {
	event.ClientIP = cmp.Or(event.ClientIP, pctx.ClientIP)
	event.Locale = cmp.Or(event.Locale, pctx.Locale)
	event.MatchedPattern = cmp.Or(event.MatchedPattern, pctx.MatchedPattern)
	event.ActionName = cmp.Or(event.ActionName, pctx.AnalyticsActionName)
	event.Hostname = cmp.Or(event.Hostname, pctx.Hostname)

	if event.UserID == "" {
		if auth, ok := pctx.CachedAuth.(daemon_dto.AuthContext); ok && auth.IsAuthenticated() {
			event.UserID = auth.UserID()
		}
	}
}
