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

package ast_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryLexer_Comprehensive(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  string
		tokens []QueryToken
	}{
		{
			name:  "Simple tag",
			input: "div",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "div"},
			},
		},
		{
			name:  "Simple ID",
			input: "#main",
			tokens: []QueryToken{
				{Type: TokenHash, Literal: "#"},
				{Type: TokenIdent, Literal: "main"},
			},
		},
		{
			name:  "Simple class",
			input: ".card",
			tokens: []QueryToken{
				{Type: TokenDot, Literal: "."},
				{Type: TokenIdent, Literal: "card"},
			},
		},
		{
			name:  "Chained class and ID",
			input: "p.intro#first",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "p"},
				{Type: TokenDot, Literal: "."},
				{Type: TokenIdent, Literal: "intro"},
				{Type: TokenHash, Literal: "#"},
				{Type: TokenIdent, Literal: "first"},
			},
		},
		{
			name:  "Identifier with hyphens and numbers",
			input: "h1-custom-title_123",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "h1-custom-title_123"},
			},
		},
		{
			name:  "Child combinator",
			input: "ul > li",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "ul"},
				{Type: TokenWhitespace, Literal: " "},
				{Type: TokenCombinator, Literal: ">"},
				{Type: TokenWhitespace, Literal: " "},
				{Type: TokenIdent, Literal: "li"},
			},
		},
		{
			name:  "Adjacent Sibling combinator",
			input: "h2+p",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "h2"},
				{Type: TokenPlus, Literal: "+"},
				{Type: TokenIdent, Literal: "p"},
			},
		},
		{
			name:  "General Sibling combinator",
			input: "h2 ~ p",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "h2"},
				{Type: TokenWhitespace, Literal: " "},
				{Type: TokenTilde, Literal: "~"},
				{Type: TokenWhitespace, Literal: " "},
				{Type: TokenIdent, Literal: "p"},
			},
		},
		{
			name:  "Selector List (comma)",
			input: "div, p, #id",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "div"},
				{Type: TokenComma, Literal: ","},
				{Type: TokenWhitespace, Literal: " "},
				{Type: TokenIdent, Literal: "p"},
				{Type: TokenComma, Literal: ","},
				{Type: TokenWhitespace, Literal: " "},
				{Type: TokenHash, Literal: "#"},
				{Type: TokenIdent, Literal: "id"},
			},
		},
		{
			name:  "All attribute operators",
			input: `[a=b][c~="d"][e|=f][g^=h][i$=j][k*='l']`,
			tokens: []QueryToken{
				{Type: TokenLBracket, Literal: "["}, {Type: TokenIdent, Literal: "a"}, {Type: TokenEquals, Literal: "="}, {Type: TokenIdent, Literal: "b"}, {Type: TokenRBracket, Literal: "]"},
				{Type: TokenLBracket, Literal: "["}, {Type: TokenIdent, Literal: "c"}, {Type: TokenIncludes, Literal: "~="}, {Type: TokenString, Literal: "d"}, {Type: TokenRBracket, Literal: "]"},
				{Type: TokenLBracket, Literal: "["}, {Type: TokenIdent, Literal: "e"}, {Type: TokenDashMatch, Literal: "|="}, {Type: TokenIdent, Literal: "f"}, {Type: TokenRBracket, Literal: "]"},
				{Type: TokenLBracket, Literal: "["}, {Type: TokenIdent, Literal: "g"}, {Type: TokenPrefix, Literal: "^="}, {Type: TokenIdent, Literal: "h"}, {Type: TokenRBracket, Literal: "]"},
				{Type: TokenLBracket, Literal: "["}, {Type: TokenIdent, Literal: "i"}, {Type: TokenSuffix, Literal: "$="}, {Type: TokenIdent, Literal: "j"}, {Type: TokenRBracket, Literal: "]"},
				{Type: TokenLBracket, Literal: "["}, {Type: TokenIdent, Literal: "k"}, {Type: TokenContains, Literal: "*="}, {Type: TokenString, Literal: "l"}, {Type: TokenRBracket, Literal: "]"},
			},
		},
		{
			name:  "Attribute with case-insensitive flag",
			input: `[lang="en" i]`,
			tokens: []QueryToken{
				{Type: TokenLBracket, Literal: "["},
				{Type: TokenIdent, Literal: "lang"},
				{Type: TokenEquals, Literal: "="},
				{Type: TokenString, Literal: "en"},
				{Type: TokenWhitespace, Literal: " "},
				{Type: TokenIdent, Literal: "i"},
				{Type: TokenRBracket, Literal: "]"},
			},
		},
		{
			name:  "Simple pseudo-class",
			input: ":first-child",
			tokens: []QueryToken{
				{Type: TokenColon, Literal: ":"},
				{Type: TokenIdent, Literal: "first-child"},
			},
		},
		{
			name:  "Functional pseudo-class :not()",
			input: ":not(.special)",
			tokens: []QueryToken{
				{Type: TokenColon, Literal: ":"}, {Type: TokenIdent, Literal: "not"}, {Type: TokenLParen, Literal: "("}, {Type: TokenDot, Literal: "."}, {Type: TokenIdent, Literal: "special"}, {Type: TokenRParen, Literal: ")"},
			},
		},
		{
			name:  "Functional pseudo-class :nth-child() with keyword",
			input: ":nth-child(even)",
			tokens: []QueryToken{
				{Type: TokenColon, Literal: ":"}, {Type: TokenIdent, Literal: "nth-child"}, {Type: TokenLParen, Literal: "("}, {Type: TokenIdent, Literal: "even"}, {Type: TokenRParen, Literal: ")"},
			},
		},
		{
			name:  "Functional pseudo-class :nth-child() with formula",
			input: ":nth-of-type(2n+1)",
			tokens: []QueryToken{
				{Type: TokenColon, Literal: ":"},
				{Type: TokenIdent, Literal: "nth-of-type"},
				{Type: TokenLParen, Literal: "("},
				{Type: TokenIdent, Literal: "2n"},
				{Type: TokenPlus, Literal: "+"},
				{Type: TokenIdent, Literal: "1"},
				{Type: TokenRParen, Literal: ")"},
			},
		},
		{
			name:  "Complex selector with multiple features",
			input: `main#app > div.card:not([disabled]):nth-child(2n), footer p`,
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "main"}, {Type: TokenHash, Literal: "#"}, {Type: TokenIdent, Literal: "app"}, {Type: TokenWhitespace, Literal: " "}, {Type: TokenCombinator, Literal: ">"}, {Type: TokenWhitespace, Literal: " "},
				{Type: TokenIdent, Literal: "div"}, {Type: TokenDot, Literal: "."}, {Type: TokenIdent, Literal: "card"}, {Type: TokenColon, Literal: ":"}, {Type: TokenIdent, Literal: "not"}, {Type: TokenLParen, Literal: "("},
				{Type: TokenLBracket, Literal: "["}, {Type: TokenIdent, Literal: "disabled"}, {Type: TokenRBracket, Literal: "]"}, {Type: TokenRParen, Literal: ")"}, {Type: TokenColon, Literal: ":"}, {Type: TokenIdent, Literal: "nth-child"},
				{Type: TokenLParen, Literal: "("}, {Type: TokenIdent, Literal: "2n"}, {Type: TokenRParen, Literal: ")"}, {Type: TokenComma, Literal: ","}, {Type: TokenWhitespace, Literal: " "}, {Type: TokenIdent, Literal: "footer"},
				{Type: TokenWhitespace, Literal: " "}, {Type: TokenIdent, Literal: "p"},
			},
		},
		{
			name:   "Empty input",
			input:  "",
			tokens: []QueryToken{},
		},
		{
			name:  "Input with only whitespace",
			input: "   \t\n   ",
			tokens: []QueryToken{
				{Type: TokenWhitespace, Literal: " "},
			},
		},
		{
			name:  "Illegal characters",
			input: "div@.class!?",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "div"},
				{Type: TokenIllegal, Literal: "@"},
				{Type: TokenDot, Literal: "."},
				{Type: TokenIdent, Literal: "class"},
				{Type: TokenIllegal, Literal: "!"},
				{Type: TokenIllegal, Literal: "?"},
			},
		},
		{
			name:  "String with escaped quote",
			input: `[title="it\'s a test"]`,
			tokens: []QueryToken{
				{Type: TokenLBracket, Literal: "["},
				{Type: TokenIdent, Literal: "title"},
				{Type: TokenEquals, Literal: "="},
				{Type: TokenString, Literal: `it\'s a test`},
				{Type: TokenRBracket, Literal: "]"},
			},
		},

		{
			name:  "Piko namespace: piko:svg",
			input: "piko:svg",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "piko:svg"},
			},
		},
		{
			name:  "Piko namespace: piko:partial",
			input: "piko:partial",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "piko:partial"},
			},
		},
		{
			name:  "Piko namespace: piko:img with class",
			input: "piko:img.icon",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "piko:img"},
				{Type: TokenDot, Literal: "."},
				{Type: TokenIdent, Literal: "icon"},
			},
		},
		{
			name:  "Piko namespace: piko:svg with attribute selector",
			input: "piko:svg[src]",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "piko:svg"},
				{Type: TokenLBracket, Literal: "["},
				{Type: TokenIdent, Literal: "src"},
				{Type: TokenRBracket, Literal: "]"},
			},
		},
		{
			name:  "Standard pseudo-class still works: div:hover",
			input: "div:hover",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "div"},
				{Type: TokenColon, Literal: ":"},
				{Type: TokenIdent, Literal: "hover"},
			},
		},
		{
			name:  "Piko without element (edge case): piko alone",
			input: "piko",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "piko"},
			},
		},
		{
			name:  "Piko colon without following ident (edge case): piko: alone",
			input: "piko:",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "piko"},
				{Type: TokenColon, Literal: ":"},
			},
		},
		{
			name:  "Piko colon followed by digits (also valid): piko:123",
			input: "piko:123",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "piko:123"},
			},
		},
		{
			name:  "Piko colon followed by non-ident char: piko:.",
			input: "piko:.",
			tokens: []QueryToken{
				{Type: TokenIdent, Literal: "piko"},
				{Type: TokenColon, Literal: ":"},
				{Type: TokenDot, Literal: "."},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lexer := NewQueryLexer(tc.input)
			for i, expectedToken := range tc.tokens {
				tok := lexer.NextToken()
				if !assert.Equal(t, expectedToken.Type, tok.Type, "Token %d in test '%s' has wrong type", i, tc.name) {
					t.FailNow()
				}
				if !assert.Equal(t, expectedToken.Literal, tok.Literal, "Token %d in test '%s' has wrong literal", i, tc.name) {
					t.FailNow()
				}
			}

			finalToken := lexer.NextToken()
			assert.Equal(t, TokenEOF, finalToken.Type, "Expected EOF after all tokens in test '%s'", tc.name)
		})
	}
}
