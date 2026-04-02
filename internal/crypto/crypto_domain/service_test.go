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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func TestNewCryptoService(t *testing.T) {
	t.Parallel()

	t.Run("creates service with valid config", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")

		service, err := NewCryptoService(context.Background(), nil, config)
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "test", provider)
		require.NoError(t, err)

		err = service.SetDefaultProvider("test")
		require.NoError(t, err)

		assert.NotNil(t, service)
	})

	t.Run("applies functional options", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		factory := newMockLocalProviderFactory()
		config := createTestConfig("test-key")

		service, err := NewCryptoService(context.Background(), nil, config, WithLocalProviderFactory(factory))
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "test", provider)
		require.NoError(t, err)

		err = service.SetDefaultProvider("test")
		require.NoError(t, err)

		assert.NotNil(t, service)
	})

	t.Run("applies withDataKeyCache option", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		cache := newMockCache()
		config := createTestConfig("test-key")

		service, err := NewCryptoService(context.Background(), nil, config, withDataKeyCache(cache))
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "test", provider)
		require.NoError(t, err)

		err = service.SetDefaultProvider("test")
		require.NoError(t, err)

		assert.NotNil(t, service)
	})

	t.Run("sets deprecated key IDs from config", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("key-2")
		config.DeprecatedKeyIDs = []string{"key-1"}

		service, err := NewCryptoService(context.Background(), nil, config)
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "test", provider)
		require.NoError(t, err)

		err = service.SetDefaultProvider("test")
		require.NoError(t, err)

		keyID, err := service.GetActiveKeyID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "key-2", keyID)
	})

	t.Run("uses default concurrency when DirectModeMaxConcurrency is zero", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createDirectModeTestConfig("test-key")
		config.DirectModeMaxConcurrency = 0

		service, err := NewCryptoService(context.Background(), nil, config)
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "test", provider)
		require.NoError(t, err)

		err = service.SetDefaultProvider("test")
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		assert.Equal(t, defaultDirectModeConcurrency, cryptoService.directModeMaxConcurrency)
	})

	t.Run("uses default concurrency when DirectModeMaxConcurrency is negative", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createDirectModeTestConfig("test-key")
		config.DirectModeMaxConcurrency = -1

		service, err := NewCryptoService(context.Background(), nil, config)
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "test", provider)
		require.NoError(t, err)

		err = service.SetDefaultProvider("test")
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		assert.Equal(t, defaultDirectModeConcurrency, cryptoService.directModeMaxConcurrency)
	})
}

func TestEncrypt(t *testing.T) {
	t.Parallel()

	t.Run("encrypts plaintext successfully", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertext, err := service.Encrypt(context.Background(), "secret-data")

		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
		assert.NotEqual(t, "secret-data", ciphertext)
	})

	t.Run("uses active key ID", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		var capturedKeyID string
		provider.encryptFunc = func(_ context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error) {
			capturedKeyID = request.KeyID
			return &crypto_dto.EncryptResponse{
				Ciphertext: "encrypted",
				KeyID:      request.KeyID,
				Provider:   crypto_dto.ProviderTypeLocalAESGCM,
			}, nil
		}

		config := createTestConfig("my-active-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		_, err = service.Encrypt(context.Background(), "data")

		require.NoError(t, err)
		assert.Equal(t, "my-active-key", capturedKeyID)
	})

	t.Run("returns error when provider fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.encryptFunc = func(_ context.Context, _ *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error) {
			return nil, errors.New("provider unavailable")
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		_, err = service.Encrypt(context.Background(), "data")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Encrypt")
	})

	t.Run("encrypts empty string", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertext, err := service.Encrypt(context.Background(), "")

		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
	})
}

func TestEncryptWithKey(t *testing.T) {
	t.Parallel()

	t.Run("encrypts with specified key", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		var capturedKeyID string
		provider.encryptFunc = func(_ context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error) {
			capturedKeyID = request.KeyID
			return &crypto_dto.EncryptResponse{
				Ciphertext: "encrypted",
				KeyID:      request.KeyID,
				Provider:   crypto_dto.ProviderTypeLocalAESGCM,
			}, nil
		}

		config := createTestConfig("active-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		_, err = service.EncryptWithKey(context.Background(), "data", "specific-key")

		require.NoError(t, err)
		assert.Equal(t, "specific-key", capturedKeyID)
	})

	t.Run("wraps ciphertext in envelope", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertext, err := service.EncryptWithKey(context.Background(), "data", "test-key")
		require.NoError(t, err)

		metadata, err := extractCiphertextMetadata(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, "test-key", metadata.KeyID)
		assert.Equal(t, string(crypto_dto.ProviderTypeLocalAESGCM), metadata.Provider)
	})
}

func TestDecrypt(t *testing.T) {
	t.Parallel()

	t.Run("decrypts valid ciphertext", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertext, err := service.Encrypt(context.Background(), "secret-message")
		require.NoError(t, err)

		plaintext, err := service.Decrypt(context.Background(), ciphertext)

		require.NoError(t, err)
		assert.Equal(t, "secret-message", plaintext)
	})

	t.Run("returns error for invalid envelope format", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		_, err = service.Decrypt(context.Background(), "not-a-valid-envelope")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ciphertext format")
	})

	t.Run("returns error when provider fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return nil, errors.New("decryption failed")
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		envelope := createTestEnvelope("test-key", "local", "encrypted-data", "")

		_, err = service.Decrypt(context.Background(), envelope)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Decrypt")
	})

	t.Run("passes correct key ID to provider", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		var capturedKeyID string
		provider.decryptFunc = func(_ context.Context, request *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			capturedKeyID = request.KeyID
			return &crypto_dto.DecryptResponse{Plaintext: "decrypted"}, nil
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		envelope := createTestEnvelope("specific-key", "local", "encrypted-data", "")
		_, err = service.Decrypt(context.Background(), envelope)

		require.NoError(t, err)
		assert.Equal(t, "specific-key", capturedKeyID)
	})

	t.Run("roundtrip encrypt/decrypt preserves data", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name      string
			plaintext string
		}{
			{name: "simple string", plaintext: "hello world"},
			{name: "empty string", plaintext: ""},
			{name: "unicode", plaintext: "こんにちは世界 🌍"},
			{name: "special chars", plaintext: "!@#$%^&*()_+-={}[]|\\:\";<>?,./"},
			{name: "newlines", plaintext: "line1\nline2\r\nline3"},
			{name: "json", plaintext: `{"key": "value", "number": 123}`},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				provider := newMockProvider()
				config := createTestConfig("test-key")
				service, err := createTestService(provider, config)
				require.NoError(t, err)

				encrypted, err := service.Encrypt(context.Background(), tc.plaintext)
				require.NoError(t, err)

				decrypted, err := service.Decrypt(context.Background(), encrypted)
				require.NoError(t, err)

				assert.Equal(t, tc.plaintext, decrypted)
			})
		}
	})
}

func TestGetActiveKeyID(t *testing.T) {
	t.Parallel()

	t.Run("returns configured active key ID", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("my-active-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		keyID, err := service.GetActiveKeyID(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "my-active-key", keyID)
	})
}
