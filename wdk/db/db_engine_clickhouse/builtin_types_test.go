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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestParseType_Primitives(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		want querier_dto.SQLTypeCategory
	}{
		{"UInt8", querier_dto.TypeCategoryInteger},
		{"UInt16", querier_dto.TypeCategoryInteger},
		{"UInt32", querier_dto.TypeCategoryInteger},
		{"UInt64", querier_dto.TypeCategoryInteger},
		{"UInt128", querier_dto.TypeCategoryInteger},
		{"UInt256", querier_dto.TypeCategoryInteger},
		{"Int8", querier_dto.TypeCategoryInteger},
		{"Int64", querier_dto.TypeCategoryInteger},
		{"Float32", querier_dto.TypeCategoryFloat},
		{"Float64", querier_dto.TypeCategoryFloat},
		{"BFloat16", querier_dto.TypeCategoryFloat},
		{"String", querier_dto.TypeCategoryText},
		{"Bool", querier_dto.TypeCategoryBoolean},
		{"Date", querier_dto.TypeCategoryTemporal},
		{"Date32", querier_dto.TypeCategoryTemporal},
		{"DateTime", querier_dto.TypeCategoryTemporal},
		{"UUID", querier_dto.TypeCategoryUUID},
		{"IPv4", querier_dto.TypeCategoryNetwork},
		{"IPv6", querier_dto.TypeCategoryNetwork},
		{"JSON", querier_dto.TypeCategoryJSON},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseClickHouseType(c.name)
			require.NoError(t, err)
			assert.Equal(t, c.want, result.SQLType.Category)
			assert.False(t, result.Nullable)
		})
	}
}

func TestParseType_NullableWrapper(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Nullable(UInt32)")
	require.NoError(t, err)
	assert.True(t, result.Nullable, "outer Nullable should set the flag")
	assert.Equal(t, querier_dto.TypeCategoryInteger, result.SQLType.Category)
}

func TestParseType_LowCardinalityWrapper(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("LowCardinality(String)")
	require.NoError(t, err)
	assert.True(t, result.LowCardinality)
	assert.Equal(t, querier_dto.TypeCategoryText, result.SQLType.Category)
}

func TestParseType_NestedNullableLowCardinality(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("LowCardinality(Nullable(String))")
	require.NoError(t, err)
	assert.True(t, result.LowCardinality)
	assert.True(t, result.Nullable)
	assert.Equal(t, querier_dto.TypeCategoryText, result.SQLType.Category)
}

func TestParseType_Array(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Array(String)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryArray, result.SQLType.Category)
	require.NotNil(t, result.SQLType.ElementType)
	assert.Equal(t, querier_dto.TypeCategoryText, result.SQLType.ElementType.Category)
}

func TestParseType_ArrayOfNullable(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Array(Nullable(Int32))")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryArray, result.SQLType.Category)
	require.NotNil(t, result.SQLType.ElementType)
	assert.Equal(t, querier_dto.TypeCategoryInteger, result.SQLType.ElementType.Category)
	assert.True(t, result.SQLType.ElementType.Nullable, "Array element should preserve inner Nullable flag")
	assert.False(t, result.Nullable, "outer Array, not Nullable")
}

func TestParseType_MapNullableValue(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Map(String, Nullable(Int32))")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryMap, result.SQLType.Category)
	require.NotNil(t, result.SQLType.KeyType)
	require.NotNil(t, result.SQLType.ElementType)
	assert.False(t, result.SQLType.KeyType.Nullable)
	assert.True(t, result.SQLType.ElementType.Nullable, "Map value should preserve inner Nullable flag")
}

func TestParseType_TupleWithNullableField(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Tuple(name String, count Nullable(UInt64))")
	require.NoError(t, err)
	require.Len(t, result.SQLType.StructFields, 2)
	assert.False(t, result.SQLType.StructFields[0].SQLType.Nullable)
	assert.True(t, result.SQLType.StructFields[1].SQLType.Nullable, "Tuple field should preserve inner Nullable flag")
}

func TestParseType_NestedWithNullableField(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Nested(name String, salary Nullable(Float64))")
	require.NoError(t, err)

	require.NotNil(t, result.SQLType.ElementType)
	require.Len(t, result.SQLType.ElementType.StructFields, 2)
	assert.True(t, result.SQLType.ElementType.StructFields[1].SQLType.Nullable)
}

func TestParseType_TupleAnonymous(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Tuple(String, UInt64)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryStruct, result.SQLType.Category)
	require.Len(t, result.SQLType.StructFields, 2)
	assert.Equal(t, "_1", result.SQLType.StructFields[0].Name)
	assert.Equal(t, querier_dto.TypeCategoryText, result.SQLType.StructFields[0].SQLType.Category)
	assert.Equal(t, "_2", result.SQLType.StructFields[1].Name)
}

func TestParseType_TupleNamed(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Tuple(name String, age UInt64)")
	require.NoError(t, err)
	require.Len(t, result.SQLType.StructFields, 2)
	assert.Equal(t, "name", result.SQLType.StructFields[0].Name)
	assert.Equal(t, "age", result.SQLType.StructFields[1].Name)
}

func TestParseType_Map(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Map(String, UInt32)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryMap, result.SQLType.Category)
	require.NotNil(t, result.SQLType.KeyType)
	require.NotNil(t, result.SQLType.ElementType)
	assert.Equal(t, querier_dto.TypeCategoryText, result.SQLType.KeyType.Category)
	assert.Equal(t, querier_dto.TypeCategoryInteger, result.SQLType.ElementType.Category)
}

func TestParseType_NestedDesugarsToArrayOfTuple(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Nested(name String, age UInt64)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryArray, result.SQLType.Category)
	require.NotNil(t, result.SQLType.ElementType)
	assert.Equal(t, querier_dto.TypeCategoryStruct, result.SQLType.ElementType.Category)
	require.Len(t, result.SQLType.ElementType.StructFields, 2)
	assert.Equal(t, "name", result.SQLType.ElementType.StructFields[0].Name)
}

func TestParseType_Enum(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Enum8('red' = 1, 'green' = 2, 'blue' = 3)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryEnum, result.SQLType.Category)
	assert.Equal(t, []string{"red", "green", "blue"}, result.SQLType.EnumValues)
}

func TestParseType_Enum16WithNegativeTag(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Enum16('low' = -1, 'high' = 1)")
	require.NoError(t, err)
	assert.Equal(t, []string{"low", "high"}, result.SQLType.EnumValues)
}

func TestParseType_FixedString(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("FixedString(16)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryText, result.SQLType.Category)
	require.NotNil(t, result.SQLType.Length)
	assert.Equal(t, 16, *result.SQLType.Length)
}

func TestParseType_Decimal(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Decimal(18, 4)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryDecimal, result.SQLType.Category)
	require.NotNil(t, result.SQLType.Precision)
	require.NotNil(t, result.SQLType.Scale)
	assert.Equal(t, 18, *result.SQLType.Precision)
	assert.Equal(t, 4, *result.SQLType.Scale)
}

func TestParseType_DecimalShortForms(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input             string
		expectedPrecision int
		expectedScale     int
	}{
		{"Decimal32(2)", 9, 2},
		{"Decimal64(4)", 18, 4},
		{"Decimal128(8)", 38, 8},
		{"Decimal256(16)", 76, 16},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			t.Parallel()
			result, err := parseClickHouseType(c.input)
			require.NoError(t, err)
			require.NotNil(t, result.SQLType.Precision)
			require.NotNil(t, result.SQLType.Scale)
			assert.Equal(t, c.expectedPrecision, *result.SQLType.Precision)
			assert.Equal(t, c.expectedScale, *result.SQLType.Scale)
		})
	}
}

func TestParseType_DateTime64(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("DateTime64(3)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryTemporal, result.SQLType.Category)
	require.NotNil(t, result.SQLType.Precision)
	assert.Equal(t, 3, *result.SQLType.Precision)
}

func TestParseType_DateTime64WithTimezone(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("DateTime64(6, 'UTC')")
	require.NoError(t, err)
	require.NotNil(t, result.SQLType.Precision)
	assert.Equal(t, 6, *result.SQLType.Precision)
}

func TestParseType_DateTimeWithTimezone(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("DateTime('UTC')")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryTemporal, result.SQLType.Category)
	assert.Equal(t, "DateTime", result.SQLType.EngineName)
}

func TestParseType_Variant(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Variant(String, UInt32, Float64)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryUnion, result.SQLType.Category)
	require.Len(t, result.SQLType.UnionMembers, 3)
}

func TestParseType_AggregateFunction(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("AggregateFunction(sum, UInt64)")
	require.NoError(t, err)

	assert.Equal(t, "AggregateFunction(sum, UInt64)", result.SQLType.EngineName)
	assert.Equal(t, querier_dto.TypeCategoryAggregateState, result.SQLType.Category)
	require.NotNil(t, result.SQLType.ElementType)
	assert.Equal(t, querier_dto.TypeCategoryInteger, result.SQLType.ElementType.Category)
}

func TestParseType_DeeplyNested(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Array(Tuple(name String, scores Array(Nullable(Float64))))")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryArray, result.SQLType.Category)
	require.NotNil(t, result.SQLType.ElementType)
	tuple := result.SQLType.ElementType
	assert.Equal(t, querier_dto.TypeCategoryStruct, tuple.Category)
	require.Len(t, tuple.StructFields, 2)
	scoresField := tuple.StructFields[1]
	assert.Equal(t, "scores", scoresField.Name)
	assert.Equal(t, querier_dto.TypeCategoryArray, scoresField.SQLType.Category)
}

func TestParseType_UnknownTypeReturnsUnknown(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("MagicalType")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryUnknown, result.SQLType.Category)
	assert.Equal(t, "MagicalType", result.SQLType.EngineName)
}

func TestParseType_MalformedReturnsError(t *testing.T) {
	t.Parallel()

	cases := []string{
		"Nullable(",
		"Nullable(UInt32",
		"Array",
		"Tuple(",
		"Map(",
		"Decimal(",
		"DateTime64",
		"DateTime64(",
		"Decimal(18 4)",
		"UInt32 trailing",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			t.Parallel()
			_, err := parseClickHouseType(c)
			assert.Error(t, err)
		})
	}
}

func TestEngine_NormaliseTypeNameStripsNullable(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	result := engine.NormaliseTypeName("Nullable(UInt32)")
	assert.Equal(t, querier_dto.TypeCategoryInteger, result.Category)
}

func TestEngine_NormaliseTypeNameHandlesArray(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	result := engine.NormaliseTypeName("Array(String)")
	assert.Equal(t, querier_dto.TypeCategoryArray, result.Category)
	require.NotNil(t, result.ElementType)
	assert.Equal(t, querier_dto.TypeCategoryText, result.ElementType.Category)
}

func TestBuildTypeCatalogue_IncludesExtras(t *testing.T) {
	t.Parallel()

	extras := map[string]querier_dto.SQLType{
		"MyCustomType": {Category: querier_dto.TypeCategoryComposite, EngineName: "MyCustomType"},
	}
	catalogue := buildTypeCatalogue(extras)
	require.NotNil(t, catalogue)
	_, hasCustom := catalogue.Types["mycustomtype"]
	assert.True(t, hasCustom)
	_, hasBuiltin := catalogue.Types["uint32"]
	assert.True(t, hasBuiltin)
}

func TestParseType_NewGeoTypes(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"LineString", "MultiLineString", "Geometry"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := parseClickHouseType(name)
			require.NoError(t, err)
			assert.Equal(t, querier_dto.TypeCategoryGeometric, result.SQLType.Category)
		})
	}
}

func TestParseType_IdentifierType(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Identifier")
	require.NoError(t, err)
	assert.Equal(t, "Identifier", result.SQLType.EngineName)
}

func TestParseType_EnumAutoTagShorthand(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Enum('a', 'b', 'c')")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryEnum, result.SQLType.Category)
	assert.Equal(t, []string{"a", "b", "c"}, result.SQLType.EnumValues)
	assert.Equal(t, "Enum8", result.SQLType.EngineName, "bare Enum should fold to Enum8")
}

func TestParseType_JSONWithParameters(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("JSON(max_dynamic_paths=100, path.to.field UInt32, SKIP other.path)")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryJSON, result.SQLType.Category)
	assert.Equal(t, "JSON", result.SQLType.EngineName)
}

func TestParseType_DynamicWithParameters(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Dynamic(max_types=10)")
	require.NoError(t, err)
	assert.Equal(t, "Dynamic", result.SQLType.EngineName)
}

func TestParseType_SimpleAggregateFunctionPreservesName(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("SimpleAggregateFunction(any, UInt64)")
	require.NoError(t, err)
	assert.Equal(t, "SimpleAggregateFunction(any, UInt64)", result.SQLType.EngineName)
	assert.Equal(t, querier_dto.TypeCategoryAggregateState, result.SQLType.Category)
}

func TestParseType_ObjectJSONAliasesToJSON(t *testing.T) {
	t.Parallel()

	result, err := parseClickHouseType("Object('json')")
	require.NoError(t, err)
	assert.Equal(t, querier_dto.TypeCategoryJSON, result.SQLType.Category)
	assert.Equal(t, "JSON", result.SQLType.EngineName)
}
