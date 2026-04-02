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

//go:build bench

package cache_bench_test

import (
	"context"
	"testing"

	"piko.sh/piko/internal/cache/cache_adapters/cache_transformer_crypto"
	"piko.sh/piko/internal/cache/cache_adapters/cache_transformer_zstd"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/crypto/crypto_adapters/local_aes_gcm"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func newBenchCryptoEncryptor(b *testing.B) cache_domain.CacheTransformerPort {
	b.Helper()

	key := make([]byte, local_aes_gcm.KeySize)
	copy(key, "benchmark-key-material-32bytes!")

	provider, err := local_aes_gcm.NewProvider(local_aes_gcm.Config{
		Key:   key,
		KeyID: "bench-key-1",
	})
	if err != nil {
		b.Fatalf("failed to create crypto provider: %v", err)
	}

	service, err := crypto_domain.NewCryptoService(context.Background(), nil, &crypto_dto.ServiceConfig{
		ActiveKeyID:              "bench-key-1",
		ProviderType:             crypto_dto.ProviderTypeLocalAESGCM,
		EnableEnvelopeEncryption: false,
	})
	if err != nil {
		b.Fatalf("failed to create crypto service: %v", err)
	}

	if err := service.RegisterProvider(context.Background(), "local-aes-gcm", provider); err != nil {
		b.Fatalf("failed to register provider: %v", err)
	}

	if err := service.SetDefaultProvider("local-aes-gcm"); err != nil {
		b.Fatalf("failed to set default provider: %v", err)
	}

	return cache_transformer_crypto.New(service, "crypto-service", 250)
}

func BenchmarkTransformer_Compression(b *testing.B) {
	ctx := context.Background()

	compressor, err := cache_transformer_zstd.NewZstdCacheTransformer(cache_transformer_zstd.Config{})
	if err != nil {
		b.Fatalf("failed to create compressor: %v", err)
	}

	for _, data := range dataSizes() {
		b.Run(data.Name, func(b *testing.B) {
			byteData := generateByteData(data.Size)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {

				compressed, err := compressor.Transform(ctx, byteData, nil)
				if err != nil {
					b.Fatalf("compression failed: %v", err)
				}
				_, err = compressor.Reverse(ctx, compressed, nil)
				if err != nil {
					b.Fatalf("decompression failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkTransformer_Encryption(b *testing.B) {
	ctx := context.Background()

	encryptor := newBenchCryptoEncryptor(b)

	for _, data := range dataSizes() {
		b.Run(data.Name, func(b *testing.B) {
			byteData := generateByteData(data.Size)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {

				encrypted, err := encryptor.Transform(ctx, byteData, nil)
				if err != nil {
					b.Fatalf("encryption failed: %v", err)
				}
				_, err = encryptor.Reverse(ctx, encrypted, nil)
				if err != nil {
					b.Fatalf("decryption failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkTransformer_CompressionOnly_Transform(b *testing.B) {
	ctx := context.Background()

	compressor, err := cache_transformer_zstd.NewZstdCacheTransformer(cache_transformer_zstd.Config{})
	if err != nil {
		b.Fatalf("failed to create compressor: %v", err)
	}

	for _, data := range dataSizes() {
		b.Run(data.Name, func(b *testing.B) {
			byteData := generateByteData(data.Size)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, err := compressor.Transform(ctx, byteData, nil)
				if err != nil {
					b.Fatalf("compression failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkTransformer_CompressionOnly_Reverse(b *testing.B) {
	ctx := context.Background()

	compressor, err := cache_transformer_zstd.NewZstdCacheTransformer(cache_transformer_zstd.Config{})
	if err != nil {
		b.Fatalf("failed to create compressor: %v", err)
	}

	for _, data := range dataSizes() {
		b.Run(data.Name, func(b *testing.B) {
			byteData := generateByteData(data.Size)

			compressed, err := compressor.Transform(ctx, byteData, nil)
			if err != nil {
				b.Fatalf("compression failed: %v", err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, err := compressor.Reverse(ctx, compressed, nil)
				if err != nil {
					b.Fatalf("decompression failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkTransformer_Pipeline(b *testing.B) {
	ctx := context.Background()

	compressor, err := cache_transformer_zstd.NewZstdCacheTransformer(cache_transformer_zstd.Config{})
	if err != nil {
		b.Fatalf("failed to create compressor: %v", err)
	}
	encryptor := newBenchCryptoEncryptor(b)

	for _, data := range dataSizes() {
		b.Run(data.Name, func(b *testing.B) {
			byteData := generateByteData(data.Size)

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {

				compressed, err := compressor.Transform(ctx, byteData, nil)
				if err != nil {
					b.Fatalf("compression failed: %v", err)
				}
				encrypted, err := encryptor.Transform(ctx, compressed, nil)
				if err != nil {
					b.Fatalf("encryption failed: %v", err)
				}

				decrypted, err := encryptor.Reverse(ctx, encrypted, nil)
				if err != nil {
					b.Fatalf("decryption failed: %v", err)
				}
				_, err = compressor.Reverse(ctx, decrypted, nil)
				if err != nil {
					b.Fatalf("decompression failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkTransformer_CompressionRatio(b *testing.B) {
	ctx := context.Background()

	compressor, err := cache_transformer_zstd.NewZstdCacheTransformer(cache_transformer_zstd.Config{})
	if err != nil {
		b.Fatalf("failed to create compressor: %v", err)
	}

	for _, data := range dataSizes() {
		byteData := generateByteData(data.Size)
		compressed, err := compressor.Transform(ctx, byteData, nil)
		if err != nil {
			b.Fatalf("compression failed: %v", err)
		}

		ratio := float64(len(compressed)) / float64(len(byteData)) * 100
		b.Logf("%s: Original=%d bytes, Compressed=%d bytes, Ratio=%.2f%%",
			data.Name, len(byteData), len(compressed), ratio)
	}
}
