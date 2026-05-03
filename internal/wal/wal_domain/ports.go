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

package wal_domain

import (
	"context"
	"iter"
)

// WAL is the driven port for write-ahead logging.
// Implementations must be safe for concurrent Append calls.
//
// The WAL provides crash-resistant persistence by writing operations to a log
// before they are applied. On recovery, the log is replayed to restore state.
//
// Key guarantees:
//   - Entries are written atomically (either fully written or not at all)
//   - CRC32 validation detects any corruption from partial writes
//   - On corruption, the WAL is truncated at the corruption point
//   - Data before corruption is preserved; data after is lost
type WAL[K comparable, V any] interface {
	// Append writes an entry to the log.
	//
	// The entry is encoded with a CRC32 checksum for integrity validation.
	// Depending on SyncMode, this may block until the data is durably written.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes entry (Entry[K, V]) which is the operation to log.
	//
	// Returns error when encoding fails or the write cannot complete.
	Append(ctx context.Context, entry Entry[K, V]) error

	// Recover returns an iterator over valid entries without loading all into
	// memory.
	//
	// The iterator yields entries one at a time and handles corruption by stopping
	// at the first corrupt entry. Entries are validated via CRC32. When corruption
	// is detected, the WAL is truncated at that point after iteration completes.
	//
	// Returns iter.Seq2[Entry[K, V], error] which yields entries and any error.
	// The iteration stops on first error or when all entries are consumed.
	//
	// The iterator holds a lock on the WAL. Callers should consume all entries
	// promptly or break out of the loop to release the lock.
	Recover(ctx context.Context) iter.Seq2[Entry[K, V], error]

	// Truncate removes all entries from the log.
	//
	// This is typically called after a successful snapshot to prevent the
	// WAL from growing unbounded.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	//
	// Returns error when the truncation fails.
	Truncate(ctx context.Context) error

	// Sync forces all buffered data to stable storage.
	//
	// This performs an fsync on the underlying file, ensuring all previously
	// written entries are durably stored.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	//
	// Returns error when the sync fails.
	Sync(ctx context.Context) error

	// Close releases resources held by the WAL.
	//
	// Any pending writes are flushed before closing. After Close returns,
	// all other methods will return ErrWALClosed.
	//
	// Returns error when the close fails.
	Close() error

	// EntryCount returns the number of entries currently in the WAL.
	// This is approximate and may not reflect entries not yet synced.
	EntryCount() int

	// Size returns the current size of the WAL file in bytes.
	Size() int64
}

// SnapshotStore is the driven port for snapshot persistence.
//
// Snapshots provide a point-in-time copy of the entire cache state.
// They are used to speed up recovery by avoiding WAL replay from the beginning.
type SnapshotStore[K comparable, V any] interface {
	// Save persists a complete snapshot of all entries.
	//
	// The snapshot is written atomically using a temp file and rename pattern.
	// If compression is enabled, the data is compressed with zstd.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	// Takes entries ([]Entry[K, V]) which is the complete cache state.
	//
	// Returns error when the snapshot cannot be written.
	Save(ctx context.Context, entries []Entry[K, V]) error

	// Load returns an iterator over snapshot entries without loading all into
	// memory.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	//
	// Returns iter.Seq2[Entry[K, V], error] yielding entries and any error. The
	// iteration stops on first error or when all entries are consumed. Returns
	// ErrSnapshotNotFound if no snapshot exists, which is not necessarily an
	// error as it may indicate a fresh start.
	//
	// Concurrency: The iterator holds a lock on the snapshot store. Callers should
	// consume all entries promptly or break out of the loop to release the lock.
	//
	// Note: Compressed snapshots must decompress fully before streaming entries
	// due to zstd's block-based decompression. Uncompressed snapshots stream
	// directly from disk.
	Load(ctx context.Context) iter.Seq2[Entry[K, V], error]

	// Delete removes the snapshot file.
	//
	// Takes ctx (context.Context) for cancellation and timeout.
	//
	// Returns error when deletion fails (ErrSnapshotNotFound is not returned
	// if the file doesn't exist).
	Delete(ctx context.Context) error

	// Close releases resources held by the snapshot store.
	//
	// Returns error when the close fails.
	Close() error

	// Exists returns true if a snapshot file exists.
	Exists() bool
}

// Codec handles binary serialisation of WAL entries.
//
// The codec is responsible for converting entries to and from bytes.
// It does NOT include the CRC32 checksum - that is handled by the WAL
// implementation to ensure the checksum covers the entire payload.
type Codec[K comparable, V any] interface {
	// Encode serialises an entry to bytes.
	//
	// The output does NOT include length prefix or CRC32 - those are added
	// by the WAL implementation.
	//
	// Takes entry (Entry[K, V]) which is the entry to encode.
	//
	// Returns []byte which is the encoded entry.
	// Returns error when encoding fails.
	Encode(entry Entry[K, V]) ([]byte, error)

	// Decode deserialises bytes to an entry.
	//
	// The input does NOT include length prefix or CRC32 - those are stripped
	// by the WAL implementation before invocation.
	//
	// Takes data ([]byte) which is the encoded entry.
	//
	// Returns Entry[K, V] which is the decoded entry.
	// Returns error when decoding fails.
	Decode(data []byte) (Entry[K, V], error)
}

// KeyCodec handles serialisation of just keys.
// This is used when the key type requires custom encoding.
type KeyCodec[K comparable] interface {
	// EncodeKey serialises a key to bytes.
	EncodeKey(key K) ([]byte, error)

	// DecodeKey deserialises bytes to a key.
	DecodeKey(data []byte) (K, error)
}

// ValueCodec handles serialisation of just values.
// This is used when the value type requires custom encoding.
type ValueCodec[V any] interface {
	// EncodeValue serialises a value to bytes.
	EncodeValue(value V) ([]byte, error)

	// DecodeValue deserialises bytes to a value.
	DecodeValue(data []byte) (V, error)
}

// FastKeyCodec extends KeyCodec with zero-allocation encoding and decoding.
// Implementations can encode directly into a provided buffer and decode
// without copying, eliminating allocations in the hot path.
//
// This is an optional interface. If a KeyCodec also implements FastKeyCodec,
// the WAL codec will use the fast methods for better performance.
//
// Safety: DecodeKeyFrom may return a key that references the input buffer
// (e.g. using unsafe.String for string keys). The caller must ensure the
// buffer outlives the returned key, or copy the key if needed for storage.
type FastKeyCodec[K comparable] interface {
	KeyCodec[K]

	// KeySize returns the encoded size of a key without actually encoding it.
	// Used to pre-calculate buffer sizes.
	KeySize(key K) int

	// EncodeKeyTo encodes a key directly into the provided buffer.
	// The buffer is guaranteed to have at least KeySize(key) bytes available.
	//
	// Returns the number of bytes written.
	// Returns error if encoding fails.
	EncodeKeyTo(key K, buffer []byte) (int, error)

	// DecodeKeyFrom decodes a key without allocating.
	//
	// SAFETY: The returned key may reference the input data buffer.
	// The caller must ensure the buffer outlives the key, or make a copy
	// if the key will be stored beyond the buffer's lifetime.
	DecodeKeyFrom(data []byte) (K, error)
}

// FastValueCodec extends ValueCodec with zero-allocation encoding and decoding.
// Implementations can encode directly into a provided buffer and decode without
// copying, eliminating allocations in the hot path.
//
// This is an optional interface. If a ValueCodec also implements FastValueCodec,
// the WAL codec will use the fast methods for better performance.
//
// SAFETY: DecodeValueFrom may return a value that references the input buffer
// (e.g., using unsafe.String for string fields). The caller must ensure the
// buffer outlives the returned value, or copy if needed for storage.
type FastValueCodec[V any] interface {
	ValueCodec[V]

	// ValueSize returns the encoded size of a value without actually encoding it.
	// Used to pre-calculate buffer sizes.
	ValueSize(value V) int

	// EncodeValueTo encodes a value directly into the provided buffer.
	// The buffer is guaranteed to have at least ValueSize(value) bytes available.
	//
	// Returns the number of bytes written.
	// Returns error if encoding fails.
	EncodeValueTo(value V, buffer []byte) (int, error)

	// DecodeValueFrom decodes a value without allocating.
	//
	// SAFETY: The returned value may reference the input data buffer.
	// The caller must ensure the buffer outlives the value, or make a copy
	// if the value will be stored beyond the buffer's lifetime.
	DecodeValueFrom(data []byte) (V, error)
}
