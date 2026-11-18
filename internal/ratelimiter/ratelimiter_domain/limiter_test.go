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
	"math"
	"sync"
	"sync/atomic"
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			store := &MockTokenBucketStore{
				TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
					return testCase.storeAllow, testCase.storeErr
				},
			}

			limiter := NewLimiter(store, nil, WithFailPolicy(testCase.failPolicy))
			err := limiter.AllowTokenBucket(t.Context(), "test-key", 1.0, config)

			if testCase.expectErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, testCase.expectErr), "expected %v, got %v", testCase.expectErr, err)
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
	err := limiter.AllowTokenBucket(t.Context(), "user:123", 1.0, config)

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
	err := limiter.AllowTokenBucket(t.Context(), "user:123", 1.0, config)

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
	ctx, cancel := context.WithCancelCause(t.Context())
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
	err := limiter.WaitTokenBucket(t.Context(), "test", 1.0, config)

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
	ctx, cancel := context.WithCancelCause(t.Context())

	baseline := mockClock.TimerCount()
	done := make(chan error, 1)
	go func() {
		done <- limiter.WaitTokenBucket(ctx, "test", 1.0, config)
	}()

	require.True(t, mockClock.AwaitTimerSetup(baseline, time.Second), "WaitTokenBucket must start waiting")
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

	baseline := mockClock.TimerCount()
	done := make(chan error, 1)
	go func() {
		done <- limiter.WaitTokenBucket(t.Context(), "test", 1.0, config)
	}()

	require.True(t, mockClock.AwaitTimerSetup(baseline, time.Second), "WaitTokenBucket must start waiting")

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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mockClock := clock.NewMockClock(now)
			count := testCase.storeCount - 1
			counter := &MockCounterStore{
				IncrementAndGetFunc: func(_ context.Context, _ string, delta int64, _ time.Duration) (ratelimiter_dto.CounterResult, error) {
					if testCase.storeErr != nil {
						return ratelimiter_dto.CounterResult{}, testCase.storeErr
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
				WithFailPolicy(testCase.failPolicy),
			)

			result, err := limiter.CheckFixedWindow(t.Context(), "ip:192.168.1.1", config)

			if testCase.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedAllowed, result.Allowed)
			assert.Equal(t, testCase.expectedRemaining, result.Remaining)
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
	_, err := limiter.CheckFixedWindow(t.Context(), "ip:10.0.0.1", config)

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

	err := limiter.DeleteBucket(t.Context(), "scope:request")

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

	err := limiter.DeleteBucket(t.Context(), "test")

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
		_ = limiter.AllowTokenBucket(t.Context(), "key", 1.0, config)
	}

	fwConfig := ratelimiter_dto.FixedWindowConfig{Limit: 100, Window: time.Minute}
	for range 2 {
		_, _ = limiter.CheckFixedWindow(t.Context(), "fw-key", fwConfig)
	}

	status, err := limiter.GetStatus(t.Context())

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
		_ = limiter.AllowTokenBucket(t.Context(), "key", 1.0, config)
	}

	status, err := limiter.GetStatus(t.Context())

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
		_ = limiter.AllowTokenBucket(t.Context(), "key", 1.0, config)
	}

	status, err := limiter.GetStatus(t.Context())

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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			limiter := &Limiter{keyPrefix: testCase.prefix}
			assert.Equal(t, testCase.expected, limiter.buildKey(testCase.key))
		})
	}
}

func TestCheckFixedWindow_AllowsAgainAfterWindow(t *testing.T) {
	t.Parallel()

	const (
		windowKey   = "ratelimit:global:203.0.113.7"
		windowLimit = 5
		window      = time.Minute
	)

	testCases := []struct {
		name        string
		advance     time.Duration
		wantRemain  int
		wantAllowed bool
	}{
		{
			name:        "still denied before the window ends",
			advance:     window - time.Second,
			wantAllowed: false,
			wantRemain:  0,
		},
		{
			name:        "allowed once the window ends",
			advance:     window,
			wantAllowed: true,
			wantRemain:  windowLimit - 1,
		},
		{
			name:        "allowed long after the window ends",
			advance:     time.Hour,
			wantAllowed: true,
			wantRemain:  windowLimit - 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
			limiter := NewLimiter(nil, newWindowCounterStore(mockClock), WithClock(mockClock))
			config := ratelimiter_dto.FixedWindowConfig{Limit: windowLimit, Window: window}

			for range windowLimit {
				result, err := limiter.CheckFixedWindow(ctx, windowKey, config)
				require.NoError(t, err)
				require.True(t, result.Allowed)
			}
			denied, err := limiter.CheckFixedWindow(ctx, windowKey, config)
			require.NoError(t, err)
			require.False(t, denied.Allowed, "the request after the limit is denied")

			mockClock.Advance(testCase.advance)

			result, err := limiter.CheckFixedWindow(ctx, windowKey, config)
			require.NoError(t, err)
			assert.Equal(t, testCase.wantAllowed, result.Allowed)
			assert.Equal(t, testCase.wantRemain, result.Remaining)
			assert.True(t, result.ResetAt.After(mockClock.Now()), "the reset time is never in the past")
		})
	}
}

func TestCheckFixedWindow_RejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		config ratelimiter_dto.FixedWindowConfig
	}{
		{name: "negative limit", config: ratelimiter_dto.FixedWindowConfig{Limit: -1, Window: time.Minute}},
		{name: "zero window", config: ratelimiter_dto.FixedWindowConfig{Limit: 10, Window: 0}},
		{name: "negative window", config: ratelimiter_dto.FixedWindowConfig{Limit: 10, Window: -time.Second}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			store := &MockCounterStore{}
			limiter := NewLimiter(nil, store)

			result, err := limiter.CheckFixedWindow(t.Context(), "key", testCase.config)

			require.ErrorIs(t, err, ErrInvalidConfig)
			assert.Contains(t, err.Error(), fmt.Sprintf("limit %d", testCase.config.Limit))
			assert.False(t, result.Allowed)
			assert.Zero(t, store.IncrementAndGetCallCount.Load(), "the store is not touched")
			status, statusErr := limiter.GetStatus(t.Context())
			require.NoError(t, statusErr)
			assert.Zero(t, status.TotalChecks, "an invalid config is not counted as a check")
		})
	}
}

func TestCheckFixedWindow_ZeroLimitDeniesEveryRequest(t *testing.T) {
	t.Parallel()

	var count int64
	store := &MockCounterStore{
		IncrementAndGetFunc: func(_ context.Context, _ string, delta int64, _ time.Duration) (ratelimiter_dto.CounterResult, error) {
			count += delta
			return ratelimiter_dto.CounterResult{Count: count}, nil
		},
	}
	limiter := NewLimiter(nil, store)
	config := ratelimiter_dto.FixedWindowConfig{Limit: 0, Window: time.Minute}

	for attempt := range 3 {
		result, err := limiter.CheckFixedWindow(t.Context(), "key", config)

		require.NoError(t, err, "a limit of zero is a valid block-everything setting")
		assert.False(t, result.Allowed, "attempt %d must be denied", attempt+1)
		assert.Zero(t, result.Remaining)
	}
}

func TestAllowTokenBucket_RejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		config ratelimiter_dto.TokenBucketConfig
	}{
		{name: "zero rate", config: ratelimiter_dto.TokenBucketConfig{Rate: 0, Burst: 1}},
		{name: "negative rate", config: ratelimiter_dto.TokenBucketConfig{Rate: -1, Burst: 1}},
		{name: "not a number rate", config: ratelimiter_dto.TokenBucketConfig{Rate: math.NaN(), Burst: 1}},
		{name: "infinite rate", config: ratelimiter_dto.TokenBucketConfig{Rate: math.Inf(1), Burst: 1}},
		{name: "negative burst", config: ratelimiter_dto.TokenBucketConfig{Rate: 1, Burst: -1}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			store := &MockTokenBucketStore{}
			limiter := NewLimiter(store, nil)

			err := limiter.AllowTokenBucket(t.Context(), "key", 1, testCase.config)
			require.ErrorIs(t, err, ErrInvalidConfig)

			waitErr := limiter.WaitTokenBucket(t.Context(), "key", 1, testCase.config)
			require.ErrorIs(t, waitErr, ErrInvalidConfig)

			assert.Zero(t, store.TryTakeCallCount.Load(), "the store is not touched")
		})
	}
}

func TestAllowTokenBucket_RejectsInvalidTokenCount(t *testing.T) {
	t.Parallel()

	config := ratelimiter_dto.TokenBucketConfig{Rate: 1, Burst: 2}
	testCases := []struct {
		name   string
		tokens float64
	}{
		{name: "not a number", tokens: math.NaN()},
		{name: "positive infinity", tokens: math.Inf(1)},
		{name: "negative infinity", tokens: math.Inf(-1)},
		{name: "negative", tokens: -1},
		{name: "more than the bucket holds", tokens: 2.5},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			store := &MockTokenBucketStore{}
			limiter := NewLimiter(store, nil)

			err := limiter.AllowTokenBucket(t.Context(), "key", testCase.tokens, config)
			require.ErrorIs(t, err, ErrInvalidConfig)

			ctx, cancel := context.WithTimeoutCause(t.Context(), time.Second, errors.New("test: wait did not return"))
			defer cancel()
			waitErr := limiter.WaitTokenBucket(ctx, "key", testCase.tokens, config)
			require.ErrorIs(t, waitErr, ErrInvalidConfig, "a request the bucket can never satisfy must not wait")

			assert.Zero(t, store.TryTakeCallCount.Load(), "the store is not touched")
		})
	}
}

func TestAllowTokenBucket_AcceptsBoundaryTokenCounts(t *testing.T) {
	t.Parallel()

	store := &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			return true, nil
		},
	}
	limiter := NewLimiter(store, nil)
	config := ratelimiter_dto.TokenBucketConfig{Rate: 1, Burst: 2}

	require.NoError(t, limiter.AllowTokenBucket(t.Context(), "key", 0, config))
	require.NoError(t, limiter.AllowTokenBucket(t.Context(), "key", 2, config))
}

func TestEstimateWaitDuration_NeverReturnsANonPositiveWait(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err      error
		name     string
		estimate time.Duration
		want     time.Duration
	}{
		{name: "store estimate is kept", estimate: time.Second, want: time.Second},
		{name: "zero waits briefly", estimate: 0, want: time.Millisecond},
		{name: "negative waits briefly", estimate: math.MinInt64, want: time.Millisecond},
		{name: "store error waits briefly", estimate: time.Hour, err: errors.New("store down"), want: time.Millisecond},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			store := &MockTokenBucketStore{
				WaitDurationFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
					return testCase.estimate, testCase.err
				},
			}
			limiter := NewLimiter(store, nil)

			got := limiter.estimateWaitDuration(t.Context(), "key", 1, ratelimiter_dto.TokenBucketConfig{Rate: 1, Burst: 1})

			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestTokenWaitDuration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		needed     float64
		refillRate float64
		want       time.Duration
	}{
		{name: "nothing needed", needed: 0, refillRate: 1, want: 0},
		{name: "exact wait", needed: 10, refillRate: 2, want: 5},
		{name: "fraction rounds up", needed: 1, refillRate: 3, want: 1},
		{name: "overflow is clamped", needed: 1, refillRate: 1e-300, want: maxTokenWait},
		{name: "zero rate is clamped", needed: 1, refillRate: 0, want: maxTokenWait},
		{name: "not a number rate is clamped", needed: 1, refillRate: math.NaN(), want: maxTokenWait},
		{name: "not a number need is clamped", needed: math.NaN(), refillRate: 1, want: maxTokenWait},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, TokenWaitDuration(testCase.needed, testCase.refillRate))
		})
	}
}

func newWindowCounterStore(mockClock *clock.MockClock) *MockCounterStore {
	var mu sync.Mutex
	counters := make(map[string]ratelimiter_dto.CounterResult)

	return &MockCounterStore{
		IncrementAndGetFunc: func(_ context.Context, key string, delta int64, window time.Duration) (ratelimiter_dto.CounterResult, error) {
			mu.Lock()
			defer mu.Unlock()

			now := mockClock.Now()
			current, found := counters[key]
			if !found || now.Sub(current.WindowStart) >= window {
				current = ratelimiter_dto.CounterResult{Count: 0, WindowStart: now}
			}
			current.Count += delta
			counters[key] = current
			return current, nil
		},
	}
}

func TestGetStatus_CountsEveryPath(t *testing.T) {
	t.Parallel()

	storeErr := errors.New("connection refused")
	tokenConfig := ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	windowConfig := ratelimiter_dto.FixedWindowConfig{Limit: 1, Window: time.Minute}

	tokenStore := func(allowed bool, err error) *MockTokenBucketStore {
		return &MockTokenBucketStore{
			TryTakeFunc: func(context.Context, string, float64, *ratelimiter_dto.TokenBucketConfig) (bool, error) {
				return allowed, err
			},
		}
	}
	counterStore := func(count int64, err error) *MockCounterStore {
		return &MockCounterStore{
			IncrementAndGetFunc: func(context.Context, string, int64, time.Duration) (ratelimiter_dto.CounterResult, error) {
				return ratelimiter_dto.CounterResult{Count: count}, err
			},
		}
	}

	testCases := []struct {
		call       func(*Limiter)
		tokens     *MockTokenBucketStore
		counters   *MockCounterStore
		name       string
		policy     ratelimiter_dto.FailPolicy
		wantChecks int64
		wantAllow  int64
		wantDeny   int64
		wantErrors int64
	}{
		{
			name:   "token bucket allowed",
			tokens: tokenStore(true, nil),
			call: func(l *Limiter) {
				_ = l.AllowTokenBucket(context.Background(), "key", 1, tokenConfig)
			},
			wantChecks: 1, wantAllow: 1,
		},
		{
			name:   "token bucket denied",
			tokens: tokenStore(false, nil),
			call: func(l *Limiter) {
				_ = l.AllowTokenBucket(context.Background(), "key", 1, tokenConfig)
			},
			wantChecks: 1, wantDeny: 1,
		},
		{
			name:   "token bucket store error fail open",
			tokens: tokenStore(false, storeErr),
			policy: ratelimiter_dto.FailOpen,
			call: func(l *Limiter) {
				_ = l.AllowTokenBucket(context.Background(), "key", 1, tokenConfig)
			},
			wantChecks: 1, wantAllow: 1, wantErrors: 1,
		},
		{
			name:   "token bucket store error fail closed",
			tokens: tokenStore(false, storeErr),
			policy: ratelimiter_dto.FailClosed,
			call: func(l *Limiter) {
				_ = l.AllowTokenBucket(context.Background(), "key", 1, tokenConfig)
			},
			wantChecks: 1, wantDeny: 1, wantErrors: 1,
		},
		{
			name:   "token bucket invalid config",
			tokens: tokenStore(true, nil),
			call: func(l *Limiter) {
				_ = l.AllowTokenBucket(context.Background(), "key", 1, ratelimiter_dto.TokenBucketConfig{Rate: 0, Burst: 1})
			},
		},
		{
			name:   "token bucket invalid token count",
			tokens: tokenStore(true, nil),
			call: func(l *Limiter) {
				_ = l.AllowTokenBucket(context.Background(), "key", math.NaN(), tokenConfig)
			},
		},
		{
			name:     "fixed window allowed",
			counters: counterStore(1, nil),
			call: func(l *Limiter) {
				_, _ = l.CheckFixedWindow(context.Background(), "key", windowConfig)
			},
			wantChecks: 1, wantAllow: 1,
		},
		{
			name:     "fixed window denied",
			counters: counterStore(2, nil),
			call: func(l *Limiter) {
				_, _ = l.CheckFixedWindow(context.Background(), "key", windowConfig)
			},
			wantChecks: 1, wantDeny: 1,
		},
		{
			name:     "fixed window store error fail open",
			counters: counterStore(0, storeErr),
			policy:   ratelimiter_dto.FailOpen,
			call: func(l *Limiter) {
				_, _ = l.CheckFixedWindow(context.Background(), "key", windowConfig)
			},
			wantChecks: 1, wantAllow: 1, wantErrors: 1,
		},
		{
			name:     "fixed window store error fail closed",
			counters: counterStore(0, storeErr),
			policy:   ratelimiter_dto.FailClosed,
			call: func(l *Limiter) {
				_, _ = l.CheckFixedWindow(context.Background(), "key", windowConfig)
			},
			wantChecks: 1, wantDeny: 1, wantErrors: 1,
		},
		{
			name:     "fixed window invalid config",
			counters: counterStore(1, nil),
			call: func(l *Limiter) {
				_, _ = l.CheckFixedWindow(context.Background(), "key", ratelimiter_dto.FixedWindowConfig{Limit: -1, Window: time.Minute})
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			limiter := NewLimiter(testCase.tokens, testCase.counters, WithFailPolicy(testCase.policy))
			testCase.call(limiter)

			status, err := limiter.GetStatus(t.Context())
			require.NoError(t, err)
			assert.Equal(t, testCase.wantChecks, status.TotalChecks, "TotalChecks")
			assert.Equal(t, testCase.wantAllow, status.TotalAllowed, "TotalAllowed")
			assert.Equal(t, testCase.wantDeny, status.TotalDenied, "TotalDenied")
			assert.Equal(t, testCase.wantErrors, status.TotalErrors, "TotalErrors")
			assert.Equal(t, status.TotalAllowed+status.TotalDenied, status.TotalChecks,
				"every check is either allowed or denied")
		})
	}
}

func TestGetStatus_CountsConcurrentChecks(t *testing.T) {
	t.Parallel()

	const (
		workers         = 8
		checksPerWorker = 500
	)

	var calls atomic.Int64
	counters := &MockCounterStore{
		IncrementAndGetFunc: func(context.Context, string, int64, time.Duration) (ratelimiter_dto.CounterResult, error) {
			return ratelimiter_dto.CounterResult{Count: calls.Add(1) % 2}, nil
		},
	}
	limiter := NewLimiter(nil, counters)
	windowConfig := ratelimiter_dto.FixedWindowConfig{Limit: 0, Window: time.Minute}

	var group sync.WaitGroup
	for range workers {
		group.Go(func() {
			for range checksPerWorker {
				_, _ = limiter.CheckFixedWindow(context.Background(), "key", windowConfig)
			}
		})
	}
	group.Wait()

	status, err := limiter.GetStatus(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(workers*checksPerWorker), status.TotalChecks)
	assert.Equal(t, int64(workers*checksPerWorker/2), status.TotalAllowed)
	assert.Equal(t, int64(workers*checksPerWorker/2), status.TotalDenied)
}

func BenchmarkCheckFixedWindow_Parallel(b *testing.B) {
	counters := &MockCounterStore{
		IncrementAndGetFunc: func(context.Context, string, int64, time.Duration) (ratelimiter_dto.CounterResult, error) {
			return ratelimiter_dto.CounterResult{Count: 1}, nil
		},
	}
	limiter := NewLimiter(nil, counters)
	windowConfig := ratelimiter_dto.FixedWindowConfig{Limit: 10, Window: time.Minute}
	ctx := context.Background()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := limiter.CheckFixedWindow(ctx, "key", windowConfig); err != nil {
				b.Error(err)
				return
			}
		}
	})
}
