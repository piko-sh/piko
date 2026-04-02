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

package events_provider_gochannel

import (
	"context"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// Name returns the provider's identifier string.
//
// Implements the healthprobe_domain.Probe interface.
//
// Returns string which is the constant "GoChannelProvider".
func (*GoChannelProvider) Name() string {
	return "GoChannelProvider"
}

// Check verifies the GoChannel provider is running and operational.
// It implements the healthprobe_domain.Probe interface.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// perform a liveness or readiness check.
//
// Returns healthprobe_dto.Status which indicates the current health state.
func (p *GoChannelProvider) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return p.checkLiveness(startTime)
	}

	return p.checkReadiness(ctx, startTime)
}

// checkLiveness checks whether the provider is set up correctly.
//
// Takes startTime (time.Time) which marks when the health check began.
//
// Returns healthprobe_dto.Status which contains the liveness state and timing.
func (p *GoChannelProvider) checkLiveness(startTime time.Time) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "GoChannel provider is initialised"

	if p.pubsub == nil {
		state = healthprobe_dto.StateUnhealthy
		message = "GoChannel pub/sub is not initialised"
	}

	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     state,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// checkReadiness checks if the provider is running and ready to serve.
// It confirms that the router has started and is running.
//
// Takes startTime (time.Time) which records when the check began.
//
// Returns healthprobe_dto.Status which holds the readiness state and health
// details for any dependencies.
//
// Safe for concurrent use; acquires a read lock on runningMutex.
func (p *GoChannelProvider) checkReadiness(_ context.Context, startTime time.Time) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "GoChannel provider is running"

	p.runningMutex.RLock()
	running := p.running
	p.runningMutex.RUnlock()

	if !running {
		state = healthprobe_dto.StateUnhealthy
		message = "GoChannel provider is not running"
	}

	if p.router == nil {
		state = healthprobe_dto.StateUnhealthy
		message = "Watermill router is not initialised"
	}

	dependencies := make([]*healthprobe_dto.Status, 0, 2)

	dependencies = append(dependencies,
		new(p.checkRouterHealth(startTime)),
		new(p.checkPubSubHealth(startTime)),
	)

	for _, dependency := range dependencies {
		if dependency.State == healthprobe_dto.StateUnhealthy {
			state = healthprobe_dto.StateUnhealthy
			message = "GoChannel provider has unhealthy dependencies"
			break
		}
		if dependency.State == healthprobe_dto.StateDegraded && state == healthprobe_dto.StateHealthy {
			state = healthprobe_dto.StateDegraded
			message = "GoChannel provider has degraded dependencies"
		}
	}

	return healthprobe_dto.Status{
		Name:         p.Name(),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: dependencies,
	}
}

// checkRouterHealth verifies the Watermill router is operational.
//
// Takes startTime (time.Time) which marks when the health check began.
//
// Returns healthprobe_dto.Status which contains the router's health state.
func (p *GoChannelProvider) checkRouterHealth(startTime time.Time) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "Watermill router is running"

	if p.router == nil {
		return healthprobe_dto.Status{
			Name:      "WatermillRouter",
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "Router is not initialised",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	select {
	case <-p.router.Running():
		state = healthprobe_dto.StateHealthy
		message = "Watermill router is running"
	default:
		state = healthprobe_dto.StateUnhealthy
		message = "Watermill router is not running"
	}

	return healthprobe_dto.Status{
		Name:      "WatermillRouter",
		State:     state,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// checkPubSubHealth verifies the GoChannel pub/sub is operational.
//
// Takes startTime (time.Time) which marks when the health check began.
//
// Returns healthprobe_dto.Status which contains the health state and timing.
func (p *GoChannelProvider) checkPubSubHealth(startTime time.Time) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "GoChannel pub/sub is operational"

	if p.pubsub == nil {
		state = healthprobe_dto.StateUnhealthy
		message = "GoChannel pub/sub is not initialised"
	}

	return healthprobe_dto.Status{
		Name:      "GoChannelPubSub",
		State:     state,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}
