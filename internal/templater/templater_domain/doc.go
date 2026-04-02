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

// Package templater_domain provides domain types and services for the
// template compilation and rendering pipeline.
//
// Register AST and cache policy functions during initialisation:
//
//	func init() {
//	    templater_domain.RegisterASTFunc("myapp/pages/home", BuildAST)
//	    templater_domain.RegisterCachePolicyFunc("myapp/pages/home", CachePolicy)
//	}
//
// For testing, use [NewIsolatedRegistry] to create independent
// registries that enable parallel test execution.
//
// # Thread safety
//
// The [FunctionRegistry] and its package-level accessor functions are
// safe for concurrent use. The [InspectionCache] is also thread-safe.
// Service implementations are thread-safe when their injected
// dependencies are thread-safe.
package templater_domain
