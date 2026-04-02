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

package piko

import (
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

//
// Piko provides a health check system with two endpoints:
//   - /live (Liveness): Returns 200 if the application is running (not
//     deadlocked)
//   - /ready (Readiness): Returns 200 if the application is ready to serve
//     traffic
//
// The health check server runs on a separate port (default: 9090) and binds to
// localhost by default for security. It can be configured via the HealthProbe
// section in your Piko configuration.
//
// Built-in Health Checks: Piko automatically monitors the health of its
// internal services:
//   - RegistryService: Artefact storage and metadata
//   - OrchestratorService: Task queue and workers
//   - CollectionService: Content providers
//   - RenderService: Template rendering pipeline
//   - StorageService: File/blob storage
//   - EmailService: Email delivery
//   - CryptoService: Encryption operations
//   - CacheService: Cache backends
//   - And more...
//
// Custom Health Checks: You can extend Piko's health monitoring with your own
// application-specific checks by implementing the HealthProbe interface and
// registering it with WithCustomHealthProbe.
//
// Example configuration (piko.yaml):
// healthProbe:
//   enabled: true
//   port: "9090"
//   bindAddress: "127.0.0.1"  # localhost only (secure)
//   livePath: "/live"
//   readyPath: "/ready"
//   checkTimeoutSeconds: 5
// To expose health checks externally (e.g., for Docker health checks):
// healthProbe:
//   bindAddress: "0.0.0.0"  # WARNING: Exposes internal health data

const (
	// HealthStateHealthy indicates the component is working normally.
	HealthStateHealthy HealthState = healthprobe_dto.StateHealthy

	// HealthStateDegraded indicates the component is working but with reduced
	// performance or limited features.
	HealthStateDegraded HealthState = healthprobe_dto.StateDegraded

	// HealthStateUnhealthy indicates the component is not working.
	HealthStateUnhealthy HealthState = healthprobe_dto.StateUnhealthy

	// HealthCheckLiveness checks if the application is running and not stuck.
	// If this check fails, the application is usually restarted.
	HealthCheckLiveness HealthCheckType = healthprobe_dto.CheckTypeLiveness

	// HealthCheckReadiness determines if the application is ready to serve
	// traffic. Failing this check typically results in traffic being withheld from
	// the application.
	HealthCheckReadiness HealthCheckType = healthprobe_dto.CheckTypeReadiness
)

// HealthProbe is the interface that custom application health checks must
// implement. Implementing this interface allows your application to take part
// in Piko's health check system, which is exposed via the /live and /ready
// endpoints.
//
// Interface definition:
//
//	type HealthProbe interface {
//	    Name() string
//	    Check(ctx context.Context, checkType HealthCheckType) HealthStatus
//	}
//
// The Check method receives the checkType parameter, which allows your probe
// to return different results for liveness vs readiness checks:
//   - Liveness: Quick check - is the service initialised and not deadlocked?
//   - Readiness: Thorough check - is the service ready to handle requests?
type HealthProbe = healthprobe_domain.Probe

// HealthStatus represents the result of a health check operation. It includes
// the component name, state, optional message, timestamp, and duration.
type HealthStatus = healthprobe_dto.Status

// HealthState represents the health status of a component.
type HealthState = healthprobe_dto.State

// HealthCheckType indicates whether this is a liveness or readiness check.
type HealthCheckType = healthprobe_dto.CheckType

// WithCustomHealthProbe registers a custom health probe with the Piko
// framework. The probe will be included in the /live and /ready health check
// endpoints.
//
// Use it to extend Piko's built-in health monitoring with application-specific
// checks (e.g., database connectivity, external API availability).
//
// Takes probe (HealthProbe) which is the health probe to register.
//
// Returns Option which configures the container with the custom probe.
//
// Example:
//
//	type RedisProbe struct {
//	    client *redis.Client
//	}
//
// func (p *RedisProbe) Name() string { return "ApplicationRedisCache" }
//
//	func (p *RedisProbe) Check(ctx context.Context, checkType piko.HealthCheckType) piko.HealthStatus {
//	    startTime := time.Now()
//	    err := p.client.Ping(ctx).Err()
//	    state := piko.HealthStateHealthy
//	    message := "Redis connection OK"
//	    if err != nil {
//	        state = piko.HealthStateUnhealthy
//	        message = fmt.Sprintf("Redis connection failed: %v", err)
//	    }
//	    return piko.HealthStatus{
//	        Name:      p.Name(),
//	        State:     state,
//	        Message:   message,
//	        Timestamp: time.Now(),
//	        Duration:  time.Since(startTime).String(),
//	    }
//	}
//
// // Register the probe
// server := piko.New(
//
//	piko.WithCustomHealthProbe(&RedisProbe{client: redisClient}),
//
// )
func WithCustomHealthProbe(probe HealthProbe) Option {
	return func(c *Container) {
		c.AddCustomHealthProbe(probe)
	}
}
