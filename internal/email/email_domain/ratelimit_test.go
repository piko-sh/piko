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

package email_domain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/clock"
)

func TestNewProviderRateLimiter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		config    ProviderRateLimitConfig
		expectNil bool
	}{
		{
			name: "Standard Case - Valid rate and burst",
			config: ProviderRateLimitConfig{
				CallsPerSecond: 10,
				Burst:          20,
			},
			expectNil: false,
		},
		{
			name: "Standard Case - Burst defaults to CallsPerSecond",
			config: ProviderRateLimitConfig{
				CallsPerSecond: 5,
				Burst:          0,
			},
			expectNil: false,
		},
		{
			name: "Edge Case - Zero CallsPerSecond should be unlimited",
			config: ProviderRateLimitConfig{
				CallsPerSecond: 0,
				Burst:          100,
			},
			expectNil: true,
		},
		{
			name: "Edge Case - Negative CallsPerSecond should be unlimited",
			config: ProviderRateLimitConfig{
				CallsPerSecond: -5.0,
				Burst:          10,
			},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := newProviderRateLimiter(tc.config)

			if tc.expectNil {
				require.Nil(t, limiter, "Expected a nil limiter for unlimited rate configuration")
			} else {
				require.NotNil(t, limiter, "Expected a non-nil limiter")
				require.NotNil(t, limiter.limiter, "Internal limiter should not be nil")
			}
		})
	}
}

func TestProviderRateLimiter_Wait(t *testing.T) {
	t.Parallel()

	t.Run("Allows burst and then blocks until token refill", func(t *testing.T) {
		t.Parallel()
		ratePerSecond := 10.0
		burst := 3
		mockClk := clock.NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

		limiter := newProviderRateLimiter(ProviderRateLimitConfig{
			CallsPerSecond: ratePerSecond,
			Burst:          burst,
			Clock:          mockClk,
		})
		require.NotNil(t, limiter)

		ctx := context.Background()

		for range burst {
			require.NoError(t, limiter.Wait(ctx))
		}

		snap := mockClk.TimerCount()
		waitDone := make(chan error, 1)
		go func() {
			waitDone <- limiter.Wait(ctx)
		}()

		require.True(t, mockClk.AwaitTimerSetup(snap, time.Second), "timer should be set up")

		mockClk.Advance(time.Duration(float64(time.Second) / ratePerSecond))

		select {
		case err := <-waitDone:
			require.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Wait did not return after advancing clock past refill interval")
		}
	})

	t.Run("Unlimited rate returns immediately without error", func(t *testing.T) {
		t.Parallel()
		var limiter *ProviderRateLimiter

		err := limiter.Wait(context.Background())
		require.NoError(t, err)
	})

	t.Run("Respects context cancellation", func(t *testing.T) {
		t.Parallel()
		mockClk := clock.NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

		limiter := newProviderRateLimiter(ProviderRateLimitConfig{
			CallsPerSecond: 10,
			Burst:          1,
			Clock:          mockClk,
		})
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
			require.Error(t, err)
		case <-time.After(time.Second):
			t.Fatal("Wait did not return after context cancellation")
		}
	})
}

func TestApplyProviderOptions(t *testing.T) {
	t.Parallel()

	defaultConfig := ProviderRateLimitConfig{
		CallsPerSecond: 1000,
		Burst:          1000,
	}

	testCases := []struct {
		name          string
		opts          []ProviderOption
		expectNil     bool
		expectedBurst int
	}{
		{
			name:          "No options provided - uses defaults",
			opts:          []ProviderOption{},
			expectNil:     false,
			expectedBurst: 1000,
		},
		{
			name: "withRateLimit option overrides defaults",
			opts: []ProviderOption{
				withRateLimit(50, 25),
			},
			expectNil:     false,
			expectedBurst: 25,
		},
		{
			name: "withUnlimitedRate option overrides defaults",
			opts: []ProviderOption{
				withUnlimitedRate(),
			},
			expectNil: true,
		},
		{
			name: "Multiple options - last one wins (withRateLimit)",
			opts: []ProviderOption{
				withUnlimitedRate(),
				withRateLimit(10, 5),
			},
			expectNil:     false,
			expectedBurst: 5,
		},
		{
			name: "Multiple options - last one wins (withUnlimitedRate)",
			opts: []ProviderOption{
				withRateLimit(10, 5),
				withUnlimitedRate(),
			},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := ApplyProviderOptions(defaultConfig, tc.opts...)

			if tc.expectNil {
				require.Nil(t, limiter, "Expected a nil limiter based on options")
				err := limiter.Wait(context.Background())
				require.NoError(t, err)
				return
			}

			require.NotNil(t, limiter, "Expected a non-nil limiter")
			require.NotNil(t, limiter.limiter, "Internal limiter should not be nil")

			ctx := context.Background()
			for i := 0; i < tc.expectedBurst; i++ {
				_ = limiter.Wait(ctx)
			}
		})
	}
}
