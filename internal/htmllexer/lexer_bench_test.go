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

package htmllexer

import (
	"strings"
	"testing"
)

func drainLexer(l *Lexer) {
	for l.Next() != ErrorToken {
	}
}

func BenchmarkLex_SimpleHTML(b *testing.B) {
	fragment := `<div class="c"><p>hello</p></div>`
	data := []byte(strings.Repeat(fragment, 50))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		drainLexer(NewLexer(data))
	}
}

func BenchmarkLex_SVGContent(b *testing.B) {
	fragment := `<div><svg viewBox="0 0 100 100"><rect x="10" y="10" width="80" height="80"/></svg></div>`
	data := []byte(strings.Repeat(fragment, 20))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		drainLexer(NewLexer(data))
	}
}

func BenchmarkLex_UTF8Content(b *testing.B) {
	fragment := "<p>\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e\xe3\x83\x86\xe3\x82\xad\xe3\x82\xb9\xe3\x83\x88</p>"
	data := []byte(strings.Repeat(fragment, 50))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		drainLexer(NewLexer(data))
	}
}

func BenchmarkLex_DeepNesting(b *testing.B) {
	var builder strings.Builder
	depth := 50
	for range depth {
		builder.WriteString("<div>")
	}
	builder.WriteString("leaf")
	for range depth {
		builder.WriteString("</div>")
	}

	data := []byte(builder.String())
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		drainLexer(NewLexer(data))
	}
}

func BenchmarkLex_ManyAttributes(b *testing.B) {
	var builder strings.Builder
	builder.WriteString("<div")
	for i := range 26 {
		builder.WriteByte(' ')
		builder.WriteByte('a' + byte(i))
		builder.WriteString(`="`)
		builder.WriteByte('0' + byte(i%10))
		builder.WriteString(`"`)
	}
	builder.WriteString(">content</div>")

	data := []byte(builder.String())
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		drainLexer(NewLexer(data))
	}
}

func BenchmarkLex_ScriptBody(b *testing.B) {
	fragment := `<script>const greet = (name) => "hello " + name; const list = [1, 2, 3];</script>`
	data := []byte(strings.Repeat(fragment, 50))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		drainLexer(NewLexer(data))
	}
}

func BenchmarkLex_ScriptBodyWithCommentLikeBytes(b *testing.B) {
	fragment := `<script>const a = '<!--'; const b = "<!-- not a comment -->"; const c = '<script>';</script>`
	data := []byte(strings.Repeat(fragment, 50))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		drainLexer(NewLexer(data))
	}
}

func BenchmarkLex_StyleBody(b *testing.B) {
	fragment := `<style>.x { color: red; } .y::before { content: '<!--'; }</style>`
	data := []byte(strings.Repeat(fragment, 50))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		drainLexer(NewLexer(data))
	}
}

func BenchmarkLex_TextareaBody(b *testing.B) {
	fragment := `<textarea>line one with <!-- visible comment --> and bare < and > characters</textarea>`
	data := []byte(strings.Repeat(fragment, 50))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		drainLexer(NewLexer(data))
	}
}

func BenchmarkLex_SimpleHTML_ValueType(b *testing.B) {
	fragment := `<div class="c"><p>hello</p></div>`
	data := []byte(strings.Repeat(fragment, 50))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var lexer Lexer
		lexer.Init(data)
		for lexer.Next() != ErrorToken {
		}
	}
}

func BenchmarkLex_ScriptBody_ValueType(b *testing.B) {
	fragment := `<script>const greet = (name) => "hello " + name; const list = [1, 2, 3];</script>`
	data := []byte(strings.Repeat(fragment, 50))
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var lexer Lexer
		lexer.Init(data)
		for lexer.Next() != ErrorToken {
		}
	}
}
