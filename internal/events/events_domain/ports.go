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

package events_domain

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	// defaultCloseTimeoutSeconds is the default timeout in seconds for closing the
	// router.
	defaultCloseTimeoutSeconds = 30
)

// Provider represents a backend-agnostic message bus provider.
//
// Each provider implementation (GoChannel, NATS, PostgreSQL, SQLite) implements
// Provider to supply the Watermill infrastructure components. Provider implements
// the events.Provider and io.Closer interfaces.
//
// Provider lifecycle:
//  1. Create provider with NewXxxProvider(config)
//  2. Call Start(ctx) to initialise and start the router
//  3. Use Router(), Publisher(), Subscriber() to access Watermill components
//  4. Call Close() to shut down gracefully
type Provider interface {
	// Start initialises the provider and starts the Watermill router.
	//
	// This must be called before using Router, Publisher, or Subscriber. The
	// provided context controls the router's lifecycle.
	//
	// Returns error when initialisation or startup fails.
	Start(ctx context.Context) error

	// Router returns the Watermill message router.
	//
	// Returns *message.Router which lets users add their own handlers.
	Router() *message.Router

	// Publisher returns the Watermill Publisher for sending messages.
	Publisher() message.Publisher

	// Subscriber returns the Watermill Subscriber for subscribing to topics.
	//
	// Returns message.Subscriber which handles message subscriptions.
	Subscriber() message.Subscriber

	// Reports whether the router has been started and is still active.
	Running() bool

	// Close shuts down the provider and all its parts in a safe way.
	// It closes the router, publisher, and subscriber in order.
	//
	// Returns error when the shutdown fails.
	Close() error
}

// ProviderConfig holds the base settings shared by all providers.
// Each provider extends this with its own settings.
type ProviderConfig struct {
	// RouterConfig holds settings for the Watermill message router.
	RouterConfig RouterConfig
}

// RouterConfig holds settings for the Watermill Router.
type RouterConfig struct {
	// CloseTimeout is the time in seconds to wait for handlers to finish when
	// closing the router. Default is 30.
	CloseTimeout int64
}

// DefaultProviderConfig returns a ProviderConfig with sensible default values.
//
// Returns ProviderConfig which contains default settings ready for use.
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		RouterConfig: RouterConfig{
			CloseTimeout: defaultCloseTimeoutSeconds,
		},
	}
}
