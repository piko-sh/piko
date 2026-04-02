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
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/cache_transformer_crypto"
	"piko.sh/piko/internal/cache/cache_adapters/cache_transformer_zstd"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/crypto/crypto_adapters/local_aes_gcm"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/crypto/crypto_dto"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
	"piko.sh/piko/wdk/cache/cache_provider_redis"
	"piko.sh/piko/wdk/cache/cache_provider_valkey"
)

type cryptoSetup struct {
	service     crypto_domain.CryptoServicePort
	transformer cache_domain.CacheTransformerPort
}

func newCryptoSetup(t *testing.T) *cryptoSetup {
	t.Helper()
	return newCryptoSetupWithKey(t, "integration-test-key-material!!", "test-key-1")
}

func newCryptoSetupWithKey(t *testing.T, keyMaterial, keyID string) *cryptoSetup {
	t.Helper()

	key := make([]byte, local_aes_gcm.KeySize)
	copy(key, keyMaterial)

	provider, err := local_aes_gcm.NewProvider(local_aes_gcm.Config{
		Key:   key,
		KeyID: keyID,
	})
	require.NoError(t, err, "creating crypto provider")

	service, err := crypto_domain.NewCryptoService(context.Background(), nil, &crypto_dto.ServiceConfig{
		ActiveKeyID:              keyID,
		ProviderType:             crypto_dto.ProviderTypeLocalAESGCM,
		EnableEnvelopeEncryption: false,
	})
	require.NoError(t, err, "creating crypto service")

	err = service.RegisterProvider(context.Background(), "local-aes-gcm", provider)
	require.NoError(t, err, "registering crypto provider")

	err = service.SetDefaultProvider("local-aes-gcm")
	require.NoError(t, err, "setting default crypto provider")

	transformer := cache_transformer_crypto.New(service, "crypto-service", 250)

	return &cryptoSetup{
		service:     service,
		transformer: transformer,
	}
}

func newZstdTransformer(t *testing.T) cache_domain.CacheTransformerPort {
	t.Helper()

	transformer, err := cache_transformer_zstd.NewZstdCacheTransformer(cache_transformer_zstd.DefaultConfig())
	require.NoError(t, err, "creating zstd transformer")
	return transformer
}

func createRedisStringCache(t *testing.T, namespace string) cache_domain.Cache[string, string] {
	t.Helper()
	require.NotNil(t, globalEnv, "test environment not initialised")

	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

	redisConfig := cache_provider_redis.Config{
		Address:            globalEnv.redisAddr,
		Password:           "",
		DB:                 0,
		DefaultTTL:         1 * time.Hour,
		Registry:           registry,
		AllowUnsafeFLUSHDB: true,
		Namespace:          namespace,
	}

	provider, err := cache_provider_redis.NewRedisProvider(redisConfig)
	require.NoError(t, err, "creating redis provider")

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache.Options[string, string]{})
	require.NoError(t, err, "creating redis cache namespace")

	c, ok := cacheAny.(*cache_provider_redis.RedisAdapter[string, string])
	require.True(t, ok, "expected *RedisAdapter[string, string], got %T", cacheAny)

	t.Cleanup(func() {
		_ = provider.Close()
	})

	return c
}

func createValkeyStringCache(t *testing.T, namespace string) cache_domain.Cache[string, string] {
	t.Helper()
	skipIfNoValkey(t)
	require.NotNil(t, globalEnv, "test environment not initialised")

	valueEncoder := cache_encoder_json.New[string]()
	registry := cache.NewEncodingRegistry(valueEncoder.(cache.AnyEncoder))

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
	require.NoError(t, err, "creating valkey cache namespace")

	c, ok := cacheAny.(*cache_provider_valkey.ValkeyAdapter[string, string])
	require.True(t, ok, "expected *ValkeyAdapter[string, string], got %T", cacheAny)

	t.Cleanup(func() {
		_ = provider.Close()
	})

	return c
}

func generateRepeatableText(size int) string {
	pattern := "The quick brown fox jumps over the lazy dog. "
	repeats := (size / len(pattern)) + 1
	data := strings.Repeat(pattern, repeats)
	return data[:size]
}

func generateRandomData(t *testing.T, size int) []byte {
	t.Helper()
	data := make([]byte, size)
	_, err := rand.Read(data)
	require.NoError(t, err, "generating random data")
	return data
}

func uniqueKey(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s:%s:%d", t.Name(), prefix, time.Now().UnixNano())
}

func getRawBytesFromRedis(t *testing.T, redisKey string) []byte {
	t.Helper()
	require.NotNil(t, globalEnv.rawClient, "raw redis client not initialised")

	ctx := t.Context()
	raw, err := globalEnv.rawClient.Get(ctx, redisKey).Bytes()
	require.NoError(t, err, "reading raw bytes from redis for key %q", redisKey)
	return raw
}

func putRawBytesToRedis(t *testing.T, redisKey string, data []byte, ttl time.Duration) {
	t.Helper()
	require.NotNil(t, globalEnv.rawClient, "raw redis client not initialised")

	ctx := t.Context()
	err := globalEnv.rawClient.Set(ctx, redisKey, data, ttl).Err()
	require.NoError(t, err, "writing raw bytes to redis for key %q", redisKey)
}

func rawRedisKeyForNamespace(namespace, key string) string {
	return namespace + key
}

func skipIfNoCluster(t *testing.T) {
	t.Helper()
	if len(globalEnv.redisClusterAddrs) == 0 {
		t.Skip("redis cluster not available")
	}
}

func skipIfNoValkey(t *testing.T) {
	t.Helper()
	if globalEnv.valkeyAddr == "" {
		t.Skip("valkey not available")
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	return t.Context()
}

type transformerSetup struct {
	name         string
	transformers []cache_domain.CacheTransformerPort
	enabledNames []string
}

func buildTransformerSetups(t *testing.T) []transformerSetup {
	t.Helper()

	crypto := newCryptoSetup(t)
	zstd := newZstdTransformer(t)

	return []transformerSetup{
		{
			name:         "zstd-only",
			transformers: []cache_domain.CacheTransformerPort{zstd},
			enabledNames: []string{"zstd"},
		},
		{
			name:         "crypto-only",
			transformers: []cache_domain.CacheTransformerPort{crypto.transformer},
			enabledNames: []string{"crypto-service"},
		},
		{
			name:         "zstd+crypto",
			transformers: []cache_domain.CacheTransformerPort{zstd, crypto.transformer},
			enabledNames: []string{"zstd", "crypto-service"},
		},
	}
}

func transformAndWrap(t *testing.T, data []byte, transformers []cache_domain.CacheTransformerPort) []byte {
	t.Helper()

	ctx := testContext(t)
	current := data

	for _, tr := range transformers {
		var err error
		current, err = tr.Transform(ctx, current, nil)
		require.NoError(t, err, "transforming with %s", tr.Name())
	}

	names := make([]string, 0, len(transformers))
	for _, tr := range transformers {
		names = append(names, tr.Name())
	}

	tv := cache_domain.NewTransformedValue(current, names)
	wrapped, err := tv.Marshal()
	require.NoError(t, err, "marshalling transformed value")
	return wrapped
}

func reverseAndUnwrap(t *testing.T, raw []byte, transformers []cache_domain.CacheTransformerPort) []byte {
	t.Helper()

	tv, err := cache_domain.UnmarshalTransformedValue(raw)
	require.NoError(t, err, "unmarshalling transformed value")

	ctx := testContext(t)
	current := tv.Data

	for i := len(transformers) - 1; i >= 0; i-- {
		current, err = transformers[i].Reverse(ctx, current, nil)
		require.NoError(t, err, "reversing with %s", transformers[i].Name())
	}

	return current
}
