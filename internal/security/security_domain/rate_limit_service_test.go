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

package security_domain

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

type counterState struct {
	err          error
	counts       map[string]int64
	windowStarts map[string]time.Time
	mu           sync.Mutex
}

func newStatefulCounter() (*ratelimiter_domain.MockCounterStore, *counterState) {
	state := &counterState{
		counts:       make(map[string]int64),
		windowStarts: make(map[string]time.Time),
	}
	mock := &ratelimiter_domain.MockCounterStore{
		IncrementAndGetFunc: func(_ context.Context, key string, delta int64, _ time.Duration) (ratelimiter_dto.CounterResult, error) {
			state.mu.Lock()
			defer state.mu.Unlock()

			if state.err != nil {
				return ratelimiter_dto.CounterResult{}, state.err
			}
			if _, ok := state.windowStarts[key]; !ok {
				state.windowStarts[key] = time.Now()
			}
			state.counts[key] += delta
			return ratelimiter_dto.CounterResult{
				Count:       state.counts[key],
				WindowStart: state.windowStarts[key],
			}, nil
		},
	}
	return mock, state
}

func newTestService(counter ratelimiter_domain.CounterStorePort) RateLimitService {
	limiter := ratelimiter_domain.NewLimiter(
		ratelimiter_adapters.NoopTokenBucketStore{},
		counter,
	)
	return NewRateLimitService(limiter)
}

func TestNewRateLimitService_Success(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)
	assert.NotNil(t, service)
}

func TestRateLimitService_CheckLimit_FirstRequest_ReturnsAllowed(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	result, err := service.CheckLimit("user:123", 10, time.Minute)

	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 9, result.Remaining)
}

func TestRateLimitService_CheckLimit_BelowLimit_ReturnsAllowed(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	for range 5 {
		result, err := service.CheckLimit("user:123", 10, time.Minute)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "request should be allowed")
	}
}

func TestRateLimitService_CheckLimit_AtLimit_ReturnsAllowed(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	for range 10 {
		result, err := service.CheckLimit("user:123", 10, time.Minute)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "request should be allowed")
	}
}

func TestRateLimitService_CheckLimit_AboveLimit_ReturnsDenied(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	for range 10 {
		result, err := service.CheckLimit("user:123", 10, time.Minute)
		require.NoError(t, err)
		require.True(t, result.Allowed)
	}

	result, err := service.CheckLimit("user:123", 10, time.Minute)

	assert.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
	assert.NotZero(t, result.RetryAfter)
}

func TestRateLimitService_CheckLimit_WellAboveLimit_ReturnsDenied(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	for range 10 {
		_, _ = service.CheckLimit("user:123", 10, time.Minute)
	}

	for i := range 5 {
		result, err := service.CheckLimit("user:123", 10, time.Minute)
		assert.NoError(t, err)
		assert.False(t, result.Allowed, "request %d over limit should be denied", i+1)
	}
}

func TestRateLimitService_CheckLimit_StorageError_FailOpen(t *testing.T) {
	storageErr := errors.New("storage failure")
	counter, state := newStatefulCounter()
	state.mu.Lock()
	state.err = storageErr
	state.mu.Unlock()

	service := newTestService(counter)

	result, err := service.CheckLimit("user:123", 10, time.Minute)

	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRateLimitService_CheckLimit_StorageError_FailClosed(t *testing.T) {
	storageErr := errors.New("storage failure")
	counter, state := newStatefulCounter()
	state.mu.Lock()
	state.err = storageErr
	state.mu.Unlock()

	limiter := ratelimiter_domain.NewLimiter(
		ratelimiter_adapters.NoopTokenBucketStore{},
		counter,
		ratelimiter_domain.WithFailPolicy(ratelimiter_dto.FailClosed),
	)
	service := NewRateLimitService(limiter)

	result, err := service.CheckLimit("user:123", 10, time.Minute)

	assert.Error(t, err)
	assert.False(t, result.Allowed)
}

func TestRateLimitService_CheckLimit_IndependentKeys(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	for range 10 {
		result, err := service.CheckLimit("user:1", 10, time.Minute)
		require.NoError(t, err)
		require.True(t, result.Allowed)
	}

	result, err := service.CheckLimit("user:2", 10, time.Minute)

	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRateLimitService_CheckLimit_DifferentLimits(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	results := make([]bool, 0, 5)
	for range 5 {
		result, err := service.CheckLimit("user:123", 3, time.Minute)
		require.NoError(t, err)
		results = append(results, result.Allowed)
	}

	assert.True(t, results[0])
	assert.True(t, results[1])
	assert.True(t, results[2])
	assert.False(t, results[3])
	assert.False(t, results[4])
}

func TestRateLimitService_CheckLimit_LimitOfOne(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	result1, err1 := service.CheckLimit("user:123", 1, time.Minute)
	assert.NoError(t, err1)
	assert.True(t, result1.Allowed)

	result2, err2 := service.CheckLimit("user:123", 1, time.Minute)
	assert.NoError(t, err2)
	assert.False(t, result2.Allowed)
}

func TestRateLimitService_CheckLimit_LimitOfZero_AlwaysDenied(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	for i := range 5 {
		result, err := service.CheckLimit("user:123", 0, time.Minute)
		assert.NoError(t, err)
		assert.False(t, result.Allowed, "request %d should be denied with limit=0", i+1)
	}
}

func TestRateLimitService_CheckLimit_EmptyKey_Allowed(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	result, err := service.CheckLimit("", 10, time.Minute)

	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRateLimitService_CheckLimit_ReturnsCorrectRemaining(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	for i := range 5 {
		result, err := service.CheckLimit("user:123", 10, time.Minute)
		require.NoError(t, err)
		assert.Equal(t, 10-(i+1), result.Remaining, "remaining should decrease with each request")
	}
}

func TestRateLimitService_CheckLimit_ReturnsRetryAfterWhenDenied(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	for range 10 {
		_, _ = service.CheckLimit("user:123", 10, time.Minute)
	}

	result, err := service.CheckLimit("user:123", 10, time.Minute)

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
}

func TestRateLimitService_CheckLimit_NoRetryAfterWhenAllowed(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	result, err := service.CheckLimit("user:123", 10, time.Minute)

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Zero(t, result.RetryAfter)
}

func TestRateLimitService_CheckLimit_RateLimitRecovery(t *testing.T) {
	counter, state := newStatefulCounter()
	service := newTestService(counter)

	for range 10 {
		result, err := service.CheckLimit("user:123", 10, time.Minute)
		require.NoError(t, err)
		require.True(t, result.Allowed)
	}

	result, err := service.CheckLimit("user:123", 10, time.Minute)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	state.mu.Lock()
	state.counts = make(map[string]int64)
	state.windowStarts = make(map[string]time.Time)
	state.err = nil
	state.mu.Unlock()

	result, err = service.CheckLimit("user:123", 10, time.Minute)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestRateLimitService_ConcurrentRequests_SameKey(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	const goroutines = 20
	const requestsPerGoroutine = 5
	const limit = 100

	results := make(chan bool, goroutines*requestsPerGoroutine)

	for range goroutines {
		go func() {
			for range requestsPerGoroutine {
				result, err := service.CheckLimit("shared:key", limit, time.Minute)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				results <- result.Allowed
			}
		}()
	}

	allowedCount := 0
	deniedCount := 0
	for range goroutines * requestsPerGoroutine {
		if <-results {
			allowedCount++
		} else {
			deniedCount++
		}
	}

	totalRequests := goroutines * requestsPerGoroutine
	assert.Equal(t, totalRequests, allowedCount+deniedCount)
	assert.LessOrEqual(t, allowedCount, limit, "allowed count should not exceed limit")
}

func TestRateLimitService_ConcurrentRequests_DifferentKeys(t *testing.T) {
	counter, _ := newStatefulCounter()
	service := newTestService(counter)

	const goroutines = 10
	const requestsPerGoroutine = 10

	done := make(chan bool, goroutines)

	for i := range goroutines {
		key := "user:" + string(rune('A'+i))
		go func(userKey string) {
			for range requestsPerGoroutine {
				result, err := service.CheckLimit(userKey, 100, time.Minute)
				if err != nil || !result.Allowed {
					t.Errorf("user %s request: allowed=%v, err=%v", userKey, result.Allowed, err)
					return
				}
			}
			done <- true
		}(key)
	}

	for range goroutines {
		<-done
	}
}
