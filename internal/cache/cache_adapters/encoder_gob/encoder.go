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
	"fmt"

	"piko.sh/piko/internal/cache/cache_domain"
)

// New creates a Gob encoder for any given type V using Go's native binary
// encoding format, which outperforms JSON for complex Go types but is
// Go-specific and not human-readable.
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
			buffer := bytes.NewBuffer(data)
			decoder := gob.NewDecoder(buffer)
			return decoder.Decode(target)
		},
	)
}
