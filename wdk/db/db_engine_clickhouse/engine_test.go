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
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

func TestEngine_SatisfiesEnginePort(t *testing.T) {
	t.Parallel()

	var _ querier_domain.EnginePort = NewClickHouseEngine()
}

func TestEngine_Dialect(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	assert.Equal(t, "clickhouse", engine.Dialect())
}

func TestEngine_DialectNameOverride(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine(WithDialectName("clickhouse-cloud"))
	assert.Equal(t, "clickhouse-cloud", engine.Dialect())
}

func TestEngine_ParameterStyleIsCurly(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	assert.Equal(t, querier_dto.ParameterStyleClickHouseCurly, engine.ParameterStyle())
}

func TestEngine_SupportedDirectivePrefixIsBrace(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	prefixes := engine.SupportedDirectivePrefixes()
	require.Len(t, prefixes, 1)
	assert.Equal(t, byte('{'), prefixes[0].Prefix)
	assert.True(t, prefixes[0].IsNamed)
}

func TestEngine_DoesNotSupportReturning(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	assert.False(t, engine.SupportsReturning())
}

func TestEngine_DefaultSchemaIsDefault(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	assert.Equal(t, "default", engine.DefaultSchema())
}

func TestEngine_CommentStyleIsStandardSQL(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	assert.Equal(t, "--", engine.CommentStyle().LinePrefix)
}

func TestEngine_NormaliseTypeNameResolvesPrimitive(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	result := engine.NormaliseTypeName("UInt32")
	assert.Equal(t, querier_dto.TypeCategoryInteger, result.Category)
	assert.Equal(t, "UInt32", result.EngineName)
}

func TestEngine_NormaliseTypeNameUnknownFallsBack(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	result := engine.NormaliseTypeName("CompletelyMadeUpType")
	assert.Equal(t, querier_dto.TypeCategoryUnknown, result.Category)
	assert.Equal(t, "CompletelyMadeUpType", result.EngineName)
}

func TestEngine_NormaliseTypeNameHonoursHook(t *testing.T) {
	t.Parallel()

	hook := func(name string, modifiers []int) *querier_dto.SQLType {
		_ = modifiers
		if name == "UInt32" {
			return &querier_dto.SQLType{
				Category:   querier_dto.TypeCategoryInteger,
				EngineName: "UInt32",
			}
		}
		return nil
	}
	engine := NewClickHouseEngine(WithTypeNormaliserHook(hook))
	result := engine.NormaliseTypeName("UInt32")
	assert.Equal(t, querier_dto.TypeCategoryInteger, result.Category)
}

func TestEngine_BuiltinCataloguesAreNonNil(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	require.NotNil(t, engine.BuiltinFunctions())
	require.NotNil(t, engine.BuiltinTypes())
}

func TestEngine_WithExtraTypes(t *testing.T) {
	t.Parallel()

	extra := map[string]querier_dto.SQLType{
		"MyCustomType": {
			Category:   querier_dto.TypeCategoryText,
			EngineName: "MyCustomType",
		},
	}
	engine := NewClickHouseEngine(WithExtraTypes(extra))
	types := engine.BuiltinTypes()
	require.NotNil(t, types)
	entry, ok := types.Types["MyCustomType"]
	if !ok {
		entry, ok = types.Types["mycustomtype"]
	}
	require.True(t, ok, "extra type not registered in Types catalogue")
	assert.Equal(t, querier_dto.TypeCategoryText, entry.Category)
}

func TestEngine_WithExtraFunctions(t *testing.T) {
	t.Parallel()

	registerExtras := func(b *FunctionCatalogueBuilder) {
		b.Register("my_extra_func", b.uint64Type, b.uint64Type)
	}
	engine := NewClickHouseEngine(WithExtraFunctions(registerExtras))
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)
	overloads, ok := functions.Functions["my_extra_func"]
	require.True(t, ok, "extra function not registered (catalogue is case-insensitive via lowercase key)")
	require.Len(t, overloads, 1)
	assert.Equal(t, "my_extra_func", overloads[0].Name)
}

func TestEngine_WithImplicitCastHook(t *testing.T) {
	t.Parallel()

	hookCalled := false
	yes := true
	engine := NewClickHouseEngine(WithImplicitCastHook(func(from, to querier_dto.SQLTypeCategory) *bool {
		hookCalled = true
		_ = from
		_ = to
		return &yes
	}))
	result := engine.CanImplicitCast(querier_dto.TypeCategoryText, querier_dto.TypeCategoryInteger)
	assert.True(t, hookCalled, "implicit cast hook should be invoked")
	assert.True(t, result, "hook return must override default")
}

func TestEngine_WithPromoteTypeHook(t *testing.T) {
	t.Parallel()

	promoted := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "Float64"}
	hookCalled := false
	engine := NewClickHouseEngine(WithPromoteTypeHook(func(left, right querier_dto.SQLType) *querier_dto.SQLType {
		hookCalled = true
		_ = left
		_ = right
		return &promoted
	}))
	leftType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"}
	rightType := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "Float32"}
	result := engine.PromoteType(leftType, rightType)
	assert.True(t, hookCalled, "promote type hook should be invoked")
	assert.Equal(t, "Float64", result.EngineName, "hook return must override default")
}

func TestEngine_PromoteTypeMixedSignEqualWidthWidensToSigned(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	cases := []struct {
		left     string
		right    string
		expected string
	}{
		{left: "Int32", right: "UInt32", expected: "Int64"},
		{left: "UInt32", right: "Int32", expected: "Int64"},
		{left: "Int8", right: "UInt8", expected: "Int16"},
		{left: "UInt16", right: "Int16", expected: "Int32"},
	}
	for _, testCase := range cases {
		t.Run(testCase.left+"_"+testCase.right, func(subtest *testing.T) {
			subtest.Parallel()
			left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: testCase.left}
			right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: testCase.right}
			result := engine.PromoteType(left, right)
			assert.Equal(t, testCase.expected, result.EngineName,
				"mixed-sign equal-width integers must widen to the next wider signed type")
			assert.Equal(t, querier_dto.TypeCategoryInteger, result.Category)
		})
	}
}

func TestEngine_PromoteTypeMixedSign64BitKeepsLeftOperand(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}
	right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}
	result := engine.PromoteType(left, right)
	assert.Equal(t, "Int64", result.EngineName,
		"Int64 vs UInt64 is out of scope (stock ClickHouse throws); the left operand is preserved")
}

func TestEngine_PromoteTypeSameSignWiderRankWins(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int16"}
	right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"}
	result := engine.PromoteType(left, right)
	assert.Equal(t, "Int64", result.EngineName,
		"same-sign integers promote to the wider rank")
}

func TestBuiltinFunctions_NewlyRegisteredFunctions(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	expected := []struct {
		key             string
		minSignatures   int
		expectedAggrTag bool
	}{
		{key: "datetrunc", minSignatures: 2},
		{key: "lengthutf8", minSignatures: 1},
		{key: "hastoken", minSignatures: 1},
		{key: "hastokencaseinsensitive", minSignatures: 1},
		{key: "hastokenornull", minSignatures: 1},
		{key: "leftpad", minSignatures: 2},
		{key: "rightpad", minSignatures: 2},
		{key: "initcap", minSignatures: 1},
		{key: "ascii", minSignatures: 1},
		{key: "regexpextract", minSignatures: 2},
		{key: "tostartofquarter", minSignatures: 1},
		{key: "toisoweek", minSignatures: 1},
		{key: "monthname", minSignatures: 1},
		{key: "fromunixtimestamp64milli", minSignatures: 1},
		{key: "makedate", minSignatures: 1},
		{key: "makedatetime64", minSignatures: 1},
		{key: "tofloat64ornull", minSignatures: 1},
		{key: "toint8ornull", minSignatures: 1},
		{key: "toint64orzero", minSignatures: 1},
		{key: "touint64ordefault", minSignatures: 1},
		{key: "todecimal32", minSignatures: 1},
		{key: "todecimal256ornull", minSignatures: 1},
		{key: "todate32", minSignatures: 1},
		{key: "todateornull", minSignatures: 1},
		{key: "avgweighted", minSignatures: 1, expectedAggrTag: true},
		{key: "sumkahan", minSignatures: 1, expectedAggrTag: true},
		{key: "groupconcat", minSignatures: 2, expectedAggrTag: true},
		{key: "quantiletiming", minSignatures: 1, expectedAggrTag: true},
		{key: "histogram", minSignatures: 1, expectedAggrTag: true},
		{key: "intervallengthsum", minSignatures: 1, expectedAggrTag: true},
		{key: "approx_top_k", minSignatures: 1, expectedAggrTag: true},
		{key: "simplejsonextractstring", minSignatures: 1},
		{key: "simplejsonhas", minSignatures: 1},
		{key: "json_value", minSignatures: 1},
		{key: "clamp", minSignatures: 1},
		{key: "pi", minSignatures: 1},
		{key: "e", minSignatures: 1},
		{key: "sign", minSignatures: 1},
		{key: "factorial", minSignatures: 1},
		{key: "cbrt", minSignatures: 1},
		{key: "widthbucket", minSignatures: 1},
		{key: "generateuuidv7", minSignatures: 2},
		{key: "generatesnowflakeid", minSignatures: 2},
		{key: "snowflakeidtodatetime", minSignatures: 1},
	}

	for index := range expected {
		testCase := expected[index]
		t.Run(testCase.key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[testCase.key]
			require.True(testRunner, found, "function %q is missing from the catalogue", testCase.key)
			require.GreaterOrEqual(testRunner, len(signatures), testCase.minSignatures, "function %q has too few signatures", testCase.key)
			if testCase.expectedAggrTag {
				assert.True(testRunner, signatures[0].IsAggregate, "function %q should be tagged as an aggregate", testCase.key)
			}
		})
	}
}

func TestBuiltinFunctions_NoDuplicateArrayDistinctOrResize(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)
	arrayDistinctSignatures := functions.Functions["arraydistinct"]
	arrayResizeSignatures := functions.Functions["arrayresize"]
	assert.Len(t, arrayDistinctSignatures, 1, "arrayDistinct should have a single registration")
	assert.Len(t, arrayResizeSignatures, 1, "arrayResize should have a single registration")
}

func TestEngine_ParameterStyleIsClickHouseCurly(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	assert.Equal(t, querier_dto.ParameterStyleClickHouseCurly, engine.ParameterStyle())
}

func TestEngine_SupportsReturningIsFalse(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	assert.False(t, engine.SupportsReturning())
}

func TestEngine_SupportedExpressionsAdvertisesAllExpectedFlags(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	flags := engine.SupportedExpressions()
	required := []querier_dto.SQLExpressionFeature{
		querier_dto.SQLFeatureScalarSubquery,
		querier_dto.SQLFeatureWindowFunction,
		querier_dto.SQLFeatureArraySubscript,
		querier_dto.SQLFeatureJSONOp,
		querier_dto.SQLFeatureBitwiseOp,
		querier_dto.SQLFeatureLambda,
		querier_dto.SQLFeatureStructFieldAccess,
	}
	for _, flag := range required {
		assert.NotZero(t, flags&flag, "missing expected SQLExpressionFeature flag")
	}
}

func TestEngine_ParseStatementsRecordsPerStatementByteLength(t *testing.T) {
	t.Parallel()

	const first = "SELECT 1"
	const second = "SELECT 22"
	sql := first + "; " + second

	engine := NewClickHouseEngine()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 2)

	assert.Equal(t, 0, statements[0].Location)
	assert.Equal(t, len(first), statements[0].Length,
		"first statement Length should span only its own tokens, not the whole source")

	secondLocation := len(first) + len("; ")
	assert.Equal(t, secondLocation, statements[1].Location)
	assert.Equal(t, len(second), statements[1].Length,
		"second statement Length should span only its own tokens")
}

func TestBuiltinFunctions_NoCaseVariantDuplicateRegistrations(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	assert.Len(t, functions.Functions["cast"], 1,
		"CAST should have a single registration despite case-insensitive keys")
	assert.Len(t, functions.Functions["hostname"], 1,
		"hostName should have a single registration despite case-insensitive keys")
}

func TestWithMaxTokensPerStatement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "positive limit is applied", limit: 42, want: 42},
		{name: "zero keeps the default", limit: 0, want: defaultMaxTokensPerStatement},
		{name: "negative keeps the default", limit: -7, want: defaultMaxTokensPerStatement},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dialect := ClickHouseDialect{}
			WithMaxTokensPerStatement(testCase.limit)(&dialect)
			assert.Equal(t, testCase.want, dialect.resolvedMaxTokensPerStatement())
		})
	}
}

func TestWithMaxTypeParseDepth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		depth int
		want  int
	}{
		{name: "positive depth is applied", depth: 5, want: 5},
		{name: "zero keeps the default", depth: 0, want: defaultMaxTypeParseDepth},
		{name: "negative keeps the default", depth: -1, want: defaultMaxTypeParseDepth},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dialect := ClickHouseDialect{}
			WithMaxTypeParseDepth(testCase.depth)(&dialect)
			assert.Equal(t, testCase.want, dialect.resolvedMaxTypeParseDepth())
		})
	}
}

func TestTokenBudgetRejectsLongStatements(t *testing.T) {
	t.Parallel()

	const limit = 20
	longSelect := "SELECT " + strings.TrimSuffix(strings.Repeat("1, ", limit), ", ")
	testCases := []struct {
		name      string
		sql       string
		isDDL     bool
		wantError bool
	}{
		{name: "query over the budget", sql: longSelect, isDDL: false, wantError: true},
		{name: "query within the budget", sql: "SELECT 1, 2", isDDL: false, wantError: false},
		{name: "DDL over the budget", sql: "CREATE TABLE t (" + strings.TrimSuffix(strings.Repeat("c UInt8, ", limit), ", ") + ") ENGINE = Memory", isDDL: true, wantError: true},
		{name: "statement without a DDL handler is not walked", sql: "INSERT INTO t VALUES " + strings.TrimSuffix(strings.Repeat("(1), ", limit), ", "), isDDL: true, wantError: false},
		{name: "statement kind AnalyseQuery does not walk", sql: "SHOW TABLES " + strings.Repeat("x ", limit), isDDL: false, wantError: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engine := NewClickHouseEngine(WithMaxTokensPerStatement(limit))
			statements, err := engine.ParseStatements(testCase.sql)
			require.NoError(t, err)
			require.Len(t, statements, 1)

			if testCase.isDDL {
				_, err = engine.ApplyDDL(t.Context(), statements[0])
			} else {
				_, err = engine.AnalyseQuery(nil, statements[0])
			}
			if !testCase.wantError {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, errTokenBudgetExceeded)
			assert.Contains(t, err.Error(), "limit is 20")
		})
	}
}

func TestDefaultTokenBudgetAdmitsLargeInLists(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	sql := "SELECT 1 WHERE x IN (" + strings.TrimSuffix(strings.Repeat("1, ", 20_000), ", ") + ")"
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 1)

	_, err = engine.AnalyseQuery(nil, statements[0])
	require.NoError(t, err)
}

func TestTypeParseDepthOptionAppliesToStatementsAndNormalisation(t *testing.T) {
	t.Parallel()

	const nesting = 6
	deepType := strings.Repeat("Array(", nesting) + "UInt8" + strings.Repeat(")", nesting)

	t.Run("default depth accepts the nesting", func(t *testing.T) {
		t.Parallel()

		engine := NewClickHouseEngine()
		assert.Equal(t, querier_dto.TypeCategoryArray, engine.NormaliseTypeName(deepType).Category)
		mutation, err := applyDDLWith(t, engine, "CREATE TABLE t (c "+deepType+") ENGINE = Memory")
		require.NoError(t, err)
		require.Len(t, mutation.Columns, 1)
		assert.Equal(t, querier_dto.TypeCategoryArray, mutation.Columns[0].SQLType.Category)
	})

	t.Run("lowered depth rejects the nesting", func(t *testing.T) {
		t.Parallel()

		engine := NewClickHouseEngine(WithMaxTypeParseDepth(3))
		assert.Equal(t, querier_dto.TypeCategoryUnknown, engine.NormaliseTypeName(deepType).Category)
		_, err := applyDDLWith(t, engine, "CREATE TABLE t (c "+deepType+") ENGINE = Memory")
		require.ErrorIs(t, err, errTypeDepthExceeded)
	})
}

func applyDDLWith(t *testing.T, engine *ClickHouseEngine, sql string) (*querier_dto.CatalogueMutation, error) {
	t.Helper()
	statements, err := engine.ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, statements, 1)
	return engine.ApplyDDL(t.Context(), statements[0])
}

func TestApplyDDLAndAnalyseQueryRejectForeignStatements(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	statement := querier_dto.ParsedStatement{Raw: nil, Location: 0, Length: 0}

	_, err := engine.ApplyDDL(t.Context(), statement)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected statement type")

	_, err = engine.AnalyseQuery(nil, statement)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected statement type")
}

func TestApplyDDLHonoursCancellation(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	statements, err := engine.ParseStatements("CREATE TABLE t (id UInt64) ENGINE = Memory")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	cause := errors.New("catalogue build abandoned")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)

	mutation, err := engine.ApplyDDL(ctx, statements[0])
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, mutation)
}

func TestEntryPointsRecoverFromParserBugsWithoutLeakingStacks(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	statement := querier_dto.ParsedStatement{Raw: (*parsedStatement)(nil), Location: 0, Length: 0}

	mutation, err := engine.ApplyDDL(t.Context(), statement)
	require.Error(t, err)
	assert.Nil(t, mutation)
	assert.Contains(t, err.Error(), "clickhouse: ddl panic:")
	assert.NotContains(t, err.Error(), "goroutine")

	analysis, err := engine.AnalyseQuery(nil, statement)
	require.Error(t, err)
	assert.Nil(t, analysis)
	assert.Contains(t, err.Error(), "clickhouse: analyse panic:")
	assert.NotContains(t, err.Error(), "goroutine")
}
