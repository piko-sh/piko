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

// GoEmitter implements CodeEmitterPort by delegating to the database/sql
// emitter. This type exists for backwards compatibility; new code should use
// emitter_go_sql.NewSQLEmitter() directly.
type GoEmitter struct {
	// sql holds the underlying database/sql emitter that performs the actual
	// code generation.
	sql *emitter_go_sql.SQLEmitter
}

var _ querier_domain.CodeEmitterPort = (*GoEmitter)(nil)

// NewGoEmitter creates a new Go code emitter that delegates to the database/sql
// emitter.
//
// Returns *GoEmitter which is ready to emit Go source code.
func NewGoEmitter() *GoEmitter {
	return &GoEmitter{
		sql: emitter_go_sql.NewSQLEmitter(),
	}
}

// EmitModels generates Go struct types for each table in the catalogue.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes catalogue (*querier_dto.Catalogue) which holds the schema state.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type
// mappings.
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
// Takes queries ([]*querier_dto.AnalysedQuery) which are the type-checked
// queries.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type
// mappings.
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

// EmitQuerier generates the top-level querier scaffold. The second
// parameter is ignored; use emitter_go_pgx for pgx-native code.
//
// Takes packageName (string) which is the Go package name.
// Takes _ (querier_dto.QueryCapabilities) which is ignored by
// this wrapper.
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
// Takes queries ([]*querier_dto.AnalysedQuery) which provide the SQL constants
// to eagerly prepare.
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
