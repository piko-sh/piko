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
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// formatVersion is the binary format version for WAL entries. Increment when
	// making breaking changes.
	formatVersion uint8 = 1

	// lengthSize is the byte size of the length prefix in encoded entries.
	lengthSize = 4

	// crcSize is the size of a CRC32 checksum in bytes.
	crcSize = 4

	// uint32Size is the byte size of a uint32 value.
	uint32Size = 4

	// uint64Size is the number of bytes in a uint64.
	uint64Size = 8

	// uint16Size is the byte size of a uint16 value.
	uint16Size = 2

	// fixedPayloadSize is the size in bytes of fixed fields in the payload:
	// version (1) + operation (1) + timestamp (8) + expiresAt (8) = 18 bytes.
	fixedPayloadSize = 1 + 1 + uint64Size + uint64Size

	// minPayloadSize is the minimum valid payload size in bytes, calculated as
	// fixedPayloadSize + keyLen (4) + valueLen (4) + tagCount (2) = 28 bytes.
	minPayloadSize = fixedPayloadSize + uint32Size + uint32Size + uint16Size

	// maxEntrySize is the maximum allowed entry size (16MB).
	// This prevents memory exhaustion from corrupted length fields.
	maxEntrySize = 16 * 1024 * 1024

	// maxTagCount is the most tags that an entry may have.
	maxTagCount = 1000

	// maxTagSize is the largest allowed size for a single tag in bytes.
	maxTagSize = 1024
)

// crcTable is the precomputed CRC32 table using IEEE polynomial.
var crcTable = crc32.MakeTable(crc32.IEEE)

// BinaryCodec implements the Codec interface using a binary format.
//
// Wire format (all fields are big-endian):
//
//	[Version:1][Operation:1][Timestamp:8][ExpiresAt:8]
//	[KeyLen:4][Key:var][ValueLen:4][Value:var]
//	[TagCount:2][Tags:var]
//
// Tags format:
// For each tag: [TagLen:2][TagData:var]
// The caller (WAL) wraps this with:
// [TotalLen:4][CRC32:4][Payload]
type BinaryCodec[K comparable, V any] struct {
	// keyCodec encodes and decodes keys for binary storage.
	keyCodec wal_domain.KeyCodec[K]

	// valueCodec encodes and decodes values for storage.
	valueCodec wal_domain.ValueCodec[V]

	// fastKeyCodec is a cached fast codec reference; nil if not supported.
	fastKeyCodec wal_domain.FastKeyCodec[K]

	// fastValueCodec encodes and decodes values for fast WAL operations.
	fastValueCodec wal_domain.FastValueCodec[V]
}

// Encode serialises an entry to bytes.
//
// The output does NOT include length prefix or CRC32 - those are added
// by the WAL implementation.
//
// Takes entry (wal_domain.Entry[K, V]) which is the entry to serialise.
//
// Returns []byte which contains the serialised entry data.
// Returns error when encoding the key or value fails.
func (c *BinaryCodec[K, V]) Encode(entry wal_domain.Entry[K, V]) ([]byte, error) {
	keyBytes, err := c.keyCodec.EncodeKey(entry.Key)
	if err != nil {
		return nil, fmt.Errorf("encoding key: %w", err)
	}

	var valueBytes []byte
	if entry.Operation == wal_domain.OpSet {
		valueBytes, err = c.valueCodec.EncodeValue(entry.Value)
		if err != nil {
			return nil, fmt.Errorf("encoding value: %w", err)
		}
	}

	tagsSize := c.calculateTagsSize(entry.Tags)
	totalSize := fixedPayloadSize + uint32Size + len(keyBytes) + uint32Size + len(valueBytes) + uint16Size + tagsSize
	buffer := make([]byte, totalSize)

	offset := encodeFixedFields(buffer, entry.Operation, entry.Timestamp, entry.ExpiresAt)
	offset = encodeBytes(buffer, offset, keyBytes)
	offset = encodeBytes(buffer, offset, valueBytes)
	encodeTags(buffer, offset, entry.Tags)

	return buffer, nil
}

// EncodePooled serialises an entry using a pooled buffer for zero-allocation
// encoding. This method uses FastKeyCodec and FastValueCodec if available,
// otherwise falls back to the standard Encode method.
//
// The returned EncodeResult MUST have Release() called when the caller is done
// with the data to return the buffer to the pool.
//
// Takes entry (wal_domain.Entry[K, V]) which is the entry to serialise.
//
// Returns EncodeResult which contains the encoded data and pool reference.
// Returns error when encoding the key or value fails.
func (c *BinaryCodec[K, V]) EncodePooled(entry wal_domain.Entry[K, V]) (EncodeResult, error) {
	if c.fastKeyCodec == nil || c.fastValueCodec == nil {
		data, err := c.Encode(entry)
		if err != nil {
			return EncodeResult{}, fmt.Errorf("encoding entry: %w", err)
		}
		return EncodeResult{Data: data, pool: nil}, nil
	}

	keySize := c.fastKeyCodec.KeySize(entry.Key)
	valueSize := c.getValueSize(entry)
	tagsSize := c.calculateTagsSize(entry.Tags)
	totalSize := fixedPayloadSize + uint32Size + keySize + uint32Size + valueSize + uint16Size + tagsSize

	pool, buffer := GetByteBuffer(totalSize)
	buffer = buffer[:totalSize]

	offset := encodeFixedFields(buffer, entry.Operation, entry.Timestamp, entry.ExpiresAt)

	var err error
	offset, err = c.encodeKeyFast(buffer, offset, entry.Key, keySize)
	if err != nil {
		PutByteBuffer(pool, buffer)
		return EncodeResult{}, fmt.Errorf("encoding key (pooled): %w", err)
	}

	offset, err = c.encodeValueFast(buffer, offset, entry, valueSize)
	if err != nil {
		PutByteBuffer(pool, buffer)
		return EncodeResult{}, fmt.Errorf("encoding value (pooled): %w", err)
	}

	encodeTags(buffer, offset, entry.Tags)
	return EncodeResult{Data: buffer, pool: pool}, nil
}

// getValueSize returns the encoded size of the value using fast codec.
//
// Takes entry (wal_domain.Entry[K, V]) which is the WAL entry to measure.
//
// Returns int which is the encoded size for set operations, or zero otherwise.
func (c *BinaryCodec[K, V]) getValueSize(entry wal_domain.Entry[K, V]) int {
	if entry.Operation == wal_domain.OpSet {
		return c.fastValueCodec.ValueSize(entry.Value)
	}
	return 0
}

// encodeKeyFast encodes a key using the fast codec.
//
// Takes buffer ([]byte) which is the destination buffer for the encoded data.
// Takes offset (int) which is the starting position in the buffer.
// Takes key (K) which is the key to encode.
// Takes keySize (int) which is the expected size of the encoded key.
//
// Returns int which is the new offset after writing the key.
// Returns error when the key encoding fails.
func (c *BinaryCodec[K, V]) encodeKeyFast(buffer []byte, offset int, key K, keySize int) (int, error) {
	binary.BigEndian.PutUint32(buffer[offset:], safeconv.IntToUint32(keySize))
	offset += uint32Size
	n, err := c.fastKeyCodec.EncodeKeyTo(key, buffer[offset:offset+keySize])
	if err != nil {
		return 0, fmt.Errorf("encoding key: %w", err)
	}
	return offset + n, nil
}

// encodeValueFast encodes a value using the fast codec.
//
// Takes buffer ([]byte) which is the buffer to write the encoded value into.
// Takes offset (int) which is the position in the buffer to start writing.
// Takes entry (wal_domain.Entry) which contains the value to encode.
// Takes valueSize (int) which specifies the size of the value in bytes.
//
// Returns int which is the new offset after encoding.
// Returns error when the value encoding fails.
func (c *BinaryCodec[K, V]) encodeValueFast(buffer []byte, offset int, entry wal_domain.Entry[K, V], valueSize int) (int, error) {
	binary.BigEndian.PutUint32(buffer[offset:], safeconv.IntToUint32(valueSize))
	offset += uint32Size
	if entry.Operation == wal_domain.OpSet {
		n, err := c.fastValueCodec.EncodeValueTo(entry.Value, buffer[offset:offset+valueSize])
		if err != nil {
			return 0, fmt.Errorf("encoding value: %w", err)
		}
		return offset + n, nil
	}
	return offset, nil
}

// decodeSlow deserialises bytes to an entry using allocating codecs.
//
// Takes data ([]byte) which contains the raw encoded entry payload.
//
// Returns wal_domain.Entry[K, V] which is the decoded entry.
// Returns error when the payload is too short, the version is invalid,
// or decoding fails.
func (c *BinaryCodec[K, V]) decodeSlow(data []byte) (wal_domain.Entry[K, V], error) {
	var entry wal_domain.Entry[K, V]

	if len(data) < minPayloadSize {
		return entry, fmt.Errorf("%w: payload too short (%d bytes)", wal_domain.ErrInvalidEntry, len(data))
	}

	offset := 0

	version := data[offset]
	offset++
	if version != formatVersion {
		return entry, fmt.Errorf("%w: got version %d, expected %d", wal_domain.ErrInvalidVersion, version, formatVersion)
	}

	entry.Operation = wal_domain.Operation(data[offset])
	offset++
	if !entry.Operation.IsValid() {
		return entry, fmt.Errorf("%w: invalid operation %d", wal_domain.ErrInvalidEntry, entry.Operation)
	}

	entry.Timestamp = safeconv.Uint64ToInt64(binary.BigEndian.Uint64(data[offset:]))
	offset += uint64Size

	entry.ExpiresAt = safeconv.Uint64ToInt64(binary.BigEndian.Uint64(data[offset:]))
	offset += uint64Size

	key, keyBytesRead, err := c.decodeKey(data, offset)
	if err != nil {
		return entry, fmt.Errorf("decoding entry key: %w", err)
	}
	entry.Key = key
	offset += keyBytesRead

	value, valueBytesRead, err := c.decodeValue(data, offset)
	if err != nil {
		return entry, fmt.Errorf("decoding entry value: %w", err)
	}
	entry.Value = value
	offset += valueBytesRead

	tags, err := c.decodeTags(data, offset)
	if err != nil {
		return entry, fmt.Errorf("decoding entry tags: %w", err)
	}
	entry.Tags = tags

	return entry, nil
}

// calculateTagsSize calculates the total size needed for tags.
//
// Takes tags ([]string) which contains the tags to measure.
//
// Returns int which is the total byte size including length prefixes.
func (*BinaryCodec[K, V]) calculateTagsSize(tags []string) int {
	size := 0
	for _, tag := range tags {
		size += uint16Size + len(tag)
	}
	return size
}

// decodeKey decodes a key from the binary data at the given offset.
//
// Takes data ([]byte) which contains the encoded binary data.
// Takes offset (int) which specifies the position to start decoding from.
//
// Returns K which is the decoded key value.
// Returns int which is the new offset after decoding.
// Returns error when decoding fails.
func (c *BinaryCodec[K, V]) decodeKey(data []byte, offset int) (K, int, error) {
	return decodeComponent(data, offset, "key", c.keyCodec.DecodeKey)
}

// decodeValue reads the value length and value data from the buffer.
//
// Takes data ([]byte) which contains the encoded buffer to read from.
// Takes offset (int) which specifies the starting position in the buffer.
//
// Returns V which is the decoded value.
// Returns int which is the number of bytes consumed.
// Returns error when the buffer is truncated or the value length is invalid.
func (c *BinaryCodec[K, V]) decodeValue(data []byte, offset int) (V, int, error) {
	return decodeComponent(data, offset, "value", c.valueCodec.DecodeValue)
}

// decodeTags reads the tag count and all tags from the buffer.
//
// Takes data ([]byte) which contains the encoded binary data.
// Takes offset (int) which specifies the position to start reading from.
//
// Returns []string which contains the decoded tags.
// Returns error when the data is truncated or contains invalid values.
func (*BinaryCodec[K, V]) decodeTags(data []byte, offset int) ([]string, error) {
	if offset+uint16Size > len(data) {
		return nil, fmt.Errorf("%w: truncated at tag count", wal_domain.ErrInvalidEntry)
	}

	tagCount := binary.BigEndian.Uint16(data[offset:])
	offset += uint16Size

	if tagCount > maxTagCount {
		return nil, fmt.Errorf("%w: too many tags %d", wal_domain.ErrInvalidEntry, tagCount)
	}

	if tagCount == 0 {
		return nil, nil
	}

	tags := make([]string, 0, tagCount)
	for range tagCount {
		if offset+uint16Size > len(data) {
			return nil, fmt.Errorf("%w: truncated at tag length", wal_domain.ErrInvalidEntry)
		}

		tagLen := binary.BigEndian.Uint16(data[offset:])
		offset += uint16Size

		if tagLen > maxTagSize || offset+int(tagLen) > len(data) {
			return nil, fmt.Errorf("%w: invalid tag length %d", wal_domain.ErrInvalidEntry, tagLen)
		}

		tags = append(tags, string(data[offset:offset+int(tagLen)]))
		offset += int(tagLen)
	}

	return tags, nil
}

// EncodeResult holds the result of an encode operation.
// The caller must call Release when done to return the buffer to the pool.
type EncodeResult struct {
	// pool is the buffer pool for returning Data; nil means the buffer is not pooled.
	pool *[]byte

	// Data holds the encoded bytes, including the CRC prefix.
	Data []byte
}

// Release returns the buffer to the pool. Safe to call multiple times, and is
// a no-op if the buffer was not pooled.
func (r *EncodeResult) Release() {
	if r.pool != nil {
		PutByteBuffer(r.pool, r.Data)
		r.pool = nil
		r.Data = nil
	}
}

// EncodeWithLengthAndCRC encodes an entry with a length prefix and CRC32
// checksum using a pooled buffer. This removes the need for the caller to
// allocate a separate buffer for the length prefix.
//
// The returned EncodeResult MUST have Release() called when the caller is done
// with the data to return the buffer to the pool.
//
// Format: [Length:4][CRC32:4][Payload]
// Where Length is the size of CRC32+Payload (i.e., crcSize + payloadSize).
//
// Takes entry (wal_domain.Entry[K, V]) which is the entry to encode.
//
// Returns EncodeResult which contains the encoded data and pooled buffer.
// Returns error when encoding fails.
func (c *BinaryCodec[K, V]) EncodeWithLengthAndCRC(entry wal_domain.Entry[K, V]) (EncodeResult, error) {
	if c.fastKeyCodec != nil && c.fastValueCodec != nil {
		return c.encodeWithLengthAndCRCFast(entry)
	}

	payload, err := c.Encode(entry)
	if err != nil {
		return EncodeResult{}, fmt.Errorf("encoding entry payload: %w", err)
	}

	crcPlusPayloadSize := crcSize + len(payload)
	totalSize := lengthSize + crcPlusPayloadSize

	pool, buffer := GetByteBuffer(totalSize)
	buffer = buffer[:totalSize]

	binary.BigEndian.PutUint32(buffer[0:lengthSize], safeconv.IntToUint32(crcPlusPayloadSize))

	checksum := crc32.Checksum(payload, crcTable)
	binary.BigEndian.PutUint32(buffer[lengthSize:lengthSize+crcSize], checksum)

	copy(buffer[lengthSize+crcSize:], payload)

	return EncodeResult{Data: buffer, pool: pool}, nil
}

// EncodeWithCRC encodes an entry with CRC32 checksum using a pooled buffer.
//
// The returned EncodeResult MUST have Release() called when the caller is done
// with the data to return the buffer to the pool.
//
// When FastKeyCodec and FastValueCodec are available, this method achieves
// zero allocations by encoding directly into the pooled buffer. Otherwise,
// it falls back to allocating for the encode then copying into a pooled buffer.
//
// Format: [CRC32:4][Payload]
//
// Takes entry (wal_domain.Entry[K, V]) which is the entry to encode.
//
// Returns EncodeResult which contains the encoded data with CRC prefix.
// Returns error when encoding fails.
func (c *BinaryCodec[K, V]) EncodeWithCRC(entry wal_domain.Entry[K, V]) (EncodeResult, error) {
	if c.fastKeyCodec != nil && c.fastValueCodec != nil {
		return c.encodeWithCRCFast(entry)
	}

	payload, err := c.Encode(entry)
	if err != nil {
		return EncodeResult{}, fmt.Errorf("encoding entry with CRC: %w", err)
	}

	totalSize := crcSize + len(payload)

	pool, buffer := GetByteBuffer(totalSize)
	buffer = buffer[:totalSize]

	checksum := crc32.Checksum(payload, crcTable)
	binary.BigEndian.PutUint32(buffer[0:crcSize], checksum)

	copy(buffer[crcSize:], payload)

	return EncodeResult{Data: buffer, pool: pool}, nil
}

// encodeWithCRCFast is the zero-allocation fast path using fast codecs.
//
// Takes entry (wal_domain.Entry[K, V]) which contains the data to encode.
//
// Returns EncodeResult which holds the encoded buffer with CRC prefix.
// Returns error when key or value encoding fails.
func (c *BinaryCodec[K, V]) encodeWithCRCFast(entry wal_domain.Entry[K, V]) (EncodeResult, error) {
	keySize := c.fastKeyCodec.KeySize(entry.Key)
	valueSize := c.getValueSize(entry)
	tagsSize := c.calculateTagsSize(entry.Tags)
	payloadSize := fixedPayloadSize + uint32Size + keySize + uint32Size + valueSize + uint16Size + tagsSize
	totalSize := crcSize + payloadSize

	pool, buffer := GetByteBuffer(totalSize)
	buffer = buffer[:totalSize]

	payload := buffer[crcSize:]
	offset := encodeFixedFields(payload, entry.Operation, entry.Timestamp, entry.ExpiresAt)

	var err error
	offset, err = c.encodeKeyFast(payload, offset, entry.Key, keySize)
	if err != nil {
		PutByteBuffer(pool, buffer)
		return EncodeResult{}, fmt.Errorf("encoding key with CRC (fast): %w", err)
	}

	offset, err = c.encodeValueFast(payload, offset, entry, valueSize)
	if err != nil {
		PutByteBuffer(pool, buffer)
		return EncodeResult{}, fmt.Errorf("encoding value with CRC (fast): %w", err)
	}

	encodeTags(payload, offset, entry.Tags)

	checksum := crc32.Checksum(payload, crcTable)
	binary.BigEndian.PutUint32(buffer[0:crcSize], checksum)

	return EncodeResult{Data: buffer, pool: pool}, nil
}

// encodeWithLengthAndCRCFast is the zero-allocation fast path for
// EncodeWithLengthAndCRC.
//
// Format: [Length:4][CRC32:4][Payload]
//
// Takes entry (wal_domain.Entry[K, V]) which is the WAL entry to encode.
//
// Returns EncodeResult which contains the encoded bytes with length prefix
// and CRC.
// Returns error when key or value encoding fails.
func (c *BinaryCodec[K, V]) encodeWithLengthAndCRCFast(entry wal_domain.Entry[K, V]) (EncodeResult, error) {
	keySize := c.fastKeyCodec.KeySize(entry.Key)
	valueSize := c.getValueSize(entry)
	tagsSize := c.calculateTagsSize(entry.Tags)
	payloadSize := fixedPayloadSize + uint32Size + keySize + uint32Size + valueSize + uint16Size + tagsSize
	crcPlusPayloadSize := crcSize + payloadSize
	totalSize := lengthSize + crcPlusPayloadSize

	pool, buffer := GetByteBuffer(totalSize)
	buffer = buffer[:totalSize]

	binary.BigEndian.PutUint32(buffer[0:lengthSize], safeconv.IntToUint32(crcPlusPayloadSize))

	payload := buffer[lengthSize+crcSize:]
	offset := encodeFixedFields(payload, entry.Operation, entry.Timestamp, entry.ExpiresAt)

	var err error
	offset, err = c.encodeKeyFast(payload, offset, entry.Key, keySize)
	if err != nil {
		PutByteBuffer(pool, buffer)
		return EncodeResult{}, fmt.Errorf("encoding key with length and CRC (fast): %w", err)
	}

	offset, err = c.encodeValueFast(payload, offset, entry, valueSize)
	if err != nil {
		PutByteBuffer(pool, buffer)
		return EncodeResult{}, fmt.Errorf("encoding value with length and CRC (fast): %w", err)
	}

	encodeTags(payload, offset, entry.Tags)

	checksum := crc32.Checksum(payload, crcTable)
	binary.BigEndian.PutUint32(buffer[lengthSize:lengthSize+crcSize], checksum)

	return EncodeResult{Data: buffer, pool: pool}, nil
}

// DecodeResult holds the result of a decode operation.
// The caller must call Release when done to return the tag slice to the pool.
//
// When using fast codecs, the Entry key, value, and tags may reference the
// original input data buffer. The caller must ensure the data buffer outlives
// the Entry, or copy values before the buffer is reused.
type DecodeResult[K comparable, V any] struct {
	// tagPool is the pool pointer used to return the tag slice when done.
	tagPool *[]string

	// Entry is the decoded entry from the write-ahead log.
	Entry wal_domain.Entry[K, V]
}

// Release returns the tag slice to the pool for reuse.
//
// Safe to call multiple times. This is a no-op if the tags were not pooled.
func (r *DecodeResult[K, V]) Release() {
	if r.tagPool != nil {
		PutTagSlice(r.tagPool, r.Entry.Tags)
		r.tagPool = nil
		r.Entry.Tags = nil
	}
}

// DecodeWithCRC decodes an entry with CRC validation using pooled resources.
// The returned DecodeResult MUST have Release() called when the caller is done.
//
// When FastKeyCodec and FastValueCodec are available, this method achieves
// zero allocations by using mem.String for string fields. Otherwise, it falls
// back to standard decoding which allocates.
//
// SAFETY: The decoded Entry may reference the input data buffer. The caller
// must ensure the buffer outlives the Entry, or copy values if needed.
//
// Format: [CRC32:4][Payload]
//
// Takes data ([]byte) which contains the CRC-prefixed encoded entry.
//
// Returns DecodeResult[K, V] which holds the decoded entry and pool
// reference.
// Returns error when the CRC check fails or decoding fails.
func (c *BinaryCodec[K, V]) DecodeWithCRC(data []byte) (DecodeResult[K, V], error) {
	if len(data) < crcSize+minPayloadSize {
		return DecodeResult[K, V]{}, fmt.Errorf("%w: data too short (%d bytes)", wal_domain.ErrCorrupted, len(data))
	}

	storedCRC := binary.BigEndian.Uint32(data[0:crcSize])
	payload := data[crcSize:]
	computedCRC := crc32.Checksum(payload, crcTable)

	if storedCRC != computedCRC {
		return DecodeResult[K, V]{}, fmt.Errorf("%w: CRC mismatch (stored=%08x, computed=%08x)",
			wal_domain.ErrCorrupted, storedCRC, computedCRC)
	}

	return c.Decode(payload)
}

// Decode deserialises bytes to an entry using pooled resources.
// The returned DecodeResult MUST have Release() called when the caller is done.
//
// The input does NOT include length prefix or CRC32 - those are stripped
// by the WAL implementation before calling this method.
//
// When FastKeyCodec and FastValueCodec are available, this method achieves
// zero allocations. Otherwise, it falls back to standard decoding.
//
// SAFETY: The decoded Entry may reference the input data buffer. The caller
// must ensure the buffer outlives the Entry, or copy values if needed.
//
// Takes data ([]byte) which contains the raw encoded entry payload.
//
// Returns DecodeResult[K, V] which holds the decoded entry and pool
// reference.
// Returns error when decoding fails.
func (c *BinaryCodec[K, V]) Decode(data []byte) (DecodeResult[K, V], error) {
	if c.fastKeyCodec != nil && c.fastValueCodec != nil {
		return c.decodeFast(data)
	}
	entry, err := c.decodeSlow(data)
	if err != nil {
		return DecodeResult[K, V]{}, fmt.Errorf("decoding entry (slow path): %w", err)
	}
	return DecodeResult[K, V]{Entry: entry}, nil
}

// decodeFast is the zero-allocation fast path using fast codecs and
// mem.String.
//
// Takes data ([]byte) which contains the raw encoded entry payload.
//
// Returns DecodeResult[K, V] which holds the decoded entry and pool
// reference.
// Returns error when the payload is too short, the version is invalid,
// or decoding fails.
func (c *BinaryCodec[K, V]) decodeFast(data []byte) (DecodeResult[K, V], error) {
	var result DecodeResult[K, V]

	if len(data) < minPayloadSize {
		return result, fmt.Errorf("%w: payload too short (%d bytes)", wal_domain.ErrInvalidEntry, len(data))
	}

	offset := 0

	version := data[offset]
	offset++
	if version != formatVersion {
		return result, fmt.Errorf("%w: got version %d, expected %d", wal_domain.ErrInvalidVersion, version, formatVersion)
	}

	result.Entry.Operation = wal_domain.Operation(data[offset])
	offset++
	if !result.Entry.Operation.IsValid() {
		return result, fmt.Errorf("%w: invalid operation %d", wal_domain.ErrInvalidEntry, result.Entry.Operation)
	}

	result.Entry.Timestamp = safeconv.Uint64ToInt64(binary.BigEndian.Uint64(data[offset:]))
	offset += uint64Size

	result.Entry.ExpiresAt = safeconv.Uint64ToInt64(binary.BigEndian.Uint64(data[offset:]))
	offset += uint64Size

	key, keyBytesRead, err := c.decodeKeyFast(data, offset)
	if err != nil {
		return result, fmt.Errorf("decoding key (fast): %w", err)
	}
	result.Entry.Key = key
	offset += keyBytesRead

	value, valueBytesRead, err := c.decodeValueFast(data, offset)
	if err != nil {
		return result, fmt.Errorf("decoding value (fast): %w", err)
	}
	result.Entry.Value = value
	offset += valueBytesRead

	tagPool, tags, err := c.decodeTagsPooled(data, offset)
	if err != nil {
		return result, fmt.Errorf("decoding tags (fast): %w", err)
	}
	result.Entry.Tags = tags
	result.tagPool = tagPool

	return result, nil
}

// decodeKeyFast decodes a key using the fast codec.
//
// Takes data ([]byte) which contains the encoded key bytes.
// Takes offset (int) which specifies the starting position in data.
//
// Returns K which is the decoded key.
// Returns int which is the number of bytes consumed.
// Returns error when the data is truncated or the key length is invalid.
func (c *BinaryCodec[K, V]) decodeKeyFast(data []byte, offset int) (K, int, error) {
	return decodeComponent(data, offset, "key", c.fastKeyCodec.DecodeKeyFrom)
}

// decodeValueFast decodes a value using the fast codec.
//
// Takes data ([]byte) which contains the encoded binary data.
// Takes offset (int) which specifies the starting position in data.
//
// Returns V which is the decoded value.
// Returns int which is the number of bytes consumed from data.
// Returns error when the data is truncated or the value length is invalid.
func (c *BinaryCodec[K, V]) decodeValueFast(data []byte, offset int) (V, int, error) {
	return decodeComponent(data, offset, "value", c.fastValueCodec.DecodeValueFrom)
}

// decodeTagsPooled decodes tags using a pooled slice and mem.String for
// zero-copy.
//
// Takes data ([]byte) which contains the encoded binary data.
// Takes offset (int) which specifies the position to start reading from.
//
// Returns *[]string which is the pooled slice for returning to the pool.
// Returns []string which contains the decoded tag values.
// Returns error when the data is truncated or contains invalid tag values.
func (*BinaryCodec[K, V]) decodeTagsPooled(data []byte, offset int) (*[]string, []string, error) {
	if offset+uint16Size > len(data) {
		return nil, nil, fmt.Errorf("%w: truncated at tag count", wal_domain.ErrInvalidEntry)
	}

	tagCount := binary.BigEndian.Uint16(data[offset:])
	offset += uint16Size

	if tagCount > maxTagCount {
		return nil, nil, fmt.Errorf("%w: too many tags %d", wal_domain.ErrInvalidEntry, tagCount)
	}

	if tagCount == 0 {
		return nil, nil, nil
	}

	tagPool, tags := GetTagSlice()

	for range tagCount {
		if offset+uint16Size > len(data) {
			PutTagSlice(tagPool, tags)
			return nil, nil, fmt.Errorf("%w: truncated at tag length", wal_domain.ErrInvalidEntry)
		}

		tagLen := binary.BigEndian.Uint16(data[offset:])
		offset += uint16Size

		if tagLen > maxTagSize || offset+int(tagLen) > len(data) {
			PutTagSlice(tagPool, tags)
			return nil, nil, fmt.Errorf("%w: invalid tag length %d", wal_domain.ErrInvalidEntry, tagLen)
		}

		tags = append(tags, mem.String(data[offset:offset+int(tagLen)]))
		offset += int(tagLen)
	}

	return tagPool, tags, nil
}

// NewBinaryCodec creates a new BinaryCodec with the provided key and value
// codecs. If the codecs implement FastKeyCodec or FastValueCodec, those
// interfaces will be used for zero-allocation encoding in the hot path.
//
// Takes keyCodec (wal_domain.KeyCodec[K]) which encodes and decodes keys.
// Takes valueCodec (wal_domain.ValueCodec[V]) which encodes and decodes
// values.
//
// Returns *BinaryCodec[K, V] which is the configured codec ready for use.
func NewBinaryCodec[K comparable, V any](
	keyCodec wal_domain.KeyCodec[K],
	valueCodec wal_domain.ValueCodec[V],
) *BinaryCodec[K, V] {
	codec := &BinaryCodec[K, V]{
		keyCodec:   keyCodec,
		valueCodec: valueCodec,
	}

	//nolint:govet // generic assertion valid at runtime
	if fk, ok := keyCodec.(wal_domain.FastKeyCodec[K]); ok {
		codec.fastKeyCodec = fk
	}
	if fv, ok := valueCodec.(wal_domain.FastValueCodec[V]); ok { //nolint:govet // generic assertion valid at runtime
		codec.fastValueCodec = fv
	}

	return codec
}

// ValidateCRC checks if the CRC in the data is valid without decoding.
// Use it to scan the WAL and find corruption points.
//
// Takes data ([]byte) which contains the CRC followed by the payload.
//
// Returns bool which is true when the stored CRC matches the computed CRC.
func ValidateCRC(data []byte) bool {
	if len(data) < crcSize+minPayloadSize {
		return false
	}

	storedCRC := binary.BigEndian.Uint32(data[0:crcSize])
	payload := data[crcSize:]
	computedCRC := crc32.Checksum(payload, crcTable)

	return storedCRC == computedCRC
}

// ComputeCRC computes the CRC32 checksum for the given data.
//
// Takes data ([]byte) which is the input bytes to compute the checksum for.
//
// Returns uint32 which is the computed CRC32 checksum.
func ComputeCRC(data []byte) uint32 {
	return crc32.Checksum(data, crcTable)
}

// encodeFixedFields writes version, operation, timestamp and expiresAt.
//
// Takes buffer ([]byte) which is the destination buffer for the encoded fields.
// Takes op (wal_domain.Operation) which specifies the operation type.
// Takes timestamp (int64) which is the record creation time.
// Takes expiresAt (int64) which is the expiry time for the record.
//
// Returns int which is the offset after writing all fields.
func encodeFixedFields(buffer []byte, op wal_domain.Operation, timestamp, expiresAt int64) int {
	offset := 0
	buffer[offset] = formatVersion
	offset++
	buffer[offset] = byte(op)
	offset++
	binary.BigEndian.PutUint64(buffer[offset:], safeconv.Int64ToUint64(timestamp))
	offset += uint64Size
	binary.BigEndian.PutUint64(buffer[offset:], safeconv.Int64ToUint64(expiresAt))
	offset += uint64Size
	return offset
}

// encodeBytes writes a length-prefixed byte slice to the buffer.
//
// Takes buffer ([]byte) which is the destination buffer to write into.
// Takes offset (int) which is the position to start writing at.
// Takes data ([]byte) which is the byte slice to encode.
//
// Returns int which is the offset after writing the length prefix and data.
func encodeBytes(buffer []byte, offset int, data []byte) int {
	binary.BigEndian.PutUint32(buffer[offset:], safeconv.IntToUint32(len(data)))
	offset += uint32Size
	copy(buffer[offset:], data)
	return offset + len(data)
}

// encodeTags writes the tag count and all tags to the buffer.
//
// Takes buffer ([]byte) which is the destination buffer to write into.
// Takes offset (int) which is the starting position in the buffer.
// Takes tags ([]string) which contains the tags to encode.
func encodeTags(buffer []byte, offset int, tags []string) {
	binary.BigEndian.PutUint16(buffer[offset:], safeconv.IntToUint16(len(tags)))
	offset += uint16Size

	for _, tag := range tags {
		binary.BigEndian.PutUint16(buffer[offset:], safeconv.IntToUint16(len(tag)))
		offset += uint16Size
		copy(buffer[offset:], tag)
		offset += len(tag)
	}
}

// decodeComponent reads a length-prefixed component from the buffer.
//
// Takes data ([]byte) which contains the encoded bytes.
// Takes offset (int) which is the position to start reading from.
// Takes compName (string) which is the name used in error messages.
// Takes decode (func(...)) which parses the raw bytes into type T.
//
// Returns T which is the decoded component value.
// Returns int which is the number of bytes consumed.
// Returns error when the data is truncated or the length is invalid.
func decodeComponent[T any](
	data []byte,
	offset int,
	compName string,
	decode func([]byte) (T, error),
) (T, int, error) {
	var zero T

	if offset+uint32Size > len(data) {
		return zero, 0, fmt.Errorf("%w: truncated at %s length", wal_domain.ErrInvalidEntry, compName)
	}

	lenVal := binary.BigEndian.Uint32(data[offset:])
	offset += uint32Size

	if lenVal > maxEntrySize || offset+int(lenVal) > len(data) {
		return zero, 0, fmt.Errorf("%w: invalid %s length %d", wal_domain.ErrInvalidEntry, compName, lenVal)
	}

	if lenVal > 0 {
		value, err := decode(data[offset : offset+int(lenVal)])
		if err != nil {
			return zero, 0, fmt.Errorf("decoding %s: %w", compName, err)
		}
		return value, uint32Size + int(lenVal), nil
	}

	return zero, uint32Size, nil
}
