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
// directory of bundle files on local disk. Intended for vendored bundles, test fixtures,
// and air-gapped deployments that ship a pre-populated bundle tree.
//
// # Directory layout
//
// The provider expects a fixed layout under the root:
//
//	<root>/<encoded-path>/<version>.pkbundle
//
// where <encoded-path> is the module path with '/' replaced by '__' so the filesystem
// treats it as a single directory name, and .pkbundle is the deterministic envelope
// produced by piko's bundler.
//
// # Envelope format
//
// Every .pkbundle file starts with the five bytes "PKBND" and a version byte. Version 2,
// which Write produces, follows the magic with three sections, each a big-endian uint32
// length and that many bytes, containing the canonical descriptor JSON, bytecode, and
// types export (which may be empty), respectively. Version 1 files hold the descriptor
// section followed by bytecode running to the end of the file and carry no types export;
// they are still read. Every declared length is checked against the bytes that remain,
// and Resolve refuses files larger than the configured limit (see WithMaxBundleBytes).
//
// # Concurrency
//
// Resolve is safe for concurrent use. The filesystem itself is the shared mutable state;
// writes performed concurrently with reads of the same path produce undefined results,
// matching standard POSIX semantics. Hosts that mutate the directory tree should quiesce
// readers or write atomically (temp + rename).
package modules_provider_filesystem
