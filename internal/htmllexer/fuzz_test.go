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

//go:build fuzz

package htmllexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzLexer(f *testing.F) {
	f.Add([]byte(`<div class="a" :b='c' d=e f>text {{ x }}</div>`))
	f.Add([]byte("<p>\n  café 日本\n</p>\n<br/>"))
	f.Add([]byte(`<!DOCTYPE html><!-- c --><!--! x --!><![CDATA[a]]><?pi?>`))
	f.Add([]byte(`<script>let a = "</scr" + "ipt>";</script><style>a{}</style>`))
	f.Add([]byte(`<textarea></textarea ><title>t</title><xmp><b></xmp><iframe></iframe>`))
	f.Add([]byte(`<svg viewBox="0 0 1 1"><path d="M0"/></svg><math><mi>x</mi></math>`))
	f.Add([]byte("<plaintext><p>everything after"))
	f.Add([]byte("<p a=\"\xff\">\x80\xe2\x82</p>\n<b>\xc3"))
	f.Add([]byte("<"))
	f.Add([]byte("</"))
	f.Add([]byte("<a"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, input []byte) {
		lexer := NewLexer(input)
		previousStart := 0
		maximumTokens := 2*len(input) + 2

		for tokenCount := 0; ; tokenCount++ {
			require.LessOrEqual(t, tokenCount, maximumTokens, "lexer did not reach the end of the input")

			tokenType := lexer.Next()
			start, end := lexer.TokenStart(), lexer.TokenEnd()
			require.GreaterOrEqual(t, start, previousStart, "token starts must not move backwards")
			require.LessOrEqual(t, start, end, "token must not end before it starts")
			require.LessOrEqual(t, end, len(input), "token must end within the input")
			previousStart = start

			wantLine, wantColumn := recountPosition(input, start)
			assert.Equal(t, wantLine, lexer.TokenLine(), "token line at offset %d", start)
			assert.Equal(t, wantColumn, lexer.TokenCol(), "token column at offset %d", start)

			line, column := lexer.PositionAt(end)
			wantLine, wantColumn = recountPosition(input, end)
			assert.Equal(t, wantLine, line, "line at token end %d", end)
			assert.Equal(t, wantColumn, column, "column at token end %d", end)

			if tokenType == ErrorToken {
				return
			}
		}
	})
}
