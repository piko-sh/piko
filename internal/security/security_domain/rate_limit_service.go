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
	"time"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

// rateLimitService implements RateLimitService by delegating to the
// centralised rate limiter's fixed window algorithm.
type rateLimitService struct {
	// limiter provides the centralised rate limiting algorithms.
	limiter *ratelimiter_domain.Limiter
}

// CheckLimit checks whether a rate limit has been exceeded for the given key.
//
// Takes ctx (context.Context) which propagates cancellation and tracing into
// the underlying counter store call.
// Takes key (string) which identifies the resource being rate limited.
// Takes limit (int) which sets the maximum number of requests allowed.
// Takes window (time.Duration) which sets the time period for the limit.
//
// Returns ratelimiter_dto.Result which contains allowed status, remaining
// count, and reset time.
// Returns error when the storage operation fails or the context is cancelled.
func (s *rateLimitService) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (ratelimiter_dto.Result, error) {
	config := ratelimiter_dto.FixedWindowConfig{
		Limit:  limit,
		Window: window,
	}

	return s.limiter.CheckFixedWindow(ctx, key, config)
}

// NewRateLimitService creates a new rate limit service backed by the
// centralised rate limiter.
//
// Takes limiter (*ratelimiter_domain.Limiter) which provides the rate limiting
// algorithms and storage.
//
// Returns RateLimitService which is ready for use.
func NewRateLimitService(limiter *ratelimiter_domain.Limiter) RateLimitService {
	return &rateLimitService{limiter: limiter}
}
