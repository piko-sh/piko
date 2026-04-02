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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/wdk/clock"
)

func TestNewProviderRateLimiter(t *testing.T) {
	tests := []struct {
		name        string
		description string
		config      storage_domain.ProviderRateLimitConfig
		expectNil   bool
	}{
		{
			name: "Valid rate limiter with burst",
			config: storage_domain.ProviderRateLimitConfig{
				CallsPerSecond: 10,
				Burst:          20,
			},
			expectNil:   false,
			description: "Should create rate limiter with specified burst",
		},
		{
			name: "Valid rate limiter without burst",
			config: storage_domain.ProviderRateLimitConfig{
				CallsPerSecond: 10,
				Burst:          0,
			},
			expectNil:   false,
			description: "Should create rate limiter with burst equal to calls per second",
		},
		{
			name: "Zero calls per second disables rate limiting",
			config: storage_domain.ProviderRateLimitConfig{
				CallsPerSecond: 0,
				Burst:          10,
			},
			expectNil:   true,
			description: "Should return nil when rate limiting is disabled",
		},
		{
			name: "Negative calls per second disables rate limiting",
			config: storage_domain.ProviderRateLimitConfig{
				CallsPerSecond: -1,
				Burst:          10,
			},
			expectNil:   true,
			description: "Should return nil for negative rate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := storage_domain.NewProviderRateLimiter(tt.config)

			if tt.expectNil {
				assert.Nil(t, limiter, tt.description)
			} else {
				require.NotNil(t, limiter, tt.description)
			}
		})
	}
}

func TestProviderRateLimiter_Wait(t *testing.T) {
	t.Run("Nil limiter returns immediately", func(t *testing.T) {
		var limiter *storage_domain.ProviderRateLimiter

		err := limiter.Wait(context.Background())
		assert.NoError(t, err)
	})

	t.Run("Wait respects rate limit", func(t *testing.T) {
		mockClk := clock.NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

		config := storage_domain.ProviderRateLimitConfig{
			CallsPerSecond: 10,
			Burst:          1,
			Clock:          mockClk,
		}
		limiter := storage_domain.NewProviderRateLimiter(config)
		require.NotNil(t, limiter)

		require.NoError(t, limiter.Wait(context.Background()))

		snap := mockClk.TimerCount()
		waitDone := make(chan error, 1)
		go func() {
			waitDone <- limiter.Wait(context.Background())
		}()

		require.True(t, mockClk.AwaitTimerSetup(snap, time.Second), "timer should be set up")

		mockClk.Advance(100 * time.Millisecond)

		select {
		case err := <-waitDone:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Wait did not return after advancing clock past refill interval")
		}
	})

	t.Run("Wait respects context cancellation", func(t *testing.T) {
		mockClk := clock.NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

		config := storage_domain.ProviderRateLimitConfig{
			CallsPerSecond: 1,
			Burst:          1,
			Clock:          mockClk,
		}
		limiter := storage_domain.NewProviderRateLimiter(config)
		require.NotNil(t, limiter)

		require.NoError(t, limiter.Wait(context.Background()))

		snap := mockClk.TimerCount()
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(fmt.Errorf("test: cleanup"))

		waitDone := make(chan error, 1)
		go func() {
			waitDone <- limiter.Wait(ctx)
		}()

		require.True(t, mockClk.AwaitTimerSetup(snap, time.Second), "timer should be set up")

		cancel(fmt.Errorf("test: simulating cancelled context"))

		select {
		case err := <-waitDone:
			assert.Error(t, err)
		case <-time.After(time.Second):
			t.Fatal("Wait did not return after context cancellation")
		}
	})
}

func TestProviderRateLimiter_Allow(t *testing.T) {
	t.Run("Nil limiter always allows", func(t *testing.T) {
		var limiter *storage_domain.ProviderRateLimiter

		allowed := limiter.Allow()
		assert.True(t, allowed, "Nil limiter should always allow")

		for range 10 {
			assert.True(t, limiter.Allow())
		}
	})

	t.Run("Allow respects rate limit", func(t *testing.T) {
		config := storage_domain.ProviderRateLimitConfig{
			CallsPerSecond: 10,
			Burst:          2,
		}
		limiter := storage_domain.NewProviderRateLimiter(config)
		require.NotNil(t, limiter)

		assert.True(t, limiter.Allow(), "First call should be allowed")
		assert.True(t, limiter.Allow(), "Second call should be allowed")

		assert.False(t, limiter.Allow(), "Third call should be rate limited")
	})
}

func TestWithRateLimit(t *testing.T) {
	option := storage_domain.WithRateLimit(100, 200)
	require.NotNil(t, option)

	opts := &storage_domain.ProviderOptions{}
	option(opts)

	assert.Equal(t, float64(100), opts.RateLimitConfig.CallsPerSecond)
	assert.Equal(t, 200, opts.RateLimitConfig.Burst)
}

func TestWithUnlimitedRate(t *testing.T) {
	option := storage_domain.WithUnlimitedRate()
	require.NotNil(t, option)

	opts := &storage_domain.ProviderOptions{}
	option(opts)

	assert.Equal(t, float64(0), opts.RateLimitConfig.CallsPerSecond)
	assert.Equal(t, 0, opts.RateLimitConfig.Burst)
}

func TestApplyProviderOptions(t *testing.T) {
	t.Run("No options uses defaults", func(t *testing.T) {
		defaults := storage_domain.ProviderRateLimitConfig{
			CallsPerSecond: 50,
			Burst:          100,
		}

		limiter := storage_domain.ApplyProviderOptions(defaults)
		assert.NotNil(t, limiter)
	})

	t.Run("Options override defaults", func(t *testing.T) {
		defaults := storage_domain.ProviderRateLimitConfig{
			CallsPerSecond: 50,
			Burst:          100,
		}

		limiter := storage_domain.ApplyProviderOptions(
			defaults,
			storage_domain.WithRateLimit(200, 300),
		)
		assert.NotNil(t, limiter)
	})

	t.Run("Unlimited rate option returns nil limiter", func(t *testing.T) {
		defaults := storage_domain.ProviderRateLimitConfig{
			CallsPerSecond: 50,
			Burst:          100,
		}

		limiter := storage_domain.ApplyProviderOptions(
			defaults,
			storage_domain.WithUnlimitedRate(),
		)
		assert.Nil(t, limiter)
	})

	t.Run("Multiple options applied in order", func(t *testing.T) {
		defaults := storage_domain.ProviderRateLimitConfig{
			CallsPerSecond: 50,
			Burst:          100,
		}

		limiter := storage_domain.ApplyProviderOptions(
			defaults,
			storage_domain.WithRateLimit(200, 300),
			storage_domain.WithUnlimitedRate(),
		)
		assert.Nil(t, limiter, "Last option should win")
	})
}

func TestProviderRateLimitConfig(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		config := storage_domain.ProviderRateLimitConfig{
			CallsPerSecond: 100,
			Burst:          200,
		}

		assert.Equal(t, float64(100), config.CallsPerSecond)
		assert.Equal(t, 200, config.Burst)
	})

	t.Run("Zero values", func(t *testing.T) {
		config := storage_domain.ProviderRateLimitConfig{}

		assert.Equal(t, float64(0), config.CallsPerSecond)
		assert.Equal(t, 0, config.Burst)
	})
}
