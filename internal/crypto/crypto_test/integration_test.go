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

package crypto_test_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_adapters/local_aes_gcm"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/crypto/crypto_test"
)

func TestCryptoService_Integration(t *testing.T) {
	ctx := context.Background()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	provider, err := local_aes_gcm.NewProvider(local_aes_gcm.Config{
		Key:   key,
		KeyID: "test-key",
	})
	require.NoError(t, err)

	serviceConfig := &crypto_dto.ServiceConfig{
		ActiveKeyID:  "test-key",
		ProviderType: crypto_dto.ProviderTypeLocalAESGCM,
	}

	service, err := crypto_domain.NewCryptoService(ctx, nil, serviceConfig)
	require.NoError(t, err)

	err = service.RegisterProvider(ctx, "local-aes-gcm", provider)
	require.NoError(t, err)

	err = service.SetDefaultProvider("local-aes-gcm")
	require.NoError(t, err)

	err = service.HealthCheck(ctx)
	assert.NoError(t, err)

	plaintext := "test_oauth_access_token_12345"
	encrypted, err := service.Encrypt(ctx, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := service.Decrypt(ctx, encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	keyID, err := service.GetActiveKeyID(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test-key", keyID)
}

func TestCryptoService_BatchOperations(t *testing.T) {
	ctx := context.Background()

	key := make([]byte, 32)
	provider, _ := local_aes_gcm.NewProvider(local_aes_gcm.Config{Key: key})
	factory := local_aes_gcm.NewFactory()

	service, err := crypto_domain.NewCryptoService(
		ctx,
		nil,
		crypto_dto.DefaultServiceConfig(),
		crypto_domain.WithLocalProviderFactory(factory),
	)
	require.NoError(t, err)

	err = service.RegisterProvider(ctx, "local-aes-gcm", provider)
	require.NoError(t, err)

	err = service.SetDefaultProvider("local-aes-gcm")
	require.NoError(t, err)

	plaintexts := []string{"token1", "token2", "token3"}
	encrypted, err := service.EncryptBatch(ctx, plaintexts)
	require.NoError(t, err)
	assert.Len(t, encrypted, 3)

	assert.NotEqual(t, encrypted[0], encrypted[1])
	assert.NotEqual(t, encrypted[1], encrypted[2])

	decrypted, err := service.DecryptBatch(ctx, encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintexts, decrypted)
}

func TestMockCryptoService_Deterministic(t *testing.T) {
	ctx := context.Background()

	mock := &crypto_test.MockCryptoService{
		EncryptFunc: func(_ context.Context, plaintext string) (string, error) {
			return "enc:" + plaintext, nil
		},
		DecryptFunc: func(_ context.Context, ciphertext string) (string, error) {
			const prefix = "enc:"
			if len(ciphertext) > len(prefix) {
				return ciphertext[len(prefix):], nil
			}
			return ciphertext, nil
		},
	}

	plaintext := "test_token"

	encrypted1, err := mock.Encrypt(ctx, plaintext)
	require.NoError(t, err)

	encrypted2, err := mock.Encrypt(ctx, plaintext)
	require.NoError(t, err)

	assert.Equal(t, encrypted1, encrypted2, "mock crypto should be deterministic")

	decrypted, err := mock.Decrypt(ctx, encrypted1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
