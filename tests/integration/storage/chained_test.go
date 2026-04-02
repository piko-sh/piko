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
	"io"
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
)

func TestChained_GzipThenCrypto_RoundTrip(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	original := generateRepeatableText(1024 * 1024)
	key := uniqueKey(t, "chained-gzip-crypto-rt")

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)

	assert.True(t, isStreamingEnvelope(raw),
		"raw bytes should start with crypto envelope (outermost layer)")
	assert.False(t, isValidGzip(raw),
		"raw bytes should NOT start with gzip magic (gzip is inside the crypto layer)")
	assert.True(t, isNotPlaintext(raw, original),
		"raw bytes should not contain plaintext")

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved, "round-trip should recover original content")
}

func TestChained_ZstdThenCrypto_RoundTrip(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	zstdT := newZstdTransformer(t)

	original := generateRepeatableText(1024 * 1024)
	key := uniqueKey(t, "chained-zstd-crypto-rt")

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{zstdT, crypto.transformer},
		[]string{"zstd", "crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	assert.True(t, isStreamingEnvelope(raw),
		"raw bytes should start with crypto envelope")
	assert.False(t, isValidZstd(raw),
		"raw bytes should NOT start with zstd magic")

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved, "round-trip should recover original content")
}

func TestChained_VerifyLayerOrder_GzipCrypto(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "chained-layer-order-gzip")

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)

	decryptReader, err := crypto.service.DecryptStream(ctx, bytes.NewReader(raw))
	require.NoError(t, err)
	defer func() { _ = decryptReader.Close() }()

	decrypted, err := io.ReadAll(decryptReader)
	require.NoError(t, err)

	assert.True(t, isValidGzip(decrypted),
		"decrypted inner layer should be valid gzip (proves chain: original -> gzip -> encrypt)")

	gzipReader, err := gzip.NewReader(bytes.NewReader(decrypted))
	require.NoError(t, err)
	defer func() { _ = gzipReader.Close() }()

	decompressed, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed,
		"fully unwrapped content should match original")
}

func TestChained_VerifyLayerOrder_ZstdCrypto(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	zstdT := newZstdTransformer(t)

	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "chained-layer-order-zstd")

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{zstdT, crypto.transformer},
		[]string{"zstd", "crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)

	decryptReader, err := crypto.service.DecryptStream(ctx, bytes.NewReader(raw))
	require.NoError(t, err)
	defer func() { _ = decryptReader.Close() }()

	decrypted, err := io.ReadAll(decryptReader)
	require.NoError(t, err)

	assert.True(t, isValidZstd(decrypted),
		"decrypted inner layer should be valid zstd (proves chain: original -> zstd -> encrypt)")

	decoder, err := zstd.NewReader(bytes.NewReader(decrypted))
	require.NoError(t, err)
	defer decoder.Close()

	decompressed, err := io.ReadAll(decoder)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed,
		"fully unwrapped content should match original")
}

func TestChained_LargeFile(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	original := generateRepeatableText(10 * 1024 * 1024)
	key := uniqueKey(t, "chained-large")

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved,
		"10MB file should round-trip correctly through gzip+crypto chain")
}

func TestChained_VariousSizes(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	testCases := []struct {
		name string
		size int
	}{
		{name: "1B", size: 1},
		{name: "100B", size: 100},
		{name: "10KB", size: 10 * 1024},
		{name: "64KB_chunk_boundary", size: 64 * 1024},
		{name: "64KB_plus_one", size: 64*1024 + 1},
		{name: "1MB", size: 1024 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := generateRepeatableText(tc.size)
			key := uniqueKey(t, "chained-sizes")

			wrapper := newTransformerWrapper(t, provider,
				[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
				[]string{"gzip", "crypto-service"})

			putObject(ctx, t, wrapper, key, original)

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved,
				"size %d should round-trip correctly through gzip+crypto", tc.size)
		})
	}
}
