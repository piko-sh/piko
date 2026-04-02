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

package cache

import (
	"context"
	"time"
)

// IncrementWithExpiry atomically increments a counter, setting TTL only on
// first increment. This is the pattern required for rate limiting (fixed
// window counters).
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes c (Cache[K, int64]) which is the cache storing the counter.
// Takes key (K) which identifies the counter.
// Takes delta (int64) which is the amount to increment by.
// Takes ttl (time.Duration) which is the TTL to set on new counters.
//
// Returns int64 which is the new counter value after incrementing.
// Returns bool which is true if the operation succeeded.
// Returns error when the operation fails.
func IncrementWithExpiry[K comparable](ctx context.Context, c Cache[K, int64], key K, delta int64, ttl time.Duration) (int64, bool, error) {
	result, present, err := c.ComputeWithTTL(ctx, key, func(oldValue int64, found bool) ComputeResult[int64] {
		newValue := oldValue + delta
		var effectiveTTL time.Duration
		if !found {
			effectiveTTL = ttl
		}
		return ComputeResult[int64]{
			Value:  newValue,
			Action: ComputeActionSet,
			TTL:    effectiveTTL,
		}
	})
	if err != nil {
		return 0, false, err
	}
	if !present {
		return 0, false, nil
	}
	return result, true, nil
}

// GetCounter retrieves the current value of a counter.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes c (Cache[K, int64]) which is the cache storing the counter.
// Takes key (K) which identifies the counter.
//
// Returns int64 which is the current counter value.
// Returns bool which is true if the counter exists.
// Returns error when the operation fails.
func GetCounter[K comparable](ctx context.Context, c Cache[K, int64], key K) (int64, bool, error) {
	return c.GetIfPresent(ctx, key)
}

// ResetCounter removes a counter from the cache.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes c (Cache[K, int64]) which is the cache that stores the counter.
// Takes key (K) which identifies the counter to remove.
//
// Returns error when the operation fails.
func ResetCounter[K comparable](ctx context.Context, c Cache[K, int64], key K) error {
	return c.Invalidate(ctx, key)
}
