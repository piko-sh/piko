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

// Package events_provider_gochannel provides an in-memory Watermill
// pub/sub provider using Go channels. This is the default provider
// for single-instance deployments and requires no external
// infrastructure.
//
// # Configuration
//
// The GoChannel provider is configured with sensible defaults:
//
//	config := events_provider_gochannel.DefaultConfig()
//	provider, err := events_provider_gochannel.NewGoChannelProvider(config)
//
// # Back-pressure
//
// By default, BlockPublishUntilSubscriberAck is true. This means:
//   - Publish() blocks until all subscribers have acknowledged
//   - No messages are ever lost (unlike fire-and-forget mode)
//   - May reduce throughput for high-volume scenarios
//
// To disable back-pressure (fire-and-forget mode):
//
//	config := events_provider_gochannel.DefaultConfig()
//	config.BlockPublishUntilSubscriberAck = false
//	provider, err := events_provider_gochannel.NewGoChannelProvider(config)
//
// # Usage with Piko
//
// The GoChannel provider is used by default when no provider is
// configured:
//
//	app := piko.New() // Uses GoChannel by default
//
// To customise the GoChannel configuration:
//
//	config := events_provider_gochannel.DefaultConfig()
//	config.OutputChannelBuffer = 2048
//	provider, err := events_provider_gochannel.NewGoChannelProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if err := provider.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	app := piko.New(
//	    piko.WithEventsProvider(provider),
//	)
//
// # When to use
//
// GoChannel is ideal for:
//   - Development and testing
//   - Single-instance deployments
//   - Scenarios where message persistence is not required
//   - High-throughput local processing
//
// For distributed systems or durability requirements, consider:
//   - events_provider_nats: NATS JetStream
//   - events_provider_postgres: PostgreSQL
//   - events_provider_sqlite: SQLite
//
// # Thread safety
//
// All methods on the provider are safe for concurrent use. The
// provider uses internal mutexes to guard lifecycle state.
package events_provider_gochannel
