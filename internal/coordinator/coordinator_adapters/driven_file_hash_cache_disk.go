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

package coordinator_adapters

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"

	"piko.sh/piko/internal/json"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// cacheDirPermissions is the file mode for cache directories (rwxr-x---).
	cacheDirPermissions = 0750

	// cacheFilePermissions defines the file mode for cache files as 0600, which
	// grants read and write access to the owner only.
	cacheFilePermissions = 0600
)

// cacheEntry stores a file hash and its last modified time.
// Saved as JSON when stored on disk.
type cacheEntry struct {
	// ModTime is when the file was last changed at the time it was hashed.
	ModTime time.Time `json:"mod_time"`

	// Hash is the stored hash value for the file content.
	Hash string `json:"hash"`
}

// diskFileHashCache provides a file hash cache that stores data on disk.
// It implements FileHashCachePort, using an in-memory map for fast lookups
// whilst saving to a JSON file so data persists across restarts.
type diskFileHashCache struct {
	// cache maps absolute file paths to their cache entries. Paths are cleaned
	// with filepath.Clean before use as keys.
	cache map[string]cacheEntry

	// sandbox provides file system access for reading and writing cache files.
	sandbox safedisk.Sandbox

	// factory creates sandboxes with validated paths. When set and sandbox is
	// nil, the factory is used before falling back to NewNoOpSandbox.
	factory safedisk.Factory

	// cacheFileName is the path to the cache file within the sandbox.
	cacheFileName string

	// mu guards the cache map for concurrent read and write access.
	mu sync.RWMutex
}

var _ coordinator_domain.FileHashCachePort = (*diskFileHashCache)(nil)

// DiskFileHashCacheOption configures a DiskFileHashCache during construction.
type DiskFileHashCacheOption func(*diskFileHashCache)

// Get retrieves the cached hash for a file if its modification time matches.
// This uses the "stat-then-read" pattern by letting the caller check the cache
// using only the file's metadata (ModTime from os.Stat).
//
// Takes path (string) which is the file path to look up in the cache.
// Takes modTime (time.Time) which is the current modification time to compare.
//
// Returns hash (string) which is the cached hash value if found and valid.
// Returns found (bool) which indicates whether a valid cache entry exists.
func (d *diskFileHashCache) Get(ctx context.Context, path string, modTime time.Time) (hash string, found bool) {
	return getFromFileHashCache(ctx, &d.mu, d.cache, "DiskFileHashCache.Get", path, modTime)
}

// Set stores or updates the hash for a file with its change time.
// Call this after computing a new hash from the file's contents.
//
// Takes path (string) which is the file path to store the hash for.
// Takes modTime (time.Time) which is the file's last change time.
// Takes hash (string) which is the computed hash of the file's contents.
func (d *diskFileHashCache) Set(ctx context.Context, path string, modTime time.Time, hash string) {
	setInFileHashCache(ctx, &d.mu, d.cache, "DiskFileHashCache.Set", path, modTime, hash)
}

// Load reads the cache from its stored JSON file into memory. Call this when
// setting up the coordinator.
//
// When the cache file does not exist (for example, on first run), this is not
// an error. The cache starts empty. When the file exists but cannot be read or
// contains invalid JSON, an error is returned.
//
// Returns error when the cache file cannot be read or contains invalid JSON.
//
// Safe for concurrent use. The in-memory cache is updated atomically.
func (d *diskFileHashCache) Load(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DiskFileHashCache.Load",
		logger_domain.String(logKeyCacheFilePath, d.cacheFileName),
	)
	defer span.End()

	if d.sandbox == nil {
		l.Internal("File hash cache sandbox not available, skipping load")
		span.SetStatus(codes.Ok, "Sandbox not available - starting fresh")
		return nil
	}

	if _, err := d.sandbox.Stat(d.cacheFileName); os.IsNotExist(err) {
		l.Internal("Cache file does not exist. Starting with empty cache. This is normal on first run.",
			logger_domain.String(logKeyCacheFilePath, d.cacheFileName),
		)
		span.SetStatus(codes.Ok, "Cache file not found - starting fresh")
		return nil
	}

	data, err := d.sandbox.ReadFile(d.cacheFileName)
	if err != nil {
		l.Warn("Failed to read cache file. Starting with empty cache.",
			logger_domain.Error(err),
			logger_domain.String(logKeyCacheFilePath, d.cacheFileName),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read cache file")
		return fmt.Errorf("reading cache file: %w", err)
	}

	var loadedCache map[string]cacheEntry
	if err := cache_domain.CacheAPI.Unmarshal(data, &loadedCache); err != nil {
		l.Warn("Failed to parse cache file (possibly corrupted). Starting with empty cache.",
			logger_domain.Error(err),
			logger_domain.String(logKeyCacheFilePath, d.cacheFileName),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse cache file")
		return fmt.Errorf("parsing cache file JSON: %w", err)
	}

	d.mu.Lock()
	d.cache = loadedCache
	d.mu.Unlock()

	l.Internal("File hash cache loaded successfully.",
		logger_domain.Int("entry_count", len(loadedCache)),
		logger_domain.String(logKeyCacheFilePath, d.cacheFileName),
	)
	span.SetStatus(codes.Ok, "Cache loaded successfully")
	return nil
}

// Persist writes the cache from memory to disk.
//
// Call this during shutdown to keep the cache for future restarts.
//
// Returns error when the cache cannot be serialised or written to disk.
//
// Safe for concurrent use. Takes a read lock to copy the cache, then releases
// it before writing to disk. The file is written to a temporary location
// first, then renamed to the target path. This stops corruption if the process
// is killed during the write.
func (d *diskFileHashCache) Persist(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DiskFileHashCache.Persist",
		logger_domain.String(logKeyCacheFilePath, d.cacheFileName),
	)
	defer span.End()

	if d.sandbox == nil {
		l.Internal("File hash cache sandbox not available, skipping persist")
		span.SetStatus(codes.Ok, "Sandbox not available - skipping persist")
		return nil
	}

	d.mu.RLock()
	snapshot := make(map[string]cacheEntry, len(d.cache))
	maps.Copy(snapshot, d.cache)
	d.mu.RUnlock()

	data, err := json.ConfigStd.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		l.Error("Failed to serialise cache to JSON.",
			logger_domain.Error(err),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to marshal cache")
		return fmt.Errorf("marshalling cache to JSON: %w", err)
	}

	if err := d.writeCacheFile(ctx, data); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to write cache file")
		return fmt.Errorf("writing file hash cache to disk: %w", err)
	}

	l.Internal("File hash cache persisted successfully.",
		logger_domain.Int("entry_count", len(snapshot)),
		logger_domain.String(logKeyCacheFilePath, d.cacheFileName),
	)
	span.SetStatus(codes.Ok, "Cache persisted successfully")
	return nil
}

// writeCacheFile atomically writes the cache data to disk. It writes to a
// temporary file first, then renames it to the target path.
//
// Takes data ([]byte) which contains the cache content to write.
//
// Returns error when creating the directory, writing the temp file, or
// renaming fails.
func (d *diskFileHashCache) writeCacheFile(ctx context.Context, data []byte) error {
	ctx, l := logger_domain.From(ctx, log)
	if d.sandbox == nil {
		l.Internal("File hash cache sandbox not available, skipping write")
		return nil
	}

	if err := d.sandbox.MkdirAll(".", cacheDirPermissions); err != nil {
		l.Error("Failed to create cache directory.",
			logger_domain.Error(err),
		)
		return fmt.Errorf("creating cache directory: %w", err)
	}

	tmpFile := d.cacheFileName + ".tmp"
	if err := d.sandbox.WriteFile(tmpFile, data, cacheFilePermissions); err != nil {
		l.Error("Failed to write cache to temporary file.",
			logger_domain.Error(err),
			logger_domain.String("tmp_file", tmpFile),
		)
		return fmt.Errorf("writing cache to temp file: %w", err)
	}

	if err := d.sandbox.Rename(tmpFile, d.cacheFileName); err != nil {
		l.Error("Failed to atomically move cache file.",
			logger_domain.Error(err),
			logger_domain.String("tmp_file", tmpFile),
			logger_domain.String("target_file", d.cacheFileName),
		)
		return fmt.Errorf("renaming cache file: %w", err)
	}

	return nil
}

// WithCacheSandbox sets a custom sandbox for the file hash cache. Inject a
// mock sandbox to test filesystem operations.
//
// If not provided, a real sandbox is created using safedisk.NewNoOpSandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access within
// the cache directory.
//
// Returns DiskFileHashCacheOption which configures the cache with the given
// sandbox.
func WithCacheSandbox(sandbox safedisk.Sandbox) DiskFileHashCacheOption {
	return func(d *diskFileHashCache) {
		d.sandbox = sandbox
	}
}

// WithCacheFactory sets the sandbox factory for fallback sandbox creation.
// When no sandbox is injected, the factory is tried before falling back to
// NewNoOpSandbox.
//
// Takes factory (safedisk.Factory) which creates sandboxes with validated
// paths.
//
// Returns DiskFileHashCacheOption which configures the cache with the factory.
func WithCacheFactory(factory safedisk.Factory) DiskFileHashCacheOption {
	return func(d *diskFileHashCache) {
		d.factory = factory
	}
}

// NewDiskFileHashCache creates a disk-backed file hash cache that stores
// hashes to avoid computing them again on later runs.
//
// The cacheFilePath must be an absolute path. The parent folder will be
// created when Load() or Persist() is called, if it does not exist.
//
// Takes cacheFilePath (string) which specifies the absolute path to the cache
// file.
// Takes opts (...DiskFileHashCacheOption) which provides optional configuration
// such as WithCacheSandbox for testing.
//
// Returns coordinator_domain.FileHashCachePort which provides the file hash
// caching interface.
func NewDiskFileHashCache(cacheFilePath string, opts ...DiskFileHashCacheOption) coordinator_domain.FileHashCachePort {
	cacheDir := filepath.Dir(cacheFilePath)
	cacheFileName := filepath.Base(cacheFilePath)

	d := &diskFileHashCache{
		cache:         make(map[string]cacheEntry),
		sandbox:       nil,
		cacheFileName: cacheFileName,
		mu:            sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(d)
	}

	if d.sandbox == nil {
		var sandbox safedisk.Sandbox
		var err error
		if d.factory != nil {
			sandbox, err = d.factory.Create("file hash cache", cacheDir, safedisk.ModeReadWrite)
		} else {
			sandbox, err = safedisk.NewNoOpSandbox(cacheDir, safedisk.ModeReadWrite)
		}
		if err != nil {
			return d
		}
		d.sandbox = sandbox
	}

	return d
}
