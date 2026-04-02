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

// Package healthprobe_domain defines port interfaces (Probe, Registry,
// Service) and the service that runs all registered probes concurrently,
// aggregating their results into a unified health status.
//
// # Health check types
//
// The package supports two standard Kubernetes-style health check types:
//
//   - Liveness: Determines if the application is running correctly
//   - Readiness: Determines if the application is ready to accept traffic
//
// # Thread safety
//
// The service executes all probes concurrently using goroutines with
// individual timeouts. Probe implementations must be safe for concurrent use.
package healthprobe_domain
