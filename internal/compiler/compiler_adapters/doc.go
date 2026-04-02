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

// Package compiler_adapters implements [compiler_domain.InputReaderPort]
// for reading SFC (Single File Component) sources.
//
// Two backends are included:
//
//   - Disk: reads SFC files via a sandboxed filesystem
//     ([safedisk.Sandbox])
//   - Memory: stores and retrieves SFC content from an in-memory map,
//     primarily used for testing or dynamically generated sources
//
// # Thread safety
//
// The disk input reader delegates thread safety to the underlying
// sandbox. The memory input reader is safe for concurrent use; it
// uses a [sync.RWMutex] to protect its internal data store.
package compiler_adapters
