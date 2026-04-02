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

// Package provider_grpc implements TUI data providers that communicate
// with the Piko monitoring gRPC server.
//
// It fulfils the [tui_domain] provider port interfaces using gRPC
// clients generated from the monitoring API protobuf definitions,
// replacing direct database access to enable remote monitoring
// scenarios such as kubectl port-forward. Each provider caches its
// data locally and refreshes at a configurable interval, communicating
// with HealthService, MetricsService, OrchestratorInspectorService,
// and RegistryInspectorService.
//
// # Usage
//
// Use [NewProviders] to create all providers with a shared connection:
//
//	providers, err := provider_grpc.NewProviders(
//	    "localhost:9091",
//	    provider_grpc.WithDialTimeout(3*time.Second),
//	    provider_grpc.WithRefreshInterval(5*time.Second),
//	)
//
// # Thread safety
//
// All provider methods are safe for concurrent use. Each provider
// guards its cached state with a [sync.RWMutex] so that concurrent
// reads proceed freely whilst writes are serialised during refresh cycles.
package provider_grpc
