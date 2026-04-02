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

package storage_integration_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/storage/storage_domain"
)

func TestCrypto_RoundTrip(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	testCases := []struct {
		name string
		size int
	}{
		{name: "small/256B", size: 256},
		{name: "medium/1MB", size: 1024 * 1024},
		{name: "large/5MB", size: 5 * 1024 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := generateRepeatableText(tc.size)
			key := uniqueKey(t, "crypto-roundtrip")
			wrapper := newTransformerWrapper(t, provider,
				[]storage_domain.StreamTransformerPort{crypto.transformer}, []string{"crypto-service"})

			putObject(ctx, t, wrapper, key, original)

			raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
			assert.True(t, isStreamingEnvelope(raw),
				"raw bytes should start with streaming envelope version byte (0x02)")
			assert.True(t, isNotPlaintext(raw, original),
				"raw bytes should not contain plaintext")

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved, "round-trip content should match original")
		})
	}
}

func TestCrypto_EncryptedIsNotPlaintext(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	key := uniqueKey(t, "crypto-plaintext-check")

	original := []byte("This is a secret message that must be encrypted and never stored in plaintext")
	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{crypto.transformer}, []string{"crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	assert.NotContains(t, string(raw), "secret",
		"encrypted data must not contain plaintext substring 'secret'")
	assert.NotContains(t, string(raw), "encrypted",
		"encrypted data must not contain plaintext substring 'encrypted'")
	assert.NotContains(t, string(raw), "plaintext",
		"encrypted data must not contain plaintext substring 'plaintext'")
}

func TestCrypto_StreamingEnvelopeFormat(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	key := uniqueKey(t, "crypto-envelope-format")

	original := generateRepeatableText(1024)
	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{crypto.transformer}, []string{"crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	require.GreaterOrEqual(t, len(raw), 5, "encrypted data must be at least 5 bytes")

	assert.Equal(t, crypto_dto.StreamingEnvelopeVersion, raw[0],
		"first byte should be streaming envelope version 0x02")

	headerLen := binary.BigEndian.Uint32(raw[1:5])
	require.LessOrEqual(t, int(headerLen), len(raw)-5,
		"header length should not exceed remaining data")

	headerJSON := raw[5 : 5+headerLen]
	var header crypto_dto.StreamingHeader
	err := json.Unmarshal(headerJSON, &header)
	require.NoError(t, err, "header should be valid JSON")

	assert.Equal(t, 2, header.Version, "header version should be 2")
	assert.NotEmpty(t, header.KeyID, "header should include key_id")
	assert.NotEmpty(t, header.Provider, "header should include provider")
	assert.NotEmpty(t, header.IV, "header should include IV")
}

func TestCrypto_NonDeterministic(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	original := generateRepeatableText(1024)

	key1 := uniqueKey(t, "crypto-nondet-1")
	key2 := uniqueKey(t, "crypto-nondet-2")

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{crypto.transformer}, []string{"crypto-service"})

	putObject(ctx, t, wrapper, key1, original)
	putObject(ctx, t, wrapper, key2, original)

	raw1 := getRawBytesFromS3(ctx, t, testBucketPrimary, key1)
	raw2 := getRawBytesFromS3(ctx, t, testBucketPrimary, key2)

	assert.NotEqual(t, raw1, raw2,
		"encrypting the same content twice should produce different ciphertext (random IV)")
}

func TestCrypto_DecryptRawWithService(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	key := uniqueKey(t, "crypto-decrypt-raw")

	original := generateRepeatableText(10 * 1024)
	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{crypto.transformer}, []string{"crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	reader, err := crypto.service.DecryptStream(ctx, bytes.NewReader(raw))
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	decrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted,
		"decrypting raw S3 bytes with crypto service should recover original")
}
