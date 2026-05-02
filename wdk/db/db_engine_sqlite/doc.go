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

// Package db_engine_sqlite implements the querier EnginePort for SQLite using a
// hand-written recursive-descent parser. It converts SQLite DDL into catalogue mutations
// and analyses DML queries to produce the raw query analysis that the domain layer
// resolves into fully typed results.
//
// # Integer widths
//
// SQLite stores every integer as a variable-length 1-8 byte value, so a declared width is
// a preference, not a storage limit. For cross-dialect parity with Postgres and MySQL the
// engine maps an ordinary table's INTEGER to a 32-bit Go int (int4); declare BIGINT for a
// 64-bit column. Two cases are always 64-bit: an INTEGER PRIMARY KEY (the rowid alias,
// which SQLite forbids spelling as BIGINT AUTOINCREMENT), and every integer column of a
// STRICT table.
//
// # STRICT tables
//
// A STRICT table (CREATE TABLE ...) STRICT) enforces its declared types and permits only
// INT, INTEGER, REAL, TEXT, BLOB, and ANY; the engine rejects any other spelling at
// generation time (validateStrictColumnTypes) instead of letting it fail at migration
// time. Because STRICT cannot express BIGINT, its INTEGER is genuinely signed 64-bit, so
// every integer column of a STRICT table resolves to a Go int64 (widenStrictIntegers).
// The same INTEGER spelling therefore yields int32 in an ordinary table but int64 inside
// a STRICT table. An ANY column resolves to Go any.
package db_engine_sqlite
