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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

type mockRateLimiterInspector struct {
	statusError  error
	statusReturn ratelimiter_dto.Status
}

func (m *mockRateLimiterInspector) GetStatus(_ context.Context) (ratelimiter_dto.Status, error) {
	return m.statusReturn, m.statusError
}

func TestNewRateLimiterInspectorService(t *testing.T) {
	t.Parallel()

	mock := &mockRateLimiterInspector{}
	service := NewRateLimiterInspectorService(mock)
	require.NotNil(t, service)
}

func TestRateLimiterInspectorService_GetRateLimiterStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		inspector   *mockRateLimiterInspector
		name        string
		expectError bool
	}{
		{
			name: "returns status successfully",
			inspector: &mockRateLimiterInspector{
				statusReturn: ratelimiter_dto.Status{
					TokenBucketStore: "cache",
					CounterStore:     "inmemory",
					FailPolicy:       "open",
					KeyPrefix:        "piko:",
					TotalChecks:      1000,
					TotalAllowed:     950,
					TotalDenied:      45,
					TotalErrors:      5,
				},
			},
			expectError: false,
		},
		{
			name: "returns empty status",
			inspector: &mockRateLimiterInspector{
				statusReturn: ratelimiter_dto.Status{},
			},
			expectError: false,
		},
		{
			name: "propagates error",
			inspector: &mockRateLimiterInspector{
				statusError: errors.New("connection refused"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewRateLimiterInspectorService(tc.inspector)
			response, err := service.GetRateLimiterStatus(context.Background(), &pb.GetRateLimiterStatusRequest{})

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "getting rate limiter status")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.inspector.statusReturn.TokenBucketStore, response.TokenBucketStore)
			assert.Equal(t, tc.inspector.statusReturn.CounterStore, response.CounterStore)
			assert.Equal(t, tc.inspector.statusReturn.FailPolicy, response.FailPolicy)
			assert.Equal(t, tc.inspector.statusReturn.KeyPrefix, response.KeyPrefix)
			assert.Equal(t, tc.inspector.statusReturn.TotalChecks, response.TotalChecks)
			assert.Equal(t, tc.inspector.statusReturn.TotalAllowed, response.TotalAllowed)
			assert.Equal(t, tc.inspector.statusReturn.TotalDenied, response.TotalDenied)
			assert.Equal(t, tc.inspector.statusReturn.TotalErrors, response.TotalErrors)
		})
	}
}
