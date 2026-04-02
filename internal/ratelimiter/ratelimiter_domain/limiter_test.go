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

package ratelimiter_domain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

func TestAllowTokenBucket(t *testing.T) {
	config := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}

	testCases := []struct {
		storeErr    error
		expectErr   error
		name        string
		storeAllow  bool
		failPolicy  ratelimiter_dto.FailPolicy
		expectAllow bool
	}{
		{
			name:        "allowed when store returns true",
			storeAllow:  true,
			expectAllow: true,
		},
		{
			name:       "denied when store returns false",
			storeAllow: false,
			expectErr:  ErrRateLimited,
		},
		{
			name:        "fail open allows on store error",
			storeErr:    errors.New("connection refused"),
			failPolicy:  ratelimiter_dto.FailOpen,
			expectAllow: true,
		},
		{
			name:       "fail closed denies on store error",
			storeErr:   errors.New("connection refused"),
			failPolicy: ratelimiter_dto.FailClosed,
			expectErr:  ErrStoreFailure,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := &MockTokenBucketStore{
				TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
					return tc.storeAllow, tc.storeErr
				},
			}

			limiter := NewLimiter(store, nil, WithFailPolicy(tc.failPolicy))
			err := limiter.AllowTokenBucket(context.Background(), "test-key", 1.0, config)

			if tc.expectErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tc.expectErr), "expected %v, got %v", tc.expectErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAllowTokenBucket_KeyPrefix(t *testing.T) {
	var capturedKey string
	store := &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, key string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			capturedKey = key
			return true, nil
		},
	}
	limiter := NewLimiter(store, nil, WithKeyPrefix("myprefix"))

	config := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	err := limiter.AllowTokenBucket(context.Background(), "user:123", 1.0, config)

	assert.NoError(t, err)
	assert.Equal(t, "myprefix:user:123", capturedKey)
}

func TestAllowTokenBucket_NoPrefix(t *testing.T) {
	var capturedKey string
	store := &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, key string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			capturedKey = key
			return true, nil
		},
	}
	limiter := NewLimiter(store, nil)

	config := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	err := limiter.AllowTokenBucket(context.Background(), "user:123", 1.0, config)

	assert.NoError(t, err)
	assert.Equal(t, "user:123", capturedKey)
}

func TestAllowTokenBucket_ContextCancelled(t *testing.T) {
	store := &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			return false, context.Canceled
		},
	}
	limiter := NewLimiter(store, nil, WithFailPolicy(ratelimiter_dto.FailClosed))

	config := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := limiter.AllowTokenBucket(ctx, "test", 1.0, config)
	assert.Error(t, err)
}

func TestWaitTokenBucket_ImmediateAllow(t *testing.T) {
	store := &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			return true, nil
		},
	}
	limiter := NewLimiter(store, nil)

	config := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	err := limiter.WaitTokenBucket(context.Background(), "test", 1.0, config)

	assert.NoError(t, err)
}

func TestWaitTokenBucket_ContextCancellation(t *testing.T) {
	store := &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			return false, nil
		},
		WaitDurationFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
			return time.Hour, nil
		},
	}
	mockClock := clock.NewMockClock(time.Now())
	limiter := NewLimiter(store, nil, WithClock(mockClock))

	config := ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}
	ctx, cancel := context.WithCancelCause(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- limiter.WaitTokenBucket(ctx, "test", 1.0, config)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel(fmt.Errorf("test: simulating cancelled context"))

	select {
	case err := <-done:
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
	case <-time.After(time.Second):
		t.Fatal("WaitTokenBucket did not return after context cancellation")
	}
}

func TestWaitTokenBucket_EventualAllow(t *testing.T) {
	var mu sync.Mutex
	allowed := false

	store := &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			mu.Lock()
			defer mu.Unlock()
			return allowed, nil
		},
		WaitDurationFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
			return 10 * time.Millisecond, nil
		},
	}
	mockClock := clock.NewMockClock(time.Now())
	limiter := NewLimiter(store, nil, WithClock(mockClock))

	config := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}

	done := make(chan error, 1)
	go func() {
		done <- limiter.WaitTokenBucket(context.Background(), "test", 1.0, config)
	}()

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	allowed = true
	mu.Unlock()

	mockClock.Advance(20 * time.Millisecond)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("WaitTokenBucket did not return after tokens became available")
	}
}

func TestCheckFixedWindow(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	windowStart := now.Add(-30 * time.Second)
	config := ratelimiter_dto.FixedWindowConfig{Limit: 10, Window: time.Minute}

	testCases := []struct {
		storeErr          error
		name              string
		storeCount        int64
		expectedRemaining int
		failPolicy        ratelimiter_dto.FailPolicy
		expectedAllowed   bool
		expectErr         bool
	}{
		{
			name:              "first request allowed",
			storeCount:        1,
			expectedAllowed:   true,
			expectedRemaining: 9,
		},
		{
			name:              "at limit still allowed",
			storeCount:        10,
			expectedAllowed:   true,
			expectedRemaining: 0,
		},
		{
			name:              "over limit denied",
			storeCount:        11,
			expectedAllowed:   false,
			expectedRemaining: 0,
		},
		{
			name:              "fail open allows on store error",
			storeErr:          errors.New("cache unavailable"),
			failPolicy:        ratelimiter_dto.FailOpen,
			expectedAllowed:   true,
			expectedRemaining: 9,
		},
		{
			name:       "fail closed errors on store error",
			storeErr:   errors.New("cache unavailable"),
			failPolicy: ratelimiter_dto.FailClosed,
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClock := clock.NewMockClock(now)
			count := tc.storeCount - 1
			counter := &MockCounterStore{
				IncrementAndGetFunc: func(_ context.Context, _ string, delta int64, _ time.Duration) (ratelimiter_dto.CounterResult, error) {
					if tc.storeErr != nil {
						return ratelimiter_dto.CounterResult{}, tc.storeErr
					}
					count += delta
					return ratelimiter_dto.CounterResult{
						Count:       count,
						WindowStart: windowStart,
					}, nil
				},
			}
			limiter := NewLimiter(nil, counter,
				WithClock(mockClock),
				WithFailPolicy(tc.failPolicy),
			)

			result, err := limiter.CheckFixedWindow(context.Background(), "ip:192.168.1.1", config)

			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedAllowed, result.Allowed)
			assert.Equal(t, tc.expectedRemaining, result.Remaining)
			assert.Equal(t, config.Limit, result.Limit)
			assert.False(t, result.ResetAt.IsZero())

			if !result.Allowed {

				assert.Equal(t, 30*time.Second, result.RetryAfter)
				assert.Equal(t, windowStart.Add(config.Window), result.ResetAt)
			} else {
				assert.Zero(t, result.RetryAfter)
			}
		})
	}
}

func TestCheckFixedWindow_KeyPrefix(t *testing.T) {
	var capturedKey string
	counter := &MockCounterStore{
		IncrementAndGetFunc: func(_ context.Context, key string, _ int64, _ time.Duration) (ratelimiter_dto.CounterResult, error) {
			capturedKey = key
			return ratelimiter_dto.CounterResult{
				Count:       1,
				WindowStart: time.Now(),
			}, nil
		},
	}
	limiter := NewLimiter(nil, counter, WithKeyPrefix("security"))

	config := ratelimiter_dto.FixedWindowConfig{Limit: 100, Window: time.Minute}
	_, err := limiter.CheckFixedWindow(context.Background(), "ip:10.0.0.1", config)

	assert.NoError(t, err)
	assert.Equal(t, "security:ip:10.0.0.1", capturedKey)
}

func TestDeleteBucket(t *testing.T) {
	var capturedKey string
	store := &MockTokenBucketStore{
		DeleteBucketFunc: func(_ context.Context, key string) error {
			capturedKey = key
			return nil
		},
	}
	limiter := NewLimiter(store, nil, WithKeyPrefix("llm"))

	err := limiter.DeleteBucket(context.Background(), "scope:request")

	assert.NoError(t, err)
	assert.Equal(t, "llm:scope:request", capturedKey)
}

func TestDeleteBucket_StoreError(t *testing.T) {
	store := &MockTokenBucketStore{
		DeleteBucketFunc: func(_ context.Context, _ string) error {
			return errors.New("delete failed")
		},
	}
	limiter := NewLimiter(store, nil)

	err := limiter.DeleteBucket(context.Background(), "test")

	assert.Error(t, err)
}

func TestNewLimiter_DefaultOptions(t *testing.T) {
	limiter := NewLimiter(nil, nil)

	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.clock)
	assert.Equal(t, ratelimiter_dto.FailOpen, limiter.failPolicy)
	assert.Empty(t, limiter.keyPrefix)
}

func TestNewLimiter_WithOptions(t *testing.T) {
	mockClock := clock.NewMockClock(time.Now())
	limiter := NewLimiter(nil, nil,
		WithClock(mockClock),
		WithFailPolicy(ratelimiter_dto.FailClosed),
		WithKeyPrefix("test"),
	)

	assert.Equal(t, ratelimiter_dto.FailClosed, limiter.failPolicy)
	assert.Equal(t, "test", limiter.keyPrefix)
}

func TestGetStatus(t *testing.T) {
	t.Parallel()

	store := &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			return true, nil
		},
	}
	counter := &MockCounterStore{}
	limiter := NewLimiter(store, counter,
		WithTokenStoreName("cache"),
		WithCounterStoreName("cache"),
		WithKeyPrefix("test"),
		WithFailPolicy(ratelimiter_dto.FailClosed),
	)

	config := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	for range 3 {
		_ = limiter.AllowTokenBucket(context.Background(), "key", 1.0, config)
	}

	fwConfig := ratelimiter_dto.FixedWindowConfig{Limit: 100, Window: time.Minute}
	for range 2 {
		_, _ = limiter.CheckFixedWindow(context.Background(), "fw-key", fwConfig)
	}

	status, err := limiter.GetStatus(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "cache", status.TokenBucketStore)
	assert.Equal(t, "cache", status.CounterStore)
	assert.Equal(t, "closed", status.FailPolicy)
	assert.Equal(t, "test", status.KeyPrefix)
	assert.Equal(t, int64(5), status.TotalChecks)
	assert.Equal(t, int64(5), status.TotalAllowed)
	assert.Equal(t, int64(0), status.TotalDenied)
	assert.Equal(t, int64(0), status.TotalErrors)
}

func TestGetStatus_WithDenials(t *testing.T) {
	t.Parallel()

	store := &MockTokenBucketStore{}
	limiter := NewLimiter(store, nil,
		WithTokenStoreName("inmemory"),
		WithCounterStoreName("noop"),
	)

	config := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	for range 4 {
		_ = limiter.AllowTokenBucket(context.Background(), "key", 1.0, config)
	}

	status, err := limiter.GetStatus(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "inmemory", status.TokenBucketStore)
	assert.Equal(t, "noop", status.CounterStore)
	assert.Equal(t, "open", status.FailPolicy)
	assert.Empty(t, status.KeyPrefix)
	assert.Equal(t, int64(4), status.TotalChecks)
	assert.Equal(t, int64(0), status.TotalAllowed)
	assert.Equal(t, int64(4), status.TotalDenied)
	assert.Equal(t, int64(0), status.TotalErrors)
}

func TestGetStatus_WithErrors(t *testing.T) {
	t.Parallel()

	store := &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			return false, errors.New("connection refused")
		},
	}
	limiter := NewLimiter(store, nil,
		WithFailPolicy(ratelimiter_dto.FailClosed),
		WithTokenStoreName("cache"),
		WithCounterStoreName("cache"),
	)

	config := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	for range 2 {
		_ = limiter.AllowTokenBucket(context.Background(), "key", 1.0, config)
	}

	status, err := limiter.GetStatus(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(2), status.TotalChecks)
	assert.Equal(t, int64(0), status.TotalAllowed)
	assert.Equal(t, int64(2), status.TotalDenied)
	assert.Equal(t, int64(2), status.TotalErrors)
}

func TestBuildKey(t *testing.T) {
	testCases := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "no prefix",
			prefix:   "",
			key:      "user:123",
			expected: "user:123",
		},
		{
			name:     "with prefix",
			prefix:   "email",
			key:      "smtp:provider",
			expected: "email:smtp:provider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := &Limiter{keyPrefix: tc.prefix}
			assert.Equal(t, tc.expected, limiter.buildKey(tc.key))
		})
	}
}
