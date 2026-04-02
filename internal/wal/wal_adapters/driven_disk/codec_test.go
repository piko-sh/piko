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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/wal/wal_domain"
)

type stringKeyCodec struct{}

func (stringKeyCodec) EncodeKey(key string) ([]byte, error) {
	return []byte(key), nil
}

func (stringKeyCodec) DecodeKey(data []byte) (string, error) {
	return string(data), nil
}

func (stringKeyCodec) KeySize(key string) int {
	return len(key)
}

func (stringKeyCodec) EncodeKeyTo(key string, buffer []byte) (int, error) {
	return copy(buffer, key), nil
}

func (stringKeyCodec) DecodeKeyFrom(data []byte) (string, error) {
	return mem.String(data), nil
}

type testValue struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type binaryValueCodec struct{}

func (binaryValueCodec) EncodeValue(value testValue) ([]byte, error) {
	size := 4 + len(value.Name) + 8
	buffer := make([]byte, size)
	offset := 0

	binary.BigEndian.PutUint32(buffer[offset:], uint32(len(value.Name)))
	offset += 4

	copy(buffer[offset:], value.Name)
	offset += len(value.Name)

	binary.BigEndian.PutUint64(buffer[offset:], uint64(value.Count))

	return buffer, nil
}

func (binaryValueCodec) DecodeValue(data []byte) (testValue, error) {
	if len(data) < 12 {
		return testValue{}, fmt.Errorf("data too short: %d bytes", len(data))
	}

	offset := 0

	nameLen := binary.BigEndian.Uint32(data[offset:])
	offset += 4

	if len(data) < int(4+nameLen+8) {
		return testValue{}, fmt.Errorf("data too short for name: %d bytes", len(data))
	}

	name := string(data[offset : offset+int(nameLen)])
	offset += int(nameLen)

	count := int(binary.BigEndian.Uint64(data[offset:]))

	return testValue{Name: name, Count: count}, nil
}

func (binaryValueCodec) ValueSize(value testValue) int {
	return 4 + len(value.Name) + 8
}

func (binaryValueCodec) EncodeValueTo(value testValue, buffer []byte) (int, error) {
	offset := 0

	binary.BigEndian.PutUint32(buffer[offset:], uint32(len(value.Name)))
	offset += 4

	copy(buffer[offset:], value.Name)
	offset += len(value.Name)

	binary.BigEndian.PutUint64(buffer[offset:], uint64(value.Count))
	offset += 8

	return offset, nil
}

func (binaryValueCodec) DecodeValueFrom(data []byte) (testValue, error) {
	if len(data) < 12 {
		return testValue{}, fmt.Errorf("data too short: %d bytes", len(data))
	}

	offset := 0
	nameLen := binary.BigEndian.Uint32(data[offset:])
	offset += 4

	if len(data) < int(4+nameLen+8) {
		return testValue{}, fmt.Errorf("data too short for name: %d bytes", len(data))
	}

	name := mem.String(data[offset : offset+int(nameLen)])
	offset += int(nameLen)
	count := int(binary.BigEndian.Uint64(data[offset:]))

	return testValue{Name: name, Count: count}, nil
}

func newTestCodec() *BinaryCodec[string, testValue] {
	return NewBinaryCodec[string, testValue](stringKeyCodec{}, binaryValueCodec{})
}

func TestBinaryCodec_EncodeDecodeRoundtrip(t *testing.T) {
	codec := newTestCodec()

	testCases := []struct {
		name  string
		entry wal_domain.Entry[string, testValue]
	}{
		{
			name: "set operation with value and tags",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "test-key",
				Value:     testValue{Name: "test", Count: 42},
				Tags:      []string{"tag1", "tag2"},
				ExpiresAt: time.Now().Add(time.Hour).UnixNano(),
				Timestamp: time.Now().UnixNano(),
			},
		},
		{
			name: "set operation without tags",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "another-key",
				Value:     testValue{Name: "another", Count: 100},
				Tags:      nil,
				ExpiresAt: 0,
				Timestamp: time.Now().UnixNano(),
			},
		},
		{
			name: "delete operation",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpDelete,
				Key:       "delete-key",
				Tags:      nil,
				ExpiresAt: 0,
				Timestamp: time.Now().UnixNano(),
			},
		},
		{
			name: "clear operation",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpClear,
				Timestamp: time.Now().UnixNano(),
			},
		},
		{
			name: "empty key",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "",
				Value:     testValue{Name: "empty key", Count: 0},
				Timestamp: time.Now().UnixNano(),
			},
		},
		{
			name: "large number of tags",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "many-tags",
				Value:     testValue{Name: "tagged", Count: 1},
				Tags:      []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9", "t10"},
				Timestamp: time.Now().UnixNano(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			encoded, err := codec.Encode(tc.entry)
			require.NoError(t, err)
			require.NotEmpty(t, encoded)

			result, err := codec.Decode(encoded)
			require.NoError(t, err)
			defer result.Release()

			assert.Equal(t, tc.entry.Operation, result.Entry.Operation)
			assert.Equal(t, tc.entry.Key, result.Entry.Key)
			assert.Equal(t, tc.entry.Timestamp, result.Entry.Timestamp)
			assert.Equal(t, tc.entry.ExpiresAt, result.Entry.ExpiresAt)
			assert.Equal(t, tc.entry.Tags, result.Entry.Tags)

			if tc.entry.Operation == wal_domain.OpSet {
				assert.Equal(t, tc.entry.Value, result.Entry.Value)
			}
		})
	}
}

func TestBinaryCodec_EncodeDecodeWithCRC(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "crc-test",
		Value:     testValue{Name: "test", Count: 123},
		Tags:      []string{"tag"},
		Timestamp: time.Now().UnixNano(),
	}

	encResult, err := codec.EncodeWithCRC(entry)
	require.NoError(t, err)
	require.NotEmpty(t, encResult.Data)
	defer encResult.Release()

	assert.True(t, ValidateCRC(encResult.Data))

	decResult, err := codec.DecodeWithCRC(encResult.Data)
	require.NoError(t, err)
	defer decResult.Release()

	assert.Equal(t, entry.Operation, decResult.Entry.Operation)
	assert.Equal(t, entry.Key, decResult.Entry.Key)
	assert.Equal(t, entry.Value, decResult.Entry.Value)
}

func TestBinaryCodec_CRCDetectsCorruption(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "corruption-test",
		Value:     testValue{Name: "test", Count: 456},
		Timestamp: time.Now().UnixNano(),
	}

	encResult, err := codec.EncodeWithCRC(entry)
	require.NoError(t, err)
	defer encResult.Release()

	testCases := []struct {
		corruptFunc func([]byte) []byte
		name        string
	}{
		{
			name: "flip bit in CRC",
			corruptFunc: func(data []byte) []byte {
				corrupted := make([]byte, len(data))
				copy(corrupted, data)
				corrupted[0] ^= 0x01
				return corrupted
			},
		},
		{
			name: "flip bit in payload",
			corruptFunc: func(data []byte) []byte {
				corrupted := make([]byte, len(data))
				copy(corrupted, data)
				corrupted[len(corrupted)-1] ^= 0x01
				return corrupted
			},
		},
		{
			name: "truncate payload",
			corruptFunc: func(data []byte) []byte {
				return data[:len(data)-5]
			},
		},
		{
			name: "zero out middle bytes",
			corruptFunc: func(data []byte) []byte {
				corrupted := make([]byte, len(data))
				copy(corrupted, data)
				mid := len(corrupted) / 2
				corrupted[mid] = 0
				corrupted[mid+1] = 0
				return corrupted
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			corrupted := tc.corruptFunc(encResult.Data)

			assert.False(t, ValidateCRC(corrupted))

			_, err := codec.DecodeWithCRC(corrupted)
			assert.True(t, errors.Is(err, wal_domain.ErrCorrupted), "expected ErrCorrupted, got: %v", err)
		})
	}
}

func TestBinaryCodec_DecodeInvalidData(t *testing.T) {
	codec := newTestCodec()

	testCases := []struct {
		expectedErr error
		name        string
		data        []byte
	}{
		{
			name:        "empty data",
			data:        []byte{},
			expectedErr: wal_domain.ErrInvalidEntry,
		},
		{
			name:        "too short",
			data:        make([]byte, minPayloadSize-1),
			expectedErr: wal_domain.ErrInvalidEntry,
		},
		{
			name: "invalid version",
			data: func() []byte {
				data := make([]byte, minPayloadSize)
				data[0] = 255
				return data
			}(),
			expectedErr: wal_domain.ErrInvalidVersion,
		},
		{
			name: "invalid operation",
			data: func() []byte {
				data := make([]byte, minPayloadSize)
				data[0] = formatVersion
				data[1] = 255
				return data
			}(),
			expectedErr: wal_domain.ErrInvalidEntry,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := codec.Decode(tc.data)
			assert.True(t, errors.Is(err, tc.expectedErr), "expected %v, got: %v", tc.expectedErr, err)
		})
	}
}

func TestValidateCRC(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "validate-test",
		Value:     testValue{Name: "test", Count: 789},
		Timestamp: time.Now().UnixNano(),
	}

	encResult, err := codec.EncodeWithCRC(entry)
	require.NoError(t, err)
	defer encResult.Release()

	assert.True(t, ValidateCRC(encResult.Data))

	assert.False(t, ValidateCRC(encResult.Data[:5]))

	corrupted := make([]byte, len(encResult.Data))
	copy(corrupted, encResult.Data)
	corrupted[10] ^= 0xFF
	assert.False(t, ValidateCRC(corrupted))
}

func TestOperation_String(t *testing.T) {
	assert.Equal(t, "SET", wal_domain.OpSet.String())
	assert.Equal(t, "DELETE", wal_domain.OpDelete.String())
	assert.Equal(t, "CLEAR", wal_domain.OpClear.String())
	assert.Equal(t, "UNKNOWN", wal_domain.Operation(99).String())
}

func TestOperation_IsValid(t *testing.T) {
	assert.True(t, wal_domain.OpSet.IsValid())
	assert.True(t, wal_domain.OpDelete.IsValid())
	assert.True(t, wal_domain.OpClear.IsValid())
	assert.False(t, wal_domain.Operation(0).IsValid())
	assert.False(t, wal_domain.Operation(99).IsValid())
}

func BenchmarkCodec_Encode(b *testing.B) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2", "tag3"},
		Timestamp: time.Now().UnixNano(),
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = codec.Encode(entry)
	}
}

func BenchmarkCodec_Decode(b *testing.B) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2", "tag3"},
		Timestamp: time.Now().UnixNano(),
	}

	encoded, _ := codec.Encode(entry)

	b.ResetTimer()
	for b.Loop() {
		result, _ := codec.Decode(encoded)
		result.Release()
	}
}

func BenchmarkCodec_EncodeWithCRC(b *testing.B) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2", "tag3"},
		Timestamp: time.Now().UnixNano(),
	}

	b.ResetTimer()
	for b.Loop() {
		result, _ := codec.EncodeWithCRC(entry)
		result.Release()
	}
}

func BenchmarkValidateCRC(b *testing.B) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2", "tag3"},
		Timestamp: time.Now().UnixNano(),
	}

	encResult, _ := codec.EncodeWithCRC(entry)
	defer encResult.Release()

	b.ResetTimer()
	for b.Loop() {
		_ = ValidateCRC(encResult.Data)
	}
}

type failingKeyCodec struct {
	failEncode bool
	failDecode bool
}

func (f failingKeyCodec) EncodeKey(string) ([]byte, error) {
	if f.failEncode {
		return nil, errors.New("key encode error")
	}
	return []byte("key"), nil
}

func (f failingKeyCodec) DecodeKey([]byte) (string, error) {
	if f.failDecode {
		return "", errors.New("key decode error")
	}
	return "key", nil
}

type failingValueCodec struct {
	failEncode bool
	failDecode bool
}

func (f failingValueCodec) EncodeValue(testValue) ([]byte, error) {
	if f.failEncode {
		return nil, errors.New("value encode error")
	}
	return []byte("value"), nil
}

func (f failingValueCodec) DecodeValue([]byte) (testValue, error) {
	if f.failDecode {
		return testValue{}, errors.New("value decode error")
	}
	return testValue{Name: "decoded"}, nil
}

func TestBinaryCodec_EncodeKeyError(t *testing.T) {
	codec := NewBinaryCodec[string, testValue](
		failingKeyCodec{failEncode: true},
		binaryValueCodec{},
	)

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	_, err := codec.Encode(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encoding key")
}

func TestBinaryCodec_EncodeValueError(t *testing.T) {
	codec := NewBinaryCodec[string, testValue](
		stringKeyCodec{},
		failingValueCodec{failEncode: true},
	)

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	_, err := codec.Encode(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encoding value")
}

func TestBinaryCodec_DecodeKeyError(t *testing.T) {

	workingCodec := newTestCodec()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	encoded, err := workingCodec.Encode(entry)
	require.NoError(t, err)

	failingCodec := NewBinaryCodec(
		failingKeyCodec{failDecode: true},
		binaryValueCodec{},
	)

	_, err = failingCodec.Decode(encoded)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decoding key")
}

func TestBinaryCodec_DecodeValueError(t *testing.T) {

	workingCodec := newTestCodec()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	encoded, err := workingCodec.Encode(entry)
	require.NoError(t, err)

	failingCodec := NewBinaryCodec(
		stringKeyCodec{},
		failingValueCodec{failDecode: true},
	)

	_, err = failingCodec.Decode(encoded)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decoding value")
}

func TestComputeCRC(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "single byte",
			data: []byte{0x42},
		},
		{
			name: "multiple bytes",
			data: []byte("hello world"),
		},
		{
			name: "binary data",
			data: []byte{0x00, 0xFF, 0x01, 0xFE, 0x02, 0xFD},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			crc := ComputeCRC(tc.data)

			crc2 := ComputeCRC(tc.data)
			assert.Equal(t, crc, crc2, "CRC should be deterministic")

			if len(tc.data) > 0 {
				modified := make([]byte, len(tc.data))
				copy(modified, tc.data)
				modified[0] ^= 0x01
				crc3 := ComputeCRC(modified)
				assert.NotEqual(t, crc, crc3, "different data should produce different CRC")
			}
		})
	}
}

type slowKeyCodec struct{}

func (slowKeyCodec) EncodeKey(key string) ([]byte, error) {
	return []byte(key), nil
}

func (slowKeyCodec) DecodeKey(data []byte) (string, error) {
	return string(data), nil
}

type slowValueCodec struct{}

func (slowValueCodec) EncodeValue(value testValue) ([]byte, error) {
	size := 4 + len(value.Name) + 8
	buffer := make([]byte, size)
	binary.BigEndian.PutUint32(buffer[0:], uint32(len(value.Name)))
	copy(buffer[4:], value.Name)
	binary.BigEndian.PutUint64(buffer[4+len(value.Name):], uint64(value.Count))
	return buffer, nil
}

func (slowValueCodec) DecodeValue(data []byte) (testValue, error) {
	if len(data) < 12 {
		return testValue{}, errors.New("data too short")
	}
	nameLen := binary.BigEndian.Uint32(data[0:])
	name := string(data[4 : 4+nameLen])
	count := int(binary.BigEndian.Uint64(data[4+nameLen:]))
	return testValue{Name: name, Count: count}, nil
}

func newSlowTestCodec() *BinaryCodec[string, testValue] {
	return NewBinaryCodec(slowKeyCodec{}, slowValueCodec{})
}

func TestBinaryCodec_SlowPath_EncodeDecodeRoundtrip(t *testing.T) {
	codec := newSlowTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "slow-test-key",
		Value:     testValue{Name: "slow-value", Count: 999},
		Tags:      []string{"tag1", "tag2"},
		Timestamp: time.Now().UnixNano(),
	}

	encResult, err := codec.EncodeWithCRC(entry)
	require.NoError(t, err)
	defer encResult.Release()

	decResult, err := codec.DecodeWithCRC(encResult.Data)
	require.NoError(t, err)
	defer decResult.Release()

	assert.Equal(t, entry.Key, decResult.Entry.Key)
	assert.Equal(t, entry.Value, decResult.Entry.Value)
	assert.Equal(t, entry.Tags, decResult.Entry.Tags)
}

func TestBinaryCodec_SlowPath_Decode(t *testing.T) {
	codec := newSlowTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key",
		Value:     testValue{Name: "val", Count: 1},
		Tags:      []string{"a", "b"},
		Timestamp: time.Now().UnixNano(),
	}

	encoded, err := codec.Encode(entry)
	require.NoError(t, err)

	result, err := codec.Decode(encoded)
	require.NoError(t, err)
	defer result.Release()

	assert.Equal(t, entry.Key, result.Entry.Key)
}

func TestEncodePooled(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "pooled-key",
		Value:     testValue{Name: "pooled-value", Count: 123},
		Tags:      []string{"tag"},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodePooled(entry)
	require.NoError(t, err)

	decResult, err := codec.Decode(result.Data)
	require.NoError(t, err)
	defer decResult.Release()

	assert.Equal(t, entry.Key, decResult.Entry.Key)
	assert.Equal(t, entry.Value, decResult.Entry.Value)

	result.Release()

	result.Release()
}

func TestDecodeResult_Release(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "release-test",
		Value:     testValue{Name: "test", Count: 1},
		Tags:      []string{"tag1", "tag2"},
		Timestamp: time.Now().UnixNano(),
	}

	encResult, err := codec.EncodeWithCRC(entry)
	require.NoError(t, err)
	defer encResult.Release()

	result, err := codec.DecodeWithCRC(encResult.Data)
	require.NoError(t, err)

	assert.NotNil(t, result.Entry.Tags)

	result.Release()

	assert.Nil(t, result.Entry.Tags)

	result.Release()
}

func TestGetByteBuffer_AllSizes(t *testing.T) {
	sizes := []int{
		0, -1,
		1, 32, 64,
		65, 100, 128,
		129, 200, 256,
		257, 400, 512,
		513, 800, 1024,
		1025, 1500, 2048,
		2049, 3000, 4096,
		4097, 6000, 8192,
		8193, 12000, 16384,
		16385, 25000, 32768,
		32769, 50000, 65536,
		65537, 100000, 131072,
		131073, 200000,
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			ptr, buffer := GetByteBuffer(size)

			if size <= 0 {
				assert.Nil(t, ptr)
				assert.Nil(t, buffer)
				return
			}

			assert.NotNil(t, buffer)
			assert.GreaterOrEqual(t, cap(buffer), size)
			assert.Equal(t, 0, len(buffer))

			PutByteBuffer(ptr, buffer)
		})
	}
}

func TestGetTagSlice(t *testing.T) {
	ptr, tags := GetTagSlice()
	require.NotNil(t, ptr)
	require.NotNil(t, tags)
	assert.Equal(t, 0, len(tags))
	assert.GreaterOrEqual(t, cap(tags), 16)

	tags = append(tags, "tag1", "tag2", "tag3")

	PutTagSlice(ptr, tags)

	ptr2, tags2 := GetTagSlice()
	assert.NotNil(t, ptr2)
	assert.Equal(t, 0, len(tags2))

	PutTagSlice(ptr2, tags2)
}

func TestPutTagSlice_Nil(t *testing.T) {

	PutTagSlice(nil, nil)
	PutTagSlice(nil, []string{"a", "b"})
}

func TestResetBytePools(t *testing.T) {

	ptr1, _ := GetByteBuffer(64)
	ptr2, _ := GetByteBuffer(128)

	PutByteBuffer(ptr1, make([]byte, 64))
	PutByteBuffer(ptr2, make([]byte, 128))

	ResetBytePools()

	ptr3, buf3 := GetByteBuffer(64)
	assert.NotNil(t, buf3)
	PutByteBuffer(ptr3, buf3)
}

func TestResetBytePools_AllSizes(t *testing.T) {

	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072}

	for _, size := range sizes {
		ptr, buffer := GetByteBuffer(size)
		require.NotNil(t, buffer)
		PutByteBuffer(ptr, buffer)
	}

	ResetBytePools()

	for _, size := range sizes {
		ptr, buffer := GetByteBuffer(size)
		require.NotNil(t, buffer)
		assert.GreaterOrEqual(t, cap(buffer), size)
		PutByteBuffer(ptr, buffer)
	}
}

type failingFastKeyCodec struct {
	stringKeyCodec
	failFastEncode bool
	failFastDecode bool
}

func (f failingFastKeyCodec) EncodeKeyTo(_ string, _ []byte) (int, error) {
	if f.failFastEncode {
		return 0, errors.New("fast key encode failed")
	}
	return 0, nil
}

func (f failingFastKeyCodec) DecodeKeyFrom(_ []byte) (string, error) {
	if f.failFastDecode {
		return "", errors.New("fast key decode failed")
	}
	return "", nil
}

type failingFastValueCodec struct {
	binaryValueCodec
	failFastEncode bool
	failFastDecode bool
}

func (f failingFastValueCodec) EncodeValueTo(_ testValue, _ []byte) (int, error) {
	if f.failFastEncode {
		return 0, errors.New("fast value encode failed")
	}
	return 0, nil
}

func (f failingFastValueCodec) DecodeValueFrom(_ []byte) (testValue, error) {
	if f.failFastDecode {
		return testValue{}, errors.New("fast value decode failed")
	}
	return testValue{}, nil
}

func TestEncodePooled_FastKeyEncodeError(t *testing.T) {
	keyCodec := failingFastKeyCodec{failFastEncode: true}
	valueCodec := binaryValueCodec{}

	codec := NewBinaryCodec(keyCodec, valueCodec)

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	_, err := codec.EncodePooled(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encoding key")
}

func TestEncodePooled_FastValueEncodeError(t *testing.T) {
	keyCodec := stringKeyCodec{}
	valueCodec := failingFastValueCodec{failFastEncode: true}

	codec := NewBinaryCodec(keyCodec, valueCodec)

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	_, err := codec.EncodePooled(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encoding value")
}

func TestEncodeWithCRC_FastKeyEncodeError(t *testing.T) {
	keyCodec := failingFastKeyCodec{failFastEncode: true}
	valueCodec := binaryValueCodec{}

	codec := NewBinaryCodec(keyCodec, valueCodec)

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	_, err := codec.EncodeWithCRC(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encoding key")
}

func TestEncodeWithCRC_FastValueEncodeError(t *testing.T) {
	keyCodec := stringKeyCodec{}
	valueCodec := failingFastValueCodec{failFastEncode: true}

	codec := NewBinaryCodec(keyCodec, valueCodec)

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	_, err := codec.EncodeWithCRC(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encoding value")
}

func TestDecode_InvalidKeyLength(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodePooled(entry)
	require.NoError(t, err)
	defer result.Release()

	data := make([]byte, len(result.Data))
	copy(data, result.Data)
	binary.BigEndian.PutUint32(data[18:], 0xFFFFFFFF)

	_, err = codec.Decode(data)
	assert.Error(t, err)
	assert.ErrorIs(t, err, wal_domain.ErrInvalidEntry)
}

func TestDecode_InvalidValueLength(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodePooled(entry)
	require.NoError(t, err)
	defer result.Release()

	data := make([]byte, len(result.Data))
	copy(data, result.Data)

	keyLen := binary.BigEndian.Uint32(data[18:])
	valueLenOffset := 18 + 4 + int(keyLen)
	binary.BigEndian.PutUint32(data[valueLenOffset:], 0xFFFFFFFF)

	_, err = codec.Decode(data)
	assert.Error(t, err)
	assert.ErrorIs(t, err, wal_domain.ErrInvalidEntry)
}

func TestDecode_TooManyTags(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test",
		Value:     testValue{Name: "test", Count: 1},
		Tags:      []string{"tag1"},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodePooled(entry)
	require.NoError(t, err)
	defer result.Release()

	data := make([]byte, len(result.Data))
	copy(data, result.Data)

	keyLen := binary.BigEndian.Uint32(data[18:])
	valueLen := binary.BigEndian.Uint32(data[22+int(keyLen):])
	tagCountOffset := 18 + 4 + int(keyLen) + 4 + int(valueLen)
	binary.BigEndian.PutUint16(data[tagCountOffset:], 0xFFFF)

	_, err = codec.Decode(data)
	assert.Error(t, err)
	assert.ErrorIs(t, err, wal_domain.ErrInvalidEntry)
}

func TestDecode_TruncatedTagLength(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test",
		Value:     testValue{Name: "test", Count: 1},
		Tags:      []string{"tag1"},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodePooled(entry)
	require.NoError(t, err)
	defer result.Release()

	keyLen := binary.BigEndian.Uint32(result.Data[18:])
	valueLen := binary.BigEndian.Uint32(result.Data[22+int(keyLen):])
	tagCountOffset := 18 + 4 + int(keyLen) + 4 + int(valueLen)

	truncatedData := result.Data[:tagCountOffset+2]

	_, err = codec.Decode(truncatedData)
	assert.Error(t, err)
	assert.ErrorIs(t, err, wal_domain.ErrInvalidEntry)
}

func TestDecode_InvalidTagLength(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test",
		Value:     testValue{Name: "test", Count: 1},
		Tags:      []string{"tag1"},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodePooled(entry)
	require.NoError(t, err)
	defer result.Release()

	data := make([]byte, len(result.Data))
	copy(data, result.Data)

	keyLen := binary.BigEndian.Uint32(data[18:])
	valueLen := binary.BigEndian.Uint32(data[22+int(keyLen):])
	tagLenOffset := 18 + 4 + int(keyLen) + 4 + int(valueLen) + 2
	binary.BigEndian.PutUint16(data[tagLenOffset:], 0xFFFF)

	_, err = codec.Decode(data)
	assert.Error(t, err)
	assert.ErrorIs(t, err, wal_domain.ErrInvalidEntry)
}

func TestDecodeWithCRC_FastKeyDecodeError(t *testing.T) {

	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodeWithCRC(entry)
	require.NoError(t, err)
	defer result.Release()

	failingCodec := NewBinaryCodec(
		failingFastKeyCodec{failFastDecode: true},
		binaryValueCodec{},
	)

	_, err = failingCodec.DecodeWithCRC(result.Data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decoding key")
}

func TestDecodeWithCRC_FastValueDecodeError(t *testing.T) {

	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test-key",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodeWithCRC(entry)
	require.NoError(t, err)
	defer result.Release()

	failingCodec := NewBinaryCodec(
		stringKeyCodec{},
		failingFastValueCodec{failFastDecode: true},
	)

	_, err = failingCodec.DecodeWithCRC(result.Data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decoding value")
}

func TestDecode_TruncatedAtKeyLength(t *testing.T) {
	codec := newTestCodec()

	data := make([]byte, 18)
	data[0] = 1
	data[1] = byte(wal_domain.OpSet)

	_, err := codec.Decode(data)
	assert.Error(t, err)
	assert.ErrorIs(t, err, wal_domain.ErrInvalidEntry)
}

func TestDecode_TruncatedAtValueLength(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodePooled(entry)
	require.NoError(t, err)
	defer result.Release()

	keyLen := binary.BigEndian.Uint32(result.Data[18:])
	truncatedData := result.Data[:18+4+int(keyLen)]

	_, err = codec.Decode(truncatedData)
	assert.Error(t, err)
	assert.ErrorIs(t, err, wal_domain.ErrInvalidEntry)
}

func TestDecode_TruncatedAtTagCount(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "test",
		Value:     testValue{Name: "test", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodePooled(entry)
	require.NoError(t, err)
	defer result.Release()

	keyLen := binary.BigEndian.Uint32(result.Data[18:])
	valueLen := binary.BigEndian.Uint32(result.Data[22+int(keyLen):])
	truncatedData := result.Data[:18+4+int(keyLen)+4+int(valueLen)]

	_, err = codec.Decode(truncatedData)
	assert.Error(t, err)
	assert.ErrorIs(t, err, wal_domain.ErrInvalidEntry)
}

func TestDecode_DeleteOperation_NoValue(t *testing.T) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpDelete,
		Key:       "test-key",
		Timestamp: time.Now().UnixNano(),
	}

	result, err := codec.EncodePooled(entry)
	require.NoError(t, err)
	defer result.Release()

	decoded, err := codec.Decode(result.Data)
	require.NoError(t, err)
	defer decoded.Release()

	assert.Equal(t, wal_domain.OpDelete, decoded.Entry.Operation)
	assert.Equal(t, "test-key", decoded.Entry.Key)
}
