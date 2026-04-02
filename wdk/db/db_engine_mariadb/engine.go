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

package db_engine_mariadb

import (
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_mysql"
)

// NewMariaDBEngine creates a MariaDB engine adapter by configuring the MySQL
// engine with MariaDB-specific dialect options.
func NewMariaDBEngine() *db_engine_mysql.MySQLEngine {
	return db_engine_mysql.NewMySQLEngine(
		db_engine_mysql.WithDialectName("mariadb"),
		db_engine_mysql.WithReturningSupport(true),
		db_engine_mysql.WithSequenceSupport(true),
		db_engine_mysql.WithExtraFunctions(registerMariaDBFunctions),
	)
}

func registerMariaDBFunctions(builder *db_engine_mysql.FunctionCatalogueBuilder) {
	uuidType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
	builder.NeverNull("uuid", nil, uuidType)
	builder.NeverNull("sys_guid", nil, uuidType)

	integerType := querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"}
	textType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}
	binaryType := querier_dto.SQLType{Category: querier_dto.TypeCategoryBytea, EngineName: "varbinary"}

	builder.NullOnNull("inet_aton", builder.Args("address", textType), integerType)
	builder.NullOnNull("inet_ntoa", builder.Args("value", integerType), textType)
	builder.NullOnNull("inet6_aton", builder.Args("address", textType), binaryType)
	builder.NullOnNull("inet6_ntoa", builder.Args("value", binaryType), textType)
}
