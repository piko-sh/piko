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

package db_engine_timescaledb

import (
	"maps"

	"piko.sh/piko/internal/querier/querier_dto"
)

// timescaleDBTypes returns the opaque aggregate-state and hyperfunction types TimescaleDB
// exposes.
//
// Each is registered as TypeCategoryUnknown with a stable EngineName so codegen does not
// error when a query returns one of these types. Users can supply piko.column(go_type:
// ...) overrides for concrete Go destinations in projects that materialise the aggregate
// state in application code.
//
// The map is assembled in three groups so the body stays under the per-function size
// guideline: core aggregate-state entries, toolkit state entries, and system aliases.
// Splitting on the conceptual boundary keeps each helper scannable when auditing the
// registered type surface.
//
// Returns map[string]querier_dto.SQLType which holds every TimescaleDB type keyed by its
// lowercased engine name.
func timescaleDBTypes() map[string]querier_dto.SQLType {
	types := timescaleDBAggregateStateTypes()
	maps.Copy(types, timescaleDBToolkitStateTypes())
	maps.Copy(types, timescaleDBSystemAliasTypes())
	return types
}

// timescaleDBAggregateStateTypes returns the core aggregate-state and hyperfunction
// types: stats summaries, time-weighted summary, candlestick, state, compact_state,
// hyperloglog, tdigest, uddsketch, and heartbeat.
//
// Each is registered as TypeCategoryUnknown because their concrete Go representation is
// opaque to codegen.
//
// Returns map[string]querier_dto.SQLType which holds the core aggregate-state types keyed
// by lowercased engine name.
func timescaleDBAggregateStateTypes() map[string]querier_dto.SQLType {
	return map[string]querier_dto.SQLType{
		"statssummary1d":  {Category: querier_dto.TypeCategoryUnknown, EngineName: "statssummary1d"},
		"statssummary2d":  {Category: querier_dto.TypeCategoryUnknown, EngineName: "statssummary2d"},
		"counter_summary": {Category: querier_dto.TypeCategoryUnknown, EngineName: "counter_summary"},
		"gauge_summary":   {Category: querier_dto.TypeCategoryUnknown, EngineName: "gauge_summary"},

		"time_weight_summary": {Category: querier_dto.TypeCategoryUnknown, EngineName: "time_weight_summary"},

		"candlestick":   {Category: querier_dto.TypeCategoryUnknown, EngineName: "candlestick"},
		"state_summary": {Category: querier_dto.TypeCategoryUnknown, EngineName: "state_summary"},

		"compact_state_agg": {Category: querier_dto.TypeCategoryUnknown, EngineName: "compact_state_agg"},

		"hyperloglog": {Category: querier_dto.TypeCategoryUnknown, EngineName: "hyperloglog"},
		"tdigest":     {Category: querier_dto.TypeCategoryUnknown, EngineName: "tdigest"},
		"uddsketch":   {Category: querier_dto.TypeCategoryUnknown, EngineName: "uddsketch"},
		"heartbeat":   {Category: querier_dto.TypeCategoryUnknown, EngineName: "heartbeat"},
	}
}

// timescaleDBToolkitStateTypes returns the long-tail toolkit aggregate state types.
//
// These cover frequency-analysis space-saving states, min_n and max_n states, the
// timevector summary, count_min_sketch, and the composite records returned by hypertable
// management functions.
//
// Returns map[string]querier_dto.SQLType which holds the toolkit aggregate-state types
// keyed by lowercased engine name.
func timescaleDBToolkitStateTypes() map[string]querier_dto.SQLType {
	return map[string]querier_dto.SQLType{
		"create_hypertable_record": {Category: querier_dto.TypeCategoryUnknown, EngineName: "create_hypertable_record"},
		"add_dimension_record":     {Category: querier_dto.TypeCategoryUnknown, EngineName: "add_dimension_record"},

		"space_saving_aggregate":        {Category: querier_dto.TypeCategoryUnknown, EngineName: "space_saving_aggregate"},
		"space_saving_bigint_aggregate": {Category: querier_dto.TypeCategoryUnknown, EngineName: "space_saving_bigint_aggregate"},
		"space_saving_text_aggregate":   {Category: querier_dto.TypeCategoryUnknown, EngineName: "space_saving_text_aggregate"},

		"min_n_state":    {Category: querier_dto.TypeCategoryUnknown, EngineName: "min_n_state"},
		"min_n_by_state": {Category: querier_dto.TypeCategoryUnknown, EngineName: "min_n_by_state"},

		"timevector_tstz_f64": {Category: querier_dto.TypeCategoryUnknown, EngineName: "timevector_tstz_f64"},

		"count_min_sketch": {Category: querier_dto.TypeCategoryUnknown, EngineName: "count_min_sketch"},
	}
}

// timescaleDBSystemAliasTypes returns system-type aliases that TimescaleDB exposes
// through its catalogue but that the host postgres engine does not bootstrap on its own.
//
// Each entry mirrors a known builtin in db_engine_postgres so the analyser treats the
// alias as a recognised type rather than reporting an unknown identifier.
//
// Returns map[string]querier_dto.SQLType which holds the system-type aliases keyed by
// lowercased engine name.
func timescaleDBSystemAliasTypes() map[string]querier_dto.SQLType {
	return map[string]querier_dto.SQLType{
		"regproc": {Category: querier_dto.TypeCategoryInteger, EngineName: "regproc"},
	}
}
