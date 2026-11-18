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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

func TestRefillBucket(t *testing.T) {
	testCases := []struct {
		state          *TokenBucketState
		name           string
		nowNano        int64
		expectedTokens float64
		expectNil      bool
	}{
		{
			name:      "nil state returns nil",
			state:     nil,
			nowNano:   1000,
			expectNil: true,
		},
		{
			name: "no elapsed time returns same state",
			state: &TokenBucketState{
				Tokens:         5.0,
				MaxTokens:      10.0,
				RefillRate:     0.001,
				LastRefillNano: 1000,
			},
			nowNano:        1000,
			expectedTokens: 5.0,
		},
		{
			name: "negative elapsed time returns same state",
			state: &TokenBucketState{
				Tokens:         5.0,
				MaxTokens:      10.0,
				RefillRate:     0.001,
				LastRefillNano: 2000,
			},
			nowNano:        1000,
			expectedTokens: 5.0,
		},
		{
			name: "partial refill adds tokens",
			state: &TokenBucketState{
				Tokens:         5.0,
				MaxTokens:      10.0,
				RefillRate:     0.001,
				LastRefillNano: 0,
			},
			nowNano:        3000,
			expectedTokens: 8.0,
		},
		{
			name: "refill capped at max tokens",
			state: &TokenBucketState{
				Tokens:         8.0,
				MaxTokens:      10.0,
				RefillRate:     0.001,
				LastRefillNano: 0,
			},
			nowNano:        10000,
			expectedTokens: 10.0,
		},
		{
			name: "empty bucket refills correctly",
			state: &TokenBucketState{
				Tokens:         0.0,
				MaxTokens:      100.0,
				RefillRate:     0.01,
				LastRefillNano: 0,
			},
			nowNano:        5000,
			expectedTokens: 50.0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := RefillBucket(testCase.state, testCase.nowNano)
			if testCase.expectNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.InDelta(t, testCase.expectedTokens, result.Tokens, 0.001)
			assert.Equal(t, testCase.state.MaxTokens, result.MaxTokens)
			assert.Equal(t, testCase.state.RefillRate, result.RefillRate)
		})
	}
}

func TestNewBucketState(t *testing.T) {
	testCases := []struct {
		config             *ratelimiter_dto.TokenBucketConfig
		name               string
		nowNano            int64
		expectedTokens     float64
		expectedMaxTokens  float64
		expectedRefillRate float64
	}{
		{
			name: "basic configuration with burst",
			config: &ratelimiter_dto.TokenBucketConfig{
				Rate:  10.0,
				Burst: 20,
			},
			nowNano:            5000,
			expectedTokens:     20.0,
			expectedMaxTokens:  20.0,
			expectedRefillRate: 10.0 / float64(time.Second),
		},
		{
			name: "burst defaults to rate when zero",
			config: &ratelimiter_dto.TokenBucketConfig{
				Rate:  50.0,
				Burst: 0,
			},
			nowNano:            1000,
			expectedTokens:     50.0,
			expectedMaxTokens:  50.0,
			expectedRefillRate: 50.0 / float64(time.Second),
		},
		{
			name: "fractional rate",
			config: &ratelimiter_dto.TokenBucketConfig{
				Rate:  0.5,
				Burst: 1,
			},
			nowNano:            0,
			expectedTokens:     1.0,
			expectedMaxTokens:  1.0,
			expectedRefillRate: 0.5 / float64(time.Second),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			state := NewBucketState(testCase.config, testCase.nowNano)

			require.NotNil(t, state)
			assert.InDelta(t, testCase.expectedTokens, state.Tokens, 0.001)
			assert.InDelta(t, testCase.expectedMaxTokens, state.MaxTokens, 0.001)
			assert.InDelta(t, testCase.expectedRefillRate, state.RefillRate, 1e-15)
			assert.Equal(t, testCase.nowNano, state.LastRefillNano)
		})
	}
}

func TestAllowTokenBucket_Refills(t *testing.T) {
	t.Parallel()

	const bucketKey = "email:provider"

	config := ratelimiter_dto.TokenBucketConfig{Rate: 2, Burst: 4}

	testCases := []struct {
		name        string
		advance     time.Duration
		wantAllowed int
	}{
		{
			name:        "empty bucket denies at once",
			advance:     0,
			wantAllowed: 0,
		},
		{
			name:        "half a second refills one token",
			advance:     500 * time.Millisecond,
			wantAllowed: 1,
		},
		{
			name:        "refill is capped at the burst",
			advance:     time.Hour,
			wantAllowed: config.Burst,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
			limiter := NewLimiter(newRefillingTokenBucketStore(mockClock), nil, WithClock(mockClock))

			for range config.Burst {
				require.NoError(t, limiter.AllowTokenBucket(ctx, bucketKey, 1, config))
			}
			mockClock.Advance(testCase.advance)

			assert.Equal(t, testCase.wantAllowed, countAllowedTokens(t, limiter, bucketKey, config))
		})
	}
}

func countAllowedTokens(t *testing.T, limiter *Limiter, key string, config ratelimiter_dto.TokenBucketConfig) int {
	t.Helper()

	allowed := 0
	for range config.Burst + 1 {
		err := limiter.AllowTokenBucket(t.Context(), key, 1, config)
		if errors.Is(err, ErrRateLimited) {
			return allowed
		}
		require.NoError(t, err)
		allowed++
	}
	return allowed
}

func newRefillingTokenBucketStore(mockClock *clock.MockClock) *MockTokenBucketStore {
	var mu sync.Mutex
	buckets := make(map[string]*TokenBucketState)

	return &MockTokenBucketStore{
		TryTakeFunc: func(_ context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			mu.Lock()
			defer mu.Unlock()

			now := mockClock.Now().UnixNano()
			state, found := buckets[key]
			if !found {
				state = NewBucketState(config, now)
			}
			state = RefillBucket(state, now)
			allowed := state.Tokens >= n
			if allowed {
				state.Tokens -= n
			}
			buckets[key] = state
			return allowed, nil
		},
	}
}
