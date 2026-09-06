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
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/goroutine"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultDeletionWorkers is the default number of goroutines dedicated to deleting
	// expired cache files.
	defaultDeletionWorkers = 4

	// defaultDeleteChanSize is the buffer size for the background deletion channel. A buffer
	// helps prevent the Get method from blocking under high load if workers are busy.
	defaultDeleteChanSize = 128

	// cacheDirectoryPermissions is the file mode for cache directories (owner: rwx, group:
	// rx, other: none).
	cacheDirectoryPermissions = 0750

	// cacheFilePermissions is the file mode for cache files (owner: rw, group: none, other:
	// none).
	cacheFilePermissions = 0600
)

var (
	// metricCacheLevelL2 is a pre-built metric attribute identifying the L2 cache level.
	metricCacheLevelL2 = metric.WithAttributes(attribute.String("cache.level", "l2"))

	// slogCacheLevelL2 is a pre-built log attribute identifying the L2 cache level.
	slogCacheLevelL2 = logger_domain.String("cache.level", "l2")
)

// fbsFileCacheConfig holds the settings for the file-based cache.
type fbsFileCacheConfig struct {
	// Clock provides time functions; nil uses the real system clock.
	Clock clock.Clock

	// Sandbox is an optional sandbox for testing filesystem operations. When nil, one is
	// created for BaseDir; callers must close injected sandboxes.
	Sandbox safedisk.Sandbox

	// SandboxFactory creates sandboxes when Sandbox is nil. When both are nil, a real
	// sandbox rooted at BaseDir is created with safedisk.NewSandbox.
	SandboxFactory safedisk.Factory

	// BaseDir is the folder path where cache files are stored.
	BaseDir string

	// NumDeletionWorkers sets how many workers run for background file deletion. A value of
	// 0 or less uses the default.
	NumDeletionWorkers int

	// DeletionQueueSize is the buffer size for the deletion channel. Zero or negative uses
	// the default.
	DeletionQueueSize int
}

// fbsFileCache implements ASTCache as a persistent, disk-based L2 cache with TTL support.
// It encodes ASTs to FlatBuffers and stores them on the local filesystem, using lazy
// eviction to purge expired items on next access.
//
// Expired and corrupt entries are removed by background deletion workers, which run
// between Start and Shutdown.
type fbsFileCache struct {
	// clock provides the current time for checking cache entry expiry.
	clock clock.Clock

	// sandbox handles file system operations for cache storage.
	sandbox safedisk.Sandbox

	// deleteChan holds cache keys that are waiting for background deletion. It is never
	// closed, so enqueueing after Shutdown cannot panic.
	deleteChan chan string

	// shutdownCh is closed by Shutdown to signal workers to stop processing.
	shutdownCh chan struct{}

	// wg tracks worker pool goroutines so Shutdown can wait for them to finish.
	wg sync.WaitGroup

	// numDeletionWorkers is how many deletion workers Start launches.
	numDeletionWorkers int

	// startOnce ensures the deletion workers are launched at most once.
	startOnce sync.Once

	// shutdownOnce ensures Shutdown releases resources at most once.
	shutdownOnce sync.Once

	// ownsSandbox is true when the cache created its sandbox and must close it on Shutdown;
	// injected sandboxes are closed by their owner.
	ownsSandbox bool
}

// Start launches the background deletion workers.
//
// Calling Start more than once has no further effect. The workers stop when ctx is
// cancelled or Shutdown is called.
func (c *fbsFileCache) Start(ctx context.Context) {
	c.startOnce.Do(func() {
		for range c.numDeletionWorkers {
			c.wg.Go(func() {
				c.runDeletionWorker(ctx)
			})
		}
	})
}

// Shutdown gracefully stops the background deletion workers, waits for them to finish and
// closes the sandbox when the cache created it. Calling Shutdown more than once has no
// further effect.
func (c *fbsFileCache) Shutdown(ctx context.Context) {
	c.shutdownOnce.Do(func() {
		close(c.shutdownCh)
		c.wg.Wait()
		if !c.ownsSandbox {
			return
		}
		if err := c.sandbox.Close(); err != nil {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Failed to close AST cache sandbox", logger_domain.Error(err))
		}
	})
}

// Get reads a file, decodes it, checks for expiration, and returns the AST. If the entry
// is found but has expired, it queues the file for background deletion, and ErrCacheMiss
// is returned, triggering a fresh load from the original source.
//
// Takes key (string) which identifies the cached template to retrieve.
//
// Returns *ast_domain.TemplateAST which is the decoded template AST.
// Returns error when the cache file does not exist, is corrupt, or has expired.
func (c *fbsFileCache) Get(ctx context.Context, key string) (*ast_domain.TemplateAST, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()
	spanCtx, span, spanLog := l.Span(ctx, "fbs_cache.get",
		logger_domain.String("cache.key", key),
		slogCacheLevelL2,
	)
	defer span.End()
	defer func() {
		l2CacheMetrics.latency.Record(spanCtx, float64(time.Since(startTime).Milliseconds()), metricCacheLevelL2)
	}()

	filePath := c.getFilePath(key)

	data, err := c.sandbox.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			l2CacheMetrics.misses.Add(spanCtx, 1, metricCacheLevelL2)
			span.SetStatus(codes.Error, "cache miss")
			return nil, ast_domain.ErrCacheMiss
		}
		spanLog.ReportError(span, err, "failed to read cache file")
		l2CacheMetrics.errors.Add(spanCtx, 1, metricCacheLevelL2)
		return nil, fmt.Errorf("reading cache file for key %q: %w", key, err)
	}

	ast, err := DecodeAST(spanCtx, data)
	if err != nil {
		l2CacheMetrics.errors.Add(spanCtx, 1, metricCacheLevelL2)
		spanLog.ReportError(span, err, "failed to decode corrupt cache file, deleting")
		c.enqueueDeletion(spanCtx, key)
		return nil, ast_domain.ErrCacheMiss
	}

	if ast.ExpiresAtUnixNano != nil {
		if c.clock.Now().UnixNano() > *ast.ExpiresAtUnixNano {
			l2CacheMetrics.evictions.Add(spanCtx, 1, metricCacheLevelL2)
			span.SetAttributes(attribute.Bool("cache.expired", true))

			c.enqueueDeletion(spanCtx, key)

			span.SetStatus(codes.Error, "cache expired")
			return nil, ast_domain.ErrCacheMiss
		}
	}

	l2CacheMetrics.hits.Add(spanCtx, 1, metricCacheLevelL2)
	span.SetStatus(codes.Ok, "")
	return ast, nil
}

// Set stores an AST in the cache with a "never expires" TTL.
//
// Takes key (string) which identifies the cache entry.
// Takes ast (*ast_domain.TemplateAST) which is the parsed template to store.
//
// Returns error when the cache operation fails.
func (c *fbsFileCache) Set(ctx context.Context, key string, ast *ast_domain.TemplateAST) error {
	return c.setInternal(ctx, key, ast, nil)
}

// SetWithTTL stores an AST in the cache with a specific time-to-live.
//
// Takes key (string) which identifies the cache entry.
// Takes ast (*ast_domain.TemplateAST) which is the parsed template to store.
// Takes ttl (time.Duration) which specifies how long the entry remains valid.
//
// Returns error when the cache write fails.
func (c *fbsFileCache) SetWithTTL(ctx context.Context, key string, ast *ast_domain.TemplateAST, ttl time.Duration) error {
	if ttl <= 0 {
		c.enqueueDeletion(ctx, key)
		return nil
	}
	return c.setInternal(ctx, key, ast, new(c.clock.Now().Add(ttl).UnixNano()))
}

// Delete removes the cache file for the given key from the file system.
//
// Takes key (string) which identifies the cache entry to remove.
//
// Returns error when the file cannot be deleted. Missing files are ignored.
func (c *fbsFileCache) Delete(ctx context.Context, key string) error {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()
	err := l.RunInSpan(ctx, "fbs_cache.delete", func(spanCtx context.Context, _ logger_domain.Logger) error {
		filePath := c.getFilePath(key)
		err := c.sandbox.Remove(filePath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			l2CacheMetrics.errors.Add(spanCtx, 1, metricCacheLevelL2)
			return fmt.Errorf("failed to delete cache file: %w", err)
		}

		l2CacheMetrics.deletes.Add(spanCtx, 1, metricCacheLevelL2)
		return nil
	}, logger_domain.String("cache.key", key), slogCacheLevelL2)

	l2CacheMetrics.latency.Record(ctx, float64(time.Since(startTime).Milliseconds()), metricCacheLevelL2)
	return err
}

// runDeletionWorker is the main loop for a background deletion worker. It handles
// deletion requests from the channel until shutdown is signalled or ctx is cancelled.
func (c *fbsFileCache) runDeletionWorker(ctx context.Context) {
	for {
		if shouldExit := c.processDeletionTask(ctx); shouldExit {
			return
		}
	}
}

// processDeletionTask waits for and handles a single deletion task.
//
// Returns bool which is true if the worker should exit because shutdown is signalled or
// ctx is cancelled, or false to continue processing.
func (c *fbsFileCache) processDeletionTask(ctx context.Context) bool {
	select {
	case key := <-c.deleteChan:
		c.executeBackgroundDeletion(ctx, key)
		return false
	case <-c.shutdownCh:
		return true
	case <-ctx.Done():
		return true
	}
}

// executeBackgroundDeletion removes a cache entry for the given key, recovering from any
// panic so that one failing deletion cannot stop the worker.
//
// Cancellation of ctx is ignored, so a deletion already taken from the queue completes.
//
// Takes key (string) which identifies the cache entry to remove.
func (c *fbsFileCache) executeBackgroundDeletion(ctx context.Context, key string) {
	ctx, l := logger_domain.From(context.WithoutCancel(ctx), log)
	err := goroutine.SafeCall(ctx, "ast_adapters.backgroundDeletion", func() error {
		return c.Delete(ctx, key)
	})
	if err != nil {
		l.Warn("Background cache deletion failed", logger_domain.String("key", key), logger_domain.Error(err))
	}
}

// enqueueDeletion sends a key to the deletion channel without blocking. If the channel is
// full, it logs a warning and drops the task to prioritise read performance.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes key (string) which identifies the cache entry to delete.
func (c *fbsFileCache) enqueueDeletion(ctx context.Context, key string) {
	select {
	case c.deleteChan <- key:
	default:
		_, l := logger_domain.From(ctx, log)
		l.Warn("background deletion channel is full, dropping delete task", logger_domain.String("key", key))
	}
}

// setInternal writes an AST and its expiry time to a file.
//
// Takes key (string) which identifies the cache entry.
// Takes ast (*ast_domain.TemplateAST) which is the AST to encode and store.
// Takes expiresAtUnixNano (*int64) which sets when the entry expires.
//
// Returns error when encoding, folder creation, or file writing fails.
func (c *fbsFileCache) setInternal(ctx context.Context, key string, ast *ast_domain.TemplateAST, expiresAtUnixNano *int64) error {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()
	err := l.RunInSpan(ctx, "fbs_cache.set", func(spanCtx context.Context, _ logger_domain.Logger) error {
		ast.ExpiresAtUnixNano = expiresAtUnixNano
		filePath := c.getFilePath(key)

		data, err := EncodeAST(ast)
		if err != nil {
			l2CacheMetrics.errors.Add(spanCtx, 1, metricCacheLevelL2)
			return fmt.Errorf("failed to encode AST for caching: %w", err)
		}

		directory := filepath.Dir(filePath)
		if err := c.sandbox.MkdirAll(directory, cacheDirectoryPermissions); err != nil {
			l2CacheMetrics.errors.Add(spanCtx, 1, metricCacheLevelL2)
			return fmt.Errorf("failed to create cache directory: %w", err)
		}

		tempPath := filepath.Join(directory, "fbs_"+uuid.NewString()+".tmp")
		if err := c.sandbox.WriteFile(tempPath, data, cacheFilePermissions); err != nil {
			l2CacheMetrics.errors.Add(spanCtx, 1, metricCacheLevelL2)
			return fmt.Errorf("failed to write temporary cache file: %w", err)
		}

		if err := c.sandbox.Rename(tempPath, filePath); err != nil {
			_ = c.sandbox.Remove(tempPath)
			l2CacheMetrics.errors.Add(spanCtx, 1, metricCacheLevelL2)
			return fmt.Errorf("failed to atomically move cache file into place: %w", err)
		}

		l2CacheMetrics.sets.Add(spanCtx, 1, metricCacheLevelL2)
		return nil
	}, logger_domain.String("cache.key", key), slogCacheLevelL2)

	l2CacheMetrics.latency.Record(ctx, float64(time.Since(startTime).Milliseconds()), metricCacheLevelL2)
	return err
}

// getFilePath creates a safe file path from a cache key.
//
// Uses xxhash for speed as this is not for security purposes.
//
// Takes key (string) which is the cache key to convert into a file path.
//
// Returns string which is a path like "ab/cd1234...fbs.bin" for use with the sandbox.
func (*fbsFileCache) getFilePath(key string) string {
	hasher := xxhash.New()
	_, _ = hasher.WriteString(key)
	hash := hex.EncodeToString(hasher.Sum(nil))
	return filepath.Join(hash[:2], hash[2:]) + ".fbs.bin"
}

var (
	_ ast_domain.ASTCache = (*fbsFileCache)(nil)
)

// newFbsFileCache creates a new file-based cache. It creates the base folder if it does
// not already exist; the background deletion workers are launched by Start.
//
// Takes config (fbsFileCacheConfig) which specifies the cache settings including base
// folder, worker count, queue size, and optional sandbox.
//
// Returns *fbsFileCache which is the configured cache ready for use.
// Returns error when BaseDir is empty or the folder cannot be created.
func newFbsFileCache(config fbsFileCacheConfig) (*fbsFileCache, error) {
	if config.BaseDir == "" {
		return nil, errors.New("baseDir must be provided")
	}

	sandbox, ownsSandbox, err := resolveCacheSandbox(config)
	if err != nil {
		return nil, err
	}

	if mkdirErr := sandbox.MkdirAll(".", cacheDirectoryPermissions); mkdirErr != nil {
		if ownsSandbox {
			mkdirErr = errors.Join(mkdirErr, sandbox.Close())
		}
		return nil, fmt.Errorf("failed to create cache base directory: %w", mkdirErr)
	}

	if config.NumDeletionWorkers <= 0 {
		config.NumDeletionWorkers = defaultDeletionWorkers
	}
	if config.DeletionQueueSize <= 0 {
		config.DeletionQueueSize = defaultDeleteChanSize
	}

	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	return &fbsFileCache{
		clock:              clk,
		sandbox:            sandbox,
		deleteChan:         make(chan string, config.DeletionQueueSize),
		shutdownCh:         make(chan struct{}),
		wg:                 sync.WaitGroup{},
		numDeletionWorkers: config.NumDeletionWorkers,
		startOnce:          sync.Once{},
		shutdownOnce:       sync.Once{},
		ownsSandbox:        ownsSandbox,
	}, nil
}

// resolveCacheSandbox selects the injected sandbox, one from the configured factory, or a
// real sandbox rooted at the base directory, in that order.
//
// Takes config (fbsFileCacheConfig) which holds the injected sandbox, factory and base
// directory.
//
// Returns safedisk.Sandbox which provides access to the cache directory.
// Returns bool which is true when the sandbox was created here and must be closed by the
// cache.
// Returns error when the sandbox cannot be created.
func resolveCacheSandbox(config fbsFileCacheConfig) (safedisk.Sandbox, bool, error) {
	if config.Sandbox != nil {
		return config.Sandbox, false, nil
	}

	if config.SandboxFactory != nil {
		sandbox, err := config.SandboxFactory.Create("ast-cache", config.BaseDir, safedisk.ModeReadWrite)
		if err != nil {
			return nil, false, fmt.Errorf("failed to create cache sandbox via factory: %w", err)
		}
		return sandbox, true, nil
	}

	sandbox, err := safedisk.NewSandbox(config.BaseDir, safedisk.ModeReadWrite)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create cache sandbox: %w", err)
	}
	return sandbox, true, nil
}
