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

package analytics_domain

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// healthProbeName is the identifier used when registering with the
// health probe system.
const healthProbeName = "AnalyticsService"

var _ interface {
	Name() string
	Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status
} = (*Service)(nil)

const (
	// channelDegradedNumerator is the numerator of the fraction of
	// channel capacity at which a collector is reported as degraded.
	channelDegradedNumerator = 9

	// channelDegradedDenominator is the denominator for the degraded
	// threshold fraction (numerator/denominator = 90%).
	channelDegradedDenominator = 10
)

// Name returns the service identifier for the health probe system.
//
// Returns string which is the health probe name.
func (*Service) Name() string {
	return healthProbeName
}

// Check performs a health check on the analytics service.
//
// Takes checkType (healthprobe_dto.CheckType) which selects liveness
// or readiness.
//
// Returns healthprobe_dto.Status which describes the service health.
func (s *Service) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return s.checkLiveness(startTime)
	}

	return s.checkReadiness(ctx, startTime)
}

// checkLiveness verifies that the analytics service is running and
// has collectors registered.
//
// Takes startTime (time.Time) which is when the check began.
//
// Returns healthprobe_dto.Status which is healthy when the service
// is running with collectors.
func (s *Service) checkLiveness(startTime time.Time) healthprobe_dto.Status {
	if s.stopped.Load() {
		return healthprobe_dto.Status{
			Name:      healthProbeName,
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "analytics service stopped",
			Timestamp: startTime,
			Duration:  time.Since(startTime).String(),
		}
	}

	if len(s.workers) == 0 {
		return healthprobe_dto.Status{
			Name:      healthProbeName,
			State:     healthprobe_dto.StateDegraded,
			Message:   "no analytics collectors configured",
			Timestamp: startTime,
			Duration:  time.Since(startTime).String(),
		}
	}

	return healthprobe_dto.Status{
		Name:      healthProbeName,
		State:     healthprobe_dto.StateHealthy,
		Message:   "analytics service operational",
		Timestamp: startTime,
		Duration:  time.Since(startTime).String(),
	}
}

// checkReadiness verifies that each collector can reach its backend
// and keep up with the event rate.
//
// Takes startTime (time.Time) which is when the check began.
//
// Returns healthprobe_dto.Status which reports per-collector health
// as dependencies.
func (s *Service) checkReadiness(ctx context.Context, startTime time.Time) healthprobe_dto.Status {
	if s.stopped.Load() {
		return healthprobe_dto.Status{
			Name:      healthProbeName,
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "analytics service stopped",
			Timestamp: startTime,
			Duration:  time.Since(startTime).String(),
		}
	}

	if len(s.workers) == 0 {
		return healthprobe_dto.Status{
			Name:      healthProbeName,
			State:     healthprobe_dto.StateDegraded,
			Message:   "no analytics collectors configured",
			Timestamp: startTime,
			Duration:  time.Since(startTime).String(),
		}
	}

	dependencies, worstState := s.buildCollectorDependencies(ctx, startTime)

	overallMessage := "analytics service operational"
	if worstState == healthprobe_dto.StateDegraded {
		overallMessage = "analytics collectors degraded"
	}

	return healthprobe_dto.Status{
		Name:         healthProbeName,
		State:        worstState,
		Message:      overallMessage,
		Timestamp:    startTime,
		Duration:     time.Since(startTime).String(),
		Dependencies: dependencies,
	}
}

// buildCollectorDependencies calls HealthCheck on each collector and
// inspects channel occupancy, returning sorted dependency statuses
// alongside the worst observed state.
//
// Takes startTime (time.Time) which is when the check began.
//
// Returns []*healthprobe_dto.Status which holds per-collector health.
// Returns healthprobe_dto.State which is the worst state observed.
func (s *Service) buildCollectorDependencies(ctx context.Context, startTime time.Time) ([]*healthprobe_dto.Status, healthprobe_dto.State) {
	dependencies := make([]*healthprobe_dto.Status, len(s.workers))
	worstState := healthprobe_dto.StateHealthy

	for i := range s.workers {
		w := &s.workers[i]

		state := healthprobe_dto.StateHealthy
		message := "collector operational"

		if healthError := w.collector.HealthCheck(ctx); healthError != nil {
			state = healthprobe_dto.StateUnhealthy
			message = healthError.Error()
		} else {
			channelUsage := len(w.eventCh)
			channelCapacity := cap(w.eventCh)
			if channelCapacity > 0 && channelUsage*channelDegradedDenominator >= channelCapacity*channelDegradedNumerator {
				state = healthprobe_dto.StateDegraded
				message = fmt.Sprintf("event channel near capacity (%d/%d)", channelUsage, channelCapacity)
			}
		}

		if state == healthprobe_dto.StateUnhealthy {
			worstState = healthprobe_dto.StateUnhealthy
		} else if state == healthprobe_dto.StateDegraded && worstState == healthprobe_dto.StateHealthy {
			worstState = healthprobe_dto.StateDegraded
		}

		dependencies[i] = &healthprobe_dto.Status{
			Name:      w.collector.Name(),
			State:     state,
			Message:   message,
			Timestamp: startTime,
			Duration:  time.Since(startTime).String(),
		}
	}

	slices.SortFunc(dependencies, func(a, b *healthprobe_dto.Status) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return dependencies, worstState
}
