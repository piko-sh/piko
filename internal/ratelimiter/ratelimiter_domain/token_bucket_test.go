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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := RefillBucket(tc.state, tc.nowNano)
			if tc.expectNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.InDelta(t, tc.expectedTokens, result.Tokens, 0.001)
			assert.Equal(t, tc.state.MaxTokens, result.MaxTokens)
			assert.Equal(t, tc.state.RefillRate, result.RefillRate)
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := NewBucketState(tc.config, tc.nowNano)

			require.NotNil(t, state)
			assert.InDelta(t, tc.expectedTokens, state.Tokens, 0.001)
			assert.InDelta(t, tc.expectedMaxTokens, state.MaxTokens, 0.001)
			assert.InDelta(t, tc.expectedRefillRate, state.RefillRate, 1e-15)
			assert.Equal(t, tc.nowNano, state.LastRefillNano)
		})
	}
}
