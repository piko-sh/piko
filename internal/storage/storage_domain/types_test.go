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

package storage_domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/wdk/clock"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := storage_domain.DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.InitialDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffFactor)
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := storage_domain.DefaultCircuitBreakerConfig()

	assert.Equal(t, 5, config.MaxConsecutiveFailures)
	assert.Equal(t, 60*time.Second, config.Timeout)
	assert.Equal(t, 10*time.Second, config.Interval)
}

func TestWithMaxUploadSizeBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected int64
	}{
		{
			name:     "Valid positive value",
			input:    500000000,
			expected: 500000000,
		},
		{
			name:     "Zero value should not change config",
			input:    0,
			expected: 104857600,
		},
		{
			name:     "Negative value should not change config",
			input:    -100,
			expected: 104857600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := storage_domain.ServiceConfig{
				MaxUploadSizeBytes: 104857600,
			}

			option := storage_domain.WithMaxUploadSizeBytes(tt.input)
			option(&config)

			assert.Equal(t, tt.expected, config.MaxUploadSizeBytes)
		})
	}
}

func TestWithMaxBatchSize(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "Valid positive value",
			input:    5000,
			expected: 5000,
		},
		{
			name:     "Zero value should not change config",
			input:    0,
			expected: 1000,
		},
		{
			name:     "Negative value should not change config",
			input:    -50,
			expected: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := storage_domain.ServiceConfig{
				MaxBatchSize: 1000,
			}

			option := storage_domain.WithMaxBatchSize(tt.input)
			option(&config)

			assert.Equal(t, tt.expected, config.MaxBatchSize)
		})
	}
}

func TestWithRetryConfig(t *testing.T) {
	customConfig := storage_domain.RetryConfig{
		MaxRetries:    5,
		InitialDelay:  2 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 3.0,
	}

	config := storage_domain.ServiceConfig{
		RetryConfig: storage_domain.DefaultRetryConfig(),
	}

	option := storage_domain.WithRetryConfig(customConfig)
	option(&config)

	assert.Equal(t, customConfig.MaxRetries, config.RetryConfig.MaxRetries)
	assert.Equal(t, customConfig.InitialDelay, config.RetryConfig.InitialDelay)
	assert.Equal(t, customConfig.MaxDelay, config.RetryConfig.MaxDelay)
	assert.Equal(t, customConfig.BackoffFactor, config.RetryConfig.BackoffFactor)
}

func TestWithCircuitBreakerConfig(t *testing.T) {
	customConfig := storage_domain.CircuitBreakerConfig{
		MaxConsecutiveFailures: 10,
		Timeout:                120 * time.Second,
		Interval:               20 * time.Second,
	}

	config := storage_domain.ServiceConfig{
		CircuitBreakerConfig: storage_domain.DefaultCircuitBreakerConfig(),
	}

	option := storage_domain.WithCircuitBreakerConfig(customConfig)
	option(&config)

	assert.Equal(t, customConfig.MaxConsecutiveFailures, config.CircuitBreakerConfig.MaxConsecutiveFailures)
	assert.Equal(t, customConfig.Timeout, config.CircuitBreakerConfig.Timeout)
	assert.Equal(t, customConfig.Interval, config.CircuitBreakerConfig.Interval)
}

func TestWithRetryEnabled(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "Enable retry",
			input:    true,
			expected: true,
		},
		{
			name:     "Disable retry",
			input:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := storage_domain.ServiceConfig{}

			option := storage_domain.WithRetryEnabled(tt.input)
			option(&config)

			assert.Equal(t, tt.expected, config.EnableRetry)
		})
	}
}

func TestWithCircuitBreakerEnabled(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "Enable circuit breaker",
			input:    true,
			expected: true,
		},
		{
			name:     "Disable circuit breaker",
			input:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := storage_domain.ServiceConfig{}

			option := storage_domain.WithCircuitBreakerEnabled(tt.input)
			option(&config)

			assert.Equal(t, tt.expected, config.EnableCircuitBreaker)
		})
	}
}

func TestWithSingleflightEnabled(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "Enable singleflight",
			input:    true,
			expected: true,
		},
		{
			name:     "Disable singleflight",
			input:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := storage_domain.ServiceConfig{}

			option := storage_domain.WithSingleflightEnabled(tt.input)
			option(&config)

			assert.Equal(t, tt.expected, config.EnableSingleflight)
		})
	}
}

func TestWithSingleflightMemoryThreshold(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected int64
	}{
		{
			name:     "Valid positive value",
			input:    50000000,
			expected: 50000000,
		},
		{
			name:     "Zero value should not change config",
			input:    0,
			expected: 10485760,
		},
		{
			name:     "Negative value should not change config",
			input:    -100,
			expected: 10485760,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := storage_domain.ServiceConfig{
				SingleflightMemoryThreshold: 10485760,
			}

			option := storage_domain.WithSingleflightMemoryThreshold(tt.input)
			option(&config)

			assert.Equal(t, tt.expected, config.SingleflightMemoryThreshold)
		})
	}
}

func TestServiceStats_Uptime(t *testing.T) {
	t.Run("Uptime returns zero for zero StartTime", func(t *testing.T) {
		stats := storage_domain.ServiceStats{}
		assert.Equal(t, time.Duration(0), stats.Uptime())
	})

	t.Run("Uptime returns correct duration", func(t *testing.T) {
		startTime := time.Now().Add(-5 * time.Minute)
		stats := storage_domain.ServiceStats{
			StartTime: startTime,
		}

		uptime := stats.Uptime()

		assert.Greater(t, uptime, 4*time.Minute+50*time.Second)
		assert.Less(t, uptime, 5*time.Minute+10*time.Second)
	})
}

func TestServiceStats_UptimeAt(t *testing.T) {
	initialTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(initialTime)
	startTime := mockClock.Now()

	stats := storage_domain.ServiceStats{
		StartTime: startTime,
	}

	mockClock.Advance(5 * time.Minute)

	uptime := stats.UptimeAt(mockClock.Now())
	assert.Equal(t, 5*time.Minute, uptime)

	mockClock.Advance(30 * time.Second)
	uptime = stats.UptimeAt(mockClock.Now())
	assert.Equal(t, 5*time.Minute+30*time.Second, uptime)
}

func TestObjectInfo(t *testing.T) {
	now := time.Now()
	metadata := map[string]string{
		"author": "test-user",
		"type":   "document",
	}

	info := storage_domain.ObjectInfo{
		LastModified: now,
		Metadata:     metadata,
		ContentType:  "application/pdf",
		ETag:         "abc123",
		Size:         1024,
	}

	assert.Equal(t, now, info.LastModified)
	assert.Equal(t, metadata, info.Metadata)
	assert.Equal(t, "application/pdf", info.ContentType)
	assert.Equal(t, "abc123", info.ETag)
	assert.Equal(t, int64(1024), info.Size)
}

func TestMultipleServiceOptions(t *testing.T) {

	config := storage_domain.ServiceConfig{
		MaxUploadSizeBytes:          104857600,
		MaxBatchSize:                1000,
		EnableRetry:                 true,
		EnableCircuitBreaker:        true,
		EnableSingleflight:          true,
		SingleflightMemoryThreshold: 10485760,
	}

	options := []storage_domain.ServiceOption{
		storage_domain.WithMaxUploadSizeBytes(200000000),
		storage_domain.WithMaxBatchSize(2000),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
		storage_domain.WithSingleflightEnabled(false),
		storage_domain.WithSingleflightMemoryThreshold(20000000),
	}

	for _, option := range options {
		option(&config)
	}

	assert.Equal(t, int64(200000000), config.MaxUploadSizeBytes)
	assert.Equal(t, 2000, config.MaxBatchSize)
	assert.False(t, config.EnableRetry)
	assert.False(t, config.EnableCircuitBreaker)
	assert.False(t, config.EnableSingleflight)
	assert.Equal(t, int64(20000000), config.SingleflightMemoryThreshold)
}

func TestRetryConfig(t *testing.T) {
	config := storage_domain.RetryConfig{
		MaxRetries:    5,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      45 * time.Second,
		BackoffFactor: 2.5,
	}

	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, config.InitialDelay)
	assert.Equal(t, 45*time.Second, config.MaxDelay)
	assert.Equal(t, 2.5, config.BackoffFactor)
}

func TestCircuitBreakerConfig(t *testing.T) {
	config := storage_domain.CircuitBreakerConfig{
		MaxConsecutiveFailures: 8,
		Timeout:                90 * time.Second,
		Interval:               15 * time.Second,
	}

	assert.Equal(t, 8, config.MaxConsecutiveFailures)
	assert.Equal(t, 90*time.Second, config.Timeout)
	assert.Equal(t, 15*time.Second, config.Interval)
}

func TestServiceStats(t *testing.T) {
	initialTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(initialTime)
	startTime := mockClock.Now()
	stats := storage_domain.ServiceStats{
		StartTime:            startTime,
		TotalOperations:      100,
		SuccessfulOperations: 95,
		FailedOperations:     5,
		RetryAttempts:        10,
		CacheHits:            50,
		CacheMisses:          50,
		DLQEntries:           2,
	}

	assert.Equal(t, startTime, stats.StartTime)
	assert.Equal(t, int64(100), stats.TotalOperations)
	assert.Equal(t, int64(95), stats.SuccessfulOperations)
	assert.Equal(t, int64(5), stats.FailedOperations)
	assert.Equal(t, int64(10), stats.RetryAttempts)
	assert.Equal(t, int64(50), stats.CacheHits)
	assert.Equal(t, int64(50), stats.CacheMisses)
	assert.Equal(t, int64(2), stats.DLQEntries)

	mockClock.Advance(2 * time.Hour)
	uptime := stats.UptimeAt(mockClock.Now())
	require.Equal(t, 2*time.Hour, uptime)
}

func TestWithClock(t *testing.T) {
	initialTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(initialTime)

	config := storage_domain.ServiceConfig{}
	option := storage_domain.WithClock(mockClock)
	option(&config)

	assert.Equal(t, mockClock, config.Clock)
}
