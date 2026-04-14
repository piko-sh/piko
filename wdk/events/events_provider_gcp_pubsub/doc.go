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

// Package events_provider_gcp_pubsub provides a Google Cloud Pub/Sub-based
// event provider for distributed messaging.
//
// This package implements the [events.Provider] interface using Google
// Cloud Pub/Sub, giving Piko applications access to fully managed
// distributed pub/sub messaging. It supports message persistence,
// at-least-once delivery, and automatic topic/subscription management.
//
// # Configuration
//
// Use [DefaultConfig] to obtain sensible defaults:
//
//	config := events_provider_gcp_pubsub.DefaultConfig()
//	config.ProjectID = "my-gcp-project"
//	provider, err := events_provider_gcp_pubsub.NewGCPPubSubProvider(config)
//
// # Emulator support
//
// For local development, set [Config.EmulatorHost] to use the GCP
// Pub/Sub emulator:
//
//	config := events_provider_gcp_pubsub.DefaultConfig()
//	config.ProjectID = "test-project"
//	config.EmulatorHost = "localhost:8085"
//
// # Usage
//
// Create, start, and register the provider with Piko:
//
//	config := events_provider_gcp_pubsub.DefaultConfig()
//	config.ProjectID = "my-gcp-project"
//	provider, err := events_provider_gcp_pubsub.NewGCPPubSubProvider(config)
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
// # When to use
//
// Google Cloud Pub/Sub is suited for:
//
//   - Deployments on Google Cloud Platform
//   - Scenarios requiring fully managed message persistence
//   - At-least-once delivery guarantees
//   - Applications that benefit from automatic scaling
//
// For single-instance in-memory messaging, consider
// [events_provider_gochannel] instead.
//
// # Thread safety
//
// The provider returned by [NewGCPPubSubProvider] is safe for
// concurrent use after Start has been called.
package events_provider_gcp_pubsub
