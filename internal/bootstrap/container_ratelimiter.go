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

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/shutdown"
)

const (
	// providerNoop is the store name used when falling back to no-op rate limiting.
	providerNoop = "noop"
)

// SetRateLimiterCacheProvider names the cache provider that holds rate limiter state.
//
// Empty, the default, keeps the state in memory in each process. Limits are shared
// between instances only when this names a remote provider they all use.
//
// Takes name (string) which is a registered cache provider or the built-in "otter".
func (c *Container) SetRateLimiterCacheProvider(name string) {
	c.rateLimiterCacheProvider = name
}

// GetRateLimiter returns the lazily initialised rate limiter.
//
// With no named provider, failure to create cache-backed stores warns and uses no-op
// stores. A named provider's failure returns an error.
//
// Returns *ratelimiter_domain.Limiter which provides token bucket and fixed window rate
// limiting.
// Returns error when the limiter could not be initialised.
func (c *Container) GetRateLimiter() (*ratelimiter_domain.Limiter, error) {
	c.rateLimiterOnce.Do(func() {
		c.createRateLimiter()
	})
	return c.rateLimiter, c.rateLimiterErr
}

// createRateLimiter sets up the centralised rate limiter with cache-backed stores.
//
// It falls back to no-op stores only when no provider is named. A named provider that
// cannot be used sets c.rateLimiterErr instead.
func (c *Container) createRateLimiter() {
	ctx, l := logger_domain.From(c.GetAppContext(), log)
	provider := c.rateLimiterCacheProvider

	cacheService, err := c.GetCacheService()
	if err != nil {
		if provider != "" {
			c.rateLimiterErr = fmt.Errorf("rate limiter cache provider %q needs a cache service: %w", provider, err)
			return
		}
		l.Warn("Rate limiting is disabled because no cache service is configured",
			logger_domain.Error(err))
		c.rateLimiter = newNoopRateLimiter()
		return
	}

	limiter, closeStores, err := newCacheRateLimiter(ctx, cacheService, provider)
	if err != nil {
		if provider != "" {
			c.rateLimiterErr = fmt.Errorf("creating rate limiter stores on cache provider %q: %w", provider, err)
			return
		}
		l.Warn("Rate limiting is disabled because its cache-backed stores could not be created",
			logger_domain.Error(err))
		c.rateLimiter = newNoopRateLimiter()
		return
	}

	c.rateLimiter = limiter
	shutdown.Register(ctx, "RateLimiterStores", closeStores)

	l.Internal("Rate limiter created using cache-backed stores",
		logger_domain.String(fieldProvider, provider))
}

// newNoopRateLimiter builds the rate limiter used when no cache-backed store is
// available, which allows every request.
//
// Returns *ratelimiter_domain.Limiter which is backed by no-op stores.
func newNoopRateLimiter() *ratelimiter_domain.Limiter {
	return ratelimiter_domain.NewLimiter(
		ratelimiter_adapters.NoopTokenBucketStore{},
		ratelimiter_adapters.NoopCounterStore{},
		ratelimiter_domain.WithTokenStoreName(providerNoop),
		ratelimiter_domain.WithCounterStoreName(providerNoop),
	)
}

// newCacheRateLimiter builds a rate limiter whose stores are held by cacheService.
//
// Takes cacheService (cache_domain.Service) which holds the stores' caches.
// Takes provider (string) which names the cache provider. Empty keeps state in memory.
//
// Returns *ratelimiter_domain.Limiter which uses the cache-backed stores.
// Returns shutdown.CleanupFunc which closes both stores and joins their errors.
// Returns error when either store cannot be created. A store already created is closed.
func newCacheRateLimiter(
	ctx context.Context,
	cacheService cache_domain.Service,
	provider string,
) (*ratelimiter_domain.Limiter, shutdown.CleanupFunc, error) {
	tokenStore, err := ratelimiter_adapters.NewCacheTokenBucketStore(ctx, ratelimiter_adapters.CacheTokenBucketStoreConfig{
		CacheService:   cacheService,
		Clock:          nil,
		Provider:       provider,
		Namespace:      "",
		MaximumEntries: 0,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating token bucket store: %w", err)
	}

	counterStore, err := ratelimiter_adapters.NewCacheCounterStore(ctx, ratelimiter_adapters.CacheCounterStoreConfig{
		CacheService:       cacheService,
		Clock:              nil,
		Provider:           provider,
		Namespace:          "",
		MaximumEntries:     0,
		ClockSkewTolerance: 0,
	})
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("creating counter store: %w", err), closeAll(ctx, tokenStore))
	}

	limiter := ratelimiter_domain.NewLimiter(tokenStore, counterStore,
		ratelimiter_domain.WithTokenStoreName("cache"),
		ratelimiter_domain.WithCounterStoreName("cache"),
	)

	closeStores := func(shutdownCtx context.Context) error {
		return closeAll(shutdownCtx, counterStore, tokenStore)
	}

	return limiter, closeStores, nil
}

// closeAll closes every closer, even when an earlier one fails.
//
// Takes closers (...contextCloser) which are the stores to close.
//
// Returns error which joins every close error, or nil when all closed cleanly.
func closeAll(ctx context.Context, closers ...contextCloser) error {
	errs := make([]error, 0, len(closers))
	for _, closer := range closers {
		errs = append(errs, closer.Close(ctx))
	}
	return errors.Join(errs...)
}
