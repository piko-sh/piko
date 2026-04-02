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

// Package typegen_adapters implements TypeScript code emission,
// action manifest serialisation, and embedded type definition
// management for the typegen module.
//
// # Serialisation
//
// Action manifests can be serialised as FlatBuffer (high-performance
// binary format with pooled builders) or JSON (human-readable with
// camelCase keys for TypeScript/JavaScript compatibility). The
// FlatBuffer deserialiser uses [mem.String] for zero-copy string
// access; callers must not modify the source byte slice while the
// returned DTO is in use.
//
// # Thread safety
//
// The FlatBuffer builder pool ([GetBuilder], [PutBuilder]) is safe
// for concurrent use. The emitters are stateless and safe to share
// across goroutines.
package typegen_adapters
