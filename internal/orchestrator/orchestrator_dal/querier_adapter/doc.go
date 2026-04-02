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

// Package querier_adapter wraps the code-generated SQLite
// queries to satisfy the orchestrator DAL and task store
// interfaces.
//
// The [Adapter] type handles JSON serialisation of payload,
// config, and result fields, Unix-second timestamp conversion,
// dynamic SQL expansion for IN-clause queries, and transaction
// lifecycle management. It bridges the gap between the domain
// layer's typed models and the flat row types produced by the
// code generator.
package querier_adapter
