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

package cache_provider_valkey

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/valkey-io/valkey-go"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

// ValkeyProvider implements the cache.Provider interface for Valkey backends.
// It manages a single shared Valkey client connection across all namespaces,
// where each namespace becomes a key prefix (e.g., "users:", "products:").
type ValkeyProvider struct {
	// client is the shared Valkey connection used by all namespaces.
	client valkey.Client

	// caches stores all created cache instances, keyed by namespace.
	caches map[string]any

	// config holds the provider-level configuration.
	config Config

	// mu guards concurrent access to provider state.
	mu sync.RWMutex
}

var _ cache.Provider = (*ValkeyProvider)(nil)

// NewValkeyProvider creates a new Valkey provider with a shared client connection.
// All namespaces created from this provider will share the same Valkey connection.
//
// Takes config (Config) which specifies the Valkey connection
// settings and timeouts.
//
// Returns *ValkeyProvider which is ready to create cache namespaces.
// Returns error when the registry is nil or the Valkey server is unreachable.
func NewValkeyProvider(config Config) (*ValkeyProvider, error) {
	applyConfigDefaults(&config)

	if config.Registry == nil {
		return nil, errors.New("valkey provider requires an EncodingRegistry in config")
	}

	clientOpt := valkey.ClientOption{
		InitAddress:  []string{config.Address},
		Password:     config.Password,
		Username:     config.Username,
		SelectDB:     config.DB,
		ClientName:   config.ClientName,
		TLSConfig:    config.TLSConfig,
		DisableCache: true,
	}

	if config.DisableAutoPipelining {
		clientOpt.AlwaysRESP2 = true
	}

	client, err := valkey.NewClient(clientOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to create valkey client for %s: %w", config.Address, err)
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), cache_domain.DefaultConnectionTimeout, fmt.Errorf("valkey connection exceeded %s timeout", cache_domain.DefaultConnectionTimeout))
	defer cancel()
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to valkey at %s: %w", config.Address, err)
	}

	provider := &ValkeyProvider{
		client: client,
		caches: make(map[string]any),
		config: config,
		mu:     sync.RWMutex{},
	}

	_, l := logger.From(context.Background(), log)
	l.Internal("Valkey provider initialised",
		logger.String("address", config.Address),
		logger.Int("db", config.DB))

	return provider, nil
}

// CreateNamespaceTyped creates a new Valkey cache instance for the given
// namespace using type erasure.
//
// The namespace is used as a key prefix, and the same Valkey client is shared
// across all namespaces. This is a non-generic method; call via
// CreateNamespace[K,V]() for type safety.
//
// Takes namespace (string) which specifies the key prefix for cache entries.
// Takes options (any) which provides type information extracted via assertion.
//
// Returns any which is the created cache instance.
// Returns error when cache creation fails.
func (p *ValkeyProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	return createValkeyCache(p, namespace, options)
}

// Close releases all resources managed by this provider.
// For Valkey, this closes the shared client connection.
//
// Returns error when the Valkey client fails to close.
//
// Safe for concurrent use. Uses a mutex to protect the close operation.
func (p *ValkeyProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, l := logger.From(context.Background(), log)

	p.client.Close()
	p.caches = make(map[string]any)

	l.Internal("Closed Valkey provider")
	return nil
}

// Name returns the provider's identifier.
//
// Returns string which is the provider name "valkey".
func (*ValkeyProvider) Name() string {
	return "valkey"
}

// ValkeyProviderFactory creates a typed Valkey cache instance for a given
// provider and namespace. This is the Valkey equivalent of
// [cache_provider_otter.OtterProviderFactory], enabling domain-specific
// types to be stored in Valkey via [cache_domain.RegisterProviderFactory].
//
// Takes provider (*ValkeyProvider) which supplies the Valkey connection.
// Takes namespace (string) which specifies the key prefix for cache entries.
// Takes opts (cache.Options[K, V]) which configures the cache behaviour.
//
// Returns cache.Cache[K, V] which is the created cache instance.
// Returns error when cache creation fails.
func ValkeyProviderFactory[K comparable, V any](provider *ValkeyProvider, namespace string, opts cache.Options[K, V]) (cache.Cache[K, V], error) {
	return createNamespaceGeneric(provider, namespace, opts)
}
