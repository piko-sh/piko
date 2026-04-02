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

package cache_provider_otter

import (
	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/cache"
)

// PersistenceConfig configures disk-based persistence for an Otter cache,
// enabling durable writes and automatic recovery on restart. Both KeyCodec
// and ValueCodec must be set when persistence is enabled.
type PersistenceConfig[K comparable, V any] = provider_otter.PersistenceConfig[K, V]

// PersistConfig configures persistence behaviour including the storage
// directory, sync mode, snapshot thresholds, and compression settings.
type PersistConfig = wal_domain.Config

// KeyCodec handles serialisation of cache keys to and from bytes for
// persistence.
type KeyCodec[K comparable] interface {
	wal_domain.KeyCodec[K]
}

// ValueCodec handles serialisation of cache values to and from bytes for
// persistence.
type ValueCodec[V any] interface {
	wal_domain.ValueCodec[V]
}

// DefaultPersistConfig returns a PersistConfig with sensible defaults for
// the given directory.
//
// Takes directory (string) which specifies the directory path for persistence.
//
// Returns PersistConfig which contains the default persistence settings.
func DefaultPersistConfig(directory string) PersistConfig {
	return wal_domain.DefaultConfig(directory)
}

// DefaultPersistConfigNamed returns a PersistConfig using the default
// .piko/wal/{name} directory.
//
// Takes name (string) which specifies the subdirectory name within the WAL path.
//
// Returns PersistConfig which is configured to use the default WAL directory.
func DefaultPersistConfigNamed(name string) PersistConfig {
	return wal_domain.DefaultConfigNamed(name)
}

// NewOtterProvider creates a new Otter provider using the namespace pattern.
// This is the RECOMMENDED way to use Otter caches.
//
// The provider can be registered with a cache service and then used to create
// multiple namespaced cache instances with different types.
//
// Returns cache.Provider which is the Otter provider ready for registration.
//
// Example:
//
//	service := cache.NewService("otter")
//	provider := cache_provider_otter.NewOtterProvider()
//	service.RegisterProvider("otter", provider)
//
//	// Create multiple namespaced caches
//	userCache, _ := cache.NewCacheBuilder[string, User](service).
//	    WithProvider("otter").
//	    WithNamespace("users").
//	    Build(ctx)
//
//	productCache, _ := cache.NewCacheBuilder[int, Product](service).
//	    WithProvider("otter").
//	    WithNamespace("products").
//	    Build(ctx)
func NewOtterProvider() cache.Provider {
	return provider_otter.NewOtterProvider()
}

// OtterProviderFactory creates a typed Otter cache instance from the given
// options. It is intended for use inside a [cache.RegisterProviderFactory]
// callback.
//
// Takes opts (cache.Options[K, V]) which configures the cache instance.
//
// Returns cache.Cache[K, V] which is the configured Otter cache.
// Returns error when the cache cannot be created.
//
// Example:
//
//	func init() {
//	    cache.RegisterProviderFactory("my-cache",
//	        func(service cache.Service, namespace string, options any) (any, error) {
//	            opts, ok := options.(cache.Options[string, *MyType])
//	            if !ok {
//	                return nil, errors.New("invalid options type")
//	            }
//	            return cache_provider_otter.OtterProviderFactory[string, *MyType](opts)
//	        },
//	    )
//	}
func OtterProviderFactory[K comparable, V any](opts cache.Options[K, V]) (cache.Cache[K, V], error) {
	return provider_otter.OtterProviderFactory[K, V](opts)
}
