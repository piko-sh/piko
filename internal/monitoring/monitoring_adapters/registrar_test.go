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

func TestDefaultServiceFactories(t *testing.T) {
	t.Parallel()

	factories := DefaultServiceFactories()
	require.NotNil(t, factories.SpanProcessorFactory)
	require.NotNil(t, factories.MetricsCollectorFactory)
}

func TestDefaultServiceFactories_SpanProcessorFactory(t *testing.T) {
	t.Parallel()

	factories := DefaultServiceFactories()
	processor := factories.SpanProcessorFactory(nil)
	require.NotNil(t, processor)
}

func TestDefaultServiceFactories_MetricsCollectorFactory(t *testing.T) {
	t.Parallel()

	factories := DefaultServiceFactories()
	collector := factories.MetricsCollectorFactory(nil, 5*time.Second)
	require.NotNil(t, collector)
}

func TestNoopSpanProcessor_Shutdown(t *testing.T) {
	t.Parallel()

	processor := &noopSpanProcessor{}
	err := processor.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNoopSpanProcessor_ForceFlush(t *testing.T) {
	t.Parallel()

	processor := &noopSpanProcessor{}
	err := processor.ForceFlush(context.Background())
	assert.NoError(t, err)
}

func TestNoopMetricReader_Shutdown(t *testing.T) {
	t.Parallel()

	reader := &noopMetricReader{}
	err := reader.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNoopMetricsCollector_Start(t *testing.T) {
	t.Parallel()

	collector := &noopMetricsCollector{}
	collector.Start(context.Background())
}

func TestNoopMetricsCollector_Stop(t *testing.T) {
	t.Parallel()

	collector := &noopMetricsCollector{}
	collector.Stop()
}

func TestNoopMetricsCollector_Reader(t *testing.T) {
	t.Parallel()

	collector := &noopMetricsCollector{}
	reader := collector.Reader()
	require.NotNil(t, reader)

	err := reader.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNoopSpanProcessor_ImplementsSpanProcessor(t *testing.T) {
	t.Parallel()

	var _ monitoring_domain.SpanProcessor = (*noopSpanProcessor)(nil)
}

func TestNoopMetricReader_ImplementsMetricReader(t *testing.T) {
	t.Parallel()

	var _ monitoring_domain.MetricReader = (*noopMetricReader)(nil)
}

func TestNoopMetricsCollector_ImplementsMetricsCollectorAdapter(t *testing.T) {
	t.Parallel()

	var _ monitoring_domain.MetricsCollectorAdapter = (*noopMetricsCollector)(nil)
}

func TestConvertDTOStatus_EmptyDependenciesSlice(t *testing.T) {
	t.Parallel()

	now := time.Now()
	status := healthprobe_dto.Status{
		Name:         "probe",
		State:        healthprobe_dto.StateHealthy,
		Timestamp:    now,
		Duration:     "2ms",
		Dependencies: []*healthprobe_dto.Status{},
	}

	result := convertDTOStatus(status)

	assert.Equal(t, "probe", result.Name)
	assert.Equal(t, "HEALTHY", result.State)
	assert.Empty(t, result.Dependencies)
}

func TestConvertDTOStatus_AllStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		inputState    healthprobe_dto.State
		expectedState string
	}{
		{
			name:          "healthy state",
			inputState:    healthprobe_dto.StateHealthy,
			expectedState: "HEALTHY",
		},
		{
			name:          "degraded state",
			inputState:    healthprobe_dto.StateDegraded,
			expectedState: "DEGRADED",
		},
		{
			name:          "unhealthy state",
			inputState:    healthprobe_dto.StateUnhealthy,
			expectedState: "UNHEALTHY",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			status := healthprobe_dto.Status{
				Name:      "state-test",
				State:     testCase.inputState,
				Timestamp: time.Now(),
				Duration:  "1ms",
			}

			result := convertDTOStatus(status)
			assert.Equal(t, testCase.expectedState, result.State)
		})
	}
}

func TestConvertDTOStatus_NilInMiddleOfDependencies(t *testing.T) {
	t.Parallel()

	now := time.Now()
	status := healthprobe_dto.Status{
		Name:      "root",
		State:     healthprobe_dto.StateHealthy,
		Timestamp: now,
		Duration:  "5ms",
		Dependencies: []*healthprobe_dto.Status{
			{
				Name:      "first",
				State:     healthprobe_dto.StateHealthy,
				Timestamp: now,
				Duration:  "1ms",
			},
			nil,
			{
				Name:      "third",
				State:     healthprobe_dto.StateDegraded,
				Timestamp: now,
				Duration:  "3ms",
			},
		},
	}

	result := convertDTOStatus(status)

	require.Len(t, result.Dependencies, 3)
	assert.Equal(t, "first", result.Dependencies[0].Name)
	assert.Equal(t, monitoring_domain.HealthProbeStatus{}, result.Dependencies[1])
	assert.Equal(t, "third", result.Dependencies[2].Name)
}

func TestConvertDTOStatus_DeeplyNestedDependencies(t *testing.T) {
	t.Parallel()

	now := time.Now()
	status := healthprobe_dto.Status{
		Name:      "level0",
		State:     healthprobe_dto.StateHealthy,
		Timestamp: now,
		Duration:  "30ms",
		Dependencies: []*healthprobe_dto.Status{
			{
				Name:      "level1",
				State:     healthprobe_dto.StateHealthy,
				Timestamp: now,
				Duration:  "20ms",
				Dependencies: []*healthprobe_dto.Status{
					{
						Name:      "level2",
						State:     healthprobe_dto.StateDegraded,
						Timestamp: now,
						Duration:  "10ms",
						Dependencies: []*healthprobe_dto.Status{
							{
								Name:      "level3",
								State:     healthprobe_dto.StateUnhealthy,
								Timestamp: now,
								Duration:  "5ms",
							},
						},
					},
				},
			},
		},
	}

	result := convertDTOStatus(status)

	require.Len(t, result.Dependencies, 1)
	require.Len(t, result.Dependencies[0].Dependencies, 1)
	require.Len(t, result.Dependencies[0].Dependencies[0].Dependencies, 1)
	assert.Equal(t, "level3", result.Dependencies[0].Dependencies[0].Dependencies[0].Name)
	assert.Equal(t, "UNHEALTHY", result.Dependencies[0].Dependencies[0].Dependencies[0].State)
}

func TestConvertDTOStatus_TimestampConversion(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	status := healthprobe_dto.Status{
		Name:      "timestamp-test",
		State:     healthprobe_dto.StateHealthy,
		Timestamp: fixedTime,
		Duration:  "1ms",
	}

	result := convertDTOStatus(status)

	assert.Equal(t, fixedTime.UnixMilli(), result.Timestamp)
}

func TestConvertDTOStatus_EmptyFields(t *testing.T) {
	t.Parallel()

	status := healthprobe_dto.Status{}

	result := convertDTOStatus(status)

	assert.Empty(t, result.Name)
	assert.Empty(t, result.State)
	assert.Empty(t, result.Message)
	assert.Empty(t, result.Duration)
	assert.Empty(t, result.Dependencies)
}
