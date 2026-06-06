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

	// goPackageDBJSON holds the Go import path for the shared dbjson package. Used by the
	// JSON / Union mappings whose Go destination is the nil-capable dbjson.JSON scanner.
	goPackageDBJSON = "piko.sh/piko/wdk/db/dbjson"

	// goTypeDBJSON holds the Go type name for dbjson.JSON.
	goTypeDBJSON = "JSON"

	// goTypeByteSlice holds the Go type name for a slice of bytes. Used by the Bytea and
	// AggregateState mappings whose destination is a raw byte buffer.
	goTypeByteSlice = "[]byte"

	// goPackageMaths holds the Go import path for Piko's maths package, the destination for
	// precise numeric types (Decimal, Money, BigInt) that have no native fixed-width Go
	// type.
	goPackageMaths = "piko.sh/piko/wdk/maths"

	// goTypeInt32 is the Go type name for a signed 32-bit integer, shared across the engine
	// integer mappings so the width string is not repeated.
	goTypeInt32 = "int32"

	// goTypeUint32 is the Go type name for an unsigned 32-bit integer, shared across the
	// engine integer mappings so the width string is not repeated.
	goTypeUint32 = "uint32"

	// maxArrayDimensions caps the number of `[]` suffixes resolveOverrideType will unwrap on
	// a user-supplied type override. Matches the Postgres parser's own limit so a malicious
	// override like `text[][][]...[]` cannot cause unbounded recursion or allocation, and so
	// the array recursion in scope_chain.arrayWrappedSQLType has a shared bound.
	maxArrayDimensions = 6
)

// TypeMapper maps SQL types to Go types using a structured category-based approach with a
// hierarchical lookup: first by exact engine name within a category, then by category
// alone.
type TypeMapper struct {
	// typeCatalogue holds the engine-specific type catalogue used to resolve SQL type names
	// to their categories.
	typeCatalogue *querier_dto.TypeCatalogue
}

// NewTypeMapper creates a new type mapper with the given type catalogue.
//
// Takes typeCatalogue (*querier_dto.TypeCatalogue) which holds the engine-specific SQL
// type definitions.
//
// Returns *TypeMapper which is ready to build mapping tables and resolve SQL-to-Go type
// mappings.
func NewTypeMapper(typeCatalogue *querier_dto.TypeCatalogue) *TypeMapper {
	return &TypeMapper{
		typeCatalogue: typeCatalogue,
	}
}

// BuildMappingTable creates a complete type mapping table by combining the framework
// defaults with user-provided overrides. User overrides take precedence over framework
// defaults.
//
// Takes overrides ([]querier_dto.TypeOverride) which holds user-provided SQL-to-Go type
// override definitions.
//
// Returns *querier_dto.TypeMappingTable which holds the combined mapping table with
// defaults followed by overrides.
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

// resolveOverrideType resolves a user-provided type override to its SQL type by looking
// up the type name in the catalogue, falling back to an unknown category if not found.
//
// Takes override (querier_dto.TypeOverride) which holds the SQL type name to resolve.
//
// Returns querier_dto.SQLType which is the resolved SQL type with its category and engine
// name.
func (m *TypeMapper) resolveOverrideType(override querier_dto.TypeOverride) querier_dto.SQLType {
	return m.resolveOverrideTypeDepth(override, 0)
}

// resolveOverrideTypeDepth performs the array-suffix recursion with a depth counter so a
// pathological override cannot allocate unboundedly.
//
// Once depth reaches maxArrayDimensions the function stops unwrapping further `[]`
// suffixes and returns an unknown type instead.
//
// Takes override (querier_dto.TypeOverride) which carries the SQL type name to resolve.
// Takes depth (int) which is the current array-unwrap recursion depth.
//
// Returns querier_dto.SQLType which is the resolved type, or the unknown category once
// the depth limit is reached.
func (m *TypeMapper) resolveOverrideTypeDepth(override querier_dto.TypeOverride, depth int) querier_dto.SQLType {
	lower := strings.ToLower(override.SQLTypeName)

	if m.typeCatalogue != nil {
		if sqlType, exists := m.typeCatalogue.Types[lower]; exists {
			return sqlType
		}
	}

	if depth < maxArrayDimensions {
		if stripped, found := strings.CutSuffix(lower, "[]"); found {
			return querier_dto.SQLType{
				Category:    querier_dto.TypeCategoryArray,
				EngineName:  lower,
				ElementType: new(m.resolveOverrideTypeDepth(querier_dto.TypeOverride{SQLTypeName: stripped}, depth+1)),
			}
		}
	}

	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryUnknown,
		EngineName: override.SQLTypeName,
	}
}

// defaultMappings returns the framework-owned default SQL-to-Go type mapping table. These
// mappings use Piko's maths types for precise numeric handling (Decimal, Money, BigInt)
// rather than the lossy float64 conversions used by most code generators.
//
// Returns []querier_dto.TypeMapping which holds the default mapping entries covering
// numeric, scalar, temporal, and complex types.
func defaultMappings() []querier_dto.TypeMapping {
	return slices.Concat(
		numericMappings(),
		clickhouseIntegerMappings(),
		mysqlIntegerMappings(),
		duckdbIntegerMappings(),
		scalarMappings(),
		temporalMappings(),
		complexMappings(),
	)
}

// integerMapping returns a fixed-width integer mapping from an engine SQL name to a Go
// type, with the nullable form as the pointer to that type.
//
// Takes sqlName (string) which is the engine-specific EngineName the column resolves to.
// Takes goName (string) which is the Go type name (for example int64 or uint32).
//
// Returns querier_dto.TypeMapping which is the integer mapping entry.
func integerMapping(sqlName string, goName string) querier_dto.TypeMapping {
	return querier_dto.TypeMapping{
		SQLCategory: querier_dto.TypeCategoryInteger,
		SQLName:     sqlName,
		NotNull:     querier_dto.GoType{Name: goName},
		Nullable:    querier_dto.GoType{Name: "*" + goName},
	}
}

// bigIntMapping returns an integer mapping to Piko's maths.BigInt, used for engine
// integer types wider than 64 bits that have no native fixed-width Go type. maths.BigInt
// is used in preference to stdlib math/big.Int for consistency with the decimal mapping
// (maths.Decimal) and because it implements driver.Valuer/sql.Scanner, where stdlib
// big.Int does not.
//
// Takes sqlName (string) which is the engine-specific EngineName the column resolves to.
//
// Returns querier_dto.TypeMapping which is the big-integer mapping entry.
func bigIntMapping(sqlName string) querier_dto.TypeMapping {
	return querier_dto.TypeMapping{
		SQLCategory: querier_dto.TypeCategoryInteger,
		SQLName:     sqlName,
		NotNull:     querier_dto.GoType{Package: goPackageMaths, Name: "BigInt"},
		Nullable:    querier_dto.GoType{Package: goPackageMaths, Name: "*BigInt"},
	}
}

// clickhouseIntegerMappings returns the width- and signedness-correct Go mappings for the
// ClickHouse fixed-width integer types. ClickHouse spells them in mixed case (Int64),
// which the case-sensitive name match in findTypeMappingCandidates keeps distinct from
// the lower-case Postgres canonical names (int2/int4/int8) even though they share a
// category.
//
// The unsigned types map to Go unsigned types so a UInt64 value above math.MaxInt64 does
// not overflow. The 128- and 256-bit types have no native fixed-width Go type, so they
// map to maths.BigInt, whose Scan accepts the *big.Int the clickhouse-go driver yields
// for them.
//
// Returns []querier_dto.TypeMapping which holds the ClickHouse integer mapping entries.
func clickhouseIntegerMappings() []querier_dto.TypeMapping {
	return []querier_dto.TypeMapping{
		integerMapping("Int8", "int8"),
		integerMapping("Int16", "int16"),
		integerMapping("Int32", goTypeInt32),
		integerMapping("Int64", "int64"),
		bigIntMapping("Int128"),
		bigIntMapping("Int256"),
		integerMapping("UInt8", "uint8"),
		integerMapping("UInt16", "uint16"),
		integerMapping("UInt32", goTypeUint32),
		integerMapping("UInt64", "uint64"),
		bigIntMapping("UInt128"),
		bigIntMapping("UInt256"),
	}
}

// mysqlIntegerMappings returns the MySQL family integer mappings.
//
// It covers MySQL and MariaDB integer types. The engine spells these with its own names
// (bigint, tinyint, "int unsigned", ...) rather than the postgres int2/int4/int8 canon.
// Without these every MySQL integer would fall to the int32 category default, silently
// truncating a bigint and dropping the unsigned contract. mediumint is 3 bytes and fits
// int32 / uint32. The BIT type is intentionally omitted: the driver yields it as raw
// bytes, not a fixed-width integer.
//
// Returns []querier_dto.TypeMapping which holds the MySQL family integer mapping entries.
func mysqlIntegerMappings() []querier_dto.TypeMapping {
	return []querier_dto.TypeMapping{
		integerMapping("tinyint", "int8"),
		integerMapping("smallint", "int16"),
		integerMapping("mediumint", goTypeInt32),
		integerMapping("int", goTypeInt32),
		integerMapping("bigint", "int64"),
		integerMapping("tinyint unsigned", "uint8"),
		integerMapping("smallint unsigned", "uint16"),
		integerMapping("mediumint unsigned", goTypeUint32),
		integerMapping("int unsigned", goTypeUint32),
		integerMapping("bigint unsigned", "uint64"),
	}
}

// duckdbIntegerMappings returns the DuckDB integer mappings.
//
// It covers types whose EngineName is not already in the postgres int2/int4/int8 canon
// (which DuckDB reuses for smallint/integer/bigint). int1 is its 1-byte signed type; the
// unsigned types map to Go unsigned types; the 128-bit hugeint and uhugeint have no
// native fixed-width Go type and map to maths.BigInt (whose Scan accepts the *big.Int
// duckdb yields).
//
// Returns []querier_dto.TypeMapping which holds the DuckDB integer mapping entries.
func duckdbIntegerMappings() []querier_dto.TypeMapping {
	return []querier_dto.TypeMapping{
		integerMapping("int1", "int8"),
		bigIntMapping("hugeint"),
		integerMapping("utinyint", "uint8"),
		integerMapping("usmallint", "uint16"),
		integerMapping("uinteger", goTypeUint32),
		integerMapping("ubigint", "uint64"),
		bigIntMapping("uhugeint"),
	}
}

// numericMappings returns the default mappings for integer, float, and decimal SQL types.
//
// Returns []querier_dto.TypeMapping which holds the numeric type mapping entries.
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
			NotNull:     querier_dto.GoType{Package: goPackageMaths, Name: "Decimal"},
			Nullable:    querier_dto.GoType{Package: goPackageMaths, Name: "*Decimal"},
		},
	}
}

// scalarMappings returns the default mappings for boolean, text, and bytea SQL types.
//
// Returns []querier_dto.TypeMapping which holds the scalar type mapping entries.
func scalarMappings() []querier_dto.TypeMapping {
	return []querier_dto.TypeMapping{
		{SQLCategory: querier_dto.TypeCategoryBoolean, NotNull: querier_dto.GoType{Name: "bool"}, Nullable: querier_dto.GoType{Name: "*bool"}},
		{SQLCategory: querier_dto.TypeCategoryText, NotNull: querier_dto.GoType{Name: "string"}, Nullable: querier_dto.GoType{Name: "*string"}},
		{SQLCategory: querier_dto.TypeCategoryBytea, NotNull: querier_dto.GoType{Name: goTypeByteSlice}, Nullable: querier_dto.GoType{Name: goTypeByteSlice}},
	}
}

// temporalMappings returns the default mappings for timestamp, date, time, and interval
// SQL types.
//
// Returns []querier_dto.TypeMapping which holds the temporal type mapping entries.
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

// complexMappings returns the default mappings for JSON, UUID, enum, array, struct, map,
// union, and aggregate-state SQL types.
//
// Union types map to encoding/json.RawMessage so callers can decode the tagged variant
// payload at runtime. Aggregate-state types map to []byte because the stored value is a
// driver-opaque binary blob produced by AggregateFunction and SimpleAggregateFunction
// columns.
//
// Returns []querier_dto.TypeMapping which holds the complex type mapping entries.
func complexMappings() []querier_dto.TypeMapping {
	goTypeAny := querier_dto.GoType{Name: "any"}
	goTypeAnySlice := querier_dto.GoType{Name: "[]any"}

	jsonType := querier_dto.GoType{Package: goPackageDBJSON, Name: goTypeDBJSON}
	byteSlice := querier_dto.GoType{Name: goTypeByteSlice}

	return []querier_dto.TypeMapping{
		{
			SQLCategory: querier_dto.TypeCategoryJSON,
			NotNull:     jsonType,
			Nullable:    jsonType,
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
		{
			SQLCategory: querier_dto.TypeCategoryUnion,
			NotNull:     jsonType,
			Nullable:    jsonType,
		},
		{
			SQLCategory: querier_dto.TypeCategoryAggregateState,
			NotNull:     byteSlice,
			Nullable:    byteSlice,
		},
	}
}
