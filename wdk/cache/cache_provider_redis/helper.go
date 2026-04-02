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
	"cmp"
	"context"
	"fmt"
	"reflect"

	"golang.org/x/sync/singleflight"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

// createRedisCache creates a Redis cache for the given namespace using type
// assertions.
//
// Takes p (*RedisProvider) which provides the Redis connection.
// Takes namespace (string) which identifies the cache namespace.
// Takes optionsAny (any) which specifies the cache options with type info.
//
// Returns any which is the created cache instance.
// Returns error when the options type is not supported.
func createRedisCache(p *RedisProvider, namespace string, optionsAny any) (any, error) {
	switch opts := optionsAny.(type) {
	case cache.Options[string, []byte]:
		return createNamespaceGeneric[string, []byte](p, namespace, opts)
	case cache.Options[string, string]:
		return createNamespaceGeneric[string, string](p, namespace, opts)
	case cache.Options[string, int]:
		return createNamespaceGeneric[string, int](p, namespace, opts)
	case cache.Options[int, string]:
		return createNamespaceGeneric[int, string](p, namespace, opts)
	case cache.Options[string, any]:
		return createNamespaceGeneric[string, any](p, namespace, opts)
	case cache.Options[int, int]:
		return createNamespaceGeneric[int, int](p, namespace, opts)
	case cache.Options[int, any]:
		return createNamespaceGeneric[int, any](p, namespace, opts)
	default:
		optionsType := reflect.TypeOf(optionsAny)
		return nil, fmt.Errorf(
			"unsupported cache type %v - for custom domain types, register a ProviderFactoryBlueprint "+
				"via cache_domain.RegisterProviderFactory()",
			optionsType,
		)
	}
}

// createNamespaceGeneric is a helper that handles the type-specific Redis cache
// creation.
//
// Takes p (*RedisProvider) which supplies the Redis client and
// configuration.
// Takes namespace (string) which identifies the cache namespace to
// create or reuse.
// Takes options (cache.Options[K, V]) which configures the cache
// behaviour including expiry and search schema.
//
// Returns the created or reused cache instance and an error when
// the namespace already exists with incompatible types.
//
// Safe for concurrent use. Access is serialised by the provider mutex.
func createNamespaceGeneric[K comparable, V any](p *RedisProvider, namespace string, options cache.Options[K, V]) (cache.Cache[K, V], error) {
	_, l := logger.From(context.Background(), log)

	namespace = cmp.Or(namespace, "default")

	p.mu.Lock()
	defer p.mu.Unlock()

	if existing, exists := p.caches[namespace]; exists {
		if c, ok := existing.(cache.Cache[K, V]); ok {
			l.Internal("Reusing existing Redis namespace",
				logger.String("namespace", namespace))
			return c, nil
		}
		return nil, fmt.Errorf("namespace '%s' already exists with different key/value types", namespace)
	}

	formattedNamespace := namespace
	if formattedNamespace[len(formattedNamespace)-1] != ':' {
		formattedNamespace = formattedNamespace + ":"
	}

	indexName := p.config.IndexPrefix + namespace

	adapter := &RedisAdapter[K, V]{
		expiryCalculator:       options.ExpiryCalculator,
		refreshCalculator:      options.RefreshCalculator,
		sf:                     singleflight.Group{},
		registry:               p.config.Registry,
		client:                 p.client,
		keyRegistry:            p.config.KeyRegistry,
		namespace:              formattedNamespace,
		ttl:                    p.config.DefaultTTL,
		operationTimeout:       p.config.OperationTimeout,
		atomicOperationTimeout: p.config.AtomicOperationTimeout,
		bulkOperationTimeout:   p.config.BulkOperationTimeout,
		flushTimeout:           p.config.FlushTimeout,
		searchTimeout:          p.config.SearchTimeout,
		maxComputeRetries:      p.config.MaxComputeRetries,
		allowUnsafeFLUSHDB:     p.config.AllowUnsafeFLUSHDB,
		schema:                 options.SearchSchema,
		indexName:              indexName,
	}

	p.caches[namespace] = adapter

	l.Internal("Created new Redis namespace",
		logger.String("namespace", namespace),
		logger.String("prefix", formattedNamespace))

	return adapter, nil
}
