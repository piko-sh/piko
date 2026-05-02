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

func span1() querier_dto.TextSpan {
	return querier_dto.TextSpan{Line: 1, Column: 1, EndLine: 1, EndColumn: 2}
}

func hasCode(diagnostics []querier_dto.SourceError, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestValidateCall_BothPositionalsFilledNoDiagnostics(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	call := &callArgs{
		closed: true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "GetUser"}, span: span1()},
			{value: parsedValue{raw: "one"}, span: span1()},
		},
	}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostics := validateCall(spec, call, errorBuilder, "piko")

	assert.Empty(t, diagnostics)
}

func TestValidateCall_MissingRequiredEmitsQ031(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	call := &callArgs{
		closed:    true,
		closeSpan: span1(),
	}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostics := validateCall(spec, call, errorBuilder, "piko")

	require.NotEmpty(t, diagnostics)
	assert.True(t, hasCode(diagnostics, querier_dto.CodeMissingRequired))
}

func TestValidateCall_ExtraPositionalEmitsSyntax(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	call := &callArgs{
		closed: true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "GetUser"}, span: span1()},
			{value: parsedValue{raw: "one"}, span: span1()},
			{value: parsedValue{raw: "extra"}, span: span1()},
		},
	}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostics := validateCall(spec, call, errorBuilder, "piko")

	assert.True(t, hasCode(diagnostics, querier_dto.CodeDirectiveSyntax))
}

func TestValidatePositionalCallArgs_BadEnumValueEmitsQ028(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	call := &callArgs{
		closed: true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "GetUser"}, span: span1()},
			{value: parsedValue{raw: "notACommand"}, span: span1()},
		},
	}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	filled := make([]bool, len(spec.Positionals))
	diagnostics := validatePositionalCallArgs(spec, call, filled, errorBuilder)

	assert.True(t, hasCode(diagnostics, querier_dto.CodeInvalidKeywordArgumentValue))
}

func TestValidatePositionalCallArgs_IgnoresExtra(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	call := &callArgs{
		closed: true,
		positionals: []parsedPositional{
			{value: parsedValue{raw: "GetUser"}, span: span1()},
			{value: parsedValue{raw: "one"}, span: span1()},
			{value: parsedValue{raw: "ignored"}, span: span1()},
		},
	}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	filled := make([]bool, len(spec.Positionals))
	diagnostics := validatePositionalCallArgs(spec, call, filled, errorBuilder)

	assert.Empty(t, diagnostics)
	assert.True(t, filled[0])
	assert.True(t, filled[1])
}

func TestValidateKeywordCallArgs_DuplicateKeyEmitsQ029(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	call := &callArgs{
		closed: true,
		keywordArguments: []parsedKeywordArgument{
			{key: "readonly", value: parsedValue{raw: "true"}, keySpan: span1(), valueSpan: span1()},
			{key: "readonly", value: parsedValue{raw: "false"}, keySpan: span1(), valueSpan: span1()},
		},
	}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	filled := make([]bool, len(spec.Positionals))
	diagnostics := validateKeywordCallArgs(spec, call, filled, errorBuilder, "piko")

	assert.True(t, hasCode(diagnostics, querier_dto.CodeDuplicateKeywordArgument))
}

func TestValidateKeywordCallArgs_RoutesToPositionalSlot(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	call := &callArgs{
		closed: true,
		keywordArguments: []parsedKeywordArgument{
			{key: "name", value: parsedValue{raw: "GetUser"}, keySpan: span1(), valueSpan: span1()},
			{key: "command", value: parsedValue{raw: "one"}, keySpan: span1(), valueSpan: span1()},
		},
	}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	filled := make([]bool, len(spec.Positionals))
	diagnostics := validateKeywordCallArgs(spec, call, filled, errorBuilder, "piko")

	assert.Empty(t, diagnostics)
	assert.True(t, filled[0])
	assert.True(t, filled[1])
}

func TestProcessKeywordCallArg_DoubleFillEmitsDuplicate(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	filled := []bool{true, false}
	keyword := &parsedKeywordArgument{key: "name", value: parsedValue{raw: "GetUser"}, keySpan: span1(), valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostics := processKeywordCallArg(spec, keyword, filled, errorBuilder, "piko")

	assert.True(t, hasCode(diagnostics, querier_dto.CodeDuplicateKeywordArgument))
}

func TestProcessKeywordCallArg_UnknownKeyEmitsQ027(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	filled := make([]bool, len(spec.Positionals))
	keyword := &parsedKeywordArgument{key: "nonexistent", value: parsedValue{raw: "x"}, keySpan: span1(), valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostics := processKeywordCallArg(spec, keyword, filled, errorBuilder, "piko")

	assert.True(t, hasCode(diagnostics, querier_dto.CodeUnknownKeywordArgument))
}

func TestProcessKeywordCallArg_KnownKeywordValidatesValue(t *testing.T) {
	t.Parallel()

	spec, ok := querier_dto.LookupDirective("piko.query")
	require.True(t, ok)

	filled := make([]bool, len(spec.Positionals))
	keyword := &parsedKeywordArgument{key: "readonly", value: parsedValue{raw: "notabool"}, keySpan: span1(), valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostics := processKeywordCallArg(spec, keyword, filled, errorBuilder, "piko")

	assert.True(t, hasCode(diagnostics, querier_dto.CodeInvalidKeywordArgumentValue))
}

func TestValidateListKeywordValue_RejectsNonList(t *testing.T) {
	t.Parallel()

	keyword := &parsedKeywordArgument{key: "columns", value: parsedValue{raw: "a"}, valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostic := validateListKeywordValue(keyword, errorBuilder)

	require.NotNil(t, diagnostic)
	assert.Equal(t, querier_dto.CodeInvalidKeywordArgumentValue, diagnostic.Code)
}

func TestValidateListKeywordValue_AcceptsList(t *testing.T) {
	t.Parallel()

	keyword := &parsedKeywordArgument{
		key:       "columns",
		value:     parsedValue{isList: true, asList: []string{"a", "b"}},
		valueSpan: span1(),
	}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostic := validateListKeywordValue(keyword, errorBuilder)

	assert.Nil(t, diagnostic)
}

func TestValidateIntKeywordValue_AcceptsInteger(t *testing.T) {
	t.Parallel()

	keyword := &parsedKeywordArgument{key: "max", value: parsedValue{raw: "100"}, valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostic := validateIntKeywordValue(keyword, errorBuilder)

	assert.Nil(t, diagnostic)
}

func TestValidateIntKeywordValue_RejectsNonInteger(t *testing.T) {
	t.Parallel()

	keyword := &parsedKeywordArgument{key: "max", value: parsedValue{raw: "manymany"}, valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostic := validateIntKeywordValue(keyword, errorBuilder)

	require.NotNil(t, diagnostic)
	assert.Equal(t, querier_dto.CodeInvalidKeywordArgumentValue, diagnostic.Code)
}

func TestValidateBoolKeywordValue_AcceptsTrueOrFalse(t *testing.T) {
	t.Parallel()

	spec := &querier_dto.KeywordArgumentSpec{Name: "readonly", Kind: querier_dto.KeywordArgumentBool, AllowedValues: []string{"true", "false"}}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	for _, raw := range []string{"true", "false", "TRUE", "False"} {
		keyword := &parsedKeywordArgument{key: "readonly", value: parsedValue{raw: raw}, valueSpan: span1()}
		diagnostic := validateBoolKeywordValue(spec, keyword, errorBuilder)
		assert.Nilf(t, diagnostic, "expected %q to be accepted as a bool", raw)
	}
}

func TestValidateBoolKeywordValue_RejectsOtherLiterals(t *testing.T) {
	t.Parallel()

	spec := &querier_dto.KeywordArgumentSpec{Name: "readonly", Kind: querier_dto.KeywordArgumentBool, AllowedValues: []string{"true", "false"}}
	keyword := &parsedKeywordArgument{key: "readonly", value: parsedValue{raw: "trueish"}, valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostic := validateBoolKeywordValue(spec, keyword, errorBuilder)

	require.NotNil(t, diagnostic)
	assert.Equal(t, querier_dto.CodeInvalidKeywordArgumentValue, diagnostic.Code)
	assert.Equal(t, "true", diagnostic.Suggestion)
}

func TestValidateIdentKeywordValue_AcceptsWhenAllowedListIsEmpty(t *testing.T) {
	t.Parallel()

	spec := &querier_dto.KeywordArgumentSpec{Name: "type", Kind: querier_dto.KeywordArgumentIdent}
	keyword := &parsedKeywordArgument{key: "type", value: parsedValue{raw: "uuid"}, valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostic := validateIdentKeywordValue(spec, keyword, errorBuilder)

	assert.Nil(t, diagnostic)
}

func TestValidateIdentKeywordValue_AcceptsAllowedValueCaseInsensitive(t *testing.T) {
	t.Parallel()

	spec := &querier_dto.KeywordArgumentSpec{Name: "dynamic", Kind: querier_dto.KeywordArgumentIdent, AllowedValues: []string{"runtime"}}
	keyword := &parsedKeywordArgument{key: "dynamic", value: parsedValue{raw: "RuntimE"}, valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostic := validateIdentKeywordValue(spec, keyword, errorBuilder)

	assert.Nil(t, diagnostic)
}

func TestValidateIdentKeywordValue_RejectsNonAllowedValue(t *testing.T) {
	t.Parallel()

	spec := &querier_dto.KeywordArgumentSpec{Name: "dynamic", Kind: querier_dto.KeywordArgumentIdent, AllowedValues: []string{"runtime"}}
	keyword := &parsedKeywordArgument{key: "dynamic", value: parsedValue{raw: "compile"}, valueSpan: span1()}
	errorBuilder := querier_dto.NewErrorBuilder("test.sql")

	diagnostic := validateIdentKeywordValue(spec, keyword, errorBuilder)

	require.NotNil(t, diagnostic)
	assert.Equal(t, querier_dto.CodeInvalidKeywordArgumentValue, diagnostic.Code)
}
