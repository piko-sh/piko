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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package db_emitter_clickhouse implements CodeEmitterPort for the native
// clickhouse-go/v2 driver.
//
// Generates Go source that uses the ClickHouse native interface
// (github.com/ClickHouse/clickhouse-go/v2/lib/driver) rather than database/sql: reads
// bind positional `?` arguments with typed Go values, and bulk writes (:batch /
// :copyfrom) compile to the native columnar path conn.PrepareBatch -> batch.Append ->
// batch.Send.
//
// This is the high-throughput counterpart to the generic database/sql emitter, mirroring
// how db_emitter_pgx provides pgx CopyFrom for PostgreSQL. It is opt-in: an application
// selects it by passing NewClickHouseEmitter() to
// db.NewQuerierService(QuerierPorts{...}).
//
// # Dependencies
//
// The emitter itself does not import clickhouse-go at build time; it only produces source
// text that references the native driver types. The clickhouse-go/v2 dependency therefore
// lands only in the module that consumes the generated code, never in Piko core.
//
// # Constraints
//
// ClickHouse's native Exec returns only an error (no sql.Result), so :exec compiles to a
// single-value `return db.Exec(...)` while :execrows and :execresult are rejected at
// generation time. The native connection has no interactive transactions, so the
// generated querier omits WithTx/RunInTx.
//
// # Thread safety
//
// Emitter values carry no mutable state and are safe for concurrent use by multiple
// goroutines.
package db_emitter_clickhouse
