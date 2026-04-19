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

package llm_domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

// CacheStorePort is the driven port for cache persistence.
// Implementations must be safe for concurrent access.
type CacheStorePort interface {
	// Get retrieves a cache entry by key.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes key (string) which is the cache key.
	//
	// Returns *llm_dto.CacheEntry if found, nil otherwise.
	// Returns error if the operation fails.
	Get(ctx context.Context, key string) (*llm_dto.CacheEntry, error)

	// Set stores a cache entry.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes key (string) which is the cache key.
	// Takes entry (*llm_dto.CacheEntry) which is the entry to store.
	//
	// Returns error if the operation fails.
	Set(ctx context.Context, key string, entry *llm_dto.CacheEntry) error

	// Delete removes a cache entry by key.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes key (string) which is the cache key.
	//
	// Returns error if the operation fails.
	Delete(ctx context.Context, key string) error

	// Clear removes all cache entries.
	//
	// Takes ctx (context.Context) which controls cancellation.
	//
	// Returns error if the operation fails.
	Clear(ctx context.Context) error

	// GetStats returns cache statistics.
	//
	// Takes ctx (context.Context) which controls cancellation.
	//
	// Returns *llm_dto.CacheStats with current statistics.
	// Returns error if the operation fails.
	GetStats(ctx context.Context) (*llm_dto.CacheStats, error)
}

// CacheKeyGenerator creates cache keys from requests.
type CacheKeyGenerator struct{}

// NewCacheKeyGenerator creates a new CacheKeyGenerator.
//
// Returns *CacheKeyGenerator which is ready to generate cache keys.
func NewCacheKeyGenerator() *CacheKeyGenerator {
	return &CacheKeyGenerator{}
}

// Generate generates a cache key from a completion request and provider. The
// key is a SHA-256 hash of the normalised request content.
//
// Takes request (*llm_dto.CompletionRequest) which is the request to hash.
// Takes provider (string) which is the provider name.
//
// Returns string which is the cache key.
func (*CacheKeyGenerator) Generate(request *llm_dto.CompletionRequest, provider string) string {
	normalised := struct {
		Temperature    *float64
		TopP           *float64
		Seed           *int64
		ResponseFormat *llm_dto.ResponseFormat
		MaxTokens      *int
		Model          string
		Provider       string
		Messages       []llm_dto.Message
		Stop           []string
		Tools          []llm_dto.ToolDefinition
	}{
		Messages:       request.Messages,
		Stop:           request.Stop,
		Tools:          request.Tools,
		Model:          request.Model,
		Provider:       provider,
		Temperature:    request.Temperature,
		TopP:           request.TopP,
		Seed:           request.Seed,
		ResponseFormat: request.ResponseFormat,
		MaxTokens:      request.MaxTokens,
	}

	data, err := json.Marshal(normalised)
	if err != nil {
		data = []byte(request.Model + provider)
		if len(request.Messages) > 0 {
			data = append(data, []byte(request.Messages[0].Content)...)
		}
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CacheManager handles LLM response caching and implements CacheStorePort.
type CacheManager struct {
	// clock provides the time source for cache entry expiration checks.
	clock clock.Clock

	// store provides the underlying storage for cache entries.
	store CacheStorePort

	// generator creates cache keys from completion requests.
	generator *CacheKeyGenerator

	// defaultTTL is the cache entry lifetime when no TTL is specified in config.
	defaultTTL time.Duration
}

// CacheManagerOption is a function type for setting up the CacheManager.
type CacheManagerOption func(*CacheManager)

// NewCacheManager creates a new CacheManager.
//
// Takes store (CacheStorePort) which is the cache storage backend.
// Takes defaultTTL (time.Duration) which is the default TTL for cache entries.
// Takes opts (...CacheManagerOption) which are optional configuration functions.
//
// Returns *CacheManager ready for use.
func NewCacheManager(store CacheStorePort, defaultTTL time.Duration, opts ...CacheManagerOption) *CacheManager {
	if defaultTTL == 0 {
		defaultTTL = time.Hour
	}
	m := &CacheManager{
		clock:      clock.RealClock(),
		store:      store,
		generator:  NewCacheKeyGenerator(),
		defaultTTL: defaultTTL,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// GetOrExecute checks the cache for a response, or executes the function if
// not cached. It handles cache reads, writes, and metrics tracking.
//
// Takes config (*llm_dto.CacheConfig) which configures caching behaviour.
// Takes request (*llm_dto.CompletionRequest) which is the request to cache.
// Takes provider (string) which is the provider name.
// Takes execute (func() (*llm_dto.CompletionResponse, error)) which generates
// the response when not cached.
//
// Returns *llm_dto.CompletionResponse which is either cached or newly generated.
// Returns bool which is true if the response was from cache.
// Returns error when the execute function fails.
func (m *CacheManager) GetOrExecute(
	ctx context.Context,
	config *llm_dto.CacheConfig,
	request *llm_dto.CompletionRequest,
	provider string,
	execute func() (*llm_dto.CompletionResponse, error),
) (*llm_dto.CompletionResponse, bool, error) {
	if config == nil || !config.Enabled {
		response, err := execute()
		return response, false, err
	}

	key := m.resolveKey(config, request, provider)

	if response, hit := m.tryReadCache(ctx, config, key); hit {
		return response, true, nil
	}

	cacheMissCount.Add(ctx, 1)
	response, err := execute()
	if err != nil {
		return nil, false, err
	}

	m.tryWriteCache(ctx, config, key, provider, request.Model, response)
	return response, false, nil
}

// Get retrieves a cached entry by key.
//
// Takes key (string) which is the cache key.
//
// Returns *llm_dto.CacheEntry if found and not expired.
// Returns error if the operation fails.
func (m *CacheManager) Get(ctx context.Context, key string) (*llm_dto.CacheEntry, error) {
	entry, err := m.store.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("retrieving cache entry: %w", err)
	}
	if entry != nil && entry.IsExpiredAt(m.clock.Now()) {
		return nil, nil
	}
	return entry, nil
}

// Set stores a cache entry.
//
// Takes key (string) which is the cache key.
// Takes entry (*llm_dto.CacheEntry) which is the entry to store.
//
// Returns error when the operation fails.
func (m *CacheManager) Set(ctx context.Context, key string, entry *llm_dto.CacheEntry) error {
	return m.store.Set(ctx, key, entry)
}

// Delete removes a cache entry.
//
// Takes key (string) which is the cache key.
//
// Returns error when the operation fails.
func (m *CacheManager) Delete(ctx context.Context, key string) error {
	return m.store.Delete(ctx, key)
}

// Clear removes all cache entries.
//
// Returns error if the operation fails.
func (m *CacheManager) Clear(ctx context.Context) error {
	return m.store.Clear(ctx)
}

// GetStats returns cache statistics.
//
// Returns *llm_dto.CacheStats which contains the current statistics.
// Returns error when the operation fails.
func (m *CacheManager) GetStats(ctx context.Context) (*llm_dto.CacheStats, error) {
	return m.store.GetStats(ctx)
}

// GenerateKey generates a cache key for a request.
//
// Takes request (*llm_dto.CompletionRequest) which is the request.
// Takes provider (string) which is the provider name.
//
// Returns string which is the cache key.
func (m *CacheManager) GenerateKey(request *llm_dto.CompletionRequest, provider string) string {
	return m.generator.Generate(request, provider)
}

// resolveKey returns the cache key, either from config or generated.
//
// Takes config (*llm_dto.CacheConfig) which provides an optional preset key.
// Takes request (*llm_dto.CompletionRequest) which is used for key generation.
// Takes provider (string) which identifies the LLM provider.
//
// Returns string which is the resolved cache key.
func (m *CacheManager) resolveKey(config *llm_dto.CacheConfig, request *llm_dto.CompletionRequest, provider string) string {
	if config.Key != "" {
		return config.Key
	}
	return m.generator.Generate(request, provider)
}

// tryReadCache attempts to read from cache and returns the response if found
// and valid.
//
// Takes config (*llm_dto.CacheConfig) which controls cache read behaviour.
// Takes key (string) which identifies the cached entry to retrieve.
//
// Returns *llm_dto.CompletionResponse which contains the cached response data.
// Returns bool which indicates whether a valid cache entry was found.
func (m *CacheManager) tryReadCache(ctx context.Context, config *llm_dto.CacheConfig, key string) (*llm_dto.CompletionResponse, bool) {
	if config.SkipRead {
		return nil, false
	}

	ctx, l := logger_domain.From(ctx, log)

	entry, err := m.store.Get(ctx, key)
	if err != nil {
		l.Debug("Cache read error",
			logger_domain.String(AttrKeyKey, key[:16]+"..."),
			logger_domain.Error(err),
		)
		return nil, false
	}

	if entry == nil || entry.IsExpiredAt(m.clock.Now()) {
		return nil, false
	}

	cacheHitCount.Add(ctx, 1)
	entry.HitCount++
	l.Debug("Cache hit",
		logger_domain.String(AttrKeyKey, key[:16]+"..."),
		logger_domain.String("model", entry.Model),
	)
	return entry.Response, true
}

// tryWriteCache attempts to write a response to the cache.
//
// Takes config (*llm_dto.CacheConfig) which specifies caching behaviour.
// Takes key (string) which is the cache key for this entry.
// Takes provider (string) which identifies the LLM provider.
// Takes model (string) which identifies the model used.
// Takes response (*llm_dto.CompletionResponse) which is the response to cache.
func (m *CacheManager) tryWriteCache(ctx context.Context, config *llm_dto.CacheConfig, key, provider, model string, response *llm_dto.CompletionResponse) {
	if config.SkipWrite {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	ttl := config.TTL
	if ttl == 0 {
		ttl = m.defaultTTL
	}

	now := m.clock.Now()
	entry := &llm_dto.CacheEntry{
		Response:    response,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
		RequestHash: key,
		Provider:    provider,
		Model:       model,
		HitCount:    0,
	}

	if err := m.store.Set(ctx, key, entry); err != nil {
		l.Debug("Cache write error",
			logger_domain.String(AttrKeyKey, key[:16]+"..."),
			logger_domain.Error(err),
		)
		return
	}

	l.Debug("Cache write",
		logger_domain.String(AttrKeyKey, key[:16]+"..."),
		logger_domain.String("model", model),
	)
}

// WithCacheManagerClock sets the clock used for time operations.
// If not set, clock.RealClock() is used.
//
// Takes c (clock.Clock) which provides time functions.
//
// Returns CacheManagerOption which applies the clock setting to the cache
// manager.
func WithCacheManagerClock(c clock.Clock) CacheManagerOption {
	return func(m *CacheManager) {
		m.clock = c
	}
}
