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

package llm_domain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
)

func TestNewRateLimiter(t *testing.T) {
	store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
	limiter := NewRateLimiter(store)

	assert.NotNil(t, limiter.store)
	assert.NotNil(t, limiter.configs)
}

func TestRateLimiter_SetAndGetLimits(t *testing.T) {
	store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
	limiter := NewRateLimiter(store)

	limiter.SetLimits("test-scope", 100, 1000)

	rpm, tpm := limiter.GetLimits("test-scope")
	assert.Equal(t, 100, rpm)
	assert.Equal(t, 1000, tpm)

	assert.True(t, limiter.HasLimits("test-scope"))
	assert.False(t, limiter.HasLimits("other-scope"))
}

func TestRateLimiter_AllowN(t *testing.T) {
	store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
	limiter := NewRateLimiter(store)
	ctx := context.Background()

	limiter.SetLimits("test-scope", 10, 100)

	err := limiter.AllowN(ctx, "test-scope", 1, 10)
	require.NoError(t, err)

	for range 9 {
		err = limiter.AllowN(ctx, "test-scope", 1, 10)
		require.NoError(t, err)
	}

	err = limiter.AllowN(ctx, "test-scope", 1, 10)
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestRateLimiter_RemoveLimits(t *testing.T) {
	store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
	limiter := NewRateLimiter(store)

	limiter.SetLimits("test-scope", 100, 1000)
	assert.True(t, limiter.HasLimits("test-scope"))

	limiter.RemoveLimits("test-scope")
	assert.False(t, limiter.HasLimits("test-scope"))
}

func TestRateLimiter_AllowWithNoLimits(t *testing.T) {
	store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
	limiter := NewRateLimiter(store)
	ctx := context.Background()

	err := limiter.AllowN(ctx, "test-scope", 1, 10)
	require.NoError(t, err)
}

func TestRateLimiter_Allow(t *testing.T) {
	ctx := context.Background()

	t.Run("allows when no limits configured", func(t *testing.T) {
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)

		err := limiter.Allow(ctx, "openai")
		assert.NoError(t, err)
	})

	t.Run("allows request within limit", func(t *testing.T) {
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		limiter.SetLimits("openai", 10, 0)

		err := limiter.Allow(ctx, "openai")
		assert.NoError(t, err)
	})

	t.Run("returns context error if cancelled", func(t *testing.T) {
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		limiter.SetLimits("openai", 10, 0)

		ctx, cancel := context.WithCancelCause(ctx)
		cancel(fmt.Errorf("test: simulating cancelled context"))

		err := limiter.Allow(ctx, "openai")
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestRateLimiter_AllowN_BucketTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("checks only request limit when no token limit", func(t *testing.T) {
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		limiter.SetLimits("openai", 60, 0)

		err := limiter.AllowN(ctx, "openai", 1, 1000)
		assert.NoError(t, err)

		for range 59 {
			_ = limiter.AllowN(ctx, "openai", 1, 0)
		}
		err = limiter.AllowN(ctx, "openai", 1, 0)
		assert.ErrorIs(t, err, ErrRateLimited, "should be rate limited by request bucket")
	})

	t.Run("checks only token limit when no request limit", func(t *testing.T) {
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		limiter.SetLimits("openai", 0, 100000)

		err := limiter.AllowN(ctx, "openai", 1, 500)
		assert.NoError(t, err)

		for range 1000 {
			err = limiter.AllowN(ctx, "openai", 0, 0)
			require.NoError(t, err, "should allow unlimited requests when RPM is 0")
		}
	})

	t.Run("checks both limits when configured", func(t *testing.T) {
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		limiter.SetLimits("openai", 60, 100000)

		err := limiter.AllowN(ctx, "openai", 1, 500)
		assert.NoError(t, err)
	})
}

func TestRateLimiter_Wait(t *testing.T) {
	t.Run("returns immediately when no limits", func(t *testing.T) {
		ctx := context.Background()
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)

		start := time.Now()
		err := limiter.Wait(ctx, "openai")
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, elapsed, 100*time.Millisecond)
	})

	t.Run("returns immediately when tokens available", func(t *testing.T) {
		ctx := context.Background()
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		limiter.SetLimits("openai", 60, 100000)

		start := time.Now()
		err := limiter.Wait(ctx, "openai")
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, elapsed, 100*time.Millisecond)
	})

	t.Run("returns context error when cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		limiter.SetLimits("openai", 60, 100000)

		err := limiter.Wait(ctx, "openai")
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestRateLimiter_WaitN(t *testing.T) {
	t.Run("returns immediately when tokens available", func(t *testing.T) {
		ctx := context.Background()
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		limiter.SetLimits("openai", 60, 100000)

		start := time.Now()
		err := limiter.WaitN(ctx, "openai", 5, 1000)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, elapsed, 100*time.Millisecond)
	})

	t.Run("returns error when context deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeoutCause(context.Background(), 50*time.Millisecond, fmt.Errorf("test: rate limiter wait deadline"))
		defer cancel()

		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		limiter.SetLimits("openai", 1, 0)

		for range 5 {
			_ = limiter.AllowN(context.Background(), "openai", 1, 0)
		}

		err := limiter.WaitN(ctx, "openai", 100, 0)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestRateLimiter_GetLimits_Unconfigured(t *testing.T) {
	store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
	limiter := NewRateLimiter(store)

	rpm, tpm := limiter.GetLimits("unconfigured-scope")
	assert.Equal(t, 0, rpm)
	assert.Equal(t, 0, tpm)
}

func TestBucketType(t *testing.T) {
	assert.Equal(t, BucketType("request"), BucketTypeRequest)
	assert.Equal(t, BucketType("token"), BucketTypeToken)
}
