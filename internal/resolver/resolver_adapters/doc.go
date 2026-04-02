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

// Package resolver_adapters implements the ResolverPort interface for resolving
// Piko component, CSS, and asset paths from different sources.
//
// It resolves import paths to absolute filesystem locations, supporting both
// local project files and external Go modules in $GOMODCACHE.
//
// # Module alias
//
// The package supports the @ alias for module-relative imports. When a file
// imports "@/partials/card.pk", the @ is expanded to the module name from
// the containing file's go.mod. This enables concise, portable imports:
//
//	import comp "@/partials/card.pk"
//	// Expands to: github.com/myorg/myproject/partials/card.pk
//
// # Resolution order
//
// When using ChainedResolver (the typical configuration), paths are resolved
// in priority order:
//
//  1. LocalModuleResolver - local project files take precedence
//  2. GoModuleCacheResolver - external modules as fallback
//
// This means local files always override external modules with the same
// path, preventing dependency hijacking.
//
// # Thread safety
//
// All resolver types are safe for concurrent use. GoModuleCacheResolver
// maintains an internal cache protected by a mutex.
package resolver_adapters
