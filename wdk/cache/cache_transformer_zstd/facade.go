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

package cache_transformer_zstd

import (
	"piko.sh/piko/internal/cache/cache_adapters/cache_transformer_zstd"
	"piko.sh/piko/wdk/cache"
)

// Config holds settings for the zstd cache transformer.
// This is re-exported from the internal adapter package.
type Config = cache_transformer_zstd.Config

// Option configures a zstd cache transformer at construction time. It is
// re-exported from the internal adapter package so callers can use the
// functional-options pattern without importing internal packages.
type Option = cache_transformer_zstd.Option

// DefaultMaxDecompressedCacheBytes is the default cap on the decompressed
// output produced by Reverse. Re-exported from the internal adapter package.
const DefaultMaxDecompressedCacheBytes = cache_transformer_zstd.DefaultMaxDecompressedCacheBytes

// ErrDecompressedCacheTooLarge is surfaced by Reverse when a decompressed
// payload exceeds the configured cap.
//
// Use errors.Is to detect this condition. Re-exported from the internal
// adapter package.
var ErrDecompressedCacheTooLarge = cache_transformer_zstd.ErrDecompressedCacheTooLarge

// WithMaxDecompressedCacheBytes sets the maximum number of decompressed bytes
// produced by Reverse before ErrDecompressedCacheTooLarge is surfaced. A
// non-positive value disables the cap.
//
// Takes maxBytes (int64) which is the cap in bytes; non-positive disables.
//
// Returns Option which applies the cap to a transformer.
func WithMaxDecompressedCacheBytes(maxBytes int64) Option {
	return cache_transformer_zstd.WithMaxDecompressedCacheBytes(maxBytes)
}

// New creates a new zstd cache compression transformer.
//
// Takes config (Config) which specifies the compression settings including name,
// priority, and compression level.
// Takes options (...Option) which override settings on the constructed
// transformer (e.g. WithMaxDecompressedCacheBytes).
//
// Returns cache.TransformerPort which is the configured transformer ready
// for use.
// Returns error when the transformer cannot be created.
//
// Example:
//
//	config := cache_transformer_zstd.Config{
//	    Name:     "zstd",
//	    Priority: 100,
//	    Level:    zstd.SpeedDefault,
//	}
//	transformer, err := cache_transformer_zstd.New(config)
func New(config Config, options ...Option) (cache.TransformerPort, error) {
	return cache_transformer_zstd.NewZstdCacheTransformer(config, options...)
}

// DefaultConfig returns sensible defaults for zstd compression.
//
// Returns Config which contains the default compression settings.
//
// Example:
//
//	config := cache_transformer_zstd.DefaultConfig()
//	transformer, err := cache_transformer_zstd.New(config)
func DefaultConfig() Config {
	return cache_transformer_zstd.DefaultConfig()
}
