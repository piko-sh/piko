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
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func generateTestKey() []byte {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic("failed to generate test key: " + err.Error())
	}
	return key
}

func TestNewProvider_ValidKey(t *testing.T) {
	key := generateTestKey()
	provider, err := NewProvider(Config{
		Key:   key,
		KeyID: "test-key",
	})

	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, crypto_dto.ProviderTypeLocalAESGCM, provider.Type())
}

func TestNewProvider_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{name: "too short", keySize: 16},
		{name: "too long", keySize: 64},
		{name: "empty", keySize: 0},
		{name: "slightly short", keySize: 31},
		{name: "slightly long", keySize: 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			provider, err := NewProvider(Config{Key: key})

			assert.Error(t, err)
			assert.Nil(t, provider)
			assert.ErrorIs(t, err, ErrInvalidKeySize)
		})
	}
}

func TestNewProvider_DefaultKeyID(t *testing.T) {
	provider, err := NewProvider(Config{
		Key: generateTestKey(),
	})

	require.NoError(t, err)

	info, err := provider.GetKeyInfo(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "local-default", info.KeyID)
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{name: "short string", plaintext: "hello"},
		{name: "medium string", plaintext: "This is a test message with some length"},
		{name: "long string", plaintext: strings.Repeat("a", 1000)},
		{name: "very long string", plaintext: strings.Repeat("test ", 10000)},
		{name: "special characters", plaintext: "!@#$%^&*()_+-=[]{}|;:',.<>?/~`"},
		{name: "unicode", plaintext: "Hello 世界 🌍 émojis 🎉"},
		{name: "newlines", plaintext: "line1\nline2\nline3"},
		{name: "tabs and spaces", plaintext: "tab\there\tand  spaces"},
		{name: "single character", plaintext: "x"},
		{name: "JSON-like", plaintext: `{"key":"value","nested":{"data":[1,2,3]}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := provider.Encrypt(ctx, &crypto_dto.EncryptRequest{
				Plaintext: tt.plaintext,
			})
			require.NoError(t, err)
			require.NotEmpty(t, encrypted.Ciphertext)

			assert.NotEqual(t, tt.plaintext, encrypted.Ciphertext)

			_, err = base64.StdEncoding.DecodeString(encrypted.Ciphertext)
			assert.NoError(t, err, "ciphertext should be valid base64")

			decrypted, err := provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
				Ciphertext: encrypted.Ciphertext,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted.Plaintext)
		})
	}
}

func TestEncrypt_ProducesUniqueIVs(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	plaintext := "test message"
	ciphertexts := make(map[string]bool)

	for i := range 100 {
		encrypted, err := provider.Encrypt(ctx, &crypto_dto.EncryptRequest{
			Plaintext: plaintext,
		})
		require.NoError(t, err)

		assert.False(t, ciphertexts[encrypted.Ciphertext], "found duplicate ciphertext at iteration %d", i)
		ciphertexts[encrypted.Ciphertext] = true
	}

	assert.Len(t, ciphertexts, 100, "all ciphertexts should be unique")
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	_, err = provider.Encrypt(ctx, &crypto_dto.EncryptRequest{
		Plaintext: "",
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, crypto_dto.ErrEmptyPlaintext)
}

func TestDecrypt_EmptyCiphertext(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	_, err = provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
		Ciphertext: "",
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, crypto_dto.ErrEmptyCiphertext)
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	tests := []string{
		"not base64!@#$",
		"invalid==padding",
		"contains spaces ",
		"unicode🎉notbase64",
	}

	for _, invalidCiphertext := range tests {
		_, err := provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
			Ciphertext: invalidCiphertext,
		})

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidBase64)
	}
}

func TestDecrypt_CiphertextTooShort(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	shortData := make([]byte, 10)
	shortCiphertext := base64.StdEncoding.EncodeToString(shortData)

	_, err = provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
		Ciphertext: shortCiphertext,
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCiphertextTooShort)
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	plaintext := "sensitive data"
	encrypted, err := provider.Encrypt(ctx, &crypto_dto.EncryptRequest{
		Plaintext: plaintext,
	})
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
	require.NoError(t, err)

	decoded[20] ^= 0xFF
	tamperedCiphertext := base64.StdEncoding.EncodeToString(decoded)

	_, err = provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
		Ciphertext: tamperedCiphertext,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decryption failed")
}

func TestDecrypt_WrongKey(t *testing.T) {
	ctx := context.Background()

	provider1, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	encrypted, err := provider1.Encrypt(ctx, &crypto_dto.EncryptRequest{
		Plaintext: "secret",
	})
	require.NoError(t, err)

	provider2, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	_, err = provider2.Decrypt(ctx, &crypto_dto.DecryptRequest{
		Ciphertext: encrypted.Ciphertext,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decryption failed")
}

func TestGenerateDataKey(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	dataKey, err := provider.GenerateDataKey(ctx, &crypto_dto.GenerateDataKeyRequest{
		KeyID: "test-key",
	})
	require.NoError(t, err)
	defer func() { _ = dataKey.Close() }()

	assert.Equal(t, KeySize, dataKey.PlaintextKey.Len(), "data key should be 32 bytes")
	assert.NotEmpty(t, dataKey.EncryptedKey, "encrypted key should not be empty")
	assert.Equal(t, crypto_dto.ProviderTypeLocalAESGCM, dataKey.Provider)

	decrypted, err := provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
		Ciphertext: dataKey.EncryptedKey,
	})
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(decrypted.Plaintext)
	require.NoError(t, err)

	var plaintextKeyBytes []byte
	err = dataKey.PlaintextKey.WithAccess(func(data []byte) error {
		plaintextKeyBytes = make([]byte, len(data))
		copy(plaintextKeyBytes, data)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, plaintextKeyBytes, decoded)
}

func TestGenerateDataKey_ProducesUnique(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	keys := make(map[string]bool)

	for i := range 50 {
		dataKey, err := provider.GenerateDataKey(ctx, &crypto_dto.GenerateDataKeyRequest{})
		require.NoError(t, err)

		var keyString string
		err = dataKey.PlaintextKey.WithAccess(func(keyBytes []byte) error {
			keyString = base64.StdEncoding.EncodeToString(keyBytes)
			return nil
		})
		require.NoError(t, err)
		_ = dataKey.Close()

		assert.False(t, keys[keyString], "found duplicate data key at iteration %d", i)
		keys[keyString] = true
	}

	assert.Len(t, keys, 50, "all data keys should be unique")
}

func TestGetKeyInfo(t *testing.T) {
	ctx := context.Background()
	keyID := "test-key-id"
	provider, err := NewProvider(Config{
		Key:   generateTestKey(),
		KeyID: keyID,
	})
	require.NoError(t, err)

	info, err := provider.GetKeyInfo(ctx, keyID)
	require.NoError(t, err)

	assert.Equal(t, keyID, info.KeyID)
	assert.Equal(t, crypto_dto.ProviderTypeLocalAESGCM, info.Provider)
	assert.Equal(t, "AES-256-GCM", info.Algorithm)
	assert.Equal(t, crypto_dto.KeyStatusActive, info.Status)
	assert.Equal(t, "LOCAL", info.Origin)
	assert.NotNil(t, info.Metadata)
}

func TestGetKeyInfo_WrongKeyID(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{
		Key:   generateTestKey(),
		KeyID: "correct-id",
	})
	require.NoError(t, err)

	_, err = provider.GetKeyInfo(ctx, "wrong-id")
	assert.Error(t, err)
	assert.ErrorIs(t, err, crypto_dto.ErrKeyNotFound)
}

func TestHealthCheck_Success(t *testing.T) {
	ctx := context.Background()
	provider, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	err = provider.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestGetKeyInfo_EmptyKeyID(t *testing.T) {
	ctx := context.Background()
	p, err := NewProvider(Config{
		Key:   generateTestKey(),
		KeyID: "my-key",
	})
	require.NoError(t, err)

	info, err := p.GetKeyInfo(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "my-key", info.KeyID)
}

func TestZeroBytes(t *testing.T) {
	data := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE}
	zeroBytes(data)

	for i, b := range data {
		assert.Equal(t, byte(0), b, "byte at index %d should be zero", i)
	}
}

func TestZeroBytes_Empty(t *testing.T) {

	zeroBytes(nil)
	zeroBytes([]byte{})
}

func TestGenerateDataKey_KeyIDPropagated(t *testing.T) {
	ctx := context.Background()
	p, err := NewProvider(Config{
		Key:   generateTestKey(),
		KeyID: "master-key-1",
	})
	require.NoError(t, err)

	dataKey, err := p.GenerateDataKey(ctx, &crypto_dto.GenerateDataKeyRequest{
		KeyID: "master-key-1",
	})
	require.NoError(t, err)
	defer func() { _ = dataKey.Close() }()

	assert.Equal(t, "master-key-1", dataKey.KeyID)
	assert.Equal(t, crypto_dto.ProviderTypeLocalAESGCM, dataKey.Provider)

	assert.Equal(t, KeySize, dataKey.PlaintextKey.Len())

	decrypted, err := p.Decrypt(ctx, &crypto_dto.DecryptRequest{
		Ciphertext: dataKey.EncryptedKey,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, decrypted.Plaintext)
}

func TestEncrypt_ResponseFields(t *testing.T) {
	ctx := context.Background()
	p, err := NewProvider(Config{
		Key:   generateTestKey(),
		KeyID: "response-test-key",
	})
	require.NoError(t, err)

	response, err := p.Encrypt(ctx, &crypto_dto.EncryptRequest{
		Plaintext: "test",
	})
	require.NoError(t, err)

	assert.Equal(t, "response-test-key", response.KeyID)
	assert.Equal(t, crypto_dto.ProviderTypeLocalAESGCM, response.Provider)
	assert.NotEmpty(t, response.Ciphertext)
}

func TestType(t *testing.T) {
	p, err := NewProvider(Config{Key: generateTestKey()})
	require.NoError(t, err)

	assert.Equal(t, crypto_dto.ProviderTypeLocalAESGCM, p.Type())
}

func BenchmarkEncrypt(b *testing.B) {
	ctx := context.Background()
	provider, _ := NewProvider(Config{Key: generateTestKey()})
	plaintext := "This is a test message for benchmarking encryption performance"

	b.ResetTimer()
	for b.Loop() {
		_, _ = provider.Encrypt(ctx, &crypto_dto.EncryptRequest{
			Plaintext: plaintext,
		})
	}
}

func BenchmarkDecrypt(b *testing.B) {
	ctx := context.Background()
	provider, _ := NewProvider(Config{Key: generateTestKey()})
	plaintext := "This is a test message for benchmarking decryption performance"

	encrypted, _ := provider.Encrypt(ctx, &crypto_dto.EncryptRequest{
		Plaintext: plaintext,
	})

	b.ResetTimer()
	for b.Loop() {
		_, _ = provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
			Ciphertext: encrypted.Ciphertext,
		})
	}
}

func BenchmarkRoundtrip(b *testing.B) {
	ctx := context.Background()
	provider, _ := NewProvider(Config{Key: generateTestKey()})
	plaintext := "This is a test message for benchmarking roundtrip performance"

	b.ResetTimer()
	for b.Loop() {
		encrypted, _ := provider.Encrypt(ctx, &crypto_dto.EncryptRequest{
			Plaintext: plaintext,
		})
		_, _ = provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
			Ciphertext: encrypted.Ciphertext,
		})
	}
}
