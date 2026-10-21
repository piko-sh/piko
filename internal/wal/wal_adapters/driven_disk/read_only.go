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

package driven_disk

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"

	"piko.sh/piko/internal/wal/wal_domain"
)

// ReadEntries yields complete log entries without modifying the log.
//
// A missing log is empty. Reading stops at the first torn or corrupt record.
//
// Takes config (wal_domain.Config) which locates the log.
// Takes codec (*BinaryCodec[K, V]) which decodes its entries.
//
// Returns iter.Seq2[wal_domain.Entry[K, V], error] which yields the entries, or an error
// when the log cannot be read or an entry cannot be decoded for another reason.
func ReadEntries[K comparable, V any](
	ctx context.Context,
	config wal_domain.Config,
	codec *BinaryCodec[K, V],
) iter.Seq2[wal_domain.Entry[K, V], error] {
	return func(yield func(wal_domain.Entry[K, V], error) bool) {
		var zero wal_domain.Entry[K, V]
		config = config.WithDefaults()
		file, err := os.Open(filepath.Join(config.Dir, config.WALFileName))
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err != nil {
			yield(zero, fmt.Errorf("opening write-ahead log read-only: %w", err))
			return
		}
		defer func() { _ = file.Close() }()

		yieldLogEntries(ctx, bufio.NewReaderSize(file, readBufferSize), codec, yield)
	}
}

// yieldLogEntries decodes the log's records in order and yields each entry they apply,
// stopping quietly at the end of the log or at the first torn or corrupt record.
//
// Takes reader (*bufio.Reader) which reads the log from its start.
// Takes codec (*BinaryCodec[K, V]) which decodes its entries.
// Takes yield (func(wal_domain.Entry[K, V], error) bool) which receives each entry, or
// the error that ends the read.
func yieldLogEntries[K comparable, V any](
	ctx context.Context,
	reader *bufio.Reader,
	codec *BinaryCodec[K, V],
	yield func(wal_domain.Entry[K, V], error) bool,
) {
	var zero wal_domain.Entry[K, V]
	var lengthBuffer [lengthSize]byte
	for {
		if err := ctx.Err(); err != nil {
			yield(zero, err)
			return
		}
		data, ok := readLogRecord(reader, &lengthBuffer)
		if !ok {
			return
		}
		result, err := codec.DecodeWithCRC(data)
		if errors.Is(err, wal_domain.ErrCorrupted) {
			return
		}
		if err != nil {
			yield(zero, fmt.Errorf("decoding write-ahead log entry: %w", err))
			return
		}
		if !EachApplied(result.Entry, func(applied wal_domain.Entry[K, V]) bool { return yield(applied, nil) }) {
			return
		}
	}
}

// readLogRecord reads the next length-prefixed record from the log.
//
// Takes reader (*bufio.Reader) which is positioned at the record's length prefix.
// Takes lengthBuffer (*[lengthSize]byte) which receives the length prefix.
//
// Returns []byte which is the record's CRC and payload.
// Returns bool which is false at the end of the log or at a torn or implausible record.
func readLogRecord(reader *bufio.Reader, lengthBuffer *[lengthSize]byte) ([]byte, bool) {
	if _, err := io.ReadFull(reader, lengthBuffer[:]); err != nil {
		return nil, false
	}
	length := binary.BigEndian.Uint32(lengthBuffer[:])
	if length == 0 || length > maxEntrySize || length < crcSize+minPayloadSize {
		return nil, false
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, false
	}
	return data, true
}
