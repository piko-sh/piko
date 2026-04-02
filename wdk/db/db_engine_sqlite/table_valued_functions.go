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

package db_engine_sqlite

import "piko.sh/piko/internal/querier/querier_dto"

var jsonTableColumns = []querier_dto.ScopedColumn{
	{Name: "key", SQLType: tvfSQLType("text", querier_dto.TypeCategoryText), Nullable: true},
	{Name: "value", SQLType: tvfSQLType("text", querier_dto.TypeCategoryText), Nullable: true},
	{Name: "type", SQLType: tvfSQLType("text", querier_dto.TypeCategoryText), Nullable: true},
	{Name: "atom", SQLType: tvfSQLType("text", querier_dto.TypeCategoryText), Nullable: true},
	{Name: "id", SQLType: tvfSQLType("integer", querier_dto.TypeCategoryInteger), Nullable: true},
	{Name: "parent", SQLType: tvfSQLType("integer", querier_dto.TypeCategoryInteger), Nullable: true},
	{Name: "fullkey", SQLType: tvfSQLType("text", querier_dto.TypeCategoryText), Nullable: true},
	{Name: "path", SQLType: tvfSQLType("text", querier_dto.TypeCategoryText), Nullable: true},
}

var tableValuedFunctionColumns = map[string][]querier_dto.ScopedColumn{
	"json_each":       jsonTableColumns,
	"json_tree":       jsonTableColumns,
	"generate_series": {{Name: "value", SQLType: tvfSQLType("integer", querier_dto.TypeCategoryInteger), Nullable: false}},
}

func tvfSQLType(engineName string, category querier_dto.SQLTypeCategory) querier_dto.SQLType {
	return querier_dto.SQLType{EngineName: engineName, Category: category}
}
