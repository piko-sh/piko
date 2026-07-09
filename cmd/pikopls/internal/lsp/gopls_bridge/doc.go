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

// Package gopls_bridge spawns and drives gopls as a child language server so that pikopls
// can offer full Go intelligence inside the embedded <script type="application/x-go">
// blocks of .pk files.
//
// pikopls stays the single language server an editor talks to. For requests whose cursor
// lands in a Go block, pikopls acts as gopls's LSP client: it presents each block to
// gopls as a never-on-disk file overlay in its own synthetic package directory, forwards
// the request, and maps positions and results back into .pk coordinates. The package owns
// one gopls process per Go module root, shared across every editor connection.
//
// The package never imports lsp_domain; the higher layer extracts the primitives gopls
// needs (module root, virtual document content, mapped position) and calls into the
// Manager.
package gopls_bridge
