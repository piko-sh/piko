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
	"math"
	"time"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

const (
	// maxTokenWait is the longest wait TokenWaitDuration returns. A bucket that refills more
	// slowly is checked again after this long, so a wait never overflows and a waiting
	// caller still sees changes to the bucket.
	maxTokenWait = time.Hour
)

// RefillBucket updates the bucket state based on elapsed time since the last refill.
// Tokens are added at the configured refill rate, capped at MaxTokens.
//
// Takes state (*TokenBucketState) which holds the current bucket state.
// Takes nowNano (int64) which is the current time in nanoseconds.
//
// Returns *TokenBucketState which is the updated bucket state, or nil if state is nil.
func RefillBucket(state *TokenBucketState, nowNano int64) *TokenBucketState {
	if state == nil {
		return nil
	}

	elapsed := nowNano - state.LastRefillNano
	if elapsed <= 0 {
		return state
	}

	newTokens := state.Tokens + float64(elapsed)*state.RefillRate
	if newTokens > state.MaxTokens {
		newTokens = state.MaxTokens
	}

	return &TokenBucketState{
		Tokens:         newTokens,
		MaxTokens:      state.MaxTokens,
		RefillRate:     state.RefillRate,
		LastRefillNano: nowNano,
	}
}

// NewBucketState creates initial bucket state from configuration. The bucket starts full
// (tokens equal to max capacity).
//
// Takes config (*ratelimiter_dto.TokenBucketConfig) which defines bucket parameters.
// Takes nowNano (int64) which is the current time in nanoseconds.
//
// Returns *TokenBucketState which is the initial bucket state.
func NewBucketState(config *ratelimiter_dto.TokenBucketConfig, nowNano int64) *TokenBucketState {
	maxTokens := config.MaxTokens()
	return &TokenBucketState{
		Tokens:         maxTokens,
		MaxTokens:      maxTokens,
		RefillRate:     config.Rate / float64(time.Second),
		LastRefillNano: nowNano,
	}
}

// TokenWaitDuration returns how long a bucket takes to refill the tokens still needed.
//
// The wait is rounded up to a whole nanosecond, so a positive need never gives a zero
// wait, and is capped at maxTokenWait, so a very slow refill rate cannot overflow
// time.Duration.
//
// Takes needed (float64) which is the number of tokens still missing.
// Takes refillRate (float64) which is the refill rate in tokens per nanosecond.
//
// Returns time.Duration which is zero when nothing is needed, and at most maxTokenWait.
func TokenWaitDuration(needed, refillRate float64) time.Duration {
	if needed <= 0 {
		return 0
	}
	if math.IsNaN(needed) || math.IsNaN(refillRate) || refillRate <= 0 {
		return maxTokenWait
	}
	waitNanos := math.Ceil(needed / refillRate)
	if waitNanos >= float64(maxTokenWait) {
		return maxTokenWait
	}
	return time.Duration(waitNanos)
}
