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

package querier_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func runParseCallArgs(content string) (*callArgs, []querier_dto.SourceError) {
	lexer := lexerFromContent(content)
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")
	return parseCallArgs(lexer, errorBuilder, "piko")
}

func runParseValue(content string) (parsedValue, querier_dto.TextSpan, []querier_dto.SourceError) {
	lexer := lexerFromContent(content)
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")
	return parseValue(lexer, errorBuilder, 0)
}

func TestParseCallArgs_EmptyParenList(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("()")

	require.NotNil(t, call)
	assert.True(t, call.closed)
	assert.Empty(t, call.positionals)
	assert.Empty(t, call.keywordArguments)
	assert.Empty(t, diagnostics)
}

func TestParseCallArgs_SinglePositional(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("(GetUser)")

	require.NotNil(t, call)
	require.Len(t, call.positionals, 1)
	assert.Equal(t, "GetUser", call.positionals[0].value.raw)
	assert.Empty(t, diagnostics)
}

func TestParseCallArgs_TwoPositionalsCommaSeparated(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("(GetUser, one)")

	require.NotNil(t, call)
	require.Len(t, call.positionals, 2)
	assert.Equal(t, "GetUser", call.positionals[0].value.raw)
	assert.Equal(t, "one", call.positionals[1].value.raw)
	assert.Empty(t, diagnostics)
}

func TestParseCallArgs_PositionalAndKeyword(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("(GetUser, command: one)")

	require.NotNil(t, call)
	require.Len(t, call.positionals, 1)
	assert.Equal(t, "GetUser", call.positionals[0].value.raw)
	require.Len(t, call.keywordArguments, 1)
	assert.Equal(t, "command", call.keywordArguments[0].key)
	assert.Equal(t, "one", call.keywordArguments[0].value.raw)
	assert.Empty(t, diagnostics)
}

func TestParseCallArgs_PositionalAfterKeywordEmitsSyntax(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("(name: GetUser, extra)")

	require.NotNil(t, call)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeDirectiveSyntax {
			found = true
		}
	}
	assert.True(t, found, "expected directive-syntax diagnostic for positional after keyword")
}

func TestParseCallArgs_MissingOpenParenEmitsSyntax(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("name: GetUser)")

	assert.Nil(t, call)
	require.NotEmpty(t, diagnostics)
	assert.Equal(t, querier_dto.CodeDirectiveSyntax, diagnostics[0].Code)
}

func TestParseCallArgs_UnclosedCallEmitsUnclosed(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("(name: GetUser")

	require.NotNil(t, call)
	assert.False(t, call.closed)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeUnclosedDirective {
			found = true
		}
	}
	assert.True(t, found, "expected unclosed-directive diagnostic")
}

func TestParseCallArgs_GarbageSeparatorRecovers(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("(name: GetUser ; command: one)")

	require.NotNil(t, call)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeDirectiveSyntax {
			found = true
		}
	}
	assert.True(t, found, "expected directive-syntax diagnostic for garbage separator")
}

func TestParseCallArgs_DottedValuePositionalJoinsWithDots(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("(orders.id)")

	require.NotNil(t, call)
	require.Len(t, call.positionals, 1)
	assert.Equal(t, "orders.id", call.positionals[0].value.raw)
	assert.Empty(t, diagnostics)
}

func TestParseSingleArg_KeywordWithStringValue(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs(`(go_type: "github.com/google/uuid.UUID")`)

	require.NotNil(t, call)
	require.Len(t, call.keywordArguments, 1)
	assert.Equal(t, "go_type", call.keywordArguments[0].key)
	assert.Equal(t, "github.com/google/uuid.UUID", call.keywordArguments[0].value.raw)
	assert.Empty(t, diagnostics)
}

func TestParseSingleArg_NumericKeyword(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("(max: 100)")

	require.NotNil(t, call)
	require.Len(t, call.keywordArguments, 1)
	assert.Equal(t, "100", call.keywordArguments[0].value.raw)
	assert.Empty(t, diagnostics)
}

func TestParseSingleArg_NegativeNumericKeyword(t *testing.T) {
	t.Parallel()

	call, diagnostics := runParseCallArgs("(default: -5)")

	require.NotNil(t, call)
	require.Len(t, call.keywordArguments, 1)
	assert.Equal(t, "-5", call.keywordArguments[0].value.raw)
	assert.Empty(t, diagnostics)
}

func TestParseValue_IdentifierValue(t *testing.T) {
	t.Parallel()

	value, _, diagnostics := runParseValue("orders")

	assert.Equal(t, "orders", value.raw)
	assert.False(t, value.isList)
	assert.Empty(t, diagnostics)
}

func TestParseValue_DottedIdentifier(t *testing.T) {
	t.Parallel()

	value, _, diagnostics := runParseValue("piko.foo.bar")

	assert.Equal(t, "piko.foo.bar", value.raw)
	assert.False(t, value.isList)
	assert.Empty(t, diagnostics)
}

func TestParseValue_StringLiteralStripsQuotes(t *testing.T) {
	t.Parallel()

	value, _, diagnostics := runParseValue(`"hello world"`)

	assert.Equal(t, "hello world", value.raw)
	assert.False(t, value.isList)
	assert.Empty(t, diagnostics)
}

func TestParseValue_NegativeNumberLiteral(t *testing.T) {
	t.Parallel()

	value, _, diagnostics := runParseValue("-123")

	assert.Equal(t, "-123", value.raw)
	assert.Empty(t, diagnostics)
}

func TestParseValue_UnexpectedTokenEmitsSyntax(t *testing.T) {
	t.Parallel()

	_, _, diagnostics := runParseValue(":")

	require.NotEmpty(t, diagnostics)
	assert.Equal(t, querier_dto.CodeDirectiveSyntax, diagnostics[0].Code)
}

func TestParseValue_DottedIdentTrailingDotEmitsSyntax(t *testing.T) {
	t.Parallel()

	value, _, diagnostics := runParseValue("piko.")

	assert.Equal(t, "piko", value.raw)
	require.NotEmpty(t, diagnostics)
	assert.Equal(t, querier_dto.CodeDirectiveSyntax, diagnostics[0].Code)
}

func TestParseListValue_EmptyListShapeIsRecognised(t *testing.T) {
	t.Parallel()

	value, _, diagnostics := runParseValue("[]")

	assert.True(t, value.isList)
	assert.Empty(t, value.asList)
	assert.Empty(t, diagnostics)
}

func TestParseListValue_PopulatedListPreservesOrder(t *testing.T) {
	t.Parallel()

	value, _, diagnostics := runParseValue("[a, b, c]")

	assert.True(t, value.isList)
	assert.Equal(t, []string{"a", "b", "c"}, value.asList)
	assert.Empty(t, diagnostics)
}

func TestParseListValue_UnterminatedListResetsAndDiagnoses(t *testing.T) {
	t.Parallel()

	value, _, diagnostics := runParseValue("[a, b")

	assert.False(t, value.isList, "unterminated list must reset isList")
	assert.Empty(t, value.asList)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeUnclosedDirective {
			found = true
		}
	}
	assert.True(t, found, "expected Q032 for unterminated list")
}

func TestParseListValue_MismatchedCloserResetsAndDiagnoses(t *testing.T) {
	t.Parallel()

	value, _, diagnostics := runParseValue("[a, b)")

	assert.False(t, value.isList, "mismatched closer must reset isList")
	assert.Empty(t, value.asList)
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == querier_dto.CodeInvalidListLiteral {
			found = true
		}
	}
	assert.True(t, found, "expected Q034 for mismatched closer")
}

func TestConsumeUntilListBoundary_StopsAtComma(t *testing.T) {
	t.Parallel()

	lexer := lexerFromContent("garbage,next")
	consumeUntilListBoundary(lexer)

	peek := lexer.peek()
	assert.Equal(t, tokenComma, peek.kind, "consume should leave the comma as next")
}

func TestConsumeUntilListBoundary_StopsAtRBracket(t *testing.T) {
	t.Parallel()

	lexer := lexerFromContent("garbage]")
	consumeUntilListBoundary(lexer)

	peek := lexer.peek()
	assert.Equal(t, tokenRBracket, peek.kind)
}

func TestConsumeUntilListBoundary_StopsAtEOF(t *testing.T) {
	t.Parallel()

	lexer := lexerFromContent("garbage_only")
	consumeUntilListBoundary(lexer)

	peek := lexer.peek()
	assert.Equal(t, tokenEOF, peek.kind)
}
