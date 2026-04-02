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

func TestChained_ZstdThenCrypto_RoundTrip(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	crypto := newCryptoSetup(t)
	transformers := []cache_domain.CacheTransformerPort{zstd, crypto.transformer}

	original := []byte(generateRepeatableText(4096))

	wrapped := transformAndWrap(t, original, transformers)
	recovered := reverseAndUnwrap(t, wrapped, transformers)

	assert.Equal(t, original, recovered, "chained zstd+crypto round-trip mismatch")
}

func TestChained_VerifyLayerOrder(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	crypto := newCryptoSetup(t)
	ctx := testContext(t)

	original := []byte(generateRepeatableText(2048))

	compressed, err := zstd.Transform(ctx, original, nil)
	require.NoError(t, err, "compressing")

	encrypted, err := crypto.transformer.Transform(ctx, compressed, nil)
	require.NoError(t, err, "encrypting")

	tv := cache_domain.NewTransformedValue(encrypted, []string{"zstd", "crypto-service"})
	wrapped, err := tv.Marshal()
	require.NoError(t, err, "marshalling TransformedValue")

	parsed, err := cache_domain.UnmarshalTransformedValue(wrapped)
	require.NoError(t, err, "unmarshalling TransformedValue")

	assert.Equal(t, []string{"zstd", "crypto-service"}, parsed.Transformers,
		"transformer names should be [zstd, crypto-service]")

	assert.False(t, bytes.Contains(parsed.Data, original),
		"outer encrypted data should not contain plaintext")

	decrypted, err := crypto.transformer.Reverse(ctx, parsed.Data, nil)
	require.NoError(t, err, "decrypting outer layer")

	decompressed, err := zstd.Reverse(ctx, decrypted, nil)
	require.NoError(t, err, "decompressing inner layer")

	assert.Equal(t, original, decompressed, "full reverse should recover original")
}

func TestChained_VariousSizes(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	crypto := newCryptoSetup(t)
	transformers := []cache_domain.CacheTransformerPort{zstd, crypto.transformer}
	namespace := uniqueKey(t, "chained-sizes") + ":"

	testCases := []struct {
		name string
		size int
	}{
		{name: "1B", size: 1},
		{name: "100B", size: 100},
		{name: "10KB", size: 10 * 1024},
		{name: "1MB", size: 1024 * 1024},
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

			assert.Equal(t, original, recovered, "chained round-trip mismatch for %s", tc.name)
		})
	}
}

func TestChained_LargeFile(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	crypto := newCryptoSetup(t)
	transformers := []cache_domain.CacheTransformerPort{zstd, crypto.transformer}
	namespace := uniqueKey(t, "chained-large") + ":"

	original := []byte(generateRepeatableText(5 * 1024 * 1024))

	wrapped := transformAndWrap(t, original, transformers)

	redisKey := rawRedisKeyForNamespace(namespace, "large-file")
	putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

	raw := getRawBytesFromRedis(t, redisKey)

	t.Logf("Original: %d bytes, Wrapped: %d bytes, Ratio: %.2f%%",
		len(original), len(raw), float64(len(raw))/float64(len(original))*100)

	recovered := reverseAndUnwrap(t, raw, transformers)
	assert.Equal(t, original, recovered, "large file chained round-trip mismatch")
}

func TestChained_AllSetups_RoundTrip(t *testing.T) {
	t.Parallel()

	setups := buildTransformerSetups(t)
	namespace := uniqueKey(t, "chained-all") + ":"

	for _, setup := range setups {
		t.Run(setup.name, func(t *testing.T) {
			t.Parallel()

			original := []byte(generateRepeatableText(2048))

			wrapped := transformAndWrap(t, original, setup.transformers)

			redisKey := rawRedisKeyForNamespace(namespace, setup.name)
			putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

			raw := getRawBytesFromRedis(t, redisKey)

			tv, err := cache_domain.UnmarshalTransformedValue(raw)
			require.NoError(t, err, "unmarshalling TransformedValue for %s", setup.name)

			assert.Equal(t, setup.enabledNames, tv.Transformers,
				"transformer names mismatch for %s", setup.name)

			recovered := reverseAndUnwrap(t, raw, setup.transformers)
			assert.Equal(t, original, recovered, "round-trip mismatch for %s", setup.name)
		})
	}
}
