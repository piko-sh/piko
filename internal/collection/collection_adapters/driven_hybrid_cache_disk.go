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

package collection_adapters

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// hybridCacheDirPermissions defines the Unix permissions for cache directories.
	// The value 0750 grants rwxr-x--- (owner: read/write/execute, group: read/execute,
	// others: none).
	hybridCacheDirPermissions = 0750

	// hybridCacheFilePermissions defines the file permissions for cache files.
	// 0600 = rw------- (owner: read/write, group: none, others: none).
	hybridCacheFilePermissions = 0600

	// logFieldPath is the log field key for file paths.
	logFieldPath = "path"
)

// hybridRegistryAccessor provides access to hybrid registry operations.
// It breaks the import cycle by allowing the adapter to interact with the
// registry without importing the domain package.
type hybridRegistryAccessor interface {
	// Register stores a build-time snapshot for use at runtime.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	// Takes providerName (string) which identifies the data provider.
	// Takes collectionName (string) which names the collection to register.
	// Takes blob ([]byte) which contains the snapshot data.
	// Takes etag (string) which provides a version identifier for the snapshot.
	// Takes config (HybridConfig) which specifies the collection settings.
	Register(ctx context.Context, providerName, collectionName string, blob []byte, etag string, config collection_dto.HybridConfig)

	// GetBlob returns the current FlatBuffer blob and whether revalidation is
	// needed.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	// Takes providerName (string) which identifies the data provider.
	// Takes collectionName (string) which identifies the collection to retrieve.
	//
	// Returns blob ([]byte) which contains the FlatBuffer data.
	// Returns needsRevalidation (bool) which indicates if the blob should be
	// refreshed.
	GetBlob(ctx context.Context, providerName, collectionName string) (blob []byte, needsRevalidation bool)

	// GetETag returns the current ETag for a hybrid collection.
	//
	// Takes providerName (string) which identifies the data provider.
	// Takes collectionName (string) which identifies the collection.
	//
	// Returns string which is the current ETag value.
	GetETag(providerName, collectionName string) string

	// List returns all registered hybrid collection keys.
	List() []string
}

// persistedHybridEntry is the JSON format for storing a hybrid cache entry.
type persistedHybridEntry struct {
	// LastRevalidated is when the entry was last checked to confirm it is still valid.
	LastRevalidated time.Time `json:"last_revalidated"`

	// ProviderName identifies the data provider for this cache entry.
	ProviderName string `json:"provider_name"`

	// CollectionName is the name of the collection within its provider.
	CollectionName string `json:"collection_name"`

	// CurrentETag is the entity tag used for cache validation.
	CurrentETag string `json:"current_etag"`

	// SnapshotETag is the HTTP ETag of the cached snapshot.
	SnapshotETag string `json:"snapshot_etag"`

	// CurrentBlob is the current blob data stored as a base64-encoded string.
	CurrentBlob string `json:"current_blob"`

	// SnapshotBlob is the snapshot data stored as a Base64-encoded string.
	SnapshotBlob string `json:"snapshot_blob"`

	// Config stores the hybrid cache settings for this entry.
	Config collection_dto.HybridConfig `json:"config"`
}

// DiskHybridCacheOption configures a disk-based hybrid cache.
type DiskHybridCacheOption func(*diskHybridCache)

// diskHybridCache is a thread-safe, on-disk implementation of
// HybridPersistencePort.
//
// It saves hybrid collection state to a JSON file so data is kept when the
// process restarts. The in-memory state is managed by the hybrid registry;
// this adapter only handles reading and writing the file.
//
// Design choices:
//   - JSON format for easy reading and debugging
//   - Base64 encoding for binary data (FlatBuffers)
//   - Safe writes using a temp file then rename
//   - Handles missing or broken files without crashing
type diskHybridCache struct {
	// registry holds cache entries by provider and collection name.
	registry hybridRegistryAccessor

	// sandboxFactory creates sandboxes when no sandbox is directly injected.
	// When non-nil and sandbox is nil, this factory is used instead of
	// safedisk.NewNoOpSandbox.
	sandboxFactory safedisk.Factory

	// sandbox handles safe file operations for cache storage.
	sandbox safedisk.Sandbox

	// cacheFileName is the file path for the cache on disk.
	cacheFileName string

	// mu guards the cache fields during Load and Persist operations.
	mu sync.RWMutex
}

// Load reads saved hybrid state from disk and registers it with the registry.
//
// This method uses graceful degradation:
//   - Missing file: Not an error, simply no state to restore (cold start).
//   - Corrupted file: Logs a warning and continues with empty state.
//   - I/O error: Returns an error for the caller to handle.
//
// Returns error when the cache file cannot be read due to an I/O problem.
//
// Safe for concurrent use; holds a lock for the entire operation.
func (d *diskHybridCache) Load(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, l := logger_domain.From(ctx, log)

	if d.sandbox == nil {
		l.Internal("Hybrid cache sandbox not available, skipping load")
		return nil
	}

	l.Internal("Loading hybrid cache from disk",
		logger_domain.String(logFieldPath, d.cacheFileName))

	if _, err := d.sandbox.Stat(d.cacheFileName); os.IsNotExist(err) {
		l.Internal("Hybrid cache file does not exist. Starting with empty cache (cold start).",
			logger_domain.String(logFieldPath, d.cacheFileName))
		return nil
	}

	data, err := d.sandbox.ReadFile(d.cacheFileName)
	if err != nil {
		l.Warn("Failed to read hybrid cache file",
			logger_domain.Error(err),
			logger_domain.String(logFieldPath, d.cacheFileName))
		return fmt.Errorf("reading hybrid cache file: %w", err)
	}

	var entries []persistedHybridEntry
	if err := cache_domain.CacheAPI.Unmarshal(data, &entries); err != nil {
		l.Warn("Failed to parse hybrid cache file (possibly corrupted). Starting with empty cache.",
			logger_domain.Error(err),
			logger_domain.String(logFieldPath, d.cacheFileName))
		return nil
	}

	registeredCount := d.registerEntriesFromCache(ctx, entries)

	l.Internal("Hybrid cache loaded successfully",
		logger_domain.Int("entry_count", registeredCount),
		logger_domain.String(logFieldPath, d.cacheFileName))

	return nil
}

// Persist writes the current hybrid cache state from the registry to disk.
//
// Uses an atomic write method to prevent data loss if the process stops
// during the write. First, it converts the state to JSON. Then it writes to
// a temporary file. Finally, it renames the temp file to the target path,
// which is atomic on POSIX systems.
//
// Returns error when JSON conversion fails or the file cannot be written.
//
// Safe for concurrent use; holds the mutex for the entire operation.
func (d *diskHybridCache) Persist(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, l := logger_domain.From(ctx, log)

	if d.sandbox == nil {
		l.Internal("Hybrid cache sandbox not available, skipping persist")
		return nil
	}

	l.Internal("Persisting hybrid cache to disk",
		logger_domain.String(logFieldPath, d.cacheFileName))

	entries := d.collectEntriesFromRegistry(ctx)

	if len(entries) == 0 {
		l.Internal("No hybrid collections to persist")
		return nil
	}

	data, err := json.ConfigStd.MarshalIndent(entries, "", "  ")
	if err != nil {
		l.Error("Failed to serialise hybrid cache to JSON",
			logger_domain.Error(err))
		return fmt.Errorf("marshalling hybrid cache to JSON: %w", err)
	}

	if err := d.writeFileAtomic(ctx, data); err != nil {
		return fmt.Errorf("writing hybrid cache file atomically: %w", err)
	}

	l.Internal("Hybrid cache persisted successfully",
		logger_domain.Int("entry_count", len(entries)),
		logger_domain.String(logFieldPath, d.cacheFileName))

	return nil
}

// registerEntriesFromCache decodes and registers each persisted entry with
// the registry.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes entries ([]persistedHybridEntry) which contains the persisted entries
// to restore.
//
// Returns int which is the number of entries that were registered.
func (d *diskHybridCache) registerEntriesFromCache(ctx context.Context, entries []persistedHybridEntry) int {
	_, l := logger_domain.From(ctx, log)
	registeredCount := 0
	for i := range entries {
		if ctx.Err() != nil {
			return registeredCount
		}

		entry := &entries[i]
		currentBlob, err := base64.StdEncoding.DecodeString(entry.CurrentBlob)
		if err != nil {
			l.Warn("Failed to decode current blob, skipping entry",
				logger_domain.String("provider", entry.ProviderName),
				logger_domain.String("collection", entry.CollectionName),
				logger_domain.Error(err))
			continue
		}

		d.registry.Register(
			ctx,
			entry.ProviderName,
			entry.CollectionName,
			currentBlob,
			entry.CurrentETag,
			entry.Config,
		)

		registeredCount++
		l.Trace("Restored hybrid collection from cache",
			logger_domain.String("provider", entry.ProviderName),
			logger_domain.String("collection", entry.CollectionName),
			logger_domain.String("etag", entry.CurrentETag))
	}
	return registeredCount
}

// collectEntriesFromRegistry gathers all hybrid entries from the registry.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
//
// Returns []persistedHybridEntry which contains all valid entries with their
// current blob data and ETags.
func (d *diskHybridCache) collectEntriesFromRegistry(ctx context.Context) []persistedHybridEntry {
	keys := d.registry.List()
	entries := make([]persistedHybridEntry, 0, len(keys))

	for _, key := range keys {
		if ctx.Err() != nil {
			return entries
		}

		providerName, collectionName := parseHybridKey(key)
		if providerName == "" {
			continue
		}

		blob, _ := d.registry.GetBlob(ctx, providerName, collectionName)
		if blob == nil {
			continue
		}

		etag := d.registry.GetETag(providerName, collectionName)

		entry := persistedHybridEntry{
			ProviderName:    providerName,
			CollectionName:  collectionName,
			CurrentETag:     etag,
			SnapshotETag:    etag,
			LastRevalidated: time.Now(),
			Config:          collection_dto.DefaultHybridConfig(),
			CurrentBlob:     base64.StdEncoding.EncodeToString(blob),
			SnapshotBlob:    base64.StdEncoding.EncodeToString(blob),
		}

		entries = append(entries, entry)
	}

	return entries
}

// writeFileAtomic writes data to the cache file using atomic rename.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes data ([]byte) which contains the content to write to the cache file.
//
// Returns error when the directory cannot be created, the temporary file
// cannot be written, or the atomic rename fails.
func (d *diskHybridCache) writeFileAtomic(ctx context.Context, data []byte) error {
	_, l := logger_domain.From(ctx, log)

	if err := d.sandbox.MkdirAll(".", hybridCacheDirPermissions); err != nil {
		l.Error("Failed to create hybrid cache directory",
			logger_domain.Error(err))
		return fmt.Errorf("creating hybrid cache directory: %w", err)
	}

	tmpFile := d.cacheFileName + ".tmp"
	if err := d.sandbox.WriteFile(tmpFile, data, hybridCacheFilePermissions); err != nil {
		l.Error("Failed to write hybrid cache to temporary file",
			logger_domain.Error(err),
			logger_domain.String("tmp_file", tmpFile))
		return fmt.Errorf("writing hybrid cache to temp file: %w", err)
	}

	if err := d.sandbox.Rename(tmpFile, d.cacheFileName); err != nil {
		l.Error("Failed to atomically move hybrid cache file",
			logger_domain.Error(err),
			logger_domain.String("tmp_file", tmpFile),
			logger_domain.String("target_file", d.cacheFileName))
		return fmt.Errorf("renaming hybrid cache file: %w", err)
	}

	return nil
}

// WithHybridCacheSandbox injects a custom sandbox for testing.
// If not provided, a real sandbox is created from the cache directory.
//
// Takes sandbox (safedisk.Sandbox) which provides the sandbox to use.
//
// Returns DiskHybridCacheOption which configures the hybrid cache.
func WithHybridCacheSandbox(sandbox safedisk.Sandbox) DiskHybridCacheOption {
	return func(d *diskHybridCache) {
		d.sandbox = sandbox
	}
}

// WithHybridCacheSandboxFactory sets a factory for creating sandboxes when no
// sandbox is directly injected.
//
// Takes factory (safedisk.Factory) which creates sandboxes for cache storage.
//
// Returns DiskHybridCacheOption which configures the hybrid cache.
func WithHybridCacheSandboxFactory(factory safedisk.Factory) DiskHybridCacheOption {
	return func(d *diskHybridCache) {
		d.sandboxFactory = factory
	}
}

// newDiskHybridCache creates a new disk-backed hybrid cache adapter.
//
// The parent folder is created if it does not exist. When sandbox creation
// fails, returns a cache with persistence turned off.
//
// Takes cacheFilePath (string) which is the full path to the JSON file for
// storing data.
// Takes registry (hybridRegistryAccessor) which is the hybrid registry to read
// from and write to.
// Takes opts (variadic DiskHybridCacheOption) which allows optional
// configuration such as injecting a custom sandbox for testing.
//
// Returns *diskHybridCache which is the configured cache adapter.
func newDiskHybridCache(
	cacheFilePath string,
	registry hybridRegistryAccessor,
	opts ...DiskHybridCacheOption,
) *diskHybridCache {
	cacheDir := filepath.Dir(cacheFilePath)
	cacheFileName := filepath.Base(cacheFilePath)

	cache := &diskHybridCache{
		sandbox:       nil,
		cacheFileName: cacheFileName,
		registry:      registry,
		mu:            sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(cache)
	}

	if cache.sandbox == nil && cache.sandboxFactory != nil {
		sandbox, err := cache.sandboxFactory.Create("hybrid-cache", cacheDir, safedisk.ModeReadWrite)
		if err == nil {
			cache.sandbox = sandbox
		}
	}
	if cache.sandbox == nil {
		sandbox, err := safedisk.NewNoOpSandbox(cacheDir, safedisk.ModeReadWrite)
		if err != nil {
			_, l := logger_domain.From(context.Background(), log)
			l.Warn("Failed to create hybrid cache sandbox, cache persistence disabled",
				logger_domain.Error(err),
				logger_domain.String(logFieldPath, cacheFilePath))
			return cache
		}
		cache.sandbox = sandbox
	}

	return cache
}

// parseHybridKey splits a key in "provider:collection" format into its parts.
//
// Takes key (string) which is the combined key to split.
//
// Returns providerName (string) which is the part before the colon.
// Returns collectionName (string) which is the part after the colon.
func parseHybridKey(key string) (providerName, collectionName string) {
	for i := range len(key) {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	return "", ""
}
