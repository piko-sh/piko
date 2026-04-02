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
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/storage/storage_domain"
)

type transformerSetup struct {
	name         string
	transformers []storage_domain.StreamTransformerPort
	enabled      []string
}

func buildTransformerSetups(t *testing.T) []transformerSetup {
	t.Helper()
	crypto := newCryptoSetup(t)
	return []transformerSetup{
		{
			name:         "gzip_only",
			transformers: singleTransformer(newGzipTransformer(t, gzip.DefaultCompression)),
			enabled:      []string{"gzip"},
		},
		{
			name:         "zstd_only",
			transformers: singleTransformer(newZstdTransformer(t)),
			enabled:      []string{"zstd"},
		},
		{
			name:         "crypto_only",
			transformers: singleTransformer(crypto.transformer),
			enabled:      []string{"crypto-service"},
		},
		{
			name:         "gzip_and_crypto",
			transformers: []storage_domain.StreamTransformerPort{newGzipTransformer(t, gzip.DefaultCompression), crypto.transformer},
			enabled:      []string{"gzip", "crypto-service"},
		},
		{
			name:         "zstd_and_crypto",
			transformers: []storage_domain.StreamTransformerPort{newZstdTransformer(t), crypto.transformer},
			enabled:      []string{"zstd", "crypto-service"},
		},
	}
}

func TestEdge_EmptyFile(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	for _, setup := range buildTransformerSetups(t) {
		t.Run(setup.name, func(t *testing.T) {
			original := []byte{}
			key := uniqueKey(t, "edge-empty-file")
			wrapper := newTransformerWrapper(t, provider, setup.transformers, setup.enabled)

			putObject(ctx, t, wrapper, key, original)

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Empty(t, retrieved, "empty file should round-trip correctly")
		})
	}
}

func TestEdge_SingleByte(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	for _, setup := range buildTransformerSetups(t) {
		t.Run(setup.name, func(t *testing.T) {
			original := []byte{0x42}
			key := uniqueKey(t, "edge-single-byte")
			wrapper := newTransformerWrapper(t, provider, setup.transformers, setup.enabled)

			putObject(ctx, t, wrapper, key, original)

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved, "single byte should round-trip correctly")
		})
	}
}

func TestEdge_ChunkBoundary(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	testCases := []struct {
		name string
		size int
	}{
		{name: "exactly_64KB", size: crypto_dto.DefaultChunkSize},
		{name: "64KB_plus_one", size: crypto_dto.DefaultChunkSize + 1},
		{name: "two_chunks", size: 2 * crypto_dto.DefaultChunkSize},
		{name: "two_chunks_plus_one", size: 2*crypto_dto.DefaultChunkSize + 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := generateRepeatableText(tc.size)
			key := uniqueKey(t, "edge-chunk-boundary")

			wrapper := newTransformerWrapper(t, provider,
				[]storage_domain.StreamTransformerPort{
					newGzipTransformer(t, gzip.DefaultCompression),
					crypto.transformer,
				},
				[]string{"gzip", "crypto-service"})

			putObject(ctx, t, wrapper, key, original)

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved,
				"chunk boundary size %d should round-trip correctly", tc.size)
		})
	}
}

func TestEdge_HighlyCompressible(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	original := make([]byte, 1024*1024)
	key := uniqueKey(t, "edge-compressible")

	transformer := newGzipTransformer(t, gzip.DefaultCompression)
	wrapper := newTransformerWrapper(t, provider, singleTransformer(transformer), []string{"gzip"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	ratio := float64(len(raw)) / float64(len(original))
	assert.Less(t, ratio, 0.01,
		"1MB of zeros should compress to less than 1%% (got %.4f%%)", ratio*100)

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved)
}

func TestEdge_Incompressible(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	original := generateRandomData(t, 100*1024)
	key := uniqueKey(t, "edge-incompressible")

	transformer := newGzipTransformer(t, gzip.DefaultCompression)
	wrapper := newTransformerWrapper(t, provider, singleTransformer(transformer), []string{"gzip"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	assert.True(t, isValidGzip(raw), "incompressible data should still produce valid gzip")

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved, "incompressible data round-trip should be exact")
}

func TestEdge_UnicodeContent(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	original := []byte("Hello World! " +
		"\xf0\x9f\x8c\x8d\xf0\x9f\x8e\x89\xf0\x9f\x9a\x80 " +
		"\xe4\xb8\xad\xe6\x96\x87\xe6\xb5\x8b\xe8\xaf\x95 " +
		"\xc3\xa9\xc3\xa0\xc3\xbc\xc3\xb1 " +
		"\xd0\x9f\xd1\x80\xd0\xb8\xd0\xb2\xd0\xb5\xd1\x82")

	for _, setup := range buildTransformerSetups(t) {
		t.Run(setup.name, func(t *testing.T) {
			key := uniqueKey(t, "edge-unicode")
			wrapper := newTransformerWrapper(t, provider, setup.transformers, setup.enabled)

			putObject(ctx, t, wrapper, key, original)

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved,
				"multi-byte UTF-8 content should be preserved exactly")
		})
	}
}

func TestEdge_NullBytes(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	original := make([]byte, 1024)
	for i := range original {
		if i%3 == 0 {
			original[i] = 0x00
		} else {
			original[i] = byte(i % 256)
		}
	}

	for _, setup := range buildTransformerSetups(t) {
		t.Run(setup.name, func(t *testing.T) {
			key := uniqueKey(t, "edge-null-bytes")
			wrapper := newTransformerWrapper(t, provider, setup.transformers, setup.enabled)

			putObject(ctx, t, wrapper, key, original)

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved,
				"data with null bytes should be preserved exactly")
		})
	}
}

func TestEdge_LargeFile(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	zstdT := newZstdTransformer(t)

	original := generateRandomData(t, 20*1024*1024)
	key := uniqueKey(t, "edge-large-file")

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{zstdT, crypto.transformer},
		[]string{"zstd", "crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved,
		"20MB random data should round-trip correctly through zstd+crypto")
}
