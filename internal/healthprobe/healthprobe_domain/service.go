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
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

// ServiceOption is a function that configures a healthprobe service.
type ServiceOption func(*service)

// service provides health status checks for the application.
// It implements healthprobe_domain.Service.
type service struct {
	// registry stores the registered health probes and email providers.
	registry Registry

	// clock provides time operations; defaults to RealClock().
	clock clock.Clock

	// applicationName is the name shown in health check status responses.
	applicationName string

	// checkTimeout is the maximum time allowed for each health probe check.
	checkTimeout time.Duration

	// shuttingDown indicates that the application is draining.
	shuttingDown atomic.Bool
}

// CheckLiveness runs all liveness health probes and aggregates their results.
//
// Returns healthprobe_dto.Status which indicates the overall liveness state.
func (s *service) CheckLiveness(ctx context.Context) healthprobe_dto.Status {
	return s.checkAll(ctx, healthprobe_dto.CheckTypeLiveness, "Liveness")
}

// CheckReadiness runs all readiness health probes and aggregates their
// results. If the service has been signalled to drain, it returns
// StateUnhealthy immediately without running probes, allowing load
// balancers to deregister the instance before the server shuts down.
//
// Returns healthprobe_dto.Status which contains the combined status of all
// readiness probes.
func (s *service) CheckReadiness(ctx context.Context) healthprobe_dto.Status {
	if s.shuttingDown.Load() {
		return healthprobe_dto.Status{
			Name:      s.applicationName,
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "Application is shutting down",
			Timestamp: s.clock.Now(),
			Duration:  "0s",
		}
	}
	return s.checkAll(ctx, healthprobe_dto.CheckTypeReadiness, "Readiness")
}

// SignalDrain marks the service as draining so that readiness checks
// return StateUnhealthy immediately.
func (s *service) SignalDrain() {
	s.shuttingDown.Store(true)
	DrainSignalledCount.Add(context.Background(), 1)
}

// checkAll runs all health checks and gathers the results.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies which type of
// health check to run.
// Takes checkName (string) which sets the name shown in status messages.
//
// Returns healthprobe_dto.Status which holds the overall health state and
// the status of each dependency.
func (s *service) checkAll(ctx context.Context, checkType healthprobe_dto.CheckType, checkName string) healthprobe_dto.Status {
	startTime := s.clock.Now()
	probes := s.registry.GetAll()

	statuses := s.executeProbesConcurrently(ctx, probes, checkType)
	aggregatedState, dependencyStatuses := aggregateProbeResults(statuses, len(probes))

	message := checkName + " check completed"
	if aggregatedState != healthprobe_dto.StateHealthy {
		message = checkName + " check found issues"
	}

	return healthprobe_dto.Status{
		Name:         s.applicationName,
		State:        aggregatedState,
		Message:      message,
		Timestamp:    s.clock.Now(),
		Duration:     s.clock.Now().Sub(startTime).String(),
		Dependencies: dependencyStatuses,
	}
}

// executeProbesConcurrently runs all probes simultaneously with
// individual timeouts, spawning one task per probe via errgroup.
//
// Takes probes ([]Probe) which contains the health probes to
// execute.
// Takes checkType (healthprobe_dto.CheckType) which specifies the
// type of health check to run.
//
// Returns chan healthprobe_dto.Status which delivers results from
// all probes once they complete.
func (s *service) executeProbesConcurrently(ctx context.Context, probes []Probe, checkType healthprobe_dto.CheckType) chan healthprobe_dto.Status {
	statuses := make(chan healthprobe_dto.Status, len(probes))
	g, gCtx := errgroup.WithContext(ctx)

	for _, p := range probes {
		probe := p
		g.Go(func() error {
			probeCtx, cancel := context.WithTimeoutCause(gCtx, s.checkTimeout,
				fmt.Errorf("health check exceeded %s timeout", s.checkTimeout))
			defer cancel()

			status := probe.Check(probeCtx, checkType)

			if probeCtx.Err() != nil {
				status.State = healthprobe_dto.StateUnhealthy
				status.Message = "Health check timed out"
			}

			statuses <- status
			return nil
		})
	}

	if waitError := g.Wait(); waitError != nil {
		_, warningLogger := logger_domain.From(ctx, nil)
		warningLogger.Warn("health probe errgroup returned an error",
			logger_domain.Error(waitError))
	}
	close(statuses)
	return statuses
}

// WithClock sets the clock used for timestamps and duration measurements.
// When not provided, the service defaults to clock.RealClock().
//
// Takes clk (clock.Clock) which provides time operations.
//
// Returns ServiceOption which configures the clock on the service.
func WithClock(clk clock.Clock) ServiceOption {
	return func(s *service) {
		s.clock = clk
	}
}

// NewService creates a new healthprobe service with the provided registry.
//
// Takes registry (Registry) which provides health check registration.
// Takes checkTimeout (time.Duration) which sets the timeout for health checks.
// Takes applicationName (string) which identifies the application in probes.
// Takes opts (...ServiceOption) which provides optional configuration.
//
// Returns Service which is the configured healthprobe service ready for use.
func NewService(registry Registry, checkTimeout time.Duration, applicationName string, opts ...ServiceOption) Service {
	s := &service{
		registry:        registry,
		checkTimeout:    checkTimeout,
		applicationName: applicationName,
		clock:           clock.RealClock(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// aggregateProbeResults collects probe statuses and determines the overall
// health state.
//
// Takes statuses (chan healthprobe_dto.Status) which provides probe results
// to collect.
// Takes probeCount (int) which specifies the expected number of results for
// pre-allocation.
//
// Returns healthprobe_dto.State which is the aggregated health state based on
// all probe results.
// Returns []*healthprobe_dto.Status which contains all collected probe
// statuses.
func aggregateProbeResults(statuses chan healthprobe_dto.Status, probeCount int) (healthprobe_dto.State, []*healthprobe_dto.Status) {
	aggregatedState := healthprobe_dto.StateHealthy
	dependencyStatuses := make([]*healthprobe_dto.Status, 0, probeCount)

	for status := range statuses {
		dependencyStatuses = append(dependencyStatuses, new(status))

		if status.State == healthprobe_dto.StateUnhealthy {
			aggregatedState = healthprobe_dto.StateUnhealthy
		} else if status.State == healthprobe_dto.StateDegraded && aggregatedState == healthprobe_dto.StateHealthy {
			aggregatedState = healthprobe_dto.StateDegraded
		}
	}

	slices.SortFunc(dependencyStatuses, func(a, b *healthprobe_dto.Status) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return aggregatedState, dependencyStatuses
}
