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
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func TestNewDisabledCryptoService(t *testing.T) {
	t.Parallel()

	service := NewDisabledCryptoService()
	require.NotNil(t, service)
}

func TestDisabledCryptoService_EncryptionOperationsReturnDisabledError(t *testing.T) {
	t.Parallel()

	service := NewDisabledCryptoService()
	ctx := context.Background()

	t.Run("Encrypt returns empty string and ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		result, err := service.Encrypt(ctx, "plaintext")
		assert.Empty(t, result)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("Decrypt returns empty string and ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		result, err := service.Decrypt(ctx, "ciphertext")
		assert.Empty(t, result)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("EncryptWithKey returns empty string and ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		result, err := service.EncryptWithKey(ctx, "plaintext", "key-id")
		assert.Empty(t, result)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("EncryptBatch returns nil and ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		result, err := service.EncryptBatch(ctx, []string{"a", "b"})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("DecryptBatch returns nil and ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		result, err := service.DecryptBatch(ctx, []string{"a", "b"})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("RotateKey returns ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		err := service.RotateKey(ctx, "old-key", "new-key")
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("GetActiveKeyID returns empty string and ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		result, err := service.GetActiveKeyID(ctx)
		assert.Empty(t, result)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("DecryptAndReEncrypt returns empty values and ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		ciphertext, keyID, rotated, err := service.DecryptAndReEncrypt(ctx, "ciphertext")
		assert.Empty(t, ciphertext)
		assert.Empty(t, keyID)
		assert.False(t, rotated)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("EncryptStream returns nil and ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		writer, err := service.EncryptStream(ctx, &bytes.Buffer{}, "key-id")
		assert.Nil(t, writer)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("DecryptStream returns nil and ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		reader, err := service.DecryptStream(ctx, &bytes.Buffer{})
		assert.Nil(t, reader)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("RegisterProvider returns ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		err := service.RegisterProvider(context.Background(), "test", nil)
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})

	t.Run("SetDefaultProvider returns ErrCryptoDisabled", func(t *testing.T) {
		t.Parallel()
		err := service.SetDefaultProvider("test")
		assert.ErrorIs(t, err, crypto_dto.ErrCryptoDisabled)
	})
}

func TestDisabledCryptoService_NonErrorOperations(t *testing.T) {
	t.Parallel()

	service := NewDisabledCryptoService()
	ctx := context.Background()

	t.Run("HealthCheck returns nil", func(t *testing.T) {
		t.Parallel()
		err := service.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("Close returns nil", func(t *testing.T) {
		t.Parallel()
		err := service.Close(ctx)
		assert.NoError(t, err)
	})

	t.Run("GetProviders returns empty slice", func(t *testing.T) {
		t.Parallel()
		providers := service.GetProviders(ctx)
		assert.Empty(t, providers)
	})

	t.Run("HasProvider returns false", func(t *testing.T) {
		t.Parallel()
		assert.False(t, service.HasProvider("any-provider"))
	})

	t.Run("ListProviders returns empty slice", func(t *testing.T) {
		t.Parallel()
		providers := service.ListProviders(ctx)
		assert.Empty(t, providers)
	})
}

func TestDisabledCryptoService_BuilderCreation(t *testing.T) {
	t.Parallel()

	service := NewDisabledCryptoService()

	t.Run("NewEncrypt returns non-nil builder", func(t *testing.T) {
		t.Parallel()
		builder := service.NewEncrypt()
		assert.NotNil(t, builder)
	})

	t.Run("NewDecrypt returns non-nil builder", func(t *testing.T) {
		t.Parallel()
		builder := service.NewDecrypt()
		assert.NotNil(t, builder)
	})

	t.Run("NewBatchEncrypt returns non-nil builder", func(t *testing.T) {
		t.Parallel()
		builder := service.NewBatchEncrypt()
		assert.NotNil(t, builder)
	})

	t.Run("NewBatchDecrypt returns non-nil builder", func(t *testing.T) {
		t.Parallel()
		builder := service.NewBatchDecrypt()
		assert.NotNil(t, builder)
	})

	t.Run("NewStreamEncrypt returns non-nil builder", func(t *testing.T) {
		t.Parallel()
		builder := service.NewStreamEncrypt()
		assert.NotNil(t, builder)
	})

	t.Run("NewStreamDecrypt returns non-nil builder", func(t *testing.T) {
		t.Parallel()
		builder := service.NewStreamDecrypt()
		assert.NotNil(t, builder)
	})
}
