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
	"testing"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestSplitCombinatorSuffixesRecognisesAllSuffixes(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		input        string
		expectedBase string
		expectedTail []string
	}{
		"countIf":              {input: "countIf", expectedBase: "count", expectedTail: []string{"If"}},
		"sumIf":                {input: "sumIf", expectedBase: "sum", expectedTail: []string{"If"}},
		"sumOrNull":            {input: "sumOrNull", expectedBase: "sum", expectedTail: []string{"OrNull"}},
		"sumOrDefault":         {input: "sumOrDefault", expectedBase: "sum", expectedTail: []string{"OrDefault"}},
		"sumArray":             {input: "sumArray", expectedBase: "sum", expectedTail: []string{"Array"}},
		"sumResample":          {input: "sumResample", expectedBase: "sum", expectedTail: []string{"Resample"}},
		"sumDistinct":          {input: "sumDistinct", expectedBase: "sum", expectedTail: []string{"Distinct"}},
		"sumMap":               {input: "sumMap", expectedBase: "sum", expectedTail: []string{"Map"}},
		"sumForEach":           {input: "sumForEach", expectedBase: "sum", expectedTail: []string{"ForEach"}},
		"sumState":             {input: "sumState", expectedBase: "sum", expectedTail: []string{"State"}},
		"sumMerge":             {input: "sumMerge", expectedBase: "sum", expectedTail: []string{"Merge"}},
		"countIfOrNull":        {input: "countIfOrNull", expectedBase: "count", expectedTail: []string{"If", "OrNull"}},
		"sumArrayOrDefault":    {input: "sumArrayOrDefault", expectedBase: "sum", expectedTail: []string{"Array", "OrDefault"}},
		"BareNameUntouched":    {input: "sum", expectedBase: "sum", expectedTail: nil},
		"EmptyName":            {input: "", expectedBase: "", expectedTail: nil},
		"UnknownSuffixNoSplit": {input: "sumUnknown", expectedBase: "sumUnknown", expectedTail: nil},
	}
	for testName, testCase := range cases {
		input := testCase.input
		expectedBase := testCase.expectedBase
		expectedTail := testCase.expectedTail
		t.Run(testName, func(testRunner *testing.T) {
			testRunner.Parallel()
			gotBase, gotTail := splitCombinatorSuffixes(input)
			if gotBase != expectedBase {
				testRunner.Errorf("base mismatch: want %q got %q", expectedBase, gotBase)
			}
			if len(gotTail) != len(expectedTail) {
				testRunner.Fatalf("tail length mismatch: want %v got %v", expectedTail, gotTail)
			}
			for index := range expectedTail {
				if gotTail[index] != expectedTail[index] {
					testRunner.Errorf("tail[%d] mismatch: want %q got %q", index, expectedTail[index], gotTail[index])
				}
			}
		})
	}
}

func TestResolveArrayMapReturnsArrayOfLambdaResult(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	argTypes := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"},
		{Category: querier_dto.TypeCategoryArray, EngineName: "Array", ElementType: &querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}},
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "arrayMap", "", argTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected a resolution for arrayMap")
	}
	if resolution.ReturnType.Category != querier_dto.TypeCategoryArray {
		t.Errorf("expected array return, got %v", resolution.ReturnType.Category)
	}
}

func TestResolveArrayFilterReturnsInputArrayType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	element := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}
	argTypes := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
		{Category: querier_dto.TypeCategoryArray, EngineName: "Array", ElementType: &element},
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "arrayFilter", "", argTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected a resolution for arrayFilter")
	}
	if resolution.ReturnType.Category != querier_dto.TypeCategoryArray {
		t.Errorf("expected array return, got %v", resolution.ReturnType.Category)
	}
}

func TestResolveArrayJoinReturnsElementType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	element := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	argTypes := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryArray, EngineName: "Array", ElementType: &element},
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "arrayJoin", "", argTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected a resolution for arrayJoin")
	}
	if resolution.ReturnType.Category != querier_dto.TypeCategoryText {
		t.Errorf("expected string return (element of Array(String)), got %v", resolution.ReturnType.Category)
	}
}

func TestResolveIfReturnsCommonType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	argTypes := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
		{Category: querier_dto.TypeCategoryText, EngineName: "String"},
		{Category: querier_dto.TypeCategoryText, EngineName: "String"},
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "if", "", argTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected a resolution for if(bool, String, String)")
	}
	if resolution.ReturnType.Category != querier_dto.TypeCategoryText {
		t.Errorf("expected String return type, got %v", resolution.ReturnType.Category)
	}
}

func TestResolveIfMixedSignEqualWidthWidensToSigned(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	cases := []struct {
		then     string
		other    string
		expected string
	}{
		{then: "Int32", other: "UInt32", expected: "Int64"},
		{then: "UInt32", other: "Int32", expected: "Int64"},
		{then: "Int8", other: "UInt8", expected: "Int16"},
		{then: "UInt16", other: "Int16", expected: "Int32"},
	}
	for _, testCase := range cases {
		t.Run(testCase.then+"_"+testCase.other, func(subtest *testing.T) {
			subtest.Parallel()
			argTypes := []querier_dto.SQLType{
				{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
				{Category: querier_dto.TypeCategoryInteger, EngineName: testCase.then},
				{Category: querier_dto.TypeCategoryInteger, EngineName: testCase.other},
			}
			resolution, err := resolver.ResolveFunctionCall(nil, "if", "", argTypes)
			if err != nil {
				subtest.Fatalf("unexpected error: %v", err)
			}
			if resolution == nil {
				subtest.Fatal("expected a resolution for if over mixed-sign integers")
			}
			if resolution.ReturnType.EngineName != testCase.expected {
				subtest.Errorf("expected %s for mixed-sign if, got %q",
					testCase.expected, resolution.ReturnType.EngineName)
			}
		})
	}
}

func TestResolveIfMixedSign64BitKeepsLeftOperand(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	argTypes := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
		{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"},
		{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "if", "", argTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected a resolution for if(bool, Int64, UInt64)")
	}
	if resolution.ReturnType.EngineName != "Int64" {
		t.Errorf("Int64 vs UInt64 is out of scope (stock ClickHouse throws); expected the left operand Int64, got %q",
			resolution.ReturnType.EngineName)
	}
}

func TestResolveCoalesceReturnsFirstNonNullType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	argTypes := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryText, EngineName: "String"},
		{Category: querier_dto.TypeCategoryText, EngineName: "String"},
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "coalesce", "", argTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected resolution for coalesce")
	}
	if resolution.ReturnType.Category != querier_dto.TypeCategoryText {
		t.Errorf("expected String, got %v", resolution.ReturnType.Category)
	}
}

func TestResolveIdentityReturnsFirstArgumentType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	argTypes := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"},
	}
	for _, name := range []string{"any", "anyLast", "anyHeavy", "min", "max"} {
		resolution, err := resolver.ResolveFunctionCall(nil, name, "", argTypes)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", name, err)
		}
		if resolution == nil {
			t.Errorf("expected resolution for %s but got nil", name)
			continue
		}
		if resolution.ReturnType.Category != querier_dto.TypeCategoryInteger {
			t.Errorf("expected integer for %s, got %v", name, resolution.ReturnType.Category)
		}
	}
}

func TestResolveArgMinMaxReturnsFirstArgumentType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	argTypes := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryText, EngineName: "String"},
		{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"},
	}
	for _, name := range []string{"argMin", "argMax"} {
		resolution, err := resolver.ResolveFunctionCall(nil, name, "", argTypes)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", name, err)
		}
		if resolution == nil {
			t.Errorf("expected resolution for %s, got nil", name)
			continue
		}
		if resolution.ReturnType.Category != querier_dto.TypeCategoryText {
			t.Errorf("expected first-arg type (String) for %s, got %v", name, resolution.ReturnType.Category)
		}
	}
}

func TestResolveTupleElementReturnsNthFieldType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	tupleType := querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "_1", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			{Name: "_2", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
		},
	}
	argTypes := []querier_dto.SQLType{
		tupleType,
		{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt8"},
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "tupleElement", "", argTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected resolution for tupleElement")
	}

}

func TestUnknownFunctionReturnsNilResolution(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	resolution, err := resolver.ResolveFunctionCall(nil, "definitelyNotAFunction", "", nil)
	if !errors.Is(err, errResolverNoOpinion) {
		t.Fatalf("expected errResolverNoOpinion, got %v", err)
	}
	if resolution != nil {
		t.Errorf("expected nil resolution, got %+v", resolution)
	}
}

func TestApplyCombinatorOrNullMarksNullable(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "OrNull")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.Category != querier_dto.TypeCategoryInteger {
		t.Errorf("OrNull should preserve the category, got %v", transformed.ReturnType.Category)
	}
	if !transformed.ReturnType.Nullable {
		t.Errorf("OrNull should mark the return type as Nullable")
	}
}

func TestApplyCombinatorArrayKeepsReturnType(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "Array")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.Category != querier_dto.TypeCategoryInteger {
		t.Errorf("Array combinator (aggregate-over-array) keeps the base return type, got %v", transformed.ReturnType.Category)
	}
}

func TestApplyCombinatorResampleWrapsReturnInArray(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "Resample")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.Category != querier_dto.TypeCategoryArray {
		t.Errorf("expected Resample to wrap return in Array, got %v", transformed.ReturnType.Category)
	}
}

func TestApplyCombinatorStateConvertsToAggregateFunction(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "State")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.EngineName != "AggregateFunction" {
		t.Errorf("expected EngineName = AggregateFunction, got %q", transformed.ReturnType.EngineName)
	}
}

func TestApplyCombinatorIfDoesNotChangeReturn(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "If")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.Category != querier_dto.TypeCategoryInteger {
		t.Errorf("If combinator should not change return type, got %v", transformed.ReturnType.Category)
	}
}

func TestApplyCombinatorOrDefaultPreservesReturn(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "OrDefault")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.Nullable {
		t.Errorf("OrDefault should keep result non-nullable (default fills the slot)")
	}
}

func TestApplyCombinatorDistinctPreservesReturn(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "Distinct")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.EngineName != "UInt64" {
		t.Errorf("Distinct should preserve the base return type, got %q", transformed.ReturnType.EngineName)
	}
}

func TestApplyCombinatorMergeUnwrapsAggregateState(t *testing.T) {
	t.Parallel()
	elementType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryAggregateState,
			EngineName:  "AggregateFunction",
			ElementType: &elementType,
		},
	}
	transformed := applyCombinator(base, "Merge")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.Category != querier_dto.TypeCategoryInteger {
		t.Errorf("Merge should finalise to the element category, got %v", transformed.ReturnType.Category)
	}
	if transformed.ReturnType.EngineName != "UInt64" {
		t.Errorf("Merge should finalise AggregateFunction(sum, UInt64) to UInt64, got %q",
			transformed.ReturnType.EngineName)
	}
}

func TestApplyCombinatorMergeLeavesNonStateUnchanged(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "Merge")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.EngineName != "UInt64" {
		t.Errorf("Merge over a non-state base should pass through, got %q", transformed.ReturnType.EngineName)
	}
}

func TestApplyCombinatorMergeStateMarksAggregateFunction(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "MergeState")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.EngineName != "AggregateFunction" {
		t.Errorf("MergeState should mark EngineName=AggregateFunction, got %q", transformed.ReturnType.EngineName)
	}
}

func TestApplyCombinatorSimpleStateMarksSimpleAggregateFunction(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "SimpleState")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.EngineName != "SimpleAggregateFunction" {
		t.Errorf("SimpleState should mark EngineName=SimpleAggregateFunction, got %q", transformed.ReturnType.EngineName)
	}
}

func TestApplyCombinatorForEachWrapsInArray(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "ForEach")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.Category != querier_dto.TypeCategoryArray {
		t.Errorf("ForEach should wrap the return in Array, got %v", transformed.ReturnType.Category)
	}
	if transformed.ReturnType.ElementType == nil || transformed.ReturnType.ElementType.EngineName != "UInt64" {
		t.Errorf("ForEach should preserve inner element type UInt64")
	}
}

func TestApplyCombinatorMapWrapsInMap(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "Map")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.Category != querier_dto.TypeCategoryMap {
		t.Errorf("Map should wrap the return in Map, got %v", transformed.ReturnType.Category)
	}
	if transformed.ReturnType.KeyType == nil || transformed.ReturnType.ElementType == nil {
		t.Fatal("Map should populate KeyType and ElementType")
	}
	if transformed.ReturnType.ElementType.EngineName != "UInt64" {
		t.Errorf("Map should preserve inner value type UInt64")
	}
}

func TestApplyCombinatorArgMinPreservesReturn(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "ArgMin")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.EngineName != "UInt64" {
		t.Errorf("ArgMin should preserve the base return type, got %q", transformed.ReturnType.EngineName)
	}
}

func TestApplyCombinatorArgMaxPreservesReturn(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	transformed := applyCombinator(base, "ArgMax")
	if transformed == nil {
		t.Fatal("expected non-nil resolution")
	}
	if transformed.ReturnType.EngineName != "UInt64" {
		t.Errorf("ArgMax should preserve the base return type, got %q", transformed.ReturnType.EngineName)
	}
}

func TestSplitCombinatorSuffixesRecognisesNewCombinators(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		input        string
		expectedBase string
		expectedTail []string
	}{
		"quantileMergeState": {input: "quantileMergeState", expectedBase: "quantile", expectedTail: []string{"MergeState"}},
		"sumSimpleState":     {input: "sumSimpleState", expectedBase: "sum", expectedTail: []string{"SimpleState"}},
		"sumArgMin":          {input: "sumArgMin", expectedBase: "sum", expectedTail: []string{"ArgMin"}},
		"sumArgMax":          {input: "sumArgMax", expectedBase: "sum", expectedTail: []string{"ArgMax"}},
		"argMaxIf":           {input: "argMaxIf", expectedBase: "argMax", expectedTail: []string{"If"}},
		"quantileExactIf":    {input: "quantileExactIf", expectedBase: "quantileExact", expectedTail: []string{"If"}},
	}
	for testName, testCase := range cases {
		input := testCase.input
		expectedBase := testCase.expectedBase
		expectedTail := testCase.expectedTail
		t.Run(testName, func(testRunner *testing.T) {
			testRunner.Parallel()
			gotBase, gotTail := splitCombinatorSuffixes(input)
			if gotBase != expectedBase {
				testRunner.Errorf("base mismatch: want %q got %q", expectedBase, gotBase)
			}
			if len(gotTail) != len(expectedTail) {
				testRunner.Fatalf("tail length mismatch: want %v got %v", expectedTail, gotTail)
			}
			for index := range expectedTail {
				if gotTail[index] != expectedTail[index] {
					testRunner.Errorf("tail[%d] mismatch: want %q got %q", index, expectedTail[index], gotTail[index])
				}
			}
		})
	}
}

func TestResolveCountIfReturnsUInt64(t *testing.T) {
	t.Parallel()

	resolver := NewClickHouseFunctionResolver()
	args := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"},
		{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
	}

	for _, name := range []string{"countIf", "count", "countDistinct"} {
		resolution, err := resolver.ResolveFunctionCall(nil, name, "", args)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", name, err)
		}
		if resolution.ReturnType.EngineName != "UInt64" {
			t.Fatalf("expected UInt64 return for %s, got %q", name, resolution.ReturnType.EngineName)
		}
		if resolution.ReturnType.Category != querier_dto.TypeCategoryInteger {
			t.Fatalf("expected integer category for %s, got %v", name, resolution.ReturnType.Category)
		}
	}
}

func TestResolveSumIfReturnsInt64(t *testing.T) {
	t.Parallel()

	resolver := NewClickHouseFunctionResolver()
	args := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryInteger, EngineName: "Int32"},
		{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "sumIf", "", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected resolution for sumIf")
	}
	if resolution.ReturnType.EngineName != "Int64" {
		t.Errorf("expected Int64 return for sumIf over Int32, got %q", resolution.ReturnType.EngineName)
	}
}

func TestResolveUniqIfNoOpinion(t *testing.T) {
	t.Parallel()

	resolver := NewClickHouseFunctionResolver()
	args := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryText, EngineName: "String"},
		{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
	}
	_, err := resolver.ResolveFunctionCall(nil, "uniqIf", "", args)
	if !errors.Is(err, errResolverNoOpinion) {
		t.Fatalf("expected errResolverNoOpinion for uniqIf, got %v", err)
	}
}

func TestResolveArgMaxIfReturnsFirstArgType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	args := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryText, EngineName: "String"},
		{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"},
		{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "argMaxIf", "", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected resolution for argMaxIf")
	}
	if resolution.ReturnType.Category != querier_dto.TypeCategoryText {
		t.Errorf("expected first-arg type (String) for argMaxIf, got %v", resolution.ReturnType.Category)
	}
}

func TestResolveCatalogueAggregatesYieldNoOpinion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []querier_dto.SQLType
	}{
		{
			name: "groupArrayIf",
			args: []querier_dto.SQLType{
				{Category: querier_dto.TypeCategoryText, EngineName: "String"},
				{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
			},
		},
		{
			name: "quantileIf",
			args: []querier_dto.SQLType{
				{Category: querier_dto.TypeCategoryFloat, EngineName: "Float64"},
				{Category: querier_dto.TypeCategoryBoolean, EngineName: "Bool"},
			},
		},
		{
			name: "uniqState",
			args: []querier_dto.SQLType{
				{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"},
			},
		},
		{
			name: "quantileMerge",
			args: []querier_dto.SQLType{
				{Category: querier_dto.TypeCategoryFloat, EngineName: "Float64"},
			},
		},
	}
	for _, testCase := range cases {
		name := testCase.name
		args := testCase.args
		t.Run(name, func(testRunner *testing.T) {
			testRunner.Parallel()
			resolver := NewClickHouseFunctionResolver()
			_, err := resolver.ResolveFunctionCall(nil, name, "", args)
			if !errors.Is(err, errResolverNoOpinion) {
				testRunner.Fatalf("expected errResolverNoOpinion for %s, got %v", name, err)
			}
		})
	}
}

func TestApplyCombinatorChainCountIfOrNull(t *testing.T) {
	t.Parallel()
	base := &querier_dto.FunctionResolution{
		ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
	}
	step1 := applyCombinator(base, "If")
	step2 := applyCombinator(step1, "OrNull")
	if step2.ReturnType.Category != querier_dto.TypeCategoryInteger {
		t.Errorf("expected integer category after If+OrNull, got %v", step2.ReturnType.Category)
	}
	if !step2.ReturnType.Nullable {
		t.Errorf("expected Nullable=true after OrNull")
	}
}

func TestResolveMapContainsReturnsBool(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	key := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	value := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}
	mapArg := querier_dto.SQLType{Category: querier_dto.TypeCategoryMap, EngineName: "Map", KeyType: &key, ElementType: &value}
	args := []querier_dto.SQLType{mapArg, key}
	resolution, err := resolver.ResolveFunctionCall(nil, "mapContains", "", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil || resolution.ReturnType.Category != querier_dto.TypeCategoryBoolean {
		t.Errorf("expected Bool resolution for mapContains, got %+v", resolution)
	}
}

func TestResolveMapAddReturnsInputMap(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	key := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	value := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}
	mapArg := querier_dto.SQLType{Category: querier_dto.TypeCategoryMap, EngineName: "Map", KeyType: &key, ElementType: &value}
	resolution, err := resolver.ResolveFunctionCall(nil, "mapAdd", "", []querier_dto.SQLType{mapArg, mapArg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil || resolution.ReturnType.Category != querier_dto.TypeCategoryMap {
		t.Errorf("expected Map resolution for mapAdd, got %+v", resolution)
	}
}

func TestResolveVariantTypeReturnsString(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	variant := querier_dto.SQLType{Category: querier_dto.TypeCategoryUnion, EngineName: "Variant"}
	resolution, err := resolver.ResolveFunctionCall(nil, "variantType", "", []querier_dto.SQLType{variant})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil || resolution.ReturnType.Category != querier_dto.TypeCategoryText {
		t.Errorf("expected String resolution for variantType, got %+v", resolution)
	}
}

func TestResolveDynamicTypeReturnsString(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	dyn := querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "Dynamic"}
	resolution, err := resolver.ResolveFunctionCall(nil, "dynamicType", "", []querier_dto.SQLType{dyn})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil || resolution.ReturnType.Category != querier_dto.TypeCategoryText {
		t.Errorf("expected String resolution for dynamicType, got %+v", resolution)
	}
}

func TestResolveVariantElementReturnsMemberType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	memberType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}
	variant := querier_dto.SQLType{
		Category:     querier_dto.TypeCategoryUnion,
		EngineName:   "Variant",
		UnionMembers: []querier_dto.UnionMember{{SQLType: memberType}},
	}
	literalType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	resolution, err := resolver.ResolveFunctionCall(nil, "variantElement", "", []querier_dto.SQLType{variant, literalType})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected resolution for variantElement")
	}
	if !resolution.ReturnType.Nullable {
		t.Error("expected variantElement to return nullable type")
	}
}

func TestResolveFinalizeAggregationUnwrapsElementType(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	inner := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}
	state := querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryAggregateState,
		EngineName:  "AggregateFunction",
		ElementType: &inner,
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "finalizeAggregation", "", []querier_dto.SQLType{state})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil || resolution.ReturnType.EngineName != "UInt64" {
		t.Errorf("expected UInt64 unwrapped type for finalizeAggregation, got %+v", resolution)
	}
}

func TestResolveSumMergeFinalisesAggregateState(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	inner := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"}
	state := querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryAggregateState,
		EngineName:  "AggregateFunction",
		ElementType: &inner,
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "sumMerge", "", []querier_dto.SQLType{state})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected resolution for sumMerge over an aggregate state")
	}
	if resolution.ReturnType.Category != querier_dto.TypeCategoryInteger {
		t.Errorf("expected integer category for sumMerge, got %v", resolution.ReturnType.Category)
	}
	if resolution.ReturnType.EngineName != "UInt64" {
		t.Errorf("expected sumMerge(AggregateFunction(sum, UInt64)) to finalise to UInt64, got %q",
			resolution.ReturnType.EngineName)
	}
}

func TestResolveMaxMergeFinalisesAggregateState(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	inner := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	state := querier_dto.SQLType{
		Category:    querier_dto.TypeCategoryAggregateState,
		EngineName:  "AggregateFunction",
		ElementType: &inner,
	}
	resolution, err := resolver.ResolveFunctionCall(nil, "maxMerge", "", []querier_dto.SQLType{state})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil {
		t.Fatal("expected resolution for maxMerge over an aggregate state")
	}
	if resolution.ReturnType.EngineName != "String" {
		t.Errorf("expected maxMerge(AggregateFunction(max, String)) to finalise to String, got %q",
			resolution.ReturnType.EngineName)
	}
}

func TestResolveInitializeAggregationWrapsValueInState(t *testing.T) {
	t.Parallel()
	resolver := NewClickHouseFunctionResolver()
	aggName := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "String"}
	value := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt32"}
	resolution, err := resolver.ResolveFunctionCall(nil, "initializeAggregation", "", []querier_dto.SQLType{aggName, value})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution == nil || resolution.ReturnType.EngineName != "AggregateFunction" {
		t.Errorf("expected AggregateFunction wrap, got %+v", resolution)
	}
}

func TestIsStandaloneFunctionRecognisesWindowAliases(t *testing.T) {
	t.Parallel()
	names := []string{"lagInFrame", "leadInFrame", "denseRank", "percentRank", "nth_value", "percent_rank", "cume_dist"}
	for _, name := range names {
		if !isStandaloneFunctionName(name) {
			t.Errorf("expected %q to be standalone", name)
		}
	}
}

func TestIsStandaloneFunctionRecognisesVariantHelpers(t *testing.T) {
	t.Parallel()
	names := []string{"variantElement", "variantType", "dynamicElement", "dynamicType"}
	for _, name := range names {
		if !isStandaloneFunctionName(name) {
			t.Errorf("expected %q to be standalone", name)
		}
	}
}
