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

package render_domain

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// Name returns the orchestrator's identifier for health probe registration.
// Implements the healthprobe_domain.Probe interface.
//
// Returns string which is the constant "RenderService".
func (*RenderOrchestrator) Name() string {
	return "RenderService"
}

// Check implements the healthprobe_domain.Probe interface.
// It verifies the rendering pipeline is functional.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which indicates the health state of the
// rendering pipeline.
func (ro *RenderOrchestrator) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()
	if checkType == healthprobe_dto.CheckTypeLiveness {
		return ro.checkLiveness(startTime)
	}
	return ro.checkReadiness(ctx, checkType, startTime)
}

// checkLiveness performs a liveness health check.
//
// Takes startTime (time.Time) which marks when the health check began.
//
// Returns healthprobe_dto.Status which contains the current health state.
func (ro *RenderOrchestrator) checkLiveness(startTime time.Time) healthprobe_dto.Status {
	state, message := healthprobe_dto.StateHealthy, "Render service is running"
	if ro.pmlEngine == nil {
		state, message = healthprobe_dto.StateUnhealthy, "PML engine is not initialised"
	} else if ro.registry == nil {
		state, message = healthprobe_dto.StateUnhealthy, "Registry adapter is not initialised"
	}
	return healthprobe_dto.Status{
		Name:         ro.Name(),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: nil,
	}
}

// checkReadiness performs a readiness health check including dependencies.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to perform.
// Takes startTime (time.Time) which marks when the check began for duration
// calculation.
//
// Returns healthprobe_dto.Status which contains the overall health state,
// message, and dependency statuses.
func (ro *RenderOrchestrator) checkReadiness(ctx context.Context, checkType healthprobe_dto.CheckType, startTime time.Time) healthprobe_dto.Status {
	dependencies := make([]*healthprobe_dto.Status, 0)
	overallState := healthprobe_dto.StateHealthy

	overallState = ro.checkRegistryHealth(ctx, checkType, &dependencies, overallState)
	ro.addTransformPipelineStatus(&dependencies)

	message := "Render service operational"
	if overallState != healthprobe_dto.StateHealthy {
		message = "Render service has dependency issues"
	}

	return healthprobe_dto.Status{
		Name:         ro.Name(),
		State:        overallState,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: dependencies,
	}
}

// checkRegistryHealth checks if the registry implements the health probe
// interface and collects its status.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to perform.
// Takes dependencies (*[]*healthprobe_dto.Status) which accumulates the
// collected health statuses.
// Takes currentState (healthprobe_dto.State) which is the current aggregate
// health state.
//
// Returns healthprobe_dto.State which is the updated health state based on
// the registry check result.
func (ro *RenderOrchestrator) checkRegistryHealth(
	ctx context.Context,
	checkType healthprobe_dto.CheckType,
	dependencies *[]*healthprobe_dto.Status,
	currentState healthprobe_dto.State,
) healthprobe_dto.State {
	probe, ok := ro.registry.(interface {
		Name() string
		Check(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status
	})
	if !ok {
		return currentState
	}

	registryStatus := probe.Check(ctx, checkType)
	*dependencies = append(*dependencies, &registryStatus)

	if registryStatus.State == healthprobe_dto.StateUnhealthy {
		return healthprobe_dto.StateUnhealthy
	}
	if registryStatus.State == healthprobe_dto.StateDegraded && currentState == healthprobe_dto.StateHealthy {
		return healthprobe_dto.StateDegraded
	}
	return currentState
}

// addTransformPipelineStatus adds the transform pipeline status to the list.
//
// Takes dependencies (*[]*healthprobe_dto.Status) which receives the new
// status entry for the transform pipeline.
func (ro *RenderOrchestrator) addTransformPipelineStatus(dependencies *[]*healthprobe_dto.Status) {
	*dependencies = append(*dependencies, &healthprobe_dto.Status{
		Name:         "TransformPipeline",
		State:        healthprobe_dto.StateHealthy,
		Message:      fmt.Sprintf("%d transformation step(s) registered", len(ro.transformSteps)),
		Timestamp:    time.Time{},
		Duration:     "",
		Dependencies: nil,
	})
}
