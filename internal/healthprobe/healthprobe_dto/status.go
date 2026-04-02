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

package healthprobe_dto

import "time"

// State represents the health state of a component.
type State string

const (
	// StateHealthy indicates the component is working normally.
	StateHealthy State = "HEALTHY"

	// StateDegraded indicates the component is working but with reduced
	// performance or features.
	StateDegraded State = "DEGRADED"

	// StateUnhealthy indicates the component has failed and cannot work.
	StateUnhealthy State = "UNHEALTHY"
)

// Status is a structured report of a single component's health.
// It can be nested to represent a tree of dependencies.
type Status struct {
	// Name is the component being checked (e.g. "DatabaseConnection", "RedisCache").
	Name string `json:"name"`

	// State indicates the health of the component: healthy, degraded, or unhealthy.
	State State `json:"state"`

	// Message provides extra details about the health check result.
	Message string `json:"message,omitempty"`

	// Timestamp is when the health check was run.
	Timestamp time.Time `json:"timestamp"`

	// Duration is the time taken to complete the check.
	Duration string `json:"duration"`

	// Dependencies holds the health status of each component this one depends on.
	Dependencies []*Status `json:"dependencies,omitempty"`
}
