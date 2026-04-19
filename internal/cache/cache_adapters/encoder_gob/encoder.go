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

package encoder_gob

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"sync/atomic"

	"piko.sh/piko/internal/cache/cache_domain"
)

// defaultMaxGobInputBytes is the default cap on the encoded gob payload size
// accepted by Unmarshal. Set to 64 MiB so legitimate cache payloads are
// unaffected while still preventing pathological gob streams from declaring
// multi-gigabyte buffers and exhausting memory during decoding.
const defaultMaxGobInputBytes int64 = 64 * 1024 * 1024

// ErrGobInputTooLarge is returned when a gob payload supplied to Unmarshal
// exceeds the configured maximum input size. Callers can use errors.Is to
// distinguish this from ordinary gob decode failures.
var ErrGobInputTooLarge = errors.New("encoder_gob: gob input exceeds maximum size")

// maxGobInputBytes holds the active maximum gob input size in bytes. It is
// initialised at package load to defaultMaxGobInputBytes and may be overridden
// via SetMaxGobInputBytes for tests or operator-controlled tuning.
var maxGobInputBytes atomic.Int64

func init() {
	maxGobInputBytes.Store(defaultMaxGobInputBytes)
}

// SetMaxGobInputBytes overrides the maximum byte size accepted when decoding
// a gob payload. A value of zero or below disables the cap, which is not
// recommended for attacker-influenced cache contents.
//
// Takes maxBytes (int64) which is the new cap in bytes.
func SetMaxGobInputBytes(maxBytes int64) {
	if maxBytes < 0 {
		maxBytes = 0
	}
	maxGobInputBytes.Store(maxBytes)
}

// MaxGobInputBytes returns the active byte-size cap enforced when decoding
// gob payloads.
//
// Returns int64 which is the current cap; a value of zero indicates no cap
// is enforced.
func MaxGobInputBytes() int64 {
	return maxGobInputBytes.Load()
}

// New creates a Gob encoder for any given type V using Go's native binary
// encoding format, which outperforms JSON for complex Go types but is
// Go-specific and not human-readable.
//
// Decoding enforces a configurable input-size cap (see SetMaxGobInputBytes)
// so a malicious or corrupt gob stream cannot trigger unbounded allocations
// during decode.
//
// Best used for:
//   - Complex structs with nested fields
//   - Types with unexported fields (if registered properly)
//   - Performance-critical paths where JSON overhead is too high
//
// Returns cache_domain.EncoderPort[V] which is a Gob encoder for
// the specified type.
func New[V any]() cache_domain.EncoderPort[V] {
	return cache_domain.NewEncoder(
		func(value V) ([]byte, error) {
			var buffer bytes.Buffer
			encoder := gob.NewEncoder(&buffer)
			if err := encoder.Encode(value); err != nil {
				return nil, fmt.Errorf("gob-encoding value: %w", err)
			}
			return buffer.Bytes(), nil
		},
		func(data []byte, target *V) error {
			if maxBytes := maxGobInputBytes.Load(); maxBytes > 0 && int64(len(data)) > maxBytes {
				return fmt.Errorf("%w: input is %d bytes, cap %d",
					ErrGobInputTooLarge, len(data), maxBytes)
			}
			buffer := bytes.NewBuffer(data)
			decoder := gob.NewDecoder(buffer)
			return decoder.Decode(target)
		},
	)
}
