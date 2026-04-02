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

package cache_integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/cache/cache_test/conformance"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
	"piko.sh/piko/wdk/cache/cache_provider_redis"
	"piko.sh/piko/wdk/cache/cache_provider_redis_cluster"
	"piko.sh/piko/wdk/cache/cache_provider_valkey"
	"piko.sh/piko/wdk/cache/cache_provider_valkey_cluster"
)

func TestRedisConformance(t *testing.T) {
	t.Parallel()

	config := conformance.StringConfig{
		ProviderFactory: func(t *testing.T, opts cache_dto.Options[string, string]) cache_domain.Cache[string, string] {
			t.Helper()
			return newRedisConformanceCache(t, opts)
		},
		SupportsSearch:             false,
		SupportsTTL:                true,
		SupportsIteration:          true,
		SupportsCompute:            true,
		SupportsMaximum:            false,
		SupportsWeightedSize:       false,
		SupportsRefresh:            false,
		HonoursContextCancellation: true,
		AdvanceTime:                nil,
	}

	conformance.RunStringSuite(t, config)
}

func TestRedisClusterConformance(t *testing.T) {
	skipIfNoCluster(t)
	t.Parallel()

	config := conformance.StringConfig{
		ProviderFactory: func(t *testing.T, opts cache_dto.Options[string, string]) cache_domain.Cache[string, string] {
			t.Helper()
			return newRedisClusterConformanceCache(t, opts)
		},
		SupportsSearch:             false,
		SupportsTTL:                true,
		SupportsIteration:          true,
		SupportsCompute:            true,
		SupportsMaximum:            false,
		SupportsWeightedSize:       false,
		SupportsRefresh:            false,
		HonoursContextCancellation: true,
		AdvanceTime:                nil,
	}

	conformance.RunStringSuite(t, config)
}

func newRedisConformanceCache(t *testing.T, _ cache_dto.Options[string, string]) cache_domain.Cache[string, string] {
	t.Helper()

	if globalEnv == nil {
		t.Fatal("test environment not initialised")
	}

	namespace := t.Name() + ":"
	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

	config := cache_provider_redis.Config{
		Address:            globalEnv.redisAddr,
		Password:           "",
		DB:                 0,
		DefaultTTL:         1 * time.Hour,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
		Namespace:          namespace,
	}

	provider, err := cache_provider_redis.NewRedisProvider(config)
	require.NoError(t, err, "creating redis provider")

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache.Options[string, string]{})
	require.NoError(t, err, "creating redis cache")

	c, ok := cacheAny.(*cache_provider_redis.RedisAdapter[string, string])
	require.True(t, ok, "expected *cache_provider_redis.RedisAdapter[string, string], got %T", cacheAny)

	t.Cleanup(func() {
		_ = c.Close(context.Background())
		_ = provider.Close()
	})

	return c
}

func newRedisClusterConformanceCache(t *testing.T, _ cache_dto.Options[string, string]) cache_domain.Cache[string, string] {
	t.Helper()

	if globalEnv == nil {
		t.Fatal("test environment not initialised")
	}

	namespace := t.Name() + ":"
	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

	config := cache_provider_redis_cluster.Config{
		Addrs:              globalEnv.redisClusterAddrs,
		Password:           "",
		DefaultTTL:         1 * time.Hour,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
		Namespace:          namespace,
	}

	provider, err := cache_provider_redis_cluster.NewRedisClusterProvider(config)
	require.NoError(t, err, "creating redis cluster provider")

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache.Options[string, string]{})
	require.NoError(t, err, "creating redis cluster cache")

	c, ok := cacheAny.(*cache_provider_redis_cluster.RedisClusterAdapter[string, string])
	require.True(t, ok, "expected *RedisClusterAdapter[string, string], got %T", cacheAny)

	t.Cleanup(func() {
		_ = c.Close(context.Background())
		_ = provider.Close()
	})

	return c
}

func TestValkeyConformance(t *testing.T) {
	skipIfNoValkey(t)
	t.Parallel()

	config := conformance.StringConfig{
		ProviderFactory: func(t *testing.T, opts cache_dto.Options[string, string]) cache_domain.Cache[string, string] {
			t.Helper()
			return newValkeyConformanceCache(t, opts)
		},
		SupportsSearch:             false,
		SupportsTTL:                true,
		SupportsIteration:          true,
		SupportsCompute:            true,
		SupportsMaximum:            false,
		SupportsWeightedSize:       false,
		SupportsRefresh:            false,
		HonoursContextCancellation: true,
		AdvanceTime:                nil,
	}

	conformance.RunStringSuite(t, config)
}

func TestValkeyClusterConformance(t *testing.T) {
	skipIfNoValkey(t)
	skipIfNoCluster(t)
	t.Parallel()

	config := conformance.StringConfig{
		ProviderFactory: func(t *testing.T, opts cache_dto.Options[string, string]) cache_domain.Cache[string, string] {
			t.Helper()
			return newValkeyClusterConformanceCache(t, opts)
		},
		SupportsSearch:             false,
		SupportsTTL:                true,
		SupportsIteration:          true,
		SupportsCompute:            true,
		SupportsMaximum:            false,
		SupportsWeightedSize:       false,
		SupportsRefresh:            false,
		HonoursContextCancellation: true,
		AdvanceTime:                nil,
	}

	conformance.RunStringSuite(t, config)
}

func newValkeyConformanceCache(t *testing.T, _ cache_dto.Options[string, string]) cache_domain.Cache[string, string] {
	t.Helper()

	if globalEnv == nil {
		t.Fatal("test environment not initialised")
	}

	namespace := t.Name() + ":"
	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

	config := cache_provider_valkey.Config{
		Address:            globalEnv.valkeyAddr,
		DefaultTTL:         1 * time.Hour,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
		Namespace:          namespace,
	}

	provider, err := cache_provider_valkey.NewValkeyProvider(config)
	require.NoError(t, err, "creating valkey provider")

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache.Options[string, string]{})
	require.NoError(t, err, "creating valkey cache")

	c, ok := cacheAny.(*cache_provider_valkey.ValkeyAdapter[string, string])
	require.True(t, ok, "expected *ValkeyAdapter[string, string], got %T", cacheAny)

	t.Cleanup(func() {
		_ = c.Close(context.Background())
		_ = provider.Close()
	})

	return c
}

func newValkeyClusterConformanceCache(t *testing.T, _ cache_dto.Options[string, string]) cache_domain.Cache[string, string] {
	t.Helper()

	if globalEnv == nil {
		t.Fatal("test environment not initialised")
	}

	namespace := t.Name() + ":"
	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

	config := cache_provider_valkey_cluster.Config{
		InitAddress:        globalEnv.redisClusterAddrs,
		DefaultTTL:         1 * time.Hour,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
		Namespace:          namespace,
	}

	provider, err := cache_provider_valkey_cluster.NewValkeyClusterProvider(config)
	require.NoError(t, err, "creating valkey cluster provider")

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache.Options[string, string]{})
	require.NoError(t, err, "creating valkey cluster cache")

	c, ok := cacheAny.(*cache_provider_valkey_cluster.ValkeyClusterAdapter[string, string])
	require.True(t, ok, "expected *ValkeyClusterAdapter[string, string], got %T", cacheAny)

	t.Cleanup(func() {
		_ = c.Close(context.Background())
		_ = provider.Close()
	})

	return c
}
