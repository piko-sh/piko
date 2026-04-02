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

package cache_provider_redis_cluster

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

// RedisClusterProvider implements the cache.Provider interface for Redis
// Cluster. It manages a single shared client connection across all namespaces,
// where each namespace becomes a key prefix (e.g., "users:", "products:").
type RedisClusterProvider struct {
	// client is the shared Redis Cluster client connection.
	client *redis.ClusterClient

	// caches stores all created cache instances, keyed by namespace.
	caches map[string]any

	// config holds the provider-level settings.
	config Config

	// mu guards concurrent access to provider state.
	mu sync.RWMutex
}

var _ cache.Provider = (*RedisClusterProvider)(nil)

// NewRedisClusterProvider creates a new Redis Cluster provider with a shared
// client connection. All namespaces created from this provider share the same
// Redis Cluster connection.
//
// Takes config (Config) which specifies the cluster addresses, password, timeouts,
// and encoding registry.
//
// Returns *RedisClusterProvider which is the configured provider ready for use.
// Returns error when the registry is nil or the cluster cannot be reached.
func NewRedisClusterProvider(config Config) (*RedisClusterProvider, error) {
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
		return nil, errors.New("redis cluster provider requires an EncodingRegistry in config")
	}

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    config.Addrs,
		Password: config.Password,
	})

	connCause := fmt.Errorf("redis cluster connection exceeded %s timeout", cache_domain.DefaultConnectionTimeout)
	ctx, cancel := context.WithTimeoutCause(context.Background(), cache_domain.DefaultConnectionTimeout, connCause)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to redis cluster at %v: %w", config.Addrs, err)
	}

	provider := &RedisClusterProvider{
		client: client,
		caches: make(map[string]any),
		config: config,
		mu:     sync.RWMutex{},
	}

	_, l := logger.From(context.Background(), log)
	l.Internal("Redis Cluster provider initialised",
		logger.Strings("addresses", config.Addrs))

	return provider, nil
}

// CreateNamespaceTyped creates a new Redis Cluster cache instance for the
// given namespace.
//
// The namespace is used as a key prefix. The same Redis Cluster client is
// shared across all namespaces. This is a non-generic method that uses type
// erasure; call CreateNamespace[K,V]() for type safety.
//
// Takes namespace (string) which specifies the key prefix for the cache.
// Takes options (any) which provides cache configuration settings.
//
// Returns any which is the created cache instance.
// Returns error when the cache cannot be created.
func (p *RedisClusterProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	return createRedisClusterCache(p, namespace, options)
}

// Close releases all resources managed by this provider.
// For Redis Cluster, this closes the shared client connection.
//
// Returns error when the Redis Cluster client fails to close.
//
// Safe for concurrent use.
func (p *RedisClusterProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, l := logger.From(context.Background(), log)

	if err := p.client.Close(); err != nil {
		l.Error("Error closing Redis Cluster client", logger.Error(err))
		return fmt.Errorf("failed to close Redis Cluster client: %w", err)
	}

	p.caches = make(map[string]any)

	l.Internal("Closed Redis Cluster provider")
	return nil
}

// Name returns the provider's identifier.
//
// Returns string which is the unique name for the Redis cluster provider.
func (*RedisClusterProvider) Name() string {
	return "redis-cluster"
}
