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

package ast_domain

import (
	"testing"
)

func populateWriter(dw *DirectWriter, bufCount int, hasName bool) {
	dw.AppendString("test")
	if hasName {
		dw.Name = "test-writer"
	}

	for range bufCount {
		bufferPointer := GetByteBuf()
		*bufferPointer = append(*bufferPointer, "borrowed"...)
		dw.borrowedBufs = append(dw.borrowedBufs, bufferPointer)
	}
}

func BenchmarkDirectWriterReset_Current(b *testing.B) {
	scenarios := []struct {
		name     string
		bufCount int
		hasName  bool
	}{
		{name: "0bufs_noname", bufCount: 0, hasName: false},
		{name: "0bufs_named", bufCount: 0, hasName: true},
		{name: "1buf_noname", bufCount: 1, hasName: false},
		{name: "1buf_named", bufCount: 1, hasName: true},
		{name: "2bufs_noname", bufCount: 2, hasName: false},
		{name: "2bufs_named", bufCount: 2, hasName: true},
		{name: "5bufs_named", bufCount: 5, hasName: true},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			dw := GetDirectWriter()
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				populateWriter(dw, sc.bufCount, sc.hasName)
				dw.Reset()
			}
		})
		PutDirectWriter(GetDirectWriter())
	}
}

func BenchmarkDirectWriterReset_CleanOnly(b *testing.B) {
	dw := GetDirectWriter()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		dw.Reset()
	}
}

func BenchmarkDirectWriterReset_SwitchUnroll(b *testing.B) {
	scenarios := []struct {
		name     string
		bufCount int
	}{
		{name: "0bufs", bufCount: 0},
		{name: "1buf", bufCount: 1},
		{name: "2bufs", bufCount: 2},
		{name: "3bufs", bufCount: 3},
		{name: "5bufs", bufCount: 5},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			dw := GetDirectWriter()
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				populateWriter(dw, sc.bufCount, true)
				resetSwitchUnroll(dw)
			}
		})
	}
}

func resetSwitchUnroll(dw *DirectWriter) {
	if dw.len == 0 && dw.Name == "" && len(dw.borrowedBufs) == 0 {
		return
	}

	dw.len = 0
	dw.overflow = dw.overflow[:0]

	switch len(dw.borrowedBufs) {
	case 0:

	case 1:
		PutByteBuf(dw.borrowedBufs[0])
	case 2:
		PutByteBuf(dw.borrowedBufs[0])
		PutByteBuf(dw.borrowedBufs[1])
	case 3:
		PutByteBuf(dw.borrowedBufs[0])
		PutByteBuf(dw.borrowedBufs[1])
		PutByteBuf(dw.borrowedBufs[2])
	default:
		for _, bufferPointer := range dw.borrowedBufs {
			PutByteBuf(bufferPointer)
		}
	}
	dw.borrowedBufs = dw.borrowedBufs[:0]

	dw.Name = ""
	dw.cachedString = ""
	dw.hasCachedString = false
}

func BenchmarkDirectWriterReset_ConditionalClear(b *testing.B) {
	scenarios := []struct {
		name     string
		bufCount int
		hasName  bool
	}{
		{name: "0bufs_noname", bufCount: 0, hasName: false},
		{name: "0bufs_named", bufCount: 0, hasName: true},
		{name: "1buf_named", bufCount: 1, hasName: true},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			dw := GetDirectWriter()
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				populateWriter(dw, sc.bufCount, sc.hasName)
				resetConditionalClear(dw)
			}
		})
	}
}

func resetConditionalClear(dw *DirectWriter) {
	if dw.len == 0 && dw.Name == "" && len(dw.borrowedBufs) == 0 {
		return
	}

	dw.len = 0
	dw.overflow = dw.overflow[:0]

	for _, bufferPointer := range dw.borrowedBufs {
		PutByteBuf(bufferPointer)
	}
	dw.borrowedBufs = dw.borrowedBufs[:0]

	if dw.Name != "" {
		dw.Name = ""
	}
	if dw.hasCachedString {
		dw.cachedString = ""
		dw.hasCachedString = false
	}
}

func BenchmarkDirectWriterReset_Combined(b *testing.B) {
	scenarios := []struct {
		name     string
		bufCount int
		hasName  bool
	}{
		{name: "0bufs_noname", bufCount: 0, hasName: false},
		{name: "1buf_named", bufCount: 1, hasName: true},
		{name: "2bufs_named", bufCount: 2, hasName: true},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			dw := GetDirectWriter()
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				populateWriter(dw, sc.bufCount, sc.hasName)
				resetCombined(dw)
			}
		})
	}
}

func resetCombined(dw *DirectWriter) {
	if dw.len == 0 && dw.Name == "" && len(dw.borrowedBufs) == 0 {
		return
	}

	dw.len = 0
	dw.overflow = dw.overflow[:0]

	switch len(dw.borrowedBufs) {
	case 0:
	case 1:
		PutByteBuf(dw.borrowedBufs[0])
	case 2:
		PutByteBuf(dw.borrowedBufs[0])
		PutByteBuf(dw.borrowedBufs[1])
	default:
		for _, bufferPointer := range dw.borrowedBufs {
			PutByteBuf(bufferPointer)
		}
	}
	dw.borrowedBufs = dw.borrowedBufs[:0]

	if dw.Name != "" {
		dw.Name = ""
	}
	if dw.hasCachedString {
		dw.cachedString = ""
		dw.hasCachedString = false
	}
}
