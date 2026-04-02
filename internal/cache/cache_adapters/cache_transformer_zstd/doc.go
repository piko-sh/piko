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

// Package cache_transformer_zstd implements the cache transformer port
// using Zstandard compression.
//
// This adapter compresses and decompresses cached values using the zstd
// algorithm. It provides good compression ratios with fast decompression. It
// self-registers under the name "zstd" so the cache builder can
// instantiate it from configuration.
//
// The compression level is configurable, ranging from fastest (level 1)
// to best compression (level 11). Priority determines execution order in
// the transformer pipeline; the recommended range for compression
// transformers is 100-199.
//
// # Thread safety
//
// The underlying zstd encoder and decoder are safe for concurrent use.
// A single transformer instance may be shared across goroutines.
package cache_transformer_zstd
