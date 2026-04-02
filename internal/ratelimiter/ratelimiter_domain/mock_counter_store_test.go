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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

func TestMockCounterStore_IncrementAndGet(t *testing.T) {
	t.Parallel()

	t.Run("nil IncrementAndGetFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockCounterStore{
			IncrementAndGetFunc:      nil,
			IncrementAndGetCallCount: 0,
		}

		ctx := context.Background()
		result, err := mock.IncrementAndGet(ctx, "key", 1, time.Minute)

		require.NoError(t, err)
		assert.Equal(t, ratelimiter_dto.CounterResult{}, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.IncrementAndGetCallCount))
	})

	t.Run("delegates to IncrementAndGetFunc", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		expected := ratelimiter_dto.CounterResult{
			WindowStart: now,
			Count:       42,
		}

		mock := &MockCounterStore{
			IncrementAndGetFunc: func(_ context.Context, _ string, _ int64, _ time.Duration) (ratelimiter_dto.CounterResult, error) {
				return expected, nil
			},
			IncrementAndGetCallCount: 0,
		}

		ctx := context.Background()
		result, err := mock.IncrementAndGet(ctx, "api-key", 5, 10*time.Second)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.IncrementAndGetCallCount))
	})

	t.Run("propagates error from IncrementAndGetFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockCounterStore{
			IncrementAndGetFunc: func(_ context.Context, _ string, _ int64, _ time.Duration) (ratelimiter_dto.CounterResult, error) {
				return ratelimiter_dto.CounterResult{}, errors.New("storage unavailable")
			},
			IncrementAndGetCallCount: 0,
		}

		ctx := context.Background()
		result, err := mock.IncrementAndGet(ctx, "key", 1, time.Minute)

		require.Error(t, err)
		assert.Equal(t, "storage unavailable", err.Error())
		assert.Equal(t, ratelimiter_dto.CounterResult{}, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.IncrementAndGetCallCount))
	})
}

func TestMockCounterStore_IncrementAndGet_PassesArguments(t *testing.T) {
	t.Parallel()

	var (
		capturedCtx    context.Context
		capturedKey    string
		capturedDelta  int64
		capturedWindow time.Duration
	)

	mock := &MockCounterStore{
		IncrementAndGetFunc: func(ctx context.Context, key string, delta int64, window time.Duration) (ratelimiter_dto.CounterResult, error) {
			capturedCtx = ctx
			capturedKey = key
			capturedDelta = delta
			capturedWindow = window
			return ratelimiter_dto.CounterResult{}, nil
		},
		IncrementAndGetCallCount: 0,
	}

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "test-value")

	_, err := mock.IncrementAndGet(ctx, "rate:user:123", 7, 30*time.Second)

	require.NoError(t, err)
	assert.Equal(t, ctx, capturedCtx)
	assert.Equal(t, "rate:user:123", capturedKey)
	assert.Equal(t, int64(7), capturedDelta)
	assert.Equal(t, 30*time.Second, capturedWindow)
}

func TestMockCounterStore_IncrementAndGet_MultipleCalls(t *testing.T) {
	t.Parallel()

	mock := &MockCounterStore{
		IncrementAndGetFunc:      nil,
		IncrementAndGetCallCount: 0,
	}

	ctx := context.Background()

	for range 5 {
		_, err := mock.IncrementAndGet(ctx, "key", 1, time.Minute)
		require.NoError(t, err)
	}

	assert.Equal(t, int64(5), atomic.LoadInt64(&mock.IncrementAndGetCallCount))
}

func TestMockCounterStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockCounterStore

	ctx := context.Background()
	result, err := mock.IncrementAndGet(ctx, "zero-key", 1, time.Minute)

	require.NoError(t, err)
	assert.Equal(t, ratelimiter_dto.CounterResult{}, result)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.IncrementAndGetCallCount))
}

func TestMockCounterStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockCounterStore{
		IncrementAndGetFunc:      nil,
		IncrementAndGetCallCount: 0,
	}

	ctx := context.Background()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = mock.IncrementAndGet(ctx, "concurrent-key", 1, time.Minute)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.IncrementAndGetCallCount))
}

func TestMockCounterStore_ImplementsCounterStorePort(t *testing.T) {
	t.Parallel()

	mock := &MockCounterStore{
		IncrementAndGetFunc:      nil,
		IncrementAndGetCallCount: 0,
	}

	var _ CounterStorePort = mock
}
