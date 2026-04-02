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

// Package lsp_domain orchestrates the Language Server Protocol
// implementation for Piko.
//
// It handles LSP requests and responses for code intelligence in .pk
// template files: completion, hover, go-to-definition, diagnostics,
// signature help, quick fixes, inlay hints, type hierarchy, and
// refactoring. Port interfaces (LSPServerPort, WorkspacePort,
// TypeInspectorPort) are defined here for adapters and test doubles.
// The package coordinates with the annotator, inspector, and coordinator
// modules for semantic analysis, whilst lsp_adapters handles transport
// (stdio/TCP).
//
// # Thread safety
//
// Server and workspace methods are safe for concurrent use. Document
// instances are immutable snapshots and may be shared between
// goroutines. The workspace uses mutex protection for document cache
// operations and cancellation tracking for in-flight analyses.
// DocumentCache is safe for concurrent use.
package lsp_domain
