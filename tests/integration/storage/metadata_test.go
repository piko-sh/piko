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
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
)

func TestMetadata_SingleTransformer(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	testCases := []struct {
		name         string
		transformers []storage_domain.StreamTransformerPort
		enabled      []string
		wantMeta     string
	}{
		{
			name:         "gzip",
			transformers: singleTransformer(newGzipTransformer(t, gzip.DefaultCompression)),
			enabled:      []string{"gzip"},
			wantMeta:     `["gzip"]`,
		},
		{
			name:         "zstd",
			transformers: singleTransformer(newZstdTransformer(t)),
			enabled:      []string{"zstd"},
			wantMeta:     `["zstd"]`,
		},
		{
			name:         "crypto",
			transformers: singleTransformer(crypto.transformer),
			enabled:      []string{"crypto-service"},
			wantMeta:     `["crypto-service"]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := generateRepeatableText(256)
			key := uniqueKey(t, "meta-single")
			wrapper := newTransformerWrapper(t, provider, tc.transformers, tc.enabled)

			putObject(ctx, t, wrapper, key, original)

			metadata := getObjectMetadataFromS3(ctx, t, testBucketPrimary, key)
			value, ok := metadata["x-piko-transformers"]
			require.True(t, ok, "metadata should contain x-piko-transformers key")
			assert.JSONEq(t, tc.wantMeta, value,
				"transformer metadata should match expected JSON")
		})
	}
}

func TestMetadata_ChainedTransformers(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	testCases := []struct {
		name         string
		transformers []storage_domain.StreamTransformerPort
		enabled      []string
		wantMeta     string
	}{
		{
			name:         "gzip_then_crypto",
			transformers: []storage_domain.StreamTransformerPort{newGzipTransformer(t, gzip.DefaultCompression), crypto.transformer},
			enabled:      []string{"gzip", "crypto-service"},
			wantMeta:     `["gzip","crypto-service"]`,
		},
		{
			name:         "zstd_then_crypto",
			transformers: []storage_domain.StreamTransformerPort{newZstdTransformer(t), crypto.transformer},
			enabled:      []string{"zstd", "crypto-service"},
			wantMeta:     `["zstd","crypto-service"]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := generateRepeatableText(256)
			key := uniqueKey(t, "meta-chained")
			wrapper := newTransformerWrapper(t, provider, tc.transformers, tc.enabled)

			putObject(ctx, t, wrapper, key, original)

			metadata := getObjectMetadataFromS3(ctx, t, testBucketPrimary, key)
			value, ok := metadata["x-piko-transformers"]
			require.True(t, ok, "metadata should contain x-piko-transformers key")
			assert.JSONEq(t, tc.wantMeta, value,
				"chained transformer metadata should be ordered by priority")
		})
	}
}

func TestMetadata_AutoDetection(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "meta-autodetect")

	uploadWrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})
	putObject(ctx, t, uploadWrapper, key, original)

	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(gzipT))
	require.NoError(t, registry.Register(crypto.transformer))
	downloadWrapper := storage_domain.NewTransformerWrapper(provider, registry, nil, "s3-test")

	retrieved := getObject(ctx, t, downloadWrapper, key)
	assert.Equal(t, original, retrieved,
		"auto-detected transformer chain should correctly reverse transformations")
}

func TestMetadata_NoTransformers(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	original := generateRepeatableText(256)
	key := uniqueKey(t, "meta-none")

	wrapper := newTransformerWrapper(t, provider, nil, nil)
	putObject(ctx, t, wrapper, key, original)

	metadata := getObjectMetadataFromS3(ctx, t, testBucketPrimary, key)
	_, ok := metadata["x-piko-transformers"]
	assert.False(t, ok, "metadata should NOT contain x-piko-transformers when no transformers used")
}
