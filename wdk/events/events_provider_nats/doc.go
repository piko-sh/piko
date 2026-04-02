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

// Package events_provider_nats provides a NATS JetStream-based event
// provider for distributed messaging with persistence.
//
// This package implements the [events.Provider] interface using NATS
// JetStream, giving Piko applications access to distributed pub/sub
// messaging. It supports message persistence, at-least-once delivery,
// durable subscriptions, and competing consumer patterns via queue
// groups.
//
// # Configuration
//
// Use [DefaultConfig] to obtain sensible defaults with
// JetStream enabled:
//
//	config := events_provider_nats.DefaultConfig()
//	config.URL = "nats://nats-server:4222"
//	provider, err := events_provider_nats.NewNATSProvider(config)
//
// To disable JetStream and use core NATS (at-most-once delivery):
//
//	config := events_provider_nats.DefaultConfig()
//	config.JetStream.Disabled = true
//
// # Usage
//
// Create, start, and register the provider with Piko:
//
//	config := events_provider_nats.DefaultConfig()
//	config.URL = "nats://nats-server:4222"
//	provider, err := events_provider_nats.NewNATSProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if err := provider.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	app := piko.New(
//	    piko.WithEventsProvider(provider),
//	)
//
// # Queue groups
//
// Enable competing consumers by setting a queue group prefix
// and subscriber count:
//
//	config := events_provider_nats.DefaultConfig()
//	config.QueueGroupPrefix = "my-app"
//	config.SubscribersCount = 4
//
// # When to use
//
// NATS JetStream is suited for:
//
//   - Distributed deployments with multiple instances
//   - Scenarios requiring message persistence
//   - At-least-once or exactly-once delivery guarantees
//   - High-throughput distributed messaging
//
// For single-instance in-memory messaging, consider
// [events_provider_gochannel] instead.
//
// # Thread safety
//
// The provider returned by [NewNATSProvider] is safe for
// concurrent use after Start has been called.
package events_provider_nats
