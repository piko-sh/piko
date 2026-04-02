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

package monitoring_adapters

import (
	"context"

	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

var _ monitoring_domain.HealthProbeService = (*HealthProbeAdapter)(nil)

// HealthProbeAdapter adapts healthprobe_domain.Service to
// monitoring_domain.HealthProbeService.
type HealthProbeAdapter struct {
	// service provides health probe checking operations.
	service healthprobe_domain.Service
}

// NewHealthProbeAdapter creates a new HealthProbeAdapter.
//
// Takes service (healthprobe_domain.Service) which is the underlying health
// probe service.
//
// Returns *HealthProbeAdapter which implements
// monitoring_domain.HealthProbeService.
func NewHealthProbeAdapter(service healthprobe_domain.Service) *HealthProbeAdapter {
	return &HealthProbeAdapter{
		service: service,
	}
}

// CheckLiveness runs all liveness health probes.
//
// Returns monitoring_domain.HealthProbeStatus which indicates the current
// liveness state of the service.
func (a *HealthProbeAdapter) CheckLiveness(ctx context.Context) monitoring_domain.HealthProbeStatus {
	status := a.service.CheckLiveness(ctx)
	return convertDTOStatus(status)
}

// CheckReadiness runs all readiness health probes.
//
// Returns monitoring_domain.HealthProbeStatus which indicates the readiness
// state of all probes.
func (a *HealthProbeAdapter) CheckReadiness(ctx context.Context) monitoring_domain.HealthProbeStatus {
	status := a.service.CheckReadiness(ctx)
	return convertDTOStatus(status)
}

// convertDTOStatus converts healthprobe_dto.Status to
// monitoring_domain.HealthProbeStatus.
//
// Takes status (healthprobe_dto.Status) which is the DTO to convert.
//
// Returns monitoring_domain.HealthProbeStatus which is the domain model
// representation.
func convertDTOStatus(status healthprobe_dto.Status) monitoring_domain.HealthProbeStatus {
	deps := make([]monitoring_domain.HealthProbeStatus, len(status.Dependencies))
	for i, dependency := range status.Dependencies {
		if dependency != nil {
			deps[i] = convertDTOStatus(*dependency)
		}
	}

	return monitoring_domain.HealthProbeStatus{
		Name:         status.Name,
		State:        string(status.State),
		Message:      status.Message,
		Timestamp:    status.Timestamp.UnixMilli(),
		Duration:     status.Duration,
		Dependencies: deps,
	}
}
