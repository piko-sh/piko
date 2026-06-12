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

// Package modules is the public surface for piko's module-loading contract.
//
// The contract is intentionally narrow:
//
//   - [ModuleRef] identifies a module dependency.
//   - [ModuleDescriptor] declares what it is.
//   - [ModuleBundle] pairs descriptor with compiled bytecode.
//   - [Capability] is one capability claim; [CapabilitySet] is a collection.
//   - [ModuleProvider] is the host's resolution port.
//   - [LoadedModule] is the handle piko returns from a successful load.
//
// Companion provider implementations live in sibling packages:
//
//   - wdk/modules/modules_provider_filesystem - reads bundles from a directory, intended
//     for piko-CLI use ("piko run --module-dir ./vendor").
//   - wdk/modules/modules_provider_inmemory - in-memory map for tests and the REPL.
//   - wdk/modules/modules_provider_goproxy - GOPROXY-backed fetch for pipit-style
//     scripting.
//   - wdk/modules/modules_provider_chain - composes other providers in priority order.
package modules
