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
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// goPackageTime holds the Go import path for the time package.
	goPackageTime = "time"

	// goTypeTime holds the Go type name for time.Time.
	goTypeTime = "Time"

	// goTypeTimePointer holds the Go type name for a pointer to time.Time.
	goTypeTimePointer = "*Time"
)

// TypeMapper maps SQL types to Go types using a structured category-based
// approach with a hierarchical lookup: first by exact engine name within a
// category, then by category alone.
type TypeMapper struct {
	// typeCatalogue holds the engine-specific type catalogue used to resolve
	// SQL type names to their categories.
	typeCatalogue *querier_dto.TypeCatalogue
}

// NewTypeMapper creates a new type mapper with the given type catalogue.
//
// Takes typeCatalogue (*querier_dto.TypeCatalogue) which holds the
// engine-specific SQL type definitions.
//
// Returns *TypeMapper which is ready to build mapping tables and resolve
// SQL-to-Go type mappings.
func NewTypeMapper(typeCatalogue *querier_dto.TypeCatalogue) *TypeMapper {
	return &TypeMapper{
		typeCatalogue: typeCatalogue,
	}
}

// BuildMappingTable creates a complete type mapping table by combining the
// framework defaults with user-provided overrides. User overrides take
// precedence over framework defaults.
//
// Takes overrides ([]querier_dto.TypeOverride) which holds user-provided
// SQL-to-Go type override definitions.
//
// Returns *querier_dto.TypeMappingTable which holds the combined mapping table
// with defaults followed by overrides.
func (m *TypeMapper) BuildMappingTable(
	overrides []querier_dto.TypeOverride,
) *querier_dto.TypeMappingTable {
	mappings := defaultMappings()

	for _, override := range overrides {
		sqlType := m.resolveOverrideType(override)
		goType := querier_dto.GoType{
			Package: override.GoPackage,
			Name:    override.GoName,
		}

		mappings = append(mappings, querier_dto.TypeMapping{
			SQLCategory: sqlType.Category,
			SQLName:     sqlType.EngineName,
			NotNull:     goType,
			Nullable:    querier_dto.GoType{Package: goType.Package, Name: "*" + goType.Name},
		})
	}

	return &querier_dto.TypeMappingTable{
		Mappings: mappings,
	}
}

// MapType maps a SQL type to its corresponding Go type based on the mapping
// table.
//
// Matches by exact engine name first, then falls back to category-only
// matching. Later entries in the table override earlier ones.
//
// Takes sqlType (querier_dto.SQLType) which is the SQL type to map.
// Takes nullable (bool) which indicates whether the column allows NULL values.
// Takes mappings (*querier_dto.TypeMappingTable) which holds the mapping table
// to search.
//
// Returns querier_dto.GoType which is the resolved Go type for the given SQL
// type and nullability.
func (*TypeMapper) MapType(
	sqlType querier_dto.SQLType,
	nullable bool,
	mappings *querier_dto.TypeMappingTable,
) querier_dto.GoType {
	var categoryMatch *querier_dto.TypeMapping
	var exactMatch *querier_dto.TypeMapping

	for i := len(mappings.Mappings) - 1; i >= 0; i-- {
		mapping := &mappings.Mappings[i]
		if mapping.SQLCategory != sqlType.Category {
			continue
		}

		if mapping.SQLName != "" && strings.EqualFold(mapping.SQLName, sqlType.EngineName) {
			exactMatch = mapping
			break
		}

		if mapping.SQLName == "" && categoryMatch == nil {
			categoryMatch = mapping
		}
	}

	chosen := exactMatch
	if chosen == nil {
		chosen = categoryMatch
	}

	if chosen == nil {
		if nullable {
			return querier_dto.GoType{Name: "any"}
		}
		return querier_dto.GoType{Name: "any"}
	}

	if nullable {
		return chosen.Nullable
	}
	return chosen.NotNull
}

// resolveOverrideType resolves a user-provided type override to its SQL type
// by looking up the type name in the catalogue, falling back to an unknown
// category if not found.
//
// Takes override (querier_dto.TypeOverride) which holds the SQL type name
// to resolve.
//
// Returns querier_dto.SQLType which is the resolved SQL type with its category
// and engine name.
func (m *TypeMapper) resolveOverrideType(override querier_dto.TypeOverride) querier_dto.SQLType {
	if m.typeCatalogue != nil {
		if sqlType, exists := m.typeCatalogue.Types[strings.ToLower(override.SQLTypeName)]; exists {
			return sqlType
		}
	}

	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryUnknown,
		EngineName: override.SQLTypeName,
	}
}

// defaultMappings returns the framework-owned default SQL-to-Go type mapping
// table. These mappings use Piko's maths types for precise numeric handling
// (Decimal, Money, BigInt) rather than the lossy float64 conversions used by
// most code generators.
//
// Returns []querier_dto.TypeMapping which holds the default mapping entries
// covering numeric, scalar, temporal, and complex types.
func defaultMappings() []querier_dto.TypeMapping {
	return slices.Concat(
		numericMappings(),
		scalarMappings(),
		temporalMappings(),
		complexMappings(),
	)
}

// numericMappings returns the default mappings for integer, float, and decimal
// SQL types.
//
// Returns []querier_dto.TypeMapping which holds the numeric type mapping
// entries.
func numericMappings() []querier_dto.TypeMapping {
	return []querier_dto.TypeMapping{
		{SQLCategory: querier_dto.TypeCategoryInteger, SQLName: querier_dto.CanonicalInt2, NotNull: querier_dto.GoType{Name: "int16"}, Nullable: querier_dto.GoType{Name: "*int16"}},
		{SQLCategory: querier_dto.TypeCategoryInteger, SQLName: querier_dto.CanonicalInt4, NotNull: querier_dto.GoType{Name: "int32"}, Nullable: querier_dto.GoType{Name: "*int32"}},
		{SQLCategory: querier_dto.TypeCategoryInteger, SQLName: querier_dto.CanonicalInt8, NotNull: querier_dto.GoType{Name: "int64"}, Nullable: querier_dto.GoType{Name: "*int64"}},
		{SQLCategory: querier_dto.TypeCategoryInteger, NotNull: querier_dto.GoType{Name: "int32"}, Nullable: querier_dto.GoType{Name: "*int32"}},
		{SQLCategory: querier_dto.TypeCategoryFloat, SQLName: querier_dto.CanonicalFloat4, NotNull: querier_dto.GoType{Name: "float32"}, Nullable: querier_dto.GoType{Name: "*float32"}},
		{SQLCategory: querier_dto.TypeCategoryFloat, SQLName: querier_dto.CanonicalFloat8, NotNull: querier_dto.GoType{Name: "float64"}, Nullable: querier_dto.GoType{Name: "*float64"}},
		{SQLCategory: querier_dto.TypeCategoryFloat, NotNull: querier_dto.GoType{Name: "float64"}, Nullable: querier_dto.GoType{Name: "*float64"}},
		{
			SQLCategory: querier_dto.TypeCategoryDecimal,
			NotNull:     querier_dto.GoType{Package: "piko.sh/piko/wdk/maths", Name: "Decimal"},
			Nullable:    querier_dto.GoType{Package: "piko.sh/piko/wdk/maths", Name: "*Decimal"},
		},
	}
}

// scalarMappings returns the default mappings for boolean, text, and bytea
// SQL types.
//
// Returns []querier_dto.TypeMapping which holds the scalar type mapping
// entries.
func scalarMappings() []querier_dto.TypeMapping {
	return []querier_dto.TypeMapping{
		{SQLCategory: querier_dto.TypeCategoryBoolean, NotNull: querier_dto.GoType{Name: "bool"}, Nullable: querier_dto.GoType{Name: "*bool"}},
		{SQLCategory: querier_dto.TypeCategoryText, NotNull: querier_dto.GoType{Name: "string"}, Nullable: querier_dto.GoType{Name: "*string"}},
		{SQLCategory: querier_dto.TypeCategoryBytea, NotNull: querier_dto.GoType{Name: "[]byte"}, Nullable: querier_dto.GoType{Name: "[]byte"}},
	}
}

// temporalMappings returns the default mappings for timestamp, date, time, and
// interval SQL types.
//
// Returns []querier_dto.TypeMapping which holds the temporal type mapping
// entries.
func temporalMappings() []querier_dto.TypeMapping {
	timeType := querier_dto.GoType{Package: goPackageTime, Name: goTypeTime}
	timePointer := querier_dto.GoType{Package: goPackageTime, Name: goTypeTimePointer}
	return []querier_dto.TypeMapping{
		{SQLCategory: querier_dto.TypeCategoryTemporal, SQLName: querier_dto.CanonicalTimestampTZ, NotNull: timeType, Nullable: timePointer},
		{SQLCategory: querier_dto.TypeCategoryTemporal, SQLName: querier_dto.CanonicalTimestamp, NotNull: timeType, Nullable: timePointer},
		{SQLCategory: querier_dto.TypeCategoryTemporal, SQLName: querier_dto.CanonicalDate, NotNull: timeType, Nullable: timePointer},
		{SQLCategory: querier_dto.TypeCategoryTemporal, SQLName: querier_dto.CanonicalTime, NotNull: timeType, Nullable: timePointer},
		{
			SQLCategory: querier_dto.TypeCategoryTemporal,
			SQLName:     querier_dto.CanonicalInterval,
			NotNull:     querier_dto.GoType{Package: goPackageTime, Name: "Duration"},
			Nullable:    querier_dto.GoType{Package: goPackageTime, Name: "*Duration"},
		},
		{SQLCategory: querier_dto.TypeCategoryTemporal, NotNull: timeType, Nullable: timePointer},
	}
}

// complexMappings returns the default mappings for JSON, UUID, enum, array,
// struct, map, and union SQL types.
//
// Returns []querier_dto.TypeMapping which holds the complex type mapping
// entries.
func complexMappings() []querier_dto.TypeMapping {
	goTypeAny := querier_dto.GoType{Name: "any"}
	goTypeAnySlice := querier_dto.GoType{Name: "[]any"}

	return []querier_dto.TypeMapping{
		{
			SQLCategory: querier_dto.TypeCategoryJSON,
			NotNull:     querier_dto.GoType{Package: "encoding/json", Name: "RawMessage"},
			Nullable:    querier_dto.GoType{Package: "encoding/json", Name: "RawMessage"},
		},
		{
			SQLCategory: querier_dto.TypeCategoryUUID,
			NotNull:     querier_dto.GoType{Package: "github.com/google/uuid", Name: "UUID"},
			Nullable:    querier_dto.GoType{Package: "github.com/google/uuid", Name: "*UUID"},
		},
		{SQLCategory: querier_dto.TypeCategoryEnum, NotNull: querier_dto.GoType{Name: "string"}, Nullable: querier_dto.GoType{Name: "*string"}},
		{SQLCategory: querier_dto.TypeCategoryArray, NotNull: goTypeAnySlice, Nullable: goTypeAnySlice},
		{SQLCategory: querier_dto.TypeCategoryStruct, NotNull: goTypeAny, Nullable: goTypeAny},
		{SQLCategory: querier_dto.TypeCategoryMap, NotNull: goTypeAny, Nullable: goTypeAny},
		{SQLCategory: querier_dto.TypeCategoryUnion, NotNull: goTypeAny, Nullable: goTypeAny},
	}
}
