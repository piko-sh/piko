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

package cache_integration_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
)

func TestCrypto_RoundTrip(t *testing.T) {
	t.Parallel()

	crypto := newCryptoSetup(t)
	ctx := testContext(t)

	testCases := []struct {
		name string
		size int
	}{
		{name: "256B", size: 256},
		{name: "1KB", size: 1024},
		{name: "1MB", size: 1024 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := []byte(generateRepeatableText(tc.size))

			encrypted, err := crypto.transformer.Transform(ctx, original, nil)
			require.NoError(t, err, "encrypting %s", tc.name)

			decrypted, err := crypto.transformer.Reverse(ctx, encrypted, nil)
			require.NoError(t, err, "decrypting %s", tc.name)

			assert.Equal(t, original, decrypted, "round-trip mismatch for %s", tc.name)
		})
	}
}

func TestCrypto_PlaintextNotInRaw(t *testing.T) {
	t.Parallel()

	crypto := newCryptoSetup(t)
	namespace := uniqueKey(t, "crypto-plaintext") + ":"
	transformers := []cache_domain.CacheTransformerPort{crypto.transformer}

	secret := []byte("this is a secret message that must not appear in raw form")

	wrapped := transformAndWrap(t, secret, transformers)

	redisKey := rawRedisKeyForNamespace(namespace, "secret-key")
	putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

	raw := getRawBytesFromRedis(t, redisKey)

	assert.False(t, bytes.Contains(raw, secret),
		"raw Redis bytes must not contain the plaintext secret")
}

func TestCrypto_NonDeterministic(t *testing.T) {
	t.Parallel()

	crypto := newCryptoSetup(t)
	ctx := testContext(t)

	original := []byte("identical content for both encryptions")

	encrypted1, err := crypto.transformer.Transform(ctx, original, nil)
	require.NoError(t, err, "first encryption")

	encrypted2, err := crypto.transformer.Transform(ctx, original, nil)
	require.NoError(t, err, "second encryption")

	assert.NotEqual(t, encrypted1, encrypted2,
		"encrypting the same content twice should produce different ciphertexts (different IVs)")

	decrypted1, err := crypto.transformer.Reverse(ctx, encrypted1, nil)
	require.NoError(t, err, "decrypting first")

	decrypted2, err := crypto.transformer.Reverse(ctx, encrypted2, nil)
	require.NoError(t, err, "decrypting second")

	assert.Equal(t, original, decrypted1, "first decryption mismatch")
	assert.Equal(t, original, decrypted2, "second decryption mismatch")
}

func TestCrypto_DecryptWithServiceDirectly(t *testing.T) {
	t.Parallel()

	crypto := newCryptoSetup(t)
	ctx := testContext(t)
	namespace := uniqueKey(t, "crypto-service") + ":"
	transformers := []cache_domain.CacheTransformerPort{crypto.transformer}

	original := []byte("data to encrypt and store in redis")

	wrapped := transformAndWrap(t, original, transformers)
	redisKey := rawRedisKeyForNamespace(namespace, "service-key")
	putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

	raw := getRawBytesFromRedis(t, redisKey)
	tv, err := cache_domain.UnmarshalTransformedValue(raw)
	require.NoError(t, err, "unmarshalling transformed value")
	require.Contains(t, tv.Transformers, "crypto-service",
		"transformer list should include crypto-service")

	decrypted, err := crypto.service.Decrypt(ctx, string(tv.Data))
	require.NoError(t, err, "decrypting with crypto service directly")

	assert.Equal(t, string(original), decrypted,
		"data decrypted via crypto service should match original")

	recovered := reverseAndUnwrap(t, raw, transformers)
	assert.Equal(t, original, recovered, "reverse pipeline should match original")
}

func TestCrypto_RoundTripViaRedis(t *testing.T) {
	t.Parallel()

	crypto := newCryptoSetup(t)
	namespace := uniqueKey(t, "crypto-redis") + ":"
	transformers := []cache_domain.CacheTransformerPort{crypto.transformer}

	testCases := []struct {
		name string
		size int
	}{
		{name: "256B", size: 256},
		{name: "1KB", size: 1024},
		{name: "64KB", size: 64 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := []byte(generateRepeatableText(tc.size))

			wrapped := transformAndWrap(t, original, transformers)

			redisKey := rawRedisKeyForNamespace(namespace, tc.name)
			putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

			raw := getRawBytesFromRedis(t, redisKey)
			recovered := reverseAndUnwrap(t, raw, transformers)

			assert.Equal(t, original, recovered, "Redis round-trip mismatch for %s", tc.name)
		})
	}
}

func TestCrypto_BinaryData(t *testing.T) {
	t.Parallel()

	crypto := newCryptoSetup(t)
	ctx := testContext(t)

	original := generateRandomData(t, 8192)

	encrypted, err := crypto.transformer.Transform(ctx, original, nil)
	require.NoError(t, err, "encrypting binary data")

	decrypted, err := crypto.transformer.Reverse(ctx, encrypted, nil)
	require.NoError(t, err, "decrypting binary data")

	assert.Equal(t, original, decrypted, "binary data round-trip mismatch")
}
