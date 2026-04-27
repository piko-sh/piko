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

package htmllexer

import (
	"errors"
	"io"
	"testing"
	"unsafe"
)

type expectedToken struct {
	tokenType       TokenType
	text            string
	attrVal         string
	attrValIsNil    bool
	tokenStart      int
	tokenEnd        int
	tokenLine       int
	tokenCol        int
	attrValStart    int
	checkStart      bool
	checkEnd        bool
	checkLine       bool
	checkCol        bool
	checkAttrValPos bool
}

func TestBasicElements(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "simple div element",
			input: "<div></div>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "div"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "element with text content",
			input: "<p>hello</p>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "p"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "hello"},
				{tokenType: EndTagToken, text: "p"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "nested elements",
			input: "<div><span>text</span></div>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: StartTagCloseToken},
				{tokenType: StartTagToken, text: "span"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "text"},
				{tokenType: EndTagToken, text: "span"},
				{tokenType: EndTagToken, text: "div"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "self-closing tag with slash",
			input: "<br/>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "br"},
				{tokenType: StartTagVoidToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "self-closing with space before slash",
			input: "<br />",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "br"},
				{tokenType: StartTagVoidToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "void element without slash",
			input: "<br>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "br"},
				{tokenType: StartTagCloseToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "text only",
			input: "hello world",
			expected: []expectedToken{
				{tokenType: TextToken, text: "hello world"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "empty input",
			input: "",
			expected: []expectedToken{
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "text with interpolation delimiters",
			input: "<p>{{ name }}</p>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "p"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "{{ name }}"},
				{tokenType: EndTagToken, text: "p"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "uppercase tag names preserved",
			input: "<DIV></DIV>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "DIV"},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "DIV"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "custom element with hyphen",
			input: "<my-component></my-component>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "my-component"},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "my-component"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "piko namespace tag",
			input: "<piko:timeline></piko:timeline>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "piko:timeline"},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "piko:timeline"},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestPlaintextElement(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "plaintext consumes everything after opening tag",
			input: "<plaintext>all <b>remaining</b> content",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "plaintext"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "all <b>remaining</b> content"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "plaintext with newlines uses multi-byte advanceCursor",
			input: "<plaintext>line1\nline2\nline3",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "plaintext"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "line1\nline2\nline3"},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestScriptCommentClosesWithoutNesting(t *testing.T) {

	input := "<script><!-- </script>"
	expected := []expectedToken{
		{tokenType: StartTagToken, text: "script"},
		{tokenType: StartTagCloseToken},
		{tokenType: TextToken, text: "<!-- "},
		{tokenType: EndTagToken, text: "script"},
		{tokenType: ErrorToken},
	}

	assertTokenSequence(t, input, expected)
}

func TestEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "bare less-than followed by space",
			input: "a < b",
			expected: []expectedToken{
				{tokenType: TextToken, text: "a < b"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "bare less-than followed by digit",
			input: "a <3 b",
			expected: []expectedToken{
				{tokenType: TextToken, text: "a <3 b"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "end tag with whitespace",
			input: "<div></div  >",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "div"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "multiple text segments between tags",
			input: "before<div>inside</div>after",
			expected: []expectedToken{
				{tokenType: TextToken, text: "before"},
				{tokenType: StartTagToken, text: "div"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "inside"},
				{tokenType: EndTagToken, text: "div"},
				{tokenType: TextToken, text: "after"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "unclosed tag at eof",
			input: "<div",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestPositionTracking(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		tokenIndex int
		expectedTT TokenType
		checkLine  int
		checkCol   int
		checkStart int
		checkEnd   int
	}{
		{
			name:       "first token position",
			input:      "<div>",
			tokenIndex: 0,
			expectedTT: StartTagToken,
			checkLine:  1,
			checkCol:   1,
			checkStart: 0,
			checkEnd:   4,
		},
		{
			name:       "text including newline on second line",
			input:      "<div>\nhello</div>",
			tokenIndex: 2,
			expectedTT: TextToken,
			checkLine:  1,
			checkCol:   6,
			checkStart: 5,
			checkEnd:   11,
		},
		{
			name:       "end tag on third line",
			input:      "<div>\nhello\n</div>",
			tokenIndex: 3,
			expectedTT: EndTagToken,
			checkLine:  3,
			checkCol:   1,
			checkStart: 12,
			checkEnd:   18,
		},
		{
			name:       "attribute position",
			input:      `<div class="foo">`,
			tokenIndex: 1,
			expectedTT: AttributeToken,
			checkLine:  1,
			checkCol:   6,
			checkStart: 5,
			checkEnd:   16,
		},
		{
			name:       "end tag column after multi-byte utf8 content",
			input:      "<p>caf\u00e9</p>",
			tokenIndex: 3,
			expectedTT: EndTagToken,
			checkLine:  1,
			checkCol:   8,
			checkStart: 8,
			checkEnd:   12,
		},
		{
			name:       "tag after CJK characters counts rune columns",
			input:      "<p>\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e</p>",
			tokenIndex: 3,
			expectedTT: EndTagToken,
			checkLine:  1,
			checkCol:   7,
			checkStart: 12,
			checkEnd:   16,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lexer := NewLexer([]byte(tc.input))

			for i := 0; i <= tc.tokenIndex; i++ {
				tt := lexer.Next()
				if i == tc.tokenIndex {
					if tt != tc.expectedTT {
						t.Fatalf("token %d: got type %v, want %v", i, tt, tc.expectedTT)
					}

					if lexer.TokenLine() != tc.checkLine {
						t.Errorf("TokenLine: got %d, want %d", lexer.TokenLine(), tc.checkLine)
					}

					if lexer.TokenCol() != tc.checkCol {
						t.Errorf("TokenCol: got %d, want %d", lexer.TokenCol(), tc.checkCol)
					}

					if lexer.TokenStart() != tc.checkStart {
						t.Errorf("TokenStart: got %d, want %d", lexer.TokenStart(), tc.checkStart)
					}

					if lexer.TokenEnd() != tc.checkEnd {
						t.Errorf("TokenEnd: got %d, want %d", lexer.TokenEnd(), tc.checkEnd)
					}
				}
			}
		})
	}
}

func TestPositionAt(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		offset       int
		expectedLine int
		expectedCol  int
	}{
		{
			name:         "start of input",
			input:        "hello\nworld",
			offset:       0,
			expectedLine: 1,
			expectedCol:  1,
		},
		{
			name:         "middle of first line",
			input:        "hello\nworld",
			offset:       3,
			expectedLine: 1,
			expectedCol:  4,
		},
		{
			name:         "start of second line",
			input:        "hello\nworld",
			offset:       6,
			expectedLine: 2,
			expectedCol:  1,
		},
		{
			name:         "middle of second line",
			input:        "hello\nworld",
			offset:       8,
			expectedLine: 2,
			expectedCol:  3,
		},
		{
			name:         "end of input",
			input:        "hello\nworld",
			offset:       11,
			expectedLine: 2,
			expectedCol:  6,
		},
		{
			name:         "multi-byte utf8 column counting",
			input:        "h\u00e9llo\nw\u00f6rld",
			offset:       8,
			expectedLine: 2,
			expectedCol:  2,
		},
		{
			name:         "multi-byte utf8 first char on second line",
			input:        "h\u00e9llo\nw\u00f6rld",
			offset:       7,
			expectedLine: 2,
			expectedCol:  1,
		},
		{
			name:         "negative offset clamped to zero",
			input:        "hello",
			offset:       -5,
			expectedLine: 1,
			expectedCol:  1,
		},
		{
			name:         "offset beyond end clamped",
			input:        "hello",
			offset:       100,
			expectedLine: 1,
			expectedCol:  6,
		},
		{
			name:         "empty input",
			input:        "",
			offset:       0,
			expectedLine: 1,
			expectedCol:  1,
		},
		{
			name:         "three lines",
			input:        "a\nb\nc",
			offset:       4,
			expectedLine: 3,
			expectedCol:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lexer := NewLexer([]byte(tc.input))
			line, col := lexer.PositionAt(tc.offset)

			if line != tc.expectedLine {
				t.Errorf("line: got %d, want %d", line, tc.expectedLine)
			}

			if col != tc.expectedCol {
				t.Errorf("col: got %d, want %d", col, tc.expectedCol)
			}
		})
	}
}

func TestAttrValStart(t *testing.T) {
	testCases := []struct {
		name              string
		input             string
		expectedAttrStart int
	}{
		{
			name:              "double quoted value",
			input:             `<div class="foo">`,
			expectedAttrStart: 11,
		},
		{
			name:              "unquoted value",
			input:             `<div class=foo>`,
			expectedAttrStart: 11,
		},
		{
			name:              "boolean attribute",
			input:             `<div checked>`,
			expectedAttrStart: -1,
		},
		{
			name:              "value with spaces around equals",
			input:             `<div class = "foo">`,
			expectedAttrStart: 13,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lexer := NewLexer([]byte(tc.input))

			for {
				tt := lexer.Next()
				if tt == AttributeToken {
					break
				}

				if tt == ErrorToken {
					t.Fatal("reached ErrorToken without finding AttributeToken")
				}
			}

			if lexer.AttrValStart() != tc.expectedAttrStart {
				t.Errorf("AttrValStart: got %d, want %d", lexer.AttrValStart(), tc.expectedAttrStart)
			}
		})
	}
}

func TestSourceBytesSlicing(t *testing.T) {
	input := "<div>hello</div>"
	lexer := NewLexer([]byte(input))

	if string(lexer.SourceBytes()) != input {
		t.Fatalf("SourceBytes: got %q, want %q", string(lexer.SourceBytes()), input)
	}

	tt := lexer.Next()
	if tt != StartTagToken {
		t.Fatalf("first token: got %v, want StartTagToken", tt)
	}

	rawToken := lexer.SourceBytes()[lexer.TokenStart():lexer.TokenEnd()]
	if string(rawToken) != "<div" {
		t.Errorf("raw start tag: got %q, want %q", string(rawToken), "<div")
	}
}

func TestZeroCopy(t *testing.T) {
	source := []byte("<div>hello</div>")
	lexer := NewLexer(source)

	lexer.Next()

	text := lexer.Text()
	if len(text) == 0 {
		t.Fatal("Text() returned empty slice")
	}

	textPtr := uintptr(unsafe.Pointer(&text[0]))
	sourceStart := uintptr(unsafe.Pointer(&source[0]))
	sourceEnd := uintptr(unsafe.Pointer(&source[len(source)-1]))

	if textPtr < sourceStart || textPtr > sourceEnd {
		t.Error("Text() does not appear to be a sub-slice of source")
	}
}

func TestErrReturnsEOF(t *testing.T) {
	lexer := NewLexer([]byte("<div>"))

	lexer.Next()
	lexer.Next()
	tt := lexer.Next()

	if tt != ErrorToken {
		t.Fatalf("expected ErrorToken, got %v", tt)
	}

	if !errors.Is(lexer.Err(), io.EOF) {
		t.Errorf("Err: got %v, want io.EOF", lexer.Err())
	}
}

func TestMultiLinePositionTracking(t *testing.T) {
	input := "<div>\n  <span\n    class=\"foo\"\n  >\n    text\n  </span>\n</div>"
	lexer := NewLexer([]byte(input))

	type tokenInfo struct {
		tokenType TokenType
		line      int
		col       int
	}

	var tokens []tokenInfo

	for {
		tt := lexer.Next()
		tokens = append(tokens, tokenInfo{
			tokenType: tt,
			line:      lexer.TokenLine(),
			col:       lexer.TokenCol(),
		})

		if tt == ErrorToken {
			break
		}
	}

	if tokens[0].line != 1 || tokens[0].col != 1 {
		t.Errorf("div tag: got line %d col %d, want line 1 col 1", tokens[0].line, tokens[0].col)
	}

	for _, tok := range tokens {
		if tok.tokenType == StartTagToken && tok.line == 2 {
			if tok.col != 3 {
				t.Errorf("span tag: got col %d, want 3", tok.col)
			}

			break
		}
	}
}

func assertTokenSequence(t *testing.T, input string, expected []expectedToken) {
	t.Helper()
	lexer := NewLexer([]byte(input))

	for i, exp := range expected {
		tt := lexer.Next()

		if tt != exp.tokenType {
			t.Fatalf("token %d: got type %v, want %v (text=%q)", i, tt, exp.tokenType, string(lexer.Text()))
		}

		if exp.text != "" && string(lexer.Text()) != exp.text {
			t.Errorf("token %d: got text %q, want %q", i, string(lexer.Text()), exp.text)
		}

		if exp.attrVal != "" && string(lexer.AttrVal()) != exp.attrVal {
			t.Errorf("token %d: got attrVal %q, want %q", i, string(lexer.AttrVal()), exp.attrVal)
		}

		if exp.attrValIsNil && lexer.AttrVal() != nil {
			t.Errorf("token %d: expected nil AttrVal, got %q", i, string(lexer.AttrVal()))
		}

		if exp.checkLine && lexer.TokenLine() != exp.tokenLine {
			t.Errorf("token %d: got line %d, want %d", i, lexer.TokenLine(), exp.tokenLine)
		}

		if exp.checkCol && lexer.TokenCol() != exp.tokenCol {
			t.Errorf("token %d: got col %d, want %d", i, lexer.TokenCol(), exp.tokenCol)
		}

		if exp.checkStart && lexer.TokenStart() != exp.tokenStart {
			t.Errorf("token %d: got start %d, want %d", i, lexer.TokenStart(), exp.tokenStart)
		}

		if exp.checkEnd && lexer.TokenEnd() != exp.tokenEnd {
			t.Errorf("token %d: got end %d, want %d", i, lexer.TokenEnd(), exp.tokenEnd)
		}

		if exp.checkAttrValPos && lexer.AttrValStart() != exp.attrValStart {
			t.Errorf("token %d: got attrValStart %d, want %d", i, lexer.AttrValStart(), exp.attrValStart)
		}
	}
}

func TestResumeAfterRawText_Clamps(t *testing.T) {
	source := []byte("<div>hello</div>")
	lexer := NewLexer(source)

	lexer.ResumeAfterRawText(-1)
	if lexer.cursor != 0 {
		t.Errorf("negative offset must clamp to 0, got %d", lexer.cursor)
	}

	lexer.ResumeAfterRawText(1 << 20)
	if lexer.cursor != len(source) {
		t.Errorf("oversize offset must clamp to len(source)=%d, got %d", len(source), lexer.cursor)
	}
}

func TestResumeAfterRawText_AdvancesPastMidRune(t *testing.T) {
	source := []byte("<p>é</p>")
	lexer := NewLexer(source)

	midRuneOffset := 4
	if RuneStart := source[midRuneOffset] & 0xC0; RuneStart != 0x80 {
		t.Fatalf("expected continuation byte at offset %d, got 0x%X", midRuneOffset, source[midRuneOffset])
	}
	lexer.ResumeAfterRawText(midRuneOffset)
	if lexer.cursor == midRuneOffset {
		t.Errorf("ResumeAfterRawText into a continuation byte must advance to next rune start, got cursor %d", lexer.cursor)
	}
}

func TestResumeAfterRawText_ResetsTagState(t *testing.T) {
	source := []byte("<script>var x = 1;</script><p>after</p>")
	lexer := NewLexer(source)

	lexer.Next()
	lexer.Next()
	endOfScript := bytesIndex(source, []byte("</script>"))
	if endOfScript < 0 {
		t.Fatal("setup failure: </script> not found")
	}
	lexer.ResumeAfterRawText(endOfScript + len("</script>"))
	if lexer.insideTag {
		t.Error("ResumeAfterRawText must clear insideTag")
	}
	if lexer.rawTextTag != rawTextNone {
		t.Error("ResumeAfterRawText must clear rawTextTag")
	}

	tt := lexer.Next()
	if tt != StartTagToken {
		t.Fatalf("expected StartTagToken after ResumeAfterRawText, got %v", tt)
	}
	if string(lexer.Text()) != "p" {
		t.Errorf("unexpected tag name after ResumeAfterRawText: %q", lexer.Text())
	}
}

func bytesIndex(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
loop:
	for index := 0; index+len(needle) <= len(haystack); index++ {
		for needleIndex, needleByte := range needle {
			if haystack[index+needleIndex] != needleByte {
				continue loop
			}
		}
		return index
	}
	return -1
}

func TestResumeAfterRawText_TokenEquivalence(t *testing.T) {
	type capturedToken struct {
		ttype TokenType
		start int
		end   int
		text  string
	}

	collect := func(l *Lexer) []capturedToken {
		var out []capturedToken
		for {
			tt := l.Next()
			if tt == ErrorToken {
				return out
			}
			out = append(out, capturedToken{tt, l.TokenStart(), l.TokenEnd(), string(l.Text())})
		}
	}

	cases := []struct {
		name   string
		source string
		marker string
	}{
		{
			name:   "plain html",
			source: "<div>before</div><span>after</span>",
			marker: "</div>",
		},
		{
			name:   "after script body",
			source: "<script>var x = 1;</script><p>hi</p>",
			marker: "</script>",
		},
		{
			name:   "after style body",
			source: "<style>.x{color:red}</style><p>hi</p>",
			marker: "</style>",
		},
		{
			name:   "across multibyte content",
			source: "<p>café</p><span>after</span>",
			marker: "</p>",
		},
		{
			name:   "across multiple newlines",
			source: "<p>line1\nline2\nline3</p>\n<span>x</span>",
			marker: "</p>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(tc.source)
			boundary := bytesIndex(source, []byte(tc.marker)) + len(tc.marker)
			if boundary <= 0 {
				t.Fatalf("setup failure: marker %q not found", tc.marker)
			}

			fullStream := collect(NewLexer(source))

			resumed := NewLexer(source)
			resumed.ResumeAfterRawText(boundary)
			resumedStream := collect(resumed)

			var fullSuffix []capturedToken
			for _, tok := range fullStream {
				if tok.start >= boundary {
					fullSuffix = append(fullSuffix, tok)
				}
			}

			if len(fullSuffix) != len(resumedStream) {
				t.Fatalf("token count mismatch past boundary %d: full=%d resumed=%d\nfull: %+v\nresumed: %+v",
					boundary, len(fullSuffix), len(resumedStream), fullSuffix, resumedStream)
			}
			for i := range fullSuffix {
				if fullSuffix[i] != resumedStream[i] {
					t.Errorf("token %d differs:\n  full:    %+v\n  resumed: %+v", i, fullSuffix[i], resumedStream[i])
				}
			}
		})
	}
}

func TestResumeAfterRawText_ZeroValueSafe(t *testing.T) {
	source := []byte("<p>x</p><div>y</div>")
	lexer := NewLexer(source)
	lexer.Next()
	lexer.Next()
	lexer.Next()

	lexer.ResumeAfterRawText(len("<p>x</p>"))

	if lexer.err != nil {
		t.Errorf("err must be nil after resume, got %v", lexer.err)
	}
	if lexer.text != nil {
		t.Errorf("text must be nil after resume, got %q", lexer.text)
	}
	if lexer.attributeValue != nil {
		t.Errorf("attributeValue must be nil after resume, got %q", lexer.attributeValue)
	}
	if lexer.attributeValueStart != -1 {
		t.Errorf("attributeValueStart must be -1 after resume, got %d", lexer.attributeValueStart)
	}
	if lexer.insideTag {
		t.Error("insideTag must be false after resume")
	}
	if lexer.rawTextTag != rawTextNone {
		t.Errorf("rawTextTag must be rawTextNone after resume, got %v", lexer.rawTextTag)
	}

	tt := lexer.Next()
	if tt != StartTagToken || string(lexer.Text()) != "div" {
		t.Errorf("expected <div> StartTagToken, got %v / %q", tt, lexer.Text())
	}
}

func TestLexer_ZeroAllocPerLexPass(t *testing.T) {
	source := []byte(`<div class="c"><p>hello</p><script>const x = 1;</script></div>`)

	allocs := testing.AllocsPerRun(100, func() {
		var lexer Lexer
		lexer.Init(source)
		for lexer.Next() != ErrorToken {
		}
	})
	if allocs != 0 {
		t.Errorf("value-type Lexer must be zero-alloc on newline-free input, got %v allocs", allocs)
	}

	heapAllocs := testing.AllocsPerRun(100, func() {
		l := NewLexer(source)
		for l.Next() != ErrorToken {
		}
	})
	if heapAllocs != 0 {
		t.Errorf("NewLexer must inline + stack-allocate on newline-free input, got %v allocs", heapAllocs)
	}
}
