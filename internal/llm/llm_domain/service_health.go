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

package llm_domain

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// Name returns the service name for health probe identification.
//
// Returns string which is "LLMService".
func (*service) Name() string {
	return "LLMService"
}

// Check performs a health check on the LLM service.
//
// For liveness: Checks that the service is running (always healthy if called).
// For readiness: Reports healthy when no providers are
// configured, degraded when a default provider is missing or
// does not match a registered provider.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies liveness or
// readiness.
//
// Returns healthprobe_dto.Status containing the health state and details.
//
// Safe for concurrent use. Uses a read lock to access provider state.
func (s *service) Check(_ context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	s.mu.RLock()
	providerCount := len(s.providers)
	hasDefault := s.defaultProvider != ""
	defaultExists := false
	if hasDefault {
		_, defaultExists = s.providers[s.defaultProvider]
	}
	s.mu.RUnlock()

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return healthprobe_dto.Status{
			Name:      s.Name(),
			State:     healthprobe_dto.StateHealthy,
			Message:   "LLM service is running",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime).String(),
		}
	}

	state := healthprobe_dto.StateHealthy
	var message string

	switch {
	case providerCount == 0:
		message = "No LLM providers configured"
	case !hasDefault:
		state = healthprobe_dto.StateDegraded
		message = fmt.Sprintf("%d provider(s) registered but no default set", providerCount)
	case !defaultExists:
		state = healthprobe_dto.StateUnhealthy
		message = fmt.Sprintf("Default provider %q not found in %d registered provider(s)", s.defaultProvider, providerCount)
	default:
		message = fmt.Sprintf("Ready with %d provider(s), default: %s", providerCount, s.defaultProvider)
	}

	return healthprobe_dto.Status{
		Name:      s.Name(),
		State:     state,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}
