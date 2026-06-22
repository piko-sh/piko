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

package readiness

import (
	"context"
)

// State is the normalised health state of a component, mirroring the three states piko's
// health-probe subsystem reports.
type State string

const (
	// StateHealthy indicates the component is working normally.
	StateHealthy State = "HEALTHY"

	// StateDegraded indicates the component is working but with reduced performance or
	// limited features.
	StateDegraded State = "DEGRADED"

	// StateUnhealthy indicates the component has failed and cannot serve.
	StateUnhealthy State = "UNHEALTHY"
)

// IsValid reports whether s is one of the three recognised health states, letting
// producers normalise unknown values before populating a Snapshot.
func (s State) IsValid() bool {
	switch s {
	case StateHealthy, StateDegraded, StateUnhealthy:
		return true
	default:
		return false
	}
}

// Dependency is one child of the readiness root: a single dependency piko checks on its
// way to declaring itself ready (e.g. a database, cache, or downstream service).
type Dependency struct {
	// Name identifies the dependency (e.g. "RegistryDatabase", "OrchestratorService").
	Name string

	// State is the dependency's current health state.
	State State

	// Message carries any human-readable detail (often empty when healthy).
	Message string

	// Duration is the time the dependency's check took, formatted as a Go duration string
	// (e.g. "1.2ms", "500us", "0s"); empty when the check reported no timing.
	Duration string

	// Info carries provider-specific detail for this dependency, flattened to plain
	// (Section, Key, Value) strings and empty for dependencies that map to no provider.
	Info []InfoEntry
}

// InfoEntry is one provider-specific detail attached to a Dependency: a single (Section,
// Key, Value) triple drawn from the provider's describe view. All fields are plain
// strings, so the DTO leaks no internal type (the same discipline as Snapshot).
type InfoEntry struct {
	// Section is the title of the provider detail section this entry came from (e.g.
	// "Overview", "Configuration", "Engine Diagnostics").
	Section string

	// Key is the label within the section (e.g. "Host", "database_size").
	Key string

	// Value is the display value for the key (e.g. "localhost", "142 MiB").
	Value string
}

// Snapshot is one readiness sample: the root status plus its immediate dependency
// children. It is the public mirror of piko's internal readiness tree, flattened to a
// single level (the children the Dependencies screen renders).
type Snapshot struct {
	// Name identifies the readiness root (the overall readiness probe).
	Name string

	// State is the aggregated readiness state of the whole application.
	State State

	// Message carries any human-readable detail about the aggregate state.
	Message string

	// Duration is the time the overall readiness check took, formatted as a Go duration
	// string.
	Duration string

	// Dependencies are the immediate children of the readiness root.
	Dependencies []Dependency
}

// Probe is the minimal public seam a readiness sampler needs: it returns the current
// readiness Snapshot. SSRServer.HealthProbe returns a Probe backed by piko's internal
// monitoring health-probe service; a telemetry collector samples it without ever naming
// an internal type.
type Probe interface {
	// CheckReadiness runs piko's readiness checks and returns a flattened Snapshot.
	CheckReadiness(ctx context.Context) Snapshot
}
