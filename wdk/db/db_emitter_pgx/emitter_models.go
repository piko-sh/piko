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

package db_emitter_pgx

import (
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// EmitModels generates Go struct types for each table in the catalogue.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes catalogue (*querier_dto.Catalogue) which is the schema state.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type
// mappings.
//
// Returns []querier_dto.GeneratedFile which contains the models.go file, or an
// empty slice if the catalogue has no tables.
// Returns error when code generation fails.
func (*PgxEmitter) EmitModels(
	packageName string,
	catalogue *querier_dto.Catalogue,
	mappings *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	return emitter_shared.EmitModels(packageName, catalogue, mappings)
}
