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

package emitter_go

import (
	"piko.sh/piko/internal/querier/querier_adapters/emitter_go_sql"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// GoEmitter implements CodeEmitterPort by delegating to the database/sql emitter. Exists
// for backwards compatibility; new code should use emitter_go_sql.NewSQLEmitter()
// directly.
type GoEmitter struct {
	// sql holds the underlying database/sql emitter that performs the actual code
	// generation.
	sql *emitter_go_sql.SQLEmitter
}

var (
	_ querier_domain.CodeEmitterPort = (*GoEmitter)(nil)
)

// NewGoEmitter creates a new Go code emitter that delegates to the database/sql emitter.
//
// Returns *GoEmitter which is ready to emit Go source code.
func NewGoEmitter() *GoEmitter {
	return &GoEmitter{
		sql: emitter_go_sql.NewSQLEmitter(),
	}
}

// NewGoEmitterForMySQL creates a Go code emitter wired for engines whose driver accepts
// only anonymous `?` placeholders such as MySQL and MariaDB.
//
// It is used by the MariaDB and MySQL test harnesses and production bootstraps so the
// slice-expansion helper emits valid SQL.
//
// Returns *GoEmitter which is ready to emit Go source code for anonymous-marker engines.
func NewGoEmitterForMySQL() *GoEmitter {
	return &GoEmitter{
		sql: emitter_go_sql.NewSQLEmitterForMySQL(),
	}
}

// NewGoEmitterForClickHouse creates a Go code emitter that wraps each parameter access in
// `clickhouse.Named("name", value)` for the ClickHouse driver.
//
// The wrapping binds each parameter to the matching `{name:Type}` placeholder, and the
// generated packages import github.com/ClickHouse/clickhouse-go/v2.
//
// Returns *GoEmitter which is ready to emit Go source code for ClickHouse.
func NewGoEmitterForClickHouse() *GoEmitter {
	return &GoEmitter{
		sql: emitter_go_sql.NewSQLEmitterForClickHouse(),
	}
}

// NewGoEmitterForPostgres creates a Go code emitter wired for the postgres family
// (postgres, cockroachdb, timescaledb), whose drivers bind `$N` placeholders
// positionally.
//
// The slice-expansion helper scans and emits `$N` so an expanded IN list is valid
// postgres SQL.
//
// Returns *GoEmitter which is ready to emit Go source code for the postgres family.
func NewGoEmitterForPostgres() *GoEmitter {
	return &GoEmitter{
		sql: emitter_go_sql.NewSQLEmitterForPostgres(),
	}
}

// NewGoEmitterForDialect creates a Go code emitter configured for the named engine
// dialect so generated placeholders match the engine's driver.
//
// Takes dialect (string) which is the engine's Dialect() name.
//
// Returns *GoEmitter configured for the dialect.
func NewGoEmitterForDialect(dialect string) *GoEmitter {
	return &GoEmitter{
		sql: emitter_go_sql.NewSQLEmitterForDialect(dialect),
	}
}

// EmitModels generates Go struct types for each table in the catalogue.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes catalogue (*querier_dto.Catalogue) which holds the schema state.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type mappings.
//
// Returns []querier_dto.GeneratedFile which contains the model source files.
// Returns error when code emission fails.
func (emitter *GoEmitter) EmitModels(
	packageName string,
	catalogue *querier_dto.Catalogue,
	mappings *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	return emitter.sql.EmitModels(packageName, catalogue, mappings)
}

// EmitQueries generates Go source code for query methods.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the type-checked queries.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type mappings.
//
// Returns []querier_dto.GeneratedFile which contains the query source files.
// Returns error when code emission fails.
func (emitter *GoEmitter) EmitQueries(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	return emitter.sql.EmitQueries(packageName, queries, mappings)
}

// EmitQuerier generates the top-level querier scaffold. The second parameter is ignored;
// use emitter_go_pgx for pgx-native code.
//
// Takes packageName (string) which is the Go package name.
// Takes _ (querier_dto.QueryCapabilities) which is ignored by this wrapper.
//
// Returns querier_dto.GeneratedFile which contains the querier source file.
// Returns error when code emission fails.
func (emitter *GoEmitter) EmitQuerier(
	packageName string,
	_ querier_dto.QueryCapabilities,
) (querier_dto.GeneratedFile, error) {
	return emitter.sql.EmitQuerier(packageName, 0)
}

// EmitPrepared generates the PreparedDBTX wrapper.
//
// Takes packageName (string) which is the Go package name.
// Takes queries ([]*querier_dto.AnalysedQuery) which provide the SQL constants to eagerly
// prepare.
//
// Returns querier_dto.GeneratedFile which contains the prepared.go source.
// Returns error when code emission fails.
func (emitter *GoEmitter) EmitPrepared(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
) (querier_dto.GeneratedFile, error) {
	return emitter.sql.EmitPrepared(packageName, queries)
}

// EmitOTel generates the otel.go file containing the QueryNameResolver.
//
// Takes packageName (string) which is the Go package name.
// Takes queries ([]*querier_dto.AnalysedQuery) which provide query names.
//
// Returns querier_dto.GeneratedFile which contains the otel.go source.
// Returns error when code emission fails.
func (emitter *GoEmitter) EmitOTel(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
) (querier_dto.GeneratedFile, error) {
	return emitter.sql.EmitOTel(packageName, queries)
}
