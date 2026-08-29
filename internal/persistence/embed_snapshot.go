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

package persistence

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"

	"github.com/klauspost/compress/zstd"
	cache_adapters_otter "piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// snapshotMagic is the magic bytes identifying a snapshot file ("PIKO"). Must match
	// driven_disk/snapshot.go snapshotMagic.
	snapshotMagic uint32 = 0x50494B4F

	// snapshotVersion is the supported snapshot format version. Must match
	// driven_disk/snapshot.go snapshotVersion.
	snapshotVersion uint8 = 1

	// snapshotHeaderSize is the total header size in bytes. Layout: Magic(4) + Version(1) +
	// Flags(1) + Reserved(2) + EntryCount(8) + Timestamp(8) + DataCRC(4) + HeaderCRC(4) =
	// 32.
	snapshotHeaderSize = 32

	// flagCompressed indicates zstd-compressed data. Must match driven_disk/snapshot.go
	// flagCompressed.
	flagCompressed uint8 = 0x01

	// headerOffsetDataCRC is the byte offset for the data CRC in the header.
	headerOffsetDataCRC = 24

	// entryLengthSize is the byte size of the length prefix per entry. Matches
	// driven_disk/codec.go lengthSize.
	entryLengthSize = 4

	// maxEntrySize is the maximum allowed entry size (16 MB). Matches driven_disk/codec.go
	// maxEntrySize.
	maxEntrySize = 16 * 1024 * 1024

	// maxSnapshotDataSize is the maximum raw snapshot data size (1 GB) to prevent unbounded
	// memory allocation from malformed snapshots.
	maxSnapshotDataSize = 1024 * 1024 * 1024

	// maxDecompressedSize is the maximum decompressed data size (2 GB) to prevent
	// decompression bomb attacks.
	maxDecompressedSize = 2 * 1024 * 1024 * 1024

	// maxPreallocEntries is the maximum number of entries to pre-allocate in the slice to
	// prevent OOM from crafted entry counts in the header.
	maxPreallocEntries = 1_000_000

	// uint32Size is the byte size of a uint32.
	uint32Size = 4

	// uint64Size is the byte size of a uint64.
	uint64Size = 8

	// uint16Size is the byte size of a uint16.
	uint16Size = 2

	// registrySnapshotPath is the well-known location of the registry snapshot within the
	// .piko directory.
	registrySnapshotPath = "wal/persistence/registry/snapshot.piko"

	// orchestratorSnapshotPath is the well-known location of the orchestrator snapshot
	// within the .piko directory.
	orchestratorSnapshotPath = "wal/persistence/orchestrator/snapshot.piko"

	// defaultCapacity is the default otter cache capacity when none is specified.
	defaultCapacity int64 = 100_000
)

var (
	// crcTable is the CRC32 table used for snapshot integrity checks, matching the IEEE
	// polynomial used by the WAL implementation.
	crcTable = crc32.MakeTable(crc32.IEEE)
)

// LoadRegistryCacheFromFS reads the registry snapshot from the given fs.FS and returns a
// populated otter cache. The fsys should be rooted at the .piko directory (i.e., after
// fs.Sub(embed, ".piko")).
//
// If the snapshot file does not exist, an empty cache is returned without error.
//
// Takes fsys (fs.FS) which provides access to the embedded .piko directory.
// Takes capacity (int64) which sets the maximum cache size; zero uses default.
//
// Returns cache_domain.ProviderPort[string, *registry_dto.ArtefactMeta] which is the
// populated cache.
// Returns error when the snapshot is corrupt or decoding fails.
func LoadRegistryCacheFromFS(
	ctx context.Context,
	fsys fs.FS,
	capacity int64,
) (cache_domain.ProviderPort[string, *registry_dto.ArtefactMeta], error) {
	codec := driven_disk.NewBinaryCodec[string, *registry_dto.ArtefactMeta](
		StringKeyCodec{},
		ArtefactMetaCodec{},
	)
	return loadAndPopulateCache(ctx, fsys, registrySnapshotPath, codec, capacity, "registry", clock.RealClock())
}

// LoadRegistryArtefactsFromFS reads the registry snapshot from fsys and returns the
// build-time artefact metadata as a slice, for replicating the seed into a SQL backend
// (SQLite/Postgres) at boot. Returns nil (no error) when the snapshot is absent.
//
// Takes fsys (fs.FS) which is rooted at the .piko directory.
//
// Returns []*registry_dto.ArtefactMeta which is the seed artefact set (empty when no
// snapshot exists).
// Returns error when the snapshot is corrupt or decoding fails.
func LoadRegistryArtefactsFromFS(ctx context.Context, fsys fs.FS) ([]*registry_dto.ArtefactMeta, error) {
	codec := driven_disk.NewBinaryCodec[string, *registry_dto.ArtefactMeta](
		StringKeyCodec{},
		ArtefactMetaCodec{},
	)
	entries, err := loadSnapshotEntries(ctx, fsys, registrySnapshotPath, codec)
	if err != nil {
		return nil, err
	}

	artefacts := make([]*registry_dto.ArtefactMeta, 0, len(entries))
	for i := range entries {
		if entries[i].Operation == wal_domain.OpSet && entries[i].Value != nil {
			artefacts = append(artefacts, entries[i].Value)
		}
	}
	return artefacts, nil
}

// LoadOrchestratorCacheFromFS reads the orchestrator snapshot from the given fs.FS and
// returns a populated otter cache. The fsys should be rooted at the .piko directory.
//
// If the snapshot file does not exist, an empty cache is returned without error.
//
// Takes fsys (fs.FS) which provides access to the embedded .piko directory.
// Takes capacity (int64) which sets the maximum cache size; zero uses default.
//
// Returns cache_domain.ProviderPort[string, *orchestrator_domain.Task] which is the
// populated cache.
// Returns error when the snapshot is corrupt or decoding fails.
func LoadOrchestratorCacheFromFS(
	ctx context.Context,
	fsys fs.FS,
	capacity int64,
) (cache_domain.ProviderPort[string, *orchestrator_domain.Task], error) {
	codec := driven_disk.NewBinaryCodec[string, *orchestrator_domain.Task](
		StringKeyCodec{},
		TaskCodec{},
	)
	return loadAndPopulateCache(ctx, fsys, orchestratorSnapshotPath, codec, capacity, "orchestrator", clock.RealClock())
}

// loadAndPopulateCache is the generic implementation shared by LoadRegistryCacheFromFS
// and LoadOrchestratorCacheFromFS.
//
// Takes fsys (fs.FS) which is rooted at the .piko directory.
// Takes snapshotPath (string) which locates the snapshot within fsys.
// Takes codec (*driven_disk.BinaryCodec[K, V]) which decodes entries.
// Takes capacity (int64) which caps the cache; zero uses the default.
// Takes label (string) which names the domain for logging and error wrapping.
//
// Returns cache_domain.ProviderPort[K, V] which is the populated cache.
// Returns error when loading or cache creation fails.
func loadAndPopulateCache[K comparable, V any](
	ctx context.Context,
	fsys fs.FS,
	snapshotPath string,
	codec *driven_disk.BinaryCodec[K, V],
	capacity int64,
	label string,
	clk clock.Clock,
) (cache_domain.ProviderPort[K, V], error) {
	ctx, l := logger_domain.From(ctx, log)

	if capacity <= 0 {
		capacity = defaultCapacity
	}

	entries, err := loadSnapshotEntries(ctx, fsys, snapshotPath, codec)
	if err != nil {
		return nil, fmt.Errorf("loading %s snapshot: %w", label, err)
	}

	cache, cacheErr := cache_adapters_otter.OtterProviderFactory(
		cache_dto.Options[K, V]{
			MaximumEntries: safeconv.Int64ToInt(capacity),
		},
	)
	if cacheErr != nil {
		return nil, fmt.Errorf("creating %s cache: %w", label, cacheErr)
	}

	populated, err := populateCache(ctx, cache, entries, clk)
	if err != nil {
		return nil, fmt.Errorf("populating %s cache: %w", label, err)
	}

	l.Internal("Loaded cache from embedded snapshot",
		logger_domain.String("domain", label),
		logger_domain.Int("entries", populated))

	return cache, nil
}

// loadSnapshotEntries reads and decodes all entries from a snapshot file in the given
// filesystem. Returns nil entries (not an error) when the snapshot file does not exist.
//
// Takes fsys (fs.FS) which is rooted at the .piko directory.
// Takes snapshotPath (string) which locates the snapshot within fsys.
// Takes codec (*driven_disk.BinaryCodec[K, V]) which decodes entries.
//
// Returns []wal_domain.Entry[K, V] which are the decoded entries.
// Returns error when the snapshot is corrupt or decoding fails.
func loadSnapshotEntries[K comparable, V any](
	ctx context.Context,
	fsys fs.FS,
	snapshotPath string,
	codec *driven_disk.BinaryCodec[K, V],
) ([]wal_domain.Entry[K, V], error) {
	header, data, err := readSnapshotFromFS(fsys, snapshotPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	entryCount, flags, err := parseSnapshotHeader(header)
	if err != nil {
		return nil, err
	}

	if err := validateDataCRC(header, data); err != nil {
		return nil, err
	}

	data, err = decompressData(data, flags)
	if err != nil {
		return nil, err
	}

	return decodeEntries(ctx, data, entryCount, codec)
}

// readSnapshotFromFS opens and reads a snapshot file from an fs.FS, returning the header
// and data portions separately. Data is limited to maxSnapshotDataSize to prevent
// unbounded allocation.
//
// Takes fsys (fs.FS) which is rooted at the .piko directory.
// Takes snapshotPath (string) which locates the snapshot within fsys.
//
// Returns header ([]byte) which is the fixed-size snapshot header.
// Returns data ([]byte) which is the remaining snapshot payload.
// Returns err (error) when the file cannot be read or exceeds the size limit.
func readSnapshotFromFS(fsys fs.FS, snapshotPath string) (header []byte, data []byte, err error) {
	file, err := fsys.Open(snapshotPath)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = file.Close() }()

	header = make([]byte, snapshotHeaderSize)
	if _, err := io.ReadFull(file, header); err != nil {
		return nil, nil, fmt.Errorf("reading snapshot header: %w", err)
	}

	limitedReader := io.LimitReader(file, maxSnapshotDataSize+1)
	data, err = io.ReadAll(limitedReader)
	if err != nil {
		return nil, nil, fmt.Errorf("reading snapshot data: %w", err)
	}
	if int64(len(data)) > maxSnapshotDataSize {
		return nil, nil, fmt.Errorf("snapshot data exceeds maximum size of %d bytes", maxSnapshotDataSize)
	}

	return header, data, nil
}

// parseSnapshotHeader validates and parses the 32-byte snapshot header.
//
// Takes header ([]byte) which is the fixed-size snapshot header.
//
// Returns entryCount (uint64) which is the declared entry count.
// Returns flags (uint8) which carry the format flags such as compression.
// Returns err (error) when the header is corrupt or the version is unsupported.
func parseSnapshotHeader(header []byte) (entryCount uint64, flags uint8, err error) {
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

// validateDataCRC checks the data CRC against the value stored in the header.
//
// Takes header ([]byte) which holds the stored data CRC.
// Takes data ([]byte) which is the payload to checksum.
//
// Returns error when the computed CRC does not match the stored one.
func validateDataCRC(header, data []byte) error {
	dataCRC := binary.BigEndian.Uint32(header[headerOffsetDataCRC : headerOffsetDataCRC+uint32Size])
	computedCRC := crc32.Checksum(data, crcTable)
	if dataCRC != computedCRC {
		return fmt.Errorf("%w: data CRC mismatch (stored=%08x, computed=%08x)",
			wal_domain.ErrCorrupted, dataCRC, computedCRC)
	}
	return nil
}

// decompressData decompresses zstd-compressed data when the compressed flag is set.
// Decompressed output is limited to maxDecompressedSize.
//
// Takes data ([]byte) which is the possibly-compressed payload.
// Takes flags (uint8) which indicate whether the payload is compressed.
//
// Returns []byte which is the decompressed payload.
// Returns error when decoder creation or decompression fails.
func decompressData(data []byte, flags uint8) ([]byte, error) {
	if flags&flagCompressed == 0 {
		return data, nil
	}

	decoder, err := zstd.NewReader(nil, zstd.WithDecoderMaxMemory(maxDecompressedSize))
	if err != nil {
		return nil, fmt.Errorf("creating zstd decoder: %w", err)
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, fmt.Errorf("decompressing snapshot data: %w", err)
	}

	return decompressed, nil
}

// decodeEntries reads snapshot entries from the decompressed data buffer, checking ctx
// for cancellation between entries. A short stream (fewer entries than the header
// declared) is treated as corruption rather than silently truncating the load, so a bad
// snapshot fails loudly instead of seeding a partial cache.
//
// Takes data ([]byte) which is the decompressed entry stream.
// Takes entryCount (uint64) which is the declared number of entries.
// Takes codec (*driven_disk.BinaryCodec[K, V]) which decodes each entry.
//
// Returns []wal_domain.Entry[K, V] which are the decoded entries.
// Returns error when an entry is corrupt, decoding fails, or too few entries decode.
func decodeEntries[K comparable, V any](
	ctx context.Context,
	data []byte,
	entryCount uint64,
	codec *driven_disk.BinaryCodec[K, V],
) ([]wal_domain.Entry[K, V], error) {
	reader := bytes.NewReader(data)

	entries := make([]wal_domain.Entry[K, V], 0, min(entryCount, maxPreallocEntries))

	var decoded uint64
	for range entryCount {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("decoding snapshot entries: %w", ctx.Err())
		}

		var lengthBuffer [entryLengthSize]byte
		if _, err := io.ReadFull(reader, lengthBuffer[:]); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("reading entry length: %w", err)
		}

		length := binary.BigEndian.Uint32(lengthBuffer[:])
		if length > maxEntrySize {
			return nil, fmt.Errorf("%w: entry length %d exceeds maximum %d",
				wal_domain.ErrCorrupted, length, maxEntrySize)
		}

		entryData := make([]byte, length)
		if _, err := io.ReadFull(reader, entryData); err != nil {
			return nil, fmt.Errorf("reading entry data: %w", err)
		}

		result, err := codec.DecodeWithCRC(entryData)
		if err != nil {
			return nil, fmt.Errorf("decoding entry: %w", err)
		}

		entries = append(entries, result.Entry)
		decoded++
	}

	if decoded != entryCount {
		return nil, fmt.Errorf("%w: snapshot declared %d entries but only %d were decoded before end of data",
			wal_domain.ErrCorrupted, entryCount, decoded)
	}

	return entries, nil
}

// populateCache sets all OpSet entries into the cache, skipping expired entries. A
// context cancellation mid-population is returned as an error so the caller cannot report
// success over a partially populated cache.
//
// Takes cache (cache_domain.ProviderPort[K, V]) which receives the entries.
// Takes entries ([]wal_domain.Entry[K, V]) which are the decoded entries to apply.
//
// Returns int which is the number of entries populated.
// Returns error when ctx is cancelled before every entry is applied.
func populateCache[K comparable, V any](
	ctx context.Context,
	cache cache_domain.ProviderPort[K, V],
	entries []wal_domain.Entry[K, V],
	clk clock.Clock,
) (int, error) {
	ctx, l := logger_domain.From(ctx, log)
	nowNano := clk.Now().UnixNano()
	populated := 0

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return populated, fmt.Errorf("populating cache from snapshot: %w", err)
		}

		if entry.Operation != wal_domain.OpSet {
			continue
		}

		if entry.IsExpired(nowNano) {
			continue
		}

		if err := cache.Set(ctx, entry.Key, entry.Value, entry.Tags...); err != nil {
			l.Warn("Failed to populate cache entry",
				logger_domain.Error(err))
			continue
		}

		populated++
	}

	return populated, nil
}
