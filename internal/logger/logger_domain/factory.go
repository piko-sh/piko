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

//go:build !bench

package logger_domain

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel"
)

// envLogLevel is the environment variable name for overriding the log level.
// When set, this takes precedence over any programmatically configured level.
//
// The level can be specified as a string ("trace", "internal", "debug",
// "info", "notice", "warn", "error") or a number (-8 for trace, -6 for
// internal, -4 for debug, 0 for info, and so on).
const envLogLevel = "PIKO_LOG_LEVEL"

// defaultFactory is the global logger factory used by GetLogger. It uses
// atomic.Pointer for safe concurrent access from InitDefaultFactory and
// GetLogger.
var defaultFactory atomic.Pointer[logFactory]

// logFactory creates logger instances with a shared base configuration.
// It provides a single place to control how loggers are made across the
// application.
type logFactory struct {
	// baseLogger is the shared logger used to create package-specific loggers.
	baseLogger *slog.Logger
}

// getLoggerForPackage creates a Logger instance for a specific package using
// this factory's configuration.
//
// The name parameter is used for OTEL tracer identification. Loggers created
// via the factory use dynamic default lookup, meaning they will automatically
// pick up handler changes made via AddPrettyOutput, AddJSONOutput, and similar
// methods.
//
// Takes name (string) which identifies the package for OTEL tracing.
//
// Returns Logger which provides logging with dynamic handler resolution.
func (f *logFactory) getLoggerForPackage(name string) Logger {
	tracer := otel.Tracer(name)
	return &slogLogger{
		logger:             f.baseLogger,
		tracer:             tracer,
		ctx:                context.Background(),
		hooks:              nil,
		hooksMutex:         sync.RWMutex{},
		stackTraceProvider: newRuntimeStackTraceProvider(),
		useDynamicDefault:  true,
		autoCaller:         true,
	}
}

// InitDefaultFactory sets up or updates the global defaultFactory with the
// given base logger.
//
// Takes baseLogger (*slog.Logger) which sets the base logger for the factory.
// If nil, slog.Default() is used.
func InitDefaultFactory(baseLogger *slog.Logger) {
	if baseLogger == nil {
		baseLogger = slog.Default()
	}
	defaultFactory.Store(&logFactory{baseLogger: baseLogger})
}

// GetLogger returns a Logger for the given package name.
//
// When the default factory is not set, it creates a standalone logger. When
// the default factory is set, it uses the factory to get the logger for the
// package.
//
// Takes name (string) which identifies the package, typically the full package
// path, used for tracing.
//
// Returns Logger which provides logging and tracing functions.
func GetLogger(name string) Logger {
	factory := defaultFactory.Load()
	if factory == nil {
		return NewLogger(
			slog.Default(),
			otel.Tracer(name),
			context.Background(),
		)
	}
	return factory.getLoggerForPackage(name)
}

// parseLogLevelFromEnv parses a log level from a string. It accepts both
// named levels (such as "debug", "warn", "error") and numeric values.
//
// Takes s (string) which is the log level name or numeric value to parse.
//
// Returns slog.Level which is the parsed level. Defaults to LevelInfo when
// the input is not recognised.
func parseLogLevelFromEnv(s string) slog.Level {
	if number, err := strconv.Atoi(s); err == nil {
		return slog.Level(number)
	}

	switch strings.ToLower(s) {
	case "trace":
		return LevelTrace
	case "internal":
		return LevelInternal
	case "debug":
		return slog.LevelDebug
	case "notice":
		return LevelNotice
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func init() {
	if levelString := os.Getenv(envLogLevel); levelString != "" {
		level := parseLogLevelFromEnv(levelString)
		levelVar := new(slog.LevelVar)
		levelVar.Set(level)

		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:       levelVar,
			AddSource:   false,
			ReplaceAttr: ReplaceLevelAttr,
		})

		logger := slog.New(handler)
		if envAttrs := EnvironmentSlogAttrs(); len(envAttrs) > 0 {
			logger = logger.With(SlogAttrsToAny(envAttrs)...)
		}
		slog.SetDefault(logger)
		InitDefaultFactory(logger)
		return
	}

	baseLogger := slog.Default()
	if envAttrs := EnvironmentSlogAttrs(); len(envAttrs) > 0 {
		baseLogger = baseLogger.With(SlogAttrsToAny(envAttrs)...)
	}
	InitDefaultFactory(baseLogger)
}
