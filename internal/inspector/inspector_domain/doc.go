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

// Package inspector_domain provides the core business logic for analysing
// Go source code and extracting type information.
//
// It contains the type builder for parsing and caching Go source files,
// and the type querier for resolving types, fields, and methods from AST
// expressions. Two build modes are supported: a full mode that uses
// go/packages for complete type-checking, and a lightweight AST-only
// mode for REPL/WASM environments where go/packages is unavailable.
//
// The type builder is not safe for concurrent use. The type querier is
// safe for concurrent use after creation, with memoisation via sync.Map.
package inspector_domain
