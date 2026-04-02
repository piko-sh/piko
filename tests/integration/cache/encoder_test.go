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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_gob"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
	"piko.sh/piko/wdk/cache/cache_provider_redis"
)

type testStruct struct {
	Name   string            `json:"name"`
	Age    int               `json:"age"`
	Tags   []string          `json:"tags"`
	Nested *nestedStruct     `json:"nested,omitempty"`
	Meta   map[string]string `json:"meta,omitempty"`
}

type nestedStruct struct {
	Value  string  `json:"value"`
	Score  float64 `json:"score"`
	Active bool    `json:"active"`
}

func TestEncoder_JSON_RoundTrip_String(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "enc-json-str")+":")

	testCases := []struct {
		name  string
		value string
	}{
		{name: "simple", value: "hello world"},
		{name: "empty", value: ""},
		{name: "long", value: generateRepeatableText(4096)},
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
			assert.Equal(t, tc.value, got, "value mismatch for %s", tc.name)
		})
	}
}

func TestEncoder_JSON_RoundTrip_Struct(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "enc-json-struct")+":")
	encoder := cache_encoder_json.New[testStruct]()

	original := testStruct{
		Name: "Alice",
		Age:  30,
		Tags: []string{"admin", "developer"},
		Nested: &nestedStruct{
			Value:  "important",
			Score:  99.5,
			Active: true,
		},
		Meta: map[string]string{
			"department": "engineering",
			"role":       "lead",
		},
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
	assert.Equal(t, original.Nested.Active, decoded.Nested.Active)
	assert.Equal(t, original.Meta, decoded.Meta)
}

func TestEncoder_JSON_Unicode(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "enc-json-unicode")+":")

	testCases := []struct {
		name  string
		value string
	}{
		{name: "emoji", value: "\U0001F600\U0001F680\U0001F4A5"},
		{name: "cjk", value: "\u4F60\u597D\u4E16\u754C"},
		{name: "cyrillic", value: "\u041F\u0440\u0438\u0432\u0435\u0442 \u043C\u0438\u0440"},
		{name: "accented", value: "caf\u00E9 na\u00EFve r\u00E9sum\u00E9"},
		{name: "mixed", value: "Hello \u4E16\u754C \U0001F600 caf\u00E9"},
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

func TestEncoder_Gob_RoundTrip_String(t *testing.T) {
	t.Parallel()

	c := createGobStringCache(t, uniqueKey(t, "enc-gob-str")+":")

	testCases := []struct {
		name  string
		value string
	}{
		{name: "simple", value: "hello gob world"},
		{name: "empty", value: ""},
		{name: "long", value: generateRepeatableText(4096)},
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
			assert.Equal(t, tc.value, got, "value mismatch for %s", tc.name)
		})
	}
}

func TestEncoder_Gob_RoundTrip_Struct(t *testing.T) {
	t.Parallel()

	c := createGobStringCache(t, uniqueKey(t, "enc-gob-struct")+":")
	encoder := cache_encoder_gob.New[testStruct]()

	original := testStruct{
		Name: "Bob",
		Age:  25,
		Tags: []string{"viewer", "tester"},
		Nested: &nestedStruct{
			Value:  "secondary",
			Score:  42.0,
			Active: false,
		},
		Meta: map[string]string{"team": "qa"},
	}

	encoded, err := encoder.Marshal(original)
	require.NoError(t, err, "gob-encoding struct")

	ctx := context.Background()
	key := uniqueKey(t, "struct")
	require.NoError(t, c.Set(ctx, key, string(encoded)))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")

	var decoded testStruct
	err = encoder.Unmarshal([]byte(got), &decoded)
	require.NoError(t, err, "gob-decoding struct")

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Tags, decoded.Tags)
	require.NotNil(t, decoded.Nested)
	assert.Equal(t, original.Nested.Value, decoded.Nested.Value)
	assert.Equal(t, original.Meta, decoded.Meta)
}

func TestEncoder_Pipeline_JSON_Zstd_Crypto(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	crypto := newCryptoSetup(t)
	transformers := []cache_domain.CacheTransformerPort{zstd, crypto.transformer}

	encoder := cache_encoder_json.New[testStruct]()

	original := testStruct{
		Name: "Pipeline Test",
		Age:  42,
		Tags: []string{"full-flow", "integration"},
		Nested: &nestedStruct{
			Value:  "deep",
			Score:  100.0,
			Active: true,
		},
	}

	encoded, err := encoder.Marshal(original)
	require.NoError(t, err, "encoding struct")

	wrapped := transformAndWrap(t, encoded, transformers)

	namespace := uniqueKey(t, "enc-pipeline-json") + ":"
	redisKey := rawRedisKeyForNamespace(namespace, "pipeline-key")
	putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

	raw := getRawBytesFromRedis(t, redisKey)
	recovered := reverseAndUnwrap(t, raw, transformers)

	var decoded testStruct
	err = encoder.Unmarshal(recovered, &decoded)
	require.NoError(t, err, "decoding struct")

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Tags, decoded.Tags)
	require.NotNil(t, decoded.Nested)
	assert.Equal(t, original.Nested.Value, decoded.Nested.Value)
}

func TestEncoder_Pipeline_Gob_Zstd(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	transformers := []cache_domain.CacheTransformerPort{zstd}

	encoder := cache_encoder_gob.New[testStruct]()

	original := testStruct{
		Name: "Gob Pipeline",
		Age:  33,
		Tags: []string{"gob", "zstd"},
	}

	encoded, err := encoder.Marshal(original)
	require.NoError(t, err, "gob-encoding struct")

	wrapped := transformAndWrap(t, encoded, transformers)

	namespace := uniqueKey(t, "enc-pipeline-gob") + ":"
	redisKey := rawRedisKeyForNamespace(namespace, "pipeline-key")
	putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

	raw := getRawBytesFromRedis(t, redisKey)
	recovered := reverseAndUnwrap(t, raw, transformers)

	var decoded testStruct
	err = encoder.Unmarshal(recovered, &decoded)
	require.NoError(t, err, "gob-decoding struct")

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Tags, decoded.Tags)
}

func createGobStringCache(t *testing.T, namespace string) cache_domain.Cache[string, string] {
	t.Helper()
	require.NotNil(t, globalEnv, "test environment not initialised")

	valueEncoder := cache_encoder_gob.New[string]()
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
	require.NoError(t, err, "creating redis gob cache")

	c, ok := cacheAny.(*cache_provider_redis.RedisAdapter[string, string])
	require.True(t, ok, "expected *RedisAdapter[string, string], got %T", cacheAny)

	t.Cleanup(func() {
		_ = c.Close(context.Background())
		_ = provider.Close()
	})

	return c
}
