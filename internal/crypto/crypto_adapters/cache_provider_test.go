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

package crypto_adapters

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func TestFactoryBlueprintRegistration(t *testing.T) {
	t.Parallel()

	t.Run("crypto-secure-bytes factory is registered", func(t *testing.T) {
		t.Parallel()

		factory, exists := cache_domain.GetProviderFactory("crypto-secure-bytes")
		assert.True(t, exists, "factory blueprint 'crypto-secure-bytes' should be registered")
		assert.NotNil(t, factory, "factory blueprint should not be nil")
	})

	t.Run("factory returns error for invalid options type", func(t *testing.T) {
		t.Parallel()

		factory, exists := cache_domain.GetProviderFactory("crypto-secure-bytes")
		require.True(t, exists)
		require.NotNil(t, factory)

		_, err := factory(nil, "", "invalid options type")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid options type")
	})
}

func TestCacheCreationViaFactoryBlueprint(t *testing.T) {
	t.Parallel()

	t.Run("creates cache with factory blueprint", func(t *testing.T) {
		t.Parallel()

		service := cache_domain.NewService("")

		cache, err := cache_domain.NewCacheBuilder[string, *crypto_dto.SecureBytes](service).
			FactoryBlueprint("crypto-secure-bytes").
			Namespace("test-secure-bytes").
			MaximumSize(100).
			Build(context.Background())

		require.NoError(t, err)
		require.NotNil(t, cache)
		defer func() { _ = cache.Close(context.Background()) }()
	})

	t.Run("created cache implements expected interface", func(t *testing.T) {
		t.Parallel()

		service := cache_domain.NewService("")

		cache, err := cache_domain.NewCacheBuilder[string, *crypto_dto.SecureBytes](service).
			FactoryBlueprint("crypto-secure-bytes").
			Namespace("test-interface").
			MaximumSize(100).
			Build(context.Background())

		require.NoError(t, err)
		defer func() { _ = cache.Close(context.Background()) }()

		var _ cache_domain.Cache[string, *crypto_dto.SecureBytes] = cache
	})
}

func TestSecureBytesCacheOperations(t *testing.T) {
	t.Parallel()

	t.Run("stores and retrieves SecureBytes", func(t *testing.T) {
		t.Parallel()

		cache := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		testData := []byte("test-encryption-key-data-32bytes!")
		secureBytes, err := crypto_dto.NewSecureBytesFromSlice(testData, crypto_dto.WithID("test-key-1"))
		require.NoError(t, err)
		defer func() { _ = secureBytes.Close() }()

		_ = cache.Set(context.Background(), "key1", secureBytes)

		retrieved, ok, _ := cache.GetIfPresent(context.Background(), "key1")
		assert.True(t, ok, "expected cache hit")
		assert.NotNil(t, retrieved)
		assert.Equal(t, secureBytes.Len(), retrieved.Len())
	})

	t.Run("returns nil for cache miss", func(t *testing.T) {
		t.Parallel()

		cache := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		retrieved, ok, _ := cache.GetIfPresent(context.Background(), "nonexistent-key")
		assert.False(t, ok, "expected cache miss")
		assert.Nil(t, retrieved)
	})

	t.Run("invalidates cached entry", func(t *testing.T) {
		t.Parallel()

		cache := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		testData := []byte("test-encryption-key-data-32bytes!")
		secureBytes, err := crypto_dto.NewSecureBytesFromSlice(testData, crypto_dto.WithID("test-key-invalidate"))
		require.NoError(t, err)
		defer func() { _ = secureBytes.Close() }()

		_ = cache.Set(context.Background(), "key-to-invalidate", secureBytes)

		_, ok, _ := cache.GetIfPresent(context.Background(), "key-to-invalidate")
		require.True(t, ok)

		_ = cache.Invalidate(context.Background(), "key-to-invalidate")

		_, ok, _ = cache.GetIfPresent(context.Background(), "key-to-invalidate")
		assert.False(t, ok, "expected cache miss after invalidation")
	})

	t.Run("overwrites existing entry", func(t *testing.T) {
		t.Parallel()

		cache := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		data1 := []byte("first-key-data-32bytes-long!!!")
		secureBytes1, err := crypto_dto.NewSecureBytesFromSlice(data1, crypto_dto.WithID("first"))
		require.NoError(t, err)
		defer func() { _ = secureBytes1.Close() }()

		data2 := []byte("second-key-data-32bytes-long!!")
		secureBytes2, err := crypto_dto.NewSecureBytesFromSlice(data2, crypto_dto.WithID("second"))
		require.NoError(t, err)
		defer func() { _ = secureBytes2.Close() }()

		_ = cache.Set(context.Background(), "overwrite-key", secureBytes1)
		_ = cache.Set(context.Background(), "overwrite-key", secureBytes2)

		retrieved, ok, _ := cache.GetIfPresent(context.Background(), "overwrite-key")
		require.True(t, ok)
		assert.Equal(t, "second", retrieved.ID())
	})
}

func TestOnDeletionCallback(t *testing.T) {
	t.Parallel()

	t.Run("OnDeletion callback is invoked on invalidation", func(t *testing.T) {
		t.Parallel()

		service := cache_domain.NewService("")

		callbackDone := make(chan struct{})
		var deletedKey string
		var deletedValue *crypto_dto.SecureBytes

		cache, err := cache_domain.NewCacheBuilder[string, *crypto_dto.SecureBytes](service).
			FactoryBlueprint("crypto-secure-bytes").
			Namespace("test-ondeletion").
			MaximumSize(100).
			OnDeletion(func(e cache_dto.DeletionEvent[string, *crypto_dto.SecureBytes]) {
				deletedKey = e.Key
				deletedValue = e.Value
				close(callbackDone)
			}).
			Build(context.Background())

		require.NoError(t, err)
		defer func() { _ = cache.Close(context.Background()) }()

		testData := []byte("test-key-for-deletion-callback!")
		secureBytes, err := crypto_dto.NewSecureBytesFromSlice(testData, crypto_dto.WithID("callback-test"))
		require.NoError(t, err)

		_ = cache.Set(context.Background(), "callback-key", secureBytes)

		_ = cache.Invalidate(context.Background(), "callback-key")

		select {
		case <-callbackDone:
		case <-time.After(time.Second):
			t.Fatal("OnDeletion callback was not invoked within timeout")
		}

		assert.Equal(t, "callback-key", deletedKey)
		assert.NotNil(t, deletedValue)
	})

	t.Run("SecureBytes.Close() can be called safely in OnDeletion", func(t *testing.T) {
		t.Parallel()

		service := cache_domain.NewService("")

		callbackDone := make(chan struct{})

		cache, err := cache_domain.NewCacheBuilder[string, *crypto_dto.SecureBytes](service).
			FactoryBlueprint("crypto-secure-bytes").
			Namespace("test-close-ondeletion").
			MaximumSize(100).
			OnDeletion(func(e cache_dto.DeletionEvent[string, *crypto_dto.SecureBytes]) {
				if e.Value != nil {
					_ = e.Value.Close()
				}
				close(callbackDone)
			}).
			Build(context.Background())

		require.NoError(t, err)
		defer func() { _ = cache.Close(context.Background()) }()

		testData := []byte("test-key-for-close-in-callback!")
		secureBytes, err := crypto_dto.NewSecureBytesFromSlice(testData, crypto_dto.WithID("close-test"))
		require.NoError(t, err)

		_ = cache.Set(context.Background(), "close-key", secureBytes)

		_ = cache.Invalidate(context.Background(), "close-key")

		select {
		case <-callbackDone:
		case <-time.After(time.Second):
			t.Fatal("OnDeletion callback was not invoked within timeout")
		}

		assert.True(t, secureBytes.IsClosed(), "SecureBytes should be closed after eviction")
	})
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles nil SecureBytes gracefully", func(t *testing.T) {
		t.Parallel()

		cache := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		_ = cache.Set(context.Background(), "nil-key", nil)

		retrieved, ok, _ := cache.GetIfPresent(context.Background(), "nil-key")
		assert.True(t, ok, "expected cache hit even for nil value")
		assert.Nil(t, retrieved)
	})

	t.Run("multiple stores and retrieves work correctly", func(t *testing.T) {
		t.Parallel()

		cache := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		keys := make([]*crypto_dto.SecureBytes, 10)
		for i := range 10 {
			data := []byte("test-key-data-for-multiple-test!")
			secureBytes, err := crypto_dto.NewSecureBytesFromSlice(data, crypto_dto.WithID("multi-"+string(rune('0'+i))))
			require.NoError(t, err)
			keys[i] = secureBytes

			_ = cache.Set(context.Background(), "multi-key-"+string(rune('0'+i)), secureBytes)
			err = secureBytes.Close()
			if err != nil {
				assert.FailNow(t, "failed to close secure bytes: %v", err)
			}
		}

		for i := range 10 {
			retrieved, ok, _ := cache.GetIfPresent(context.Background(), "multi-key-"+string(rune('0'+i)))
			assert.True(t, ok, "expected cache hit for key %d", i)
			assert.NotNil(t, retrieved)
		}
	})
}

func setupTestCache(t *testing.T) cache_domain.Cache[string, *crypto_dto.SecureBytes] {
	t.Helper()

	service := cache_domain.NewService("")

	cache, err := cache_domain.NewCacheBuilder[string, *crypto_dto.SecureBytes](service).
		FactoryBlueprint("crypto-secure-bytes").
		Namespace("test-" + t.Name()).
		MaximumSize(1000).
		Build(context.Background())

	require.NoError(t, err)
	return cache
}
