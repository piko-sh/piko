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

package db_engine_duckdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestNewDuckDBEngine(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	assert.Equal(t, "duckdb", engine.Dialect(), "default dialect should be duckdb")
	assert.Equal(t, "main", engine.DefaultSchema(), "DuckDB default schema should be main")
	assert.Equal(t, querier_dto.ParameterStyleDollar, engine.ParameterStyle(), "DuckDB uses dollar-sign parameters")
	assert.True(t, engine.SupportsReturning(), "DuckDB supports RETURNING clauses")
	assert.NotNil(t, engine.BuiltinFunctions(), "function catalogue should be initialised")
	assert.NotNil(t, engine.BuiltinTypes(), "type catalogue should be initialised")
}

func TestNormaliseTypeName(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	tests := []struct {
		name           string
		input          string
		wantEngineName string
		wantCategory   querier_dto.SQLTypeCategory
	}{

		{
			name:           "int normalises to int4",
			input:          "int",
			wantEngineName: "int4",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "integer normalises to int4",
			input:          "integer",
			wantEngineName: "int4",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "bigint normalises to int8",
			input:          "bigint",
			wantEngineName: "int8",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "smallint normalises to int2",
			input:          "smallint",
			wantEngineName: "int2",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "tinyint normalises to int1",
			input:          "tinyint",
			wantEngineName: "int1",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},

		{
			name:           "hugeint normalises to hugeint",
			input:          "hugeint",
			wantEngineName: "hugeint",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "uinteger normalises to uinteger",
			input:          "uinteger",
			wantEngineName: "uinteger",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "ubigint normalises to ubigint",
			input:          "ubigint",
			wantEngineName: "ubigint",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "utinyint normalises to utinyint",
			input:          "utinyint",
			wantEngineName: "utinyint",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "uhugeint normalises to uhugeint",
			input:          "uhugeint",
			wantEngineName: "uhugeint",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},

		{
			name:           "real normalises to float4",
			input:          "real",
			wantEngineName: "float4",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "double normalises to float8",
			input:          "double",
			wantEngineName: "float8",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "float normalises to float8",
			input:          "float",
			wantEngineName: "float8",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},

		{
			name:           "varchar normalises to varchar",
			input:          "varchar",
			wantEngineName: "varchar",
			wantCategory:   querier_dto.TypeCategoryText,
		},
		{
			name:           "text normalises to varchar",
			input:          "text",
			wantEngineName: "varchar",
			wantCategory:   querier_dto.TypeCategoryText,
		},

		{
			name:           "boolean normalises to bool",
			input:          "boolean",
			wantEngineName: "bool",
			wantCategory:   querier_dto.TypeCategoryBoolean,
		},
		{
			name:           "bool normalises to bool",
			input:          "bool",
			wantEngineName: "bool",
			wantCategory:   querier_dto.TypeCategoryBoolean,
		},

		{
			name:           "numeric normalises to numeric",
			input:          "numeric",
			wantEngineName: "numeric",
			wantCategory:   querier_dto.TypeCategoryDecimal,
		},
		{
			name:           "decimal normalises to numeric",
			input:          "decimal",
			wantEngineName: "numeric",
			wantCategory:   querier_dto.TypeCategoryDecimal,
		},

		{
			name:           "timestamp normalises to timestamp",
			input:          "timestamp",
			wantEngineName: "timestamp",
			wantCategory:   querier_dto.TypeCategoryTemporal,
		},
		{
			name:           "date normalises to date",
			input:          "date",
			wantEngineName: "date",
			wantCategory:   querier_dto.TypeCategoryTemporal,
		},

		{
			name:           "json normalises to json",
			input:          "json",
			wantEngineName: "json",
			wantCategory:   querier_dto.TypeCategoryJSON,
		},

		{
			name:           "uuid normalises to uuid",
			input:          "uuid",
			wantEngineName: "uuid",
			wantCategory:   querier_dto.TypeCategoryUUID,
		},

		{
			name:           "case insensitive normalisation",
			input:          "VARCHAR",
			wantEngineName: "varchar",
			wantCategory:   querier_dto.TypeCategoryText,
		},

		{
			name:           "unknown type falls back to unknown category",
			input:          "nonexistent_type",
			wantEngineName: "nonexistent_type",
			wantCategory:   querier_dto.TypeCategoryUnknown,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := engine.NormaliseTypeName(testCase.input)

			assert.Equal(t, testCase.wantEngineName, result.EngineName, "engine name mismatch")
			assert.Equal(t, testCase.wantCategory, result.Category, "category mismatch")
		})
	}
}

func TestPromoteType(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	t.Run("integer promotion selects the wider type", func(t *testing.T) {
		t.Parallel()

		int1 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int1"}
		integer8 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"}

		result := engine.PromoteType(int1, integer8)
		assert.Equal(t, "int8", result.EngineName, "int8 should win over int1")

		result = engine.PromoteType(integer8, int1)
		assert.Equal(t, "int8", result.EngineName, "int8 should still win when on the left")
	})

	t.Run("float promotion selects the wider type", func(t *testing.T) {
		t.Parallel()

		float4 := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float4"}
		float8 := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}

		result := engine.PromoteType(float4, float8)
		assert.Equal(t, "float8", result.EngineName, "float8 should win over float4")
	})

	t.Run("hugeint promotion over int4", func(t *testing.T) {
		t.Parallel()

		int4 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
		hugeint := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "hugeint"}

		result := engine.PromoteType(int4, hugeint)
		assert.Equal(t, "hugeint", result.EngineName, "hugeint should win over int4")
	})

	t.Run("same type returns identity", func(t *testing.T) {
		t.Parallel()

		int4 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}

		result := engine.PromoteType(int4, int4)
		assert.Equal(t, "int4", result.EngineName, "same type should return itself")
	})
}

func TestCanImplicitCast(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	t.Run("integer to float is allowed", func(t *testing.T) {
		t.Parallel()

		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryFloat))
	})

	t.Run("integer to decimal is allowed", func(t *testing.T) {
		t.Parallel()

		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryDecimal))
	})

	t.Run("float to integer is not allowed", func(t *testing.T) {
		t.Parallel()

		assert.False(t, engine.CanImplicitCast(querier_dto.TypeCategoryFloat, querier_dto.TypeCategoryInteger))
	})

	t.Run("text to text is allowed", func(t *testing.T) {
		t.Parallel()

		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryText, querier_dto.TypeCategoryText))
	})

	t.Run("text to integer is not allowed", func(t *testing.T) {
		t.Parallel()

		assert.False(t, engine.CanImplicitCast(querier_dto.TypeCategoryText, querier_dto.TypeCategoryInteger))
	})
}

func TestSupportedExpressions(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()
	features := engine.SupportedExpressions()

	assert.NotZero(t, features&querier_dto.SQLFeatureLambda,
		"DuckDB should support lambda expressions")

	assert.NotZero(t, features&querier_dto.SQLFeatureStructFieldAccess,
		"DuckDB should support struct field access")

	assert.NotZero(t, features&querier_dto.SQLFeatureStringConcat,
		"DuckDB should support string concatenation")

	assert.NotZero(t, features&querier_dto.SQLFeatureWindowFunction,
		"DuckDB should support window functions")

	assert.NotZero(t, features&querier_dto.SQLFeatureJSONOp,
		"DuckDB should support JSON operators")
}

func TestDefaultSchema(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	assert.Equal(t, "main", engine.DefaultSchema(),
		"DuckDB default schema should be main")
}

func TestBuiltinFunctions(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()
	catalogue := engine.BuiltinFunctions()

	require.NotNil(t, catalogue, "function catalogue must not be nil")
	require.NotNil(t, catalogue.Functions, "function map must not be nil")

	expectedFunctions := []string{"concat", "count", "abs"}
	for _, name := range expectedFunctions {
		signatures, exists := catalogue.Functions[name]
		assert.True(t, exists, "expected built-in function %q to be registered", name)
		assert.NotEmpty(t, signatures, "expected at least one signature for %q", name)
	}
}

func TestBuiltinTypes(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()
	catalogue := engine.BuiltinTypes()

	require.NotNil(t, catalogue, "type catalogue must not be nil")
	require.NotNil(t, catalogue.Types, "type map must not be nil")

	expectedTypes := []string{"int", "bigint", "varchar", "boolean", "json", "hugeint", "uinteger", "uuid", "timestamp"}
	for _, name := range expectedTypes {
		_, exists := catalogue.Types[name]
		assert.True(t, exists, "expected built-in type %q to be registered", name)
	}
}

func TestDuckDB_Config(t *testing.T) {
	t.Parallel()

	config := DuckDB()

	assert.Equal(t, "duckdb", config.DriverName, "driver name should be duckdb")
	assert.NotNil(t, config.Engine, "engine should not be nil")
}

func TestIsParsedStatement(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()
	statements, err := engine.ParseStatements("SELECT 1")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	parsed, ok := statements[0].Raw.(*parsedStatement)
	require.True(t, ok, "raw should be a *parsedStatement")
	parsed.IsParsedStatement()
}

func TestWithDialectName(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine(WithDialectName("motherduck"))

	assert.Equal(t, "duckdb", engine.Dialect())
}

func TestWithExtraTypes(t *testing.T) {
	t.Parallel()

	customTypes := map[string]querier_dto.SQLType{
		"custom_geo": {Category: querier_dto.TypeCategoryUnknown, EngineName: "custom_geo"},
	}
	engine := NewDuckDBEngine(WithExtraTypes(customTypes))

	catalogue := engine.BuiltinTypes()
	_, exists := catalogue.Types["custom_geo"]
	assert.True(t, exists, "custom type should be merged into catalogue")
}

func TestWithExtraFunctions(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine(WithExtraFunctions(func(builder *FunctionCatalogueBuilder) {
		builder.NullOnNull("my_custom_fn", nil, querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"})
	}))

	catalogue := engine.BuiltinFunctions()
	_, exists := catalogue.Functions["my_custom_fn"]
	assert.True(t, exists, "custom function should be registered")
}

func TestWithTypeNormaliserHook(t *testing.T) {
	t.Parallel()

	hook := func(name string, _ []int) *querier_dto.SQLType {
		if name == "my_special_type" {
			return &querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "my_special_type"}
		}
		return nil
	}
	engine := NewDuckDBEngine(WithTypeNormaliserHook(hook))

	result := engine.NormaliseTypeName("my_special_type")
	assert.Equal(t, "my_special_type", result.EngineName)
	assert.Equal(t, querier_dto.TypeCategoryText, result.Category)

	standard := engine.NormaliseTypeName("integer")
	assert.Equal(t, "int4", standard.EngineName)
}

func TestWithImplicitCastHook(t *testing.T) {
	t.Parallel()

	hook := func(from, to querier_dto.SQLTypeCategory) *bool {

		if from == querier_dto.TypeCategoryText && to == querier_dto.TypeCategoryInteger {
			return new(true)
		}
		return nil
	}
	engine := NewDuckDBEngine(WithImplicitCastHook(hook))

	assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryText, querier_dto.TypeCategoryInteger),
		"hook should allow text to integer cast")

	assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryFloat))
}

func TestWithPromoteTypeHook(t *testing.T) {
	t.Parallel()

	hook := func(left, right querier_dto.SQLType) *querier_dto.SQLType {
		if left.EngineName == "int4" && right.EngineName == "int4" {
			result := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"}
			return &result
		}
		return nil
	}
	engine := NewDuckDBEngine(WithPromoteTypeHook(hook))

	int4 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
	result := engine.PromoteType(int4, int4)
	assert.Equal(t, "int8", result.EngineName, "hook should promote int4+int4 to int8")
}

func TestTableValuedFunctionColumnsFromCatalogue(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	t.Run("resolves composite type columns", func(t *testing.T) {
		t.Parallel()

		catalogue := &querier_dto.Catalogue{
			DefaultSchema: "main",
			Schemas: map[string]*querier_dto.Schema{
				"main": {
					Name: "main",
					Functions: map[string][]*querier_dto.FunctionSignature{
						"get_records": {
							{
								Name:       "get_records",
								ReturnsSet: true,
								ReturnType: querier_dto.SQLType{
									Category:   querier_dto.TypeCategoryUnknown,
									EngineName: "point",
								},
							},
						},
					},
					CompositeTypes: map[string]*querier_dto.CompositeType{
						"point": {
							Name: "point",
							Fields: []querier_dto.Column{
								{Name: "x", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}},
								{Name: "y", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}},
							},
						},
					},
					Tables:    map[string]*querier_dto.Table{},
					Views:     map[string]*querier_dto.View{},
					Enums:     map[string]*querier_dto.Enum{},
					Sequences: map[string]*querier_dto.Sequence{},
				},
			},
		}

		columns := engine.TableValuedFunctionColumnsFromCatalogue(catalogue, "get_records")
		require.Len(t, columns, 2)
		assert.Equal(t, "x", columns[0].Name)
		assert.Equal(t, "y", columns[1].Name)
	})

	t.Run("returns nil for unknown function", func(t *testing.T) {
		t.Parallel()

		catalogue := &querier_dto.Catalogue{
			DefaultSchema: "main",
			Schemas: map[string]*querier_dto.Schema{
				"main": {
					Name:           "main",
					Functions:      map[string][]*querier_dto.FunctionSignature{},
					CompositeTypes: map[string]*querier_dto.CompositeType{},
					Tables:         map[string]*querier_dto.Table{},
					Views:          map[string]*querier_dto.View{},
					Enums:          map[string]*querier_dto.Enum{},
					Sequences:      map[string]*querier_dto.Sequence{},
				},
			},
		}

		columns := engine.TableValuedFunctionColumnsFromCatalogue(catalogue, "nonexistent")
		assert.Nil(t, columns)
	})

	t.Run("returns nil for non-set-returning function", func(t *testing.T) {
		t.Parallel()

		catalogue := &querier_dto.Catalogue{
			DefaultSchema: "main",
			Schemas: map[string]*querier_dto.Schema{
				"main": {
					Name: "main",
					Functions: map[string][]*querier_dto.FunctionSignature{
						"scalar_fn": {
							{Name: "scalar_fn", ReturnsSet: false},
						},
					},
					CompositeTypes: map[string]*querier_dto.CompositeType{},
					Tables:         map[string]*querier_dto.Table{},
					Views:          map[string]*querier_dto.View{},
					Enums:          map[string]*querier_dto.Enum{},
					Sequences:      map[string]*querier_dto.Sequence{},
				},
			},
		}

		columns := engine.TableValuedFunctionColumnsFromCatalogue(catalogue, "scalar_fn")
		assert.Nil(t, columns)
	})
}

func TestResolveFunctionCall(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()
	catalogue := &querier_dto.Catalogue{}

	t.Run("array_agg returns array type", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "array_agg", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryArray, result.ReturnType.Category)
		assert.True(t, result.IsAggregate)
	})

	t.Run("unnest with single array argument", func(t *testing.T) {
		t.Parallel()

		elementType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
		result, err := engine.ResolveFunctionCall(catalogue, "unnest", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryArray, EngineName: "int4[]", ElementType: &elementType}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryInteger, result.ReturnType.Category)
		assert.True(t, result.ReturnsSet)
	})

	t.Run("unnest with multiple arguments returns record", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "unnest", "",
			[]querier_dto.SQLType{
				{Category: querier_dto.TypeCategoryArray, EngineName: "int4[]"},
				{Category: querier_dto.TypeCategoryArray, EngineName: "varchar[]"},
			})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "record", result.ReturnType.EngineName)
		assert.True(t, result.ReturnsSet)
	})

	t.Run("unnest with non-array argument", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "unnest", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.ReturnsSet)
	})

	t.Run("sum with small integer returns hugeint", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "sum", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "hugeint", result.ReturnType.EngineName)
	})

	t.Run("sum with large integer returns numeric", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "sum", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryInteger, EngineName: "hugeint"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "numeric", result.ReturnType.EngineName)
	})

	t.Run("sum with float returns float8", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "sum", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryFloat, EngineName: "float4"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "float8", result.ReturnType.EngineName)
	})

	t.Run("sum with decimal returns numeric", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "sum", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "numeric", result.ReturnType.EngineName)
	})

	t.Run("avg with float returns float8", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "avg", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryFloat, EngineName: "float4"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "float8", result.ReturnType.EngineName)
	})

	t.Run("avg with integer returns numeric", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "avg", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "numeric", result.ReturnType.EngineName)
	})

	t.Run("min returns same type", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "min", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "int4", result.ReturnType.EngineName)
	})

	t.Run("max returns same type", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "max", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "varchar", result.ReturnType.EngineName)
	})

	t.Run("coalesce returns first known type", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "coalesce", "",
			[]querier_dto.SQLType{
				{Category: querier_dto.TypeCategoryUnknown},
				{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
			})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "varchar", result.ReturnType.EngineName)
	})

	t.Run("coalesce with all unknown types", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "coalesce", "",
			[]querier_dto.SQLType{
				{Category: querier_dto.TypeCategoryUnknown},
			})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, result.ReturnType.Category)
	})

	t.Run("typeof returns varchar", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "typeof", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "varchar", result.ReturnType.EngineName)
	})

	t.Run("struct_pack returns struct", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "struct_pack", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryStruct, result.ReturnType.Category)
	})

	t.Run("struct_extract returns unknown", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "struct_extract", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryStruct, EngineName: "struct"}})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("struct_insert returns input type", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "struct_insert", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryStruct, EngineName: "my_struct"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "my_struct", result.ReturnType.EngineName)
	})

	t.Run("struct_insert with no arguments", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "struct_insert", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryStruct, result.ReturnType.Category)
	})

	t.Run("map returns map type", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "map", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryMap, result.ReturnType.Category)
	})

	t.Run("map_keys returns array of key type", func(t *testing.T) {
		t.Parallel()

		keyType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
		result, err := engine.ResolveFunctionCall(catalogue, "map_keys", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryMap, EngineName: "map", KeyType: &keyType}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryArray, result.ReturnType.Category)
	})

	t.Run("map_keys with no arguments", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "map_keys", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryArray, result.ReturnType.Category)
	})

	t.Run("map_values returns array of value type", func(t *testing.T) {
		t.Parallel()

		valueType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
		result, err := engine.ResolveFunctionCall(catalogue, "map_values", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryMap, EngineName: "map", ElementType: &valueType}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryArray, result.ReturnType.Category)
	})

	t.Run("map_values with no arguments", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "map_values", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryArray, result.ReturnType.Category)
	})

	t.Run("map_entries returns array", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "map_entries", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryArray, result.ReturnType.Category)
	})

	t.Run("element_at with map type", func(t *testing.T) {
		t.Parallel()

		valueType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
		result, err := engine.ResolveFunctionCall(catalogue, "element_at", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryMap, EngineName: "map", ElementType: &valueType}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "varchar", result.ReturnType.EngineName)
	})

	t.Run("element_at with array type", func(t *testing.T) {
		t.Parallel()

		elementType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
		result, err := engine.ResolveFunctionCall(catalogue, "element_at", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryArray, EngineName: "int4[]", ElementType: &elementType}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "int4", result.ReturnType.EngineName)
	})

	t.Run("element_at with no arguments", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "element_at", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, result.ReturnType.Category)
	})

	t.Run("list_transform returns input type", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "list_transform", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryArray, EngineName: "int4[]"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryArray, result.ReturnType.Category)
	})

	t.Run("list_filter with no arguments", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "list_filter", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, querier_dto.TypeCategoryArray, result.ReturnType.Category)
	})

	t.Run("array_append returns array passthrough", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "array_append", "",
			[]querier_dto.SQLType{{Category: querier_dto.TypeCategoryArray, EngineName: "int4[]"}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "int4[]", result.ReturnType.EngineName)
	})

	t.Run("array_prepend returns second argument type", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "array_prepend", "",
			[]querier_dto.SQLType{
				{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"},
				{Category: querier_dto.TypeCategoryArray, EngineName: "int4[]"},
			})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "int4[]", result.ReturnType.EngineName)
	})

	t.Run("unknown function returns nil", func(t *testing.T) {
		t.Parallel()

		result, err := engine.ResolveFunctionCall(catalogue, "unknown_function", "", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty arguments return nil for aggregate functions", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"array_agg", "unnest", "min", "sum", "avg"} {
			result, err := engine.ResolveFunctionCall(catalogue, name, "", nil)
			require.NoError(t, err)
			assert.Nil(t, result, "expected nil for %s with no arguments", name)
		}
	})
}

func TestNormaliseTypeName_ArrayType(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	t.Run("varchar array normalises to array category", func(t *testing.T) {
		t.Parallel()

		result := engine.NormaliseTypeName("varchar[]")
		assert.Equal(t, querier_dto.TypeCategoryArray, result.Category)
		require.NotNil(t, result.ElementType)
		assert.Equal(t, "varchar", result.ElementType.EngineName)
	})

	t.Run("multi-dimensional array normalises element type", func(t *testing.T) {
		t.Parallel()

		result := engine.NormaliseTypeName("int4[][]")
		assert.Equal(t, querier_dto.TypeCategoryArray, result.Category)
	})

	t.Run("empty type name normalises to varchar", func(t *testing.T) {
		t.Parallel()

		result := engine.NormaliseTypeName("")
		assert.Equal(t, "varchar", result.EngineName)
		assert.Equal(t, querier_dto.TypeCategoryText, result.Category)
	})
}

func TestNormaliseTypeName_Modifiers(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	t.Run("numeric with precision and scale", func(t *testing.T) {
		t.Parallel()

		result := engine.NormaliseTypeName("numeric", 10, 2)
		assert.Equal(t, querier_dto.TypeCategoryDecimal, result.Category)
		require.NotNil(t, result.Precision)
		assert.Equal(t, 10, *result.Precision)
		require.NotNil(t, result.Scale)
		assert.Equal(t, 2, *result.Scale)
	})

	t.Run("varchar with length modifier", func(t *testing.T) {
		t.Parallel()

		result := engine.NormaliseTypeName("varchar", 255)
		assert.Equal(t, querier_dto.TypeCategoryText, result.Category)
		require.NotNil(t, result.Length)
		assert.Equal(t, 255, *result.Length)
	})

	t.Run("timestamp with precision modifier", func(t *testing.T) {
		t.Parallel()

		result := engine.NormaliseTypeName("timestamp", 6)
		assert.Equal(t, querier_dto.TypeCategoryTemporal, result.Category)
		require.NotNil(t, result.Precision)
		assert.Equal(t, 6, *result.Precision)
	})
}

func TestPromoteType_CrossCategory(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()

	t.Run("different categories return left", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
		result := engine.PromoteType(left, right)
		assert.Equal(t, "int4", result.EngineName, "cross-category should return left type")
	})

	t.Run("text category returns left", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}
		result := engine.PromoteType(left, right)
		assert.Equal(t, "varchar", result.EngineName)
	})

	t.Run("usmallint promoted over int1", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int1"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "usmallint"}
		result := engine.PromoteType(left, right)
		assert.Equal(t, "usmallint", result.EngineName)
	})

	t.Run("ubigint promoted over int4", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "ubigint"}
		result := engine.PromoteType(left, right)
		assert.Equal(t, "ubigint", result.EngineName)
	})

	t.Run("uhugeint promoted over hugeint", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "hugeint"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "uhugeint"}
		result := engine.PromoteType(left, right)
		assert.Equal(t, "uhugeint", result.EngineName)
	})

	t.Run("utinyint promoted over int1", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int1"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "utinyint"}
		result := engine.PromoteType(left, right)
		assert.Equal(t, "utinyint", result.EngineName)
	})

	t.Run("uinteger promoted over int2", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int2"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "uinteger"}
		result := engine.PromoteType(left, right)
		assert.Equal(t, "uinteger", result.EngineName)
	})
}

func TestCanImplicitCast_FloatToDecimal(t *testing.T) {
	t.Parallel()

	engine := NewDuckDBEngine()
	assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryFloat, querier_dto.TypeCategoryDecimal),
		"float to decimal should be allowed")
}
