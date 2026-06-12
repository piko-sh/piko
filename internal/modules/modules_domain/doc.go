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

// Package modules_domain defines the language-level contract for loading Go-as-bytecode
// modules into a piko interpreter at runtime.
//
// Owns the types that cross the boundary between piko (the interpreter) and its hosts:
//
//   - [ModuleRef]: how compiled code references a module dependency by path + version +
//     integrity pin.
//   - [ModuleDescriptor]: what a module IS - declared capabilities, pinned stdlib
//     version, exported symbol allowlist.
//   - [ModuleBundle]: the on-disk / on-wire artefact pairing a descriptor with its
//     compiled bytecode.
//   - [Capability]: a single capability claim (axis + scope) the host's CapabilityHook
//     interprets.
//   - [ModuleProvider]: the port a host implements to resolve a ModuleRef into a
//     ModuleBundle.
package modules_domain
