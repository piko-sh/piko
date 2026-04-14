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

package cache_provider_dynamodb

import (
	"cmp"
	"context"
	"fmt"
	"reflect"

	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

// createDynamoDBCache creates a DynamoDB cache for the given namespace using
// type assertions.
//
// Takes p (*DynamoDBProvider) which provides the DynamoDB connection.
// Takes namespace (string) which identifies the cache namespace.
// Takes optionsAny (any) which specifies the cache options with type info.
//
// Returns any which is the created cache instance.
// Returns error when the options type is not supported.
func createDynamoDBCache(p *DynamoDBProvider, namespace string, optionsAny any) (any, error) {
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

// createNamespaceGeneric is a helper that handles the type-specific DynamoDB
// cache creation.
//
// Takes p (*DynamoDBProvider) which supplies the DynamoDB client and
// configuration.
// Takes namespace (string) which identifies the cache namespace to create or
// reuse.
// Takes options (cache.Options[K, V]) which configures the cache behaviour
// including expiry and search schema.
//
// Returns the created or reused cache instance and an error when the namespace
// already exists with incompatible types.
//
// Safe for concurrent use. Access is serialised by the provider mutex.
func createNamespaceGeneric[K comparable, V any](p *DynamoDBProvider, namespace string, options cache.Options[K, V]) (cache.Cache[K, V], error) {
	_, l := logger.From(context.Background(), log)

	namespace = cmp.Or(namespace, "default")

	p.mu.Lock()
	defer p.mu.Unlock()

	if existing, exists := p.caches[namespace]; exists {
		if c, ok := existing.(cache.Cache[K, V]); ok {
			l.Internal("Reusing existing DynamoDB namespace",
				logger.String("namespace", namespace))
			return c, nil
		}
		return nil, fmt.Errorf("namespace '%s' already exists with different key/value types", namespace)
	}

	formattedNamespace := namespace
	if formattedNamespace[len(formattedNamespace)-1] != ':' {
		formattedNamespace = formattedNamespace + ":"
	}

	adapter := &DynamoDBAdapter[K, V]{
		expiryCalculator:       options.ExpiryCalculator,
		refreshCalculator:      options.RefreshCalculator,
		sf:                     singleflight.Group{},
		registry:               p.config.Registry,
		client:                 p.client,
		keyRegistry:            p.config.KeyRegistry,
		namespace:              formattedNamespace,
		tableName:              p.config.TableName,
		ttl:                    p.config.DefaultTTL,
		operationTimeout:       p.config.OperationTimeout,
		atomicOperationTimeout: p.config.AtomicOperationTimeout,
		bulkOperationTimeout:   p.config.BulkOperationTimeout,
		flushTimeout:           p.config.FlushTimeout,
		searchTimeout:          p.config.SearchTimeout,
		maxComputeRetries:      p.config.MaxComputeRetries,
		consistentReads:        p.config.ConsistentReads,
		schema:                 options.SearchSchema,
	}

	if options.SearchSchema != nil {
		configureSearchSchema(adapter, options.SearchSchema)
		configureFieldGSIs(p, adapter, l, options.SearchSchema)
	}

	p.caches[namespace] = adapter

	l.Internal("Created new DynamoDB namespace",
		logger.String("namespace", namespace),
		logger.String("prefix", formattedNamespace),
		logger.String(logTableField, p.config.TableName))

	return adapter, nil
}

// configureSearchSchema sets up the field extractor on the adapter from the
// provided search schema.
//
// Takes adapter (*DynamoDBAdapter[K, V]) which is the adapter to configure.
// Takes schema (*cache.SearchSchema) which defines the field configuration.
func configureSearchSchema[K comparable, V any](adapter *DynamoDBAdapter[K, V], schema *cache.SearchSchema) {
	adapter.fieldExtractor = cache_domain.NewFieldExtractor[V](schema)
}

// configureFieldGSIs creates DynamoDB GSIs for searchable fields when
// CreateFieldGSIs is enabled on the provider config.
//
// Takes p (*DynamoDBProvider) which supplies the configuration and client.
// Takes adapter (*DynamoDBAdapter[K, V]) which receives the GSI field mapping.
// Takes l (logger.Logger) which logs warnings when GSI creation fails.
// Takes schema (*cache.SearchSchema) which defines the searchable fields.
func configureFieldGSIs[K comparable, V any](
	p *DynamoDBProvider,
	adapter *DynamoDBAdapter[K, V],
	l logger.Logger,
	schema *cache.SearchSchema,
) {
	if !p.config.CreateFieldGSIs {
		return
	}

	gsiFields, err := ensureFieldGSIs(
		context.Background(), p.client, p.config.TableName,
		p.config.BillingMode, p.config.ReadCapacityUnits, p.config.WriteCapacityUnits,
		schema,
	)
	if err != nil {
		l.Warn("Failed to create field GSIs, queries will use Scan fallback",
			logger.Error(err))
		return
	}

	adapter.gsiFields = gsiFields
}
