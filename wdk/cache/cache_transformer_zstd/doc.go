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

// Package cache_transformer_zstd provides Zstandard compression
// transformation for cache values.
//
// Zstd typically achieves better compression than gzip with faster
// decompression. It is most effective for values larger than about
// 1 KB. The compression level is configurable from SpeedFastest (1)
// through SpeedBestCompression (11), with SpeedDefault (3) offering
// a balanced trade-off.
//
// When combining with encryption, register compression at a lower
// priority number so it runs first.
//
// All methods are safe for concurrent use.
package cache_transformer_zstd
