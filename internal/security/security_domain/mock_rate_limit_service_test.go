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
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

func TestMockRateLimitService_CheckLimit(t *testing.T) {
	t.Parallel()

	t.Run("nil CheckLimitFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockRateLimitService{
			CheckLimitFunc:      nil,
			CheckLimitCallCount: 0,
		}

		result, err := mock.CheckLimit("key-1", 100, time.Minute)

		require.NoError(t, err)
		assert.Equal(t, ratelimiter_dto.Result{}, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CheckLimitCallCount))
	})

	t.Run("delegates to CheckLimitFunc", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		expected := ratelimiter_dto.Result{
			ResetAt:    now.Add(time.Minute),
			Limit:      50,
			Remaining:  49,
			RetryAfter: 0,
			Allowed:    true,
		}

		var capturedKey string
		var capturedLimit int
		var capturedWindow time.Duration

		mock := &MockRateLimitService{
			CheckLimitFunc: func(key string, limit int, window time.Duration) (ratelimiter_dto.Result, error) {
				capturedKey = key
				capturedLimit = limit
				capturedWindow = window
				return expected, nil
			},
			CheckLimitCallCount: 0,
		}

		result, err := mock.CheckLimit("api:user:42", 50, 5*time.Minute)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
		assert.Equal(t, "api:user:42", capturedKey)
		assert.Equal(t, 50, capturedLimit)
		assert.Equal(t, 5*time.Minute, capturedWindow)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CheckLimitCallCount))
	})

	t.Run("propagates error from CheckLimitFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockRateLimitService{
			CheckLimitFunc: func(_ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
				return ratelimiter_dto.Result{}, errors.New("rate limiter unavailable")
			},
			CheckLimitCallCount: 0,
		}

		result, err := mock.CheckLimit("key", 10, time.Second)

		require.Error(t, err)
		assert.Equal(t, "rate limiter unavailable", err.Error())
		assert.Equal(t, ratelimiter_dto.Result{}, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CheckLimitCallCount))
	})
}

func TestMockRateLimitService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockRateLimitService

	result, err := mock.CheckLimit("zero-key", 100, time.Minute)

	require.NoError(t, err)
	assert.Equal(t, ratelimiter_dto.Result{}, result)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CheckLimitCallCount))
}

func TestMockRateLimitService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockRateLimitService{
		CheckLimitFunc:      nil,
		CheckLimitCallCount: 0,
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = mock.CheckLimit("concurrent-key", 10, time.Second)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.CheckLimitCallCount))
}
