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

// Package db_catalogue_sqlite provides a PRAGMA-based catalogue provider for
// SQLite databases. It implements CatalogueProviderPort by introspecting a
// live database via PRAGMA commands rather than replaying migration files.
//
// This is useful for users with existing SQLite databases who want to generate
// type-safe query code without maintaining migration files.
package db_catalogue_sqlite
