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
	"hash/crc32"
	"testing"

	"piko.sh/piko/internal/wal/wal_domain"
)

func TestDecodeRawEntries(t *testing.T) {
	t.Parallel()

	entry1 := buildRawTestEntry("key1", "value1", wal_domain.OpSet, 1000000000000000000, 0, nil)
	entry2 := buildRawTestEntry("key2", "", wal_domain.OpDelete, 1000000000000000000, 0, nil)

	data := make([]byte, 0, len(entry1)+len(entry2))
	data = append(data, entry1...)
	data = append(data, entry2...)

	entries, err := DecodeRawEntries(data)
	if err != nil {
		t.Fatalf("DecodeRawEntries: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("entry count = %d, want 2", len(entries))
	}

	if string(entries[0].Key) != "key1" {
		t.Errorf("entry 0 key = %q, want %q", entries[0].Key, "key1")
	}
	if entries[0].Operation != wal_domain.OpSet {
		t.Errorf("entry 0 op = %v, want OpSet", entries[0].Operation)
	}
	if string(entries[0].Value) != "value1" {
		t.Errorf("entry 0 value = %q, want %q", entries[0].Value, "value1")
	}
	if !entries[0].CRCValid {
		t.Error("entry 0 CRC should be valid")
	}
	if entries[0].SizeBytes == 0 {
		t.Error("entry 0 SizeBytes should be non-zero")
	}

	if string(entries[1].Key) != "key2" {
		t.Errorf("entry 1 key = %q, want %q", entries[1].Key, "key2")
	}
	if entries[1].Operation != wal_domain.OpDelete {
		t.Errorf("entry 1 op = %v, want OpDelete", entries[1].Operation)
	}
	if !entries[1].CRCValid {
		t.Error("entry 1 CRC should be valid")
	}
}

func TestDecodeRawEntries_Empty(t *testing.T) {
	t.Parallel()

	entries, err := DecodeRawEntries(nil)
	if err != nil {
		t.Fatalf("DecodeRawEntries(nil): %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entry count = %d, want 0", len(entries))
	}
}

func TestDecodeRawEntries_WithTags(t *testing.T) {
	t.Parallel()

	data := buildRawTestEntry("k", "v", wal_domain.OpSet, 1000000000000000000, 0, []string{"tag1", "tag2"})
	entries, err := DecodeRawEntries(data)
	if err != nil {
		t.Fatalf("DecodeRawEntries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}
	if len(entries[0].Tags) != 2 {
		t.Fatalf("tags count = %d, want 2", len(entries[0].Tags))
	}
	if entries[0].Tags[0] != "tag1" || entries[0].Tags[1] != "tag2" {
		t.Errorf("tags = %v, want [tag1 tag2]", entries[0].Tags)
	}
}

func TestDecodeRawEntries_WithExpiry(t *testing.T) {
	t.Parallel()

	data := buildRawTestEntry("k", "v", wal_domain.OpSet, 1000000000000000000, 2000000000000000000, nil)
	entries, err := DecodeRawEntries(data)
	if err != nil {
		t.Fatalf("DecodeRawEntries: %v", err)
	}

	if entries[0].ExpiresAt != 2000000000000000000 {
		t.Errorf("expiresAt = %d, want 2000000000000000000", entries[0].ExpiresAt)
	}
}

func TestDecodeRawEntries_ClearOperation(t *testing.T) {
	t.Parallel()

	data := buildRawTestEntry("", "", wal_domain.OpClear, 1000000000000000000, 0, nil)
	entries, err := DecodeRawEntries(data)
	if err != nil {
		t.Fatalf("DecodeRawEntries: %v", err)
	}

	if entries[0].Operation != wal_domain.OpClear {
		t.Errorf("op = %v, want OpClear", entries[0].Operation)
	}
}

func TestDecodeRawEntries_InvalidLength(t *testing.T) {
	t.Parallel()

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, 1)
	_, err := DecodeRawEntries(data)
	if err == nil {
		t.Fatal("expected error for invalid length")
	}
}

func TestDecodeRawEntries_Truncated(t *testing.T) {
	t.Parallel()

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, 100)
	_, err := DecodeRawEntries(data)
	if err == nil {
		t.Fatal("expected error for truncated entry")
	}
}

func TestDecodeRawEntry_CorruptedCRC(t *testing.T) {
	t.Parallel()

	data := buildRawTestEntry("key", "value", wal_domain.OpSet, 1000000000000000000, 0, nil)

	crcAndPayload := data[lengthSize:]
	crcAndPayload[0] ^= 0xFF

	entry, err := DecodeRawEntry(crcAndPayload)
	if err != nil {
		t.Fatalf("DecodeRawEntry should succeed even with bad CRC: %v", err)
	}
	if entry.CRCValid {
		t.Error("CRCValid should be false for corrupted entry")
	}
}

func TestDecodeRawEntry_InvalidVersion(t *testing.T) {
	t.Parallel()

	data := buildRawTestEntry("key", "value", wal_domain.OpSet, 1000000000000000000, 0, nil)
	crcAndPayload := data[lengthSize:]

	crcAndPayload[crcSize] = 99

	_, err := DecodeRawEntry(crcAndPayload)
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestDecodeRawEntry_BinaryKeyValue(t *testing.T) {
	t.Parallel()

	binaryKey := []byte{0xFF, 0xFE, 0xFD}
	binaryValue := []byte{0x00, 0x01, 0x02, 0x03}

	data := buildRawTestEntryBytes(binaryKey, binaryValue, wal_domain.OpSet, 1000000000000000000, 0, nil)
	entries, err := DecodeRawEntries(data)
	if err != nil {
		t.Fatalf("DecodeRawEntries: %v", err)
	}

	if string(entries[0].Key) != string(binaryKey) {
		t.Errorf("key bytes mismatch")
	}
	if string(entries[0].Value) != string(binaryValue) {
		t.Errorf("value bytes mismatch")
	}
}

func TestDecodeRawEntries_MatchesCodec(t *testing.T) {
	t.Parallel()

	codec := NewBinaryCodec[string, []byte](stringKeyCodec{}, bytesValueCodec{})

	entry := wal_domain.Entry[string, []byte]{
		Key:       "test-key",
		Value:     []byte("test-value"),
		Tags:      []string{"alpha", "beta"},
		ExpiresAt: 2000000000000000000,
		Timestamp: 1000000000000000000,
		Operation: wal_domain.OpSet,
	}

	encoded, err := codec.EncodeWithLengthAndCRC(entry)
	if err != nil {
		t.Fatalf("EncodeWithLengthAndCRC: %v", err)
	}
	defer encoded.Release()

	raw, err := DecodeRawEntries(encoded.Data)
	if err != nil {
		t.Fatalf("DecodeRawEntries: %v", err)
	}

	if len(raw) != 1 {
		t.Fatalf("entry count = %d, want 1", len(raw))
	}

	r := raw[0]
	if string(r.Key) != "test-key" {
		t.Errorf("key = %q, want %q", r.Key, "test-key")
	}
	if string(r.Value) != "test-value" {
		t.Errorf("value = %q, want %q", r.Value, "test-value")
	}
	if r.Operation != wal_domain.OpSet {
		t.Errorf("op = %v, want OpSet", r.Operation)
	}
	if r.Timestamp != 1000000000000000000 {
		t.Errorf("timestamp = %d, want 1000000000000000000", r.Timestamp)
	}
	if r.ExpiresAt != 2000000000000000000 {
		t.Errorf("expiresAt = %d, want 2000000000000000000", r.ExpiresAt)
	}
	if len(r.Tags) != 2 || r.Tags[0] != "alpha" || r.Tags[1] != "beta" {
		t.Errorf("tags = %v, want [alpha beta]", r.Tags)
	}
	if !r.CRCValid {
		t.Error("CRC should be valid")
	}
}

func TestDecodeRawEntries_AlignmentPaddingAfterEntries(t *testing.T) {
	t.Parallel()

	entry := buildRawTestEntry("key", "value", wal_domain.OpSet, 1000000000000000000, 0, nil)

	padSize := alignment - (len(entry) % alignment)
	data := make([]byte, len(entry)+padSize)
	copy(data, entry)

	entries, err := DecodeRawEntries(data)
	if err != nil {
		t.Fatalf("DecodeRawEntries with alignment padding: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}
	if string(entries[0].Key) != "key" {
		t.Errorf("key = %q, want %q", entries[0].Key, "key")
	}
}

func TestDecodeRawEntries_AllZeros(t *testing.T) {
	t.Parallel()

	data := make([]byte, alignment)

	entries, err := DecodeRawEntries(data)
	if err != nil {
		t.Fatalf("DecodeRawEntries with all-zero data: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entry count = %d, want 0", len(entries))
	}
}

func TestDecodeRawEntries_MultipleEntriesWithPadding(t *testing.T) {
	t.Parallel()

	entry1 := buildRawTestEntry("key1", "value1", wal_domain.OpSet, 1000000000000000000, 0, nil)
	entry2 := buildRawTestEntry("key2", "value2", wal_domain.OpDelete, 1000000000000000000, 0, nil)

	entryData := make([]byte, 0, len(entry1)+len(entry2))
	entryData = append(entryData, entry1...)
	entryData = append(entryData, entry2...)

	padSize := (alignment - (len(entryData) % alignment)) % alignment
	data := make([]byte, len(entryData)+padSize+alignment)
	copy(data, entryData)

	entries, err := DecodeRawEntries(data)
	if err != nil {
		t.Fatalf("DecodeRawEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entry count = %d, want 2", len(entries))
	}
	if string(entries[0].Key) != "key1" {
		t.Errorf("entry 0 key = %q, want %q", entries[0].Key, "key1")
	}
	if string(entries[1].Key) != "key2" {
		t.Errorf("entry 1 key = %q, want %q", entries[1].Key, "key2")
	}
}

func TestDecodeRawEntries_ZeroPaddingShorterThanLengthPrefix(t *testing.T) {
	t.Parallel()

	entry := buildRawTestEntry("k", "v", wal_domain.OpSet, 1000000000000000000, 0, nil)

	data := make([]byte, len(entry)+3)
	copy(data, entry)

	entries, err := DecodeRawEntries(data)
	if err != nil {
		t.Fatalf("DecodeRawEntries with trailing short padding: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}
}

func buildRawTestEntry(key, value string, op wal_domain.Operation, timestamp, expiresAt int64, tags []string) []byte {
	return buildRawTestEntryBytes([]byte(key), []byte(value), op, timestamp, expiresAt, tags)
}

func buildRawTestEntryBytes(key, value []byte, op wal_domain.Operation, timestamp, expiresAt int64, tags []string) []byte {
	tagsSize := 0
	for _, tag := range tags {
		tagsSize += uint16Size + len(tag)
	}

	payloadSize := 1 + 1 + uint64Size + uint64Size + uint32Size + len(key) + uint32Size + len(value) + uint16Size + tagsSize
	payload := make([]byte, payloadSize)

	offset := 0
	payload[offset] = formatVersion
	offset++
	payload[offset] = byte(op)
	offset++
	binary.BigEndian.PutUint64(payload[offset:], uint64(timestamp))
	offset += uint64Size
	binary.BigEndian.PutUint64(payload[offset:], uint64(expiresAt))
	offset += uint64Size
	binary.BigEndian.PutUint32(payload[offset:], uint32(len(key)))
	offset += uint32Size
	copy(payload[offset:], key)
	offset += len(key)
	binary.BigEndian.PutUint32(payload[offset:], uint32(len(value)))
	offset += uint32Size
	copy(payload[offset:], value)
	offset += len(value)
	binary.BigEndian.PutUint16(payload[offset:], uint16(len(tags)))
	offset += uint16Size
	for _, tag := range tags {
		binary.BigEndian.PutUint16(payload[offset:], uint16(len(tag)))
		offset += uint16Size
		copy(payload[offset:], tag)
		offset += len(tag)
	}

	checksum := crc32.Checksum(payload, crcTable)

	crcAndPayload := make([]byte, crcSize+len(payload))
	binary.BigEndian.PutUint32(crcAndPayload[:crcSize], checksum)
	copy(crcAndPayload[crcSize:], payload)

	result := make([]byte, lengthSize+len(crcAndPayload))
	binary.BigEndian.PutUint32(result[:lengthSize], uint32(len(crcAndPayload)))
	copy(result[lengthSize:], crcAndPayload)

	return result
}

type bytesValueCodec struct{}

func (bytesValueCodec) EncodeValue(value []byte) ([]byte, error) { return value, nil }
func (bytesValueCodec) DecodeValue(data []byte) ([]byte, error)  { return data, nil }
