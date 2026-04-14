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
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

// DynamoDBProvider implements the cache.Provider interface for DynamoDB
// backends. It manages a single shared DynamoDB client connection across all
// namespaces, where each namespace becomes a partition key prefix.
type DynamoDBProvider struct {
	// client is the shared DynamoDB client used by all namespaces.
	client *dynamodb.Client

	// caches stores all created cache instances, keyed by namespace.
	caches map[string]any

	// config holds the provider-level configuration.
	config Config

	// mu guards concurrent access to provider state.
	mu sync.RWMutex
}

var _ cache.Provider = (*DynamoDBProvider)(nil)

// NewDynamoDBProvider creates a new DynamoDB provider with a shared client
// connection. All namespaces created from this provider will share the same
// DynamoDB client.
//
// Takes config (Config) which specifies the DynamoDB connection settings and
// timeouts.
//
// Returns *DynamoDBProvider which is ready to create cache namespaces.
// Returns error when the registry is nil or the DynamoDB table cannot be
// accessed.
func NewDynamoDBProvider(config Config) (*DynamoDBProvider, error) {
	cache_domain.ApplyProviderDefaults(cache_domain.ProviderDefaultsParams{
		DefaultTTL:             &config.DefaultTTL,
		OperationTimeout:       &config.OperationTimeout,
		AtomicOperationTimeout: &config.AtomicOperationTimeout,
		BulkOperationTimeout:   &config.BulkOperationTimeout,
		FlushTimeout:           &config.FlushTimeout,
		MaxComputeRetries:      &config.MaxComputeRetries,
		SearchTimeout:          &config.SearchTimeout,
	})

	if config.Registry == nil {
		return nil, errors.New("dynamodb provider requires an EncodingRegistry in config")
	}

	if config.TableName == "" {
		config.TableName = defaultTableName
	}

	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		cache_domain.DefaultConnectionTimeout,
		fmt.Errorf("dynamodb connection exceeded %s timeout", cache_domain.DefaultConnectionTimeout),
	)
	defer cancel()

	client, awsCfg, err := buildDynamoDBClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("building DynamoDB client: %w", err)
	}

	if err := ensureTableExists(ctx, client, config); err != nil {
		return nil, fmt.Errorf("ensuring DynamoDB table exists: %w", err)
	}

	provider := &DynamoDBProvider{
		client: client,
		caches: make(map[string]any),
		config: config,
		mu:     sync.RWMutex{},
	}

	_, l := logger.From(context.Background(), log)
	l.Internal("DynamoDB provider initialised",
		logger.String(logTableField, config.TableName),
		logger.String("region", awsCfg.Region))

	return provider, nil
}

// buildDynamoDBClient creates a DynamoDB client from the provided config,
// loading the default configuration from the environment when AWSConfig is nil.
//
// Takes config (Config) which provides the AWS and endpoint settings.
//
// Returns *dynamodb.Client which is the configured DynamoDB client.
// Returns aws.Config which is the resolved AWS configuration.
// Returns error when loading the AWS config fails.
func buildDynamoDBClient(ctx context.Context, config Config) (*dynamodb.Client, aws.Config, error) {
	var awsCfg aws.Config
	if config.AWSConfig != nil {
		awsCfg = *config.AWSConfig
	} else {
		var loadOpts []func(*awsconfig.LoadOptions) error
		if config.Region != "" {
			loadOpts = append(loadOpts, awsconfig.WithRegion(config.Region))
		}

		if config.EndpointURL != "" {
			loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider("test", "test", ""),
			))
		}
		var err error
		awsCfg, err = awsconfig.LoadDefaultConfig(ctx, loadOpts...)
		if err != nil {
			return nil, aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
		}
	}

	var clientOpts []func(*dynamodb.Options)
	if config.EndpointURL != "" {
		clientOpts = append(clientOpts, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(config.EndpointURL)
		})
	}

	return dynamodb.NewFromConfig(awsCfg, clientOpts...), awsCfg, nil
}

// CreateNamespaceTyped creates a new DynamoDB cache instance for the given
// namespace using type erasure.
//
// The namespace is used as a partition key prefix, and the same DynamoDB
// client is shared across all namespaces. This is a non-generic method; call
// via CreateNamespace[K,V]() for type safety.
//
// Takes namespace (string) which specifies the key prefix for cache entries.
// Takes options (any) which provides type information extracted via assertion.
//
// Returns any which is the created cache instance.
// Returns error when cache creation fails.
func (p *DynamoDBProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	return createDynamoDBCache(p, namespace, options)
}

// Close releases all resources managed by this provider.
// For DynamoDB, this is a no-op as the SDK client does not require explicit
// closing.
//
// Returns error (always nil for DynamoDB).
//
// Safe for concurrent use. Uses a mutex to protect the close operation.
func (p *DynamoDBProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, l := logger.From(context.Background(), log)

	p.caches = make(map[string]any)

	l.Internal("Closed DynamoDB provider")
	return nil
}

// Name returns the provider's identifier.
//
// Returns string which is the provider name "dynamodb".
func (*DynamoDBProvider) Name() string {
	return "dynamodb"
}

// DynamoDBProviderFactory creates a typed DynamoDB cache instance for a given
// provider and namespace. This is the DynamoDB equivalent of
// [cache_provider_otter.OtterProviderFactory], enabling domain-specific types
// to be stored in DynamoDB via [cache_domain.RegisterProviderFactory].
//
// Takes provider (*DynamoDBProvider) which supplies the DynamoDB connection.
// Takes namespace (string) which specifies the key prefix for cache entries.
// Takes opts (cache.Options[K, V]) which configures the cache behaviour.
//
// Returns the created cache instance and an error when cache creation fails.
func DynamoDBProviderFactory[K comparable, V any](provider *DynamoDBProvider, namespace string, opts cache.Options[K, V]) (cache.Cache[K, V], error) {
	return createNamespaceGeneric(provider, namespace, opts)
}
