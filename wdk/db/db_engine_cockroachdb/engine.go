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

package db_engine_cockroachdb

import (
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

const (
	// crdbInternalSchema is the schema under which CockroachDB's crdb_internal.* builtins
	// are registered so a schema-qualified call resolves to them.
	crdbInternalSchema = "crdb_internal"

	// engineNameInt8 is the 64-bit integer engine type that CockroachDB's INT / INTEGER /
	// SERIAL family maps to, unlike PostgreSQL's 32-bit int4.
	engineNameInt8 = "int8"
)

// NewCockroachDBEngine creates a CockroachDB engine adapter.
//
// Configures the PostgreSQL engine with CockroachDB-specific dialect options.
//
// Returns *db_engine_postgres.PostgresEngine which is ready for catalogue introspection
// and code generation against CockroachDB.
func NewCockroachDBEngine() *db_engine_postgres.PostgresEngine {
	return db_engine_postgres.NewPostgresEngine(
		db_engine_postgres.WithDialectName("cockroachdb"),
		db_engine_postgres.WithExtraTypes(cockroachDBTypes()),
		db_engine_postgres.WithTypeNormaliserHook(normaliseCockroachDBType),
		db_engine_postgres.WithExtraFunctions(registerCockroachDBFunctions),
	)
}

// cockroachDBTypes returns the extra CockroachDB type aliases that map onto the
// PostgreSQL engine's structured SQLType values. These populate the type catalogue used
// by resolution lookups; the DDL column-type normaliser is handled separately by
// normaliseCockroachDBType.
//
// Returns map[string]querier_dto.SQLType keyed by raw CockroachDB type name.
func cockroachDBTypes() map[string]querier_dto.SQLType {
	return map[string]querier_dto.SQLType{
		"string": {Category: querier_dto.TypeCategoryText, EngineName: "text"},
		"bytes":  {Category: querier_dto.TypeCategoryBytea, EngineName: "bytea"},

		"int":     {Category: querier_dto.TypeCategoryInteger, EngineName: engineNameInt8},
		"integer": {Category: querier_dto.TypeCategoryInteger, EngineName: engineNameInt8},
		"int64":   {Category: querier_dto.TypeCategoryInteger, EngineName: engineNameInt8},
		"serial":  {Category: querier_dto.TypeCategoryInteger, EngineName: engineNameInt8},
	}
}

// normaliseCockroachDBType maps CockroachDB's idiomatic STRING / BYTES column types onto
// the PostgreSQL text / bytea categories.
//
// The DDL normaliser skips WithExtraTypes, so without this hook these column types would
// resolve to Unknown and emit `any` Go fields. Length modifiers (STRING(50)) do not
// affect the Go type and are ignored.
//
// Takes name (string) which is the lower-cased type name.
//
// Returns *querier_dto.SQLType for a recognised CockroachDB type, or nil to defer to the
// PostgreSQL normaliser.
func normaliseCockroachDBType(name string, _ []int) *querier_dto.SQLType {
	switch name {
	case "string":
		return new(querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"})
	case "bytes":
		return new(querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "bytea"})
	case "int", "integer", "int64", "serial":

		return new(querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: engineNameInt8})
	default:
		return nil
	}
}

// registerCockroachDBFunctions registers CockroachDB-specific built-in functions onto the
// shared PostgreSQL function catalogue.
//
// Takes builder (*db_engine_postgres.FunctionCatalogueBuilder) which receives the extra
// function signatures.
func registerCockroachDBFunctions(builder *db_engine_postgres.FunctionCatalogueBuilder) {
	integerType := builder.Bigint
	textType := builder.Text
	booleanType := builder.Boolean
	byteaType := builder.Bytea
	numericType := builder.Numeric
	uuidType := builder.UUID
	timestamptzType := builder.Timestamptz

	builder.NeverNull("unique_rowid", nil, integerType)
	builder.NeverNull("cluster_logical_timestamp", nil, numericType)
	builder.NeverNull("gateway_region", nil, textType)
	builder.NeverNull("rehome_row", nil, textType)

	builder.NullOnNull("cluster_id", nil, uuidType).Schema = crdbInternalSchema
	builder.NeverNull("node_id", nil, integerType).Schema = crdbInternalSchema
	builder.NeverNull("is_admin", nil, booleanType).Schema = crdbInternalSchema
	builder.NullOnNull("locality_value", builder.Args(db_engine_postgres.Arg{Name: "key", Type: textType}), textType).Schema = crdbInternalSchema
	builder.NullOnNull("from_ip", builder.Args(db_engine_postgres.Arg{Name: "value", Type: byteaType}), textType)
	builder.NullOnNull("to_ip", builder.Args(db_engine_postgres.Arg{Name: "address", Type: textType}), byteaType)
	builder.NullOnNull("experimental_strftime", builder.Args(db_engine_postgres.Arg{Name: "input", Type: timestamptzType}, db_engine_postgres.Arg{Name: "format", Type: textType}), textType)
	builder.NullOnNull("experimental_strptime", builder.Args(db_engine_postgres.Arg{Name: "input", Type: textType}, db_engine_postgres.Arg{Name: "format", Type: textType}), timestamptzType)
}
