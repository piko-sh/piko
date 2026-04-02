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

package monitoring_adapters

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

type mockHealthprobeDomainService struct {
	livenessReturn  healthprobe_dto.Status
	readinessReturn healthprobe_dto.Status
}

func (m *mockHealthprobeDomainService) CheckLiveness(_ context.Context) healthprobe_dto.Status {
	return m.livenessReturn
}

func (m *mockHealthprobeDomainService) CheckReadiness(_ context.Context) healthprobe_dto.Status {
	return m.readinessReturn
}

func TestNewHealthProbeAdapter(t *testing.T) {
	t.Parallel()

	mock := &mockHealthprobeDomainService{}
	adapter := NewHealthProbeAdapter(mock)
	require.NotNil(t, adapter)
}

func TestHealthProbeAdapter_CheckLiveness(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockHealthprobeDomainService{
		livenessReturn: healthprobe_dto.Status{
			Name:      "liveness",
			State:     healthprobe_dto.StateHealthy,
			Message:   "all good",
			Timestamp: now,
			Duration:  "1ms",
		},
	}

	adapter := NewHealthProbeAdapter(mock)
	result := adapter.CheckLiveness(context.Background())

	assert.Equal(t, "liveness", result.Name)
	assert.Equal(t, "HEALTHY", result.State)
	assert.Equal(t, "all good", result.Message)
	assert.Equal(t, now.UnixMilli(), result.Timestamp)
	assert.Equal(t, "1ms", result.Duration)
}

func TestHealthProbeAdapter_CheckReadiness(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mock := &mockHealthprobeDomainService{
		readinessReturn: healthprobe_dto.Status{
			Name:      "readiness",
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "database down",
			Timestamp: now,
			Duration:  "5ms",
		},
	}

	adapter := NewHealthProbeAdapter(mock)
	result := adapter.CheckReadiness(context.Background())

	assert.Equal(t, "readiness", result.Name)
	assert.Equal(t, "UNHEALTHY", result.State)
	assert.Equal(t, "database down", result.Message)
	assert.Equal(t, now.UnixMilli(), result.Timestamp)
	assert.Equal(t, "5ms", result.Duration)
}

func TestConvertDTOStatus_NoDependencies(t *testing.T) {
	t.Parallel()

	now := time.Now()
	status := healthprobe_dto.Status{
		Name:      "test-probe",
		State:     healthprobe_dto.StateDegraded,
		Message:   "partial failure",
		Timestamp: now,
		Duration:  "3ms",
	}

	result := convertDTOStatus(status)

	assert.Equal(t, "test-probe", result.Name)
	assert.Equal(t, "DEGRADED", result.State)
	assert.Equal(t, "partial failure", result.Message)
	assert.Equal(t, now.UnixMilli(), result.Timestamp)
	assert.Equal(t, "3ms", result.Duration)
	assert.Empty(t, result.Dependencies)
}

func TestConvertDTOStatus_WithDependencies(t *testing.T) {
	t.Parallel()

	now := time.Now()
	status := healthprobe_dto.Status{
		Name:      "overall",
		State:     healthprobe_dto.StateHealthy,
		Message:   "",
		Timestamp: now,
		Duration:  "10ms",
		Dependencies: []*healthprobe_dto.Status{
			{
				Name:      "database",
				State:     healthprobe_dto.StateHealthy,
				Timestamp: now,
				Duration:  "2ms",
			},
			{
				Name:      "cache",
				State:     healthprobe_dto.StateDegraded,
				Message:   "high latency",
				Timestamp: now,
				Duration:  "8ms",
			},
		},
	}

	result := convertDTOStatus(status)

	assert.Equal(t, "overall", result.Name)
	require.Len(t, result.Dependencies, 2)
	assert.Equal(t, "database", result.Dependencies[0].Name)
	assert.Equal(t, "HEALTHY", result.Dependencies[0].State)
	assert.Equal(t, "cache", result.Dependencies[1].Name)
	assert.Equal(t, "DEGRADED", result.Dependencies[1].State)
	assert.Equal(t, "high latency", result.Dependencies[1].Message)
}

func TestConvertDTOStatus_NilDependency(t *testing.T) {
	t.Parallel()

	now := time.Now()
	status := healthprobe_dto.Status{
		Name:      "overall",
		State:     healthprobe_dto.StateHealthy,
		Timestamp: now,
		Duration:  "1ms",
		Dependencies: []*healthprobe_dto.Status{
			nil,
		},
	}

	result := convertDTOStatus(status)

	require.Len(t, result.Dependencies, 1)

	assert.Equal(t, monitoring_domain.HealthProbeStatus{}, result.Dependencies[0])
}

func TestConvertDTOStatus_NestedDependencies(t *testing.T) {
	t.Parallel()

	now := time.Now()
	status := healthprobe_dto.Status{
		Name:      "root",
		State:     healthprobe_dto.StateHealthy,
		Timestamp: now,
		Duration:  "20ms",
		Dependencies: []*healthprobe_dto.Status{
			{
				Name:      "level1",
				State:     healthprobe_dto.StateHealthy,
				Timestamp: now,
				Duration:  "5ms",
				Dependencies: []*healthprobe_dto.Status{
					{
						Name:      "level2",
						State:     healthprobe_dto.StateHealthy,
						Timestamp: now,
						Duration:  "1ms",
					},
				},
			},
		},
	}

	result := convertDTOStatus(status)

	require.Len(t, result.Dependencies, 1)
	require.Len(t, result.Dependencies[0].Dependencies, 1)
	assert.Equal(t, "level2", result.Dependencies[0].Dependencies[0].Name)
}

func TestHealthProbeAdapter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ monitoring_domain.HealthProbeService = (*HealthProbeAdapter)(nil)
}
