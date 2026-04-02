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

package provider_grpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/logger"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

var _ tui_domain.HealthProvider = (*HealthProvider)(nil)

// HealthProvider provides health status information using gRPC.
// It implements tui_domain.HealthProvider.
type HealthProvider struct {
	// conn holds the gRPC connection for health checks.
	conn *Connection

	// liveness stores the current liveness health status; nil means no data available.
	liveness *tui_domain.HealthStatus

	// readiness holds the current readiness status; nil until first Refresh.
	readiness *tui_domain.HealthStatus

	// mu guards access to liveness and readiness fields.
	mu sync.RWMutex

	// interval is the time between health check refreshes.
	interval time.Duration
}

// NewHealthProvider creates a new HealthProvider.
//
// Takes conn (*Connection) which is the shared gRPC connection.
// Takes interval (time.Duration) which is the refresh interval.
//
// Returns *HealthProvider which is the configured provider.
func NewHealthProvider(conn *Connection, interval time.Duration) *HealthProvider {
	return &HealthProvider{
		conn:      conn,
		liveness:  nil,
		readiness: nil,
		mu:        sync.RWMutex{},
		interval:  interval,
	}
}

// Name returns the provider name.
//
// Returns string which is the identifier "grpc-health".
func (*HealthProvider) Name() string {
	return "grpc-health"
}

// Health checks if the gRPC connection is healthy.
//
// Returns error when the health check fails.
func (p *HealthProvider) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking health provider health via gRPC: %w", err)
	}
	return nil
}

// Close releases resources.
//
// Returns error when resources cannot be released.
func (*HealthProvider) Close() error {
	return nil
}

// RefreshInterval returns the refresh interval.
//
// Returns time.Duration which is the interval between health checks.
func (p *HealthProvider) RefreshInterval() time.Duration {
	return p.interval
}

// Refresh fetches the latest health status via gRPC.
//
// Returns error when the gRPC call fails.
//
// Safe for concurrent use. Updates internal state under mutex protection.
func (p *HealthProvider) Refresh(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	return instrumentedCall(ctx, func() error {
		response, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
		if err != nil {
			l.Debug("Failed to fetch health status", logger.Error(err))
			return fmt.Errorf("fetching health: %w", err)
		}

		p.mu.Lock()
		p.liveness = parseProtoHealthStatus(response.GetLiveness())
		p.readiness = parseProtoHealthStatus(response.GetReadiness())
		p.mu.Unlock()

		return nil
	})
}

// Liveness returns the current liveness status.
//
// Returns *tui_domain.HealthStatus which contains the current liveness data.
// Returns error when no liveness data is available.
//
// Safe for concurrent use.
func (p *HealthProvider) Liveness(_ context.Context) (*tui_domain.HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.liveness == nil {
		return nil, errors.New("no liveness data available")
	}

	return p.liveness, nil
}

// Readiness returns the current readiness status.
//
// Returns *tui_domain.HealthStatus which contains the current readiness state.
// Returns error when no readiness data is available.
//
// Safe for concurrent use.
func (p *HealthProvider) Readiness(_ context.Context) (*tui_domain.HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.readiness == nil {
		return nil, errors.New("no readiness data available")
	}

	return p.readiness, nil
}

// parseProtoHealthStatus converts a proto HealthStatus to a domain
// HealthStatus, recursively converting any dependencies.
//
// Takes pbStatus (*pb.HealthStatus) which is the protobuf status to convert.
//
// Returns *tui_domain.HealthStatus which is the converted domain object.
func parseProtoHealthStatus(pbStatus *pb.HealthStatus) *tui_domain.HealthStatus {
	if pbStatus == nil {
		return &tui_domain.HealthStatus{
			Name:         "unknown",
			State:        tui_domain.HealthStateUnknown,
			Message:      "",
			Timestamp:    time.Now(),
			Duration:     0,
			Dependencies: nil,
		}
	}

	var state tui_domain.HealthState
	switch pbStatus.GetState() {
	case "HEALTHY", "ok":
		state = tui_domain.HealthStateHealthy
	case "DEGRADED":
		state = tui_domain.HealthStateDegraded
	case "UNHEALTHY":
		state = tui_domain.HealthStateUnhealthy
	default:
		state = tui_domain.HealthStateUnknown
	}

	var duration time.Duration
	if pbStatus.GetDuration() != "" {
		if d, err := time.ParseDuration(pbStatus.GetDuration()); err == nil {
			duration = d
		}
	}

	pbDeps := pbStatus.GetDependencies()
	deps := make([]*tui_domain.HealthStatus, 0, len(pbDeps))
	for _, dependency := range pbDeps {
		deps = append(deps, parseProtoHealthStatus(dependency))
	}

	return &tui_domain.HealthStatus{
		Name:         pbStatus.GetName(),
		State:        state,
		Message:      pbStatus.GetMessage(),
		Timestamp:    time.UnixMilli(pbStatus.GetTimestampMs()),
		Duration:     duration,
		Dependencies: deps,
	}
}
