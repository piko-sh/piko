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
	"encoding/base64"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEnvelopedCiphertext(t *testing.T) {
	t.Parallel()

	t.Run("creates valid base64 encoded envelope", func(t *testing.T) {
		t.Parallel()

		envelope, err := createEnvelopedCiphertext("key-123", "local", "encrypted-data", "")

		require.NoError(t, err)
		assert.NotEmpty(t, envelope)

		decoded, err := base64.StdEncoding.DecodeString(envelope)
		require.NoError(t, err)
		assert.NotEmpty(t, decoded)
	})

	t.Run("includes all metadata in envelope", func(t *testing.T) {
		t.Parallel()

		envelope, err := createEnvelopedCiphertext("my-key", "aws-kms", "ct-data", "edk-data")
		require.NoError(t, err)

		decoded, err := base64.StdEncoding.DecodeString(envelope)
		require.NoError(t, err)

		var envData envelopeData
		err = json.Unmarshal(decoded, &envData)
		require.NoError(t, err)

		assert.Equal(t, "my-key", envData.KeyID)
		assert.Equal(t, "aws-kms", envData.Provider)
		assert.Equal(t, "ct-data", envData.Ciphertext)
		assert.Equal(t, "edk-data", envData.EncryptedDataKey)
		assert.Equal(t, 1, envData.Version)
	})

	t.Run("handles empty encrypted data key", func(t *testing.T) {
		t.Parallel()

		envelope, err := createEnvelopedCiphertext("key", "local", "ciphertext", "")

		require.NoError(t, err)

		decoded, err := base64.StdEncoding.DecodeString(envelope)
		require.NoError(t, err)

		var envData envelopeData
		err = json.Unmarshal(decoded, &envData)
		require.NoError(t, err)

		assert.Empty(t, envData.EncryptedDataKey)
	})

	t.Run("handles special characters in ciphertext", func(t *testing.T) {
		t.Parallel()

		specialChars := "aGVsbG8gd29ybGQ="
		envelope, err := createEnvelopedCiphertext("key", "local", specialChars, "")

		require.NoError(t, err)

		metadata, err := extractCiphertextMetadata(envelope)
		require.NoError(t, err)
		assert.Equal(t, specialChars, metadata.Ciphertext)
	})

	t.Run("handles unicode in metadata", func(t *testing.T) {
		t.Parallel()

		envelope, err := createEnvelopedCiphertext("key-日本語", "local", "ciphertext", "")

		require.NoError(t, err)

		metadata, err := extractCiphertextMetadata(envelope)
		require.NoError(t, err)
		assert.Equal(t, "key-日本語", metadata.KeyID)
	})
}

func TestExtractCiphertextMetadata(t *testing.T) {
	t.Parallel()

	t.Run("extracts metadata from valid envelope", func(t *testing.T) {
		t.Parallel()

		envelope, err := createEnvelopedCiphertext("test-key", "local", "encrypted-data", "data-key")
		require.NoError(t, err)

		metadata, err := extractCiphertextMetadata(envelope)

		require.NoError(t, err)
		assert.Equal(t, "test-key", metadata.KeyID)
		assert.Equal(t, "local", metadata.Provider)
		assert.Equal(t, "encrypted-data", metadata.Ciphertext)
		assert.Equal(t, "data-key", metadata.EncryptedDataKey)
	})

	t.Run("returns error for invalid base64", func(t *testing.T) {
		t.Parallel()

		_, err := extractCiphertextMetadata("not-valid-base64!!!")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode envelope")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		t.Parallel()

		invalidJSON := base64.StdEncoding.EncodeToString([]byte("not json"))

		_, err := extractCiphertextMetadata(invalidJSON)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse envelope")
	})

	t.Run("returns error for unsupported version", func(t *testing.T) {
		t.Parallel()

		envData := envelopeData{
			KeyID:      "key",
			Provider:   "local",
			Ciphertext: "ct",
			Version:    2,
		}
		jsonData, err := json.Marshal(envData)
		require.NoError(t, err)

		envelope := base64.StdEncoding.EncodeToString(jsonData)

		_, err = extractCiphertextMetadata(envelope)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported envelope version")
	})

	t.Run("returns error for version 0", func(t *testing.T) {
		t.Parallel()

		envData := envelopeData{
			KeyID:      "key",
			Provider:   "local",
			Ciphertext: "ct",
			Version:    0,
		}
		jsonData, err := json.Marshal(envData)
		require.NoError(t, err)

		envelope := base64.StdEncoding.EncodeToString(jsonData)

		_, err = extractCiphertextMetadata(envelope)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported envelope version")
	})

	t.Run("handles empty encrypted data key", func(t *testing.T) {
		t.Parallel()

		envelope, err := createEnvelopedCiphertext("key", "local", "ct", "")
		require.NoError(t, err)

		metadata, err := extractCiphertextMetadata(envelope)

		require.NoError(t, err)
		assert.Empty(t, metadata.EncryptedDataKey)
	})

	t.Run("handles empty string input", func(t *testing.T) {
		t.Parallel()

		_, err := extractCiphertextMetadata("")

		require.Error(t, err)
	})
}

func TestEnvelopeRoundtrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		keyID            string
		provider         string
		ciphertext       string
		encryptedDataKey string
	}{
		{
			name:             "basic envelope",
			keyID:            "key-1",
			provider:         "local",
			ciphertext:       "encrypted-data",
			encryptedDataKey: "",
		},
		{
			name:             "envelope with data key",
			keyID:            "key-2",
			provider:         "aws-kms",
			ciphertext:       "ct-with-edk",
			encryptedDataKey: "encrypted-data-key",
		},
		{
			name:             "complex key ID",
			keyID:            "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			provider:         "aws-kms",
			ciphertext:       "ct",
			encryptedDataKey: "edk",
		},
		{
			name:             "unicode content",
			keyID:            "key-unicode-日本語",
			provider:         "local",
			ciphertext:       "暗号化されたデータ",
			encryptedDataKey: "",
		},
		{
			name:             "long ciphertext",
			keyID:            "key",
			provider:         "local",
			ciphertext:       string(make([]byte, 10000)),
			encryptedDataKey: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			envelope, err := createEnvelopedCiphertext(tc.keyID, tc.provider, tc.ciphertext, tc.encryptedDataKey)
			require.NoError(t, err)

			metadata, err := extractCiphertextMetadata(envelope)
			require.NoError(t, err)

			assert.Equal(t, tc.keyID, metadata.KeyID)
			assert.Equal(t, tc.provider, metadata.Provider)
			assert.Equal(t, tc.ciphertext, metadata.Ciphertext)
			assert.Equal(t, tc.encryptedDataKey, metadata.EncryptedDataKey)
		})
	}
}
