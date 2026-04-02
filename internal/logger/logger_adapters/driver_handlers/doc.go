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

// Package driver_handlers implements [log/slog.Handler] for the Piko
// logging system, including pretty-printed console output with
// colours and OpenTelemetry integration for distributed tracing.
//
// For broadcasting to multiple handlers, use the standard library's
// [log/slog.NewMultiHandler].
//
// # Usage
//
// Create a pretty handler for development:
//
//	handler := driver_handlers.NewPrettyHandler(os.Stdout,
//		&driver_handlers.Options{
//			Level:     slog.LevelDebug,
//			AddSource: true,
//		})
//
// Use [log/slog.NewMultiHandler] to log to multiple destinations:
//
//	multi := slog.NewMultiHandler(
//		prettyHandler, jsonHandler)
//
// # Thread safety
//
// All handler implementations are safe for concurrent use. Each
// call to WithAttrs or WithGroup returns a new handler instance.
package driver_handlers
