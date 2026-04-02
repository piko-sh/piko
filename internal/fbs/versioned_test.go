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

package fbs

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"
)

func TestComputeSchemaHash(t *testing.T) {
	content := []byte("table FooFB { id: int32; }")
	hash := ComputeSchemaHash(content)

	expected := sha256.Sum256(content)
	if hash != expected {
		t.Errorf("hash mismatch: got %x, want %x", hash, expected)
	}

	content2 := []byte("table BarFB { id: int32; }")
	hash2 := ComputeSchemaHash(content2)
	if hash == hash2 {
		t.Error("different content produced same hash")
	}
}

func TestPackedSize(t *testing.T) {
	if got := PackedSize(0); got != hashSize {
		t.Errorf("PackedSize(0) = %d, want %d", got, hashSize)
	}
	if got := PackedSize(100); got != hashSize+100 {
		t.Errorf("PackedSize(100) = %d, want %d", got, hashSize+100)
	}
}

func TestPackAndUnpack(t *testing.T) {
	schema := []byte("table TestFB { value: string; }")
	hash := ComputeSchemaHash(schema)
	payload := []byte{0x04, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00}

	dst := make([]byte, PackedSize(len(payload)))
	n := Pack(dst, hash, payload)

	if n != len(dst) {
		t.Errorf("Pack returned %d, want %d", n, len(dst))
	}

	var storedHash SchemaHash
	copy(storedHash[:], dst[:hashSize])
	if storedHash != hash {
		t.Error("stored hash doesn't match")
	}

	if !bytes.Equal(dst[hashSize:], payload) {
		t.Error("stored payload doesn't match")
	}

	extracted, err := Unpack(hash, dst)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}
	if !bytes.Equal(extracted, payload) {
		t.Error("extracted payload doesn't match original")
	}

	if &extracted[0] != &dst[hashSize] {
		t.Error("Unpack should return zero-copy slice")
	}
}

func TestPackAlloc(t *testing.T) {
	schema := []byte("table TestFB { value: string; }")
	hash := ComputeSchemaHash(schema)
	payload := []byte{0x01, 0x02, 0x03, 0x04}

	packed := PackAlloc(hash, payload)

	if len(packed) != PackedSize(len(payload)) {
		t.Errorf("PackAlloc length = %d, want %d", len(packed), PackedSize(len(payload)))
	}

	extracted, err := Unpack(hash, packed)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}
	if !bytes.Equal(extracted, payload) {
		t.Error("roundtrip failed")
	}
}

func TestUnpackErrors(t *testing.T) {
	schema := []byte("table TestFB { value: string; }")
	hash := ComputeSchemaHash(schema)

	t.Run("data too short", func(t *testing.T) {
		shortData := make([]byte, hashSize-1)
		_, err := Unpack(hash, shortData)
		if !errors.Is(err, errDataTooShort) {
			t.Errorf("got %v, want errDataTooShort", err)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		_, err := Unpack(hash, nil)
		if !errors.Is(err, errDataTooShort) {
			t.Errorf("got %v, want errDataTooShort", err)
		}
	})

	t.Run("hash mismatch", func(t *testing.T) {
		differentSchema := []byte("table DifferentFB { id: int64; }")
		differentHash := ComputeSchemaHash(differentSchema)

		packed := PackAlloc(hash, []byte{0x01, 0x02})
		_, err := Unpack(differentHash, packed)
		if !errors.Is(err, ErrSchemaVersionMismatch) {
			t.Errorf("got %v, want ErrSchemaVersionMismatch", err)
		}
	})

	t.Run("corrupted hash", func(t *testing.T) {
		packed := PackAlloc(hash, []byte{0x01, 0x02})
		packed[0] ^= 0xFF

		_, err := Unpack(hash, packed)
		if !errors.Is(err, ErrSchemaVersionMismatch) {
			t.Errorf("got %v, want ErrSchemaVersionMismatch", err)
		}
	})
}

func TestValidateHash(t *testing.T) {
	schema := []byte("table TestFB { value: string; }")
	hash := ComputeSchemaHash(schema)
	payload := []byte{0x01, 0x02, 0x03}

	packed := PackAlloc(hash, payload)

	if !ValidateHash(hash, packed) {
		t.Error("ValidateHash returned false for valid data")
	}

	differentHash := ComputeSchemaHash([]byte("different"))
	if ValidateHash(differentHash, packed) {
		t.Error("ValidateHash returned true for wrong hash")
	}

	if ValidateHash(hash, make([]byte, hashSize-1)) {
		t.Error("ValidateHash returned true for short data")
	}
}

func Test_extractHash(t *testing.T) {
	schema := []byte("table TestFB { value: string; }")
	hash := ComputeSchemaHash(schema)
	payload := []byte{0x01, 0x02, 0x03}

	packed := PackAlloc(hash, payload)
	extracted := extractHash(packed)

	if extracted != hash {
		t.Errorf("extractHash mismatch: got %x, want %x", extracted, hash)
	}

	shortData := make([]byte, hashSize-1)
	zeroHash := extractHash(shortData)
	if zeroHash != (SchemaHash{}) {
		t.Error("extractHash should return zero hash for short data")
	}
}

func BenchmarkPack(b *testing.B) {
	schema := []byte("table TestFB { value: string; name: string; id: int32; }")
	hash := ComputeSchemaHash(schema)
	payload := make([]byte, 1024)

	dst := make([]byte, PackedSize(len(payload)))

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		Pack(dst, hash, payload)
	}
}

func BenchmarkPackAlloc(b *testing.B) {
	schema := []byte("table TestFB { value: string; name: string; id: int32; }")
	hash := ComputeSchemaHash(schema)
	payload := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = PackAlloc(hash, payload)
	}
}

func BenchmarkUnpack(b *testing.B) {
	schema := []byte("table TestFB { value: string; name: string; id: int32; }")
	hash := ComputeSchemaHash(schema)
	payload := make([]byte, 1024)
	packed := PackAlloc(hash, payload)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = Unpack(hash, packed)
	}
}

func BenchmarkValidateHash(b *testing.B) {
	schema := []byte("table TestFB { value: string; name: string; id: int32; }")
	hash := ComputeSchemaHash(schema)
	payload := make([]byte, 1024)
	packed := PackAlloc(hash, payload)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = ValidateHash(hash, packed)
	}
}

func BenchmarkUnpackMismatch(b *testing.B) {
	schema1 := []byte("table TestFB { value: string; }")
	schema2 := []byte("table OtherFB { value: string; }")
	hash1 := ComputeSchemaHash(schema1)
	hash2 := ComputeSchemaHash(schema2)
	payload := make([]byte, 1024)
	packed := PackAlloc(hash1, payload)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = Unpack(hash2, packed)
	}
}
