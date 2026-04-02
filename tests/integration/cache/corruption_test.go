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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
)

func TestCorruption_CorruptedCompressedData(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	ctx := testContext(t)

	original := []byte(generateRepeatableText(1024))
	compressed, err := zstd.Transform(ctx, original, nil)
	require.NoError(t, err, "compressing original data")

	corrupted := make([]byte, len(compressed))
	copy(corrupted, compressed)
	if len(corrupted) > 10 {
		corrupted[5] ^= 0xFF
		corrupted[10] ^= 0xFF
	}

	_, err = zstd.Reverse(ctx, corrupted, nil)
	assert.Error(t, err, "decompressing corrupted data should fail")
}

func TestCorruption_CorruptedEncryptedData(t *testing.T) {
	t.Parallel()

	crypto := newCryptoSetup(t)
	ctx := testContext(t)

	original := []byte("secret data for corruption test")
	encrypted, err := crypto.transformer.Transform(ctx, original, nil)
	require.NoError(t, err, "encrypting original data")

	corrupted := make([]byte, len(encrypted))
	copy(corrupted, encrypted)
	if len(corrupted) > 10 {
		corrupted[len(corrupted)-1] ^= 0xFF
		corrupted[len(corrupted)-5] ^= 0xFF
	}

	_, err = crypto.transformer.Reverse(ctx, corrupted, nil)
	assert.Error(t, err, "decrypting corrupted data should fail")
}

func TestCorruption_WrongCryptoKey(t *testing.T) {
	t.Parallel()

	cryptoA := newCryptoSetupWithKey(t, "key-material-for-writer-a-32b!", "key-a")
	cryptoB := newCryptoSetupWithKey(t, "key-material-for-writer-b-32b!", "key-b")
	namespace := uniqueKey(t, "corruption-wrongkey") + ":"
	transformersA := []cache_domain.CacheTransformerPort{cryptoA.transformer}

	original := []byte("encrypted with key A")

	wrapped := transformAndWrap(t, original, transformersA)
	redisKey := rawRedisKeyForNamespace(namespace, "wrong-key")
	putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

	raw := getRawBytesFromRedis(t, redisKey)
	tv, err := cache_domain.UnmarshalTransformedValue(raw)
	require.NoError(t, err, "unmarshalling transformed value")

	ctx := testContext(t)

	_, err = cryptoB.transformer.Reverse(ctx, tv.Data, nil)
	assert.Error(t, err, "decrypting with wrong key should fail")
}

func TestCorruption_TruncatedTransformedData(t *testing.T) {
	t.Parallel()

	namespace := uniqueKey(t, "corruption-truncated") + ":"

	truncated := []byte(`{"data":"aGVsbG8=","tr`)
	redisKey := rawRedisKeyForNamespace(namespace, "truncated")
	putRawBytesToRedis(t, redisKey, truncated, 1*time.Hour)

	raw := getRawBytesFromRedis(t, redisKey)

	_, err := cache_domain.UnmarshalTransformedValue(raw)
	assert.Error(t, err, "unmarshalling truncated data should fail")
}

func TestCorruption_MalformedWrapperJSON(t *testing.T) {
	t.Parallel()

	namespace := uniqueKey(t, "corruption-malformed") + ":"

	testCases := []struct {
		name string
		data []byte
	}{
		{name: "not-json", data: []byte("this is not json at all")},
		{name: "empty-json", data: []byte("{}")},
		{name: "invalid-utf8", data: []byte{0xFF, 0xFE, 0xFD}},
		{name: "null", data: []byte("null")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			redisKey := rawRedisKeyForNamespace(namespace, tc.name)
			putRawBytesToRedis(t, redisKey, tc.data, 1*time.Hour)

			raw := getRawBytesFromRedis(t, redisKey)

			tv, err := cache_domain.UnmarshalTransformedValue(raw)
			if err == nil {

				t.Logf("Unmarshalled %q: data=%d bytes, transformers=%v",
					tc.name, len(tv.Data), tv.Transformers)
			}
		})
	}
}

func TestCorruption_BitFlip(t *testing.T) {
	t.Parallel()

	crypto := newCryptoSetup(t)
	ctx := testContext(t)

	namespace := uniqueKey(t, "corruption-bitflip") + ":"
	transformers := []cache_domain.CacheTransformerPort{crypto.transformer}

	original := []byte("data for bit flip test - must be authenticated")

	wrapped := transformAndWrap(t, original, transformers)

	tv, err := cache_domain.UnmarshalTransformedValue(wrapped)
	require.NoError(t, err, "unmarshalling for bit flip")

	if len(tv.Data) > 5 {

		flippedData := make([]byte, len(tv.Data))
		copy(flippedData, tv.Data)
		flippedData[len(flippedData)/2] ^= 0x01

		flippedTV := cache_domain.NewTransformedValue(flippedData, tv.Transformers)
		flippedWrapped, err := flippedTV.Marshal()
		require.NoError(t, err, "marshalling flipped value")

		redisKey := rawRedisKeyForNamespace(namespace, "bitflip")
		putRawBytesToRedis(t, redisKey, flippedWrapped, 1*time.Hour)

		raw := getRawBytesFromRedis(t, redisKey)
		parsedTV, err := cache_domain.UnmarshalTransformedValue(raw)
		require.NoError(t, err, "unmarshalling flipped value")

		_, err = crypto.transformer.Reverse(ctx, parsedTV.Data, nil)
		assert.Error(t, err, "decrypting with flipped bit should fail (authentication failure)")
	}
}

func TestCorruption_CorruptedChainedData(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	crypto := newCryptoSetup(t)
	ctx := testContext(t)

	original := []byte(generateRepeatableText(2048))

	compressed, err := zstd.Transform(ctx, original, nil)
	require.NoError(t, err, "compressing")

	encrypted, err := crypto.transformer.Transform(ctx, compressed, nil)
	require.NoError(t, err, "encrypting")

	corrupted := make([]byte, len(encrypted))
	copy(corrupted, encrypted)
	if len(corrupted) > 10 {
		corrupted[len(corrupted)-3] ^= 0xFF
	}

	_, err = crypto.transformer.Reverse(ctx, corrupted, nil)
	assert.Error(t, err, "decrypting corrupted outer layer should fail")
}
