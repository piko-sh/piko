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

package cache_provider_redis

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

// RedisProvider implements the cache.Provider interface for Redis backends.
// It manages a single shared Redis client connection across all namespaces,
// where each namespace becomes a key prefix (e.g., "users:", "products:").
type RedisProvider struct {
	// client is the shared Redis connection used by all namespaces.
	client *redis.Client

	// caches stores all created cache instances, keyed by namespace.
	caches map[string]any

	// config holds the provider-level configuration.
	config Config

	// mu guards concurrent access to provider state.
	mu sync.RWMutex
}

var _ cache.Provider = (*RedisProvider)(nil)

// NewRedisProvider creates a new Redis provider with a shared client connection.
// All namespaces created from this provider will share the same Redis connection.
//
// Takes config (Config) which specifies the Redis connection
// settings and timeouts.
//
// Returns *RedisProvider which is ready to create cache namespaces.
// Returns error when the registry is nil or the Redis server is unreachable.
func NewRedisProvider(config Config) (*RedisProvider, error) {
	cache_domain.ApplyProviderDefaults(cache_domain.ProviderDefaultsParams{
		DefaultTTL:             &config.DefaultTTL,
		OperationTimeout:       &config.OperationTimeout,
		AtomicOperationTimeout: &config.AtomicOperationTimeout,
		BulkOperationTimeout:   &config.BulkOperationTimeout,
		FlushTimeout:           &config.FlushTimeout,
		MaxComputeRetries:      &config.MaxComputeRetries,
		SearchTimeout:          &config.SearchTimeout,
		IndexPrefix:            &config.IndexPrefix,
	})

	if config.Registry == nil {
		return nil, errors.New("redis provider requires an EncodingRegistry in config")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       config.DB,
		Protocol: 2,
	})

	ctx, cancel := context.WithTimeoutCause(context.Background(), cache_domain.DefaultConnectionTimeout, fmt.Errorf("redis connection exceeded %s timeout", cache_domain.DefaultConnectionTimeout))
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to redis at %s: %w", config.Address, err)
	}

	provider := &RedisProvider{
		client: client,
		caches: make(map[string]any),
		config: config,
		mu:     sync.RWMutex{},
	}

	_, l := logger.From(context.Background(), log)
	l.Internal("Redis provider initialised",
		logger.String("address", config.Address),
		logger.Int("db", config.DB))

	return provider, nil
}

// CreateNamespaceTyped creates a new Redis cache instance for the given
// namespace using type erasure.
//
// The namespace is used as a key prefix, and the same Redis client is shared
// across all namespaces. This is a non-generic method; call via
// CreateNamespace[K,V]() for type safety.
//
// Takes namespace (string) which specifies the key prefix for cache entries.
// Takes options (any) which provides type information extracted via assertion.
//
// Returns any which is the created cache instance.
// Returns error when cache creation fails.
func (p *RedisProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	return createRedisCache(p, namespace, options)
}

// Close releases all resources managed by this provider.
// For Redis, this closes the shared client connection.
//
// Returns error when the Redis client fails to close.
//
// Safe for concurrent use. Uses a mutex to protect the close operation.
func (p *RedisProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, l := logger.From(context.Background(), log)

	if err := p.client.Close(); err != nil {
		l.Error("Error closing Redis client", logger.Error(err))
		return fmt.Errorf("failed to close Redis client: %w", err)
	}

	p.caches = make(map[string]any)

	l.Internal("Closed Redis provider")
	return nil
}

// Name returns the provider's identifier.
//
// Returns string which is the provider name "redis".
func (*RedisProvider) Name() string {
	return "redis"
}

// RedisProviderFactory creates a typed Redis cache instance for a given
// provider and namespace. This is the Redis equivalent of
// [cache_provider_otter.OtterProviderFactory], enabling domain-specific
// types to be stored in Redis via [cache_domain.RegisterProviderFactory].
//
// Takes provider (*RedisProvider) which supplies the Redis connection.
// Takes namespace (string) which specifies the key prefix for cache entries.
// Takes opts (cache.Options[K, V]) which configures the cache behaviour.
//
// Returns the created cache instance and an error when cache
// creation fails.
func RedisProviderFactory[K comparable, V any](provider *RedisProvider, namespace string, opts cache.Options[K, V]) (cache.Cache[K, V], error) {
	return createNamespaceGeneric(provider, namespace, opts)
}
