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
	"fmt"
	"testing"

	"piko.sh/piko/internal/render/render_dto"
)

func BenchmarkCollectAndSortSVGIDs_5(b *testing.B) {
	benchmarkCollectAndSortSVGIDs(b, 5)
}

func BenchmarkCollectAndSortSVGIDs_20(b *testing.B) {
	benchmarkCollectAndSortSVGIDs(b, 20)
}

func BenchmarkCollectAndSortSVGIDs_50(b *testing.B) {
	benchmarkCollectAndSortSVGIDs(b, 50)
}

func benchmarkCollectAndSortSVGIDs(b *testing.B, count int) {
	entries := make([]svgSymbolEntry, count)
	for i := range count {
		entries[i] = svgSymbolEntry{id: fmt.Sprintf("icon-%d", i)}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		result := collectAndSortSVGIDs(entries)
		_ = result
	}
}

func BenchmarkExtractSVGIDs_Pooled_20(b *testing.B) {
	entries := make([]svgSymbolEntry, 20)
	for i := range 20 {
		entries[i] = svgSymbolEntry{id: fmt.Sprintf("icon-%d", i)}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ptr := getSvgIDSlice()
		extractSVGIDs(entries, ptr)
		_ = *ptr
		putSvgIDSlice(ptr)
	}
}

func BenchmarkAssembleSpriteSheet_5(b *testing.B) {
	benchmarkAssembleSpriteSheet(b, 5)
}

func BenchmarkAssembleSpriteSheet_20(b *testing.B) {
	benchmarkAssembleSpriteSheet(b, 20)
}

func benchmarkAssembleSpriteSheet(b *testing.B, count int) {
	symbols := make([]string, count)
	for i := range count {

		symbols[i] = fmt.Sprintf(`<symbol id="icon-%d" viewBox="0 0 24 24"><path d="M12 2L2 7l10 5 10-5-10-5z"/></symbol>`, i)
	}

	rctx := NewTestRenderContextBuilder().Build()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		result := assembleSpriteSheet(symbols, rctx)
		_ = result
	}
}

func BenchmarkAddLinkHeaderIfUnique_10(b *testing.B) {
	benchmarkAddLinkHeaderIfUnique(b, 10)
}

func BenchmarkAddLinkHeaderIfUnique_50(b *testing.B) {
	benchmarkAddLinkHeaderIfUnique(b, 50)
}

func BenchmarkAddLinkHeaderIfUnique_100(b *testing.B) {
	benchmarkAddLinkHeaderIfUnique(b, 100)
}

func benchmarkAddLinkHeaderIfUnique(b *testing.B, numHeaders int) {

	headers := make([]render_dto.LinkHeader, numHeaders)
	for i := range numHeaders {
		headers[i] = render_dto.LinkHeader{
			URL: fmt.Sprintf("/test%d.js", i),
			Rel: "preload",
			As:  "script",
		}
	}

	b.ResetTimer()

	for b.Loop() {
		rctx := NewTestRenderContextBuilder().Build()

		for _, h := range headers {
			rctx.addLinkHeaderIfUnique(h)
			rctx.addLinkHeaderIfUnique(h)
		}

		if len(rctx.collectedLinkHeaders) != numHeaders {
			b.Fatalf("expected %d headers, got %d", numHeaders, len(rctx.collectedLinkHeaders))
		}
	}
}
