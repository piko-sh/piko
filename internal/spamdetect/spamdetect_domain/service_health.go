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

package spamdetect_domain

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// healthProbeName is the identifier used in the health probe system.
const healthProbeName = "SpamDetectService"

var _ interface {
	Name() string
	Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status
} = (*spamDetectService)(nil)

// Name returns the service identifier for the health probe system.
//
// Returns string which is the health probe name.
func (*spamDetectService) Name() string {
	return healthProbeName
}

// Check performs a health check on the spam detection service.
//
// Takes checkType (healthprobe_dto.CheckType) which selects liveness or
// readiness.
//
// Returns healthprobe_dto.Status which describes the service health.
func (s *spamDetectService) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return s.checkLiveness(startTime)
	}

	return s.checkReadiness(ctx, startTime)
}

// checkLiveness returns a liveness probe status.
//
// Takes startTime (time.Time) which marks the check start.
//
// Returns healthprobe_dto.Status which describes the liveness state.
func (s *spamDetectService) checkLiveness(startTime time.Time) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "spam detection service operational"

	if !s.IsEnabled() {
		state = healthprobe_dto.StateDegraded
		message = "no spam detection detectors configured"
	}

	return healthprobe_dto.Status{
		Name:      healthProbeName,
		State:     state,
		Message:   message,
		Timestamp: startTime,
		Duration:  time.Since(startTime).String(),
	}
}

// checkReadiness returns a readiness probe status including
// per-detector health.
//
// Takes startTime (time.Time) which marks the check start.
//
// Returns healthprobe_dto.Status which describes the readiness state.
func (s *spamDetectService) checkReadiness(ctx context.Context, startTime time.Time) healthprobe_dto.Status {
	if !s.IsEnabled() {
		return healthprobe_dto.Status{
			Name:      healthProbeName,
			State:     healthprobe_dto.StateDegraded,
			Message:   "no spam detection detectors configured",
			Timestamp: startTime,
			Duration:  time.Since(startTime).String(),
		}
	}

	dependencies := s.buildDetectorDependencies(ctx, startTime)

	overallState := healthprobe_dto.StateHealthy
	overallMessage := "spam detection service operational"

	unhealthyCount := 0
	for _, dependency := range dependencies {
		if dependency.State == healthprobe_dto.StateUnhealthy {
			unhealthyCount++
		}
	}

	if unhealthyCount > 0 {
		overallState = healthprobe_dto.StateUnhealthy
		overallMessage = fmt.Sprintf("%d detector(s) unhealthy", unhealthyCount)
	}

	return healthprobe_dto.Status{
		Name:         healthProbeName,
		State:        overallState,
		Message:      overallMessage,
		Timestamp:    startTime,
		Duration:     time.Since(startTime).String(),
		Dependencies: dependencies,
	}
}

// buildDetectorDependencies builds health status entries for all
// detectors.
//
// Takes startTime (time.Time) which marks the check start.
//
// Returns []*healthprobe_dto.Status which contains per-detector health.
func (s *spamDetectService) buildDetectorDependencies(ctx context.Context, startTime time.Time) []*healthprobe_dto.Status {
	detectors := s.registry.ListProviders(ctx)
	dependencies := make([]*healthprobe_dto.Status, 0, len(detectors))

	for _, info := range detectors {
		detector, err := s.registry.GetProvider(ctx, info.Name)
		if err != nil {
			dependencies = append(dependencies, &healthprobe_dto.Status{
				Name:      info.Name,
				State:     healthprobe_dto.StateUnhealthy,
				Message:   fmt.Sprintf("failed to resolve detector: %v", err),
				Timestamp: startTime,
				Duration:  time.Since(startTime).String(),
			})
			continue
		}

		detectorStatus := &healthprobe_dto.Status{
			Name:      info.Name,
			Timestamp: startTime,
			Duration:  time.Since(startTime).String(),
		}

		if detectorErr := detector.HealthCheck(ctx); detectorErr != nil {
			detectorStatus.State = healthprobe_dto.StateUnhealthy
			detectorStatus.Message = detectorErr.Error()
		} else {
			detectorStatus.State = healthprobe_dto.StateHealthy
			detectorStatus.Message = "detector healthy"
		}

		dependencies = append(dependencies, detectorStatus)
	}

	return dependencies
}
