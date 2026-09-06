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
	"context"
	"strings"
	"testing"
)

func BenchmarkParse_SingleLineTemplate(b *testing.B) {
	element := `<span class="b" :title="item.name">{{ a }} text</span><!-- c --><p>x</p>`
	benchmarkParse(b, strings.Repeat(element, (1<<20)/len(element)))
}

func BenchmarkParse_MultiLineTemplate(b *testing.B) {
	element := "<span class=\"b\" :title=\"item.name\">{{ a }} text</span><!-- c --><p>x</p>\n"
	benchmarkParse(b, strings.Repeat(element, (1<<20)/len(element)))
}

func benchmarkParse(b *testing.B, source string) {
	b.Helper()
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if _, err := Parse(ctx, source, "bench.pk", nil); err != nil {
			b.Fatal(err)
		}
	}
}
