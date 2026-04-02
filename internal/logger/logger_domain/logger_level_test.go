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

package logger_domain_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestLogger_RespectsConfiguredLogLevel(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		configLevel slog.Level
		logLevel    slog.Level
		shouldLog   bool
	}{
		{
			name:        "Info level allows Info logs",
			configLevel: slog.LevelInfo,
			logLevel:    slog.LevelInfo,
			shouldLog:   true,
			message:     "info message",
		},
		{
			name:        "Info level blocks Debug logs",
			configLevel: slog.LevelInfo,
			logLevel:    slog.LevelDebug,
			shouldLog:   false,
			message:     "debug message",
		},
		{
			name:        "Info level allows Warn logs",
			configLevel: slog.LevelInfo,
			logLevel:    slog.LevelWarn,
			shouldLog:   true,
			message:     "warn message",
		},
		{
			name:        "Info level allows Error logs",
			configLevel: slog.LevelInfo,
			logLevel:    slog.LevelError,
			shouldLog:   true,
			message:     "error message",
		},
		{
			name:        "Warn level blocks Info logs",
			configLevel: slog.LevelWarn,
			logLevel:    slog.LevelInfo,
			shouldLog:   false,
			message:     "info message",
		},
		{
			name:        "Warn level allows Warn logs",
			configLevel: slog.LevelWarn,
			logLevel:    slog.LevelWarn,
			shouldLog:   true,
			message:     "warn message",
		},
		{
			name:        "Error level blocks Warn logs",
			configLevel: slog.LevelError,
			logLevel:    slog.LevelWarn,
			shouldLog:   false,
			message:     "warn message",
		},
		{
			name:        "Error level allows Error logs",
			configLevel: slog.LevelError,
			logLevel:    slog.LevelError,
			shouldLog:   true,
			message:     "error message",
		},
		{
			name:        "Debug level allows Debug logs",
			configLevel: slog.LevelDebug,
			logLevel:    slog.LevelDebug,
			shouldLog:   true,
			message:     "debug message",
		},
		{
			name:        "Debug level allows Info logs",
			configLevel: slog.LevelDebug,
			logLevel:    slog.LevelInfo,
			shouldLog:   true,
			message:     "info message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer

			handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				Level: tt.configLevel,
			})

			baseLogger := slog.New(handler)
			logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

			switch tt.logLevel {
			case slog.LevelDebug:
				logger.Debug(tt.message)
			case slog.LevelInfo:
				logger.Info(tt.message)
			case slog.LevelWarn:
				logger.Warn(tt.message)
			case slog.LevelError:
				logger.Error(tt.message)
			}

			output := buffer.String()

			if tt.shouldLog {
				assert.Contains(t, output, tt.message, "log should be written")
			} else {
				assert.Empty(t, output, "log should be filtered by level")
			}
		})
	}
}

func TestLogger_LevelPropagatedToWithLogger(t *testing.T) {
	var buffer bytes.Buffer

	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	derivedLogger := logger.With(logger_domain.String("key", "value"))

	derivedLogger.Info("info message")
	assert.Empty(t, buffer.String(), "info should be filtered")

	derivedLogger.Warn("warn message")
	assert.Contains(t, buffer.String(), "warn message", "warn should be logged")
}

func TestLogger_LevelPropagatedToWithContext(t *testing.T) {
	var buffer bytes.Buffer

	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: slog.LevelError,
	})

	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	derivedLogger := logger.WithContext(logger.GetContext())

	derivedLogger.Warn("warn message")
	assert.Empty(t, buffer.String(), "warn should be filtered")

	derivedLogger.Error("error message")
	assert.Contains(t, buffer.String(), "error message", "error should be logged")
}

func TestLogger_CustomLevels_Trace(t *testing.T) {
	var buffer bytes.Buffer

	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: logger_domain.LevelTrace,
	})

	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	logger.Trace("trace message")
	assert.Contains(t, buffer.String(), "trace message", "trace should be logged")
	buffer.Reset()

	logger.Debug("debug message")
	assert.Contains(t, buffer.String(), "debug message", "debug should be logged")
}

func TestLogger_CustomLevels_Notice(t *testing.T) {
	var buffer bytes.Buffer

	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: logger_domain.LevelNotice,
	})

	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	logger.Info("info message")
	assert.Empty(t, buffer.String(), "info should be filtered")

	logger.Notice("notice message")
	assert.Contains(t, buffer.String(), "notice message", "notice should be logged")
	buffer.Reset()

	logger.Warn("warn message")
	assert.Contains(t, buffer.String(), "warn message", "warn should be logged")
}

func TestLogger_WithMockStackTraceProvider_RespectsLevel(t *testing.T) {
	var buffer bytes.Buffer

	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: slog.LevelError,
	})

	baseLogger := slog.New(handler)

	mockProvider := logger_domain.NewMockStackTraceProvider(
		uintptr(123),
		logger_domain.NewStackTraceFromFrames([]string{"\t/test.go:1"}),
	)

	logger := logger_domain.NewLoggerWithStackTraceProvider(
		baseLogger,
		otel.Tracer("test"),
		mockProvider,
	)

	logger.Info("info message")
	assert.Empty(t, buffer.String(), "info should be filtered by level")

	logger.Error("error message")
	output := buffer.String()
	assert.Contains(t, output, "error message", "error should be logged")
	assert.Contains(t, output, "/test.go:1", "mock stack trace should be included")
}

func TestLevelName_ReturnsCorrectNames(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		level    slog.Level
	}{
		{
			name:     "Trace level returns TRACE",
			expected: "TRACE",
			level:    logger_domain.LevelTrace,
		},
		{
			name:     "Internal level returns INTERNAL",
			expected: "INTERNAL",
			level:    logger_domain.LevelInternal,
		},
		{
			name:     "Debug level returns DEBUG",
			expected: "DEBUG",
			level:    slog.LevelDebug,
		},
		{
			name:     "Info level returns INFO",
			expected: "INFO",
			level:    slog.LevelInfo,
		},
		{
			name:     "Notice level returns NOTICE",
			expected: "NOTICE",
			level:    logger_domain.LevelNotice,
		},
		{
			name:     "Warn level returns WARN",
			expected: "WARN",
			level:    slog.LevelWarn,
		},
		{
			name:     "Error level returns ERROR",
			expected: "ERROR",
			level:    slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger_domain.LevelName(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceLevelAttr_FormatsCustomLevels(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		level    slog.Level
	}{
		{
			name:     "Trace level formatted as TRACE",
			expected: "TRACE",
			level:    logger_domain.LevelTrace,
		},
		{
			name:     "Internal level formatted as INTERNAL",
			expected: "INTERNAL",
			level:    logger_domain.LevelInternal,
		},
		{
			name:     "Notice level formatted as NOTICE",
			expected: "NOTICE",
			level:    logger_domain.LevelNotice,
		},
		{
			name:     "Info level formatted as INFO",
			expected: "INFO",
			level:    slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			attr := slog.Any(slog.LevelKey, tt.level)

			result := logger_domain.ReplaceLevelAttr(nil, attr)
			assert.Equal(t, tt.expected, result.Value.String())
		})
	}
}

func TestReplaceLevelAttr_PreservesNonLevelAttrs(t *testing.T) {
	attr := slog.String("message", "test message")
	result := logger_domain.ReplaceLevelAttr(nil, attr)

	assert.Equal(t, "message", result.Key)
	assert.Equal(t, "test message", result.Value.String())
}

func TestJSONHandler_WithReplaceLevelAttr_FormatsCustomLevels(t *testing.T) {
	var buffer bytes.Buffer

	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level:       logger_domain.LevelTrace,
		ReplaceAttr: logger_domain.ReplaceLevelAttr,
	})

	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	logger.Trace("trace message")
	output := buffer.String()
	assert.Contains(t, output, `"level":"TRACE"`, "TRACE level should be formatted correctly")
	buffer.Reset()

	logger.Internal("internal message")
	output = buffer.String()
	assert.Contains(t, output, `"level":"INTERNAL"`, "INTERNAL level should be formatted correctly")
	buffer.Reset()

	logger.Notice("notice message")
	output = buffer.String()
	assert.Contains(t, output, `"level":"NOTICE"`, "NOTICE level should be formatted correctly")
}

func TestLogger_CustomLevels_Internal(t *testing.T) {
	var buffer bytes.Buffer

	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: logger_domain.LevelInternal,
	})

	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	logger.Trace("trace message")
	assert.Empty(t, buffer.String(), "trace should be filtered")

	logger.Internal("internal message")
	assert.Contains(t, buffer.String(), "internal message", "internal should be logged")
	buffer.Reset()

	logger.Debug("debug message")
	assert.Contains(t, buffer.String(), "debug message", "debug should be logged")
}

func TestLevelOrdering_TwoZoneModel(t *testing.T) {

	assert.Less(t, int(logger_domain.LevelTrace), int(logger_domain.LevelInternal),
		"TRACE should be less verbose than INTERNAL")
	assert.Less(t, int(logger_domain.LevelInternal), int(slog.LevelDebug),
		"INTERNAL should be less verbose than DEBUG")
	assert.Less(t, int(slog.LevelDebug), int(slog.LevelInfo),
		"DEBUG should be less verbose than INFO")
	assert.Less(t, int(slog.LevelInfo), int(logger_domain.LevelNotice),
		"INFO should be less verbose than NOTICE")
	assert.Less(t, int(logger_domain.LevelNotice), int(slog.LevelWarn),
		"NOTICE should be less verbose than WARN")
	assert.Less(t, int(slog.LevelWarn), int(slog.LevelError),
		"WARN should be less verbose than ERROR")
}

func TestLogger_DebugLevelFiltersFrameworkLevels(t *testing.T) {

	var buffer bytes.Buffer

	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	logger.Trace("framework loop internal")
	assert.Empty(t, buffer.String(), "TRACE should be filtered at DEBUG level")

	logger.Internal("framework surface operation")
	assert.Empty(t, buffer.String(), "INTERNAL should be filtered at DEBUG level")

	logger.Debug("user debug message")
	assert.Contains(t, buffer.String(), "user debug message", "DEBUG should be logged")
	buffer.Reset()

	logger.Info("user info message")
	assert.Contains(t, buffer.String(), "user info message", "INFO should be logged")
}
