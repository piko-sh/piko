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

package encoder_json

import (
	"piko.sh/piko/internal/cache/cache_domain"
)

// New creates a JSON encoder for any given type V, using CacheAPI with
// CopyString enabled to prevent memory issues when decoded objects outlive
// the original JSON buffer.
//
// Returns cache_domain.EncoderPort[V] which is a JSON encoder for
// the specified type.
func New[V any]() cache_domain.EncoderPort[V] {
	return cache_domain.NewEncoder(
		func(value V) ([]byte, error) {
			return cache_domain.CacheAPI.Marshal(value)
		},
		func(data []byte, target *V) error {
			return cache_domain.CacheAPI.Unmarshal(data, target)
		},
	)
}
