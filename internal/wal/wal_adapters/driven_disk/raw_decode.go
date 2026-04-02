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

	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/safeconv"
)

// DecodeRawEntries decodes a complete WAL file into raw entries without
// requiring typed key/value codecs. This is intended for inspection and
// debugging tools.
//
// The WAL wire format is a sequence of entries, each prefixed with:
// [Length:4][CRC32:4][Payload]
//
// Takes data ([]byte) which is the raw WAL file contents.
//
// Returns []wal_domain.RawEntry which contains all decoded entries.
// Returns error when the data contains an invalid or truncated entry.
func DecodeRawEntries(data []byte) ([]wal_domain.RawEntry, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var entries []wal_domain.RawEntry
	offset := 0
	index := 0

	for offset < len(data) {
		if offset+lengthSize > len(data) {
			break
		}

		entryStart := offset
		length := binary.BigEndian.Uint32(data[offset:])
		offset += lengthSize

		if length > maxEntrySize || length < uint32(crcSize+minPayloadSize) {
			return nil, fmt.Errorf("entry %d: invalid length %d at byte offset %d", index, length, entryStart)
		}

		if offset+int(length) > len(data) {
			return nil, fmt.Errorf("entry %d: truncated at byte offset %d (need %d bytes, have %d)",
				index, entryStart, length, len(data)-offset)
		}

		entryData := data[offset : offset+int(length)]
		offset += int(length)

		entry, err := DecodeRawEntry(entryData)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", index, err)
		}
		entry.SizeBytes = lengthSize + int(length)

		entries = append(entries, entry)
		index++
	}

	return entries, nil
}

// DecodeRawEntry decodes a single [CRC32:4][Payload] buffer into a RawEntry.
// Key and value are returned as raw byte slices referencing the input data.
//
// Takes crcAndPayload ([]byte) which is the CRC followed by the entry payload.
//
// Returns wal_domain.RawEntry which is the decoded entry.
// Returns error when the data is malformed or truncated.
func DecodeRawEntry(crcAndPayload []byte) (wal_domain.RawEntry, error) {
	if len(crcAndPayload) < crcSize+minPayloadSize {
		return wal_domain.RawEntry{}, fmt.Errorf("%w: too short (%d bytes)", wal_domain.ErrInvalidEntry, len(crcAndPayload))
	}

	storedCRC := binary.BigEndian.Uint32(crcAndPayload[:crcSize])
	payload := crcAndPayload[crcSize:]
	computedCRC := crc32.Checksum(payload, crcTable)

	offset := 0

	version := payload[offset]
	offset++
	if version != formatVersion {
		return wal_domain.RawEntry{}, fmt.Errorf("%w: got version %d, expected %d",
			wal_domain.ErrInvalidVersion, version, formatVersion)
	}

	operation := wal_domain.Operation(payload[offset])
	offset++
	if !operation.IsValid() {
		return wal_domain.RawEntry{}, fmt.Errorf("%w: invalid operation %d", wal_domain.ErrInvalidEntry, operation)
	}

	timestamp := safeconv.Uint64ToInt64(binary.BigEndian.Uint64(payload[offset:]))
	offset += uint64Size

	expiresAt := safeconv.Uint64ToInt64(binary.BigEndian.Uint64(payload[offset:]))
	offset += uint64Size

	keyBytes, n, err := readRawLengthPrefixed(payload, offset, "key")
	if err != nil {
		return wal_domain.RawEntry{}, err
	}
	offset += n

	valueBytes, n, err := readRawLengthPrefixed(payload, offset, "value")
	if err != nil {
		return wal_domain.RawEntry{}, err
	}
	offset += n

	tags, err := decodeRawTags(payload, offset)
	if err != nil {
		return wal_domain.RawEntry{}, err
	}

	return wal_domain.RawEntry{
		Key:       keyBytes,
		Value:     valueBytes,
		Tags:      tags,
		ExpiresAt: expiresAt,
		Timestamp: timestamp,
		Operation: operation,
		CRCValid:  storedCRC == computedCRC,
	}, nil
}

// readRawLengthPrefixed reads a uint32 length-prefixed byte slice.
//
// Takes data ([]byte) which is the source buffer.
// Takes offset (int) which is the reading position.
// Takes name (string) which identifies the field for error messages.
//
// Returns []byte which is the extracted data (a sub-slice of the input).
// Returns int which is the total bytes consumed (length prefix + data).
// Returns error when the buffer is truncated.
func readRawLengthPrefixed(data []byte, offset int, name string) ([]byte, int, error) {
	if offset+uint32Size > len(data) {
		return nil, 0, fmt.Errorf("%w: truncated at %s length", wal_domain.ErrInvalidEntry, name)
	}

	length := binary.BigEndian.Uint32(data[offset:])
	offset += uint32Size

	if length > maxEntrySize || int(length) > len(data)-offset {
		return nil, 0, fmt.Errorf("%w: invalid %s length %d", wal_domain.ErrInvalidEntry, name, length)
	}

	return data[offset : offset+int(length)], uint32Size + int(length), nil
}

// decodeRawTags decodes the tag section of a WAL entry.
//
// Takes data ([]byte) which contains the remaining payload.
// Takes offset (int) which is the position of the tag count.
//
// Returns []string which contains the decoded tags.
// Returns error when the tag data is malformed.
func decodeRawTags(data []byte, offset int) ([]string, error) {
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

		if tagLen > maxTagSize || int(tagLen) > len(data)-offset {
			return nil, fmt.Errorf("%w: invalid tag length %d", wal_domain.ErrInvalidEntry, tagLen)
		}

		tags = append(tags, string(data[offset:offset+int(tagLen)]))
		offset += int(tagLen)
	}

	return tags, nil
}
