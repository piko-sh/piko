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

//go:build bench

package render_domain

import (
	"bytes"
	"sync"
	"testing"

	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
)

func writeFNVOriginal(s string, qw *qt.Writer) {
	qw.N().SZ(ast_domain.AppendFNVString(nil, s))
}

var (
	hexStrings = [16]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}
	hexPairs   [256]string
	fnvBufPool = sync.Pool{
		New: func() any {
			return new(make([]byte, 8))
		},
	}
	hexTableLocal = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
)

func writeFNVHexStrings(sum uint32, qw *qt.Writer) {
	w := qw.N()
	w.S(hexStrings[(sum>>28)&0xf])
	w.S(hexStrings[(sum>>24)&0xf])
	w.S(hexStrings[(sum>>20)&0xf])
	w.S(hexStrings[(sum>>16)&0xf])
	w.S(hexStrings[(sum>>12)&0xf])
	w.S(hexStrings[(sum>>8)&0xf])
	w.S(hexStrings[(sum>>4)&0xf])
	w.S(hexStrings[sum&0xf])
}

func init() {
	hexTable := [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
	for i := range 256 {
		hexPairs[i] = string([]byte{hexTable[i>>4], hexTable[i&0xf]})
	}
}

func writeFNVHexPairs(sum uint32, qw *qt.Writer) {
	w := qw.N()
	w.S(hexPairs[byte(sum>>24)])
	w.S(hexPairs[byte(sum>>16)])
	w.S(hexPairs[byte(sum>>8)])
	w.S(hexPairs[byte(sum)])
}

func writeFNVPooled(sum uint32, qw *qt.Writer) {
	bufferPointer, ok := fnvBufPool.Get().(*[]byte)
	if !ok {
		panic("fnvBufPool returned wrong type")
	}
	buffer := *bufferPointer

	buffer[0] = hexTableLocal[(sum>>28)&0xf]
	buffer[1] = hexTableLocal[(sum>>24)&0xf]
	buffer[2] = hexTableLocal[(sum>>20)&0xf]
	buffer[3] = hexTableLocal[(sum>>16)&0xf]
	buffer[4] = hexTableLocal[(sum>>12)&0xf]
	buffer[5] = hexTableLocal[(sum>>8)&0xf]
	buffer[6] = hexTableLocal[(sum>>4)&0xf]
	buffer[7] = hexTableLocal[sum&0xf]

	qw.N().SZ(buffer)

	fnvBufPool.Put(bufferPointer)
}

func getFNVSum(s string) uint32 {

	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}

func BenchmarkFNV_Original(b *testing.B) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	testString := "test-string-for-hashing"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buffer.Reset()
		writeFNVOriginal(testString, qw)
	}
}

func BenchmarkFNV_HexStrings(b *testing.B) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	testString := "test-string-for-hashing"
	sum := getFNVSum(testString)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buffer.Reset()
		writeFNVHexStrings(sum, qw)
	}
}

func BenchmarkFNV_HexPairs(b *testing.B) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	testString := "test-string-for-hashing"
	sum := getFNVSum(testString)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buffer.Reset()
		writeFNVHexPairs(sum, qw)
	}
}

func BenchmarkFNV_Pooled(b *testing.B) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	testString := "test-string-for-hashing"
	sum := getFNVSum(testString)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buffer.Reset()
		writeFNVPooled(sum, qw)
	}
}

func BenchmarkFNVFull_Original(b *testing.B) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	testString := "test-string-for-hashing"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buffer.Reset()
		writeFNVOriginal(testString, qw)
	}
}

func BenchmarkFNVFull_HexStrings(b *testing.B) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	testString := "test-string-for-hashing"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buffer.Reset()
		sum := getFNVSum(testString)
		writeFNVHexStrings(sum, qw)
	}
}

func BenchmarkFNVFull_HexPairs(b *testing.B) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	testString := "test-string-for-hashing"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buffer.Reset()
		sum := getFNVSum(testString)
		writeFNVHexPairs(sum, qw)
	}
}

func BenchmarkFNVFull_Pooled(b *testing.B) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	testString := "test-string-for-hashing"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buffer.Reset()
		sum := getFNVSum(testString)
		writeFNVPooled(sum, qw)
	}
}

func BenchmarkFNVParallel_Original(b *testing.B) {
	testString := "test-string-for-hashing"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		for pb.Next() {
			buffer.Reset()
			writeFNVOriginal(testString, qw)
		}
	})
}

func BenchmarkFNVParallel_HexStrings(b *testing.B) {
	testString := "test-string-for-hashing"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		for pb.Next() {
			buffer.Reset()
			sum := getFNVSum(testString)
			writeFNVHexStrings(sum, qw)
		}
	})
}

func BenchmarkFNVParallel_HexPairs(b *testing.B) {
	testString := "test-string-for-hashing"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		for pb.Next() {
			buffer.Reset()
			sum := getFNVSum(testString)
			writeFNVHexPairs(sum, qw)
		}
	})
}

func BenchmarkFNVParallel_Pooled(b *testing.B) {
	testString := "test-string-for-hashing"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		var buffer bytes.Buffer
		qw := qt.AcquireWriter(&buffer)
		defer qt.ReleaseWriter(qw)

		for pb.Next() {
			buffer.Reset()
			sum := getFNVSum(testString)
			writeFNVPooled(sum, qw)
		}
	})
}

func TestFNVApproachesProduceSameOutput(t *testing.T) {
	testString := "test-string-for-hashing"
	sum := getFNVSum(testString)

	var buf1, buf2, buf3, buf4 bytes.Buffer
	qw1 := qt.AcquireWriter(&buf1)
	qw2 := qt.AcquireWriter(&buf2)
	qw3 := qt.AcquireWriter(&buf3)
	qw4 := qt.AcquireWriter(&buf4)

	writeFNVOriginal(testString, qw1)
	writeFNVHexStrings(sum, qw2)
	writeFNVHexPairs(sum, qw3)
	writeFNVPooled(sum, qw4)

	qt.ReleaseWriter(qw1)
	qt.ReleaseWriter(qw2)
	qt.ReleaseWriter(qw3)
	qt.ReleaseWriter(qw4)

	original := buf1.String()
	hexStringsOut := buf2.String()
	hexPairsOut := buf3.String()
	pooledOut := buf4.String()

	if original != hexStringsOut {
		t.Errorf("HexStrings mismatch: got %q, want %q", hexStringsOut, original)
	}
	if original != hexPairsOut {
		t.Errorf("HexPairs mismatch: got %q, want %q", hexPairsOut, original)
	}
	if original != pooledOut {
		t.Errorf("Pooled mismatch: got %q, want %q", pooledOut, original)
	}

	t.Logf("All approaches produce: %q", original)
}

func TestWriteWriterPartFNV(t *testing.T) {
	testString := "test-string-for-hashing"
	expectedHash := "a1d4e13a"

	var buf1 bytes.Buffer
	qw1 := qt.AcquireWriter(&buf1)
	part1 := &ast_domain.WriterPart{
		Type:        ast_domain.WriterPartFNVString,
		StringValue: testString,
	}
	writeWriterPart(part1, qw1)
	qt.ReleaseWriter(qw1)

	if got := buf1.String(); got != expectedHash {
		t.Errorf("WriterPartFNVString: got %q, want %q", got, expectedHash)
	}

	var buf2 bytes.Buffer
	qw2 := qt.AcquireWriter(&buf2)
	part2 := &ast_domain.WriterPart{
		Type:       ast_domain.WriterPartFNVFloat,
		FloatValue: 3.14159,
	}
	writeWriterPart(part2, qw2)
	qt.ReleaseWriter(qw2)

	if len(buf2.String()) != 8 {
		t.Errorf("WriterPartFNVFloat: expected 8 chars, got %d", len(buf2.String()))
	}

	var buf3 bytes.Buffer
	qw3 := qt.AcquireWriter(&buf3)
	part3 := &ast_domain.WriterPart{
		Type:     ast_domain.WriterPartFNVAny,
		AnyValue: testString,
	}
	writeWriterPart(part3, qw3)
	qt.ReleaseWriter(qw3)

	if got := buf3.String(); got != expectedHash {
		t.Errorf("WriterPartFNVAny: got %q, want %q", got, expectedHash)
	}

	var buf4 bytes.Buffer
	qw4 := qt.AcquireWriter(&buf4)
	part4 := &ast_domain.WriterPart{
		Type:     ast_domain.WriterPartFNVAny,
		AnyValue: nil,
	}
	writeWriterPart(part4, qw4)
	qt.ReleaseWriter(qw4)

	if got := buf4.String(); got != "" {
		t.Errorf("WriterPartFNVAny with nil: got %q, want empty", got)
	}
}

func BenchmarkWriteWriterPartFNVString(b *testing.B) {
	var buffer bytes.Buffer
	qw := qt.AcquireWriter(&buffer)
	defer qt.ReleaseWriter(qw)

	part := &ast_domain.WriterPart{
		Type:        ast_domain.WriterPartFNVString,
		StringValue: "test-string-for-hashing",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buffer.Reset()
		writeWriterPart(part, qw)
	}
}
