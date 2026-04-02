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

package local_aes_gcm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)
}

func TestFactory_CreateWithKey(t *testing.T) {
	t.Run("valid key creates provider", func(t *testing.T) {
		factory := NewFactory()
		key := generateTestKey()

		secureKey, err := crypto_dto.NewSecureBytesFromSlice(key, crypto_dto.WithID("test-key"))
		require.NoError(t, err)
		defer func() { _ = secureKey.Close() }()

		provider, err := factory.CreateWithKey(secureKey, "factory-test-key")
		require.NoError(t, err)
		require.NotNil(t, provider)

		assert.Equal(t, crypto_dto.ProviderTypeLocalAESGCM, provider.Type())

		ctx := context.Background()
		encrypted, err := provider.Encrypt(ctx, &crypto_dto.EncryptRequest{
			Plaintext: "factory test",
		})
		require.NoError(t, err)

		decrypted, err := provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
			Ciphertext: encrypted.Ciphertext,
		})
		require.NoError(t, err)
		assert.Equal(t, "factory test", decrypted.Plaintext)
	})

	t.Run("invalid key size returns error", func(t *testing.T) {
		factory := NewFactory()
		shortKey := make([]byte, 16)

		secureKey, err := crypto_dto.NewSecureBytesFromSlice(shortKey, crypto_dto.WithID("short-key"))
		require.NoError(t, err)
		defer func() { _ = secureKey.Close() }()

		provider, err := factory.CreateWithKey(secureKey, "bad-key")
		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("key ID propagates to provider", func(t *testing.T) {
		factory := NewFactory()
		key := generateTestKey()

		secureKey, err := crypto_dto.NewSecureBytesFromSlice(key, crypto_dto.WithID("test-key"))
		require.NoError(t, err)
		defer func() { _ = secureKey.Close() }()

		provider, err := factory.CreateWithKey(secureKey, "my-custom-key-id")
		require.NoError(t, err)

		info, err := provider.GetKeyInfo(context.Background(), "my-custom-key-id")
		require.NoError(t, err)
		assert.Equal(t, "my-custom-key-id", info.KeyID)
	})
}
