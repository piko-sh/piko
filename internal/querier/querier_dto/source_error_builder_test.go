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

package querier_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevenshteinDistance_ReturnsEditDistanceBetweenStrings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		source   string
		target   string
		expected int
	}{
		{name: "identical strings have zero distance", source: "kitten", target: "kitten", expected: 0},
		{name: "single substitution", source: "kitten", target: "sitten", expected: 1},
		{name: "classic kitten to sitting is three", source: "kitten", target: "sitting", expected: 3},
		{name: "empty source returns target length", source: "", target: "abc", expected: 3},
		{name: "empty target returns source length", source: "abc", target: "", expected: 3},
		{name: "both empty returns zero", source: "", target: "", expected: 0},
		{name: "single insertion", source: "cat", target: "cats", expected: 1},
		{name: "single deletion", source: "cats", target: "cat", expected: 1},
		{name: "multibyte runes counted per rune", source: "café", target: "cafe", expected: 1},
		{name: "completely different equal length", source: "abc", target: "xyz", expected: 3},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, levenshteinDistance(testCase.source, testCase.target))
		})
	}
}

func TestLevenshteinDistance_IsSymmetric(t *testing.T) {
	t.Parallel()

	forward := levenshteinDistance("sunday", "saturday")
	backward := levenshteinDistance("saturday", "sunday")
	assert.Equal(t, forward, backward)
	assert.Equal(t, 3, forward)
}

func TestFindClosestSuggestion_PicksNearestCandidateWithinThreshold(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		input      string
		candidates []string
		expected   string
	}{
		{
			name:       "empty input yields no suggestion",
			input:      "",
			candidates: []string{"alpha", "beta"},
			expected:   "",
		},
		{
			name:       "nil candidates yield no suggestion",
			input:      "alpha",
			candidates: nil,
			expected:   "",
		},
		{
			name:       "empty candidate slice yields no suggestion",
			input:      "alpha",
			candidates: []string{},
			expected:   "",
		},
		{
			name:       "single character typo returns the close candidate",
			input:      "embedd",
			candidates: []string{"embed", "column", "param"},
			expected:   "embed",
		},
		{
			name:       "case is folded before comparison",
			input:      "EMBED",
			candidates: []string{"embed", "column"},
			expected:   "embed",
		},
		{
			name:       "candidate casing is preserved in the returned value",
			input:      "embad",
			candidates: []string{"Embed", "Column"},
			expected:   "Embed",
		},
		{
			name:       "wildly different input returns no suggestion",
			input:      "zzzzzzzzzz",
			candidates: []string{"embed", "column"},
			expected:   "",
		},
		{
			name:       "first close candidate wins on equal distance",
			input:      "aaa",
			candidates: []string{"aab", "aac"},
			expected:   "aab",
		},
		{
			name:       "short input below minimum threshold still matches one edit",
			input:      "ab",
			candidates: []string{"ac"},
			expected:   "ac",
		},
		{
			name:       "exact match is its own suggestion",
			input:      "param",
			candidates: []string{"param", "embed"},
			expected:   "param",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, findClosestSuggestion(testCase.input, testCase.candidates))
		})
	}
}

func TestFindClosestSuggestion_AppliesLengthProportionalThreshold(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "transmission", findClosestSuggestion("transaction", []string{"transmission"}))

	assert.Equal(t, "", findClosestSuggestion("abc", []string{"xyz"}))
}

func newSpan() TextSpan {
	return TextSpan{Line: 3, Column: 5, EndLine: 3, EndColumn: 12}
}

func TestNewErrorBuilder_BindsFilename(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("queries/users.sql")
	assert.Equal(t, "queries/users.sql", builder.Filename)
}

func TestErrorBuilderAt_ProducesErrorSeverityWithSpanCopiedAcross(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.At(newSpan(), CodeDirectiveSyntax, "boom")

	assert.Equal(t, SourceError{
		Filename:  "file.sql",
		Message:   "boom",
		Code:      CodeDirectiveSyntax,
		Line:      3,
		Column:    5,
		EndLine:   3,
		EndColumn: 12,
		Severity:  SeverityError,
	}, source)
}

func TestErrorBuilderUnknownDirective_AttachesSuggestionWhenCandidateIsClose(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.UnknownDirective(newSpan(), "piko.embedd", []string{"piko.embed", "piko.column"})

	assert.Equal(t, CodeUnknownDirective, source.Code)
	assert.Equal(t, "piko.embed", source.Suggestion)
	assert.Equal(t, `unknown directive "piko.embedd"; did you mean "piko.embed"?`, source.Message)
}

func TestErrorBuilderUnknownDirective_OmitsSuggestionWhenNoCandidateIsClose(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.UnknownDirective(newSpan(), "totallyunrelated", []string{"piko.embed"})

	assert.Equal(t, CodeUnknownDirective, source.Code)
	assert.Empty(t, source.Suggestion)
	assert.Equal(t, `unknown directive "totallyunrelated"`, source.Message)
}

func TestErrorBuilderUnknownKeywordArgument_AttachesSuggestionAndNamesDirective(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.UnknownKeywordArgument(newSpan(), "piko.embed", "form", []string{"from", "as"})

	assert.Equal(t, CodeUnknownKeywordArgument, source.Code)
	assert.Equal(t, "from", source.Suggestion)
	assert.Equal(t, `unknown keyword argument "form" on piko.embed; did you mean "from"?`, source.Message)
}

func TestErrorBuilderUnknownKeywordArgument_OmitsSuggestionWhenFarFromCandidates(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.UnknownKeywordArgument(newSpan(), "piko.embed", "nonsense", []string{"from"})

	assert.Equal(t, CodeUnknownKeywordArgument, source.Code)
	assert.Empty(t, source.Suggestion)
	assert.Equal(t, `unknown keyword argument "nonsense" on piko.embed`, source.Message)
}

func TestErrorBuilderInvalidEnum_AttachesSuggestionAndListsAllowedValues(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.InvalidEnum(newSpan(), "command", "ome", []string{"one", "many", "exec"})

	assert.Equal(t, CodeInvalidKeywordArgumentValue, source.Code)
	assert.Equal(t, "one", source.Suggestion)
	assert.Equal(t, `invalid value "ome" for keyword argument "command"; did you mean "one"? allowed: one, many, exec`, source.Message)
}

func TestErrorBuilderInvalidEnum_OmitsSuggestionButStillListsAllowedValues(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.InvalidEnum(newSpan(), "command", "zzzzz", []string{"one", "many"})

	assert.Equal(t, CodeInvalidKeywordArgumentValue, source.Code)
	assert.Empty(t, source.Suggestion)
	assert.Equal(t, `invalid value "zzzzz" for keyword argument "command"; allowed: one, many`, source.Message)
}

func TestErrorBuilderInvalidValueShape_DescribesExpectedShapeWithoutSuggestion(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.InvalidValueShape(newSpan(), "max", "an integer")

	assert.Equal(t, CodeInvalidKeywordArgumentValue, source.Code)
	assert.Empty(t, source.Suggestion)
	assert.Equal(t, `invalid value for keyword argument "max": expected an integer`, source.Message)
}

func TestErrorBuilderDuplicate_ReportsRepeatedKey(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.Duplicate(newSpan(), "type")

	assert.Equal(t, CodeDuplicateKeywordArgument, source.Code)
	assert.Equal(t, `duplicate keyword argument "type"`, source.Message)
}

func TestErrorBuilderMissingRequired_NamesDirectiveAndMissingPart(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.MissingRequired(newSpan(), "piko", "positional argument name")

	assert.Equal(t, CodeMissingRequired, source.Code)
	assert.Equal(t, "missing required positional argument name on piko", source.Message)
}

func TestErrorBuilderUnclosed_RequestsMatchingParen(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.Unclosed(newSpan(), "directive call")

	assert.Equal(t, CodeUnclosedDirective, source.Code)
	assert.Equal(t, "unclosed directive call; expected matching ')'", source.Message)
}

func TestErrorBuilderDirectiveSyntax_FallsBackToSyntaxCode(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.DirectiveSyntax(newSpan(), "unexpected character '#'")

	assert.Equal(t, CodeDirectiveSyntax, source.Code)
	assert.Equal(t, "unexpected character '#'", source.Message)
}

func TestErrorBuilderParameterBindingSyntax_AttachesSuggestionWhenTokenIsNearMiss(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.ParameterBindingSyntax(newSpan(), `"as"`, "az", []string{"as", "piko"})

	assert.Equal(t, CodeParameterBindingSyntax, source.Code)
	assert.Equal(t, "as", source.Suggestion)
	assert.Equal(t, `expected "as", got "az"; did you mean "as"?`, source.Message)
}

func TestErrorBuilderParameterBindingSyntax_OmitsSuggestionWhenNoCandidateClose(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.ParameterBindingSyntax(newSpan(), `"as"`, "wildlydifferent", []string{"as"})

	assert.Equal(t, CodeParameterBindingSyntax, source.Code)
	assert.Empty(t, source.Suggestion)
	assert.Equal(t, `expected "as", got "wildlydifferent"`, source.Message)
}

func TestErrorBuilderParameterBindingSyntax_OmitsSuggestionWhenCandidatesEmpty(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.ParameterBindingSyntax(newSpan(), `"as"`, "as", nil)

	assert.Equal(t, CodeParameterBindingSyntax, source.Code)
	assert.Empty(t, source.Suggestion)
	assert.Equal(t, `expected "as", got "as"`, source.Message)
}

func TestErrorBuilderInvalidListLiteral_ReportsUnexpectedClosingToken(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.InvalidListLiteral(newSpan(), ")")

	assert.Equal(t, CodeInvalidListLiteral, source.Code)
	assert.Equal(t, `expected ']' to close list literal, got ")"`, source.Message)
}

func TestErrorBuilderUnterminatedString_NamesExpectedClosingQuote(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.UnterminatedString(newSpan(), `"`)

	assert.Equal(t, CodeUnterminatedString, source.Code)
	assert.Equal(t, `unterminated string literal; expected closing "`, source.Message)
}

func TestErrorBuilderUnknownOverrideColumn_AttachesSuggestionWhenColumnClose(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.UnknownOverrideColumn(newSpan(), "emial", []string{"email", "name"})

	assert.Equal(t, CodeUnknownOverrideColumn, source.Code)
	assert.Equal(t, "email", source.Suggestion)
	assert.Equal(t, `piko.column references unknown output column "emial"; did you mean "email"?`, source.Message)
}

func TestErrorBuilderUnknownOverrideColumn_OmitsSuggestionWhenColumnFar(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.UnknownOverrideColumn(newSpan(), "nonexistent", []string{"email"})

	assert.Equal(t, CodeUnknownOverrideColumn, source.Code)
	assert.Empty(t, source.Suggestion)
	assert.Equal(t, `piko.column references unknown output column "nonexistent"`, source.Message)
}

func TestErrorBuilderUnknownOverrideMigrationColumn_AttachesSuggestionWhenClose(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.UnknownOverrideMigrationColumn(newSpan(), "users.emial", []string{"users.email", "users.name"})

	assert.Equal(t, CodeUnknownOverrideMigrationColumn, source.Code)
	assert.Equal(t, "users.email", source.Suggestion)
	assert.Equal(t, `piko.column references unknown column "users.emial"; did you mean "users.email"?`, source.Message)
}

func TestErrorBuilderUnknownOverrideMigrationColumn_OmitsSuggestionWhenFar(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.UnknownOverrideMigrationColumn(newSpan(), "wholly.different", []string{"users.email"})

	assert.Equal(t, CodeUnknownOverrideMigrationColumn, source.Code)
	assert.Empty(t, source.Suggestion)
	assert.Equal(t, `piko.column references unknown column "wholly.different"`, source.Message)
}

func TestErrorBuilderMutuallyExclusiveKeywordArgument_NamesBothKeywordArguments(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.MutuallyExclusiveKeywordArgument(newSpan(), "piko.column", "type", "go_type")

	assert.Equal(t, CodeMutuallyExclusiveKeywordArgument, source.Code)
	assert.Equal(t, `piko.column: keyword arguments "type" and "go_type" are mutually exclusive; pick one`, source.Message)
}

func TestErrorBuilderWithSuggestion_SetsSuggestionFieldOnEmittedError(t *testing.T) {
	t.Parallel()

	builder := NewErrorBuilder("file.sql")
	source := builder.withSuggestion(newSpan(), CodeUnknownDirective, "msg", "didyoumean")

	require.Equal(t, "didyoumean", source.Suggestion)
	assert.Equal(t, CodeUnknownDirective, source.Code)
	assert.Equal(t, "msg", source.Message)
	assert.Equal(t, "file.sql", source.Filename)
}
