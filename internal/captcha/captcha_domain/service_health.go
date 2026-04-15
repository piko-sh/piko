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

package captcha_domain

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// healthProbeName is the identifier used when registering with the health
// probe system.
const healthProbeName = "CaptchaService"

var _ interface {
	Name() string
	Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status
} = (*captchaService)(nil)

// Name returns the service identifier for the health probe system.
//
// Returns string which is the health probe name.
func (*captchaService) Name() string {
	return healthProbeName
}

// Check performs a health check on the captcha service, checking provider
// registration for liveness or calling the provider's HealthCheck for
// readiness.
//
// Takes checkType (healthprobe_dto.CheckType) which selects liveness or
// readiness.
//
// Returns healthprobe_dto.Status which describes the service health state.
func (s *captchaService) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := s.clock.Now()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return s.checkLiveness(startTime)
	}

	return s.checkReadiness(ctx, startTime)
}

// checkLiveness verifies that a captcha provider is registered.
//
// Takes startTime (time.Time) which is when the check began.
//
// Returns healthprobe_dto.Status which is healthy when a provider exists.
func (s *captchaService) checkLiveness(startTime time.Time) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "captcha service operational"

	if !s.IsEnabled() {
		state = healthprobe_dto.StateDegraded
		message = "no captcha provider configured"
	}

	return healthprobe_dto.Status{
		Name:      healthProbeName,
		State:     state,
		Message:   message,
		Timestamp: startTime,
		Duration:  s.clock.Now().Sub(startTime).String(),
	}
}

// checkReadiness calls the default provider's HealthCheck to verify end-to-end
// readiness.
//
// Takes startTime (time.Time) which is when the check began.
//
// Returns healthprobe_dto.Status which reports the provider's health.
func (s *captchaService) checkReadiness(ctx context.Context, startTime time.Time) healthprobe_dto.Status {
	if !s.IsEnabled() {
		return healthprobe_dto.Status{
			Name:      healthProbeName,
			State:     healthprobe_dto.StateDegraded,
			Message:   "no captcha provider configured",
			Timestamp: startTime,
			Duration:  s.clock.Now().Sub(startTime).String(),
		}
	}

	provider, err := s.getProvider(ctx)
	if err != nil {
		return healthprobe_dto.Status{
			Name:      healthProbeName,
			State:     healthprobe_dto.StateUnhealthy,
			Message:   fmt.Sprintf("failed to resolve provider: %v", err),
			Timestamp: startTime,
			Duration:  s.clock.Now().Sub(startTime).String(),
		}
	}

	healthErr := provider.HealthCheck(ctx)

	providerStatus := &healthprobe_dto.Status{
		Name:      string(provider.Type()),
		Timestamp: startTime,
		Duration:  s.clock.Now().Sub(startTime).String(),
	}

	if healthErr != nil {
		providerStatus.State = healthprobe_dto.StateUnhealthy
		providerStatus.Message = healthErr.Error()
	} else {
		providerStatus.State = healthprobe_dto.StateHealthy
		providerStatus.Message = "provider healthy"
	}

	overallState := providerStatus.State
	overallMessage := "captcha service operational"
	if overallState == healthprobe_dto.StateUnhealthy {
		overallMessage = "captcha provider unhealthy"
	}

	return healthprobe_dto.Status{
		Name:         healthProbeName,
		State:        overallState,
		Message:      overallMessage,
		Timestamp:    startTime,
		Duration:     s.clock.Now().Sub(startTime).String(),
		Dependencies: []*healthprobe_dto.Status{providerStatus},
	}
}
