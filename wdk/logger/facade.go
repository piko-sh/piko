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

package logger

import (
	"log/slog"

	"piko.sh/piko/internal/logger/logger_domain"
)

// Logger is the main interface for Piko's logging system. It provides methods
// for logging at different levels and works with OpenTelemetry for tracing.
type Logger = logger_domain.Logger

// Attr represents a single key-value pair for adding structured context to log
// messages. It is an alias for the underlying slog.Attr type.
type Attr = logger_domain.Attr

const (
	// LevelTrace is the most verbose level for framework internals: loop
	// iterations, variable states.
	LevelTrace = logger_domain.LevelTrace

	// LevelDebug is for detailed debugging information in applications.
	LevelDebug = slog.LevelDebug

	// LevelInfo is the default level for general operational messages.
	LevelInfo = slog.LevelInfo

	// LevelNotice is for important events that sit between Info and Warn levels.
	LevelNotice = logger_domain.LevelNotice

	// LevelWarn is for recoverable issues and deprecation warnings.
	LevelWarn = slog.LevelWarn

	// LevelError is for errors that need attention and may trigger alerts.
	LevelError = slog.LevelError

	// FieldStrContext is the standard key for context attributes in log entries.
	FieldStrContext = logger_domain.FieldStrContext

	// FieldStrMethod is the standard key for HTTP method attributes.
	FieldStrMethod = logger_domain.FieldStrMethod

	// FieldStrComponent is the standard key for component name attributes.
	FieldStrComponent = logger_domain.FieldStrComponent

	// FieldStrAdapter is the standard key for adapter name attributes.
	FieldStrAdapter = logger_domain.FieldStrAdapter

	// FieldStrService is the standard key for service name attributes.
	FieldStrService = logger_domain.FieldStrService

	// FieldStrError is the standard key used for error attributes in log entries.
	FieldStrError = logger_domain.FieldStrError

	// FieldStrPath is the standard key for URL path attributes.
	FieldStrPath = logger_domain.FieldStrPath

	// FieldStrFile is the standard key for file path attributes.
	FieldStrFile = logger_domain.FieldStrFile

	// FieldStrDir is the standard key for directory path attributes.
	FieldStrDir = logger_domain.FieldStrDir
)

var (
	// GetLogger retrieves a logger for a specific package or component.
	GetLogger = logger_domain.GetLogger

	// From retrieves the logger from context, enriching it with the fallback if
	// no logger was previously stored.
	//
	// When a logger is already in the context (hot path), it returns the context
	// and logger unchanged with zero allocations. When no logger is found (cold
	// path), it stores the fallback in the context so that all downstream calls
	// to From find it immediately, costing one context.WithValue allocation.
	From = logger_domain.From

	// WithLogger stores a logger in the given context for later retrieval.
	//
	// Request-scoped data (such as request_id or user_id) then flows through the
	// call stack without passing the logger as a parameter.
	WithLogger = logger_domain.WithLogger

	// MustFrom retrieves the logger from context, panicking if not present.
	//
	// Use this in code paths where a logger MUST be in context (e.g., after
	// middleware that guarantees it). Panicking early catches middleware
	// misconfiguration during development.
	MustFrom = logger_domain.MustFrom

	// HasLogger reports whether a logger is stored in the context.
	HasLogger = logger_domain.HasLogger

	// String creates a string attribute for structured logging.
	String = logger_domain.String

	// Strings creates a string slice attribute for structured logging.
	Strings = logger_domain.Strings

	// Int creates an integer attribute for structured logging.
	Int = logger_domain.Int

	// Int64 creates a 64-bit integer attribute for structured logging.
	Int64 = logger_domain.Int64

	// Uint64 creates an unsigned 64-bit integer attribute for structured logging.
	Uint64 = logger_domain.Uint64

	// Float64 creates a 64-bit floating point attribute for structured logging.
	Float64 = logger_domain.Float64

	// Bool creates a boolean attribute for structured logging.
	Bool = logger_domain.Bool

	// Time creates a time.Time attribute for structured logging.
	Time = logger_domain.Time

	// Duration creates a time.Duration attribute for structured logging.
	Duration = logger_domain.Duration

	// Error creates an error attribute for structured logging.
	Error = logger_domain.Error

	// Field creates a custom attribute for structured logging.
	Field = logger_domain.Field
)
