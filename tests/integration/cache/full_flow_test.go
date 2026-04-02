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
	"bytes"
	"context"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_gob"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
	"piko.sh/piko/wdk/cache/cache_provider_redis"
	"piko.sh/piko/wdk/cache/cache_provider_redis_cluster"
)

func TestFullFlow_Standalone_AllCombinations(t *testing.T) {
	t.Parallel()

	setups := buildTransformerSetups(t)

	encoderFactories := []struct {
		name    string
		factory func(t *testing.T) cache.AnyEncoder
	}{
		{name: "json", factory: func(t *testing.T) cache.AnyEncoder {
			t.Helper()
			enc, ok := cache_encoder_json.New[string]().(cache.AnyEncoder)
			require.True(t, ok, "cache_encoder_json.New[string]() must implement cache.AnyEncoder")
			return enc
		}},
		{name: "gob", factory: func(t *testing.T) cache.AnyEncoder {
			t.Helper()
			enc, ok := cache_encoder_gob.New[string]().(cache.AnyEncoder)
			require.True(t, ok, "cache_encoder_gob.New[string]() must implement cache.AnyEncoder")
			return enc
		}},
	}

	for _, enc := range encoderFactories {
		for _, setup := range setups {
			testName := enc.name + "/" + setup.name

			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				namespace := uniqueKey(t, "fullflow") + ":"
				encoder := enc.factory(t)

				registry := cache.NewEncodingRegistry(encoder)
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
				require.NoError(t, err, "creating typed cache")

				c, ok := cacheAny.(*cache_provider_redis.RedisAdapter[string, string])
				require.True(t, ok, "expected *RedisAdapter")

				ctx := context.Background()
				t.Cleanup(func() {
					_ = c.Close(ctx)
					_ = provider.Close()
				})

				key := uniqueKey(t, "flow")
				value := "test-value-for-" + testName
				require.NoError(t, c.Set(ctx, key, value))

				got, present, err := c.GetIfPresent(ctx, key)
				require.NoError(t, err)
				require.True(t, present, "key should be present")
				assert.Equal(t, value, got, "value mismatch")

				redisKey := rawRedisKeyForNamespace(namespace, key)
				raw := getRawBytesFromRedis(t, redisKey)
				require.NotEmpty(t, raw, "raw bytes should not be empty")

				encoded, err := encoder.MarshalAny(value)
				require.NoError(t, err, "encoding value")

				wrapped := transformAndWrap(t, encoded, setup.transformers)

				recovered := reverseAndUnwrap(t, wrapped, setup.transformers)
				assert.Equal(t, encoded, recovered, "transformer round-trip on encoded data")

				decoded, err := encoder.UnmarshalAny(recovered)
				require.NoError(t, err, "decoding recovered data")
				assert.Equal(t, value, decoded, "full pipeline round-trip")
			})
		}
	}
}

func TestFullFlow_Standalone_StructValues(t *testing.T) {
	t.Parallel()

	setups := buildTransformerSetups(t)

	for _, setup := range setups {
		t.Run(setup.name, func(t *testing.T) {
			t.Parallel()

			encoder := cache_encoder_json.New[testStruct]()

			original := testStruct{
				Name: "Full Flow",
				Age:  42,
				Tags: []string{"integration", "full-flow", setup.name},
				Nested: &nestedStruct{
					Value:  "nested-val",
					Score:  95.5,
					Active: true,
				},
				Meta: map[string]string{
					"transformer": setup.name,
					"test":        "true",
				},
			}

			encoded, err := encoder.Marshal(original)
			require.NoError(t, err, "encoding struct")

			c := createRedisStringCache(t, uniqueKey(t, "fullflow-struct")+":")
			ctx := context.Background()
			key := uniqueKey(t, "struct")
			require.NoError(t, c.Set(ctx, key, string(encoded)))

			got, present, err := c.GetIfPresent(ctx, key)
			require.NoError(t, err)
			require.True(t, present, "struct key should be present")

			var decoded testStruct
			err = encoder.Unmarshal([]byte(got), &decoded)
			require.NoError(t, err, "decoding struct from cache")

			assert.Equal(t, original.Name, decoded.Name)
			assert.Equal(t, original.Age, decoded.Age)
			assert.Equal(t, original.Tags, decoded.Tags)
			require.NotNil(t, decoded.Nested)
			assert.Equal(t, original.Nested.Value, decoded.Nested.Value)
			assert.Equal(t, original.Meta, decoded.Meta)

			wrapped := transformAndWrap(t, encoded, setup.transformers)
			recovered := reverseAndUnwrap(t, wrapped, setup.transformers)

			var decoded2 testStruct
			err = encoder.Unmarshal(recovered, &decoded2)
			require.NoError(t, err, "decoding struct from transformers")

			assert.Equal(t, original.Name, decoded2.Name)
			assert.Equal(t, original.Age, decoded2.Age)
		})
	}
}

func TestFullFlow_Cluster_AllCombinations(t *testing.T) {
	skipIfNoCluster(t)
	t.Parallel()

	setups := buildTransformerSetups(t)

	for _, setup := range setups {
		t.Run(setup.name, func(t *testing.T) {
			t.Parallel()

			namespace := uniqueKey(t, "cluster-flow") + ":"
			encoder := cache_encoder_json.New[string]()

			registry := cache.NewEncodingRegistry(encoder.(cache.AnyEncoder))
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
			require.NoError(t, err, "creating typed cluster cache")

			c, ok := cacheAny.(*cache_provider_redis_cluster.RedisClusterAdapter[string, string])
			require.True(t, ok, "expected *RedisClusterAdapter")

			ctx := context.Background()
			t.Cleanup(func() {
				_ = c.Close(ctx)
				_ = provider.Close()
			})

			key := uniqueKey(t, "cluster")
			value := "cluster-test-" + setup.name
			require.NoError(t, c.Set(ctx, key, value))

			got, present, err := c.GetIfPresent(ctx, key)
			require.NoError(t, err)
			require.True(t, present, "cluster key should be present")
			assert.Equal(t, value, got, "cluster value mismatch")

			encoded, err := encoder.Marshal(value)
			require.NoError(t, err, "encoding cluster value")

			wrapped := transformAndWrap(t, encoded, setup.transformers)
			recovered := reverseAndUnwrap(t, wrapped, setup.transformers)
			assert.Equal(t, encoded, recovered, "cluster transformer round-trip")
		})
	}
}

func TestFullFlow_Cluster_StructValues(t *testing.T) {
	skipIfNoCluster(t)
	t.Parallel()

	encoder := cache_encoder_json.New[testStruct]()

	original := testStruct{
		Name: "Cluster Struct",
		Age:  55,
		Tags: []string{"cluster", "struct"},
		Nested: &nestedStruct{
			Value:  "cluster-nested",
			Score:  88.8,
			Active: true,
		},
	}

	encoded, err := encoder.Marshal(original)
	require.NoError(t, err, "encoding struct")

	namespace := uniqueKey(t, "cluster-struct") + ":"
	stringEncoder := cache_encoder_json.New[string]()

	registry := cache.NewEncodingRegistry(stringEncoder.(cache.AnyEncoder))
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
	require.NoError(t, err, "creating typed cluster string cache")

	c, ok := cacheAny.(*cache_provider_redis_cluster.RedisClusterAdapter[string, string])
	require.True(t, ok, "expected *RedisClusterAdapter[string, string]")

	ctx := context.Background()
	t.Cleanup(func() {
		_ = c.Close(ctx)
		_ = provider.Close()
	})

	key := uniqueKey(t, "cluster-struct")
	require.NoError(t, c.Set(ctx, key, string(encoded)))

	got, present, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, present, "cluster struct key should be present")

	var decoded testStruct
	err = encoder.Unmarshal([]byte(got), &decoded)
	require.NoError(t, err, "decoding cluster struct")

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Tags, decoded.Tags)
	require.NotNil(t, decoded.Nested)
	assert.Equal(t, original.Nested.Value, decoded.Nested.Value)
}

func TestFullFlow_InvalidateAll(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "fullflow-invalidateall")+":")
	ctx := context.Background()

	keys := make([]string, 10)
	for i := range 10 {
		keys[i] = uniqueKey(t, "inv-all")
		require.NoError(t, c.Set(ctx, keys[i], "value"))
	}

	for _, key := range keys {
		_, ok, err := c.GetIfPresent(ctx, key)
		require.NoError(t, err)
		require.True(t, ok, "key %q should be present before invalidateAll", key)
	}

	require.NoError(t, c.InvalidateAll(ctx))

	for _, key := range keys {
		_, ok, err := c.GetIfPresent(ctx, key)
		require.NoError(t, err)
		assert.False(t, ok, "key %q should be absent after invalidateAll", key)
	}
}

func TestFullFlow_BulkGet(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "fullflow-bulkget")+":")
	ctx := context.Background()

	items := map[string]string{
		uniqueKey(t, "bg-1"): "val-1",
		uniqueKey(t, "bg-2"): "val-2",
		uniqueKey(t, "bg-3"): "val-3",
	}

	err := c.BulkSet(ctx, items)
	require.NoError(t, err, "bulk setting values")

	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}

	result, err := c.BulkGet(ctx, keys, nil)
	require.NoError(t, err, "bulk getting values")

	for key, expected := range items {
		got, ok := result[key]
		require.True(t, ok, "key %q should be in bulk get result", key)
		assert.Equal(t, expected, got, "value mismatch for key %q", key)
	}
}

func TestFullFlow_Iterator(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "fullflow-iter")+":")
	ctx := context.Background()

	items := map[string]string{
		uniqueKey(t, "iter-1"): "val-1",
		uniqueKey(t, "iter-2"): "val-2",
		uniqueKey(t, "iter-3"): "val-3",
	}

	for k, v := range items {
		require.NoError(t, c.Set(ctx, k, v))
	}

	found := maps.Collect(c.All())

	for key, expected := range items {
		got, ok := found[key]
		require.True(t, ok, "key %q should be in iterator output", key)
		assert.Equal(t, expected, got, "value mismatch for key %q", key)
	}
}

func TestFullFlow_TransformersPlusRedis_EndToEnd(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	crypto := newCryptoSetup(t)
	transformers := []cache_domain.CacheTransformerPort{zstd, crypto.transformer}
	namespace := uniqueKey(t, "e2e") + ":"

	encoder := cache_encoder_json.New[testStruct]()

	original := testStruct{
		Name: "End to End",
		Age:  100,
		Tags: []string{"e2e", "transformers", "redis"},
		Nested: &nestedStruct{
			Value:  "deeply-nested",
			Score:  99.99,
			Active: true,
		},
		Meta: map[string]string{
			"purpose": "integration-test",
		},
	}

	encoded, err := encoder.Marshal(original)
	require.NoError(t, err, "encoding struct")

	wrapped := transformAndWrap(t, encoded, transformers)

	redisKey := rawRedisKeyForNamespace(namespace, "e2e-key")
	putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

	raw := getRawBytesFromRedis(t, redisKey)
	assert.False(t, bytes.Contains(raw, []byte("End to End")),
		"raw Redis bytes should not contain plaintext struct data")
	assert.False(t, bytes.Contains(raw, []byte("deeply-nested")),
		"raw Redis bytes should not contain nested struct data")

	recovered := reverseAndUnwrap(t, raw, transformers)

	var decoded testStruct
	err = encoder.Unmarshal(recovered, &decoded)
	require.NoError(t, err, "decoding struct")

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Tags, decoded.Tags)
	require.NotNil(t, decoded.Nested)
	assert.Equal(t, original.Nested.Value, decoded.Nested.Value)
	assert.Equal(t, original.Nested.Score, decoded.Nested.Score)
	assert.Equal(t, original.Meta, decoded.Meta)
}
