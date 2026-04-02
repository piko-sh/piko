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
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

// MockTokenBucketStore is a test double for TokenBucketStorePort where
// nil function fields return zero values and call counts are tracked
// atomically.
type MockTokenBucketStore struct {
	// TryTakeFunc is the function called by TryTake.
	TryTakeFunc func(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error)

	// WaitDurationFunc is the function called by
	// WaitDuration.
	WaitDurationFunc func(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (time.Duration, error)

	// DeleteBucketFunc is the function called by
	// DeleteBucket.
	DeleteBucketFunc func(ctx context.Context, key string) error

	// TryTakeCallCount tracks how many times TryTake
	// was called.
	TryTakeCallCount int64

	// WaitDurationCallCount tracks how many times
	// WaitDuration was called.
	WaitDurationCallCount int64

	// DeleteBucketCallCount tracks how many times
	// DeleteBucket was called.
	DeleteBucketCallCount int64
}

var _ TokenBucketStorePort = (*MockTokenBucketStore)(nil)

// TryTake delegates to TryTakeFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes key (string) which identifies the token bucket.
// Takes n (float64) which is the number of tokens to take.
// Takes config (*ratelimiter_dto.TokenBucketConfig) which
// provides the bucket configuration.
//
// Returns (false, nil) if TryTakeFunc is nil.
func (m *MockTokenBucketStore) TryTake(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error) {
	atomic.AddInt64(&m.TryTakeCallCount, 1)
	if m.TryTakeFunc != nil {
		return m.TryTakeFunc(ctx, key, n, config)
	}
	return false, nil
}

// WaitDuration delegates to WaitDurationFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes key (string) which identifies the token bucket.
// Takes n (float64) which is the number of tokens to wait for.
// Takes config (*ratelimiter_dto.TokenBucketConfig) which
// provides the bucket configuration.
//
// Returns (0, nil) if WaitDurationFunc is nil.
func (m *MockTokenBucketStore) WaitDuration(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
	atomic.AddInt64(&m.WaitDurationCallCount, 1)
	if m.WaitDurationFunc != nil {
		return m.WaitDurationFunc(ctx, key, n, config)
	}
	return 0, nil
}

// DeleteBucket delegates to DeleteBucketFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes key (string) which identifies the token bucket to delete.
//
// Returns nil if DeleteBucketFunc is nil.
func (m *MockTokenBucketStore) DeleteBucket(ctx context.Context, key string) error {
	atomic.AddInt64(&m.DeleteBucketCallCount, 1)
	if m.DeleteBucketFunc != nil {
		return m.DeleteBucketFunc(ctx, key)
	}
	return nil
}
