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

// Package goastutil provides Go AST construction, type-string
// parsing, and type classification utilities.
//
// It offers AST node builders for programmatic code generation, type
// string parsing and formatting with package qualification support,
// and helpers for determining whether types are primitives, built-ins,
// or standard library types. It also checks Go internal package
// accessibility.
//
// # Usage
//
// Parse a type string and convert it back:
//
//	expr := goastutil.TypeStringToAST("map[string]*User")
//	str := goastutil.ASTToTypeString(expr, "models")
//
// Build AST nodes programmatically:
//
//	call := goastutil.CallExpr(
//	    goastutil.SelectorExpr("fmt", "Sprintf"),
//	    goastutil.StrLit("hello %s"),
//	    goastutil.CachedIdent("name"),
//	)
//
// # Thread safety
//
// The identifier and literal caches (CachedIdent, StrLit) use sync.Map
// and are safe for concurrent use. The static identifier cache is
// populated at init time and is read-only thereafter. RegisterIdent
// must only be called during package initialisation.
//
// # Performance
//
// The package caches pre-parsed AST expressions for primitive types
// and reuses a shared token.FileSet to minimise allocations in hot
// paths. The two-tier identifier cache (static map + sync.Map) avoids
// repeated heap allocations for commonly occurring identifiers.
package goastutil
