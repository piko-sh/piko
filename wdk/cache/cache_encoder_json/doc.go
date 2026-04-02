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

// Package cache_encoder_json provides JSON-based encoding for cache
// values.
//
// JSON is the default encoder: human-readable, cross-language
// compatible, and works with any type that has exported fields or
// implements json.Marshaler. It is slower and larger on the wire than
// binary formats, but the interoperability trade-off is usually worth
// it.
//
// This package implements [cache.EncoderPort] and works with any cache
// provider that supports encoding registries.
package cache_encoder_json
