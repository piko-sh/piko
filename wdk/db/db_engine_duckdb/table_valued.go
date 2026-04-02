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

package db_engine_duckdb

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

var tableValuedFunctionColumns = map[string][]querier_dto.ScopedColumn{
	"generate_series": {
		{Name: "generate_series", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"}, Nullable: false},
	},
	"range": {
		{Name: "range", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "int8"}, Nullable: false},
	},
	"unnest": {
		{Name: "unnest", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: ""}, Nullable: true},
	},
	"regexp_matches": {
		{
			Name: "regexp_matches",
			SQLType: querier_dto.SQLType{
				Category:    querier_dto.TypeCategoryArray,
				EngineName:  "varchar[]",
				ElementType: &querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
			},
			Nullable: true,
		},
	},
	"read_parquet":   nil,
	"read_csv":       nil,
	"read_csv_auto":  nil,
	"read_json":      nil,
	"read_json_auto": nil,
}
