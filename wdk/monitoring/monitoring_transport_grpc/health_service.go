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

package monitoring_transport_grpc

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/clock"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

// HealthServiceOption configures a HealthService.
type HealthServiceOption func(*HealthService)

// HealthService implements the gRPC health service interface.
type HealthService struct {
	pb.UnimplementedHealthServiceServer

	// healthProbe provides liveness and readiness checks; nil falls back to
	// simple health status.
	healthProbe monitoring_domain.HealthProbeService

	// clock provides time operations for timestamps and tickers.
	clock clock.Clock
}

// NewHealthService creates a new HealthService.
//
// Takes healthProbe (monitoring_domain.HealthProbeService) which provides
// detailed health checks. May be nil for basic health reporting.
//
// Returns *HealthService which is the configured service ready for use.
func NewHealthService(healthProbe monitoring_domain.HealthProbeService, opts ...HealthServiceOption) *HealthService {
	s := &HealthService{
		healthProbe: healthProbe,
		clock:       clock.RealClock(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetHealth returns the current health status.
//
// Returns *pb.GetHealthResponse which contains liveness and readiness status.
// Returns error when the health check fails.
func (s *HealthService) GetHealth(ctx context.Context, _ *pb.GetHealthRequest) (*pb.GetHealthResponse, error) {
	now := s.clock.Now()

	if s.healthProbe == nil {
		simpleStatus := &pb.HealthStatus{
			Name:        "piko",
			State:       "HEALTHY",
			Message:     "Health probe not configured",
			TimestampMs: now.UnixMilli(),
			Duration:    "0s",
		}
		return &pb.GetHealthResponse{
			Liveness:    simpleStatus,
			Readiness:   simpleStatus,
			TimestampMs: now.UnixMilli(),
		}, nil
	}

	liveness := s.healthProbe.CheckLiveness(ctx)
	readiness := s.healthProbe.CheckReadiness(ctx)

	return &pb.GetHealthResponse{
		Liveness:    convertHealthStatus(liveness),
		Readiness:   convertHealthStatus(readiness),
		TimestampMs: now.UnixMilli(),
	}, nil
}

// WatchHealth streams health status updates at the requested interval.
//
// Takes request (*pb.WatchHealthRequest) which specifies the watch interval.
// Takes stream (pb.HealthService_WatchHealthServer) which receives updates.
//
// Returns error when the stream context is cancelled or sending fails.
func (s *HealthService) WatchHealth(request *pb.WatchHealthRequest, stream pb.HealthService_WatchHealthServer) error {
	interval := time.Duration(request.GetIntervalMs()) * time.Millisecond
	if interval < minWatchIntervalMs*time.Millisecond {
		interval = 1 * time.Second
	}

	ticker := s.clock.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C():
			now := s.clock.Now()
			var liveness, readiness *pb.HealthStatus

			if s.healthProbe == nil {
				simpleStatus := &pb.HealthStatus{
					Name:        "piko",
					State:       "HEALTHY",
					Message:     "Health probe not configured",
					TimestampMs: now.UnixMilli(),
					Duration:    "0s",
				}
				liveness = simpleStatus
				readiness = simpleStatus
			} else {
				liveness = convertHealthStatus(s.healthProbe.CheckLiveness(stream.Context()))
				readiness = convertHealthStatus(s.healthProbe.CheckReadiness(stream.Context()))
			}

			if err := stream.Send(&pb.HealthUpdate{
				Liveness:    liveness,
				Readiness:   readiness,
				TimestampMs: now.UnixMilli(),
			}); err != nil {
				return fmt.Errorf("sending health update: %w", err)
			}
		}
	}
}

// WithHealthServiceClock sets the clock used for timestamp generation. If not
// provided, the real system clock is used.
//
// Takes clk (clock.Clock) which provides time operations.
//
// Returns HealthServiceOption which configures the service's clock.
func WithHealthServiceClock(clk clock.Clock) HealthServiceOption {
	return func(s *HealthService) {
		if clk != nil {
			s.clock = clk
		}
	}
}

// convertHealthStatus converts a domain HealthProbeStatus to a proto
// HealthStatus.
//
// Takes status (monitoring_domain.HealthProbeStatus) which is the domain health
// status to convert.
//
// Returns *pb.HealthStatus which is the proto representation of the health
// status, including recursively converted dependencies.
func convertHealthStatus(status monitoring_domain.HealthProbeStatus) *pb.HealthStatus {
	deps := make([]*pb.HealthStatus, len(status.Dependencies))
	for i, dependency := range status.Dependencies {
		deps[i] = convertHealthStatus(dependency)
	}

	return &pb.HealthStatus{
		Name:         status.Name,
		State:        status.State,
		Message:      status.Message,
		TimestampMs:  status.Timestamp,
		Duration:     status.Duration,
		Dependencies: deps,
	}
}
