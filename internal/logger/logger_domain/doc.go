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

// Package logger_domain defines the [Logger] interface and implements
// structured logging with OpenTelemetry integration.
//
// It handles automatic trace correlation, span lifecycle management,
// configurable log levels, error notification batching, and a
// pluggable integration system for third-party observability services.
// Context propagation helpers ([WithLogger], [From], [MustFrom])
// thread loggers through call chains.
//
// # Log levels
//
// Piko uses a seven-level hierarchy with clear separation between
// framework internals and user application logs. TRACE (-8) covers
// framework loop internals and per-node processing. INTERNAL (-6) is
// for service registration, cache ops, and adapter lifecycle. DEBUG
// (-4) is for user application debugging. INFO (0) is the production
// default for normal operational events. NOTICE (2) marks critical
// lifecycle events. WARN (4) covers recoverable issues and deprecated
// features. ERROR (8) indicates failures requiring attention.
//
// # Usage
//
// Obtain a logger for a package using [GetLogger]:
//
//	var log = logger_domain.GetLogger("piko.sh/piko/internal/mypkg")
//
//	func DoWork(ctx context.Context) error {
//	    return log.RunInSpan(ctx, "DoWork", func(ctx context.Context, l logger_domain.Logger) error {
//	        l.Info("Processing request", logger_domain.String("id", "123"))
//	        return nil
//	    })
//	}
//
// Retrieve a logger from context using [From]:
//
//	func HandleRequest(ctx context.Context) {
//	    ctx, l := logger_domain.From(ctx, log) // Falls back to package logger
//	    l.Info("Handling request")
//	}
//
// # Environment configuration
//
// The PIKO_LOG_LEVEL environment variable overrides the log level.
// It accepts level names (e.g. "trace") or numeric values (e.g.
// "-8").
//
// # Thread safety
//
// All [Logger] methods are safe for concurrent use. The LogFactory
// and [lifecycleManager] use appropriate synchronisation for their
// mutable state. The [NotificationHandler] uses mutex-protected
// shared state across handler instances created by WithAttrs and
// WithGroup.
package logger_domain
