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

// Package events_provider_sqs provides an AWS SQS-based event provider
// for distributed messaging with managed infrastructure.
//
// This package implements the [events.Provider] interface using AWS SQS,
// giving Piko applications access to fully managed distributed pub/sub
// messaging. SQS provides at-least-once delivery, automatic scaling,
// and requires no server management.
//
// # Configuration
//
// Use [DefaultConfig] to obtain sensible defaults:
//
//	config := events_provider_sqs.DefaultConfig()
//	config.Region = "eu-west-1"
//	provider, err := events_provider_sqs.NewSQSProvider(config)
//
// For local development with LocalStack:
//
//	config := events_provider_sqs.DefaultConfig()
//	config.Region = "us-east-1"
//	config.EndpointURL = "http://localhost:4566"
//	provider, err := events_provider_sqs.NewSQSProvider(config)
//
// # Usage
//
// Create, start, and register the provider with Piko:
//
//	config := events_provider_sqs.DefaultConfig()
//	config.Region = "eu-west-1"
//	provider, err := events_provider_sqs.NewSQSProvider(config)
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
// AWS SQS is suited for:
//
//   - AWS-native deployments
//   - Serverless architectures (Lambda, Fargate)
//   - Scenarios requiring fully managed infrastructure
//   - At-least-once delivery without operational overhead
//
// For single-instance in-memory messaging, consider
// [events_provider_gochannel] instead. For self-hosted distributed
// messaging, consider [events_provider_nats].
//
// # Thread safety
//
// The provider returned by [NewSQSProvider] is safe for concurrent use
// after Start has been called.
package events_provider_sqs
