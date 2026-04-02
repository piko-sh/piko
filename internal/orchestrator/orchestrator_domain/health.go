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

package orchestrator_domain

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// Name returns the service identifier and implements the
// healthprobe_domain.Probe interface.
//
// Returns string which is the service name "OrchestratorService".
func (*orchestratorService) Name() string {
	return "OrchestratorService"
}

// Check implements the healthprobe_domain.Probe interface.
// It verifies the orchestrator is running and can process tasks.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which indicates the health state of the
// orchestrator.
func (s *orchestratorService) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return s.checkLiveness(startTime)
	}

	return s.checkReadiness(ctx, checkType, startTime)
}

// checkLiveness verifies the service is initialised and not stopped.
//
// Takes startTime (time.Time) which marks when the health check began.
//
// Returns healthprobe_dto.Status which contains the liveness state and timing
// information.
//
// Safe for concurrent use. Acquires stopMutex to read the stopped state.
func (s *orchestratorService) checkLiveness(startTime time.Time) healthprobe_dto.Status {
	s.stopMutex.Lock()
	stopped := s.isStopped
	s.stopMutex.Unlock()

	input := healthCheckInput{
		IsStopped:     stopped,
		TaskStoreNil:  s.taskStore == nil,
		ExecutorCount: 0,
	}
	state, message := determineLivenessState(input)

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: nil,
	}
}

// checkReadiness checks that executors are registered and the service is
// ready.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to run.
// Takes startTime (time.Time) which marks when the check began for timing.
//
// Returns healthprobe_dto.Status which holds the readiness state, message,
// and dependency details.
//
// Safe for concurrent use; acquires read locks on executors and stop state.
func (s *orchestratorService) checkReadiness(ctx context.Context, checkType healthprobe_dto.CheckType, startTime time.Time) healthprobe_dto.Status {
	s.executorsMutex.RLock()
	executorCount := len(s.executors)
	s.executorsMutex.RUnlock()

	s.stopMutex.Lock()
	stopped := s.isStopped
	s.stopMutex.Unlock()

	state, message := s.determineReadinessState(stopped, executorCount)
	dependencies := s.buildReadinessDependencies(ctx, checkType, &state, &message)

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: dependencies,
	}
}

// determineReadinessState determines the health state based on service status.
// It delegates to the extracted pure function determineReadinessState for
// testability.
//
// Takes stopped (bool) which indicates whether the service has been stopped.
// Takes executorCount (int) which is the number of active executors.
//
// Returns healthprobe_dto.State which represents the current readiness state.
// Returns string which provides a human-readable status message.
func (*orchestratorService) determineReadinessState(stopped bool, executorCount int) (healthprobe_dto.State, string) {
	input := healthCheckInput{
		IsStopped:     stopped,
		TaskStoreNil:  false,
		ExecutorCount: executorCount,
	}
	return determineReadinessState(input)
}

// buildReadinessDependencies builds the list of dependencies for readiness
// checks.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to perform.
// Takes state (*healthprobe_dto.State) which receives the combined health
// state.
// Takes message (*string) which receives any status message.
//
// Returns []*healthprobe_dto.Status which contains the dependency statuses,
// including active tasks and task store health.
func (s *orchestratorService) buildReadinessDependencies(ctx context.Context, checkType healthprobe_dto.CheckType, state *healthprobe_dto.State, message *string) []*healthprobe_dto.Status {
	var activeTasks int32
	if s.taskDispatcher != nil {
		stats := s.taskDispatcher.Stats()
		activeTasks = stats.ActiveWorkers
	}

	dependencies := []*healthprobe_dto.Status{
		{
			Name:    "Active Tasks",
			State:   healthprobe_dto.StateHealthy,
			Message: fmt.Sprintf("%d task(s) currently processing", activeTasks),
		},
	}

	s.aggregateTaskStoreHealth(ctx, checkType, state, message, &dependencies)
	return dependencies
}

// aggregateTaskStoreHealth checks and aggregates task store health into the
// overall status.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to perform.
// Takes state (*healthprobe_dto.State) which receives the aggregated health
// state.
// Takes message (*string) which receives a status message when unhealthy.
// Takes dependencies (*[]*healthprobe_dto.Status) which collects dependency
// statuses.
func (s *orchestratorService) aggregateTaskStoreHealth(
	ctx context.Context,
	checkType healthprobe_dto.CheckType,
	state *healthprobe_dto.State,
	message *string,
	dependencies *[]*healthprobe_dto.Status,
) {
	probe, ok := s.taskStore.(interface {
		Name() string
		Check(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status
	})
	if !ok {
		return
	}

	storeStatus := probe.Check(ctx, checkType)
	*dependencies = append(*dependencies, &storeStatus)

	if storeStatus.State == healthprobe_dto.StateUnhealthy {
		*state = healthprobe_dto.StateUnhealthy
		*message = "Orchestrator unhealthy: task store unavailable"
	} else if storeStatus.State == healthprobe_dto.StateDegraded && *state == healthprobe_dto.StateHealthy {
		*state = healthprobe_dto.StateDegraded
		*message = "Orchestrator degraded: task store issues"
	}
}
