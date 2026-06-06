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

package db_engine_clickhouse

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

var (
	// clickhouseTableValuedFunctionColumns maps known ClickHouse table-valued functions to
	// their fixed-shape output columns. Only the functions with deterministic column shape
	// appear here; dynamic functions like remote() and cluster() return the source table's
	// columns and resolve against the live catalogue.
	clickhouseTableValuedFunctionColumns = map[string][]querier_dto.ScopedColumn{
		"numbers": {
			{
				Name:    "number",
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
			},
		},
		"numbers_mt": {
			{
				Name:    "number",
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "UInt64"},
			},
		},
		"generateSeries": {
			{
				Name:    "value",
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"},
			},
		},
		"generate_series": {
			{
				Name:    "value",
				SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "Int64"},
			},
		},
		"input": nil,
	}
)
