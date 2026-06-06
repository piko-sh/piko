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

package emitter_go_sql

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

func TestBuildValuesTuplePositionalProducesPlaceholderTuple(t *testing.T) {
	testCases := []struct {
		name     string
		count    int
		expected string
	}{
		{name: "single column", count: 1, expected: "(?)"},
		{name: "two columns", count: 2, expected: "(?,?)"},
		{name: "three columns", count: 3, expected: "(?,?,?)"},
		{name: "zero columns yields an empty tuple", count: 0, expected: "()"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, buildValuesTuple(testCase.count, false))
		})
	}
}

func TestBuildValuesTupleNumberedReturnsEmptyForRuntimeGeneration(t *testing.T) {

	assert.Empty(t, buildValuesTuple(3, true))
}

func TestConcatStringsJoinsExpressionsWithPlus(t *testing.T) {
	expr := concatStrings(goastutil.StrLit("a"), goastutil.StrLit("b"), goastutil.StrLit("c"))
	assert.Equal(t, `"a" + "b" + "c"`, renderExpr(t, expr))
}

func TestConcatStringsReturnsEmptyLiteralForNoParts(t *testing.T) {
	expr := concatStrings()
	assert.Equal(t, `""`, renderExpr(t, expr))
}

func TestChainCombinesExpressionsWithOperator(t *testing.T) {
	expr := chain(token.LAND, eqZero(goastutil.IntLit(1)), eqZero(goastutil.IntLit(2)))
	assert.Equal(t, "1 == 0 && 2 == 0", renderExpr(t, expr))
}

func TestChainReturnsNilForNoParts(t *testing.T) {
	assert.Nil(t, chain(token.LAND))
}

func TestBatchHelperSourceEmitsNumberedTupleHelperWhenRequested(t *testing.T) {
	withNumbered := batchHelperSource("db", true)
	withoutNumbered := batchHelperSource("db", false)

	assert.Contains(t, withNumbered, "func pikoBatchNumberedTuple(columns int, startAt int) string")
	assert.Contains(t, withNumbered, "func pikoBatchExpandValues(query string, multiValues string) string")
	assert.NotContains(t, withoutNumbered, "func pikoBatchNumberedTuple(")
	assert.Contains(t, withoutNumbered, "func pikoBatchExpandValues(query string, multiValues string) string")
}

func TestEmitPreparedExcludesCopyFromQueries(t *testing.T) {
	emitter := NewSQLEmitter()
	queries := []*querier_dto.AnalysedQuery{
		{
			Name:    "GetUserByID",
			Command: querier_dto.QueryCommandOne,
			SQL:     "SELECT id FROM users WHERE id = $1",
		},
		{
			Name:    "BulkInsertEvents",
			Command: querier_dto.QueryCommandCopyFrom,
			SQL:     "INSERT INTO events (name) VALUES (?)",
			Parameters: []querier_dto.QueryParameter{
				{Number: 1, Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			},
		},
	}

	file, err := emitter.EmitPrepared("mydb", queries)
	require.NoError(t, err)

	source := string(file.Content)

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "prepared.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated code must be valid Go:\n%s", source)

	assert.Contains(t, source, "getuserbyid")
	assert.NotContains(t, source, "bulkinsertevents")
}

func renderExpr(t *testing.T, expr ast.Expr) string {
	t.Helper()
	var builder strings.Builder
	fileSet := token.NewFileSet()
	require.NoError(t, printer.Fprint(&builder, fileSet, expr))
	return builder.String()
}
