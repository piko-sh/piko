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

package ast_adapters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/maypok86/otter/v2"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

// multiLevelCache implements the ASTCache interface by composing a fast
// in-memory L1 cache with a potentially slower, persistent L2 cache.
//
// This implementation uses the idiomatic Otter v2 pattern, using its built-in
// loader mechanism to provide automatic cache-aside logic and thundering herd
// protection.
type multiLevelCache struct {
	// l1Cache is the fast in-memory cache using Otter v2.
	l1Cache *otter.Cache[string, *ast_domain.TemplateAST]

	// l2Cache is the slower second-level cache for persistent storage.
	l2Cache ast_domain.ASTCache

	// clock provides time operations for TTL calculations.
	clock clock.Clock

	// l1DefaultTTL is the default time-to-live for L1 cache entries; 0 uses the
	// remaining L2 TTL instead.
	l1DefaultTTL time.Duration
}

var _ ast_domain.ASTCache = (*multiLevelCache)(nil)

// Get retrieves a template AST using L1 -> L2 read-through caching with TTL
// synchronisation.
//
// Takes key (string) which identifies the cached template.
//
// Returns *ast_domain.TemplateAST which is the cached AST if found.
// Returns error when the key is not found or the L2 cache fails.
func (c *multiLevelCache) Get(ctx context.Context, key string) (*ast_domain.TemplateAST, error) {
	loader := otter.LoaderFunc[string, *ast_domain.TemplateAST](func(ctx context.Context, key string) (*ast_domain.TemplateAST, error) {
		return c.l2Cache.Get(ctx, key)
	})

	ast, err := c.l1Cache.Get(ctx, key, loader)
	if err != nil {
		if errors.Is(err, ast_domain.ErrCacheMiss) {
			return nil, ast_domain.ErrCacheMiss
		}
		return nil, fmt.Errorf("cache lookup for key %q: %w", key, err)
	}

	if ast.ExpiresAtUnixNano != nil {
		expiresAt := time.Unix(0, *ast.ExpiresAtUnixNano)
		remainingL2TTL := expiresAt.Sub(c.clock.Now())

		if remainingL2TTL <= 0 {
			c.l1Cache.Invalidate(key)
			return nil, ast_domain.ErrCacheMiss
		}

		finalL1TTL := remainingL2TTL
		if c.l1DefaultTTL > 0 {
			finalL1TTL = min(remainingL2TTL, c.l1DefaultTTL)
		}
		c.l1Cache.SetExpiresAfter(key, finalL1TTL)
	}

	return ast, nil
}

// Set stores an AST in both cache levels using a write-through strategy.
//
// Takes key (string) which identifies the cache entry.
// Takes ast (*ast_domain.TemplateAST) which is the parsed template to cache.
//
// Returns error when the L2 cache write fails. On L2 failure, the L1 entry is
// removed to keep both cache levels in sync.
func (c *multiLevelCache) Set(ctx context.Context, key string, ast *ast_domain.TemplateAST) error {
	return log.RunInSpan(ctx, "multilevel_cache.set", func(spanCtx context.Context, _ logger_domain.Logger) error {
		c.l1Cache.Set(key, ast)

		if err := c.l2Cache.Set(spanCtx, key, ast); err != nil {
			c.l1Cache.Invalidate(key)
			return fmt.Errorf("writing to L2 cache: %w", err)
		}
		return nil
	}, logger_domain.String("cache.key", key))
}

// SetWithTTL stores an AST in both cache levels with a custom TTL using a
// write-through strategy.
//
// Takes key (string) which identifies the cache entry.
// Takes ast (*ast_domain.TemplateAST) which is the parsed template to cache.
// Takes ttl (time.Duration) which sets how long the entry stays valid.
//
// Returns error when the L2 cache write fails. On L2 failure, the L1 entry is
// removed to keep the caches in sync.
func (c *multiLevelCache) SetWithTTL(ctx context.Context, key string, ast *ast_domain.TemplateAST, ttl time.Duration) error {
	return log.RunInSpan(ctx, "multilevel_cache.set_with_ttl", func(spanCtx context.Context, _ logger_domain.Logger) error {
		c.l1Cache.Set(key, ast)
		c.l1Cache.SetExpiresAfter(key, ttl)

		if err := c.l2Cache.SetWithTTL(spanCtx, key, ast, ttl); err != nil {
			c.l1Cache.Invalidate(key)
			return fmt.Errorf("writing to L2 cache with TTL: %w", err)
		}
		return nil
	}, logger_domain.String("cache.key", key), logger_domain.Duration("cache.ttl", ttl))
}

// Delete removes an entry from both cache levels, using write-through
// invalidation.
//
// Takes key (string) which identifies the cache entry to remove.
//
// Returns error when the operation fails.
func (c *multiLevelCache) Delete(ctx context.Context, key string) error {
	return log.RunInSpan(ctx, "multilevel_cache.delete", func(spanCtx context.Context, _ logger_domain.Logger) error {
		c.l1Cache.Invalidate(key)
		_ = c.l2Cache.Delete(spanCtx, key)

		return nil
	}, logger_domain.String("cache.key", key))
}

// Shutdown stops the L1 cache's background tasks and the L2 cache if it
// supports graceful shutdown.
func (c *multiLevelCache) Shutdown(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	c.l1Cache.StopAllGoroutines()

	if shutdowner, ok := c.l2Cache.(interface {
		Shutdown(context.Context)
	}); ok {
		l.Internal("Shutting down L2 cache...")
		shutdowner.Shutdown(ctx)
	}
}

// newMultiLevelCache creates a new two-level cache for AST storage.
//
// The l1Cache must be a set-up Otter v2 cache. The l2Cache is a component that
// fulfils the ASTCache interface.
//
// Takes l1Cache (*otter.Cache) which provides fast in-memory first-level
// caching.
// Takes l2Cache (ASTCache) which provides second-level storage that lasts
// longer.
// Takes l1DefaultTTL (time.Duration) which sets the default time before L1
// cache entries expire.
// Takes clk (clock.Clock) which provides the time source for TTL
// calculations. If nil, the real system clock is used.
//
// Returns *multiLevelCache which is ready for use with a logger included.
func newMultiLevelCache(
	l1Cache *otter.Cache[string, *ast_domain.TemplateAST],
	l2Cache ast_domain.ASTCache,
	l1DefaultTTL time.Duration,
	clk clock.Clock,
) *multiLevelCache {
	if clk == nil {
		clk = clock.RealClock()
	}
	return &multiLevelCache{
		l1Cache:      l1Cache,
		l2Cache:      l2Cache,
		l1DefaultTTL: l1DefaultTTL,

		clock: clk,
	}
}
