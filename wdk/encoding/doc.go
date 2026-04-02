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

// Package encoding provides base-X encoding utilities and UUID
// generation.
//
// The [Encoding] type supports both byte slice and uint64 encoding
// with generic base-N alphabets. When a standard alphabet is
// detected (Base64, Base32, Hex), the encoder automatically
// delegates to the optimised standard library. Convenience
// wrappers for Base36 and Base58 are provided as package-level
// functions.
//
// The package also provides UUID v7 generation with custom
// timestamp support, and functions for encoding UUIDs to compact
// string representations.
//
// All pre-initialised encodings and the [Encoding] type methods
// are safe for concurrent use.
package encoding
