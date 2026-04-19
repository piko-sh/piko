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
)

// cacheKeyAttr is the attribute key for logging cache operations.
const cacheKeyAttr = "cache.key"

// ASTCacheService provides a high-level, configured caching solution for ASTs.
// It internally manages a multi-level cache (in-memory L1 + file-based L2)
// and exposes it through the standard ast_domain.ASTCache interface.
type ASTCacheService struct {
	// cache stores parsed AST entries and provides Get, Set, and Delete methods.
	cache ast_domain.ASTCache
}

// ASTCacheConfig holds all the necessary configuration for the cache service.
type ASTCacheConfig struct {
	// L2CacheBaseDir is the folder path for the persistent FlatBuffers cache.
	// This must be a folder that the application can read and write to.
	L2CacheBaseDir string

	// L1CacheTTL is the time-to-live for items in the in-memory cache.
	//
	// After this duration, an item is considered expired and a fetch from
	// the L2 cache is triggered on next access. Must be positive. A value
	// of 1 hour is a reasonable default.
	L1CacheTTL time.Duration

	// L1CacheCapacity is the maximum number of ASTs to hold in the
	// fast in-memory cache. A value of 1000 is a reasonable default
	// for many applications.
	L1CacheCapacity int
}

var _ ast_domain.ASTCacheService = (*ASTCacheService)(nil)

// NewASTCacheService creates and sets up the full caching stack based on the
// given configuration.
//
// Takes ctx (context.Context) which is the parent context for background
// worker goroutines.
// Takes config (ASTCacheConfig) which specifies the cache settings including
// L1 capacity, TTL, and L2 base directory.
//
// Returns *ASTCacheService which is the configured caching service ready for
// use.
// Returns error when the configuration is not valid, such as non-positive
// capacity or TTL, empty base directory, or when L2 cache creation fails.
func NewASTCacheService(ctx context.Context, config ASTCacheConfig) (*ASTCacheService, error) {
	if config.L1CacheCapacity <= 0 {
		return nil, fmt.Errorf("L1CacheCapacity must be positive, but was %d", config.L1CacheCapacity)
	}
	if config.L1CacheTTL <= 0 {
		return nil, errors.New("L1CacheTTL must be a positive duration")
	}
	if config.L2CacheBaseDir == "" {
		return nil, errors.New("L2CacheBaseDir must be provided")
	}

	refreshTTL := time.Duration(float64(config.L1CacheTTL) * 0.9)

	l1Cache := otter.Must(&otter.Options[string, *ast_domain.TemplateAST]{
		MaximumSize:       config.L1CacheCapacity,
		ExpiryCalculator:  otter.ExpiryWriting[string, *ast_domain.TemplateAST](config.L1CacheTTL),
		RefreshCalculator: otter.RefreshWriting[string, *ast_domain.TemplateAST](refreshTTL),
	})

	l2Cache, err := newFbsFileCache(fbsFileCacheConfig{Ctx: ctx, BaseDir: config.L2CacheBaseDir})
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 file cache: %w", err)
	}

	multiLevelCache := newMultiLevelCache(l1Cache, l2Cache, config.L1CacheTTL, nil)

	return &ASTCacheService{
		cache: multiLevelCache,
	}, nil
}

// Get retrieves an AST using the multi-level cache.
//
// Takes key (string) which identifies the cached AST entry to retrieve.
//
// Returns *ast_domain.CachedASTEntry which contains the AST and its metadata.
// Returns error when the cache lookup fails or the key is not found.
func (s *ASTCacheService) Get(ctx context.Context, key string) (*ast_domain.CachedASTEntry, error) {
	startTime := time.Now()
	var astEntry *ast_domain.CachedASTEntry
	var ast *ast_domain.TemplateAST

	err := log.RunInSpan(ctx, "cache_service.get", func(spanCtx context.Context, _ logger_domain.Logger) error {
		var getErr error
		ast, getErr = s.cache.Get(spanCtx, key)

		if getErr != nil {
			if errors.Is(getErr, ast_domain.ErrCacheMiss) {
				l2CacheMetrics.serviceMisses.Add(spanCtx, 1)
			}
			return getErr
		}

		l2CacheMetrics.serviceHits.Add(spanCtx, 1)
		return nil
	}, logger_domain.String(cacheKeyAttr, key))

	l2CacheMetrics.serviceLatency.Record(ctx, float64(time.Since(startTime).Milliseconds()))

	if ast != nil {
		astEntry = &ast_domain.CachedASTEntry{
			AST: ast,
		}
		if ast.Metadata != nil {
			astEntry.Metadata = *ast.Metadata
		}
	}

	return astEntry, err
}

// Set stores an AST entry in the multi-level cache.
//
// Takes key (string) which identifies the cache entry.
// Takes astEntry (*ast_domain.CachedASTEntry) which contains the AST and its
// metadata.
//
// Returns error when the cache operation fails.
func (s *ASTCacheService) Set(ctx context.Context, key string, astEntry *ast_domain.CachedASTEntry) error {
	startTime := time.Now()
	err := log.RunInSpan(ctx, "cache_service.set", func(spanCtx context.Context, _ logger_domain.Logger) error {
		ast := astEntry.AST
		ast.Metadata = &astEntry.Metadata
		return s.cache.Set(spanCtx, key, ast)
	}, logger_domain.String(cacheKeyAttr, key))

	l2CacheMetrics.serviceLatency.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	return err
}

// SetWithTTL stores an AST in the multi-level cache with a custom TTL.
//
// Takes key (string) which identifies the cache entry.
// Takes astEntry (*ast_domain.CachedASTEntry) which contains the AST to store.
// Takes ttl (time.Duration) which sets how long the entry stays in the cache.
//
// Returns error when the cache operation fails.
func (s *ASTCacheService) SetWithTTL(ctx context.Context, key string, astEntry *ast_domain.CachedASTEntry, ttl time.Duration) error {
	startTime := time.Now()
	err := log.RunInSpan(ctx, "cache_service.set_with_ttl", func(spanCtx context.Context, _ logger_domain.Logger) error {
		ast := astEntry.AST
		ast.Metadata = &astEntry.Metadata
		return s.cache.SetWithTTL(spanCtx, key, ast, ttl)
	},
		logger_domain.String("cache.key", key),
		logger_domain.Duration("cache.ttl", ttl),
	)

	l2CacheMetrics.serviceLatency.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	return err
}

// Delete removes an AST from the multi-level cache.
//
// Takes key (string) which identifies the cached AST to remove.
//
// Returns error when the cache deletion fails.
func (s *ASTCacheService) Delete(ctx context.Context, key string) error {
	startTime := time.Now()
	err := log.RunInSpan(ctx, "cache_service.delete", func(spanCtx context.Context, _ logger_domain.Logger) error {
		return s.cache.Delete(spanCtx, key)
	}, logger_domain.String(cacheKeyAttr, key))

	l2CacheMetrics.serviceLatency.Record(ctx, float64(time.Since(startTime).Milliseconds()))
	return err
}

// Shutdown stops the cache service in a clean way.
//
// If the cache supports shutdown, calls its Shutdown method.
// Otherwise, does nothing.
func (s *ASTCacheService) Shutdown(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	if shutdowner, ok := s.cache.(interface {
		Shutdown(context.Context)
	}); ok {
		l.Internal("Shutting down underlying AST cache...")
		shutdowner.Shutdown(ctx)
	}
}
