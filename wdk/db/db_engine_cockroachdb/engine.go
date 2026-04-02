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

// NewCockroachDBEngine creates a CockroachDB engine adapter by configuring the
// PostgreSQL engine with CockroachDB-specific dialect options.
func NewCockroachDBEngine() *db_engine_postgres.PostgresEngine {
	return db_engine_postgres.NewPostgresEngine(
		db_engine_postgres.WithDialectName("cockroachdb"),
		db_engine_postgres.WithExtraTypes(cockroachDBTypes()),
		db_engine_postgres.WithExtraFunctions(registerCockroachDBFunctions),
	)
}

func cockroachDBTypes() map[string]querier_dto.SQLType {
	return map[string]querier_dto.SQLType{
		"string": {Category: querier_dto.TypeCategoryText, EngineName: "text"},
		"bytes":  {Category: querier_dto.TypeCategoryBytea, EngineName: "bytea"},
	}
}

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
	builder.NullOnNull("crdb_internal.cluster_id", nil, uuidType)
	builder.NeverNull("gateway_region", nil, textType)
	builder.NeverNull("rehome_row", nil, textType)
	builder.NeverNull("crdb_internal.node_id", nil, integerType)
	builder.NeverNull("crdb_internal.is_admin", nil, booleanType)
	builder.NullOnNull("crdb_internal.locality_value", builder.Args("key", textType), textType)
	builder.NullOnNull("from_ip", builder.Args("value", byteaType), textType)
	builder.NullOnNull("to_ip", builder.Args("address", textType), byteaType)
	builder.NullOnNull("experimental_strftime", builder.Args("input", timestamptzType, "format", textType), textType)
	builder.NullOnNull("experimental_strptime", builder.Args("input", textType, "format", textType), timestamptzType)
}
