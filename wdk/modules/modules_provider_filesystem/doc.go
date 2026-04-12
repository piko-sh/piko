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

// Package modules_provider_filesystem implements [modules.ModuleProvider] backed by a
// directory of bundle files on local disk. Intended for piko CLI use ("piko run
// --module-dir ./vendor"), test fixtures, and air-gapped deployments that ship a
// pre-populated bundle tree.
//
// # Directory layout
//
// The provider expects a fixed layout under the root:
//
//	<root>/<encoded-path>/<version>.pkbundle
//
// where <encoded-path> is the module path with '/' replaced by '__' so the filesystem
// treats it as a single directory name, and .pkbundle is the deterministic envelope
// produced by piko's bundler. Each .pkbundle file contains the descriptor JSON followed
// by the bytecode payload, separated by a single null byte.
//
// # Concurrency
//
// Resolve is safe for concurrent use. The filesystem itself is the shared mutable state;
// writes performed concurrently with reads of the same path produce undefined results,
// matching standard POSIX semantics. Hosts that mutate the directory tree should quiesce
// readers or write atomically (temp + rename).
package modules_provider_filesystem
