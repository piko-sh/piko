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

package provider_otter

import "piko.sh/piko/internal/wal/wal_domain"

// PersistenceConfig configures optional WAL-based persistence for the Otter cache.
//
// When enabled, the cache persists all writes to a Write-Ahead Log (WAL)
// and periodically creates snapshots for fast recovery, letting the cache
// survive process restarts without losing data.
type PersistenceConfig[K comparable, V any] struct {
	// KeyCodec handles serialisation of cache keys to bytes.
	// Required when Enabled is true.
	KeyCodec wal_domain.KeyCodec[K]

	// ValueCodec handles serialisation of cache values to bytes.
	// Required when Enabled is true.
	ValueCodec wal_domain.ValueCodec[V]

	// WALConfig configures the WAL and snapshot behaviour.
	// See wal_domain.Config for details.
	WALConfig wal_domain.Config

	// Enabled controls whether persistence is active; when false, the cache
	// operates purely in-memory. Default: false.
	Enabled bool
}

// Validate reports whether the persistence configuration is correct.
//
// Returns error when persistence is enabled but codecs are missing or the WAL
// configuration is invalid.
func (c PersistenceConfig[K, V]) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.KeyCodec == nil {
		return wal_domain.ErrCodecRequired
	}
	if c.ValueCodec == nil {
		return wal_domain.ErrCodecRequired
	}

	return c.WALConfig.Validate()
}
