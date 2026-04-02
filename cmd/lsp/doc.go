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

// Package main implements the Piko LSP server binary.
//
// This command starts a Language Server Protocol (LSP) server that
// provides IDE features for Piko templates, including hover
// information, autocompletion, go-to-definition, diagnostics,
// and document symbols. It bootstraps the dependency injection
// container and wires up LSP-specific adapters before running the
// server.
//
// # Transport modes
//
// The server supports two transport modes:
//
//   - stdio: Communicates via stdin/stdout (default). Suitable for
//     editor integrations that launch the server as a subprocess.
//   - tcp: Listens on a configurable TCP address. Useful for
//     debugging and remote connections.
//
// The mode is selected via the --tcp flag or the PIKO_LSP_DRIVER
// environment variable. TCP address is configured via --host/--port
// flags or the PIKO_LSP_TCP_ADDR environment variable.
//
// # Flags
//
//   - --tcp: Use TCP mode instead of stdio
//   - --host: TCP host to bind to (default 127.0.0.1)
//   - --port: TCP port to listen on (default 4389)
//   - --formatting: Enable document formatting capabilities
//   - --file-logging: Enable file logging to /tmp/piko-lsp-<pid>.log
//   - --pprof: Enable pprof profiling server
//   - --pprof-port: Port for the pprof HTTP server (default 6060)
//
// # Integration
//
// The binary bootstraps a [bootstrap.Container] and overrides
// several service implementations with LSP-specific adapters:
//
//   - File system reads are routed through a document cache so
//     the server operates on in-memory buffer contents.
//   - Diagnostic output is silenced to prevent compiler diagnostics
//     from interfering with LSP protocol messages.
//   - The render registry is replaced with a no-op implementation
//     since rendering is not required during language analysis.
package main
