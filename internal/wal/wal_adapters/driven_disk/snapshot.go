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
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"iter"
	"os"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// snapshotMagic is the magic bytes identifying a snapshot file.
	// "PIKO" in ASCII.
	snapshotMagic uint32 = 0x50494B4F

	// snapshotVersion is the current version of the snapshot file format.
	snapshotVersion uint8 = 1

	// snapshotHeaderSize is the size of the snapshot header in bytes.
	// Layout: Magic(4) + Version(1) + Flags(1) + Reserved(2) + EntryCount(8) +
	// Timestamp(8) + DataCRC(4) + HeaderCRC(4) = 32.
	snapshotHeaderSize = 32

	// flagCompressed indicates that the data has been compressed using zstd.
	flagCompressed uint8 = 0x01

	// filePermissions is the Unix permission mode for snapshot and WAL files.
	filePermissions = 0o600

	// headerOffsetDataCRC is the byte offset for the data CRC field in the header.
	headerOffsetDataCRC = 24
)

// DiskSnapshot implements the SnapshotStore interface using disk-based storage.
// It also implements io.Closer.
//
// Snapshot file format:
// [Header: 32 bytes]
// [Entries: variable, optionally compressed]
// Header format:
// [Magic: 4 bytes "PIKO"]
// [Version: 1 byte]
// [Flags: 1 byte]
// [Reserved: 2 bytes]
// [EntryCount: 8 bytes]
// [Timestamp: 8 bytes]
// [DataCRC: 4 bytes]
// [HeaderCRC: 4 bytes]
type DiskSnapshot[K comparable, V any] struct {
	// clock provides time functions; tests may replace this.
	clock clock.Clock

	// sandbox provides safe file operations within the snapshot folder.
	sandbox safedisk.Sandbox

	// codec encodes and decodes key-value pairs for storage.
	codec *BinaryCodec[K, V]

	// encoder is the reusable zstd encoder for compressing data.
	encoder *zstd.Encoder

	// decoder is the reusable zstd decoder for reading compressed data.
	decoder *zstd.Decoder

	// config holds the WAL domain configuration settings.
	config wal_domain.Config

	// mu guards concurrent access to the snapshot fields.
	mu sync.Mutex
}

var _ wal_domain.SnapshotStore[string, any] = (*DiskSnapshot[string, any])(nil)

// SnapshotOption configures a DiskSnapshot instance.
type SnapshotOption[K comparable, V any] func(*DiskSnapshot[K, V])

// initialiseCompression creates the zstd encoder and decoder,
// cleaning up on partial failure.
//
// Takes sandbox (safedisk.Sandbox) which is closed on failure to avoid resource
// leaks.
//
// Returns error when the encoder or decoder cannot be created.
func (s *DiskSnapshot[K, V]) initialiseCompression(ctx context.Context, sandbox safedisk.Sandbox) error {
	_, l := logger_domain.From(context.WithoutCancel(ctx), log)
	level := zstd.EncoderLevelFromZstd(s.config.CompressionLevel)

	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(level))
	if err != nil {
		if closeErr := sandbox.Close(); closeErr != nil {
			l.Warn("closing sandbox after zstd encoder creation failure", logger_domain.Error(closeErr))
		}
		return fmt.Errorf("creating zstd encoder: %w", err)
	}
	s.encoder = encoder

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		if closeErr := encoder.Close(); closeErr != nil {
			l.Warn("closing zstd encoder after decoder creation failure", logger_domain.Error(closeErr))
		}
		if closeErr := sandbox.Close(); closeErr != nil {
			l.Warn("closing sandbox after zstd decoder creation failure", logger_domain.Error(closeErr))
		}
		return fmt.Errorf("creating zstd decoder: %w", err)
	}
	s.decoder = decoder

	return nil
}

// Save persists a complete snapshot of all entries.
//
// Takes entries ([]wal_domain.Entry[K, V]) which contains all entries to
// include in the snapshot.
//
// Returns error when encoding, compression, or writing to disk fails.
//
// Safe for concurrent use. Holds a mutex lock for the entire operation.
func (s *DiskSnapshot[K, V]) Save(ctx context.Context, entries []wal_domain.Entry[K, V]) error {
	startTime := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	entriesData, err := s.encodeEntries(entries)
	if err != nil {
		return fmt.Errorf("encoding snapshot entries: %w", err)
	}

	finalData, flags := s.compressData(entriesData)

	header := s.buildSnapshotHeader(len(entries), finalData, flags)

	if err := s.writeSnapshotFile(ctx, header, finalData); err != nil {
		return fmt.Errorf("writing snapshot file: %w", err)
	}

	s.recordSaveMetrics(ctx, startTime, len(entries), len(finalData), flags)

	return nil
}

// encodeEntries encodes all entries to a byte buffer.
//
// Takes entries ([]Entry) which contains the entries to encode.
//
// Returns []byte which contains the encoded entries with length prefixes.
// Returns error when encoding any entry fails.
func (s *DiskSnapshot[K, V]) encodeEntries(entries []wal_domain.Entry[K, V]) ([]byte, error) {
	var buffer bytes.Buffer
	var lenBuf [lengthSize]byte

	for i, entry := range entries {
		result, err := s.codec.EncodeWithCRC(entry)
		if err != nil {
			return nil, fmt.Errorf("encoding entry %d: %w", i, err)
		}

		binary.BigEndian.PutUint32(lenBuf[:], safeconv.IntToUint32(len(result.Data)))
		_, _ = buffer.Write(lenBuf[:])
		_, _ = buffer.Write(result.Data)
		result.Release()
	}

	return buffer.Bytes(), nil
}

// compressData optionally compresses the data and returns the flags.
//
// Takes data ([]byte) which is the raw data to compress.
//
// Returns []byte which is the compressed data, or the original if compression
// is disabled.
// Returns uint8 which is the compression flag indicating the data state.
func (s *DiskSnapshot[K, V]) compressData(data []byte) ([]byte, uint8) {
	if s.config.EnableCompression && s.encoder != nil {
		return s.encoder.EncodeAll(data, nil), flagCompressed
	}
	return data, 0
}

// buildSnapshotHeader creates the snapshot header with CRCs.
//
// Takes entryCount (int) which specifies the number of entries in the snapshot.
// Takes data ([]byte) which contains the snapshot data to compute CRC for.
// Takes flags (uint8) which provides header flags for the snapshot.
//
// Returns []byte which contains the complete header with magic, version, flags,
// entry count, timestamp, and CRC checksums.
func (s *DiskSnapshot[K, V]) buildSnapshotHeader(entryCount int, data []byte, flags uint8) []byte {
	header := make([]byte, snapshotHeaderSize)
	offset := 0

	binary.BigEndian.PutUint32(header[offset:], snapshotMagic)
	offset += uint32Size

	header[offset] = snapshotVersion
	offset++

	header[offset] = flags
	offset++

	offset += uint16Size

	binary.BigEndian.PutUint64(header[offset:], safeconv.IntToUint64(entryCount))
	offset += uint64Size

	binary.BigEndian.PutUint64(header[offset:], safeconv.Int64ToUint64(s.clock.Now().UnixNano()))
	offset += uint64Size

	dataCRC := crc32.Checksum(data, crcTable)
	binary.BigEndian.PutUint32(header[offset:], dataCRC)
	offset += uint32Size

	headerCRC := crc32.Checksum(header[:offset], crcTable)
	binary.BigEndian.PutUint32(header[offset:], headerCRC)

	return header
}

// writeSnapshotFile writes header and data to a temp file, then atomically
// renames it to the final snapshot file.
//
// Takes header ([]byte) which contains the snapshot metadata.
// Takes data ([]byte) which contains the serialised snapshot content.
//
// Returns error when the temp file cannot be created, written, or renamed.
func (s *DiskSnapshot[K, V]) writeSnapshotFile(ctx context.Context, header, data []byte) error {
	tempName := s.config.SnapshotFileName + ".tmp"

	file, err := s.sandbox.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions)
	if err != nil {
		return fmt.Errorf("creating temp snapshot file: %w", err)
	}

	if err := s.writeAndSync(ctx, file, tempName, header, data); err != nil {
		return fmt.Errorf("writing and syncing snapshot: %w", err)
	}

	if err := s.sandbox.Rename(tempName, s.config.SnapshotFileName); err != nil {
		if rmErr := s.sandbox.Remove(tempName); rmErr != nil {
			_, l := logger_domain.From(ctx, log)
			l.Warn("removing temp snapshot after rename failure", logger_domain.Error(rmErr))
		}
		return fmt.Errorf("renaming snapshot file: %w", err)
	}

	s.syncSnapshotDirectory(ctx)

	return nil
}

// syncSnapshotDirectory fsyncs the sandbox root so the rename of the snapshot
// file is durable on filesystems with journaled metadata. The snapshot file
// itself was fsynced before the rename; this final step flushes the directory
// entry for the rename so a crash cannot leave the file invisible after
// recovery.
//
// Errors are best-effort and silently dropped because the data on disk is
// already durable; the worst case is a re-replay from the WAL on next start.
//
// Takes ctx (context.Context) which carries the logger for warning trace.
func (s *DiskSnapshot[K, V]) syncSnapshotDirectory(ctx context.Context) {
	dirHandle, err := s.sandbox.OpenFile(".", os.O_RDONLY, 0)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Trace("opening snapshot directory for fsync skipped", logger_domain.Error(err))
		return
	}
	if syncErr := dirHandle.Sync(); syncErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Trace("fsync of snapshot directory failed", logger_domain.Error(syncErr))
	}
	if closeErr := dirHandle.Close(); closeErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Trace("closing snapshot directory handle failed", logger_domain.Error(closeErr))
	}
}

// writeAndSync writes header and data to the file, syncs, and closes it.
//
// Takes file (safedisk.FileHandle) which is the open file to write to.
// Takes tempName (string) which is the temporary file path for cleanup on error.
// Takes header ([]byte) which contains the snapshot header bytes.
// Takes data ([]byte) which contains the snapshot data bytes.
//
// Returns error when writing, syncing, or closing the file fails.
func (s *DiskSnapshot[K, V]) writeAndSync(ctx context.Context, file safedisk.FileHandle, tempName string, header, data []byte) error {
	_, l := logger_domain.From(ctx, log)
	cleanup := func() {
		if closeErr := file.Close(); closeErr != nil {
			l.Warn("closing snapshot file during cleanup", logger_domain.Error(closeErr))
		}
		if rmErr := s.sandbox.Remove(tempName); rmErr != nil {
			l.Warn("removing temp snapshot during cleanup", logger_domain.Error(rmErr))
		}
	}

	if _, err := file.Write(header); err != nil {
		cleanup()
		return fmt.Errorf("writing snapshot header: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("writing snapshot data: %w", err)
	}

	if err := file.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("syncing snapshot file: %w", err)
	}

	if err := file.Close(); err != nil {
		if rmErr := s.sandbox.Remove(tempName); rmErr != nil {
			l.Warn("removing temp snapshot after close failure", logger_domain.Error(rmErr))
		}
		return fmt.Errorf("closing snapshot file: %w", err)
	}

	return nil
}

// recordSaveMetrics records metrics and logs after a successful save.
//
// Takes startTime (time.Time) which is when the save operation began.
// Takes entryCount (int) which is the number of entries saved.
// Takes dataLen (int) which is the size of the data in bytes.
// Takes flags (uint8) which contains flags indicating save options.
func (s *DiskSnapshot[K, V]) recordSaveMetrics(ctx context.Context, startTime time.Time, entryCount, dataLen int, flags uint8) {
	ctx, l := logger_domain.From(ctx, log)
	totalSize := int64(snapshotHeaderSize + dataLen)
	wal_domain.SnapshotSaveTotal.Add(ctx, 1)
	wal_domain.SnapshotSaveDuration.Record(ctx, float64(s.clock.Now().Sub(startTime).Milliseconds()))
	wal_domain.SnapshotSizeBytes.Record(ctx, totalSize)

	l.Internal("Snapshot saved",
		logger_domain.Int("entry_count", entryCount),
		logger_domain.Int64("size_bytes", totalSize),
		logger_domain.Bool("compressed", flags&flagCompressed != 0))
}

// Load returns an iterator over snapshot entries without loading
// all into memory.
//
// Returns iter.Seq2[wal_domain.Entry[K, V], error] which yields
// each entry and any error encountered during iteration.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex held for the duration of iteration.
//
// NOTE: Compressed snapshots must still decompress fully before
// streaming entries, due to zstd's block-based decompression.
// Uncompressed snapshots stream directly from the in-memory data
// after CRC validation.
//
// IMPORTANT: The iterator holds the snapshot lock during iteration.
// Callers should consume all entries promptly or break out of the
// loop to release.
func (s *DiskSnapshot[K, V]) Load(ctx context.Context) iter.Seq2[wal_domain.Entry[K, V], error] {
	return func(yield func(wal_domain.Entry[K, V], error) bool) {
		startTime := s.clock.Now()
		s.mu.Lock()
		defer s.mu.Unlock()

		header, data, err := s.readSnapshotFile()
		if err != nil {
			yield(wal_domain.Entry[K, V]{}, err)
			return
		}

		entryCount, flags, err := s.parseHeader(header)
		if err != nil {
			yield(wal_domain.Entry[K, V]{}, err)
			return
		}

		if err := s.validateDataCRC(header, data); err != nil {
			yield(wal_domain.Entry[K, V]{}, err)
			return
		}

		data, err = s.decompressData(data, flags)
		if err != nil {
			yield(wal_domain.Entry[K, V]{}, err)
			return
		}

		actualCount := s.yieldEntries(ctx, data, entryCount, yield)

		s.recordLoadMetrics(ctx, startTime, actualCount, flags)
	}
}

// yieldEntries streams entries from the data buffer to the yield function.
//
// Takes data ([]byte) which contains the serialised snapshot entries.
// Takes entryCount (uint64) which specifies the maximum entries to read.
// Takes yield (func(...)) which receives each entry or error and returns
// false to stop iteration.
//
// Returns int which is the number of entries successfully yielded.
func (s *DiskSnapshot[K, V]) yieldEntries(
	ctx context.Context,
	data []byte,
	entryCount uint64,
	yield func(wal_domain.Entry[K, V], error) bool,
) int {
	reader := bytes.NewReader(data)
	var count int

	for range entryCount {
		if ctx.Err() != nil {
			yield(wal_domain.Entry[K, V]{}, ctx.Err())
			return count
		}

		entry, done, err := s.readSnapshotEntry(reader)
		if err != nil {
			yield(wal_domain.Entry[K, V]{}, err)
			return count
		}
		if done {
			return count
		}

		count++
		if !yield(entry, nil) {
			return count
		}
	}

	return count
}

// readSnapshotEntry reads a single entry from the reader.
//
// Takes reader (*bytes.Reader) which provides the input data stream.
//
// Returns wal_domain.Entry[K, V] which is the decoded entry.
// Returns bool which is true when EOF has been reached.
// Returns error when reading or decoding fails.
func (s *DiskSnapshot[K, V]) readSnapshotEntry(reader *bytes.Reader) (wal_domain.Entry[K, V], bool, error) {
	var lenBuf [lengthSize]byte

	if _, err := io.ReadFull(reader, lenBuf[:]); err != nil {
		if errors.Is(err, io.EOF) {
			return wal_domain.Entry[K, V]{}, true, nil
		}
		return wal_domain.Entry[K, V]{}, false, fmt.Errorf("reading entry length: %w", err)
	}

	length := binary.BigEndian.Uint32(lenBuf[:])
	if length > maxEntrySize {
		return wal_domain.Entry[K, V]{}, false, fmt.Errorf("%w: invalid entry length %d", wal_domain.ErrCorrupted, length)
	}

	entryData := make([]byte, length)
	if _, err := io.ReadFull(reader, entryData); err != nil {
		return wal_domain.Entry[K, V]{}, false, fmt.Errorf("reading entry data: %w", err)
	}

	result, err := s.codec.DecodeWithCRC(entryData)
	if err != nil {
		return wal_domain.Entry[K, V]{}, false, fmt.Errorf("decoding entry: %w", err)
	}

	return result.Entry, false, nil
}

// readSnapshotFile opens and reads the snapshot file header and data.
//
// Returns header ([]byte) which contains the snapshot header bytes.
// Returns data ([]byte) which contains the snapshot data bytes.
// Returns err (error) when the file cannot be opened or read.
func (s *DiskSnapshot[K, V]) readSnapshotFile() (header, data []byte, err error) {
	file, err := s.sandbox.Open(s.config.SnapshotFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, wal_domain.ErrSnapshotNotFound
		}
		return nil, nil, fmt.Errorf("opening snapshot file: %w", err)
	}
	defer func() { _ = file.Close() }()

	header = make([]byte, snapshotHeaderSize)
	if _, err := io.ReadFull(file, header); err != nil {
		return nil, nil, fmt.Errorf("reading snapshot header: %w", err)
	}

	data, err = io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("reading snapshot data: %w", err)
	}

	return header, data, nil
}

// validateDataCRC checks the data CRC against the stored value in the header.
//
// Takes header ([]byte) which contains the stored CRC value.
// Takes data ([]byte) which is the data to compute the CRC for.
//
// Returns error when the computed CRC does not match the stored CRC.
func (*DiskSnapshot[K, V]) validateDataCRC(header, data []byte) error {
	dataCRC := binary.BigEndian.Uint32(header[headerOffsetDataCRC : headerOffsetDataCRC+uint32Size])
	computedCRC := crc32.Checksum(data, crcTable)
	if dataCRC != computedCRC {
		return fmt.Errorf("%w: data CRC mismatch (stored=%08x, computed=%08x)",
			wal_domain.ErrCorrupted, dataCRC, computedCRC)
	}
	return nil
}

// decompressData decompresses the data if the compressed flag is set.
//
// Takes data ([]byte) which is the raw data to decompress.
// Takes flags (uint8) which contains compression status bits.
//
// Returns []byte which is the decompressed data, or the original data if not
// compressed.
// Returns error when compression is disabled but data is compressed, or when
// decompression fails.
func (s *DiskSnapshot[K, V]) decompressData(data []byte, flags uint8) ([]byte, error) {
	if flags&flagCompressed == 0 {
		return data, nil
	}

	if s.decoder == nil {
		return nil, errors.New("snapshot is compressed but compression is disabled")
	}

	decompressed, err := s.decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, fmt.Errorf("decompressing snapshot data: %w", err)
	}

	return decompressed, nil
}

// recordLoadMetrics records metrics and logs after a successful load.
//
// Takes startTime (time.Time) which marks when the load operation began.
// Takes entryCount (int) which is the number of entries that were loaded.
// Takes flags (uint8) which contains bitflags indicating load options.
func (s *DiskSnapshot[K, V]) recordLoadMetrics(ctx context.Context, startTime time.Time, entryCount int, flags uint8) {
	ctx, l := logger_domain.From(ctx, log)
	wal_domain.SnapshotLoadDuration.Record(ctx, float64(s.clock.Now().Sub(startTime).Milliseconds()))

	l.Internal("Snapshot loaded",
		logger_domain.Int("entry_count", entryCount),
		logger_domain.Bool("compressed", flags&flagCompressed != 0))
}

// parseHeader validates and parses the snapshot header.
//
// Takes header ([]byte) which contains the raw header bytes to parse.
//
// Returns entryCount (uint64) which is the number of entries in the
// snapshot.
// Returns flags (uint8) which contains the header flags.
// Returns err (error) when the header is corrupt or has an unsupported
// version.
func (*DiskSnapshot[K, V]) parseHeader(header []byte) (entryCount uint64, flags uint8, err error) {
	offset := 0

	magic := binary.BigEndian.Uint32(header[offset:])
	offset += uint32Size
	if magic != snapshotMagic {
		return 0, 0, fmt.Errorf("%w: invalid magic bytes %08x", wal_domain.ErrCorrupted, magic)
	}

	version := header[offset]
	offset++
	if version != snapshotVersion {
		return 0, 0, fmt.Errorf("%w: unsupported version %d", wal_domain.ErrInvalidVersion, version)
	}

	flags = header[offset]
	offset++

	offset += uint16Size

	entryCount = binary.BigEndian.Uint64(header[offset:])
	offset += uint64Size

	offset += uint64Size

	offset += uint32Size

	storedHeaderCRC := binary.BigEndian.Uint32(header[offset:])
	computedHeaderCRC := crc32.Checksum(header[:offset], crcTable)
	if storedHeaderCRC != computedHeaderCRC {
		return 0, 0, fmt.Errorf("%w: header CRC mismatch", wal_domain.ErrCorrupted)
	}

	return entryCount, flags, nil
}

// Delete removes the snapshot file.
//
// Returns error when the file cannot be removed.
//
// Safe for concurrent use.
func (s *DiskSnapshot[K, V]) Delete(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.sandbox.Remove(s.config.SnapshotFileName); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("removing snapshot file: %w", err)
	}

	l.Internal("Snapshot deleted")
	return nil
}

// Close releases resources held by the snapshot store.
//
// Returns error when the encoder or sandbox fails to close.
//
// Safe for concurrent use.
func (s *DiskSnapshot[K, V]) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error

	if s.encoder != nil {
		if err := s.encoder.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing zstd encoder: %w", err))
		}
		s.encoder = nil
	}

	if s.decoder != nil {
		s.decoder.Close()
		s.decoder = nil
	}

	if s.sandbox != nil {
		if err := s.sandbox.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing snapshot sandbox: %w", err))
		}
		s.sandbox = nil
	}

	return errors.Join(errs...)
}

// Exists reports whether a snapshot file exists.
//
// Returns bool which is true if the file exists, false otherwise.
func (s *DiskSnapshot[K, V]) Exists() bool {
	_, err := s.sandbox.Stat(s.config.SnapshotFileName)
	return err == nil
}

// Path returns the path to the snapshot file.
//
// Returns string which is the full path to the snapshot file.
func (s *DiskSnapshot[K, V]) Path() string {
	return s.sandbox.Root() + "/" + s.config.SnapshotFileName
}

// WithSnapshotClock sets the clock for the snapshot store (for testing).
//
// Takes clk (clock.Clock) which provides the time source.
//
// Returns SnapshotOption[K, V] which configures the snapshot store.
func WithSnapshotClock[K comparable, V any](clk clock.Clock) SnapshotOption[K, V] {
	return func(s *DiskSnapshot[K, V]) {
		s.clock = clk
	}
}

// NewDiskSnapshot creates a new disk-based snapshot store.
//
// Takes ctx (context.Context) which is stored for logging in
// initialisation and cleanup paths.
// Takes config (wal_domain.Config) which specifies the snapshot directory
// and file settings.
// Takes codec (*BinaryCodec[K, V]) which encodes and decodes entries.
// Takes opts (...SnapshotOption[K, V]) which provide optional configuration.
//
// Returns *DiskSnapshot[K, V] which is the initialised snapshot store.
// Returns error when the codec is nil, configuration is invalid, or
// the sandbox cannot be created.
func NewDiskSnapshot[K comparable, V any](
	ctx context.Context,
	config wal_domain.Config,
	codec *BinaryCodec[K, V],
	opts ...SnapshotOption[K, V],
) (*DiskSnapshot[K, V], error) {
	if codec == nil {
		return nil, wal_domain.ErrCodecRequired
	}

	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validating snapshot config: %w", err)
	}

	sandbox, err := createSnapshotSandbox(config.Dir)
	if err != nil {
		return nil, fmt.Errorf("creating snapshot sandbox: %w", err)
	}

	snapshot := &DiskSnapshot[K, V]{
		codec:   codec,
		config:  config,
		sandbox: sandbox,

		clock: clock.RealClock(),
	}

	for _, opt := range opts {
		opt(snapshot)
	}

	if config.EnableCompression {
		if err := snapshot.initialiseCompression(ctx, sandbox); err != nil {
			return nil, err
		}
	}

	return snapshot, nil
}

// createSnapshotSandbox creates a sandboxed filesystem for the snapshot
// directory. The directory is created automatically by safedisk for
// ModeReadWrite.
//
// Takes directory (string) which specifies the path to the snapshot directory.
//
// Returns safedisk.Sandbox which provides restricted filesystem access.
// Returns error when the sandbox factory or sandbox cannot be created.
func createSnapshotSandbox(directory string) (safedisk.Sandbox, error) {
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		Enabled:      true,
		AllowedPaths: []string{directory},
	})
	if err != nil {
		return nil, fmt.Errorf("creating sandbox factory: %w", err)
	}

	sandbox, err := factory.Create("snapshot", directory, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("creating sandbox: %w", err)
	}

	return sandbox, nil
}
