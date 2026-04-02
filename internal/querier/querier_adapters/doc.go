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

// Package querier_adapters provides infrastructure implementations for the
// querier domain's port interfaces.
//
// Inbound engine adapters parse dialect-specific SQL into the domain's neutral
// IR. Each adapter wraps an engine-native parser and converts the
// engine-specific AST into the querier's structured types. Engine adapters are
// available in sub-packages, each implementing [querier_domain.EnginePort] and
// handling all dialect-specific concerns: parsing, DDL catalogue mutations,
// built-in type and function catalogues, and parameter placeholder styles.
//
// Outbound code emitter adapters generate Go source code from analysed
// queries, producing typed structs, query methods, enum definitions, and
// transaction helpers.
package querier_adapters
