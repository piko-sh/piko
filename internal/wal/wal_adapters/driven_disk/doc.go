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

// Package driven_disk implements disk-based write-ahead log and snapshot
// storage.
//
// It implements [wal_domain.WAL] and [wal_domain.SnapshotStore] using a
// compact binary format with CRC32 integrity checks. A group commit
// pattern handles high-throughput concurrent writes, and snapshots
// support optional zstd compression. All filesystem operations are
// sandboxed via [safedisk.Sandbox].
//
// # Wire format
//
// Each WAL entry on disk is stored as:
//
//	[Length:4][CRC32:4][Version:1][Operation:1][Timestamp:8]
//	[ExpiresAt:8][KeyLen:4][Key:var][ValueLen:4][Value:var]
//	[TagCount:2][Tags:var]
//
// All multi-byte fields use big-endian encoding.
//
// # Thread safety
//
// [DiskWAL] and [DiskSnapshot] are safe for concurrent use. DiskWAL
// uses a dedicated commit goroutine to batch and serialise writes,
// minimising lock contention and fsync overhead.
package driven_disk
