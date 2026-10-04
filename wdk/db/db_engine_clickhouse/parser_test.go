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

package db_engine_clickhouse

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func tokeniseStatement(t *testing.T, sql string) []token {
	t.Helper()
	tokens, err := tokenise(sql)
	require.NoError(t, err)
	statements := splitStatements(tokens)
	require.Len(t, statements, 1)
	return statements[0]
}

func TestCollectParenthesisedSharesParserTokens(t *testing.T) {
	t.Parallel()

	p := newParser(tokeniseStatement(t, "(a, (b), c) d"))
	inner, err := p.collectParenthesised()
	require.NoError(t, err)

	require.Len(t, inner, 7)
	assert.Same(t, &p.tokens[1], &inner[0], "the group must be a view of the parser's tokens, not a copy")
	assert.Equal(t, len(inner), cap(inner), "capacity must be clipped so an append cannot overwrite later tokens")
	assert.Equal(t, "d", p.current().value)

	extended := append(inner, token{kind: tokenIdentifier, value: "x", position: 0})
	require.Len(t, extended, 8)
	assert.Equal(t, tokenRightParen, p.tokens[8].kind, "appending to the group must leave the parser's tokens intact")
}

func TestCollectParenthesisedErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr error
		name    string
		sql     string
	}{
		{name: "cursor not on an opening paren", sql: "a (b)", wantErr: nil},
		{name: "unterminated group", sql: "(a, (b)", wantErr: errUnmatchedParenthesis},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			p := newParser(tokeniseStatement(t, testCase.sql))
			inner, err := p.collectParenthesised()
			require.Error(t, err)
			assert.Nil(t, inner)
			if testCase.wantErr != nil {
				assert.ErrorIs(t, err, testCase.wantErr)
			}
		})
	}
}

func TestCollectInSubqueryBodyTokensSharesParserTokens(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		sql           string
		wantRemaining string
		wantBody      int
	}{
		{name: "balanced body", sql: "SELECT (1)) tail", wantBody: 4, wantRemaining: "tail"},
		{name: "unterminated body runs to the end", sql: "SELECT (1", wantBody: 3, wantRemaining: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			p := newParser(tokeniseStatement(t, testCase.sql))
			body := p.collectInSubqueryBodyTokens()

			require.Len(t, body, testCase.wantBody)
			assert.Same(t, &p.tokens[0], &body[0])
			assert.Equal(t, len(body), cap(body))
			assert.Equal(t, testCase.wantRemaining, p.current().value)
		})
	}
}

func TestDeeplyNestedDerivedTablesAllocateLinearly(t *testing.T) {
	const (
		nesting     = 200
		paddingRows = 5_000
		allowance   = 32 << 20
	)
	var builder strings.Builder
	builder.WriteString(strings.Repeat("SELECT * FROM (", nesting))
	builder.WriteString("SELECT 1 AS x WHERE x IN (")
	for index := range paddingRows {
		if index > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("1")
	}
	builder.WriteString(")")
	builder.WriteString(strings.Repeat(") AS t", nesting))

	engine := NewClickHouseEngine()
	statements, err := engine.ParseStatements(builder.String())
	require.NoError(t, err)
	require.Len(t, statements, 1)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	analysis, err := engine.AnalyseQuery(nil, statements[0])
	runtime.ReadMemStats(&after)

	require.NoError(t, err)
	require.NotNil(t, analysis)
	allocated := after.TotalAlloc - before.TotalAlloc
	assert.Less(t, allocated, uint64(allowance),
		"re-parsing each nesting level must not copy the remaining tokens")
}

func TestExpectKeywordSequence(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		sql          string
		wantErr      string
		keywords     []string
		wantPosition int
	}{
		{name: "all keywords present", sql: "DROP TABLE t", keywords: []string{"DROP", "TABLE"}, wantPosition: 2},
		{name: "case insensitive", sql: "drop table t", keywords: []string{"DROP", "TABLE"}, wantPosition: 2},
		{name: "second keyword missing", sql: "DROP VIEW t", keywords: []string{"DROP", "TABLE"}, wantErr: `expected keyword [TABLE], got "VIEW"`, wantPosition: 1},
		{name: "first keyword missing", sql: "SELECT 1", keywords: []string{"DROP"}, wantErr: `expected keyword [DROP], got "SELECT"`, wantPosition: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			p := newParser(tokeniseStatement(t, testCase.sql))
			err := p.expectKeywordSequence(testCase.keywords...)
			if testCase.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.wantErr)
			}
			assert.Equal(t, testCase.wantPosition, p.position)
		})
	}
}

func TestNewChildParserInheritsLimits(t *testing.T) {
	t.Parallel()

	parent := newParser(nil)
	parent.analysisDepth = 3
	parent.maxParseDepth = 17
	parent.maxTypeParseDepth = 9
	parent.syntaxError = errExpressionDepthExceeded

	child := parent.newChildParser([]token{{kind: tokenIdentifier, value: "x", position: 0}})

	assert.Equal(t, 3, child.analysisDepth)
	assert.Equal(t, 17, child.maxParseDepth)
	assert.Equal(t, 9, child.maxTypeParseDepth)
	assert.NoError(t, child.syntaxError, "a child starts without its parent's syntax error")
	assert.Equal(t, 0, child.position)
}

func TestRecordSyntaxErrorKeepsFirst(t *testing.T) {
	t.Parallel()

	first := errors.New("first")
	p := newParser(nil)
	p.recordSyntaxError(first)
	p.recordSyntaxError(errors.New("second"))

	assert.Same(t, first, p.syntaxError)
}

func TestAbsorbChildFailureRecordsOnlyDepthLimits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err       error
		name      string
		wantStick bool
	}{
		{name: "nil failure", err: nil, wantStick: false},
		{name: "ordinary syntax failure", err: errors.New("expected SELECT"), wantStick: false},
		{name: "analysis depth", err: errAnalysisDepthExceeded, wantStick: true},
		{name: "expression depth", err: errExpressionDepthExceeded, wantStick: true},
		{name: "type depth wrapped", err: errors.Join(errors.New("context"), errTypeDepthExceeded), wantStick: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			p := newParser(nil)
			p.absorbChildFailure(testCase.err)
			if testCase.wantStick {
				assert.ErrorIs(t, p.syntaxError, testCase.err)
				return
			}
			assert.NoError(t, p.syntaxError)
		})
	}
}

func TestEveryDDLHandlerRejectsAWrongLeadingKeyword(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	tokens := tokeniseStatement(t, "SELECT 1")
	for kind, handler := range ddlParserDispatch {
		if handler == nil {
			continue
		}
		statement := querier_dto.ParsedStatement{
			Raw:      &parsedStatement{tokens: tokens, kind: kind},
			Location: 0,
			Length:   len("SELECT 1"),
		}
		mutation, err := engine.ApplyDDL(t.Context(), statement)
		require.Error(t, err, "statement kind %d", kind)
		assert.Nil(t, mutation)
		assert.NotContains(t, err.Error(), "panic", "statement kind %d must fail as a syntax error", kind)
	}
}

func TestApplyDDLReportsSyntaxErrorsWithoutPanicking(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		sql     string
		wantErr string
	}{
		{name: "IF NOT EXISTS before FUNCTION", sql: "CREATE IF NOT EXISTS FUNCTION f AS (x) -> x", wantErr: `expected keyword [FUNCTION], got "IF"`},
		{name: "IF NOT EXISTS before VIEW", sql: "CREATE IF NOT EXISTS VIEW v AS SELECT 1", wantErr: `expected keyword [VIEW], got "IF"`},
		{name: "IF NOT EXISTS before DATABASE", sql: "CREATE IF NOT EXISTS DATABASE d", wantErr: `expected keyword [DATABASE], got "IF"`},
		{name: "IF NOT EXISTS before DICTIONARY", sql: "CREATE IF NOT EXISTS DICTIONARY d", wantErr: `expected keyword [DICTIONARY], got "IF"`},
		{name: "IF NOT EXISTS before MATERIALIZED VIEW", sql: "CREATE IF NOT EXISTS MATERIALIZED VIEW v AS SELECT 1", wantErr: `expected keyword [MATERIALIZED], got "IF"`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mutation, err := applyDDL(t, testCase.sql)
			require.Error(t, err)
			assert.Nil(t, mutation)
			assert.Contains(t, err.Error(), testCase.wantErr)
			assert.NotContains(t, err.Error(), "panic")
		})
	}
}
