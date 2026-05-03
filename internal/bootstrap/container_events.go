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

// This file contains event bus and events provider related container methods.

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/events/events_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/wdk/events/events_provider_gochannel"
)

// GetEventBus returns the application event bus for publish/subscribe
// messaging.
//
// Returns orchestrator_domain.EventBus which provides publish/subscribe
// messaging capabilities.
// Returns error when the underlying events provider cannot be initialised.
func (c *Container) GetEventBus() (orchestrator_domain.EventBus, error) {
	c.eventBusOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.eventBusOverride != nil {
			l.Internal("Using provided EventBus override.")
			c.eventBus = c.eventBusOverride
			return
		}

		provider, err := c.GetEventsProvider()
		if err != nil {
			l.Error("Failed to get events provider", logger_domain.Error(err))
			c.eventBusErr = fmt.Errorf("initialising event bus: events provider unavailable: %w", err)
			return
		}

		l.Internal("Creating EventBus using Watermill provider...")
		c.eventBus = orchestrator_adapters.NewWatermillEventBus(
			provider.Publisher(),
			provider.Subscriber(),
			provider.Router(),
		)
	})
	return c.eventBus, c.eventBusErr
}

// GetEventsProvider returns the events infrastructure provider, initialising a
// default GoChannel provider if none was provided. The provider gives access to
// Watermill Router, Publisher, and Subscriber for advanced use cases.
//
// Returns events_domain.Provider which provides event infrastructure access.
// Returns error when the provider could not be created.
func (c *Container) GetEventsProvider() (events_domain.Provider, error) {
	c.eventsProviderOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.eventsProviderOverride != nil {
			l.Internal("Using provided EventsProvider override.")
			c.eventsProvider = c.eventsProviderOverride
			return
		}
		c.createDefaultEventsProvider()
	})
	return c.eventsProvider, c.eventsProviderErr
}

// SetEventsProvider sets a custom events provider implementation.
// This must be called before GetEventBus or GetEventsProvider to take effect.
//
// Takes provider (events_domain.Provider) which is the custom provider to use.
func (c *Container) SetEventsProvider(provider events_domain.Provider) {
	c.eventsProviderOverride = provider
}

// createDefaultEventsProvider sets up the default GoChannel events provider.
func (c *Container) createDefaultEventsProvider() {
	ctx := c.GetAppContext()
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Creating default EventsProvider (GoChannel)...")

	config := events_provider_gochannel.DefaultConfig()
	provider, err := events_provider_gochannel.NewGoChannelProvider(config)
	if err != nil {
		c.eventsProviderErr = fmt.Errorf("failed to create GoChannel provider: %w", err)
		l.Error("Failed to create events provider", logger_domain.Error(err))
		return
	}

	if err := provider.Start(ctx); err != nil {
		if closeErr := provider.Close(); closeErr != nil {
			l.Warn("Failed to close events provider during cleanup", logger_domain.Error(closeErr))
		}
		c.eventsProviderErr = fmt.Errorf("failed to start GoChannel provider: %w", err)
		l.Error("Failed to start events provider", logger_domain.Error(err))
		return
	}

	c.eventsProvider = provider

	shutdown.Register(ctx, "EventsProvider", func(_ context.Context) error {
		return provider.Close()
	})

	l.Internal("Events provider started",
		logger_domain.Bool("blockPublishUntilSubscriberAck", config.BlockPublishUntilSubscriberAck))
}
