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

func TestNewTypeMapper(t *testing.T) {
	t.Parallel()

	t.Run("creates non-nil mapper with catalogue", func(t *testing.T) {
		t.Parallel()

		catalogue := &querier_dto.TypeCatalogue{
			Types: map[string]querier_dto.SQLType{
				"int4": {Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt4},
			},
		}

		mapper := NewTypeMapper(catalogue)

		require.NotNil(t, mapper)
		assert.Equal(t, catalogue, mapper.typeCatalogue)
	})

	t.Run("creates non-nil mapper with nil catalogue", func(t *testing.T) {
		t.Parallel()

		mapper := NewTypeMapper(nil)

		require.NotNil(t, mapper)
		assert.Nil(t, mapper.typeCatalogue)
	})
}

func TestBuildMappingTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		catalogue        *querier_dto.TypeCatalogue
		overrides        []querier_dto.TypeOverride
		wantMinMappings  int
		wantLastSQLName  string
		wantLastNotNull  querier_dto.GoType
		wantLastNullable querier_dto.GoType
		wantLastCategory querier_dto.SQLTypeCategory
	}{
		{
			name:      "no overrides returns default mappings",
			catalogue: nil,
			overrides: nil,

			wantMinMappings: 10,
		},
		{
			name:      "with override appends user mapping to table",
			catalogue: nil,
			overrides: []querier_dto.TypeOverride{
				{
					SQLTypeName: "custom_money",
					GoPackage:   "example.com/types",
					GoName:      "Money",
				},
			},
			wantMinMappings:  11,
			wantLastSQLName:  "custom_money",
			wantLastNotNull:  querier_dto.GoType{Package: "example.com/types", Name: "Money"},
			wantLastNullable: querier_dto.GoType{Package: "example.com/types", Name: "*Money"},
			wantLastCategory: querier_dto.TypeCategoryUnknown,
		},
		{
			name: "override resolves type from catalogue when name is known",
			catalogue: &querier_dto.TypeCatalogue{
				Types: map[string]querier_dto.SQLType{
					"citext": {
						Category:   querier_dto.TypeCategoryText,
						EngineName: "citext",
					},
				},
			},
			overrides: []querier_dto.TypeOverride{
				{
					SQLTypeName: "citext",
					GoPackage:   "example.com/ci",
					GoName:      "CIString",
				},
			},
			wantMinMappings:  11,
			wantLastSQLName:  "citext",
			wantLastNotNull:  querier_dto.GoType{Package: "example.com/ci", Name: "CIString"},
			wantLastNullable: querier_dto.GoType{Package: "example.com/ci", Name: "*CIString"},
			wantLastCategory: querier_dto.TypeCategoryText,
		},
		{
			name: "override with unknown catalogue type falls back to TypeCategoryUnknown",
			catalogue: &querier_dto.TypeCatalogue{
				Types: map[string]querier_dto.SQLType{
					"int4": {Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt4},
				},
			},
			overrides: []querier_dto.TypeOverride{
				{
					SQLTypeName: "not_in_catalogue",
					GoPackage:   "example.com/custom",
					GoName:      "Widget",
				},
			},
			wantMinMappings:  11,
			wantLastSQLName:  "not_in_catalogue",
			wantLastNotNull:  querier_dto.GoType{Package: "example.com/custom", Name: "Widget"},
			wantLastNullable: querier_dto.GoType{Package: "example.com/custom", Name: "*Widget"},
			wantLastCategory: querier_dto.TypeCategoryUnknown,
		},
		{
			name:      "multiple overrides are all appended in order",
			catalogue: nil,
			overrides: []querier_dto.TypeOverride{
				{SQLTypeName: "first_type", GoPackage: "example.com/a", GoName: "First"},
				{SQLTypeName: "second_type", GoPackage: "example.com/b", GoName: "Second"},
			},
			wantMinMappings:  12,
			wantLastSQLName:  "second_type",
			wantLastNotNull:  querier_dto.GoType{Package: "example.com/b", Name: "Second"},
			wantLastNullable: querier_dto.GoType{Package: "example.com/b", Name: "*Second"},
			wantLastCategory: querier_dto.TypeCategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mapper := NewTypeMapper(tt.catalogue)
			table := mapper.BuildMappingTable(tt.overrides)

			require.NotNil(t, table)
			assert.GreaterOrEqual(t, len(table.Mappings), tt.wantMinMappings,
				"expected at least %d mappings, got %d", tt.wantMinMappings, len(table.Mappings))

			if tt.wantLastSQLName != "" {
				last := table.Mappings[len(table.Mappings)-1]
				assert.Equal(t, tt.wantLastSQLName, last.SQLName)
				assert.Equal(t, tt.wantLastNotNull, last.NotNull)
				assert.Equal(t, tt.wantLastNullable, last.Nullable)
				assert.Equal(t, tt.wantLastCategory, last.SQLCategory)
			}
		})
	}
}

func TestMapType(t *testing.T) {
	t.Parallel()

	mapper := NewTypeMapper(nil)
	mappings := mapper.BuildMappingTable(nil)

	tests := []struct {
		name       string
		sqlType    querier_dto.SQLType
		nullable   bool
		mappings   *querier_dto.TypeMappingTable
		wantGoType querier_dto.GoType
	}{

		{
			name:       "int2 not null maps to int16",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt2},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "int16"},
		},
		{
			name:       "int4 not null maps to int32",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt4},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "int32"},
		},
		{
			name:       "int8 not null maps to int64",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt8},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "int64"},
		},

		{
			name:       "int2 nullable maps to pointer int16",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt2},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "*int16"},
		},
		{
			name:       "int4 nullable maps to pointer int32",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt4},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "*int32"},
		},
		{
			name:       "int8 nullable maps to pointer int64",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt8},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "*int64"},
		},

		{
			name:       "float4 not null maps to float32",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: querier_dto.CanonicalFloat4},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "float32"},
		},
		{
			name:       "float8 not null maps to float64",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: querier_dto.CanonicalFloat8},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "float64"},
		},

		{
			name:       "float4 nullable maps to pointer float32",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: querier_dto.CanonicalFloat4},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "*float32"},
		},
		{
			name:       "float8 nullable maps to pointer float64",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: querier_dto.CanonicalFloat8},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "*float64"},
		},

		{
			name:       "boolean not null maps to bool",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: querier_dto.CanonicalBoolean},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "bool"},
		},
		{
			name:       "boolean nullable maps to pointer bool",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean, EngineName: querier_dto.CanonicalBoolean},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "*bool"},
		},

		{
			name:       "text not null maps to string",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: querier_dto.CanonicalText},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "string"},
		},
		{
			name:       "varchar not null maps to string via category fallback",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: querier_dto.CanonicalVarchar},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "string"},
		},
		{
			name:       "text nullable maps to pointer string",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: querier_dto.CanonicalText},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "*string"},
		},

		{
			name:       "timestamptz not null maps to time.Time",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: querier_dto.CanonicalTimestampTZ},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "time", Name: "Time"},
		},
		{
			name:       "timestamp not null maps to time.Time",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: querier_dto.CanonicalTimestamp},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "time", Name: "Time"},
		},
		{
			name:       "date not null maps to time.Time",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: querier_dto.CanonicalDate},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "time", Name: "Time"},
		},
		{
			name:       "time not null maps to time.Time",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: querier_dto.CanonicalTime},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "time", Name: "Time"},
		},
		{
			name:       "interval not null maps to time.Duration",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: querier_dto.CanonicalInterval},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "time", Name: "Duration"},
		},
		{
			name:       "timestamptz nullable maps to pointer time.Time",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: querier_dto.CanonicalTimestampTZ},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "time", Name: "*Time"},
		},
		{
			name:       "interval nullable maps to pointer time.Duration",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: querier_dto.CanonicalInterval},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "time", Name: "*Duration"},
		},

		{
			name:       "json not null maps to json.RawMessage",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "encoding/json", Name: "RawMessage"},
		},
		{
			name:       "jsonb not null maps to json.RawMessage",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "jsonb"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "encoding/json", Name: "RawMessage"},
		},
		{
			name:       "json nullable also maps to json.RawMessage (not pointer)",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON, EngineName: "json"},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "encoding/json", Name: "RawMessage"},
		},

		{
			name:       "uuid not null maps to uuid.UUID",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryUUID, EngineName: "uuid"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "github.com/google/uuid", Name: "UUID"},
		},
		{
			name:       "uuid nullable maps to pointer uuid.UUID",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryUUID, EngineName: "uuid"},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "github.com/google/uuid", Name: "*UUID"},
		},

		{
			name:       "bytea not null maps to byte slice",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "bytea"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "[]byte"},
		},
		{
			name:       "bytea nullable maps to byte slice (no pointer for slices)",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "bytea"},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "[]byte"},
		},

		{
			name:       "decimal not null maps to maths.Decimal",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: querier_dto.CanonicalNumeric},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "piko.sh/piko/wdk/maths", Name: "Decimal"},
		},
		{
			name:       "decimal nullable maps to pointer maths.Decimal",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryDecimal, EngineName: querier_dto.CanonicalNumeric},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "piko.sh/piko/wdk/maths", Name: "*Decimal"},
		},

		{
			name:       "enum not null maps to string",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryEnum, EngineName: "user_status"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "string"},
		},
		{
			name:       "enum nullable maps to pointer string",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryEnum, EngineName: "user_status"},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "*string"},
		},

		{
			name:       "array not null maps to any slice",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryArray, EngineName: "int4[]"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "[]any"},
		},

		{
			name:       "struct not null maps to any",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryStruct, EngineName: "struct"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "any"},
		},

		{
			name:       "map not null maps to any",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryMap, EngineName: "map"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "any"},
		},

		{
			name:       "union not null maps to any",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryUnion, EngineName: "union"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "any"},
		},

		{
			name:       "unknown type not null returns any",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "completely_unknown"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "any"},
		},
		{
			name:       "unknown type nullable returns any",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "completely_unknown"},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "any"},
		},

		{
			name:       "integer category with unknown engine name falls back to category default",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "mediumint"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "int32"},
		},
		{
			name:       "integer category with unknown engine name nullable falls back to category default",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "mediumint"},
			nullable:   true,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "*int32"},
		},

		{
			name:       "float category with unknown engine name falls back to float64",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "custom_float"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "float64"},
		},

		{
			name:       "temporal category with unknown engine name falls back to time.Time",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal, EngineName: "datetime2"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Package: "time", Name: "Time"},
		},

		{
			name:       "engine name matching is case-insensitive",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "INT4"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "int32"},
		},
		{
			name:       "engine name matching is case-insensitive mixed case",
			sqlType:    querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat, EngineName: "Float8"},
			nullable:   false,
			mappings:   mappings,
			wantGoType: querier_dto.GoType{Name: "float64"},
		},

		{
			name:     "category with no default mapping and no exact match returns any",
			sqlType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryNetwork, EngineName: "inet"},
			nullable: false,
			mappings: mappings,

			wantGoType: querier_dto.GoType{Name: "any"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapper.MapType(tt.sqlType, tt.nullable, tt.mappings)
			assert.Equal(t, tt.wantGoType, got)
		})
	}
}

func TestMapType_OverridesTakePrecedence(t *testing.T) {
	t.Parallel()

	catalogue := &querier_dto.TypeCatalogue{
		Types: map[string]querier_dto.SQLType{
			"int4": {Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt4},
		},
	}
	mapper := NewTypeMapper(catalogue)

	overrides := []querier_dto.TypeOverride{
		{
			SQLTypeName: "int4",
			GoPackage:   "example.com/custom",
			GoName:      "CustomInt",
		},
	}
	mappings := mapper.BuildMappingTable(overrides)

	got := mapper.MapType(
		querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt4},
		false,
		mappings,
	)
	assert.Equal(t, querier_dto.GoType{Package: "example.com/custom", Name: "CustomInt"}, got)

	gotNullable := mapper.MapType(
		querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt4},
		true,
		mappings,
	)
	assert.Equal(t, querier_dto.GoType{Package: "example.com/custom", Name: "*CustomInt"}, gotNullable)
}

func TestMapType_EmptyMappingTable(t *testing.T) {
	t.Parallel()

	mapper := NewTypeMapper(nil)
	emptyTable := &querier_dto.TypeMappingTable{
		Mappings: nil,
	}

	t.Run("not null with empty table returns any", func(t *testing.T) {
		t.Parallel()

		got := mapper.MapType(
			querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: querier_dto.CanonicalInt4},
			false,
			emptyTable,
		)
		assert.Equal(t, querier_dto.GoType{Name: "any"}, got)
	})

	t.Run("nullable with empty table returns any", func(t *testing.T) {
		t.Parallel()

		got := mapper.MapType(
			querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
			true,
			emptyTable,
		)
		assert.Equal(t, querier_dto.GoType{Name: "any"}, got)
	})
}

func TestDefaultMappings(t *testing.T) {
	t.Parallel()

	mappings := defaultMappings()

	categories := make(map[querier_dto.SQLTypeCategory]bool)
	for _, m := range mappings {
		categories[m.SQLCategory] = true
	}

	expectedCategories := []struct {
		category querier_dto.SQLTypeCategory
		label    string
	}{
		{category: querier_dto.TypeCategoryInteger, label: "integer"},
		{category: querier_dto.TypeCategoryFloat, label: "float"},
		{category: querier_dto.TypeCategoryDecimal, label: "decimal"},
		{category: querier_dto.TypeCategoryBoolean, label: "boolean"},
		{category: querier_dto.TypeCategoryText, label: "text"},
		{category: querier_dto.TypeCategoryBytea, label: "bytea"},
		{category: querier_dto.TypeCategoryTemporal, label: "temporal"},
		{category: querier_dto.TypeCategoryJSON, label: "json"},
		{category: querier_dto.TypeCategoryUUID, label: "uuid"},
		{category: querier_dto.TypeCategoryEnum, label: "enum"},
		{category: querier_dto.TypeCategoryArray, label: "array"},
		{category: querier_dto.TypeCategoryStruct, label: "struct"},
		{category: querier_dto.TypeCategoryMap, label: "map"},
		{category: querier_dto.TypeCategoryUnion, label: "union"},
	}

	for _, ec := range expectedCategories {
		t.Run("has mapping for "+ec.label+" category", func(t *testing.T) {
			t.Parallel()

			assert.True(t, categories[ec.category],
				"expected default mappings to include category %s", ec.label)
		})
	}
}

func TestDefaultMappings_IntegerVariants(t *testing.T) {
	t.Parallel()

	mappings := defaultMappings()

	intNames := make(map[string]bool)
	for _, m := range mappings {
		if m.SQLCategory == querier_dto.TypeCategoryInteger && m.SQLName != "" {
			intNames[m.SQLName] = true
		}
	}

	assert.True(t, intNames[querier_dto.CanonicalInt2], "expected int2 mapping")
	assert.True(t, intNames[querier_dto.CanonicalInt4], "expected int4 mapping")
	assert.True(t, intNames[querier_dto.CanonicalInt8], "expected int8 mapping")
}

func TestDefaultMappings_FloatVariants(t *testing.T) {
	t.Parallel()

	mappings := defaultMappings()

	floatNames := make(map[string]bool)
	for _, m := range mappings {
		if m.SQLCategory == querier_dto.TypeCategoryFloat && m.SQLName != "" {
			floatNames[m.SQLName] = true
		}
	}

	assert.True(t, floatNames[querier_dto.CanonicalFloat4], "expected float4 mapping")
	assert.True(t, floatNames[querier_dto.CanonicalFloat8], "expected float8 mapping")
}

func TestDefaultMappings_TemporalVariants(t *testing.T) {
	t.Parallel()

	mappings := defaultMappings()

	temporalNames := make(map[string]bool)
	for _, m := range mappings {
		if m.SQLCategory == querier_dto.TypeCategoryTemporal && m.SQLName != "" {
			temporalNames[m.SQLName] = true
		}
	}

	assert.True(t, temporalNames[querier_dto.CanonicalTimestampTZ], "expected timestamptz mapping")
	assert.True(t, temporalNames[querier_dto.CanonicalTimestamp], "expected timestamp mapping")
	assert.True(t, temporalNames[querier_dto.CanonicalDate], "expected date mapping")
	assert.True(t, temporalNames[querier_dto.CanonicalTime], "expected time mapping")
	assert.True(t, temporalNames[querier_dto.CanonicalInterval], "expected interval mapping")
}

func TestDefaultMappings_CategoryFallbackEntries(t *testing.T) {
	t.Parallel()

	mappings := defaultMappings()

	categoriesWithFallback := make(map[querier_dto.SQLTypeCategory]bool)
	for _, m := range mappings {
		if m.SQLName == "" {
			categoriesWithFallback[m.SQLCategory] = true
		}
	}

	assert.True(t, categoriesWithFallback[querier_dto.TypeCategoryInteger],
		"expected integer category fallback entry")
	assert.True(t, categoriesWithFallback[querier_dto.TypeCategoryFloat],
		"expected float category fallback entry")
	assert.True(t, categoriesWithFallback[querier_dto.TypeCategoryTemporal],
		"expected temporal category fallback entry")
}

func TestResolveOverrideType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		catalogue *querier_dto.TypeCatalogue
		override  querier_dto.TypeOverride
		want      querier_dto.SQLType
	}{
		{
			name: "resolves known type from catalogue",
			catalogue: &querier_dto.TypeCatalogue{
				Types: map[string]querier_dto.SQLType{
					"citext": {Category: querier_dto.TypeCategoryText, EngineName: "citext"},
				},
			},
			override: querier_dto.TypeOverride{SQLTypeName: "citext", GoPackage: "x", GoName: "X"},
			want:     querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "citext"},
		},
		{
			name: "case-folds to lower when looking up catalogue",
			catalogue: &querier_dto.TypeCatalogue{
				Types: map[string]querier_dto.SQLType{
					"citext": {Category: querier_dto.TypeCategoryText, EngineName: "citext"},
				},
			},
			override: querier_dto.TypeOverride{SQLTypeName: "CITEXT", GoPackage: "x", GoName: "X"},
			want:     querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "citext"},
		},
		{
			name: "unknown type name falls back to unknown category",
			catalogue: &querier_dto.TypeCatalogue{
				Types: map[string]querier_dto.SQLType{},
			},
			override: querier_dto.TypeOverride{SQLTypeName: "mystery", GoPackage: "x", GoName: "X"},
			want:     querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "mystery"},
		},
		{
			name:      "nil catalogue falls back to unknown category",
			catalogue: nil,
			override:  querier_dto.TypeOverride{SQLTypeName: "anything", GoPackage: "x", GoName: "X"},
			want:      querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "anything"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mapper := NewTypeMapper(tt.catalogue)
			got := mapper.resolveOverrideType(tt.override)
			assert.Equal(t, tt.want, got)
		})
	}
}
