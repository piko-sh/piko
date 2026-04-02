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

// Package wal_domain defines the core abstractions and business types for
// write-ahead logging.
//
// It defines port interfaces ([WAL], [SnapshotStore]) and codec
// interfaces for binary serialisation used to persist cache operations
// to disk. Entries are written to the log before being applied; on
// recovery the log is replayed to reconstruct state. Snapshots
// complement the WAL with periodic point-in-time copies that reduce
// recovery time.
//
// # Thread safety
//
// Implementations of the [WAL] interface must be safe for concurrent
// [WAL.Append] calls. The [WAL.Recover] iterator holds a lock and
// should be consumed promptly.
package wal_domain
