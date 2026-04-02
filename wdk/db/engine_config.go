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

package db

import "piko.sh/piko/internal/bootstrap"

// EngineConfig bundles the components needed for a database engine: the
// codegen parser (used at build time), the migration dialect (used at
// runtime for migration execution), and an optional factory for creating
// live-database catalogue providers.
//
// Engine sub-packages (db_engine_postgres, db_engine_mysql, etc.) provide
// pre-built EngineConfig values via constructor functions like Postgres(),
// MySQL(), and SQLite().
type EngineConfig = bootstrap.EngineConfig

// DatabaseHealthDiagnostic is a single diagnostic measurement from an
// engine-specific health checker.
type DatabaseHealthDiagnostic = bootstrap.DatabaseHealthDiagnostic

// DatabaseHealthChecker is an optional interface that engine implementations
// can satisfy to provide engine-specific health diagnostics beyond a basic
// ping.
type DatabaseHealthChecker = bootstrap.DatabaseHealthChecker
