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

	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

// RateLimiterInspectorService implements the gRPC service for inspecting
// rate limiter state.
type RateLimiterInspectorService struct {
	pb.UnimplementedRateLimiterInspectorServiceServer

	// inspector provides rate limiter querying operations.
	inspector ratelimiter_domain.RateLimiterInspector
}

// NewRateLimiterInspectorService creates a new RateLimiterInspectorService.
//
// Takes inspector (RateLimiterInspector) which provides rate limiter
// inspection capabilities.
//
// Returns *RateLimiterInspectorService which is ready for use as a gRPC
// service handler.
func NewRateLimiterInspectorService(inspector ratelimiter_domain.RateLimiterInspector) *RateLimiterInspectorService {
	return &RateLimiterInspectorService{
		UnimplementedRateLimiterInspectorServiceServer: pb.UnimplementedRateLimiterInspectorServiceServer{},
		inspector: inspector,
	}
}

// GetRateLimiterStatus returns the current rate limiter configuration and
// aggregate counters.
//
// Returns *pb.GetRateLimiterStatusResponse which contains the rate limiter
// status.
// Returns error when the status cannot be retrieved.
func (s *RateLimiterInspectorService) GetRateLimiterStatus(ctx context.Context, _ *pb.GetRateLimiterStatusRequest) (*pb.GetRateLimiterStatusResponse, error) {
	status, err := s.inspector.GetStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting rate limiter status: %w", err)
	}

	return &pb.GetRateLimiterStatusResponse{
		TokenBucketStore: status.TokenBucketStore,
		CounterStore:     status.CounterStore,
		FailPolicy:       status.FailPolicy,
		KeyPrefix:        status.KeyPrefix,
		TotalChecks:      status.TotalChecks,
		TotalAllowed:     status.TotalAllowed,
		TotalDenied:      status.TotalDenied,
		TotalErrors:      status.TotalErrors,
	}, nil
}
