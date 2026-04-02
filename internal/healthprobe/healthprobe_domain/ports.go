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

package healthprobe_domain

import (
	"context"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// Probe is a driven port that allows components to be monitored for health.
// Any component or adapter that needs health checking must implement this
// interface.
type Probe interface {
	// Name returns the unique, human-readable name of the component being checked.
	Name() string

	// Check performs the actual health check and returns a status. The context
	// should be used to enforce timeouts.
	//
	// Takes checkType (healthprobe_dto.CheckType) which indicates whether this is
	// a liveness or readiness check, allowing the probe to return different status.
	//
	// Returns healthprobe_dto.Status which indicates the health state of the
	// component.
	Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status
}

// Registry defines a storage interface for health probes. It implements
// healthprobe_domain.Registry as a driven port in hexagonal architecture.
type Registry interface {
	// Register adds a probe to the registry.
	//
	// Takes probe (Probe) which is the probe to add.
	Register(probe Probe)

	// GetAll returns all registered probes.
	GetAll() []Probe
}

// Service is the main entry point for health checks in the domain.
// It runs all registered probes and collects their results.
type Service interface {
	// CheckLiveness runs all liveness health probes at the same time and combines
	// their results into a single status report.
	//
	// Returns healthprobe_dto.Status which contains the combined health status.
	CheckLiveness(ctx context.Context) healthprobe_dto.Status

	// CheckReadiness runs all readiness health probes at the same time and
	// combines their results into a single status report.
	//
	// Returns healthprobe_dto.Status which contains the combined result of all
	// readiness probes.
	CheckReadiness(ctx context.Context) healthprobe_dto.Status
}
