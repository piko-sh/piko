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
	"context"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

// MockRateLimitService is a test double for RateLimitService that returns
// zero values from nil function fields and tracks call counts atomically.
type MockRateLimitService struct {
	// CheckLimitFunc is the function called by
	// CheckLimit.
	CheckLimitFunc func(ctx context.Context, key string, limit int, window time.Duration) (ratelimiter_dto.Result, error)

	// CheckLimitCallCount tracks how many times
	// CheckLimit was called.
	CheckLimitCallCount int64
}

var _ RateLimitService = (*MockRateLimitService)(nil)

// CheckLimit checks whether the given key has exceeded its rate limit.
//
// Takes ctx (context.Context) which carries cancellation for the underlying
// store call.
// Takes key (string) which identifies the rate limit bucket.
// Takes limit (int) which is the maximum number of requests allowed.
// Takes window (time.Duration) which specifies the time window for the limit.
//
// Returns (Result, error), or (Result{}, nil) if CheckLimitFunc is nil.
func (m *MockRateLimitService) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (ratelimiter_dto.Result, error) {
	atomic.AddInt64(&m.CheckLimitCallCount, 1)
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, key, limit, window)
	}
	return ratelimiter_dto.Result{}, nil
}
