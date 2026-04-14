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

package cache_provider_firestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

const (
	// defaultDatabaseID is the Firestore database ID used when none is
	// configured.
	defaultDatabaseID = "(default)"

	// defaultCollectionPrefix is the top-level collection name used when none
	// is configured.
	defaultCollectionPrefix = "piko_cache"

	// defaultBatchSize is the maximum number of documents processed in a single
	// batch operation.
	defaultBatchSize = 500
)

// FirestoreProvider implements the cache.Provider interface for Firestore
// backends. It manages a single shared Firestore client connection across all
// namespaces, where each namespace becomes a subcollection under the
// configured collection prefix.
type FirestoreProvider struct {
	// client is the shared Firestore connection used by all namespaces.
	client *firestore.Client

	// caches stores all created cache instances, keyed by namespace.
	caches map[string]any

	// config holds the provider-level configuration.
	config Config

	// mu guards concurrent access to provider state.
	mu sync.RWMutex
}

var _ cache.Provider = (*FirestoreProvider)(nil)

// NewFirestoreProvider creates a new Firestore provider with a shared client
// connection. All namespaces created from this provider will share the same
// Firestore connection.
//
// Takes config (Config) which specifies the Firestore connection settings and
// timeouts.
//
// Returns *FirestoreProvider which is ready to create cache namespaces.
// Returns error when the registry is nil or the Firestore client cannot be
// created.
func NewFirestoreProvider(config Config) (*FirestoreProvider, error) {
	applyConfigDefaults(&config)

	if config.Registry == nil {
		return nil, errors.New("firestore provider requires an EncodingRegistry in config")
	}

	if config.EmulatorHost != "" {
		previousHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
		if err := os.Setenv("FIRESTORE_EMULATOR_HOST", config.EmulatorHost); err != nil {
			return nil, fmt.Errorf("failed to set FIRESTORE_EMULATOR_HOST: %w", err)
		}
		defer func() {
			if previousHost == "" {
				_ = os.Unsetenv("FIRESTORE_EMULATOR_HOST")
			} else {
				_ = os.Setenv("FIRESTORE_EMULATOR_HOST", previousHost)
			}
		}()
	}

	client, err := buildFirestoreClient(config)
	if err != nil {
		return nil, fmt.Errorf("creating Firestore client: %w", err)
	}

	provider := &FirestoreProvider{
		client: client,
		caches: make(map[string]any),
		config: config,
		mu:     sync.RWMutex{},
	}

	_, l := logger.From(context.Background(), log)
	l.Internal("Firestore provider initialised",
		logger.String("project", config.ProjectID),
		logger.String("database", config.DatabaseID),
		logger.String("collection_prefix", config.CollectionPrefix))

	return provider, nil
}

// buildFirestoreClient creates a Firestore client from the provided config.
//
// Takes config (Config) which supplies the project, database, and credentials.
//
// Returns *firestore.Client which is the connected Firestore client.
// Returns error when the client cannot be created.
func buildFirestoreClient(config Config) (*firestore.Client, error) {
	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		cache_domain.DefaultConnectionTimeout,
		fmt.Errorf("firestore connection exceeded %s timeout", cache_domain.DefaultConnectionTimeout),
	)
	defer cancel()

	var clientOpts []option.ClientOption
	if len(config.CredentialsJSON) > 0 {
		clientOpts = append(clientOpts, option.WithAuthCredentialsJSON(option.ServiceAccount, config.CredentialsJSON))
	} else if config.CredentialsFile != "" {
		clientOpts = append(clientOpts, option.WithAuthCredentialsFile(option.ServiceAccount, config.CredentialsFile))
	}

	var (
		client *firestore.Client
		err    error
	)

	if config.DatabaseID != defaultDatabaseID {
		client, err = firestore.NewClientWithDatabase(ctx, config.ProjectID, config.DatabaseID, clientOpts...)
	} else {
		client, err = firestore.NewClient(ctx, config.ProjectID, clientOpts...)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client for project %s: %w", config.ProjectID, err)
	}

	return client, nil
}

// applyConfigDefaults fills zero-valued configuration fields with sensible
// defaults.
//
// Takes config (*Config) which is the configuration to populate.
func applyConfigDefaults(config *Config) {
	if config.DatabaseID == "" {
		config.DatabaseID = defaultDatabaseID
	}
	if config.CollectionPrefix == "" {
		config.CollectionPrefix = defaultCollectionPrefix
	}
	if config.BatchSize <= 0 {
		config.BatchSize = defaultBatchSize
	}
	if !config.EnableTTLClientCheck {
		config.EnableTTLClientCheck = true
	}

	cache_domain.ApplyProviderDefaults(cache_domain.ProviderDefaultsParams{
		DefaultTTL:             &config.DefaultTTL,
		OperationTimeout:       &config.OperationTimeout,
		AtomicOperationTimeout: &config.AtomicOperationTimeout,
		BulkOperationTimeout:   &config.BulkOperationTimeout,
		FlushTimeout:           &config.FlushTimeout,
		MaxComputeRetries:      &config.MaxComputeRetries,
		SearchTimeout:          &config.SearchTimeout,
	})
}

// CreateNamespaceTyped creates a new Firestore cache instance for the given
// namespace using type erasure.
//
// The namespace maps to a Firestore subcollection, and the same Firestore
// client is shared across all namespaces. This is a non-generic method; call
// via CreateNamespace[K,V]() for type safety.
//
// Takes namespace (string) which specifies the subcollection name for cache
// entries.
// Takes options (any) which provides type information extracted via assertion.
//
// Returns any which is the created cache instance.
// Returns error when cache creation fails.
func (p *FirestoreProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	return createFirestoreCache(p, namespace, options)
}

// Close releases all resources managed by this provider.
// For Firestore, this closes the shared client connection.
//
// Returns error when the Firestore client fails to close.
//
// Safe for concurrent use. Uses a mutex to protect the close operation.
func (p *FirestoreProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, l := logger.From(context.Background(), log)

	if err := p.client.Close(); err != nil {
		l.Error("Error closing Firestore client", logger.Error(err))
		return fmt.Errorf("failed to close Firestore client: %w", err)
	}

	p.caches = make(map[string]any)

	l.Internal("Closed Firestore provider")
	return nil
}

// Name returns the provider's identifier.
//
// Returns string which is the provider name "firestore".
func (*FirestoreProvider) Name() string {
	return "firestore"
}

// FirestoreProviderFactory creates a typed Firestore cache instance for a
// given provider and namespace. This is the Firestore equivalent of
// [cache_provider_redis.RedisProviderFactory], enabling domain-specific
// types to be stored in Firestore via [cache_domain.RegisterProviderFactory].
//
// Takes provider (*FirestoreProvider) which supplies the Firestore connection.
// Takes namespace (string) which specifies the subcollection for cache
// entries.
// Takes opts (cache.Options[K, V]) which configures the cache behaviour.
//
// Returns the created cache instance and an error when cache creation fails.
func FirestoreProviderFactory[K comparable, V any](provider *FirestoreProvider, namespace string, opts cache.Options[K, V]) (cache.Cache[K, V], error) {
	return createNamespaceGeneric(provider, namespace, opts)
}
