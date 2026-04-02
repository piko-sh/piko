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

//go:build integration

package llm_integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_adapters/vector_cache"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
	"piko.sh/piko/wdk/cache/cache_linguistics"
	"piko.sh/piko/wdk/cache/cache_provider_redis"
	"piko.sh/piko/wdk/cache/cache_provider_valkey"
	_ "piko.sh/piko/wdk/linguistics/linguistics_language_english"
	"piko.sh/piko/wdk/llm/llm_provider_ollama"
	"piko.sh/piko/wdk/llm/llm_provider_zoltai"
)

const perTestTimeout = 10 * time.Minute

func skipIfNoOllama(t *testing.T) {
	t.Helper()
	if !globalEnv.ollamaAvailable {
		t.Skip("Ollama not available")
	}
}

func createOllamaProvider(t *testing.T) (*ollamaProviderHandle, context.Context) {
	t.Helper()
	require.NotNil(t, globalEnv, "test environment not initialised")
	skipIfNoOllama(t)

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout, fmt.Errorf("test: integration test exceeded %s timeout", perTestTimeout))
	t.Cleanup(cancel)

	ollamaConfig := llm_provider_ollama.Config{
		Host:                  globalEnv.ollamaHost,
		DefaultModel:          modelRef(globalEnv.completionModel, globalEnv.completionDigest),
		DefaultEmbeddingModel: modelRef(globalEnv.embeddingModel, globalEnv.embeddingDigest),
		AutoStart:             new(false),
	}

	provider, err := llm_provider_ollama.NewOllamaProvider(ollamaConfig)
	require.NoError(t, err, "creating ollama provider")

	t.Cleanup(func() {
		_ = provider.Close(t.Context())
	})

	return &ollamaProviderHandle{
		llm:       provider,
		embedding: provider,
	}, ctx
}

type ollamaProviderHandle struct {
	llm       llm_domain.LLMProviderPort
	embedding llm_domain.EmbeddingProviderPort
}

func createLLMService(t *testing.T) (llm_domain.Service, context.Context) {
	t.Helper()

	handle, ctx := createOllamaProvider(t)

	service := llm_domain.NewService("ollama")

	err := service.RegisterProvider(ctx, "ollama", handle.llm)
	require.NoError(t, err, "registering ollama LLM provider")

	err = service.RegisterEmbeddingProvider(ctx, "ollama", handle.embedding)
	require.NoError(t, err, "registering ollama embedding provider")

	err = service.SetDefaultProvider(ctx, "ollama")
	require.NoError(t, err, "setting default LLM provider")

	err = service.SetDefaultEmbeddingProvider("ollama")
	require.NoError(t, err, "setting default embedding provider")

	t.Cleanup(func() {
		_ = service.Close(t.Context())
	})

	return service, ctx
}

func skipIfNoToolSupport(t *testing.T) {
	t.Helper()
	handle, _ := createOllamaProvider(t)
	type toolChecker interface{ SupportsTools() bool }
	if tc, ok := handle.llm.(toolChecker); ok && !tc.SupportsTools() {
		t.Skip("LLM provider does not support tool calling")
	}
	if globalEnv.toolModel == "" {
		t.Skip("no tool model configured")
	}
}

func skipIfNoRedisStack(t *testing.T) {
	t.Helper()
	if globalEnv.redisStackAddr == "" {
		t.Skip("Redis Stack not available")
	}
}

func skipIfNoValkey(t *testing.T) {
	t.Helper()
	if globalEnv.valkeyAddr == "" {
		t.Skip("Valkey not available")
	}
}

func createOtterVectorStore(t *testing.T, dimension int, metric string) *vector_cache.Store {
	t.Helper()

	bp := "otter-vec-" + t.Name()
	cache_domain.RegisterProviderFactory(bp, func(_ cache_domain.Service, _ string, options any) (any, error) {
		opts, ok := options.(cache_dto.Options[string, llm_dto.VectorDocument])
		if !ok {
			return nil, errors.New("invalid options type")
		}
		return provider_otter.OtterProviderFactory[string, llm_dto.VectorDocument](opts)
	})

	cacheService := cache_domain.NewService("otter")

	return vector_cache.New(func(ns string, namespaceConfig *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		m := metric
		if namespaceConfig != nil && namespaceConfig.Metric != "" {
			m = string(namespaceConfig.Metric)
		}
		d := dimension
		if namespaceConfig != nil && namespaceConfig.Dimension > 0 {
			d = namespaceConfig.Dimension
		}
		schema := cache_dto.NewSearchSchema(
			cache_dto.VectorFieldWithMetric("Vector", d, m),
			cache_dto.TextField("Content"),
		)
		return cache_domain.NewCacheBuilder[string, llm_dto.VectorDocument](cacheService).
			FactoryBlueprint(bp).
			Namespace(ns).
			MaximumSize(100000).
			Searchable(schema).
			Build(context.Background())
	})
}

func createRedisVectorStore(t *testing.T, dimension int) *vector_cache.Store {
	t.Helper()
	skipIfNoRedisStack(t)

	valueEncoder, ok := cache_encoder_json.New[llm_dto.VectorDocument]().(cache.AnyEncoder)
	require.True(t, ok, "encoder must implement AnyEncoder")

	registry := cache.NewEncodingRegistry(valueEncoder)

	redisConfig := cache_provider_redis.Config{
		Address:    globalEnv.redisStackAddr,
		Registry:   registry,
		DefaultTTL: 10 * time.Minute,
	}

	provider, err := cache_provider_redis.NewRedisProvider(redisConfig)
	require.NoError(t, err, "creating redis provider")
	t.Cleanup(func() { _ = provider.Close() })

	cacheService := cache_domain.NewService("redis")
	require.NoError(t, cacheService.RegisterProvider(context.Background(), "redis", provider))

	bp := "redis-vec-" + t.Name()
	cache_domain.RegisterProviderFactory(bp, func(_ cache_domain.Service, ns string, options any) (any, error) {
		opts, ok := options.(cache_dto.Options[string, llm_dto.VectorDocument])
		if !ok {
			return nil, errors.New("invalid options type")
		}
		return cache_provider_redis.RedisProviderFactory[string, llm_dto.VectorDocument](provider, ns, opts)
	})

	return vector_cache.New(func(ns string, namespaceConfig *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		return cache_domain.NewCacheBuilder[string, llm_dto.VectorDocument](cacheService).
			FactoryBlueprint(bp).
			Namespace(ns).
			Searchable(cache_dto.NewSearchSchema(
				cache_dto.VectorFieldWithMetric("Vector", dimension, string(namespaceConfig.Metric)),
			)).
			Build(context.Background())
	})
}

func createValkeyVectorStore(t *testing.T, dimension int) *vector_cache.Store {
	t.Helper()
	skipIfNoValkey(t)

	valueEncoder, ok := cache_encoder_json.New[llm_dto.VectorDocument]().(cache.AnyEncoder)
	require.True(t, ok, "encoder must implement AnyEncoder")

	registry := cache.NewEncodingRegistry(valueEncoder)

	valkeyConfig := cache_provider_valkey.Config{
		Address:    globalEnv.valkeyAddr,
		Registry:   registry,
		DefaultTTL: 10 * time.Minute,
	}

	provider, err := cache_provider_valkey.NewValkeyProvider(valkeyConfig)
	require.NoError(t, err, "creating valkey provider")
	t.Cleanup(func() { _ = provider.Close() })

	cacheService := cache_domain.NewService("valkey")
	require.NoError(t, cacheService.RegisterProvider(context.Background(), "valkey", provider))

	bp := "valkey-vec-" + t.Name()
	cache_domain.RegisterProviderFactory(bp, func(_ cache_domain.Service, ns string, options any) (any, error) {
		opts, ok := options.(cache_dto.Options[string, llm_dto.VectorDocument])
		if !ok {
			return nil, errors.New("invalid options type")
		}
		return cache_provider_valkey.ValkeyProviderFactory[string, llm_dto.VectorDocument](provider, ns, opts)
	})

	return vector_cache.New(func(ns string, namespaceConfig *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		return cache_domain.NewCacheBuilder[string, llm_dto.VectorDocument](cacheService).
			FactoryBlueprint(bp).
			Namespace(ns).
			Searchable(cache_dto.NewSearchSchema(
				cache_dto.VectorFieldWithMetric("Vector", dimension, string(namespaceConfig.Metric)),
			)).
			Build(context.Background())
	})
}

func createOtterHybridVectorStore(t *testing.T) *vector_cache.Store {
	t.Helper()

	bp := "otter-hybrid-" + t.Name()
	cache_domain.RegisterProviderFactory(bp, func(_ cache_domain.Service, _ string, options any) (any, error) {
		opts, ok := options.(cache_dto.Options[string, llm_dto.VectorDocument])
		if !ok {
			return nil, errors.New("invalid options type")
		}
		return provider_otter.OtterProviderFactory[string, llm_dto.VectorDocument](opts)
	})

	cacheService := cache_domain.NewService("otter")

	return vector_cache.New(func(ns string, namespaceConfig *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		dim := 0
		if namespaceConfig != nil && namespaceConfig.Dimension > 0 {
			dim = namespaceConfig.Dimension
		}
		analyser := cache_linguistics.NewEnglishTextAnalyser()
		schema := cache_dto.NewSearchSchemaWithAnalyser(
			analyser,
			cache_dto.VectorFieldWithMetric("Vector", dim, "cosine"),
			cache_dto.TextField("Content"),
		)
		return cache_domain.NewCacheBuilder[string, llm_dto.VectorDocument](cacheService).
			FactoryBlueprint(bp).
			Namespace(ns).
			MaximumSize(100000).
			Searchable(schema).
			Build(context.Background())
	})
}

func truncateChunks(maxLen int) llm_domain.TransformFunc {
	return func(doc llm_domain.Document) llm_domain.Document {
		if len(doc.Content) > maxLen {
			doc.Content = doc.Content[:maxLen]
		}
		return doc
	}
}

func modelRef(name, digest string) llm_provider_ollama.ModelRef {
	if digest != "" {
		return llm_provider_ollama.ModelWithDigest(name, digest)
	}
	return llm_provider_ollama.Model(name)
}

func createZoltaiService(t *testing.T, opts ...llm_domain.ServiceOption) (llm_domain.Service, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout, fmt.Errorf("test: integration test exceeded %s timeout", perTestTimeout))
	t.Cleanup(cancel)

	provider, err := llm_provider_zoltai.NewZoltaiProvider(llm_provider_zoltai.Config{
		Seed:     42,
		Fortunes: []string{"The oracle speaks truth"},
	})
	require.NoError(t, err)

	service := llm_domain.NewService("zoltai", opts...)

	require.NoError(t, service.RegisterProvider(ctx, "zoltai", provider))
	require.NoError(t, service.RegisterEmbeddingProvider(ctx, "zoltai", provider))
	require.NoError(t, service.SetDefaultProvider(ctx, "zoltai"))
	require.NoError(t, service.SetDefaultEmbeddingProvider("zoltai"))

	t.Cleanup(func() { _ = service.Close(t.Context()) })

	return service, ctx
}

func createFailingZoltaiService(t *testing.T, primaryProvider llm_domain.LLMProviderPort, opts ...llm_domain.ServiceOption) (llm_domain.Service, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeoutCause(t.Context(), perTestTimeout, fmt.Errorf("test: integration test exceeded %s timeout", perTestTimeout))
	t.Cleanup(cancel)

	fallbackProvider, err := llm_provider_zoltai.NewZoltaiProvider(llm_provider_zoltai.Config{
		Seed:     42,
		Fortunes: []string{"fallback fortune"},
	})
	require.NoError(t, err)

	service := llm_domain.NewService("failing", opts...)

	require.NoError(t, service.RegisterProvider(ctx, "failing", primaryProvider))
	require.NoError(t, service.RegisterProvider(ctx, "zoltai", fallbackProvider))
	require.NoError(t, service.RegisterEmbeddingProvider(ctx, "zoltai", fallbackProvider))
	require.NoError(t, service.SetDefaultProvider(ctx, "failing"))
	require.NoError(t, service.SetDefaultEmbeddingProvider("zoltai"))

	t.Cleanup(func() { _ = service.Close(t.Context()) })

	return service, ctx
}
