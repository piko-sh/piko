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

package db_engine_mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestNewMySQLEngine(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	assert.Equal(t, "mysql", engine.Dialect(), "default dialect should be mysql")
	assert.Equal(t, "", engine.DefaultSchema(), "MySQL has no default schema in the PostgreSQL sense")
	assert.Equal(t, querier_dto.ParameterStyleQuestion, engine.ParameterStyle(), "MySQL uses question-mark parameters")
	assert.False(t, engine.SupportsReturning(), "standard MySQL does not support RETURNING")
	assert.NotNil(t, engine.BuiltinFunctions(), "function catalogue should be initialised")
	assert.NotNil(t, engine.BuiltinTypes(), "type catalogue should be initialised")
}

func TestNormaliseTypeName(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	tests := []struct {
		name           string
		input          string
		wantEngineName string
		wantCategory   querier_dto.SQLTypeCategory
	}{
		{
			name:           "int normalises to int",
			input:          "int",
			wantEngineName: "int",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "bigint normalises to bigint",
			input:          "bigint",
			wantEngineName: "bigint",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "tinyint normalises to tinyint",
			input:          "tinyint",
			wantEngineName: "tinyint",
			wantCategory:   querier_dto.TypeCategoryInteger,
		},
		{
			name:           "varchar normalises to varchar",
			input:          "varchar",
			wantEngineName: "varchar",
			wantCategory:   querier_dto.TypeCategoryText,
		},
		{
			name:           "text normalises to text",
			input:          "text",
			wantEngineName: "text",
			wantCategory:   querier_dto.TypeCategoryText,
		},
		{
			name:           "datetime normalises to datetime",
			input:          "datetime",
			wantEngineName: "datetime",
			wantCategory:   querier_dto.TypeCategoryTemporal,
		},
		{
			name:           "timestamp normalises to timestamp",
			input:          "timestamp",
			wantEngineName: "timestamp",
			wantCategory:   querier_dto.TypeCategoryTemporal,
		},
		{
			name:           "boolean normalises to tinyint",
			input:          "boolean",
			wantEngineName: "tinyint",
			wantCategory:   querier_dto.TypeCategoryBoolean,
		},
		{
			name:           "bool normalises to tinyint",
			input:          "bool",
			wantEngineName: "tinyint",
			wantCategory:   querier_dto.TypeCategoryBoolean,
		},
		{
			name:           "json normalises to json",
			input:          "json",
			wantEngineName: "json",
			wantCategory:   querier_dto.TypeCategoryJSON,
		},
		{
			name:           "decimal normalises to decimal",
			input:          "decimal",
			wantEngineName: "decimal",
			wantCategory:   querier_dto.TypeCategoryDecimal,
		},
		{
			name:           "numeric normalises to decimal",
			input:          "numeric",
			wantEngineName: "decimal",
			wantCategory:   querier_dto.TypeCategoryDecimal,
		},
		{
			name:           "float normalises to float",
			input:          "float",
			wantEngineName: "float",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "double normalises to double",
			input:          "double",
			wantEngineName: "double",
			wantCategory:   querier_dto.TypeCategoryFloat,
		},
		{
			name:           "case insensitive normalisation",
			input:          "VARCHAR",
			wantEngineName: "varchar",
			wantCategory:   querier_dto.TypeCategoryText,
		},
		{
			name:           "mixed case normalisation",
			input:          "BigInt",
			wantEngineName: "bigint",
			wantCategory:   querier_dto.TypeCategoryInteger,
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

	engine := NewMySQLEngine()

	t.Run("integer promotion selects the wider type", func(t *testing.T) {
		t.Parallel()

		tinyint := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "tinyint"}
		bigint := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"}

		result := engine.PromoteType(tinyint, bigint)
		assert.Equal(t, "bigint", result.EngineName, "bigint should win over tinyint")

		result = engine.PromoteType(bigint, tinyint)
		assert.Equal(t, "bigint", result.EngineName, "bigint should still win when on the left")
	})

	t.Run("float promotion selects the wider type", func(t *testing.T) {
		t.Parallel()

		floatType := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float"}
		doubleType := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "double"}

		result := engine.PromoteType(floatType, doubleType)
		assert.Equal(t, "double", result.EngineName, "double should win over float")
	})

	t.Run("same type returns identity", func(t *testing.T) {
		t.Parallel()

		intType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int"}

		result := engine.PromoteType(intType, intType)
		assert.Equal(t, "int", result.EngineName, "same type should return itself")
	})
}

func TestCanImplicitCast(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	t.Run("integer to float is allowed", func(t *testing.T) {
		t.Parallel()

		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryFloat))
	})

	t.Run("float to integer is not allowed", func(t *testing.T) {
		t.Parallel()

		assert.False(t, engine.CanImplicitCast(querier_dto.TypeCategoryFloat, querier_dto.TypeCategoryInteger))
	})

	t.Run("integer to decimal is allowed", func(t *testing.T) {
		t.Parallel()

		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryDecimal))
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

	engine := NewMySQLEngine()
	features := engine.SupportedExpressions()

	assert.Zero(t, features&querier_dto.SQLFeatureStringConcat,
		"MySQL should not support string concatenation via ||")

	assert.NotZero(t, features&querier_dto.SQLFeatureWindowFunction,
		"MySQL should support window functions")

	assert.NotZero(t, features&querier_dto.SQLFeatureJSONOp,
		"MySQL should support JSON operators")
}

func TestParameterStyle(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	assert.Equal(t, querier_dto.ParameterStyleQuestion, engine.ParameterStyle(),
		"MySQL should use question-mark parameter style")
}

func TestSupportsReturning(t *testing.T) {
	t.Parallel()

	t.Run("false by default", func(t *testing.T) {
		t.Parallel()

		engine := NewMySQLEngine()
		assert.False(t, engine.SupportsReturning(), "standard MySQL does not support RETURNING")
	})

	t.Run("true with WithReturningSupport override", func(t *testing.T) {
		t.Parallel()

		engine := NewMySQLEngine(WithReturningSupport(true))
		assert.True(t, engine.SupportsReturning(), "engine with returning support should report true")
	})
}

func TestDialect(t *testing.T) {
	t.Parallel()

	t.Run("default dialect is mysql", func(t *testing.T) {
		t.Parallel()

		engine := NewMySQLEngine()
		assert.Equal(t, "mysql", engine.Dialect())
	})

	t.Run("custom dialect name via option", func(t *testing.T) {
		t.Parallel()

		engine := NewMySQLEngine(WithDialectName("mariadb"))
		assert.Equal(t, "mariadb", engine.Dialect())
	})
}

func TestDefaultSchema(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	assert.Equal(t, "", engine.DefaultSchema(),
		"MySQL should return an empty default schema")
}

func TestBuiltinFunctions(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	catalogue := engine.BuiltinFunctions()

	require.NotNil(t, catalogue, "function catalogue must not be nil")
	require.NotNil(t, catalogue.Functions, "function map must not be nil")

	expectedFunctions := []string{"concat", "count", "abs", "now"}
	for _, name := range expectedFunctions {
		signatures, exists := catalogue.Functions[name]
		assert.True(t, exists, "expected built-in function %q to be registered", name)
		assert.NotEmpty(t, signatures, "expected at least one signature for %q", name)
	}
}

func TestBuiltinTypes(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	catalogue := engine.BuiltinTypes()

	require.NotNil(t, catalogue, "type catalogue must not be nil")
	require.NotNil(t, catalogue.Types, "type map must not be nil")

	expectedTypes := []string{"int", "varchar", "text", "boolean", "json", "datetime", "bigint", "float", "double", "decimal"}
	for _, name := range expectedTypes {
		_, exists := catalogue.Types[name]
		assert.True(t, exists, "expected built-in type %q to be registered", name)
	}
}

func TestCommentStyle(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	style := engine.CommentStyle()

	assert.NotEmpty(t, style.LinePrefix, "line comment prefix should not be empty")
}

func TestSupportedDirectivePrefixes(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	prefixes := engine.SupportedDirectivePrefixes()

	require.NotEmpty(t, prefixes, "MySQL should have at least one directive prefix")
}

func TestTableValuedFunctionColumns(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	columns := engine.TableValuedFunctionColumns("nonexistent_tvf")
	assert.Nil(t, columns, "unknown TVF should return nil columns")
}

func TestTableValuedFunctionColumnsFromCatalogue(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	catalogue := &querier_dto.Catalogue{
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name:      "public",
				Functions: map[string][]*querier_dto.FunctionSignature{},
			},
		},
	}

	columns := engine.TableValuedFunctionColumnsFromCatalogue(catalogue, "nonexistent_fn")
	assert.Nil(t, columns, "unknown function should return nil columns")
}

func TestResolveFunctionCall(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	catalogue := &querier_dto.Catalogue{
		Schemas: map[string]*querier_dto.Schema{},
	}

	_, err := engine.ResolveFunctionCall(catalogue, "nonexistent_fn", "", nil)
	_ = err
}

func TestWithExtraTypes(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine(WithExtraTypes(map[string]querier_dto.SQLType{
		"custom_type": {Category: querier_dto.TypeCategoryText, EngineName: "custom_type"},
	}))

	catalogue := engine.BuiltinTypes()
	_, exists := catalogue.Types["custom_type"]
	assert.True(t, exists, "extra type should be registered")
}

func TestWithExtraFunctions(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine(WithExtraFunctions(func(builder *FunctionCatalogueBuilder) {
		builder.NullOnNull("custom_func",
			builder.Args("x", builder.Integer()),
			builder.Integer(),
		)
	}))

	catalogue := engine.BuiltinFunctions()
	_, exists := catalogue.Functions["custom_func"]
	assert.True(t, exists, "extra function should be registered")
}

func TestWithSequenceSupport(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine(WithSequenceSupport(true))
	assert.True(t, engine.dialect.SupportsSequences, "sequence support should be enabled")
}

func TestWithJSONTypeOverride(t *testing.T) {
	t.Parallel()

	override := querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "longtext"}
	engine := NewMySQLEngine(WithJSONTypeOverride(override))
	require.NotNil(t, engine.dialect.JSONTypeOverride)
	assert.Equal(t, "longtext", engine.dialect.JSONTypeOverride.EngineName)
}

func TestWithTypeNormaliserHook(t *testing.T) {
	t.Parallel()

	hook := func(name string, modifiers []int) *querier_dto.SQLType {
		if name == "custom" {
			return &querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "custom_resolved"}
		}
		return nil
	}
	engine := NewMySQLEngine(WithTypeNormaliserHook(hook))
	result := engine.NormaliseTypeName("custom")
	assert.Equal(t, "custom_resolved", result.EngineName)
}

func TestWithImplicitCastHook(t *testing.T) {
	t.Parallel()

	trueVal := true
	hook := func(from, to querier_dto.SQLTypeCategory) *bool {
		if from == querier_dto.TypeCategoryText && to == querier_dto.TypeCategoryInteger {
			return &trueVal
		}
		return nil
	}
	engine := NewMySQLEngine(WithImplicitCastHook(hook))
	assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryText, querier_dto.TypeCategoryInteger),
		"hook should allow text to integer cast")
}

func TestWithPromoteTypeHook(t *testing.T) {
	t.Parallel()

	hook := func(left, right querier_dto.SQLType) *querier_dto.SQLType {
		if left.Category == querier_dto.TypeCategoryText && right.Category == querier_dto.TypeCategoryText {
			result := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "longtext"}
			return &result
		}
		return nil
	}
	engine := NewMySQLEngine(WithPromoteTypeHook(hook))
	result := engine.PromoteType(
		querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
	)
	assert.Equal(t, "longtext", result.EngineName)
}

func TestMySQLFunctionResolver_ResolveFunctionCall(t *testing.T) {
	t.Parallel()

	resolver := NewMySQLFunctionResolver()

	t.Run("IF with integer arguments returns integer", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "IF", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryBoolean, EngineName: "tinyint"},
			{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
			{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryInteger, resolution.ReturnType.Category)
		assert.Equal(t, querier_dto.FunctionNullableCalledOnNull, resolution.NullableBehaviour)
	})

	t.Run("IF with mixed integer and text returns text", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "if", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryBoolean, EngineName: "tinyint"},
			{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
			{Category: querier_dto.TypeCategoryText, EngineName: "text"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryText, resolution.ReturnType.Category)
	})

	t.Run("IF with too few arguments returns nil", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "IF", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryBoolean, EngineName: "tinyint"},
		})
		require.NoError(t, err)
		assert.Nil(t, resolution)
	})

	t.Run("IFNULL returns promoted type", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "IFNULL", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
			{Category: querier_dto.TypeCategoryFloat, EngineName: "double"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryFloat, resolution.ReturnType.Category)
	})

	t.Run("IFNULL with too few arguments returns nil", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "IFNULL", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
		})
		require.NoError(t, err)
		assert.Nil(t, resolution)
	})

	t.Run("COALESCE with known types returns promoted type", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "COALESCE", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
			{Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"},
			{Category: querier_dto.TypeCategoryFloat, EngineName: "double"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)

		assert.Equal(t, querier_dto.TypeCategoryDecimal, resolution.ReturnType.Category)
	})

	t.Run("COALESCE with all unknown returns unknown", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "COALESCE", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
			{Category: querier_dto.TypeCategoryUnknown, EngineName: ""},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, resolution.ReturnType.Category)
	})

	t.Run("GREATEST delegates to coalesce resolver", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "GREATEST", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
			{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryText, resolution.ReturnType.Category)
	})

	t.Run("LEAST delegates to coalesce resolver", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "LEAST", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryFloat, EngineName: "float"},
			{Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryDecimal, resolution.ReturnType.Category)
	})

	t.Run("GROUP_CONCAT returns text aggregate", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "GROUP_CONCAT", "", nil)
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryText, resolution.ReturnType.Category)
		assert.True(t, resolution.IsAggregate)
	})

	t.Run("JSON_EXTRACT returns JSON", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "JSON_EXTRACT", "", nil)
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryJSON, resolution.ReturnType.Category)
		assert.Equal(t, querier_dto.FunctionNullableReturnsNullOnNull, resolution.NullableBehaviour)
	})

	t.Run("JSON_UNQUOTE returns text", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "JSON_UNQUOTE", "", nil)
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryText, resolution.ReturnType.Category)
	})

	t.Run("CONCAT returns text", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "CONCAT", "", nil)
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryText, resolution.ReturnType.Category)
	})

	t.Run("CONCAT_WS returns text", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "CONCAT_WS", "", nil)
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, querier_dto.TypeCategoryText, resolution.ReturnType.Category)
	})

	t.Run("SUM of float returns double", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "SUM", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryFloat, EngineName: "float"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, "double", resolution.ReturnType.EngineName)
		assert.True(t, resolution.IsAggregate)
	})

	t.Run("SUM of integer returns decimal", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "SUM", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryInteger, EngineName: "int"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, "decimal", resolution.ReturnType.EngineName)
	})

	t.Run("SUM with no arguments returns nil", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "SUM", "", nil)
		require.NoError(t, err)
		assert.Nil(t, resolution)
	})

	t.Run("AVG returns double", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "AVG", "", nil)
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, "double", resolution.ReturnType.EngineName)
		assert.True(t, resolution.IsAggregate)
	})

	t.Run("MIN returns identity type", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "MIN", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, "varchar", resolution.ReturnType.EngineName)
		assert.True(t, resolution.IsAggregate)
	})

	t.Run("MAX returns identity type", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "MAX", "", []querier_dto.SQLType{
			{Category: querier_dto.TypeCategoryInteger, EngineName: "bigint"},
		})
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, "bigint", resolution.ReturnType.EngineName)
		assert.True(t, resolution.IsAggregate)
	})

	t.Run("MIN with no arguments returns nil", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "MIN", "", nil)
		require.NoError(t, err)
		assert.Nil(t, resolution)
	})

	t.Run("COUNT returns bigint", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "COUNT", "", nil)
		require.NoError(t, err)
		require.NotNil(t, resolution)
		assert.Equal(t, "bigint", resolution.ReturnType.EngineName)
		assert.Equal(t, querier_dto.FunctionNullableNeverNull, resolution.NullableBehaviour)
		assert.True(t, resolution.IsAggregate)
	})

	t.Run("unknown function returns nil nil", func(t *testing.T) {
		t.Parallel()

		resolution, err := resolver.ResolveFunctionCall(nil, "UNKNOWN_FUNC", "", nil)
		require.NoError(t, err)
		assert.Nil(t, resolution)
	})
}

func TestPromoteType_CrossCategory(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	t.Run("different categories returns left", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int"}
		result := engine.PromoteType(left, right)
		assert.Equal(t, "varchar", result.EngineName, "different categories should return left")
	})

	t.Run("default category returns left", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"}
		result := engine.PromoteType(left, right)
		assert.Equal(t, "json", result.EngineName, "non-integer non-float same category should return left")
	})
}

func TestPromoteTypes_Internal(t *testing.T) {
	t.Parallel()

	t.Run("left unknown returns right", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int"}
		result := promoteTypes(left, right)
		assert.Equal(t, "int", result.EngineName)
	})

	t.Run("right unknown returns left", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
		result := promoteTypes(left, right)
		assert.Equal(t, "float", result.EngineName)
	})

	t.Run("same category returns left", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}
		result := promoteTypes(left, right)
		assert.Equal(t, "decimal", result.EngineName)
	})

	t.Run("integer vs text promotes to text", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}
		result := promoteTypes(left, right)
		assert.Equal(t, querier_dto.TypeCategoryText, result.Category)
	})

	t.Run("float vs decimal promotes to decimal", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "double"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "decimal"}
		result := promoteTypes(left, right)
		assert.Equal(t, querier_dto.TypeCategoryDecimal, result.Category)
	})

	t.Run("unknown category has zero rank", func(t *testing.T) {
		t.Parallel()

		left := querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"}
		right := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int"}
		result := promoteTypes(left, right)
		assert.Equal(t, querier_dto.TypeCategoryInteger, result.Category,
			"json has rank 0 so integer with rank 1 should win")
	})
}

func TestFunctionCatalogueBuilder_TypeAccessors(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine(WithExtraFunctions(func(builder *FunctionCatalogueBuilder) {

		builder.NullOnNull("test_bigint", builder.Args("x", builder.Bigint()), builder.Bigint())
		builder.NullOnNull("test_float", builder.Args("x", builder.Float()), builder.Float())
		builder.NullOnNull("test_double", builder.Args("x", builder.Double()), builder.Double())
		builder.NullOnNull("test_decimal", builder.Args("x", builder.Decimal()), builder.Decimal())
		builder.NullOnNull("test_text", builder.Args("x", builder.Text()), builder.Text())
		builder.NullOnNull("test_varchar", builder.Args("x", builder.Varchar()), builder.Varchar())
		builder.NullOnNull("test_boolean", builder.Args("x", builder.Boolean()), builder.Boolean())
		builder.NullOnNull("test_bytea", builder.Args("x", builder.Bytea()), builder.Bytea())
		builder.NullOnNull("test_date", builder.Args("x", builder.Date()), builder.Date())
		builder.NullOnNull("test_time", builder.Args("x", builder.Time()), builder.Time())
		builder.NullOnNull("test_datetime", builder.Args("x", builder.Datetime()), builder.Datetime())
		builder.NullOnNull("test_timestamp", builder.Args("x", builder.Timestamp()), builder.Timestamp())
		builder.NullOnNull("test_json", builder.Args("x", builder.JSON()), builder.JSON())
		builder.NullOnNull("test_geometry", builder.Args("x", builder.Geometry()), builder.Geometry())
		builder.Variadic("test_variadic",
			builder.Args("values", builder.Text()),
			1,
			builder.Text(),
		)
	}))

	catalogue := engine.BuiltinFunctions()

	expectedFunctions := []string{
		"test_bigint", "test_float", "test_double", "test_decimal",
		"test_text", "test_varchar", "test_boolean", "test_bytea",
		"test_date", "test_time", "test_datetime", "test_timestamp",
		"test_json", "test_geometry", "test_variadic",
	}
	for _, name := range expectedFunctions {
		signatures, exists := catalogue.Functions[name]
		assert.True(t, exists, "expected function %q to be registered", name)
		assert.NotEmpty(t, signatures, "expected at least one signature for %q", name)
	}

	variadicSignatures := catalogue.Functions["test_variadic"]
	require.Len(t, variadicSignatures, 1)
	assert.True(t, variadicSignatures[0].IsVariadic)
	assert.Equal(t, 1, variadicSignatures[0].MinArguments)
}

func TestTableValuedFunctionColumnsFromCatalogue_CompositeType(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	t.Run("resolves composite type columns from set-returning function", func(t *testing.T) {
		t.Parallel()

		catalogue := &querier_dto.Catalogue{
			Schemas: map[string]*querier_dto.Schema{
				"public": {
					Name: "public",
					Functions: map[string][]*querier_dto.FunctionSignature{
						"get_users": {
							{
								Name:       "get_users",
								ReturnsSet: true,
								ReturnType: querier_dto.SQLType{
									Category:   querier_dto.TypeCategoryUnknown,
									EngineName: "user_type",
								},
							},
						},
					},
					CompositeTypes: map[string]*querier_dto.CompositeType{
						"user_type": {
							Fields: []querier_dto.Column{
								{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int"}},
								{Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}},
							},
						},
					},
				},
			},
		}

		columns := engine.TableValuedFunctionColumnsFromCatalogue(catalogue, "get_users")
		require.Len(t, columns, 2)
		assert.Equal(t, "id", columns[0].Name)
		assert.Equal(t, "name", columns[1].Name)
	})

	t.Run("returns nil for non-set-returning function", func(t *testing.T) {
		t.Parallel()

		catalogue := &querier_dto.Catalogue{
			Schemas: map[string]*querier_dto.Schema{
				"public": {
					Name: "public",
					Functions: map[string][]*querier_dto.FunctionSignature{
						"scalar_fn": {
							{
								Name:       "scalar_fn",
								ReturnsSet: false,
								ReturnType: querier_dto.SQLType{
									Category:   querier_dto.TypeCategoryInteger,
									EngineName: "int",
								},
							},
						},
					},
				},
			},
		}

		columns := engine.TableValuedFunctionColumnsFromCatalogue(catalogue, "scalar_fn")
		assert.Nil(t, columns)
	})

	t.Run("resolves composite type from different schema", func(t *testing.T) {
		t.Parallel()

		catalogue := &querier_dto.Catalogue{
			Schemas: map[string]*querier_dto.Schema{
				"public": {
					Name: "public",
					Functions: map[string][]*querier_dto.FunctionSignature{
						"get_items": {
							{
								Name:       "get_items",
								ReturnsSet: true,
								ReturnType: querier_dto.SQLType{
									Category:   querier_dto.TypeCategoryUnknown,
									EngineName: "item_type",
									Schema:     "other",
								},
							},
						},
					},
				},
				"other": {
					Name: "other",
					CompositeTypes: map[string]*querier_dto.CompositeType{
						"item_type": {
							Fields: []querier_dto.Column{
								{Name: "item_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int"}},
							},
						},
					},
				},
			},
		}

		columns := engine.TableValuedFunctionColumnsFromCatalogue(catalogue, "get_items")
		require.Len(t, columns, 1)
		assert.Equal(t, "item_id", columns[0].Name)
	})
}

func TestCanImplicitCast_FloatToDecimal(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryFloat, querier_dto.TypeCategoryDecimal),
		"float to decimal cast should be allowed")
}

func TestIsParsedStatement(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	statements, err := engine.ParseStatements("SELECT 1;")
	require.NoError(t, err)
	require.Len(t, statements, 1)

	parsed, ok := statements[0].Raw.(*parsedStatement)
	require.True(t, ok)
	parsed.IsParsedStatement()
}

type fakeStatement struct{}

func (*fakeStatement) IsParsedStatement() {}

func TestApplyDDL_InvalidStatementType(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	_, err := engine.ApplyDDL(querier_dto.ParsedStatement{
		Raw: &fakeStatement{},
	})
	assert.Error(t, err, "non-parsedStatement should produce an error")
}

func TestAnalyseQuery_InvalidStatementType(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()
	_, err := engine.AnalyseQuery(nil, querier_dto.ParsedStatement{
		Raw: &fakeStatement{},
	})
	assert.Error(t, err, "non-parsedStatement should produce an error")
}

func TestResolveFunctionCall_ViaEngine(t *testing.T) {
	t.Parallel()

	engine := NewMySQLEngine()

	resolution, err := engine.ResolveFunctionCall(nil, "COUNT", "", nil)
	require.NoError(t, err)
	require.NotNil(t, resolution)
	assert.Equal(t, "bigint", resolution.ReturnType.EngineName)
}
