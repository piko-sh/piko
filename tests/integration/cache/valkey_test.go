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
	"encoding/base64"
	"fmt"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_gob"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
	"piko.sh/piko/wdk/cache/cache_provider_valkey"
	"piko.sh/piko/wdk/cache/cache_provider_valkey_cluster"
)

func TestValkey_EmptyStringValue(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-empty")+":")
	ctx := context.Background()

	key := uniqueKey(t, "empty")
	require.NoError(t, c.Set(ctx, key, ""))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, "", got, "empty string should round-trip")
}

func TestValkey_SingleByteValue(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-single")+":")
	ctx := context.Background()

	key := uniqueKey(t, "single")
	require.NoError(t, c.Set(ctx, key, "x"))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, "x", got, "single byte value should round-trip")
}

func TestValkey_VeryLargeValue(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-large")+":")
	ctx := context.Background()

	largeValue := generateRepeatableText(2 * 1024 * 1024)

	key := uniqueKey(t, "large")
	require.NoError(t, c.Set(ctx, key, largeValue))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, largeValue, got, "2MB value should round-trip")
}

func TestValkey_SpecialCharactersInKeys(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-keys")+":")

	testCases := []struct {
		name string
		key  string
	}{
		{name: "spaces", key: "key with spaces"},
		{name: "unicode", key: "key-\u4F60\u597D-\U0001F600"},
		{name: "colons", key: "a:b:c:d"},
		{name: "slashes", key: "path/to/resource"},
		{name: "dots", key: "config.setting.value"},
		{name: "brackets", key: "items[0].name"},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			value := "value-for-" + tc.name
			require.NoError(t, c.Set(ctx, tc.key, value))

			got, ok, err := c.GetIfPresent(ctx, tc.key)
			require.NoError(t, err)
			require.True(t, ok, "key %q should be present", tc.key)
			assert.Equal(t, value, got, "value mismatch for key %q", tc.key)
		})
	}
}

func TestValkey_UnicodeValues(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-unicode")+":")

	testCases := []struct {
		name  string
		value string
	}{
		{name: "emoji", value: "\U0001F600\U0001F680\U0001F4A5"},
		{name: "cjk", value: "\u4F60\u597D\u4E16\u754C"},
		{name: "cyrillic", value: "\u041F\u0440\u0438\u0432\u0435\u0442 \u043C\u0438\u0440"},
		{name: "mixed-scripts", value: "Hello \u4E16\u754C \U0001F600 caf\u00E9"},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			key := uniqueKey(t, tc.name)
			require.NoError(t, c.Set(ctx, key, tc.value))

			got, ok, err := c.GetIfPresent(ctx, key)
			require.NoError(t, err)
			require.True(t, ok, "key %q should be present", key)
			assert.Equal(t, tc.value, got, "unicode value mismatch for %s", tc.name)
		})
	}
}

func TestValkey_NullBytesInValues(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-null")+":")
	ctx := context.Background()

	value := "before\x00middle\x00after"

	key := uniqueKey(t, "null-bytes")
	require.NoError(t, c.Set(ctx, key, value))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, value, got, "value with null bytes should round-trip")
}

func TestValkey_TTL_Expiry(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-ttl")+":")
	ctx := context.Background()

	key := uniqueKey(t, "ttl")
	err := c.SetWithTTL(ctx, key, "temporary", 1*time.Second)
	require.NoError(t, err, "setting value with TTL")

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present immediately after set")
	assert.Equal(t, "temporary", got)

	time.Sleep(2 * time.Second)

	_, ok, err = c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	assert.False(t, ok, "key should have expired after TTL")
}

func TestValkey_Invalidate(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-inv")+":")
	ctx := context.Background()

	key := uniqueKey(t, "inv")
	require.NoError(t, c.Set(ctx, key, "to-be-invalidated"))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present before invalidation")
	assert.Equal(t, "to-be-invalidated", got)

	require.NoError(t, c.Invalidate(ctx, key))

	_, ok, err = c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	assert.False(t, ok, "key should be absent after invalidation")
}

func TestValkey_Overwrite(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-overwrite")+":")
	ctx := context.Background()

	key := uniqueKey(t, "overwrite")
	require.NoError(t, c.Set(ctx, key, "original"))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, "original", got)

	require.NoError(t, c.Set(ctx, key, "updated"))

	got, ok, err = c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should still be present")
	assert.Equal(t, "updated", got, "value should be updated after overwrite")
}

func TestValkey_Compute(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-compute")+":")
	ctx := context.Background()

	key := uniqueKey(t, "compute")

	got, computed, err := c.ComputeIfAbsent(ctx, key, func() string {
		return "computed-value"
	})
	require.NoError(t, err)
	assert.Equal(t, "computed-value", got)
	assert.True(t, computed, "computation should have occurred for absent key")

	got, computed, err = c.ComputeIfAbsent(ctx, key, func() string {
		return "should-not-be-used"
	})
	require.NoError(t, err)
	assert.Equal(t, "computed-value", got)
	assert.False(t, computed, "computation should not occur for present key")
}

func TestValkey_BulkSet(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-edge-bulk")+":")
	ctx := context.Background()

	items := map[string]string{
		uniqueKey(t, "bulk-1"): "value-1",
		uniqueKey(t, "bulk-2"): "value-2",
		uniqueKey(t, "bulk-3"): "value-3",
	}

	err := c.BulkSet(ctx, items)
	require.NoError(t, err, "bulk setting values")

	for key, expected := range items {
		got, ok, err := c.GetIfPresent(ctx, key)
		require.NoError(t, err)
		require.True(t, ok, "key %q should be present", key)
		assert.Equal(t, expected, got, "value mismatch for key %q", key)
	}
}

func TestValkey_Encoder_JSON_RoundTrip_Struct(t *testing.T) {
	t.Parallel()
	skipIfNoValkey(t)

	c := createValkeyStringCache(t, uniqueKey(t, "vk-enc-json-struct")+":")
	encoder := cache_encoder_json.New[testStruct]()

	original := testStruct{
		Name: "Valkey JSON",
		Age:  30,
		Tags: []string{"valkey", "json"},
		Nested: &nestedStruct{
			Value:  "important",
			Score:  99.5,
			Active: true,
		},
		Meta: map[string]string{"source": "valkey"},
	}

	encoded, err := encoder.Marshal(original)
	require.NoError(t, err, "encoding struct")

	ctx := context.Background()
	key := uniqueKey(t, "struct")
	require.NoError(t, c.Set(ctx, key, string(encoded)))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")

	var decoded testStruct
	err = encoder.Unmarshal([]byte(got), &decoded)
	require.NoError(t, err, "decoding struct")

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Tags, decoded.Tags)
	require.NotNil(t, decoded.Nested)
	assert.Equal(t, original.Nested.Value, decoded.Nested.Value)
	assert.Equal(t, original.Nested.Score, decoded.Nested.Score)
	assert.Equal(t, original.Meta, decoded.Meta)
}

func TestValkey_Encoder_Gob_RoundTrip_Struct(t *testing.T) {
	t.Parallel()
	skipIfNoValkey(t)

	c := createValkeyStringCache(t, uniqueKey(t, "vk-enc-gob-struct")+":")
	encoder := cache_encoder_gob.New[testStruct]()

	original := testStruct{
		Name: "Valkey Gob",
		Age:  25,
		Tags: []string{"valkey", "gob"},
		Nested: &nestedStruct{
			Value:  "secondary",
			Score:  42.0,
			Active: false,
		},
	}

	encoded, err := encoder.Marshal(original)
	require.NoError(t, err, "gob-encoding struct")

	ctx := context.Background()
	key := uniqueKey(t, "struct")
	require.NoError(t, c.Set(ctx, key, base64.StdEncoding.EncodeToString(encoded)))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")

	decodedBytes, err := base64.StdEncoding.DecodeString(got)
	require.NoError(t, err, "base64-decoding gob data")

	var decoded testStruct
	err = encoder.Unmarshal(decodedBytes, &decoded)
	require.NoError(t, err, "gob-decoding struct")

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Tags, decoded.Tags)
	require.NotNil(t, decoded.Nested)
	assert.Equal(t, original.Nested.Value, decoded.Nested.Value)
}

func TestValkey_Concurrency_ParallelSetGet(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-conc-setget")+":")
	ctx := context.Background()

	const goroutines = 10
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()

			for i := range opsPerGoroutine {
				key := fmt.Sprintf("g%d-k%d", id, i)
				value := fmt.Sprintf("v%d-%d", id, i)

				assert.NoError(t, c.Set(ctx, key, value))

				got, ok, err := c.GetIfPresent(ctx, key)
				assert.NoError(t, err)
				assert.True(t, ok, "key %q should be present", key)
				assert.Equal(t, value, got, "value mismatch for key %q", key)
			}
		}(g)
	}

	wg.Wait()
}

func TestValkey_Concurrency_HotKey(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-conc-hotkey")+":")
	ctx := context.Background()

	const goroutines = 50
	hotKey := uniqueKey(t, "hot")

	require.NoError(t, c.Set(ctx, hotKey, "initial-value"))

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make(chan error, goroutines)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()

			value := fmt.Sprintf("writer-%d", id)
			if err := c.Set(ctx, hotKey, value); err != nil {
				errors <- fmt.Errorf("goroutine %d: set failed: %w", id, err)
				return
			}

			got, ok, err := c.GetIfPresent(ctx, hotKey)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: get failed: %w", id, err)
				return
			}
			if !ok {
				errors <- fmt.Errorf("goroutine %d: key not found after set", id)
				return
			}
			if len(got) == 0 {
				errors <- fmt.Errorf("goroutine %d: got empty value", id)
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestValkey_Concurrency_Compute_Contention(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-conc-compute")+":")
	ctx := context.Background()

	const goroutines = 20
	key := uniqueKey(t, "compute-contention")

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			_, _, _ = c.ComputeIfAbsent(ctx, key, func() string {
				return fmt.Sprintf("writer-%d", id)
			})
		}(g)
	}

	wg.Wait()

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present after compute contention")
	assert.NotEmpty(t, got, "value should not be empty")
}

func TestValkey_FullFlow_Standalone(t *testing.T) {
	skipIfNoValkey(t)
	t.Parallel()

	namespace := uniqueKey(t, "vk-fullflow") + ":"
	encoder := cache_encoder_json.New[string]()

	registry := cache.NewEncodingRegistry(encoder.(cache.AnyEncoder))
	valkeyConfig := cache_provider_valkey.Config{
		Address:            globalEnv.valkeyAddr,
		DefaultTTL:         1 * time.Hour,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
		Namespace:          namespace,
	}

	provider, err := cache_provider_valkey.NewValkeyProvider(valkeyConfig)
	require.NoError(t, err, "creating valkey provider")

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache.Options[string, string]{})
	require.NoError(t, err, "creating typed valkey cache")

	c, ok := cacheAny.(*cache_provider_valkey.ValkeyAdapter[string, string])
	require.True(t, ok, "expected *ValkeyAdapter")

	ctx := context.Background()
	t.Cleanup(func() {
		_ = c.Close(ctx)
		_ = provider.Close()
	})

	key := uniqueKey(t, "flow")
	value := "valkey-full-flow-value"
	require.NoError(t, c.Set(ctx, key, value))

	got, present, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, present, "key should be present")
	assert.Equal(t, value, got, "value mismatch")

	require.NoError(t, c.Invalidate(ctx, key))
	_, present, err = c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	assert.False(t, present, "key should be absent after invalidation")
}

func TestValkey_FullFlow_Standalone_StructValues(t *testing.T) {
	skipIfNoValkey(t)
	t.Parallel()

	encoder := cache_encoder_json.New[testStruct]()
	c := createValkeyStringCache(t, uniqueKey(t, "vk-fullflow-struct")+":")

	original := testStruct{
		Name: "Valkey Full Flow",
		Age:  42,
		Tags: []string{"valkey", "full-flow"},
		Nested: &nestedStruct{
			Value:  "nested-val",
			Score:  95.5,
			Active: true,
		},
		Meta: map[string]string{"purpose": "integration-test"},
	}

	encoded, err := encoder.Marshal(original)
	require.NoError(t, err, "encoding struct")

	ctx := context.Background()
	key := uniqueKey(t, "struct")
	require.NoError(t, c.Set(ctx, key, string(encoded)))

	got, present, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, present, "struct key should be present")

	var decoded testStruct
	err = encoder.Unmarshal([]byte(got), &decoded)
	require.NoError(t, err, "decoding struct")

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Tags, decoded.Tags)
	require.NotNil(t, decoded.Nested)
	assert.Equal(t, original.Nested.Value, decoded.Nested.Value)
	assert.Equal(t, original.Meta, decoded.Meta)
}

func TestValkey_FullFlow_InvalidateAll(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-fullflow-invall")+":")
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

func TestValkey_FullFlow_BulkGet(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-fullflow-bulkget")+":")
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

func TestValkey_FullFlow_Iterator(t *testing.T) {
	t.Parallel()

	c := createValkeyStringCache(t, uniqueKey(t, "vk-fullflow-iter")+":")
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

func TestValkeyCluster_SetGet(t *testing.T) {
	skipIfNoValkey(t)
	skipIfNoCluster(t)
	t.Parallel()

	namespace := uniqueKey(t, "vkc-setget") + ":"
	encoder := cache_encoder_json.New[string]()

	registry := cache.NewEncodingRegistry(encoder.(cache.AnyEncoder))
	clusterConfig := cache_provider_valkey_cluster.Config{
		InitAddress:        globalEnv.redisClusterAddrs,
		DefaultTTL:         1 * time.Hour,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
		Namespace:          namespace,
	}

	provider, err := cache_provider_valkey_cluster.NewValkeyClusterProvider(clusterConfig)
	require.NoError(t, err, "creating valkey cluster provider")

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache.Options[string, string]{})
	require.NoError(t, err, "creating typed valkey cluster cache")

	c, ok := cacheAny.(*cache_provider_valkey_cluster.ValkeyClusterAdapter[string, string])
	require.True(t, ok, "expected *ValkeyClusterAdapter")

	ctx := context.Background()
	t.Cleanup(func() {
		_ = c.Close(ctx)
		_ = provider.Close()
	})

	key := uniqueKey(t, "cluster")
	value := "valkey-cluster-value"
	require.NoError(t, c.Set(ctx, key, value))

	got, present, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, present, "cluster key should be present")
	assert.Equal(t, value, got, "cluster value mismatch")
}

func TestValkeyCluster_StructValues(t *testing.T) {
	skipIfNoValkey(t)
	skipIfNoCluster(t)
	t.Parallel()

	encoder := cache_encoder_json.New[testStruct]()
	stringEncoder := cache_encoder_json.New[string]()

	namespace := uniqueKey(t, "vkc-struct") + ":"
	registry := cache.NewEncodingRegistry(stringEncoder.(cache.AnyEncoder))
	clusterConfig := cache_provider_valkey_cluster.Config{
		InitAddress:        globalEnv.redisClusterAddrs,
		DefaultTTL:         1 * time.Hour,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
		Namespace:          namespace,
	}

	provider, err := cache_provider_valkey_cluster.NewValkeyClusterProvider(clusterConfig)
	require.NoError(t, err, "creating valkey cluster provider")

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache.Options[string, string]{})
	require.NoError(t, err, "creating typed valkey cluster string cache")

	c, ok := cacheAny.(*cache_provider_valkey_cluster.ValkeyClusterAdapter[string, string])
	require.True(t, ok, "expected *ValkeyClusterAdapter[string, string]")

	ctx := context.Background()
	t.Cleanup(func() {
		_ = c.Close(ctx)
		_ = provider.Close()
	})

	original := testStruct{
		Name: "Valkey Cluster Struct",
		Age:  55,
		Tags: []string{"valkey", "cluster"},
		Nested: &nestedStruct{
			Value:  "cluster-nested",
			Score:  88.8,
			Active: true,
		},
	}

	encoded, err := encoder.Marshal(original)
	require.NoError(t, err, "encoding struct")

	key := uniqueKey(t, "cluster-struct")
	require.NoError(t, c.Set(ctx, key, string(encoded)))

	got, present, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, present, "cluster struct key should be present")

	var decoded testStruct
	err = encoder.Unmarshal([]byte(got), &decoded)
	require.NoError(t, err, "decoding struct")

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Tags, decoded.Tags)
	require.NotNil(t, decoded.Nested)
	assert.Equal(t, original.Nested.Value, decoded.Nested.Value)
}

func TestValkeyCluster_Concurrency(t *testing.T) {
	skipIfNoValkey(t)
	skipIfNoCluster(t)
	t.Parallel()

	namespace := uniqueKey(t, "vkc-conc") + ":"
	encoder := cache_encoder_json.New[string]()

	registry := cache.NewEncodingRegistry(encoder.(cache.AnyEncoder))
	clusterConfig := cache_provider_valkey_cluster.Config{
		InitAddress:        globalEnv.redisClusterAddrs,
		DefaultTTL:         1 * time.Hour,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
		Namespace:          namespace,
	}

	provider, err := cache_provider_valkey_cluster.NewValkeyClusterProvider(clusterConfig)
	require.NoError(t, err, "creating valkey cluster provider")

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache.Options[string, string]{})
	require.NoError(t, err, "creating typed valkey cluster cache")

	c, ok := cacheAny.(*cache_provider_valkey_cluster.ValkeyClusterAdapter[string, string])
	require.True(t, ok, "expected *ValkeyClusterAdapter")

	ctx := context.Background()
	t.Cleanup(func() {
		_ = c.Close(ctx)
		_ = provider.Close()
	})

	const goroutines = 10
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make(chan error, goroutines*opsPerGoroutine)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()

			for i := range opsPerGoroutine {
				key := fmt.Sprintf("g%d-k%d", id, i)
				value := fmt.Sprintf("v%d-%d", id, i)

				if err := c.Set(ctx, key, value); err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: set failed: %w", id, i, err)
					continue
				}

				got, ok, err := c.GetIfPresent(ctx, key)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: get failed: %w", id, i, err)
					continue
				}
				if !ok {
					errors <- fmt.Errorf("goroutine %d, op %d: key not found", id, i)
					continue
				}
				if got != value {
					errors <- fmt.Errorf("goroutine %d, op %d: value mismatch: got %q, want %q", id, i, got, value)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
