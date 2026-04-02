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

// Package templater_adapters implements the templater port interfaces for
// manifest loading, template execution, and rendering.
//
// It implements ManifestRunnerPort for three execution modes: compiled
// (production), interpreted (development with JIT compilation), and
// caching (decorator for performance). It also includes renderer
// adapters, a manifest store view for development mode, and a virtual
// filesystem for the Go interpreter.
//
// # Execution modes
//
// The package supports three execution modes through different runner
// implementations:
//
// Compiled Mode (Production):
//
//	store, _ := NewManifestStore(provider)
//	runner := NewCompiledManifestRunner(store, i18nService, "en")
//
// Interpreted Mode (Development):
//
//	runner := NewInterpretedManifestRunner(i18nService, cache, orchestrator, "en")
//
// Cached Mode (Decorator):
//
//	cachedRunner := NewCachingManifestRunner(runner, cacheService)
//
// # Thread safety
//
// All manifest runner implementations are safe for concurrent use.
// InterpretedManifestRunner uses internal locking for cache access.
// ManifestStore is read-only after construction and safe for concurrent
// reads. RegistryVFSAdapter uses a read-write mutex to guard its path
// map and fresh artefacts cache.
package templater_adapters
