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

// Package logger_state provides shared state management for the
// logger facade.
//
// Centralises the global logging configuration, including handler
// registration, handler composition, shutdown lifecycle, and shared HTTP
// client management. Used by logger sub-packages (output providers,
// notification integrations) to register themselves into a unified handler
// chain without circular dependencies.
//
// Handlers and wrappers are composed into a chain that feeds into the global
// [log/slog] default logger. Destination handlers receive log records directly,
// whilst wrapper factories decorate the composed handler to add cross-cutting
// behaviour. When multiple destination handlers are registered, they are
// combined into a multi-handler that fans out records to all destinations.
//
// On package init, a default pretty-print handler writing to stdout is
// configured automatically. All exported functions are safe for concurrent use.
package logger_state
