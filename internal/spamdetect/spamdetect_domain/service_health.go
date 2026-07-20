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
	"sync"
	"time"

	"piko.sh/piko/wdk/goroutine"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (

	// healthProbeName is the identifier used in the health probe system.
	healthProbeName = "SpamDetectService"
)

var (
	_ interface {
		Name() string
		Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status
	} = (*spamDetectService)(nil)
)

// Name returns the service identifier for the health probe system.
//
// Returns string which is the health probe name.
func (*spamDetectService) Name() string {
	return healthProbeName
}

// Check performs a health check on the spam detection service.
//
// Takes checkType (healthprobe_dto.CheckType) which selects liveness or readiness.
//
// Returns healthprobe_dto.Status which describes the service health.
func (s *spamDetectService) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := s.clock.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return s.checkLiveness(ctx, startTime)
	}

	return s.checkReadiness(ctx, startTime)
}

// checkLiveness returns a liveness probe status.
//
// Takes ctx (context.Context) which is the caller context.
// Takes startTime (time.Time) which marks the check start.
//
// Returns healthprobe_dto.Status which describes the liveness state.
func (s *spamDetectService) checkLiveness(ctx context.Context, startTime time.Time) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "spam detection service operational"

	if !s.IsEnabled(ctx) {
		state = healthprobe_dto.StateDegraded
		message = "no spam detection detectors configured"
	}

	return healthprobe_dto.Status{
		Name:      healthProbeName,
		State:     state,
		Message:   message,
		Timestamp: startTime,
		Duration:  s.clock.Now().Sub(startTime).String(),
	}
}

// checkReadiness returns a readiness probe status including per-detector health.
//
// Takes ctx (context.Context) which is the caller context.
// Takes startTime (time.Time) which marks the check start.
//
// Returns healthprobe_dto.Status which describes the readiness state.
func (s *spamDetectService) checkReadiness(ctx context.Context, startTime time.Time) healthprobe_dto.Status {
	if !s.IsEnabled(ctx) {
		return healthprobe_dto.Status{
			Name:      healthProbeName,
			State:     healthprobe_dto.StateDegraded,
			Message:   "no spam detection detectors configured",
			Timestamp: startTime,
			Duration:  s.clock.Now().Sub(startTime).String(),
		}
	}

	dependencies := s.buildDetectorDependencies(ctx, startTime)

	overallState := healthprobe_dto.StateHealthy
	overallMessage := "spam detection service operational"

	unhealthyCount := 0
	for _, dependency := range dependencies {
		if dependency != nil && dependency.State == healthprobe_dto.StateUnhealthy {
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
		Duration:     s.clock.Now().Sub(startTime).String(),
		Dependencies: dependencies,
	}
}

// buildDetectorDependencies builds health status entries for all detectors. Each
// per-detector probe runs in parallel under a healthCheckTimeout deadline so one slow
// detector cannot stall the readiness response.
//
// Takes ctx (context.Context) which is the caller context.
// Takes startTime (time.Time) which marks the check start.
//
// Returns []*healthprobe_dto.Status which contains per-detector health.
func (s *spamDetectService) buildDetectorDependencies(ctx context.Context, startTime time.Time) []*healthprobe_dto.Status {
	detectors := s.registry.ListProviders(ctx)
	dependencies := make([]*healthprobe_dto.Status, len(detectors))

	var waitGroup sync.WaitGroup
	for index, info := range detectors {
		probeIndex := index
		probeInfo := info
		waitGroup.Go(func() {
			defer goroutine.RecoverPanic(ctx, "spamdetect.buildDetectorDependencies")
			dependencies[probeIndex] = s.probeDetectorHealth(ctx, probeInfo.Name, startTime)
		})
	}
	waitGroup.Wait()

	return dependencies
}

// probeDetectorHealth resolves a single detector and runs its HealthCheck under a bounded
// deadline.
//
// Takes ctx (context.Context) which is the caller context.
// Takes name (string) which identifies the detector.
// Takes startTime (time.Time) which marks the check start.
//
// Returns *healthprobe_dto.Status which describes the detector's liveness in the
// dependency list.
func (s *spamDetectService) probeDetectorHealth(ctx context.Context, name string, startTime time.Time) *healthprobe_dto.Status {
	detector, err := s.registry.GetProvider(ctx, name)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to resolve detector for readiness probe",
			logger_domain.String(attributeKeyDetector, name),
			logger_domain.Error(err),
		)
		return &healthprobe_dto.Status{
			Name:      name,
			State:     healthprobe_dto.StateUnhealthy,
			Message:   fmt.Sprintf("failed to resolve detector: %v", err),
			Timestamp: startTime,
			Duration:  s.clock.Now().Sub(startTime).String(),
		}
	}

	status := &healthprobe_dto.Status{
		Name:      name,
		Timestamp: startTime,
	}

	if detectorErr := s.invokeDetectorHealthCheck(ctx, name, detector); detectorErr != nil {
		status.State = healthprobe_dto.StateUnhealthy
		status.Message = detectorErr.Error()
		status.Duration = s.clock.Now().Sub(startTime).String()
		return status
	}

	status.State = healthprobe_dto.StateHealthy
	status.Message = "detector healthy"
	status.Duration = s.clock.Now().Sub(startTime).String()
	return status
}
