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

package sfcparser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"piko.sh/piko/internal/sfcparser"
)

func loadFixture(b *testing.B, filename string) []byte {
	b.Helper()
	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("Failed to load fixture %s: %v", filename, err)
	}
	return data
}

func BenchmarkParse_Minimal(b *testing.B) {
	data := loadFixture(b, "minimal.pk")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_Typical(b *testing.B) {
	data := loadFixture(b, "typical.pk")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_Large(b *testing.B) {
	data := loadFixture(b, "large.pk")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_ManyBlocks(b *testing.B) {
	data := loadFixture(b, "many_scripts.pk")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_DeepNesting(b *testing.B) {
	data := loadFixture(b, "deep_nesting.pk")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_ComplexAttributes(b *testing.B) {
	data := loadFixture(b, "complex_attrs.pk")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_ScalingBySize(b *testing.B) {
	fixtures := []struct {
		name string
		file string
	}{
		{name: "Minimal", file: "minimal.pk"},
		{name: "Typical", file: "typical.pk"},
		{name: "Large", file: "large.pk"},
	}

	for _, fixture := range fixtures {
		data := loadFixture(b, fixture.file)
		b.Run(fixture.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, err := sfcparser.Parse(data)
				if err != nil {
					b.Fatalf("Parse failed: %v", err)
				}
			}
		})
	}
}

func setupParsedResult(b *testing.B) *sfcparser.ParseResult {
	b.Helper()
	data := loadFixture(b, "many_scripts.pk")
	result, err := sfcparser.Parse(data)
	if err != nil {
		b.Fatalf("Parse failed: %v", err)
	}
	return result
}

func BenchmarkParseResult_JavaScriptScripts(b *testing.B) {
	result := setupParsedResult(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		scripts := result.JavaScriptScripts()
		_ = scripts
	}
}

func BenchmarkParseResult_JavaScriptScript(b *testing.B) {
	result := setupParsedResult(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		script, _ := result.JavaScriptScript()
		_ = script
	}
}

func BenchmarkParseResult_GoScripts(b *testing.B) {
	result := setupParsedResult(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		scripts := result.GoScripts()
		_ = scripts
	}
}

func BenchmarkParseResult_GoScript(b *testing.B) {
	result := setupParsedResult(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		script, _ := result.GoScript()
		_ = script
	}
}

func BenchmarkParseResult_ClientScripts(b *testing.B) {
	result := setupParsedResult(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		scripts := result.ClientScripts()
		_ = scripts
	}
}

func BenchmarkParseResult_ClientScript(b *testing.B) {
	result := setupParsedResult(b)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		script, _ := result.ClientScript()
		_ = script
	}
}

func setupScripts() []sfcparser.Script {
	return []sfcparser.Script{
		{Attributes: map[string]string{}},
		{Attributes: map[string]string{"type": "application/javascript"}},
		{Attributes: map[string]string{"type": "application/x-go"}},
		{Attributes: map[string]string{"lang": "go"}},
		{Attributes: map[string]string{"lang": "ts"}},
		{Attributes: map[string]string{"type": "module"}},
		{Attributes: map[string]string{"type": "text/javascript"}},
		{Attributes: map[string]string{"lang": "typescript"}},
		{Attributes: map[string]string{"type": "application/go", "lang": "go"}},
	}
}

func BenchmarkScript_IsJavaScript(b *testing.B) {
	scripts := setupScripts()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for j := range scripts {
			_ = scripts[j].IsJavaScript()
		}
	}
}

func BenchmarkScript_IsGo(b *testing.B) {
	scripts := setupScripts()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for j := range scripts {
			_ = scripts[j].IsGo()
		}
	}
}

func BenchmarkScript_IsTypeScript(b *testing.B) {
	scripts := setupScripts()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for j := range scripts {
			_ = scripts[j].IsTypeScript()
		}
	}
}

func BenchmarkScript_IsClientScript(b *testing.B) {
	scripts := setupScripts()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for j := range scripts {
			_ = scripts[j].IsClientScript()
		}
	}
}

func BenchmarkScript_Type(b *testing.B) {
	scripts := setupScripts()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for j := range scripts {
			_ = scripts[j].Type()
		}
	}
}

func BenchmarkParseResult_HasCollectionDirective(b *testing.B) {
	data := loadFixture(b, "complex_attrs.pk")
	result, err := sfcparser.Parse(data)
	if err != nil {
		b.Fatalf("Parse failed: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = result.HasCollectionDirective()
	}
}

func BenchmarkParseResult_GetCollectionName(b *testing.B) {
	data := loadFixture(b, "complex_attrs.pk")
	result, err := sfcparser.Parse(data)
	if err != nil {
		b.Fatalf("Parse failed: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = result.GetCollectionName()
	}
}

func BenchmarkParseResult_GetCollectionProvider(b *testing.B) {
	data := loadFixture(b, "complex_attrs.pk")
	result, err := sfcparser.Parse(data)
	if err != nil {
		b.Fatalf("Parse failed: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = result.GetCollectionProvider()
	}
}

func BenchmarkParse_EmptyInput(b *testing.B) {
	data := []byte{}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = sfcparser.Parse(data)
	}
}

func BenchmarkParse_TemplateOnly(b *testing.B) {
	data := []byte(`<template><div>Hello World</div></template>`)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_ScriptOnly(b *testing.B) {
	data := []byte(`<script type="application/x-go">
package main

func main() {
	println("Hello, World!")
}
</script>`)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_StyleOnly(b *testing.B) {
	data := []byte(`<style>
.container { max-width: 1200px; margin: 0 auto; }
.header { background: #333; color: white; }
.footer { border-top: 1px solid #eee; }
</style>`)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_ScriptWithClosingTagInString(b *testing.B) {
	data := []byte(`<script>
const html = '</script>';
const more = '</script>';
console.log(html);
</script>`)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_LargeScript(b *testing.B) {
	var builder strings.Builder
	builder.WriteString("<script type=\"application/x-go\">\npackage main\n\n")
	for i := range 100 {
		builder.WriteString("func function")
		builder.WriteString(string(rune('A' + (i % 26))))
		builder.WriteString("() {\n\tprintln(\"Hello from function\")\n}\n\n")
	}
	builder.WriteString("</script>")

	data := []byte(builder.String())
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_ManyAttributes(b *testing.B) {
	var builder strings.Builder
	builder.WriteString("<template")
	for i := range 50 {
		builder.WriteString(" data-attr-")
		builder.WriteString(string(rune('a' + (i % 26))))
		builder.WriteString("=\"value")
		builder.WriteString(string(rune('0' + (i % 10))))
		builder.WriteString("\"")
	}
	builder.WriteString("><div>Content</div></template>")

	data := []byte(builder.String())
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := sfcparser.Parse(data)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkParse_Parallel(b *testing.B) {
	data := loadFixture(b, "typical.pk")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := sfcparser.Parse(data)
			if err != nil {
				b.Fatalf("Parse failed: %v", err)
			}
		}
	})
}

func BenchmarkParse_Parallel_Large(b *testing.B) {
	data := loadFixture(b, "large.pk")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := sfcparser.Parse(data)
			if err != nil {
				b.Fatalf("Parse failed: %v", err)
			}
		}
	})
}
