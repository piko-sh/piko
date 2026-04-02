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

package encoding_test

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"testing"

	"piko.sh/piko/wdk/encoding"
)

var (
	resultString     string
	resultBytes      []byte
	resultUint       uint64
	resultErr        error
	MyBase64Encoding *encoding.Encoding
	benchmarkData    = make(map[string][]byte)
)

func init() {
	var err error
	MyBase64Encoding, err = encoding.NewEncoding(encoding.StdBase64Alphabet)
	if err != nil {
		panic("Failed to initialise custom Base64 encoder for benchmarks")
	}
}

func BenchmarkEncodeUint64(b *testing.B) {
	value := uint64(999999999999)

	b.Run("Base36", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			resultString = encoding.EncodeUint64Base36(value)
		}
	})

	b.Run("Base58", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			resultString = encoding.EncodeUint64Base58(value)
		}
	})

	b.Run("Base64_StdLib", func(b *testing.B) {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, value)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			resultString = base64.StdEncoding.EncodeToString(data)
		}
	})

	b.Run("Base64_OurPackage", func(b *testing.B) {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, value)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			resultString = MyBase64Encoding.EncodeBytes(data)
		}
	})
}

func BenchmarkDecodeUint64(b *testing.B) {
	value := uint64(999999999999)

	b.Run("Base36", func(b *testing.B) {
		encoded := encoding.EncodeUint64Base36(value)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			resultUint, resultErr = encoding.DecodeUint64Base36(encoded)
		}
	})

	b.Run("Base58", func(b *testing.B) {
		encoded := encoding.EncodeUint64Base58(value)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			resultUint, resultErr = encoding.DecodeUint64Base58(encoded)
		}
	})

	b.Run("Base64_StdLib", func(b *testing.B) {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, value)
		encoded := base64.StdEncoding.EncodeToString(data)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			resultBytes, resultErr = base64.StdEncoding.DecodeString(encoded)
		}
	})

	b.Run("Base64_OurPackage", func(b *testing.B) {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, value)
		encoded := MyBase64Encoding.EncodeBytes(data)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			resultBytes, resultErr = MyBase64Encoding.DecodeBytes(encoded)
		}
	})
}

func init() {
	sizes := map[string]int{
		"small":  16,
		"medium": 128,
		"large":  4096,
	}
	for name, size := range sizes {
		data := make([]byte, size)
		_, err := rand.Read(data)
		if err != nil {
			panic(fmt.Sprintf("Failed to generate random data for benchmarks: %v", err))
		}
		benchmarkData[name] = data
	}
}

func BenchmarkEncodeBytes(b *testing.B) {
	for sizeName, data := range benchmarkData {
		b.Run(fmt.Sprintf("%s-%d_bytes", sizeName, len(data)), func(b *testing.B) {
			b.Run("Base36", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					resultString = encoding.EncodeBytesBase36(data)
				}
			})
			b.Run("Base58", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					resultString = encoding.EncodeBytesBase58(data)
				}
			})
			b.Run("Base64_StdLib", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					resultString = base64.StdEncoding.EncodeToString(data)
				}
			})
			b.Run("Base64_OurPackage", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					resultString = MyBase64Encoding.EncodeBytes(data)
				}
			})
		})
	}
}

func BenchmarkDecodeBytes(b *testing.B) {
	for sizeName, data := range benchmarkData {
		b.Run(fmt.Sprintf("%s-%d_bytes", sizeName, len(data)), func(b *testing.B) {
			b.Run("Base36", func(b *testing.B) {
				encoded := encoding.EncodeBytesBase36(data)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					resultBytes, resultErr = encoding.DecodeBytesBase36(encoded)
				}
			})
			b.Run("Base58", func(b *testing.B) {
				encoded := encoding.EncodeBytesBase58(data)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					resultBytes, resultErr = encoding.DecodeBytesBase58(encoded)
				}
			})
			b.Run("Base64_StdLib", func(b *testing.B) {
				encoded := base64.StdEncoding.EncodeToString(data)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					resultBytes, resultErr = base64.StdEncoding.DecodeString(encoded)
				}
			})
			b.Run("Base64_OurPackage", func(b *testing.B) {
				encoded := MyBase64Encoding.EncodeBytes(data)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					resultBytes, resultErr = MyBase64Encoding.DecodeBytes(encoded)
				}
			})
		})
	}
}
