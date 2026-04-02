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

package collection_domain

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// Name returns the service identifier and implements the
// healthprobe_domain.Probe interface.
//
// Returns string which is the service name "CollectionService".
func (*collectionService) Name() string {
	return "CollectionService"
}

// Check implements the healthprobe_domain.Probe interface.
// It verifies that collection providers are registered and optionally checks
// their health.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which indicates the health status of the
// collection service.
func (s *collectionService) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := s.clock.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return s.checkLiveness(startTime)
	}

	return s.checkReadiness(ctx, checkType, startTime)
}

// checkLiveness performs a simple liveness check.
//
// Takes startTime (time.Time) which records when the check began.
//
// Returns healthprobe_dto.Status which contains the health state and details.
func (s *collectionService) checkLiveness(startTime time.Time) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "Collection service is running"

	if s.registry == nil {
		state = healthprobe_dto.StateUnhealthy
		message = "Provider registry is not initialised"
	}

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        state,
		Message:      message,
		Timestamp:    s.clock.Now(),
		Duration:     s.clock.Now().Sub(startTime).String(),
		Dependencies: nil,
	}
}

// checkReadiness performs a readiness check including provider health.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to perform.
// Takes startTime (time.Time) which marks when the check began for duration
// calculation.
//
// Returns healthprobe_dto.Status which contains the readiness state, message,
// and any provider dependency statuses.
func (s *collectionService) checkReadiness(ctx context.Context, checkType healthprobe_dto.CheckType, startTime time.Time) healthprobe_dto.Status {
	providerNames := s.registry.List()

	if len(providerNames) == 0 {
		return healthprobe_dto.Status{
			Name:         s.Name(),
			State:        healthprobe_dto.StateHealthy,
			Message:      "No collection providers configured",
			Timestamp:    s.clock.Now(),
			Duration:     s.clock.Now().Sub(startTime).String(),
			Dependencies: nil,
		}
	}

	dependencies, overallState := s.checkProviderHealth(ctx, checkType, providerNames)

	message := fmt.Sprintf("Collection service ready with %d provider(s)", len(providerNames))
	if overallState != healthprobe_dto.StateHealthy {
		message = "Collection service has provider issues"
	}

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        overallState,
		Message:      message,
		Timestamp:    s.clock.Now(),
		Duration:     s.clock.Now().Sub(startTime).String(),
		Dependencies: dependencies,
	}
}

// checkProviderHealth checks the health of each provider and combines the
// results.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to run.
// Takes providerNames ([]string) which lists the providers to check.
//
// Returns []*healthprobe_dto.Status which contains the health status of each
// checked provider.
// Returns healthprobe_dto.State which is the combined overall health state.
func (s *collectionService) checkProviderHealth(
	ctx context.Context,
	checkType healthprobe_dto.CheckType,
	providerNames []string,
) ([]*healthprobe_dto.Status, healthprobe_dto.State) {
	dependencies := make([]*healthprobe_dto.Status, 0, len(providerNames))
	overallState := healthprobe_dto.StateHealthy

	for _, name := range providerNames {
		provider, ok := s.registry.Get(name)
		if !ok {
			continue
		}

		status := getProviderStatus(ctx, checkType, name, provider)
		dependencies = append(dependencies, status)
		overallState = aggregateState(overallState, status.State)
	}

	return dependencies, overallState
}

// getProviderStatus gets the health status for a single provider.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to run.
// Takes name (string) which identifies the provider in status messages.
// Takes provider (CollectionProvider) which is checked for health status.
//
// Returns *healthprobe_dto.Status which contains the provider's health state.
func getProviderStatus(
	ctx context.Context,
	checkType healthprobe_dto.CheckType,
	name string,
	provider CollectionProvider,
) *healthprobe_dto.Status {
	probe, ok := provider.(interface {
		Name() string
		Check(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status
	})
	if !ok {
		return &healthprobe_dto.Status{
			Name:         fmt.Sprintf("%s (Provider)", name),
			State:        healthprobe_dto.StateHealthy,
			Message:      "Provider does not support health checks (skipped)",
			Timestamp:    time.Time{},
			Duration:     "",
			Dependencies: nil,
		}
	}

	return new(probe.Check(ctx, checkType))
}

// aggregateState returns the worse of two health states.
//
// Takes current (healthprobe_dto.State) which is the existing combined state.
// Takes incoming (healthprobe_dto.State) which is the new state to compare.
//
// Returns healthprobe_dto.State which is the more serious of the two states.
func aggregateState(current, incoming healthprobe_dto.State) healthprobe_dto.State {
	if incoming == healthprobe_dto.StateUnhealthy {
		return healthprobe_dto.StateUnhealthy
	}
	if incoming == healthprobe_dto.StateDegraded && current == healthprobe_dto.StateHealthy {
		return healthprobe_dto.StateDegraded
	}
	return current
}
