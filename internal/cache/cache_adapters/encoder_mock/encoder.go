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

package encoder_mock

import (
	"fmt"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/cache/cache_domain"
)

// New creates a mock encoder for testing purposes.
// The mock encoder wraps JSON encoding with a recognisable prefix,
// so tests can verify that the correct encoder was used.
//
// Format: "MOCK:<type>:<json_data>"
//
// Returns cache_domain.EncoderPort[V] which is a mock encoder for
// the specified type.
func New[V any]() cache_domain.EncoderPort[V] {
	var v V
	typeName := fmt.Sprintf("%T", v)

	return cache_domain.NewEncoder(
		func(value V) ([]byte, error) {
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			prefixed := fmt.Sprintf("MOCK:%s:%s", typeName, string(jsonBytes))
			return []byte(prefixed), nil
		},
		func(data []byte, target *V) error {
			str := string(data)
			expectedPrefix := fmt.Sprintf("MOCK:%s:", typeName)

			if len(str) < len(expectedPrefix) {
				return fmt.Errorf("mock encoder: data too short, expected prefix %q", expectedPrefix)
			}

			actualPrefix := str[:len(expectedPrefix)]
			if actualPrefix != expectedPrefix {
				return fmt.Errorf("mock encoder: expected prefix %q, got %q", expectedPrefix, actualPrefix)
			}

			jsonData := str[len(expectedPrefix):]
			return json.Unmarshal([]byte(jsonData), target)
		},
	)
}
