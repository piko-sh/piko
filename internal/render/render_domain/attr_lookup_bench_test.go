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
	"slices"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func makeTestWriters(count int) []*ast_domain.DirectWriter {
	names := []string{"class", "style", "id", "data-value", "aria-label", "href", "src", "alt", "title", "role"}
	writers := make([]*ast_domain.DirectWriter, count)
	for i := range count {
		writers[i] = &ast_domain.DirectWriter{Name: names[i%len(names)]}
	}
	return writers
}

func makeTestAttrs(count int) []ast_domain.HTMLAttribute {
	names := []string{"class", "id", "data-testid", "aria-hidden", "role", "tabindex", "href", "src", "alt", "title", "lang", "dir", "hidden", "disabled", "readonly"}
	attrs := make([]ast_domain.HTMLAttribute, count)
	for i := range count {
		attrs[i] = ast_domain.HTMLAttribute{Name: names[i%len(names)], Value: "value"}
	}
	return attrs
}

func hasAttributeWriterMap(writerNames map[string]struct{}, attributeName string) bool {
	_, ok := writerNames[attributeName]
	return ok
}

func buildWriterNameMap(writers []*ast_domain.DirectWriter) map[string]struct{} {
	if len(writers) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(writers))
	for _, w := range writers {
		if w != nil {
			m[w.Name] = struct{}{}
		}
	}
	return m
}

func hasAttributeWriterLinear(writers []*ast_domain.DirectWriter, attributeName string) bool {
	for _, w := range writers {
		if w != nil && w.Name == attributeName {
			return true
		}
	}
	return false
}

func buildSortedWriterNames(writers []*ast_domain.DirectWriter) []string {
	if len(writers) == 0 {
		return nil
	}
	names := make([]string, 0, len(writers))
	for _, w := range writers {
		if w != nil {
			names = append(names, w.Name)
		}
	}
	slices.Sort(names)
	return names
}

func hasAttributeWriterBinarySearch(sortedNames []string, attributeName string) bool {
	_, found := slices.BinarySearch(sortedNames, attributeName)
	return found
}

func BenchmarkAttrLookup_Baseline(b *testing.B) {
	scenarios := []struct {
		name           string
		writerCount    int
		attributeCount int
	}{
		{name: "0w_6a", writerCount: 0, attributeCount: 6},
		{name: "1w_6a", writerCount: 1, attributeCount: 6},
		{name: "2w_6a", writerCount: 2, attributeCount: 6},
		{name: "3w_6a", writerCount: 3, attributeCount: 6},
		{name: "3w_10a", writerCount: 3, attributeCount: 10},
		{name: "5w_15a", writerCount: 5, attributeCount: 15},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			writers := makeTestWriters(sc.writerCount)
			attrs := makeTestAttrs(sc.attributeCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				for i := range attrs {
					_ = hasAttributeWriterLinear(writers, attrs[i].Name)
				}
			}
		})
	}
}

func BenchmarkAttrLookup_MapBuild(b *testing.B) {
	scenarios := []struct {
		name           string
		writerCount    int
		attributeCount int
	}{
		{name: "0w_6a", writerCount: 0, attributeCount: 6},
		{name: "1w_6a", writerCount: 1, attributeCount: 6},
		{name: "2w_6a", writerCount: 2, attributeCount: 6},
		{name: "3w_6a", writerCount: 3, attributeCount: 6},
		{name: "3w_10a", writerCount: 3, attributeCount: 10},
		{name: "5w_15a", writerCount: 5, attributeCount: 15},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			writers := makeTestWriters(sc.writerCount)
			attrs := makeTestAttrs(sc.attributeCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				writerNames := buildWriterNameMap(writers)
				for i := range attrs {
					_ = hasAttributeWriterMap(writerNames, attrs[i].Name)
				}
			}
		})
	}
}

func BenchmarkAttrLookup_BinarySearch(b *testing.B) {
	scenarios := []struct {
		name           string
		writerCount    int
		attributeCount int
	}{
		{name: "0w_6a", writerCount: 0, attributeCount: 6},
		{name: "1w_6a", writerCount: 1, attributeCount: 6},
		{name: "2w_6a", writerCount: 2, attributeCount: 6},
		{name: "3w_6a", writerCount: 3, attributeCount: 6},
		{name: "3w_10a", writerCount: 3, attributeCount: 10},
		{name: "5w_15a", writerCount: 5, attributeCount: 15},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			writers := makeTestWriters(sc.writerCount)
			attrs := makeTestAttrs(sc.attributeCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				sortedNames := buildSortedWriterNames(writers)
				for i := range attrs {
					_ = hasAttributeWriterBinarySearch(sortedNames, attrs[i].Name)
				}
			}
		})
	}
}

func BenchmarkAttrLookup_MapLookupOnly(b *testing.B) {
	scenarios := []struct {
		name           string
		writerCount    int
		attributeCount int
	}{
		{name: "0w_6a", writerCount: 0, attributeCount: 6},
		{name: "1w_6a", writerCount: 1, attributeCount: 6},
		{name: "2w_6a", writerCount: 2, attributeCount: 6},
		{name: "3w_6a", writerCount: 3, attributeCount: 6},
		{name: "3w_10a", writerCount: 3, attributeCount: 10},
		{name: "5w_15a", writerCount: 5, attributeCount: 15},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			writers := makeTestWriters(sc.writerCount)
			attrs := makeTestAttrs(sc.attributeCount)
			writerNames := buildWriterNameMap(writers)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				for i := range attrs {
					_ = hasAttributeWriterMap(writerNames, attrs[i].Name)
				}
			}
		})
	}
}

func BenchmarkAttrLookup_FullFunction_Baseline(b *testing.B) {
	scenarios := []struct {
		name          string
		writerCount   int
		nodeAttrCount int
		fragAttrCount int
	}{
		{name: "typical", writerCount: 2, nodeAttrCount: 5, fragAttrCount: 3},
		{name: "no_writers", writerCount: 0, nodeAttrCount: 5, fragAttrCount: 3},
		{name: "many_attrs", writerCount: 3, nodeAttrCount: 10, fragAttrCount: 5},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			writers := makeTestWriters(sc.writerCount)
			nodeAttrs := makeTestAttrs(sc.nodeAttrCount)
			fragAttrs := makeTestAttrs(sc.fragAttrCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {

				for i := range nodeAttrs {
					_ = hasAttributeWriterLinear(writers, nodeAttrs[i].Name)
				}
				for i := range fragAttrs {
					_ = hasAttrByName(nodeAttrs, fragAttrs[i].Name)
					_ = hasAttributeWriterLinear(writers, fragAttrs[i].Name)
				}
			}
		})
	}
}

func BenchmarkAttrLookup_FullFunction_MapOptimised(b *testing.B) {
	scenarios := []struct {
		name          string
		writerCount   int
		nodeAttrCount int
		fragAttrCount int
	}{
		{name: "typical", writerCount: 2, nodeAttrCount: 5, fragAttrCount: 3},
		{name: "no_writers", writerCount: 0, nodeAttrCount: 5, fragAttrCount: 3},
		{name: "many_attrs", writerCount: 3, nodeAttrCount: 10, fragAttrCount: 5},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			writers := makeTestWriters(sc.writerCount)
			nodeAttrs := makeTestAttrs(sc.nodeAttrCount)
			fragAttrs := makeTestAttrs(sc.fragAttrCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {

				writerNames := buildWriterNameMap(writers)
				for i := range nodeAttrs {
					if writerNames != nil {
						_ = hasAttributeWriterMap(writerNames, nodeAttrs[i].Name)
					}
				}
				for i := range fragAttrs {
					_ = hasAttrByName(nodeAttrs, fragAttrs[i].Name)
					if writerNames != nil {
						_ = hasAttributeWriterMap(writerNames, fragAttrs[i].Name)
					}
				}
			}
		})
	}
}

func BenchmarkAttrLookup_FullFunction_LengthCheck(b *testing.B) {
	scenarios := []struct {
		name          string
		writerCount   int
		nodeAttrCount int
		fragAttrCount int
	}{
		{name: "typical", writerCount: 2, nodeAttrCount: 5, fragAttrCount: 3},
		{name: "no_writers", writerCount: 0, nodeAttrCount: 5, fragAttrCount: 3},
		{name: "many_attrs", writerCount: 3, nodeAttrCount: 10, fragAttrCount: 5},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			writers := makeTestWriters(sc.writerCount)
			nodeAttrs := makeTestAttrs(sc.nodeAttrCount)
			fragAttrs := makeTestAttrs(sc.fragAttrCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {

				hasWriters := len(writers) > 0
				for i := range nodeAttrs {
					if hasWriters && hasAttributeWriterLinear(writers, nodeAttrs[i].Name) {
						continue
					}
				}
				for i := range fragAttrs {
					_ = hasAttrByName(nodeAttrs, fragAttrs[i].Name)
					if hasWriters && hasAttributeWriterLinear(writers, fragAttrs[i].Name) {
						continue
					}
				}
			}
		})
	}
}
