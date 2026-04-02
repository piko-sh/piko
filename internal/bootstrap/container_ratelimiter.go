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

package bootstrap

// This file contains rate limiter related container methods.

import (
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
)

// providerNoop is the store name used when falling back to no-op rate limiting.
const providerNoop = "noop"

// GetRateLimiter returns the centralised rate limiter, initialising it lazily.
// It attempts to create cache-backed stores; if the cache is unavailable, it
// falls back to no-op stores with a warning.
//
// Returns *ratelimiter_domain.Limiter which provides token bucket and fixed
// window rate limiting.
// Returns error when the limiter could not be initialised.
func (c *Container) GetRateLimiter() (*ratelimiter_domain.Limiter, error) {
	c.rateLimiterOnce.Do(func() {
		c.createRateLimiter()
	})
	return c.rateLimiter, c.rateLimiterErr
}

// createRateLimiter sets up the centralised rate limiter with cache-backed
// stores, falling back to no-op stores if the cache service is unavailable.
func (c *Container) createRateLimiter() {
	_, l := logger_domain.From(c.GetAppContext(), log)

	cacheService, err := c.GetCacheService()
	if err != nil {
		l.Internal("Cache service not available, using no-op rate limiter stores",
			logger_domain.Error(err))
		c.rateLimiter = ratelimiter_domain.NewLimiter(
			ratelimiter_adapters.NoopTokenBucketStore{},
			ratelimiter_adapters.NoopCounterStore{},
			ratelimiter_domain.WithTokenStoreName(providerNoop),
			ratelimiter_domain.WithCounterStoreName(providerNoop),
		)
		return
	}

	tokenStore, err := ratelimiter_adapters.NewCacheTokenBucketStore(c.GetAppContext(), ratelimiter_adapters.CacheTokenBucketStoreConfig{
		CacheService: cacheService,
	})
	if err != nil {
		l.Warn("Failed to create token bucket store, using no-op stores",
			logger_domain.Error(err))
		c.rateLimiter = ratelimiter_domain.NewLimiter(
			ratelimiter_adapters.NoopTokenBucketStore{},
			ratelimiter_adapters.NoopCounterStore{},
			ratelimiter_domain.WithTokenStoreName(providerNoop),
			ratelimiter_domain.WithCounterStoreName(providerNoop),
		)
		return
	}

	counterStore, err := ratelimiter_adapters.NewCacheCounterStore(c.GetAppContext(), ratelimiter_adapters.CacheCounterStoreConfig{
		CacheService: cacheService,
	})
	if err != nil {
		l.Warn("Failed to create counter store, using no-op stores",
			logger_domain.Error(err))
		tokenStore.Close()
		c.rateLimiter = ratelimiter_domain.NewLimiter(
			ratelimiter_adapters.NoopTokenBucketStore{},
			ratelimiter_adapters.NoopCounterStore{},
			ratelimiter_domain.WithTokenStoreName(providerNoop),
			ratelimiter_domain.WithCounterStoreName(providerNoop),
		)
		return
	}

	c.rateLimiter = ratelimiter_domain.NewLimiter(tokenStore, counterStore,
		ratelimiter_domain.WithTokenStoreName("cache"),
		ratelimiter_domain.WithCounterStoreName("cache"),
	)
	l.Internal("Rate limiter created using cache-backed stores")
}
