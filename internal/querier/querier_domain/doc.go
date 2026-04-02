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

// Package querier_domain provides the core SQL analysis and code
// generation pipeline for Piko's database integration. It transforms
// SQL migration and query files into fully typed Go code with
// end-to-end type safety between database schemas and Go code.
//
// The pipeline runs as Phase 1 of the build, before the inspector.
// Migration replay builds a versioned schema catalogue. Query files
// are parsed via the engine adapter, analysed with nested scope chains
// (CTEs, subqueries, LATERAL), and emitted as Go structs, query
// methods, enum types, and transaction helpers.
//
// Generated packages import only the standard library and Piko
// framework types. They never import user code, so broken user code
// cannot prevent regeneration. The inspector (Phase 2) then validates
// that user code correctly uses the generated types.
//
// Users write real, valid SQL. All metadata lives in SQL comments
// using piko-prefixed directives, so SQL files are copy-pasteable
// into any database client. The domain works with a dialect-neutral
// intermediate representation; engine-specific parsing stays in
// inbound adapters. Every expression node carries its resolved type
// and nullability, propagated through JOINs, functions, aggregates,
// COALESCE, CASE, and subqueries.
package querier_domain
