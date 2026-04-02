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

// Package healthprobe_dto defines the types used to represent health
// check requests and responses, including the hierarchical status
// structure that supports nested dependency reporting.
//
// # Check types
//
// Two standard Kubernetes-style health checks are supported:
//
//   - Liveness: Determines if the application is running and not deadlocked.
//     Failure typically triggers a restart.
//   - Readiness: Determines if the application can serve traffic.
//     Failure temporarily removes the instance from the load balancer.
package healthprobe_dto
