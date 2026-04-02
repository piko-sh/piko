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

package ratelimiter_adapters

import (
	"context"
	"sync"
	"time"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

var _ ratelimiter_domain.TokenBucketStorePort = (*InMemoryTokenBucketStore)(nil)

// InMemoryTokenBucketStoreOption configures an InMemoryTokenBucketStore.
type InMemoryTokenBucketStoreOption func(*InMemoryTokenBucketStore)

// InMemoryTokenBucketStore is an in-process implementation of
// TokenBucketStorePort that stores bucket state in a mutex-protected map.
// It uses the pure RefillBucket algorithm from ratelimiter_domain for token
// replenishment.
//
// This store is suitable for single-instance rate limiting where distributed
// state is not required, such as per-provider email throttling.
//
// All methods are safe for concurrent use.
type InMemoryTokenBucketStore struct {
	// clock provides time operations for token bucket calculations.
	clock clock.Clock

	// buckets maps keys to their token bucket states.
	buckets map[string]*ratelimiter_domain.TokenBucketState

	// mu guards access to the buckets map.
	mu sync.Mutex
}

// NewInMemoryTokenBucketStore creates an in-memory token bucket store.
//
// Takes opts (...InMemoryTokenBucketStoreOption) which are optional
// configuration functions.
//
// Returns *InMemoryTokenBucketStore ready for use.
func NewInMemoryTokenBucketStore(opts ...InMemoryTokenBucketStoreOption) *InMemoryTokenBucketStore {
	s := &InMemoryTokenBucketStore{
		clock:   clock.RealClock(),
		buckets: make(map[string]*ratelimiter_domain.TokenBucketState),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// TryTake atomically refills the bucket and attempts to take n tokens.
//
// Takes key (string) which identifies the rate limit bucket.
// Takes n (float64) which is the number of tokens to take.
// Takes config (*ratelimiter_dto.TokenBucketConfig) which defines bucket
// parameters.
//
// Returns bool which is true if tokens were successfully taken.
// Returns error which is always nil for the in-memory store.
//
// Safe for concurrent use. Access is protected by a mutex.
func (s *InMemoryTokenBucketStore) TryTake(_ context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now().UnixNano()
	state := s.getOrCreate(key, config, now)
	state = ratelimiter_domain.RefillBucket(state, now)

	if state.Tokens < n {
		s.buckets[key] = state
		return false, nil
	}

	s.buckets[key] = &ratelimiter_domain.TokenBucketState{
		Tokens:         state.Tokens - n,
		MaxTokens:      state.MaxTokens,
		RefillRate:     state.RefillRate,
		LastRefillNano: state.LastRefillNano,
	}
	return true, nil
}

// WaitDuration returns the estimated time until n tokens become available.
//
// Takes key (string) which identifies the rate limit bucket.
// Takes n (float64) which is the number of tokens needed.
// Takes config (*ratelimiter_dto.TokenBucketConfig) which defines bucket
// parameters.
//
// Returns time.Duration which is the estimated wait time.
// Returns error which is always nil for the in-memory store.
//
// Safe for concurrent use.
func (s *InMemoryTokenBucketStore) WaitDuration(_ context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now().UnixNano()
	state := s.getOrCreate(key, config, now)
	state = ratelimiter_domain.RefillBucket(state, now)

	if state.Tokens >= n {
		return 0, nil
	}

	deficit := n - state.Tokens
	if state.RefillRate <= 0 {
		return time.Second, nil
	}
	waitNanos := deficit / state.RefillRate
	return time.Duration(waitNanos), nil
}

// DeleteBucket removes a bucket's state from the store.
//
// Takes key (string) which identifies the rate limit bucket to remove.
//
// Returns error which is always nil for the in-memory store.
//
// Safe for concurrent use.
func (s *InMemoryTokenBucketStore) DeleteBucket(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buckets, key)
	return nil
}

// getOrCreate returns the existing bucket state or creates a new one.
//
// Takes key (string) which identifies the bucket to retrieve or create.
// Takes config (*ratelimiter_dto.TokenBucketConfig) which specifies the bucket
// settings for new buckets.
// Takes nowNano (int64) which provides the current time in nanoseconds.
//
// Returns *ratelimiter_domain.TokenBucketState which is the existing or newly
// created bucket state.
func (s *InMemoryTokenBucketStore) getOrCreate(key string, config *ratelimiter_dto.TokenBucketConfig, nowNano int64) *ratelimiter_domain.TokenBucketState {
	state := s.buckets[key]
	if state == nil {
		state = ratelimiter_domain.NewBucketState(config, nowNano)
		s.buckets[key] = state
	}
	return state
}

// WithInMemoryClock sets the clock used for time operations, defaulting to
// clock.RealClock(). This is primarily useful for testing.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns InMemoryTokenBucketStoreOption to apply to the store.
func WithInMemoryClock(c clock.Clock) InMemoryTokenBucketStoreOption {
	return func(s *InMemoryTokenBucketStore) {
		s.clock = c
	}
}
