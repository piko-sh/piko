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

package crypto_domain

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// HealthCheck verifies the crypto service is operational.
//
// Returns error when the underlying provider health check fails.
func (s *cryptoService) HealthCheck(ctx context.Context) error {
	provider, err := s.getProvider(ctx)
	if err != nil {
		return fmt.Errorf("getting provider for health check: %w", err)
	}
	return provider.HealthCheck(ctx)
}

// Name implements the healthprobe_domain.Probe interface.
//
// Returns string which is the service name "CryptoService".
func (*cryptoService) Name() string {
	return "CryptoService"
}

// Check implements the healthprobe_domain.Probe interface.
// It verifies the encryption provider is functional.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to perform.
//
// Returns healthprobe_dto.Status which contains the health state, message,
// and any provider dependencies.
func (s *cryptoService) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	provider, err := s.getProvider(ctx)
	if err != nil {
		return healthprobe_dto.Status{
			Name:         s.Name(),
			State:        healthprobe_dto.StateUnhealthy,
			Message:      fmt.Sprintf("Encryption provider not configured: %v", err),
			Timestamp:    time.Now(),
			Duration:     time.Since(startTime).String(),
			Dependencies: nil,
		}
	}

	state := healthprobe_dto.StateHealthy
	message := fmt.Sprintf("Crypto service operational with %s provider", provider.Type())

	dependencies := make([]*healthprobe_dto.Status, 0)

	if probe, ok := provider.(interface {
		Name() string
		Check(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status
	}); ok {
		providerStatus := probe.Check(ctx, checkType)
		dependencies = append(dependencies, &providerStatus)

		switch providerStatus.State {
		case healthprobe_dto.StateUnhealthy:
			state = healthprobe_dto.StateUnhealthy
			message = "Crypto provider unavailable"
		case healthprobe_dto.StateDegraded:
			state = healthprobe_dto.StateDegraded
			message = "Crypto provider degraded"
		}
	}

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        state,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: dependencies,
	}
}
