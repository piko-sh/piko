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

// Package db_engine_clickhouse provides the Piko querier engine adapter for ClickHouse.
// It implements the querier_domain.EnginePort contract: SQL parsing, type system,
// directive prefix configuration, builtin function and type catalogues, and per-engine
// metadata.
//
// ClickHouse diverges from the SQL-standard family that postgres and mysql inhabit in
// three notable ways the adapter must handle.
//
// Parameter binding uses the brace form `{name:Type}` where the Go code generator must
// emit both the placeholder identifier and its ClickHouse-native type tag. The engine
// declares querier_dto.ParameterStyleClickHouseCurly and the emitter calls
// FormatParameter to obtain the literal SQL string for each parameter site.
//
// Nullability is encoded inside the SQL type name as `Nullable(T)` rather than tracked as
// a separate column flag. The engine strips the wrapper during NormaliseTypeName and sets
// the consumer's Nullable flag instead, so the rest of the pipeline sees the same shape
// it sees from other engines.
//
// `CREATE TABLE ... ENGINE = MergeTree() PARTITION BY ... ORDER BY ... SETTINGS ...`
// carries query-planning metadata that codegen does not interpret. The parser captures
// these clauses verbatim into CatalogueMutation.EngineSpecific so later passes such as
// PREWHERE eligibility and FINAL semantics can read them without round-trip loss.
//
// The adapter ships codegen and migration execution support. Migration execution uses a
// table-based version tracker because ClickHouse has no advisory locks; concurrent
// migrators rely on external coordination.
package db_engine_clickhouse
