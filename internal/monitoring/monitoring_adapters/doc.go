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

// Package monitoring_adapters implements gRPC services and noop factories for
// the monitoring hexagon.
//
// It exposes telemetry data (metrics, traces, health checks, system
// statistics) to external consumers via gRPC, and supplies noop defaults
// for span processing and metrics collection when the OTEL SDK module
// (wdk/logger/logger_otel_sdk) is not imported.
//
// # Integration
//
// Service registration is centralised through [DefaultServiceFactories],
// which returns noop factory functions. For real OTEL SDK factories,
// import wdk/logger/logger_otel_sdk and use
// piko.WithMonitoringOtelFactories().
//
// # Thread safety
//
// All gRPC service methods are safe for concurrent use.
package monitoring_adapters
