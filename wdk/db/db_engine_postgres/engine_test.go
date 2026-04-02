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

package db_engine_postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestNewPostgresEngine(t *testing.T) {
	t.Parallel()

	t.Run("default construction", func(t *testing.T) {
		t.Parallel()

		engine := NewPostgresEngine()

		assert.Equal(t, "postgres", engine.Dialect(), "dialect should default to postgres")
		assert.Equal(t, "public", engine.DefaultSchema(), "default schema should be public")
		assert.True(t, engine.SupportsReturning(), "PostgreSQL supports RETURNING")
		assert.Equal(t, querier_dto.ParameterStyleDollar, engine.ParameterStyle(), "parameter style should be dollar")
		assert.NotNil(t, engine.BuiltinFunctions(), "function catalogue should not be nil")
		assert.NotNil(t, engine.BuiltinTypes(), "type catalogue should not be nil")
	})

	t.Run("with dialect name option", func(t *testing.T) {
		t.Parallel()

		engine := NewPostgresEngine(WithDialectName("cockroachdb"))

		assert.Equal(t, "cockroachdb", engine.Dialect(), "dialect should reflect custom name")

		assert.Equal(t, "public", engine.DefaultSchema())
		assert.True(t, engine.SupportsReturning())
	})

	t.Run("with extra types option", func(t *testing.T) {
		t.Parallel()

		extra := map[string]querier_dto.SQLType{
			"crdb_internal_type": {Category: querier_dto.TypeCategoryText, EngineName: "crdb_internal_type"},
		}
		engine := NewPostgresEngine(WithExtraTypes(extra))

		catalogue := engine.BuiltinTypes()
		_, exists := catalogue.Types["crdb_internal_type"]
		assert.True(t, exists, "extra type should appear in the type catalogue")
	})

	t.Run("with extra functions option", func(t *testing.T) {
		t.Parallel()

		engine := NewPostgresEngine(WithExtraFunctions(func(builder *FunctionCatalogueBuilder) {
			builder.Catalogue.Functions["custom_fn"] = []*querier_dto.FunctionSignature{
				{
					ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				},
			}
		}))

		catalogue := engine.BuiltinFunctions()
		_, exists := catalogue.Functions["custom_fn"]
		assert.True(t, exists, "extra function should appear in the function catalogue")
	})
}

func TestNormaliseTypeName(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	tests := []struct {
		name            string
		input           string
		modifiers       []int
		wantCategory    querier_dto.SQLTypeCategory
		wantEngineName  string
		wantIsArray     bool
		wantElementName string
	}{

		{
			name:           "uppercase INTEGER normalises to int4",
			input:          "INTEGER",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int4",
		},
		{
			name:           "lowercase integer normalises to int4",
			input:          "integer",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int4",
		},
		{
			name:           "mixed case Integer normalises to int4",
			input:          "Integer",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int4",
		},

		{
			name:           "int alias normalises to int4",
			input:          "int",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int4",
		},
		{
			name:           "int4 stays int4",
			input:          "int4",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int4",
		},
		{
			name:           "bigint normalises to int8",
			input:          "bigint",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int8",
		},
		{
			name:           "int8 stays int8",
			input:          "int8",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int8",
		},
		{
			name:           "smallint normalises to int2",
			input:          "smallint",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int2",
		},
		{
			name:           "int2 stays int2",
			input:          "int2",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int2",
		},

		{
			name:           "serial normalises to int4",
			input:          "serial",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int4",
		},
		{
			name:           "bigserial normalises to int8",
			input:          "bigserial",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int8",
		},
		{
			name:           "smallserial normalises to int2",
			input:          "smallserial",
			wantCategory:   querier_dto.TypeCategoryInteger,
			wantEngineName: "int2",
		},

		{
			name:           "double precision normalises to float8",
			input:          "double precision",
			wantCategory:   querier_dto.TypeCategoryFloat,
			wantEngineName: "float8",
		},
		{
			name:           "real normalises to float4",
			input:          "real",
			wantCategory:   querier_dto.TypeCategoryFloat,
			wantEngineName: "float4",
		},
		{
			name:           "float normalises to float8",
			input:          "float",
			wantCategory:   querier_dto.TypeCategoryFloat,
			wantEngineName: "float8",
		},
		{
			name:           "float4 stays float4",
			input:          "float4",
			wantCategory:   querier_dto.TypeCategoryFloat,
			wantEngineName: "float4",
		},
		{
			name:           "float8 stays float8",
			input:          "float8",
			wantCategory:   querier_dto.TypeCategoryFloat,
			wantEngineName: "float8",
		},

		{
			name:           "character varying normalises to varchar",
			input:          "character varying",
			wantCategory:   querier_dto.TypeCategoryText,
			wantEngineName: "varchar",
		},
		{
			name:           "varchar normalises to varchar",
			input:          "varchar",
			wantCategory:   querier_dto.TypeCategoryText,
			wantEngineName: "varchar",
		},
		{
			name:           "text normalises to text",
			input:          "text",
			wantCategory:   querier_dto.TypeCategoryText,
			wantEngineName: "text",
		},
		{
			name:           "char normalises to char",
			input:          "char",
			wantCategory:   querier_dto.TypeCategoryText,
			wantEngineName: "char",
		},
		{
			name:           "character normalises to char",
			input:          "character",
			wantCategory:   querier_dto.TypeCategoryText,
			wantEngineName: "char",
		},

		{
			name:           "timestamp with time zone normalises to timestamptz",
			input:          "timestamp with time zone",
			wantCategory:   querier_dto.TypeCategoryTemporal,
			wantEngineName: "timestamptz",
		},
		{
			name:           "timestamptz stays timestamptz",
			input:          "timestamptz",
			wantCategory:   querier_dto.TypeCategoryTemporal,
			wantEngineName: "timestamptz",
		},
		{
			name:           "timestamp without time zone normalises to timestamp",
			input:          "timestamp without time zone",
			wantCategory:   querier_dto.TypeCategoryTemporal,
			wantEngineName: "timestamp",
		},
		{
			name:           "timestamp normalises to timestamp",
			input:          "timestamp",
			wantCategory:   querier_dto.TypeCategoryTemporal,
			wantEngineName: "timestamp",
		},
		{
			name:           "date normalises to date",
			input:          "date",
			wantCategory:   querier_dto.TypeCategoryTemporal,
			wantEngineName: "date",
		},
		{
			name:           "interval normalises to interval",
			input:          "interval",
			wantCategory:   querier_dto.TypeCategoryTemporal,
			wantEngineName: "interval",
		},

		{
			name:           "boolean normalises to bool",
			input:          "boolean",
			wantCategory:   querier_dto.TypeCategoryBoolean,
			wantEngineName: "bool",
		},
		{
			name:           "bool stays bool",
			input:          "bool",
			wantCategory:   querier_dto.TypeCategoryBoolean,
			wantEngineName: "bool",
		},

		{
			name:           "json normalises to json",
			input:          "json",
			wantCategory:   querier_dto.TypeCategoryJSON,
			wantEngineName: "json",
		},
		{
			name:           "jsonb normalises to jsonb",
			input:          "jsonb",
			wantCategory:   querier_dto.TypeCategoryJSON,
			wantEngineName: "jsonb",
		},

		{
			name:           "uuid normalises to uuid",
			input:          "uuid",
			wantCategory:   querier_dto.TypeCategoryUUID,
			wantEngineName: "uuid",
		},

		{
			name:           "bytea normalises to bytea",
			input:          "bytea",
			wantCategory:   querier_dto.TypeCategoryBytea,
			wantEngineName: "bytea",
		},

		{
			name:           "numeric normalises to numeric",
			input:          "numeric",
			wantCategory:   querier_dto.TypeCategoryDecimal,
			wantEngineName: "numeric",
		},
		{
			name:           "decimal normalises to numeric",
			input:          "decimal",
			wantCategory:   querier_dto.TypeCategoryDecimal,
			wantEngineName: "numeric",
		},

		{
			name:            "integer array normalises to int4 array",
			input:           "integer[]",
			wantCategory:    querier_dto.TypeCategoryArray,
			wantEngineName:  "integer[]",
			wantIsArray:     true,
			wantElementName: "int4",
		},
		{
			name:            "text array normalises to text array",
			input:           "text[]",
			wantCategory:    querier_dto.TypeCategoryArray,
			wantEngineName:  "text[]",
			wantIsArray:     true,
			wantElementName: "text",
		},

		{
			name:           "inet normalises to inet",
			input:          "inet",
			wantCategory:   querier_dto.TypeCategoryNetwork,
			wantEngineName: "inet",
		},

		{
			name:           "int4range normalises to int4range",
			input:          "int4range",
			wantCategory:   querier_dto.TypeCategoryRange,
			wantEngineName: "int4range",
		},

		{
			name:           "unknown type preserves engine name",
			input:          "my_custom_type",
			wantCategory:   querier_dto.TypeCategoryUnknown,
			wantEngineName: "my_custom_type",
		},
		{
			name:           "unknown type is lowered",
			input:          "MY_CUSTOM_TYPE",
			wantCategory:   querier_dto.TypeCategoryUnknown,
			wantEngineName: "my_custom_type",
		},

		{
			name:           "empty string defaults to text",
			input:          "",
			wantCategory:   querier_dto.TypeCategoryText,
			wantEngineName: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := engine.NormaliseTypeName(tt.input, tt.modifiers...)

			assert.Equal(t, tt.wantCategory, got.Category, "category mismatch")
			assert.Equal(t, tt.wantEngineName, got.EngineName, "engine name mismatch")

			if tt.wantIsArray {
				require.NotNil(t, got.ElementType, "expected non-nil ElementType for array type")
				assert.Equal(t, tt.wantElementName, got.ElementType.EngineName, "element engine name mismatch")
			} else {
				assert.Nil(t, got.ElementType, "expected nil ElementType for non-array type")
			}
		})
	}
}

func TestNormaliseTypeName_Modifiers(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	t.Run("numeric with precision and scale", func(t *testing.T) {
		t.Parallel()

		got := engine.NormaliseTypeName("numeric", 10, 2)

		assert.Equal(t, querier_dto.TypeCategoryDecimal, got.Category)
		assert.Equal(t, "numeric", got.EngineName)
		require.NotNil(t, got.Precision, "precision should be set")
		assert.Equal(t, 10, *got.Precision)
		require.NotNil(t, got.Scale, "scale should be set")
		assert.Equal(t, 2, *got.Scale)
	})

	t.Run("varchar with length", func(t *testing.T) {
		t.Parallel()

		got := engine.NormaliseTypeName("varchar", 255)

		assert.Equal(t, querier_dto.TypeCategoryText, got.Category)
		assert.Equal(t, "varchar", got.EngineName)
		require.NotNil(t, got.Length, "length should be set")
		assert.Equal(t, 255, *got.Length)
	})

	t.Run("timestamp with precision", func(t *testing.T) {
		t.Parallel()

		got := engine.NormaliseTypeName("timestamp", 6)

		assert.Equal(t, querier_dto.TypeCategoryTemporal, got.Category)
		assert.Equal(t, "timestamp", got.EngineName)
		require.NotNil(t, got.Precision, "precision should be set")
		assert.Equal(t, 6, *got.Precision)
	})

	t.Run("integer ignores modifiers", func(t *testing.T) {
		t.Parallel()

		got := engine.NormaliseTypeName("integer", 42)

		assert.Equal(t, querier_dto.TypeCategoryInteger, got.Category)
		assert.Nil(t, got.Precision, "integer should not have precision")
		assert.Nil(t, got.Scale, "integer should not have scale")
		assert.Nil(t, got.Length, "integer should not have length")
	})
}

func TestNormaliseTypeName_Hook(t *testing.T) {
	t.Parallel()

	hookedType := querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryText,
		EngineName: "hooked_text",
	}

	engine := NewPostgresEngine(WithTypeNormaliserHook(func(name string, _ []int) *querier_dto.SQLType {
		if name == "my_special_type" {
			return &hookedType
		}
		return nil
	}))

	t.Run("hook intercepts matching type", func(t *testing.T) {
		t.Parallel()

		got := engine.NormaliseTypeName("MY_SPECIAL_TYPE")
		assert.Equal(t, hookedType, got)
	})

	t.Run("hook falls through for non-matching type", func(t *testing.T) {
		t.Parallel()

		got := engine.NormaliseTypeName("integer")
		assert.Equal(t, querier_dto.TypeCategoryInteger, got.Category)
		assert.Equal(t, "int4", got.EngineName)
	})
}

func TestPromoteType(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	int2 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int2"}
	int4 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
	integer8 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"}
	float4 := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float4"}
	float8 := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}
	text := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}

	tests := []struct {
		name  string
		left  querier_dto.SQLType
		right querier_dto.SQLType
		want  querier_dto.SQLType
	}{

		{
			name:  "int2 + int4 promotes to int4",
			left:  int2,
			right: int4,
			want:  int4,
		},
		{
			name:  "int4 + int2 promotes to int4",
			left:  int4,
			right: int2,
			want:  int4,
		},
		{
			name:  "int4 + int8 promotes to int8",
			left:  int4,
			right: integer8,
			want:  integer8,
		},
		{
			name:  "int8 + int4 promotes to int8",
			left:  integer8,
			right: int4,
			want:  integer8,
		},
		{
			name:  "int2 + int8 promotes to int8",
			left:  int2,
			right: integer8,
			want:  integer8,
		},

		{
			name:  "float4 + float8 promotes to float8",
			left:  float4,
			right: float8,
			want:  float8,
		},
		{
			name:  "float8 + float4 promotes to float8",
			left:  float8,
			right: float4,
			want:  float8,
		},

		{
			name:  "int4 + int4 returns int4",
			left:  int4,
			right: int4,
			want:  int4,
		},
		{
			name:  "float8 + float8 returns float8",
			left:  float8,
			right: float8,
			want:  float8,
		},

		{
			name:  "int4 + float8 returns left (different categories)",
			left:  int4,
			right: float8,
			want:  int4,
		},
		{
			name:  "float4 + int4 returns left (different categories)",
			left:  float4,
			right: int4,
			want:  float4,
		},
		{
			name:  "text + int4 returns left (different categories)",
			left:  text,
			right: int4,
			want:  text,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := engine.PromoteType(tt.left, tt.right)
			assert.Equal(t, tt.want.Category, got.Category, "category mismatch")
			assert.Equal(t, tt.want.EngineName, got.EngineName, "engine name mismatch")
		})
	}
}

func TestPromoteType_Hook(t *testing.T) {
	t.Parallel()

	promoted := querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: "numeric"}

	engine := NewPostgresEngine(WithPromoteTypeHook(func(left, right querier_dto.SQLType) *querier_dto.SQLType {

		if left.Category == querier_dto.TypeCategoryInteger && right.Category == querier_dto.TypeCategoryFloat {
			return &promoted
		}
		return nil
	}))

	t.Run("hook intercepts cross-category promotion", func(t *testing.T) {
		t.Parallel()

		int4 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int4"}
		float8 := querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "float8"}

		got := engine.PromoteType(int4, float8)
		assert.Equal(t, promoted, got)
	})

	t.Run("hook falls through for same-category promotion", func(t *testing.T) {
		t.Parallel()

		int2 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int2"}
		integer8 := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"}

		got := engine.PromoteType(int2, integer8)
		assert.Equal(t, querier_dto.TypeCategoryInteger, got.Category)
		assert.Equal(t, "int8", got.EngineName)
	})
}

func TestCanImplicitCast(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	tests := []struct {
		name string
		from querier_dto.SQLTypeCategory
		to   querier_dto.SQLTypeCategory
		want bool
	}{

		{
			name: "integer to float is allowed",
			from: querier_dto.TypeCategoryInteger,
			to:   querier_dto.TypeCategoryFloat,
			want: true,
		},
		{
			name: "integer to decimal is allowed",
			from: querier_dto.TypeCategoryInteger,
			to:   querier_dto.TypeCategoryDecimal,
			want: true,
		},
		{
			name: "float to decimal is allowed",
			from: querier_dto.TypeCategoryFloat,
			to:   querier_dto.TypeCategoryDecimal,
			want: true,
		},
		{
			name: "text to text is allowed",
			from: querier_dto.TypeCategoryText,
			to:   querier_dto.TypeCategoryText,
			want: true,
		},

		{
			name: "float to integer is not allowed",
			from: querier_dto.TypeCategoryFloat,
			to:   querier_dto.TypeCategoryInteger,
			want: false,
		},
		{
			name: "boolean to integer is not allowed",
			from: querier_dto.TypeCategoryBoolean,
			to:   querier_dto.TypeCategoryInteger,
			want: false,
		},
		{
			name: "text to integer is not allowed",
			from: querier_dto.TypeCategoryText,
			to:   querier_dto.TypeCategoryInteger,
			want: false,
		},
		{
			name: "integer to text is not allowed",
			from: querier_dto.TypeCategoryInteger,
			to:   querier_dto.TypeCategoryText,
			want: false,
		},
		{
			name: "integer to boolean is not allowed",
			from: querier_dto.TypeCategoryInteger,
			to:   querier_dto.TypeCategoryBoolean,
			want: false,
		},
		{
			name: "decimal to integer is not allowed",
			from: querier_dto.TypeCategoryDecimal,
			to:   querier_dto.TypeCategoryInteger,
			want: false,
		},
		{
			name: "json to text is not allowed",
			from: querier_dto.TypeCategoryJSON,
			to:   querier_dto.TypeCategoryText,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := engine.CanImplicitCast(tt.from, tt.to)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCanImplicitCast_Hook(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine(WithImplicitCastHook(func(from, to querier_dto.SQLTypeCategory) *bool {

		if from == querier_dto.TypeCategoryBoolean && to == querier_dto.TypeCategoryInteger {
			return new(true)
		}
		return nil
	}))

	t.Run("hook allows normally disallowed cast", func(t *testing.T) {
		t.Parallel()

		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryBoolean, querier_dto.TypeCategoryInteger))
	})

	t.Run("hook falls through for default rules", func(t *testing.T) {
		t.Parallel()

		assert.True(t, engine.CanImplicitCast(querier_dto.TypeCategoryInteger, querier_dto.TypeCategoryFloat))
		assert.False(t, engine.CanImplicitCast(querier_dto.TypeCategoryFloat, querier_dto.TypeCategoryInteger))
	})
}

func TestSupportedExpressions(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()
	features := engine.SupportedExpressions()

	expectedFlags := []struct {
		name string
		flag querier_dto.SQLExpressionFeature
	}{
		{name: "binary arithmetic", flag: querier_dto.SQLFeatureBinaryArithmetic},
		{name: "binary comparison", flag: querier_dto.SQLFeatureBinaryComparison},
		{name: "string concat", flag: querier_dto.SQLFeatureStringConcat},
		{name: "case when", flag: querier_dto.SQLFeatureCaseWhen},
		{name: "scalar subquery", flag: querier_dto.SQLFeatureScalarSubquery},
		{name: "exists", flag: querier_dto.SQLFeatureExists},
		{name: "is null", flag: querier_dto.SQLFeatureIsNull},
		{name: "in list", flag: querier_dto.SQLFeatureInList},
		{name: "between", flag: querier_dto.SQLFeatureBetween},
		{name: "logical op", flag: querier_dto.SQLFeatureLogicalOp},
		{name: "unary op", flag: querier_dto.SQLFeatureUnaryOp},
		{name: "window function", flag: querier_dto.SQLFeatureWindowFunction},
		{name: "array subscript", flag: querier_dto.SQLFeatureArraySubscript},
		{name: "json op", flag: querier_dto.SQLFeatureJSONOp},
		{name: "pattern match", flag: querier_dto.SQLFeaturePatternMatch},
		{name: "bitwise op", flag: querier_dto.SQLFeatureBitwiseOp},
	}

	for _, tt := range expectedFlags {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.True(t, features.Has(tt.flag), "expected %s to be supported", tt.name)
		})
	}

	t.Run("lambda not supported", func(t *testing.T) {
		t.Parallel()
		assert.False(t, features.Has(querier_dto.SQLFeatureLambda), "lambda should not be supported")
	})

	t.Run("struct field access not supported", func(t *testing.T) {
		t.Parallel()
		assert.False(t, features.Has(querier_dto.SQLFeatureStructFieldAccess), "struct field access should not be supported")
	})
}

func TestParameterStyle(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	assert.Equal(t, querier_dto.ParameterStyleDollar, engine.ParameterStyle())
}

func TestDefaultSchema(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	assert.Equal(t, "public", engine.DefaultSchema())
}

func TestSupportsReturning(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	assert.True(t, engine.SupportsReturning())
}

func TestDialect(t *testing.T) {
	t.Parallel()

	t.Run("default dialect is postgres", func(t *testing.T) {
		t.Parallel()

		engine := NewPostgresEngine()
		assert.Equal(t, "postgres", engine.Dialect())
	})

	t.Run("custom dialect name", func(t *testing.T) {
		t.Parallel()

		engine := NewPostgresEngine(WithDialectName("yugabytedb"))
		assert.Equal(t, "yugabytedb", engine.Dialect())
	})
}

func TestBuiltinFunctions(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()
	catalogue := engine.BuiltinFunctions()

	require.NotNil(t, catalogue, "function catalogue should not be nil")
	require.NotNil(t, catalogue.Functions, "function map should not be nil")

	expectedFunctions := []string{
		"count",
		"array_agg",
		"string_agg",
		"bool_and",
		"bool_or",
		"concat",
		"concat_ws",
		"json_agg",
		"jsonb_agg",
		"json_build_object",
		"jsonb_build_object",
		"stddev",
		"variance",
	}

	for _, name := range expectedFunctions {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			signatures, exists := catalogue.Functions[name]
			assert.True(t, exists, "expected built-in function %q to exist", name)
			if exists {
				assert.NotEmpty(t, signatures, "expected at least one signature for %q", name)
			}
		})
	}
}

func TestBuiltinTypes(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()
	catalogue := engine.BuiltinTypes()

	require.NotNil(t, catalogue, "type catalogue should not be nil")
	require.NotNil(t, catalogue.Types, "type map should not be nil")

	expectedTypes := []struct {
		name     string
		category querier_dto.SQLTypeCategory
	}{
		{name: "int4", category: querier_dto.TypeCategoryInteger},
		{name: "int2", category: querier_dto.TypeCategoryInteger},
		{name: "int8", category: querier_dto.TypeCategoryInteger},
		{name: "float4", category: querier_dto.TypeCategoryFloat},
		{name: "float8", category: querier_dto.TypeCategoryFloat},
		{name: "numeric", category: querier_dto.TypeCategoryDecimal},
		{name: "bool", category: querier_dto.TypeCategoryBoolean},
		{name: "boolean", category: querier_dto.TypeCategoryBoolean},
		{name: "text", category: querier_dto.TypeCategoryText},
		{name: "varchar", category: querier_dto.TypeCategoryText},
		{name: "bytea", category: querier_dto.TypeCategoryBytea},
		{name: "timestamp", category: querier_dto.TypeCategoryTemporal},
		{name: "timestamptz", category: querier_dto.TypeCategoryTemporal},
		{name: "date", category: querier_dto.TypeCategoryTemporal},
		{name: "json", category: querier_dto.TypeCategoryJSON},
		{name: "jsonb", category: querier_dto.TypeCategoryJSON},
		{name: "uuid", category: querier_dto.TypeCategoryUUID},
		{name: "inet", category: querier_dto.TypeCategoryNetwork},
		{name: "int4range", category: querier_dto.TypeCategoryRange},
		{name: "point", category: querier_dto.TypeCategoryGeometric},
	}

	for _, tt := range expectedTypes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sqlType, exists := catalogue.Types[tt.name]
			require.True(t, exists, "expected type %q to exist in catalogue", tt.name)
			assert.Equal(t, tt.category, sqlType.Category, "category mismatch for type %q", tt.name)
		})
	}
}

func TestTableValuedFunctionColumns(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()

	t.Run("generate_series returns expected columns", func(t *testing.T) {
		t.Parallel()

		columns := engine.TableValuedFunctionColumns("generate_series")
		require.NotNil(t, columns, "generate_series should return columns")
		require.Len(t, columns, 1, "generate_series should have one column")
		assert.Equal(t, "generate_series", columns[0].Name)
		assert.Equal(t, querier_dto.TypeCategoryInteger, columns[0].SQLType.Category)
		assert.Equal(t, "int4", columns[0].SQLType.EngineName)
		assert.False(t, columns[0].Nullable)
	})

	t.Run("json_each returns key and value columns", func(t *testing.T) {
		t.Parallel()

		columns := engine.TableValuedFunctionColumns("json_each")
		require.NotNil(t, columns, "json_each should return columns")
		require.Len(t, columns, 2, "json_each should have two columns")

		assert.Equal(t, "key", columns[0].Name)
		assert.Equal(t, querier_dto.TypeCategoryText, columns[0].SQLType.Category)
		assert.False(t, columns[0].Nullable)

		assert.Equal(t, "value", columns[1].Name)
		assert.Equal(t, querier_dto.TypeCategoryJSON, columns[1].SQLType.Category)
		assert.True(t, columns[1].Nullable)
	})

	t.Run("unnest returns unknown-type column", func(t *testing.T) {
		t.Parallel()

		columns := engine.TableValuedFunctionColumns("unnest")
		require.NotNil(t, columns, "unnest should return columns")
		require.Len(t, columns, 1)
		assert.Equal(t, querier_dto.TypeCategoryUnknown, columns[0].SQLType.Category)
	})

	t.Run("unknown function returns nil", func(t *testing.T) {
		t.Parallel()

		columns := engine.TableValuedFunctionColumns("nonexistent_function")
		assert.Nil(t, columns, "unknown function should return nil")
	})

	t.Run("returned slice is a copy", func(t *testing.T) {
		t.Parallel()

		columns1 := engine.TableValuedFunctionColumns("generate_series")
		columns2 := engine.TableValuedFunctionColumns("generate_series")

		require.NotNil(t, columns1)
		require.NotNil(t, columns2)

		columns1[0].Name = "modified"
		assert.NotEqual(t, columns1[0].Name, columns2[0].Name,
			"returned slices should be independent copies")
	})
}

func TestCommentStyle(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()
	style := engine.CommentStyle()

	assert.Equal(t, "--", style.LinePrefix, "comment style should use standard SQL line prefix")
}

func TestSupportedDirectivePrefixes(t *testing.T) {
	t.Parallel()

	engine := NewPostgresEngine()
	prefixes := engine.SupportedDirectivePrefixes()

	require.Len(t, prefixes, 2, "PostgreSQL should support two directive prefixes")

	assert.Equal(t, byte('$'), prefixes[0].Prefix)
	assert.False(t, prefixes[0].IsNamed, "dollar prefix is positional, not named")

	assert.Equal(t, byte(':'), prefixes[1].Prefix)
	assert.True(t, prefixes[1].IsNamed, "colon prefix is named")
}

func TestIntegerPromotionRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		engineName string
		wantRank   int
	}{
		{name: "int2 has lowest rank", engineName: "int2", wantRank: 1},
		{name: "int4 has middle rank", engineName: "int4", wantRank: 2},
		{name: "int8 has highest rank", engineName: "int8", wantRank: integerPromotionRankWidest},
		{name: "unknown defaults to middle rank", engineName: "oid", wantRank: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := integerPromotionRank(tt.engineName)
			assert.Equal(t, tt.wantRank, got)
		})
	}
}

func TestFloatPromotionRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		engineName string
		wantRank   int
	}{
		{name: "float4 has lowest rank", engineName: "float4", wantRank: 1},
		{name: "float8 has highest rank", engineName: "float8", wantRank: 2},
		{name: "unknown defaults to highest rank", engineName: "something", wantRank: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := floatPromotionRank(tt.engineName)
			assert.Equal(t, tt.wantRank, got)
		})
	}
}
