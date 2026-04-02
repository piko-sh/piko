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

// MockCounterStore is a test double for CounterStorePort where nil
// function fields return zero values and call counts are tracked
// atomically.
type MockCounterStore struct {
	// IncrementAndGetFunc is the function called by
	// IncrementAndGet.
	IncrementAndGetFunc func(ctx context.Context, key string, delta int64, window time.Duration) (ratelimiter_dto.CounterResult, error)

	// IncrementAndGetCallCount tracks how many times
	// IncrementAndGet was called.
	IncrementAndGetCallCount int64
}

var _ CounterStorePort = (*MockCounterStore)(nil)

// IncrementAndGet delegates to IncrementAndGetFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes key (string) which identifies the counter to increment.
// Takes delta (int64) which is the amount to add to the counter.
// Takes window (time.Duration) which specifies the time window for the counter.
//
// Returns (CounterResult{}, nil) if IncrementAndGetFunc is nil.
func (m *MockCounterStore) IncrementAndGet(ctx context.Context, key string, delta int64, window time.Duration) (ratelimiter_dto.CounterResult, error) {
	atomic.AddInt64(&m.IncrementAndGetCallCount, 1)
	if m.IncrementAndGetFunc != nil {
		return m.IncrementAndGetFunc(ctx, key, delta, window)
	}
	return ratelimiter_dto.CounterResult{}, nil
}
