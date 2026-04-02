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

package crypto_domain

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func TestEncryptBatch(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		result, err := service.EncryptBatch(context.Background(), []string{})

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("encrypts single item batch", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		result, err := service.EncryptBatch(context.Background(), []string{"secret"})

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.NotEqual(t, "secret", result[0])
	})

	t.Run("encrypts multiple items", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		plaintexts := []string{"secret1", "secret2", "secret3"}
		result, err := service.EncryptBatch(context.Background(), plaintexts)

		require.NoError(t, err)
		require.Len(t, result, 3)

		assert.NotEqual(t, result[0], result[1])
		assert.NotEqual(t, result[1], result[2])
	})

	t.Run("generates data key once for batch", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		generateKeyCallCount := 0
		provider.generateDataKey = func(_ context.Context, _ *crypto_dto.GenerateDataKeyRequest) (*crypto_dto.DataKey, error) {
			generateKeyCallCount++
			secureKey, err := crypto_dto.NewSecureBytesFromSlice([]byte("01234567890123456789012345678901"), crypto_dto.WithID("test-datakey"))
			if err != nil {
				return nil, err
			}
			return &crypto_dto.DataKey{
				PlaintextKey: secureKey,
				EncryptedKey: "encrypted-data-key",
				KeyID:        "test-key",
				Provider:     crypto_dto.ProviderTypeLocalAESGCM,
			}, nil
		}

		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		_, err = service.EncryptBatch(context.Background(), []string{"a", "b", "c", "d", "e"})

		require.NoError(t, err)
		assert.Equal(t, 1, generateKeyCallCount, "should only generate one data key for entire batch")
	})

	t.Run("creates ephemeral provider for local encryption", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		_, err = service.EncryptBatch(context.Background(), []string{"secret"})

		require.NoError(t, err)
		assert.Equal(t, 1, factory.getCreateWithKeyCallCount())
	})

	t.Run("returns error when GenerateDataKey fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.generateDataKey = func(_ context.Context, _ *crypto_dto.GenerateDataKeyRequest) (*crypto_dto.DataKey, error) {
			return nil, errors.New("KMS unavailable")
		}

		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		_, err = service.EncryptBatch(context.Background(), []string{"secret"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate data key")
	})

	t.Run("returns error when local provider factory is nil", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		_, err = service.EncryptBatch(context.Background(), []string{"secret"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "local provider factory is required")
	})

	t.Run("returns error when factory CreateWithKey fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		factory.createWithKeyFunc = func(_ *crypto_dto.SecureBytes, _ string) (EncryptionProvider, error) {
			return nil, errors.New("failed to create provider")
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		_, err = service.EncryptBatch(context.Background(), []string{"secret"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create ephemeral encryption provider")
	})

	t.Run("includes encrypted data key in envelope", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.generateDataKey = func(_ context.Context, _ *crypto_dto.GenerateDataKeyRequest) (*crypto_dto.DataKey, error) {
			secureKey, err := crypto_dto.NewSecureBytesFromSlice([]byte("01234567890123456789012345678901"), crypto_dto.WithID("test-datakey"))
			if err != nil {
				return nil, err
			}
			return &crypto_dto.DataKey{
				PlaintextKey: secureKey,
				EncryptedKey: "my-encrypted-data-key",
				KeyID:        "test-key",
				Provider:     crypto_dto.ProviderTypeLocalAESGCM,
			}, nil
		}

		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		result, err := service.EncryptBatch(context.Background(), []string{"secret"})
		require.NoError(t, err)

		metadata, err := extractCiphertextMetadata(result[0])
		require.NoError(t, err)
		assert.Equal(t, "my-encrypted-data-key", metadata.EncryptedDataKey)
	})
}

func TestDecryptBatch(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		result, err := service.DecryptBatch(context.Background(), []string{})

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("decrypts non-envelope ciphertexts via standard decrypt", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		envelope := createTestEnvelope("test-key", "local", base64.StdEncoding.EncodeToString([]byte("secret")), "")

		result, err := service.DecryptBatch(context.Background(), []string{envelope})

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "secret", result[0])
	})

	t.Run("decrypts envelope-encrypted batch", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		plaintexts := []string{"secret1", "secret2", "secret3"}
		encrypted, err := service.EncryptBatch(context.Background(), plaintexts)
		require.NoError(t, err)

		decrypted, err := service.DecryptBatch(context.Background(), encrypted)

		require.NoError(t, err)
		assert.Equal(t, plaintexts, decrypted)
	})

	t.Run("decrypts data key only once for envelope batch", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		mainProviderDecryptCalls := 0
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			mainProviderDecryptCalls++
			return &crypto_dto.DecryptResponse{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901")),
			}, nil
		}

		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		validCiphertext := base64.StdEncoding.EncodeToString([]byte("test plaintext"))
		envelopes := make([]string, 5)
		for i := range envelopes {
			envelopes[i] = createTestEnvelope("test-key", "local", validCiphertext, "encrypted-data-key")
		}

		_, err = service.DecryptBatch(context.Background(), envelopes)

		require.NoError(t, err)
		assert.Equal(t, 1, mainProviderDecryptCalls, "should only decrypt data key once")
	})

	t.Run("returns error when data key decryption fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return nil, errors.New("KMS unavailable")
		}

		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		envelope := createTestEnvelope("test-key", "local", "ciphertext", "encrypted-data-key")
		_, err = service.DecryptBatch(context.Background(), []string{envelope})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decrypt data key")
	})

	t.Run("returns error when local provider factory is nil for envelope decryption", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return &crypto_dto.DecryptResponse{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901")),
			}, nil
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		envelope := createTestEnvelope("test-key", "local", "ciphertext", "encrypted-data-key")
		_, err = service.DecryptBatch(context.Background(), []string{envelope})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "local provider factory is required")
	})

	t.Run("roundtrip batch encrypt/decrypt preserves data", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		testCases := [][]string{
			{"single"},
			{"first", "second"},
			{"a", "b", "c", "d", "e"},
			{"unicode: 日本語", "emoji: 🎉", "special: <>&\"'"},
		}

		for _, plaintexts := range testCases {
			encrypted, err := service.EncryptBatch(context.Background(), plaintexts)
			require.NoError(t, err)

			decrypted, err := service.DecryptBatch(context.Background(), encrypted)
			require.NoError(t, err)

			assert.Equal(t, plaintexts, decrypted)
		}
	})
}

func TestDecryptDataKeyWithCache(t *testing.T) {
	t.Parallel()

	t.Run("returns cached data key on cache hit", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		decryptCallCount := 0
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			decryptCallCount++
			return &crypto_dto.DecryptResponse{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("cached-key-data")),
			}, nil
		}

		cache := newMockCache()
		cachedKey, _ := crypto_dto.NewSecureBytesFromSlice([]byte("01234567890123456789012345678901"), crypto_dto.WithID("cached-key"))
		cache.data["encrypted-key"] = cachedKey

		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory), withDataKeyCache(cache))
		require.NoError(t, err)

		validCiphertext := base64.StdEncoding.EncodeToString([]byte("test plaintext"))
		envelope := createTestEnvelope("test-key", "local", validCiphertext, "encrypted-key")
		_, err = service.DecryptBatch(context.Background(), []string{envelope})

		require.NoError(t, err)
		assert.Equal(t, 0, decryptCallCount, "should not call provider decrypt when cache hits")
		assert.Equal(t, 1, cache.getGetIfPresentCallCount())
	})

	t.Run("stores decrypted data key in cache on cache miss", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return &crypto_dto.DecryptResponse{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901")),
			}, nil
		}

		cache := newMockCache()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory), withDataKeyCache(cache))
		require.NoError(t, err)

		validCiphertext := base64.StdEncoding.EncodeToString([]byte("test plaintext"))
		envelope := createTestEnvelope("test-key", "local", validCiphertext, "encrypted-key")
		_, err = service.DecryptBatch(context.Background(), []string{envelope})

		require.NoError(t, err)
		assert.Equal(t, 1, cache.getSetCallCount(), "should store key in cache")
		require.NotNil(t, cache.data["encrypted-key"])
		var storedKey []byte
		_ = cache.data["encrypted-key"].WithAccess(func(data []byte) error {
			storedKey = make([]byte, len(data))
			copy(storedKey, data)
			return nil
		})
		assert.Equal(t, []byte("01234567890123456789012345678901"), storedKey)
	})

	t.Run("returns copy of cached data to prevent modification", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		cache := newMockCache()
		originalKey, _ := crypto_dto.NewSecureBytesFromSlice([]byte("01234567890123456789012345678901"), crypto_dto.WithID("original-key"))
		cache.data["encrypted-key"] = originalKey

		factory := newMockLocalProviderFactory()
		factory.createWithKeyFunc = func(key *crypto_dto.SecureBytes, _ string) (EncryptionProvider, error) {
			_ = key.WithAccess(func(keyBytes []byte) error {
				for i := range keyBytes {
					keyBytes[i] = 'X'
				}
				return nil
			})
			return newMockProvider(), nil
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory), withDataKeyCache(cache))
		require.NoError(t, err)

		validCiphertext := base64.StdEncoding.EncodeToString([]byte("test plaintext"))
		envelope := createTestEnvelope("test-key", "local", validCiphertext, "encrypted-key")
		_, err = service.DecryptBatch(context.Background(), []string{envelope})

		require.NoError(t, err)
		var cachedBytes []byte
		_ = cache.data["encrypted-key"].WithAccess(func(data []byte) error {
			cachedBytes = make([]byte, len(data))
			copy(cachedBytes, data)
			return nil
		})
		assert.Equal(t, []byte("01234567890123456789012345678901"), cachedBytes)
	})

	t.Run("works without cache configured", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return &crypto_dto.DecryptResponse{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901")),
			}, nil
		}

		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		validCiphertext := base64.StdEncoding.EncodeToString([]byte("test plaintext"))
		envelope := createTestEnvelope("test-key", "local", validCiphertext, "encrypted-key")
		_, err = service.DecryptBatch(context.Background(), []string{envelope})

		require.NoError(t, err)
	})
}

func TestDirectModeEncryption(t *testing.T) {
	t.Parallel()

	t.Run("encrypts batch directly without envelope", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createDirectModeTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		plaintexts := []string{"secret1", "secret2", "secret3"}
		result, err := service.EncryptBatch(context.Background(), plaintexts)

		require.NoError(t, err)
		require.Len(t, result, 3)

		assert.Equal(t, 3, provider.encryptCallCount())
		assert.Equal(t, 0, provider.generateDataKeyCallCount())
	})

	t.Run("direct mode does not require local provider factory", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createDirectModeTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		result, err := service.EncryptBatch(context.Background(), []string{"secret"})

		require.NoError(t, err)
		require.Len(t, result, 1)
	})

	t.Run("direct mode roundtrip preserves data", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createDirectModeTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		plaintexts := []string{"hello", "world", "test123"}

		ciphertexts, err := service.EncryptBatch(context.Background(), plaintexts)
		require.NoError(t, err)

		decrypted, err := service.DecryptBatch(context.Background(), ciphertexts)
		require.NoError(t, err)

		assert.Equal(t, plaintexts, decrypted)
	})

	t.Run("direct mode handles concurrent encryption", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createDirectModeTestConfig("test-key")
		config.DirectModeMaxConcurrency = 3
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		plaintexts := make([]string, 20)
		for i := range plaintexts {
			plaintexts[i] = fmt.Sprintf("secret-%d", i)
		}

		result, err := service.EncryptBatch(context.Background(), plaintexts)

		require.NoError(t, err)
		require.Len(t, result, 20)

		for i, ct := range result {
			assert.NotEmpty(t, ct, "ciphertext at index %d should not be empty", i)
		}
	})

	t.Run("direct mode propagates encryption errors", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.encryptFunc = func(_ context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error) {
			if request.Plaintext == "fail" {
				return nil, errors.New("encryption failed")
			}
			return &crypto_dto.EncryptResponse{
				Ciphertext: base64.StdEncoding.EncodeToString([]byte(request.Plaintext)),
				KeyID:      "test-key",
				Provider:   crypto_dto.ProviderTypeLocalAESGCM,
			}, nil
		}

		config := createDirectModeTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		_, err = service.EncryptBatch(context.Background(), []string{"good", "fail", "good2"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "direct batch encryption had")
	})
}

func TestDirectModeDecryption(t *testing.T) {
	t.Parallel()

	t.Run("decrypts batch directly", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createDirectModeTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertexts, err := service.EncryptBatch(context.Background(), []string{"a", "b", "c"})
		require.NoError(t, err)

		result, err := service.DecryptBatch(context.Background(), ciphertexts)

		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("direct mode handles concurrent decryption", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createDirectModeTestConfig("test-key")
		config.DirectModeMaxConcurrency = 3
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		plaintexts := make([]string, 15)
		for i := range plaintexts {
			plaintexts[i] = "data"
		}
		ciphertexts, err := service.EncryptBatch(context.Background(), plaintexts)
		require.NoError(t, err)

		result, err := service.DecryptBatch(context.Background(), ciphertexts)

		require.NoError(t, err)
		assert.Equal(t, plaintexts, result)
	})
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("direct batch encryption respects cancelled context", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.encryptFunc = func(ctx context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &crypto_dto.EncryptResponse{
					Ciphertext: "encrypted",
					KeyID:      request.KeyID,
					Provider:   crypto_dto.ProviderTypeLocalAESGCM,
				}, nil
			}
		}

		config := createDirectModeTestConfig("test-key")
		config.DirectModeMaxConcurrency = 1
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		_, err = service.EncryptBatch(ctx, []string{"a", "b", "c"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("direct batch decryption respects cancelled context", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createDirectModeTestConfig("test-key")
		config.DirectModeMaxConcurrency = 1
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertexts, err := service.EncryptBatch(context.Background(), []string{"a", "b", "c"})
		require.NoError(t, err)

		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		_, err = service.DecryptBatch(ctx, ciphertexts)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestBatchPartialFailures(t *testing.T) {
	t.Parallel()

	t.Run("envelope encryption reports partial failures", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()

		failingProvider := newMockProvider()
		failingProvider.encryptFunc = func(_ context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error) {
			if request.Plaintext == "fail" {
				return nil, errors.New("encryption failed for this item")
			}
			return &crypto_dto.EncryptResponse{
				Ciphertext: "encrypted-" + request.Plaintext,
				KeyID:      request.KeyID,
				Provider:   crypto_dto.ProviderTypeLocalAESGCM,
			}, nil
		}
		factory.shouldReturnProvider = failingProvider

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		_, err = service.EncryptBatch(context.Background(), []string{"good", "fail", "good2"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "batch encryption had")
		assert.Contains(t, err.Error(), "failures")
	})

	t.Run("envelope decryption reports partial failures", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()

		failingProvider := newMockProvider()
		var failCallCount atomic.Int32
		failingProvider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			n := failCallCount.Add(1)
			if n == 2 {
				return nil, errors.New("decryption failed")
			}
			return &crypto_dto.DecryptResponse{Plaintext: "decrypted"}, nil
		}
		factory.shouldReturnProvider = failingProvider

		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return &crypto_dto.DecryptResponse{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901")),
			}, nil
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		validCiphertext := base64.StdEncoding.EncodeToString([]byte("test"))
		envelopes := []string{
			createTestEnvelope("test-key", "local", validCiphertext, "encrypted-data-key"),
			createTestEnvelope("test-key", "local", validCiphertext, "encrypted-data-key"),
			createTestEnvelope("test-key", "local", validCiphertext, "encrypted-data-key"),
		}

		_, err = service.DecryptBatch(context.Background(), envelopes)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "batch decryption had")
		assert.Contains(t, err.Error(), "failures")
	})

	t.Run("direct batch decryption reports partial failures", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		var decryptCount atomic.Int32
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			n := decryptCount.Add(1)
			if n == 2 {
				return nil, errors.New("decryption failed")
			}
			return &crypto_dto.DecryptResponse{Plaintext: "decrypted"}, nil
		}

		config := createDirectModeTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		envelopes := []string{
			createTestEnvelope("test-key", "local", "cipher1", ""),
			createTestEnvelope("test-key", "local", "cipher2", ""),
			createTestEnvelope("test-key", "local", "cipher3", ""),
		}

		_, err = service.DecryptBatch(context.Background(), envelopes)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "direct batch decryption had")
	})
}

func TestSetupEnvelopeDecryptionErrors(t *testing.T) {
	t.Parallel()

	t.Run("returns error when CreateWithKey fails during envelope decryption", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return &crypto_dto.DecryptResponse{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901")),
			}, nil
		}

		factory := newMockLocalProviderFactory()
		factory.createWithKeyFunc = func(_ *crypto_dto.SecureBytes, _ string) (EncryptionProvider, error) {
			return nil, errors.New("failed to create ephemeral provider")
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		envelope := createTestEnvelope("test-key", "local", "ciphertext", "encrypted-data-key")
		_, err = service.DecryptBatch(context.Background(), []string{envelope})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create ephemeral provider")
	})
}

func TestDataKeyDecryption(t *testing.T) {
	t.Parallel()

	t.Run("handles raw bytes when base64 decode fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		rawKey := "!@#$%^&*()_+{}|:<>?12345678901!"
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return &crypto_dto.DecryptResponse{
				Plaintext: rawKey,
			}, nil
		}

		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		validCiphertext := base64.StdEncoding.EncodeToString([]byte("test plaintext"))
		envelope := createTestEnvelope("test-key", "local", validCiphertext, "encrypted-data-key")

		_, err = service.DecryptBatch(context.Background(), []string{envelope})
		require.NoError(t, err)
		assert.Equal(t, []byte(rawKey), factory.lastKeyUsed)
	})
}

func TestDecryptBatchItemWithEnvelopeErrors(t *testing.T) {
	t.Parallel()

	t.Run("envelope decryption handles provider decrypt failure", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return &crypto_dto.DecryptResponse{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901")),
			}, nil
		}

		factory := newMockLocalProviderFactory()
		failingProvider := newMockProvider()
		failingProvider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return nil, errors.New("ephemeral provider decryption failed")
		}
		factory.shouldReturnProvider = failingProvider

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		validCiphertext := base64.StdEncoding.EncodeToString([]byte("test"))
		envelope := createTestEnvelope("test-key", "local", validCiphertext, "encrypted-data-key")

		_, err = service.DecryptBatch(context.Background(), []string{envelope})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "batch decryption had")
	})
}
