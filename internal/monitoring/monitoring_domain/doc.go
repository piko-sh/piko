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

// Package monitoring_domain defines the core abstractions and business
// logic for the monitoring subsystem.
//
// It defines port interfaces for telemetry collection, system statistics,
// and file descriptor tracking, along with in-memory storage for metrics
// and trace spans. The [Service] manages the lifecycle of collectors, the
// [TelemetryStore], and the gRPC server that exposes this data to external
// consumers such as the TUI dashboard.
//
// # Integration
//
// The [Service] integrates with the OpenTelemetry SDK by providing a
// [sdktrace.SpanProcessor] and [sdkmetric.Reader] that should be
// registered with the application's trace and meter providers. It also
// accepts inspectors from the orchestrator and registry hexagons via
// [Service.SetInspectors] for exposing their state through gRPC.
//
// # Thread safety
//
// [TelemetryStore], [SystemCollector], and [ResourceCollector]
// are all safe for concurrent use. The [Service] protects its mutable
// inspector fields with a read-write mutex.
package monitoring_domain
