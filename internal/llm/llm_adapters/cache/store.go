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
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"

	"piko.sh/piko/internal/cache/cache_adapters/encoder_json"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
	"piko.sh/safeconv"
	"piko.sh/sharded"
)

const (
	// DefaultCacheMaximumSize is the default maximum number of entries for in-memory cache
	// providers.
	DefaultCacheMaximumSize = 10000

	// defaultCacheMaximumWeightBytes is the default approximate memory, in bytes, that the
	// in-memory cache may hold.
	defaultCacheMaximumWeightBytes uint64 = 256 << 20

	// defaultCacheMaximumEntryBytes is the default approximate size, in bytes, above which a
	// response is not cached.
	defaultCacheMaximumEntryBytes uint32 = 8 << 20

	// defaultNamespace is the cache namespace used when the config leaves it empty.
	defaultNamespace = "llm:cache"

	// entryOverheadBytes is the approximate fixed size of an entry and its response before
	// any text is counted.
	entryOverheadBytes = 256

	// metadataValueBytes estimates metadata values of unknown type or beyond walk limits.
	metadataValueBytes = 64

	// metadataScalarBytes is the approximate size of a metadata value header (an interface
	// value), and of a number or boolean held in it.
	metadataScalarBytes = 16

	// metadataMaximumDepth is how deeply nested metadata values are walked when they are
	// measured.
	metadataMaximumDepth = 8

	// metadataMaximumValues is how many metadata values one entry measurement visits before
	// it estimates the rest.
	metadataMaximumValues = 4096

	// float32Bytes is the size of one vector element.
	float32Bytes = 4
)

var (
	// log is the package-level logger for the LLM cache store.
	log = logger_domain.GetLogger("piko/internal/llm/llm_adapters/cache")

	// errEntryLimitExceedsWeight is returned when one entry could be larger than the whole
	// cache.
	errEntryLimitExceedsWeight = errors.New("maximum entry bytes exceeds maximum weight bytes")
)

// Config configures the unified cache store.
type Config struct {
	// CacheService is the cache service used for storing entries; must not be nil.
	CacheService cache_domain.Service

	// Clock provides time operations. If nil, defaults to RealClock.
	Clock clock.Clock

	// Provider names the cache provider.
	//
	// An empty name keeps entries in this process's typed in-memory cache. A named provider
	// stores JSON entries shared by instances using that provider and namespace.
	Provider string

	// Namespace is the key prefix for all cache entries (e.g. "llm:cache").
	Namespace string

	// MaximumEntries is the maximum number of entries for the in-memory cache. Defaults to
	// DefaultCacheMaximumSize.
	MaximumEntries int

	// MaximumWeightBytes limits the in-memory cache's approximate size in bytes.
	//
	// It defaults to 256 MiB. Remote providers manage their own memory and ignore it.
	MaximumWeightBytes uint64

	// MaximumEntryBytes is the approximate size, in bytes, above which a response is not
	// cached. Defaults to 8 MiB and applies to every provider.
	MaximumEntryBytes uint32
}

// Store provides an LLM cache using the internal cache service. It implements
// llm_domain.CacheStorePort.
//
// Get and Set clone responses. Source metadata maps are copied one level deep, so nested
// values in those maps remain shared.
type Store struct {
	// cache stores LLM response entries for fast retrieval.
	cache cache_domain.Cache[string, *storedEntry]

	// clock provides the current time for expiry checks and timestamps.
	clock clock.Clock

	// hits counts the number of successful cache lookups.
	hits *sharded.Adder

	// misses counts cache lookups that found no valid entry.
	misses *sharded.Adder

	// maximumEntryBytes is the approximate size above which a response is not cached.
	maximumEntryBytes uint32
}

// storedEntry holds a cached entry and its read count in this process.
type storedEntry struct {
	// Entry is the cached entry. It is never changed once stored.
	Entry *llm_dto.CacheEntry `json:"entry"`

	// hits counts reads of this entry in this process. It is not encoded, so an entry
	// decoded from a remote provider starts at zero.
	hits sharded.Counter

	// weight is the approximate size of the entry in bytes, measured when it is stored.
	weight uint32
}

// StoreOption configures optional Store dependencies.
type StoreOption func(*Store)

// valueSizer measures metadata values with a limit on how many it visits.
type valueSizer struct {
	// remaining is how many more values may be visited.
	remaining int
}

var (
	_ llm_domain.CacheStorePort = (*Store)(nil)
)

// Get retrieves a cache entry by key and counts the read.
//
// Takes key (string) which identifies the cache entry to retrieve.
//
// Returns *llm_dto.CacheEntry which is a copy of the cached entry owned by the caller, or
// nil if not found or expired.
// Returns error when the retrieval fails.
func (s *Store) Get(ctx context.Context, key string) (*llm_dto.CacheEntry, error) {
	stored, found, err := s.cache.GetIfPresent(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("retrieving cache entry for key %q: %w", key, err)
	}
	if !found || stored == nil || stored.Entry == nil {
		s.misses.Add(1)
		return nil, nil
	}

	if stored.Entry.IsExpiredAt(s.clock.Now()) {
		s.dropExpired(ctx, key)
		s.misses.Add(1)
		return nil, nil
	}

	s.hits.Add(1)
	return stored.recordHit(), nil
}

// Set stores a copy of a cache entry.
//
// It skips responses larger than the maximum entry size. The copy omits
// FallbackInfo.Errors because those errors describe only the original request and cannot
// be encoded for remote providers.
//
// Takes key (string) which identifies the cache entry.
// Takes entry (*llm_dto.CacheEntry) which contains the data to store.
//
// Returns error when the underlying cache operation fails.
func (s *Store) Set(ctx context.Context, key string, entry *llm_dto.CacheEntry) error {
	if entry == nil {
		return nil
	}
	ctx, l := logger_domain.From(ctx, log)

	stored := newStoredEntry(entry)
	if stored.weight > s.maximumEntryBytes {
		l.Debug("Response too large to cache",
			logger_domain.Uint64("approximate_bytes", uint64(stored.weight)),
			logger_domain.Uint64("maximum_bytes", uint64(s.maximumEntryBytes)))
		return nil
	}

	ttl := entry.ExpiresAt.Sub(entry.CreatedAt)
	if ttl <= 0 {
		return s.cache.Set(ctx, key, stored)
	}

	return s.cache.SetWithTTL(ctx, key, stored, ttl)
}

// Delete removes a cache entry by key.
//
// Takes key (string) which identifies the entry to remove.
//
// Returns error when the deletion fails.
func (s *Store) Delete(ctx context.Context, key string) error {
	return s.cache.Invalidate(ctx, key)
}

// Clear removes all cache entries.
//
// Returns error when the operation fails.
func (s *Store) Clear(ctx context.Context) error {
	return s.cache.InvalidateAll(ctx)
}

// GetStats returns cache statistics.
//
// Returns *llm_dto.CacheStats which contains the current hit and miss counts, cache size,
// and estimated cost saved.
// Returns error which is always nil.
func (s *Store) GetStats(_ context.Context) (*llm_dto.CacheStats, error) {
	return &llm_dto.CacheStats{
		Hits:               s.hits.Value(),
		Misses:             s.misses.Value(),
		Size:               int64(s.cache.EstimatedSize()),
		EstimatedCostSaved: maths.ZeroMoney(llm_dto.CostCurrency),
	}, nil
}

// Close releases resources held by the store.
//
// Returns error when the underlying cache fails to close.
func (s *Store) Close(ctx context.Context) error {
	return s.cache.Close(ctx)
}

// dropExpired removes an expired entry, logging a failure instead of returning it, since
// the read is a miss either way.
//
// Takes key (string) which identifies the expired entry.
func (s *Store) dropExpired(ctx context.Context, key string) {
	if err := s.cache.Invalidate(ctx, key); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to remove expired LLM cache entry",
			logger_domain.String(llm_domain.AttrKeyKey, key),
			logger_domain.Error(err))
	}
}

// recordHit counts a read and returns a copy of the entry carrying the count.
//
// Returns *llm_dto.CacheEntry which is a copy the caller owns.
func (e *storedEntry) recordHit() *llm_dto.CacheEntry {
	e.hits.Add(1)

	copied := *e.Entry
	copied.Response = e.Entry.Response.Clone()
	copied.HitCount = e.Entry.HitCount + e.hits.Value()
	return &copied
}

// size measures one value.
//
// Takes value (any) which is the value to measure.
// Takes depth (int) which is how deeply value is nested.
//
// Returns int which is the approximate size in bytes.
func (s *valueSizer) size(value any, depth int) int {
	if value == nil {
		return 0
	}
	if depth > metadataMaximumDepth || s.remaining <= 0 {
		return metadataValueBytes
	}
	s.remaining--

	switch typed := value.(type) {
	case string:
		return metadataScalarBytes + len(typed)
	case []byte:
		return metadataScalarBytes + len(typed)
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return metadataScalarBytes
	case []string:
		return stringsBytes(typed)
	case map[string]string:
		return stringMapBytes(typed)
	case []any:
		return s.sliceBytes(typed, depth)
	case map[string]any:
		return s.mapBytes(typed, depth)
	default:
		return metadataValueBytes
	}
}

// sliceBytes measures the items of a slice, estimating those past the walk limit without
// visiting them.
//
// Takes items ([]any) which is the slice to measure.
// Takes depth (int) which is how deeply the slice is nested.
//
// Returns int which is the approximate size in bytes.
func (s *valueSizer) sliceBytes(items []any, depth int) int {
	size := metadataScalarBytes
	for index, item := range items {
		if s.remaining <= 0 {
			return size + (len(items)-index)*metadataValueBytes
		}
		size += s.size(item, depth+1)
	}
	return size
}

// mapBytes measures the keys and values of a map, estimating the values past the walk
// limit without visiting them.
//
// Takes items (map[string]any) which is the map to measure.
// Takes depth (int) which is how deeply the map is nested.
//
// Returns int which is the approximate size in bytes.
func (s *valueSizer) mapBytes(items map[string]any, depth int) int {
	size := metadataScalarBytes
	visited := 0
	for key, item := range items {
		if s.remaining <= 0 {
			return size + (len(items)-visited)*metadataValueBytes
		}
		size += len(key) + s.size(item, depth+1)
		visited++
	}
	return size
}

// withCache injects a pre-built cache, bypassing the default cache builder.
//
// Takes cache (cache_domain.Cache[string, *storedEntry]) which is the cache to use.
//
// Returns StoreOption which applies this setting to the store.
func withCache(cache cache_domain.Cache[string, *storedEntry]) StoreOption {
	return func(s *Store) {
		s.cache = cache
	}
}

// New creates a new unified cache store backed by internal/cache.
//
// Takes config (Config) which configures the store.
// Takes options (...StoreOption) which are optional functions to configure the store.
//
// Returns *Store which is ready for use.
// Returns error when the config is invalid or cache creation fails.
func New(ctx context.Context, config Config, options ...StoreOption) (*Store, error) {
	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	maximumWeight := cmp.Or(config.MaximumWeightBytes, defaultCacheMaximumWeightBytes)
	maximumEntryBytes := cmp.Or(config.MaximumEntryBytes, defaultCacheMaximumEntryBytes)
	if uint64(maximumEntryBytes) > maximumWeight {
		return nil, fmt.Errorf("%w: %d > %d", errEntryLimitExceedsWeight, maximumEntryBytes, maximumWeight)
	}

	store := &Store{
		cache:             nil,
		clock:             clk,
		hits:              sharded.NewAdder(),
		misses:            sharded.NewAdder(),
		maximumEntryBytes: maximumEntryBytes,
	}

	for _, option := range options {
		option(store)
	}

	if store.cache != nil {
		return store, nil
	}

	if config.CacheService == nil {
		return nil, errors.New("cache service is required")
	}

	maximumSize := config.MaximumEntries
	if maximumSize <= 0 {
		maximumSize = DefaultCacheMaximumSize
	}

	builder := cache_domain.NewCacheBuilder[string, *storedEntry](config.CacheService).
		Namespace(cmp.Or(config.Namespace, defaultNamespace)).
		Clock(clk).
		PerEntryExpiration()
	if config.Provider == "" {
		builder = builder.FactoryBlueprint(FactoryBlueprintName).
			MaximumWeight(maximumWeight).
			Weigher(newWeigher(maximumWeight, maximumSize))
	} else {
		builder = builder.Provider(config.Provider).
			MaximumEntries(maximumSize).
			TypedEncoder(encoder_json.New[*storedEntry]())
	}

	cache, err := builder.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}
	store.cache = cache

	return store, nil
}

// newStoredEntry copies entry for storage and measures it.
//
// Takes entry (*llm_dto.CacheEntry) which is the caller's entry.
//
// Returns *storedEntry which owns its own copy of the entry.
func newStoredEntry(entry *llm_dto.CacheEntry) *storedEntry {
	copied := *entry
	copied.Response = entry.Response.Clone()
	if copied.Response != nil && copied.Response.FallbackInfo != nil {
		copied.Response.FallbackInfo.Errors = nil
	}

	return &storedEntry{
		Entry:  &copied,
		hits:   sharded.Counter{},
		weight: safeconv.IntToUint32(approximateEntryBytes(&copied)),
	}
}

// newWeigher returns the weigher for the in-memory cache. Each entry weighs at least
// maximumWeight / maximumEntries, so the cache never holds more than maximumEntries
// entries, while large responses weigh their approximate size in bytes.
//
// An entry's hit counter may inflate to stripes after the entry is weighed, and the cache
// does not weigh it again, so every entry also counts the counter's inflated size.
//
// Takes maximumWeight (uint64) which is the cache's total weight.
// Takes maximumEntries (int) which is the most entries the cache may hold.
//
// Returns func(string, *storedEntry) uint32 which weighs one entry.
func newWeigher(maximumWeight uint64, maximumEntries int) func(string, *storedEntry) uint32 {
	entries := safeconv.IntToUint64(maximumEntries)
	minimumWeight := safeconv.Uint64ToUint32((maximumWeight + entries - 1) / entries)
	counterWeight := safeconv.IntToUint32(sharded.InflatedCounterBytes())

	return func(_ string, stored *storedEntry) uint32 {
		weight := stored.weight
		if weight <= math.MaxUint32-counterWeight {
			weight += counterWeight
		} else {
			weight = math.MaxUint32
		}
		return max(weight, minimumWeight)
	}
}

// approximateEntryBytes estimates the memory an entry uses, counting the text it holds.
//
// Takes entry (*llm_dto.CacheEntry) which is the entry to measure.
//
// Returns int which is the approximate size in bytes.
func approximateEntryBytes(entry *llm_dto.CacheEntry) int {
	size := entryOverheadBytes + len(entry.RequestHash) + len(entry.Provider) + len(entry.Model)

	response := entry.Response
	if response == nil {
		return size
	}

	size += len(response.ID) + len(response.Model)
	for index := range response.Choices {
		size += approximateMessageBytes(&response.Choices[index].Message)
	}
	for index := range response.Sources {
		source := &response.Sources[index]
		size += len(source.ID) + len(source.Content) + len(source.Vector)*float32Bytes
		if len(source.Metadata) > 0 {
			size += approximateValueBytes(source.Metadata)
		}
	}

	return size
}

// approximateValueBytes estimates the memory a metadata value uses, counting the text it
// holds. The walk stops at metadataMaximumDepth levels and after metadataMaximumValues
// values, and estimates what it does not visit.
//
// Takes value (any) which is the value to measure.
//
// Returns int which is the approximate size in bytes.
func approximateValueBytes(value any) int {
	sizer := valueSizer{remaining: metadataMaximumValues}
	return sizer.size(value, 0)
}

// approximateMessageBytes estimates the memory a message uses, counting the text it
// holds.
//
// Takes message (*llm_dto.Message) which is the message to measure.
//
// Returns int which is the approximate size in bytes.
func approximateMessageBytes(message *llm_dto.Message) int {
	size := len(message.Content)
	for index := range message.ContentParts {
		part := &message.ContentParts[index]
		if part.Text != nil {
			size += len(*part.Text)
		}
		if part.ImageURL != nil {
			size += len(part.ImageURL.URL)
		}
		if part.ImageData != nil {
			size += len(part.ImageData.Data)
		}
	}
	for index := range message.ToolCalls {
		call := &message.ToolCalls[index]
		size += len(call.ID) + len(call.Type) + len(call.Function.Name) + len(call.Function.Arguments)
	}
	return size
}

// stringsBytes estimates the memory a string slice uses.
//
// Takes items ([]string) which is the slice to measure.
//
// Returns int which is the approximate size in bytes.
func stringsBytes(items []string) int {
	size := metadataScalarBytes
	for _, item := range items {
		size += metadataScalarBytes + len(item)
	}
	return size
}

// stringMapBytes estimates the memory a string map uses.
//
// Takes items (map[string]string) which is the map to measure.
//
// Returns int which is the approximate size in bytes.
func stringMapBytes(items map[string]string) int {
	size := metadataScalarBytes
	for key, item := range items {
		size += len(key) + metadataScalarBytes + len(item)
	}
	return size
}
