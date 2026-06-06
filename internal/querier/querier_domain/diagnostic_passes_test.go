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

func TestDiagnosticAnalyser_Analyse(t *testing.T) {
	t.Parallel()

	catalogue := newTestCatalogue("main")
	analyser := newDiagnosticAnalyser(catalogue)

	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:          "BrokenQuery",
			Command:       querier_dto.QueryCommandOne,
			OutputColumns: nil,
			Line:          10,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{
			ParameterReferences: []querier_dto.RawParameterReference{},
		},
		ParameterDirectives: []*querier_dto.ParameterDirective{
			{Number: 1, Name: "unused_param", Kind: querier_dto.ParameterDirectiveParam},
		},
	}

	diagnostics := analyser.Analyse(context)

	assert.GreaterOrEqual(t, len(diagnostics), 2,
		"analyser should collect diagnostics from multiple passes")

	codeSet := make(map[string]int)
	for _, diag := range diagnostics {
		codeSet[diag.Code]++
	}

	assert.GreaterOrEqual(t, codeSet["Q009"], 1,
		"command/output mismatch should emit Q009")
	assert.GreaterOrEqual(t, codeSet[querier_dto.CodeUnreferencedParameter], 1,
		"unreferenced parameter should emit its dedicated code")
}

func TestDiagnosticAnalyser_NoDiagnostics(t *testing.T) {
	t.Parallel()

	catalogue := newTestCatalogue("main")
	analyser := newDiagnosticAnalyser(catalogue)

	context := &diagnosticContext{
		Filename: "test.sql",
		Query: &querier_dto.AnalysedQuery{
			Name:    "GoodQuery",
			Command: querier_dto.QueryCommandMany,
			OutputColumns: []querier_dto.OutputColumn{
				{Name: "id"},
				{Name: "email"},
			},
			Line: 1,
		},
		RawAnalysis: &querier_dto.RawQueryAnalysis{
			ParameterReferences: []querier_dto.RawParameterReference{
				{Number: 1},
			},
		},
		ParameterDirectives: []*querier_dto.ParameterDirective{
			{Number: 1, Name: "user_id", Kind: querier_dto.ParameterDirectiveParam},
		},
	}

	diagnostics := analyser.Analyse(context)
	assert.Empty(t, diagnostics)
}

func TestBaseQueryShapeDiagnostics(t *testing.T) {
	t.Parallel()

	type expected struct {
		code     string
		severity querier_dto.ErrorSeverity
	}

	tests := []struct {
		name  string
		query *querier_dto.AnalysedQuery
		want  []expected
	}{
		{
			name: "no_where_no_terminator",
			query: &querier_dto.AnalysedQuery{
				SQL:  "SELECT 1 FROM users",
				Line: 1,
			},
		},
		{
			name: "has_where_emits_hint",
			query: &querier_dto.AnalysedQuery{
				SQL:                     "SELECT 1 FROM users WHERE id = $1",
				BaseQueryHasWhereClause: true,
				Line:                    1,
			},
			want: []expected{{querier_dto.CodeRuntimeBuilderBaseHasWhere, querier_dto.SeverityHint}},
		},
		{
			name: "trailing_semicolon_emits_error",
			query: &querier_dto.AnalysedQuery{
				SQL:  "SELECT 1 FROM users;",
				Line: 1,
			},
			want: []expected{{querier_dto.CodeRuntimeBuilderTrailingSemicolon, querier_dto.SeverityError}},
		},
		{
			name: "trailing_semicolon_then_newline_emits_error",
			query: &querier_dto.AnalysedQuery{
				SQL:  "SELECT 1 FROM users;\n",
				Line: 1,
			},
			want: []expected{{querier_dto.CodeRuntimeBuilderTrailingSemicolon, querier_dto.SeverityError}},
		},
		{
			name: "trailing_semicolon_then_crlf_emits_error",
			query: &querier_dto.AnalysedQuery{
				SQL:  "SELECT 1 FROM users;\r\n",
				Line: 1,
			},
			want: []expected{{querier_dto.CodeRuntimeBuilderTrailingSemicolon, querier_dto.SeverityError}},
		},
		{
			name: "count_wrapped_emits_hint",
			query: &querier_dto.AnalysedQuery{
				SQL:             "SELECT DISTINCT category FROM posts",
				CountSQLWrapped: true,
				Line:            1,
			},
			want: []expected{{querier_dto.CodeCountSemanticsWrapped, querier_dto.SeverityHint}},
		},
		{
			name: "all_three_in_order",
			query: &querier_dto.AnalysedQuery{
				SQL:                     "SELECT DISTINCT category FROM posts WHERE id = $1;",
				BaseQueryHasWhereClause: true,
				CountSQLWrapped:         true,
				Line:                    1,
			},
			want: []expected{
				{querier_dto.CodeRuntimeBuilderBaseHasWhere, querier_dto.SeverityHint},
				{querier_dto.CodeRuntimeBuilderTrailingSemicolon, querier_dto.SeverityError},
				{querier_dto.CodeCountSemanticsWrapped, querier_dto.SeverityHint},
			},
		},
		{
			name: "trailing_whitespace_only_does_not_emit_terminator",
			query: &querier_dto.AnalysedQuery{
				SQL:  "SELECT 1 FROM users  \n\t",
				Line: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			context := &diagnosticContext{
				Filename: "test.sql",
				Query:    tt.query,
			}
			diagnostics := baseQueryShapeDiagnostics(context)
			require.Len(t, diagnostics, len(tt.want))
			for i, expect := range tt.want {
				assert.Equal(t, expect.code, diagnostics[i].Code, "diagnostic %d code", i)
				assert.Equal(t, expect.severity, diagnostics[i].Severity, "diagnostic %d severity", i)
				assert.Equal(t, "test.sql", diagnostics[i].Filename, "diagnostic %d filename", i)
				assert.Equal(t, 1, diagnostics[i].Line, "diagnostic %d line", i)
				assert.Equal(t, 1, diagnostics[i].Column, "diagnostic %d column", i)
			}
		})
	}
}

func TestEndsWithStatementTerminator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{input: ";", want: true},
		{input: "a;", want: true},
		{input: "a; ", want: true},
		{input: "a;\n", want: true},
		{input: "a;\r\n", want: true},
		{input: "a;\t", want: true},
		{input: ";;", want: true},
		{input: "a", want: false},
		{input: "", want: false},
		{input: "  ", want: false},
		{input: "a;b", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, endsWithStatementTerminator(tt.input))
		})
	}
}
