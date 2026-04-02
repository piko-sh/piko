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

// Package logger provides the public API for Piko's structured,
// context-aware logging system.
//
// This package is the primary entry point for all logging in Piko
// applications. It re-exports core types and functions from the
// internal logger implementation and provides convenience methods
// for configuring output handlers, integrations, and
// OpenTelemetry tracing.
//
// # Getting started
//
// Obtain a logger for your package using [GetLogger], then call
// its level methods (Trace, Debug, Info, Warn, Error) with
// structured attributes:
//
//	log := logger.GetLogger("mypackage")
//	log.Info("Request processed",
//	    logger.String("user_id", userID),
//	    logger.Int("status_code", 200),
//	)
//
// # Configuring outputs
//
// Add output handlers before or after [Initialise]. The package
// supports pretty-printed console output, JSON output, and
// rotating file output:
//
//	logger.AddPrettyOutput()
//	logger.AddJSONOutput(logger.WithLevel(slog.LevelDebug))
//	logger.AddFileOutput(ctx, "app", "/var/log/app.log",
//	    logger.WithLevel(slog.LevelError),
//	    logger.WithJSON(),
//	)
//
// Use [AddFileOutputOnly] when stdout/stderr must remain clean,
// such as in LSP servers.
//
// # Structured attributes
//
// All dynamic data should be passed as attributes rather than
// interpolated into log messages. The package provides typed
// attribute constructors: [String], [Int], [Int64], [Uint64],
// [Float64], [Bool], [Time], [Duration], [Error], and [Field].
//
// Use the standard field key constants ([FieldStrMethod],
// [FieldStrComponent], [FieldStrError], etc.) for consistent,
// queryable log entries.
//
// # Notification integrations
//
// Notification integrations are available in the
// logger_integration_* sub-packages. Import them explicitly so
// that their SDKs are only included in your binary when needed.
//
// # Thread safety
//
// All exported functions and the [Logger] interface are safe for
// concurrent use. Output handlers are managed through an
// internal mutex-protected registry.
package logger
