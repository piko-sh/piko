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
	"sync"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
)

var (
	// analyticsService is the lazily-created singleton.
	analyticsService *analytics_domain.Service

	// analyticsServiceOnce guards single initialisation.
	analyticsServiceOnce sync.Once
)

// AddAnalyticsCollector registers a backend analytics collector.
//
// If the collector implements a shutdown interface (Close, Shutdown,
// or Stop), it will be automatically registered for graceful shutdown.
//
// Takes collector (analytics_domain.Collector) which handles event
// delivery.
func (c *Container) AddAnalyticsCollector(collector analytics_domain.Collector) {
	c.analyticsCollectors = append(c.analyticsCollectors, collector)
	registerCloseableForShutdown(c.GetAppContext(), "AnalyticsCollector-"+collector.Name(), collector)
}

// GetAnalyticsService returns the analytics service, creating it
// lazily on first call. Returns nil when no collectors are registered,
// which causes the analytics middleware to not be installed.
//
// Returns *analytics_domain.Service which distributes events to
// collectors, or nil when analytics is not enabled.
func (c *Container) GetAnalyticsService() *analytics_domain.Service {
	analyticsServiceOnce.Do(func() {
		if len(c.analyticsCollectors) == 0 {
			return
		}

		_, l := logger_domain.From(c.GetAppContext(), log)

		analyticsService = analytics_domain.NewService(c.analyticsCollectors)
		analyticsService.Start(c.GetAppContext())

		shutdown.Register(c.GetAppContext(), "analytics-service", func(ctx context.Context) error {
			return analyticsService.Close(ctx)
		})

		l.Internal("Backend analytics service initialised",
			logger_domain.Int("collector_count", len(c.analyticsCollectors)))
	})
	return analyticsService
}

// GetGlobalAnalyticsService returns the analytics service singleton
// without requiring a Container reference. Returns nil when no
// collectors are registered or the service has not been initialised.
//
// Returns *analytics_domain.Service which distributes events, or nil.
func GetGlobalAnalyticsService() *analytics_domain.Service {
	return analyticsService
}
