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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func TestRotateKey(t *testing.T) {
	t.Parallel()

	t.Run("successfully rotates key", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("old-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "old-key", "new-key")

		require.NoError(t, err)
		keyID, err := service.GetActiveKeyID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "new-key", keyID)
	})

	t.Run("marks old key as deprecated after rotation", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("old-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "old-key", "new-key")
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		assert.True(t, cryptoService.isDeprecatedKey("old-key"))
	})

	t.Run("returns error for empty old key ID", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("active-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "", "new-key")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be non-empty")
	})

	t.Run("returns error for empty new key ID", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("active-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "active-key", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be non-empty")
	})

	t.Run("returns error when old and new key IDs are the same", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("same-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "same-key", "same-key")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be the same")
	})

	t.Run("returns error when old key is not the active key", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("active-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "wrong-key", "new-key")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not the active key")
	})

	t.Run("returns error when old key is already deprecated", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("key-2")
		config.DeprecatedKeyIDs = []string{"key-1"}
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "key-2", "key-3")
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "key-2", "key-4")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not the active key")
	})

	t.Run("supports multiple consecutive rotations", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("key-1")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "key-1", "key-2")
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "key-2", "key-3")
		require.NoError(t, err)

		keyID, err := service.GetActiveKeyID(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "key-3", keyID)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		assert.True(t, cryptoService.isDeprecatedKey("key-1"))
		assert.True(t, cryptoService.isDeprecatedKey("key-2"))
		assert.False(t, cryptoService.isDeprecatedKey("key-3"))
	})
}

func TestDecryptAndReEncrypt(t *testing.T) {
	t.Parallel()

	t.Run("returns plaintext without re-encryption for active key", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("active-key")
		config.EnableAutoReEncrypt = true
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertext, err := service.Encrypt(context.Background(), "secret")
		require.NoError(t, err)

		plaintext, newCiphertext, wasReEncrypted, err := service.DecryptAndReEncrypt(context.Background(), ciphertext)

		require.NoError(t, err)
		assert.Equal(t, "secret", plaintext)
		assert.Empty(t, newCiphertext)
		assert.False(t, wasReEncrypted)
	})

	t.Run("re-encrypts data encrypted with deprecated key when auto re-encrypt enabled", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("old-key")
		config.EnableAutoReEncrypt = true
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertext, err := service.Encrypt(context.Background(), "secret")
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "old-key", "new-key")
		require.NoError(t, err)

		plaintext, newCiphertext, wasReEncrypted, err := service.DecryptAndReEncrypt(context.Background(), ciphertext)

		require.NoError(t, err)
		assert.Equal(t, "secret", plaintext)
		assert.NotEmpty(t, newCiphertext)
		assert.True(t, wasReEncrypted)

		metadata, err := extractCiphertextMetadata(newCiphertext)
		require.NoError(t, err)
		assert.Equal(t, "new-key", metadata.KeyID)
	})

	t.Run("does not re-encrypt when auto re-encrypt is disabled", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("old-key")
		config.EnableAutoReEncrypt = false
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertext, err := service.Encrypt(context.Background(), "secret")
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "old-key", "new-key")
		require.NoError(t, err)

		plaintext, newCiphertext, wasReEncrypted, err := service.DecryptAndReEncrypt(context.Background(), ciphertext)

		require.NoError(t, err)
		assert.Equal(t, "secret", plaintext)
		assert.Empty(t, newCiphertext)
		assert.False(t, wasReEncrypted)
	})

	t.Run("returns plaintext even when re-encryption fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		encryptCallCount := 0
		provider.encryptFunc = func(_ context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error) {
			encryptCallCount++
			if encryptCallCount > 1 {
				return nil, errors.New("encryption failed")
			}
			return &crypto_dto.EncryptResponse{
				Ciphertext: base64.StdEncoding.EncodeToString([]byte(request.Plaintext)),
				KeyID:      request.KeyID,
				Provider:   crypto_dto.ProviderTypeLocalAESGCM,
			}, nil
		}

		config := createTestConfig("old-key")
		config.EnableAutoReEncrypt = true
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		ciphertext, err := service.Encrypt(context.Background(), "secret")
		require.NoError(t, err)

		err = service.RotateKey(context.Background(), "old-key", "new-key")
		require.NoError(t, err)

		plaintext, newCiphertext, wasReEncrypted, err := service.DecryptAndReEncrypt(context.Background(), ciphertext)

		require.NoError(t, err)
		assert.Equal(t, "secret", plaintext)
		assert.Empty(t, newCiphertext)
		assert.False(t, wasReEncrypted)
	})

	t.Run("returns error when decrypt fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.decryptFunc = func(_ context.Context, _ *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
			return nil, errors.New("decryption failed")
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		envelope := createTestEnvelope("test-key", "local", "ciphertext", "")
		_, _, _, err = service.DecryptAndReEncrypt(context.Background(), envelope)

		require.Error(t, err)
	})

	t.Run("handles invalid envelope format gracefully", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		_, _, _, err = service.DecryptAndReEncrypt(context.Background(), "not-a-valid-envelope")

		require.Error(t, err)
	})
}

func TestIsDeprecatedKey(t *testing.T) {
	t.Parallel()

	t.Run("returns false for active key", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("active-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		assert.False(t, cryptoService.isDeprecatedKey("active-key"))
	})

	t.Run("returns true for deprecated key", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("new-key")
		config.DeprecatedKeyIDs = []string{"old-key-1", "old-key-2"}
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		assert.True(t, cryptoService.isDeprecatedKey("old-key-1"))
		assert.True(t, cryptoService.isDeprecatedKey("old-key-2"))
	})

	t.Run("returns false for unknown key", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("active-key")
		config.DeprecatedKeyIDs = []string{"old-key"}
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		assert.False(t, cryptoService.isDeprecatedKey("unknown-key"))
	})

	t.Run("returns true after key rotation", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("key-1")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")

		assert.False(t, cryptoService.isDeprecatedKey("key-1"))

		err = service.RotateKey(context.Background(), "key-1", "key-2")
		require.NoError(t, err)

		assert.True(t, cryptoService.isDeprecatedKey("key-1"))
	})
}
