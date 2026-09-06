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

const (
	// mariaDBDefaultOptionCount is the number of options NewMariaDBEngine applies before the
	// caller's own.
	mariaDBDefaultOptionCount = 3
)

// NewMariaDBEngine creates a MariaDB engine adapter.
//
// Configures the MySQL engine with MariaDB-specific dialect options, then applies the
// caller's options so parser limits such as db_engine_mysql.WithMaxParseDepth and
// db_engine_mysql.WithMaxTokensPerStatement can be tuned for MariaDB too.
//
// Takes options (...db_engine_mysql.Option) which further configure the dialect after the
// MariaDB defaults.
//
// Returns *db_engine_mysql.MySQLEngine which is ready for catalogue introspection and
// code generation against MariaDB.
func NewMariaDBEngine(options ...db_engine_mysql.Option) *db_engine_mysql.MySQLEngine {
	mariaDBOptions := make([]db_engine_mysql.Option, 0, mariaDBDefaultOptionCount+len(options))
	mariaDBOptions = append(mariaDBOptions,
		db_engine_mysql.WithDialectName("mariadb"),
		db_engine_mysql.WithReturningSupport(true),
		db_engine_mysql.WithExtraFunctions(registerMariaDBFunctions),
	)
	return db_engine_mysql.NewMySQLEngine(append(mariaDBOptions, options...)...)
}

// registerMariaDBFunctions registers MariaDB-specific built-in functions onto the shared
// MySQL function catalogue.
//
// INET6_ATON / INET6_NTOA are standard MySQL functions and now live in the base
// catalogue, so only genuinely MariaDB-exclusive functions (SYS_GUID) are registered
// here.
//
// Takes builder (*db_engine_mysql.FunctionCatalogueBuilder) which receives the extra
// function signatures.
func registerMariaDBFunctions(builder *db_engine_mysql.FunctionCatalogueBuilder) {
	guidType := querier_dto.NewSQLType(querier_dto.TypeCategoryText, "varchar")
	builder.NeverNull("sys_guid", nil, guidType)
}
