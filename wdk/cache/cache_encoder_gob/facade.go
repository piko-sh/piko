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

package cache_encoder_gob

import (
	"piko.sh/piko/internal/cache/cache_adapters/encoder_gob"
	"piko.sh/piko/wdk/cache"
)

// ErrGobInputTooLarge is surfaced by Unmarshal when a gob payload exceeds
// the configured cap.
//
// Use errors.Is to detect this condition. Re-exported from the internal
// adapter package.
var ErrGobInputTooLarge = encoder_gob.ErrGobInputTooLarge

// SetMaxGobInputBytes overrides the maximum byte size accepted when decoding
// a gob payload. A value of zero or below disables the cap, which is not
// recommended for attacker-influenced cache contents.
//
// Takes maxBytes (int64) which is the new cap in bytes.
func SetMaxGobInputBytes(maxBytes int64) {
	encoder_gob.SetMaxGobInputBytes(maxBytes)
}

// MaxGobInputBytes returns the active byte-size cap enforced when decoding
// gob payloads.
//
// Returns int64 which is the current cap; a value of zero indicates no cap
// is enforced.
func MaxGobInputBytes() int64 {
	return encoder_gob.MaxGobInputBytes()
}

// New creates a Gob encoder for any given type V.
//
// Decoding enforces a configurable input-size cap (see SetMaxGobInputBytes)
// so a malicious or corrupt gob stream cannot trigger unbounded allocations.
//
// Returns EncoderPort[V] which handles Gob encoding and decoding for the
// given type.
//
// Example:
//
//	userEncoder := cache_encoder_gob.New[User]()
//	registry.Register(userEncoder)
func New[V any]() cache.EncoderPort[V] {
	return encoder_gob.New[V]()
}
