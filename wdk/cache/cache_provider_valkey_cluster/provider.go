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

package cache_provider_valkey_cluster

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

// ValkeyClusterProvider implements the cache.Provider interface for Valkey
// Cluster. It manages a single shared client connection across all namespaces,
// where each namespace becomes a key prefix (e.g., "users:", "products:").
type ValkeyClusterProvider struct {
	// client is the shared Valkey Cluster client connection.
	client valkey.Client

	// caches stores all created cache instances, keyed by namespace.
	caches map[string]any

	// config holds the provider-level settings.
	config Config

	// mu guards concurrent access to provider state.
	mu sync.RWMutex
}

var _ cache.Provider = (*ValkeyClusterProvider)(nil)

// NewValkeyClusterProvider creates a new Valkey Cluster provider with a shared
// client connection. All namespaces created from this provider share the same
// Valkey Cluster connection.
//
// Takes config (Config) which specifies the cluster addresses, password, timeouts,
// and encoding registry.
//
// Returns *ValkeyClusterProvider which is the configured provider ready for use.
// Returns error when the registry is nil or the cluster cannot be reached.
func NewValkeyClusterProvider(config Config) (*ValkeyClusterProvider, error) {
	applyConfigDefaults(&config)

	if config.Registry == nil {
		return nil, errors.New("valkey cluster provider requires an EncodingRegistry in config")
	}

	clientOpt := valkey.ClientOption{
		InitAddress:  config.InitAddress,
		Password:     config.Password,
		Username:     config.Username,
		ClientName:   config.ClientName,
		TLSConfig:    config.TLSConfig,
		DisableCache: true,
	}

	if config.SendToReplicas {
		clientOpt.SendToReplicas = func(command valkey.Completed) bool {
			return command.IsReadOnly()
		}
	}

	client, err := valkey.NewClient(clientOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to create valkey cluster client for %v: %w", config.InitAddress, err)
	}

	connCause := fmt.Errorf("valkey cluster connection exceeded %s timeout", cache_domain.DefaultConnectionTimeout)
	ctx, cancel := context.WithTimeoutCause(context.Background(), cache_domain.DefaultConnectionTimeout, connCause)
	defer cancel()
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to valkey cluster at %v: %w", config.InitAddress, err)
	}

	provider := &ValkeyClusterProvider{
		client: client,
		caches: make(map[string]any),
		config: config,
		mu:     sync.RWMutex{},
	}

	_, l := logger.From(context.Background(), log)
	l.Internal("Valkey Cluster provider initialised",
		logger.Strings("addresses", config.InitAddress))

	return provider, nil
}

// CreateNamespaceTyped creates a new Valkey Cluster cache instance for the
// given namespace.
//
// The namespace is used as a key prefix, and the same Valkey Cluster client
// is shared across all namespaces. This is a non-generic method that uses
// type erasure; call via CreateNamespace[K,V]() for type safety.
//
// Takes namespace (string) which specifies the key prefix for cache entries.
// Takes options (any) which provides configuration for the cache instance.
//
// Returns any which is the created cache instance.
// Returns error when the cache instance cannot be created.
func (p *ValkeyClusterProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	return createValkeyClusterCache(p, namespace, options)
}

// Close releases all resources managed by this provider.
// For Valkey Cluster, this closes the shared client connection.
//
// Returns error when the Valkey Cluster client fails to close.
//
// Safe for concurrent use.
func (p *ValkeyClusterProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, l := logger.From(context.Background(), log)

	p.client.Close()
	p.caches = make(map[string]any)

	l.Internal("Closed Valkey Cluster provider")
	return nil
}

// Name returns the provider's identifier.
//
// Returns string which is the unique name for the Valkey cluster provider.
func (*ValkeyClusterProvider) Name() string {
	return "valkey-cluster"
}
