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

package bootstrap

import (
	"context"
	"sync/atomic"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
)

var (
	// activeAnalyticsService points at the analytics service of the most recently started
	// container, for package-level helpers that have no container reference.
	activeAnalyticsService atomic.Pointer[analytics_domain.Service]
)

// AddAnalyticsCollector registers a backend analytics collector.
//
// If the collector implements a shutdown interface (Close, Shutdown, or Stop), it will be
// automatically registered for graceful shutdown.
//
// Takes collector (analytics_domain.Collector) which handles event delivery.
func (c *Container) AddAnalyticsCollector(collector analytics_domain.Collector) {
	if collector == nil {
		_, l := logger_domain.From(c.GetAppContext(), log)
		l.Warn("Nil analytics collector ignored")
		return
	}
	c.analyticsCollectors = append(c.analyticsCollectors, collector)
	registerCloseableForShutdown(c.GetAppContext(), "AnalyticsCollector-"+collector.Name(), collector)
}

// GetAnalyticsService returns the analytics service, creating it lazily on first call.
//
// When no collectors are registered, the service is not created and the analytics
// middleware is not installed.
//
// Returns *analytics_domain.Service which distributes events to collectors, or nil when
// analytics is not enabled.
func (c *Container) GetAnalyticsService() *analytics_domain.Service {
	c.analyticsOnce.Do(func() {
		if len(c.analyticsCollectors) == 0 {
			return
		}

		ctx, l := logger_domain.From(c.GetAppContext(), log)

		service := analytics_domain.NewService(c.analyticsCollectors)
		service.Start(ctx)
		c.analyticsService = service
		activeAnalyticsService.Store(service)

		shutdown.Register(ctx, "analytics-service", func(ctx context.Context) error {
			activeAnalyticsService.CompareAndSwap(service, nil)
			return service.Close(ctx)
		})

		l.Internal("Backend analytics service initialised",
			logger_domain.Int("collector_count", len(c.analyticsCollectors)))
	})
	return c.analyticsService
}

// GetGlobalAnalyticsService returns the analytics service of the most recently started
// container without requiring a Container reference.
//
// Each container owns its own service; this accessor exists for package-level helpers.
// Returns nil when no container has started an analytics service, or once that service
// has been shut down.
//
// Returns *analytics_domain.Service which distributes events, or nil.
func GetGlobalAnalyticsService() *analytics_domain.Service {
	return activeAnalyticsService.Load()
}
