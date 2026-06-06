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
	"testing"

	"piko.sh/piko/internal/querier/querier_dto"
)

func parseExpressionFromSource(t *testing.T, source string) *parsedExpression {
	t.Helper()
	tokens, err := tokenise(source)
	if err != nil {
		t.Fatalf("tokenise failed: %v", err)
	}
	parserInstance := newParser(tokens)
	expr := parserInstance.parseExpression()
	if expr == nil {
		t.Fatalf("parseExpression returned nil for: %s", source)
	}
	return expr
}

func TestParseExpressionRecognisesLiteralNumber(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "42")
	if expr.kind != expressionKindLiteral {
		t.Errorf("expected literal kind, got %v", expr.kind)
	}
}

func TestParseExpressionRecognisesLiteralString(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "'hello'")
	if expr.kind != expressionKindLiteral {
		t.Errorf("expected literal kind, got %v", expr.kind)
	}
}

func TestParseExpressionRecognisesLiteralNull(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "NULL")
	if expr.kind != expressionKindLiteral {
		t.Errorf("expected literal kind for NULL, got %v", expr.kind)
	}
}

func TestParseExpressionRecognisesBareColumn(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "user_id")
	if expr.kind != expressionKindColumn {
		t.Errorf("expected column kind, got %v", expr.kind)
	}
	if len(expr.referencedColumns) != 1 {
		t.Errorf("expected 1 column reference, got %d", len(expr.referencedColumns))
	}
	if len(expr.referencedColumns) > 0 && expr.referencedColumns[0].ColumnName != "user_id" {
		t.Errorf("expected column user_id, got %q", expr.referencedColumns[0].ColumnName)
	}
}

func TestParseExpressionRecognisesQualifiedColumn(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "users.id")
	if expr.kind != expressionKindColumn {
		t.Errorf("expected column kind, got %v", expr.kind)
	}
	if len(expr.referencedColumns) != 1 {
		t.Fatalf("expected 1 column reference, got %d", len(expr.referencedColumns))
	}
	if expr.referencedColumns[0].TableAlias != "users" {
		t.Errorf("expected alias users, got %q", expr.referencedColumns[0].TableAlias)
	}
	if expr.referencedColumns[0].ColumnName != "id" {
		t.Errorf("expected column id, got %q", expr.referencedColumns[0].ColumnName)
	}
}

func TestParseExpressionRecognisesParameter(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "{user_id:UInt64}")
	if !expr.hasParameter {
		t.Errorf("expected hasParameter flag to be set")
	}
}

func TestParseExpressionHandlesAdditive(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "a + b")
	if expr.kind != expressionKindBinary {
		t.Errorf("expected binary kind, got %v", expr.kind)
	}
	if len(expr.referencedColumns) != 2 {
		t.Errorf("expected 2 referenced columns for a + b, got %d", len(expr.referencedColumns))
	}
}

func TestParseExpressionHandlesMultiplicative(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "a * b + c")
	if expr.kind != expressionKindBinary {
		t.Errorf("expected binary kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesComparison(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "age > 21")
	if expr.kind != expressionKindBinary {
		t.Errorf("expected binary kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesAndOr(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "age > 21 AND status = 'active'")
	if expr.kind != expressionKindBinary {
		t.Errorf("expected binary kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesNot(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "NOT active")
	if expr.kind != expressionKindUnary {
		t.Errorf("expected unary kind for NOT, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesIsNull(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "name IS NULL")
	if expr.kind != expressionKindIsNull {
		t.Errorf("expected IsNull kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesIsNotNull(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "name IS NOT NULL")
	if expr.kind != expressionKindIsNull {
		t.Errorf("expected IsNull kind (IS NOT NULL is a variant), got %v", expr.kind)
	}
}

func TestParseExpressionHandlesBetween(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "age BETWEEN 18 AND 65")
	if expr.kind != expressionKindBetween {
		t.Errorf("expected Between kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesIn(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "status IN ('active', 'pending')")
	if expr.kind != expressionKindIn {
		t.Errorf("expected In kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesLike(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "name LIKE 'a%'")
	if expr.kind != expressionKindBinary {
		t.Errorf("expected binary kind for LIKE, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesCastDoubleColon(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "x::String")
	if expr.kind != expressionKindCast {
		t.Errorf("expected Cast kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesCastKeyword(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "CAST(x AS String)")
	if expr.kind != expressionKindCast {
		t.Errorf("expected Cast kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesArraySubscript(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "arr[1]")
	if expr.kind != expressionKindArraySubscript {
		t.Errorf("expected ArraySubscript kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesFunctionCall(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "count()")
	if expr.kind != expressionKindFunction {
		t.Errorf("expected Function kind, got %v", expr.kind)
	}
	if !expr.hasAggregate {
		t.Errorf("expected hasAggregate=true for count()")
	}
}

func TestParseExpressionHandlesFunctionCallWithArgs(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "concat(a, b, c)")
	if expr.kind != expressionKindFunction {
		t.Errorf("expected Function kind, got %v", expr.kind)
	}
	if len(expr.referencedColumns) != 3 {
		t.Errorf("expected 3 referenced columns, got %d", len(expr.referencedColumns))
	}
}

func TestParseExpressionHandlesNestedFunctionCalls(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "upper(concat(first_name, ' ', last_name))")
	if expr.kind != expressionKindFunction {
		t.Errorf("expected Function kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesCaseWhen(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "CASE WHEN x > 0 THEN 'positive' ELSE 'other' END")
	if expr.kind != expressionKindCase {
		t.Errorf("expected Case kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesLambda(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "x -> x + 1")
	if expr.kind != expressionKindLambda {
		t.Errorf("expected Lambda kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesMultiParamLambda(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "(x, y) -> x + y")
	if expr.kind != expressionKindLambda {
		t.Errorf("expected Lambda kind, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesParenthesised(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "(a + b) * c")
	if expr.kind != expressionKindBinary {
		t.Errorf("expected binary kind for outermost multiplication, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesUnaryMinus(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "-x")
	if expr.kind != expressionKindUnary {
		t.Errorf("expected unary kind for -x, got %v", expr.kind)
	}
}

func TestParseExpressionHandlesUnaryPlus(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "+x")

	_ = expr
}

func TestParseExpressionPropagatesParameterFlag(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "user_id = {user_id:UInt64}")
	if !expr.hasParameter {
		t.Errorf("expected hasParameter=true to propagate up through binary operator")
	}
}

func TestParseExpressionPropagatesColumnReferences(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "users.name = customers.name")
	if len(expr.referencedColumns) != 2 {
		t.Errorf("expected 2 column refs to propagate up, got %d", len(expr.referencedColumns))
	}
}

func TestParseExpressionHandlesArrayLiteral(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "[1, 2, 3]")

	_ = expr
}

func TestParseExpressionHandlesTupleAccess(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "tuple_col.1")

	if expr == nil {
		t.Fatal("expected non-nil expression for tuple.1")
	}
}

func TestParseExpressionHandlesIntervalLiteral(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "INTERVAL 1 DAY")
	_ = expr
}

func TestParseExpressionAggregateFlagSetForSum(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "sum(x)")
	if !expr.hasAggregate {
		t.Errorf("expected hasAggregate=true for sum(x)")
	}
}

func TestParseExpressionAggregateFlagSetForCountIf(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "countIf(status = 'active')")
	if !expr.hasAggregate {
		t.Errorf("expected hasAggregate=true for countIf")
	}
}

func TestParseExpressionAggregateFlagNotSetForNonAggregate(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "upper(name)")
	if expr.hasAggregate {
		t.Errorf("expected hasAggregate=false for upper(name)")
	}
}

func TestParseExpressionAggregateFlagPropagatesThroughExpression(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "sum(x) + 1")
	if !expr.hasAggregate {
		t.Errorf("expected hasAggregate to propagate through binary +")
	}
}

func TestParseExpressionLikeStringWithEscape(t *testing.T) {
	t.Parallel()

	expr := parseExpressionFromSource(t, "name LIKE '%test%'")
	if expr.kind != expressionKindBinary {
		t.Errorf("expected binary kind for LIKE, got %v", expr.kind)
	}
}

func TestParseExpressionILikeOperator(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "name ILIKE '%test%'")
	if expr.kind != expressionKindBinary {
		t.Errorf("expected binary kind for ILIKE, got %v", expr.kind)
	}
}

func TestParseExpressionNotInOperator(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "status NOT IN ('active')")
	if expr.kind != expressionKindIn {
		t.Errorf("expected In kind (NOT IN is a variant), got %v", expr.kind)
	}
}

func TestParseExpressionRecognisesExistsSubquery(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "EXISTS (SELECT 1 FROM users WHERE id = 1)")
	if expr == nil {
		t.Fatal("expected non-nil expression for EXISTS")
	}
	if expr.resultType.Category != querier_dto.TypeCategoryBoolean {
		t.Errorf("expected Bool result type for EXISTS, got %v", expr.resultType.Category)
	}
}

func TestParseExpressionExistsRegistersInnerParameters(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "EXISTS (SELECT 1 FROM users WHERE id = {uid:UInt32})")
	if !expr.hasParameter {
		t.Errorf("expected hasParameter=true for EXISTS body containing {uid:UInt32}")
	}
}

func TestParseExpressionTupleAccessOnNumericIndex(t *testing.T) {
	t.Parallel()
	expr := parseExpressionFromSource(t, "tuple_col.1")
	if expr == nil {
		t.Fatal("expected non-nil expression for tuple.1")
	}
}

func TestSubcolumnAccessSize0OnArrayReturnsUInt64(t *testing.T) {
	t.Parallel()
	element := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	base := querier_dto.SQLType{Category: querier_dto.TypeCategoryArray, EngineName: "Array", ElementType: &element}
	resolved, ok := subcolumnAccessType(base, token{kind: tokenIdentifier, value: "size0"})
	if !ok {
		t.Fatal("expected size0 subcolumn to resolve on Array(String)")
	}
	if resolved.Category != querier_dto.TypeCategoryInteger || resolved.EngineName != "UInt64" {
		t.Errorf("expected UInt64 for array.size0, got %v %q", resolved.Category, resolved.EngineName)
	}
}

func TestSubcolumnAccessNullOnNullableReturnsUInt8(t *testing.T) {
	t.Parallel()
	base := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String", Nullable: true}
	resolved, ok := subcolumnAccessType(base, token{kind: tokenIdentifier, value: "null"})
	if !ok {
		t.Fatal("expected null subcolumn to resolve on Nullable(String)")
	}
	if resolved.Category != querier_dto.TypeCategoryInteger || resolved.EngineName != "UInt8" {
		t.Errorf("expected UInt8 for nullable.null, got %v %q", resolved.Category, resolved.EngineName)
	}
}

func TestSubcolumnAccessMapKeysAndValues(t *testing.T) {
	t.Parallel()
	key := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	value := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"}
	base := querier_dto.SQLType{Category: querier_dto.TypeCategoryMap, EngineName: "Map", KeyType: &key, ElementType: &value}
	keys, ok := subcolumnAccessType(base, token{kind: tokenIdentifier, value: "keys"})
	if !ok {
		t.Fatal("expected keys subcolumn to resolve on Map")
	}
	if keys.Category != querier_dto.TypeCategoryArray || keys.ElementType == nil || keys.ElementType.EngineName != "String" {
		t.Errorf("expected Array(String) for map.keys, got %+v", keys)
	}
	values, ok := subcolumnAccessType(base, token{kind: tokenIdentifier, value: "values"})
	if !ok {
		t.Fatal("expected values subcolumn to resolve on Map")
	}
	if values.Category != querier_dto.TypeCategoryArray || values.ElementType == nil || values.ElementType.EngineName != "UInt32" {
		t.Errorf("expected Array(UInt32) for map.values, got %+v", values)
	}
}

func TestSubcolumnAccessMissesOnUnrelatedType(t *testing.T) {
	t.Parallel()
	base := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	_, ok := subcolumnAccessType(base, token{kind: tokenIdentifier, value: "size0"})
	if ok {
		t.Error("size0 should not resolve on non-Array base")
	}
}

func TestLiteralNumberType_PrefixedIntegersNotFloats(t *testing.T) {
	t.Parallel()
	cases := []struct {
		literal  string
		expected querier_dto.SQLTypeCategory
	}{

		{literal: "0xAE", expected: querier_dto.TypeCategoryInteger},
		{literal: "0xE0", expected: querier_dto.TypeCategoryInteger},
		{literal: "0xDEADBEEF", expected: querier_dto.TypeCategoryInteger},
		{literal: "0Xe", expected: querier_dto.TypeCategoryInteger},
		{literal: "0b1011", expected: querier_dto.TypeCategoryInteger},
		{literal: "0o17", expected: querier_dto.TypeCategoryInteger},
		{literal: "-0xFF", expected: querier_dto.TypeCategoryInteger},

		{literal: "42", expected: querier_dto.TypeCategoryInteger},
		{literal: "3.14", expected: querier_dto.TypeCategoryFloat},
		{literal: "1e10", expected: querier_dto.TypeCategoryFloat},
		{literal: "1.5E-3", expected: querier_dto.TypeCategoryFloat},
	}
	for _, testCase := range cases {
		t.Run(testCase.literal, func(subtest *testing.T) {
			subtest.Parallel()
			got := literalNumberType(testCase.literal)
			if got.Category != testCase.expected {
				subtest.Errorf("literalNumberType(%q).Category = %v, want %v",
					testCase.literal, got.Category, testCase.expected)
			}
		})
	}
}

func TestHasIntegerRadixPrefix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		literal  string
		expected bool
	}{
		{literal: "0x1F", expected: true},
		{literal: "0X1f", expected: true},
		{literal: "0b10", expected: true},
		{literal: "0B10", expected: true},
		{literal: "0o7", expected: true},
		{literal: "0O7", expected: true},
		{literal: "+0xFF", expected: true},
		{literal: "-0b1", expected: true},
		{literal: "0", expected: false},
		{literal: "00", expected: false},
		{literal: "42", expected: false},
		{literal: "1e10", expected: false},
		{literal: "", expected: false},
		{literal: "0.5", expected: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.literal, func(subtest *testing.T) {
			subtest.Parallel()
			if got := hasIntegerRadixPrefix(testCase.literal); got != testCase.expected {
				subtest.Errorf("hasIntegerRadixPrefix(%q) = %v, want %v",
					testCase.literal, got, testCase.expected)
			}
		})
	}
}
