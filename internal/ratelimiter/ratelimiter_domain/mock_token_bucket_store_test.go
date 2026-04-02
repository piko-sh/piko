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

func TestMockTokenBucketStore_TryTake(t *testing.T) {
	t.Parallel()

	t.Run("nil TryTakeFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockTokenBucketStore{
			TryTakeFunc:           nil,
			WaitDurationFunc:      nil,
			DeleteBucketFunc:      nil,
			TryTakeCallCount:      0,
			WaitDurationCallCount: 0,
			DeleteBucketCallCount: 0,
		}

		ctx := context.Background()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 20}

		allowed, err := mock.TryTake(ctx, "key", 1.0, config)

		require.NoError(t, err)
		assert.False(t, allowed)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TryTakeCallCount))
	})

	t.Run("delegates to TryTakeFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockTokenBucketStore{
			TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
				return true, nil
			},
			WaitDurationFunc:      nil,
			DeleteBucketFunc:      nil,
			TryTakeCallCount:      0,
			WaitDurationCallCount: 0,
			DeleteBucketCallCount: 0,
		}

		ctx := context.Background()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 5.0, Burst: 10}

		allowed, err := mock.TryTake(ctx, "user:abc", 2.0, config)

		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TryTakeCallCount))
	})

	t.Run("propagates error from TryTakeFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockTokenBucketStore{
			TryTakeFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
				return false, errors.New("bucket corrupted")
			},
			WaitDurationFunc:      nil,
			DeleteBucketFunc:      nil,
			TryTakeCallCount:      0,
			WaitDurationCallCount: 0,
			DeleteBucketCallCount: 0,
		}

		ctx := context.Background()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

		allowed, err := mock.TryTake(ctx, "key", 1.0, config)

		require.Error(t, err)
		assert.Equal(t, "bucket corrupted", err.Error())
		assert.False(t, allowed)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TryTakeCallCount))
	})
}

func TestMockTokenBucketStore_TryTake_PassesArguments(t *testing.T) {
	t.Parallel()

	var (
		capturedCtx    context.Context
		capturedKey    string
		capturedN      float64
		capturedConfig *ratelimiter_dto.TokenBucketConfig
	)

	mock := &MockTokenBucketStore{
		TryTakeFunc: func(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			capturedCtx = ctx
			capturedKey = key
			capturedN = n
			capturedConfig = config
			return true, nil
		},
		WaitDurationFunc:      nil,
		DeleteBucketFunc:      nil,
		TryTakeCallCount:      0,
		WaitDurationCallCount: 0,
		DeleteBucketCallCount: 0,
	}

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "trytake-ctx")
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 100.0, Burst: 200}

	_, err := mock.TryTake(ctx, "rate:endpoint:/api/v1", 3.5, config)

	require.NoError(t, err)
	assert.Equal(t, ctx, capturedCtx)
	assert.Equal(t, "rate:endpoint:/api/v1", capturedKey)
	assert.InDelta(t, 3.5, capturedN, 0.001)
	assert.Equal(t, config, capturedConfig)
}

func TestMockTokenBucketStore_WaitDuration(t *testing.T) {
	t.Parallel()

	t.Run("nil WaitDurationFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockTokenBucketStore{
			TryTakeFunc:           nil,
			WaitDurationFunc:      nil,
			DeleteBucketFunc:      nil,
			TryTakeCallCount:      0,
			WaitDurationCallCount: 0,
			DeleteBucketCallCount: 0,
		}

		ctx := context.Background()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 20}

		dur, err := mock.WaitDuration(ctx, "key", 1.0, config)

		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), dur)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.WaitDurationCallCount))
	})

	t.Run("delegates to WaitDurationFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockTokenBucketStore{
			TryTakeFunc: nil,
			WaitDurationFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
				return 500 * time.Millisecond, nil
			},
			DeleteBucketFunc:      nil,
			TryTakeCallCount:      0,
			WaitDurationCallCount: 0,
			DeleteBucketCallCount: 0,
		}

		ctx := context.Background()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 2.0, Burst: 5}

		dur, err := mock.WaitDuration(ctx, "user:xyz", 3.0, config)

		require.NoError(t, err)
		assert.Equal(t, 500*time.Millisecond, dur)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.WaitDurationCallCount))
	})

	t.Run("propagates error from WaitDurationFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockTokenBucketStore{
			TryTakeFunc: nil,
			WaitDurationFunc: func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
				return 0, errors.New("calculation failed")
			},
			DeleteBucketFunc:      nil,
			TryTakeCallCount:      0,
			WaitDurationCallCount: 0,
			DeleteBucketCallCount: 0,
		}

		ctx := context.Background()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

		dur, err := mock.WaitDuration(ctx, "key", 1.0, config)

		require.Error(t, err)
		assert.Equal(t, "calculation failed", err.Error())
		assert.Equal(t, time.Duration(0), dur)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.WaitDurationCallCount))
	})
}

func TestMockTokenBucketStore_WaitDuration_PassesArguments(t *testing.T) {
	t.Parallel()

	var (
		capturedCtx    context.Context
		capturedKey    string
		capturedN      float64
		capturedConfig *ratelimiter_dto.TokenBucketConfig
	)

	mock := &MockTokenBucketStore{
		TryTakeFunc: nil,
		WaitDurationFunc: func(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
			capturedCtx = ctx
			capturedKey = key
			capturedN = n
			capturedConfig = config
			return 0, nil
		},
		DeleteBucketFunc:      nil,
		TryTakeCallCount:      0,
		WaitDurationCallCount: 0,
		DeleteBucketCallCount: 0,
	}

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "wait-ctx")
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 50.0, Burst: 100}

	_, err := mock.WaitDuration(ctx, "rate:api-gateway", 7.5, config)

	require.NoError(t, err)
	assert.Equal(t, ctx, capturedCtx)
	assert.Equal(t, "rate:api-gateway", capturedKey)
	assert.InDelta(t, 7.5, capturedN, 0.001)
	assert.Equal(t, config, capturedConfig)
}

func TestMockTokenBucketStore_DeleteBucket(t *testing.T) {
	t.Parallel()

	t.Run("nil DeleteBucketFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockTokenBucketStore{
			TryTakeFunc:           nil,
			WaitDurationFunc:      nil,
			DeleteBucketFunc:      nil,
			TryTakeCallCount:      0,
			WaitDurationCallCount: 0,
			DeleteBucketCallCount: 0,
		}

		ctx := context.Background()
		err := mock.DeleteBucket(ctx, "key-to-delete")

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DeleteBucketCallCount))
	})

	t.Run("delegates to DeleteBucketFunc", func(t *testing.T) {
		t.Parallel()

		var called bool

		mock := &MockTokenBucketStore{
			TryTakeFunc:      nil,
			WaitDurationFunc: nil,
			DeleteBucketFunc: func(_ context.Context, _ string) error {
				called = true
				return nil
			},
			TryTakeCallCount:      0,
			WaitDurationCallCount: 0,
			DeleteBucketCallCount: 0,
		}

		ctx := context.Background()
		err := mock.DeleteBucket(ctx, "bucket-abc")

		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DeleteBucketCallCount))
	})

	t.Run("propagates error from DeleteBucketFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockTokenBucketStore{
			TryTakeFunc:      nil,
			WaitDurationFunc: nil,
			DeleteBucketFunc: func(_ context.Context, _ string) error {
				return errors.New("delete not permitted")
			},
			TryTakeCallCount:      0,
			WaitDurationCallCount: 0,
			DeleteBucketCallCount: 0,
		}

		ctx := context.Background()
		err := mock.DeleteBucket(ctx, "protected-bucket")

		require.Error(t, err)
		assert.Equal(t, "delete not permitted", err.Error())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DeleteBucketCallCount))
	})
}

func TestMockTokenBucketStore_DeleteBucket_PassesArguments(t *testing.T) {
	t.Parallel()

	var (
		capturedCtx context.Context
		capturedKey string
	)

	mock := &MockTokenBucketStore{
		TryTakeFunc:      nil,
		WaitDurationFunc: nil,
		DeleteBucketFunc: func(ctx context.Context, key string) error {
			capturedCtx = ctx
			capturedKey = key
			return nil
		},
		TryTakeCallCount:      0,
		WaitDurationCallCount: 0,
		DeleteBucketCallCount: 0,
	}

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "delete-ctx")

	err := mock.DeleteBucket(ctx, "rate:expired-session")

	require.NoError(t, err)
	assert.Equal(t, ctx, capturedCtx)
	assert.Equal(t, "rate:expired-session", capturedKey)
}

func TestMockTokenBucketStore_CallCountsAreIndependent(t *testing.T) {
	t.Parallel()

	mock := &MockTokenBucketStore{
		TryTakeFunc:           nil,
		WaitDurationFunc:      nil,
		DeleteBucketFunc:      nil,
		TryTakeCallCount:      0,
		WaitDurationCallCount: 0,
		DeleteBucketCallCount: 0,
	}

	ctx := context.Background()
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	_, _ = mock.TryTake(ctx, "k", 1.0, config)
	_, _ = mock.TryTake(ctx, "k", 1.0, config)
	_, _ = mock.TryTake(ctx, "k", 1.0, config)
	_, _ = mock.WaitDuration(ctx, "k", 1.0, config)
	_, _ = mock.WaitDuration(ctx, "k", 1.0, config)
	_ = mock.DeleteBucket(ctx, "k")

	assert.Equal(t, int64(3), atomic.LoadInt64(&mock.TryTakeCallCount))
	assert.Equal(t, int64(2), atomic.LoadInt64(&mock.WaitDurationCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DeleteBucketCallCount))
}

func TestMockTokenBucketStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockTokenBucketStore

	ctx := context.Background()
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 20}

	allowed, err := mock.TryTake(ctx, "zero-key", 1.0, config)
	require.NoError(t, err)
	assert.False(t, allowed)

	dur, err := mock.WaitDuration(ctx, "zero-key", 1.0, config)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), dur)

	err = mock.DeleteBucket(ctx, "zero-key")
	require.NoError(t, err)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TryTakeCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.WaitDurationCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DeleteBucketCallCount))
}

func TestMockTokenBucketStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockTokenBucketStore{
		TryTakeFunc:           nil,
		WaitDurationFunc:      nil,
		DeleteBucketFunc:      nil,
		TryTakeCallCount:      0,
		WaitDurationCallCount: 0,
		DeleteBucketCallCount: 0,
	}

	ctx := context.Background()
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 20}
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = mock.TryTake(ctx, "concurrent", 1.0, config)
		}()
		go func() {
			defer wg.Done()
			_, _ = mock.WaitDuration(ctx, "concurrent", 1.0, config)
		}()
		go func() {
			defer wg.Done()
			_ = mock.DeleteBucket(ctx, "concurrent")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.TryTakeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.WaitDurationCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.DeleteBucketCallCount))
}

func TestMockTokenBucketStore_ImplementsTokenBucketStorePort(t *testing.T) {
	t.Parallel()

	mock := &MockTokenBucketStore{
		TryTakeFunc:           nil,
		WaitDurationFunc:      nil,
		DeleteBucketFunc:      nil,
		TryTakeCallCount:      0,
		WaitDurationCallCount: 0,
		DeleteBucketCallCount: 0,
	}

	var _ TokenBucketStorePort = mock
}
