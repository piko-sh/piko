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

package events

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/events/events_domain"
	watermilllogger "piko.sh/piko/internal/logger/logger_adapters/integrations/watermill"
	"piko.sh/piko/wdk/logger"
)

// Provider is a backend-agnostic message bus provider.
// Each provider (GoChannel, NATS, PostgreSQL, SQLite) implements this
// interface to supply the Watermill infrastructure components.
type Provider = events_domain.Provider

// ProviderConfig is the base settings shared by all providers.
type ProviderConfig = events_domain.ProviderConfig

// RouterConfig holds settings for the Watermill Router.
type RouterConfig = events_domain.RouterConfig

// GetRouter returns the Watermill Router configured by Piko.
// Users can add their own handlers to this router for custom event
// processing.
//
// Example:
//
//	router, err := events.GetRouter()
//	if err != nil {
//	    return err
//	}
//	subscriber, err := events.GetSubscriber()
//	if err != nil {
//	    return err
//	}
//	router.AddNoPublisherHandler(
//	    "order-processor",
//	    "orders.created",
//	    subscriber,
//	    func(msg *message.Message) error {
//	        // Process the order
//	        return nil
//	    },
//	)
//
// Returns *message.Router which is the configured events router.
// Returns error when the framework is not initialised or the provider cannot
// be created.
func GetRouter() (*message.Router, error) {
	return bootstrap.GetEventsRouter()
}

// GetPublisher returns the Watermill Publisher configured by Piko. Use this to
// publish your own messages to the event bus.
//
// Example:
//
//	publisher, err := events.GetPublisher()
//	if err != nil {
//	    return err
//	}
//	wmMessage := message.NewMessage(watermill.NewUUID(), []byte(`{"order_id": "123"}`))
//	err = publisher.Publish("orders.created", wmMessage)
//
// Returns message.Publisher which is the configured events publisher.
// Returns error when the framework is not initialised or the provider cannot be
// created.
func GetPublisher() (message.Publisher, error) {
	return bootstrap.GetEventsPublisher()
}

// GetSubscriber returns the Watermill Subscriber configured by Piko. Use this
// to create subscriptions in combination with GetRouter().
//
// Example:
//
//	router, _ := events.GetRouter()
//	subscriber, _ := events.GetSubscriber()
//	router.AddNoPublisherHandler("handler-name", "topic", subscriber, handler)
//
// Returns message.Subscriber which is the configured events subscriber.
// Returns error when the framework is not initialised or the provider cannot be
// created.
func GetSubscriber() (message.Subscriber, error) {
	return bootstrap.GetEventsSubscriber()
}

// GetProvider returns the underlying events provider for advanced use cases.
// Most users should use GetRouter, GetPublisher, and GetSubscriber instead.
//
// Returns Provider which is the configured events provider.
// Returns error when the framework is not initialised or the provider cannot
// be created.
func GetProvider() (Provider, error) {
	return bootstrap.GetEventsProvider()
}

// IsRunning reports whether the events router has been started and is running.
// This can be used to check if the events infrastructure is ready before
// adding handlers or publishing messages.
//
// Returns bool which is true if the events router is running.
func IsRunning() bool {
	return bootstrap.IsEventsRunning()
}

// DefaultProviderConfig returns sensible defaults for provider configuration.
//
// Returns ProviderConfig which contains the default settings for a provider.
func DefaultProviderConfig() ProviderConfig {
	return events_domain.DefaultProviderConfig()
}

// NewWatermillLoggerAdapter creates a Watermill LoggerAdapter from a Piko
// logger. Provider implementations use this to integrate with Watermill's
// logging system, bridging Piko's structured logging to Watermill's internal
// log calls.
//
// Takes l (logger.Logger) which is the Piko logger to adapt.
//
// Returns watermill.LoggerAdapter which wraps the Piko logger for use with
// Watermill components.
func NewWatermillLoggerAdapter(l logger.Logger) watermill.LoggerAdapter {
	return watermilllogger.NewAdapter(l)
}
