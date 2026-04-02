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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestRuntimeStackTraceProvider_CapturesStackTrace(t *testing.T) {
	provider := logger_domain.NewRuntimeStackTraceProvider()

	pc, trace := provider.CaptureStackTrace(1, 64)
	defer trace.Release()

	assert.NotEqual(t, uintptr(0), pc, "program counter should be captured")
	assert.NotEmpty(t, trace.Frames(), "stack trace should not be empty")

	for _, frame := range trace.Frames() {
		assert.True(t, strings.HasPrefix(frame, "\t"), "frame should start with tab")
		assert.Contains(t, frame, ":", "frame should contain file:line")
	}
}

func TestRuntimeStackTraceProvider_SkipFrames(t *testing.T) {
	provider := logger_domain.NewRuntimeStackTraceProvider()

	pc1, trace1 := provider.CaptureStackTrace(1, 64)
	defer trace1.Release()
	pc2, trace2 := provider.CaptureStackTrace(2, 64)
	defer trace2.Release()

	assert.NotEqual(t, pc1, pc2, "different skip values should produce different PCs")
	assert.NotEqual(t, len(trace1.Frames()), len(trace2.Frames()), "different skip values should produce different trace lengths")
}

func TestRuntimeStackTraceProvider_MaxFrames(t *testing.T) {
	provider := logger_domain.NewRuntimeStackTraceProvider()

	_, trace := provider.CaptureStackTrace(1, 5)
	defer trace.Release()

	assert.LessOrEqual(t, len(trace.Frames()), 5, "trace should not exceed maxFrames")
}

func TestMockStackTraceProvider_ReturnsPredefinedValues(t *testing.T) {
	expectedPC := uintptr(12345)
	expectedFrames := []string{
		"\t/fake/file.go:10",
		"\t/fake/file.go:20",
		"\t/fake/file.go:30",
	}
	expectedTrace := logger_domain.NewStackTraceFromFrames(expectedFrames)

	provider := logger_domain.NewMockStackTraceProvider(expectedPC, expectedTrace)

	pc, trace := provider.CaptureStackTrace(0, 0)

	assert.Equal(t, expectedPC, pc, "should return predefined PC")
	assert.Equal(t, expectedFrames, trace.Frames(), "should return predefined trace")
}

func TestMockStackTraceProvider_IgnoresParameters(t *testing.T) {
	expectedPC := uintptr(99999)
	expectedFrames := []string{"\t/test.go:1"}
	expectedTrace := logger_domain.NewStackTraceFromFrames(expectedFrames)

	provider := logger_domain.NewMockStackTraceProvider(expectedPC, expectedTrace)

	pc1, trace1 := provider.CaptureStackTrace(0, 10)
	pc2, trace2 := provider.CaptureStackTrace(5, 100)

	assert.Equal(t, pc1, pc2, "mock should ignore skip parameter")
	assert.Equal(t, trace1.Frames(), trace2.Frames(), "mock should ignore maxFrames parameter")
	assert.Equal(t, expectedPC, pc1)
	assert.Equal(t, expectedFrames, trace1.Frames())
}

func TestLogger_WithMockStackTraceProvider(t *testing.T) {
	var buffer bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	mockPC := uintptr(42)
	mockTrace := logger_domain.NewStackTraceFromFrames([]string{
		"\t/mock/test.go:100",
		"\t/mock/test.go:200",
	})
	mockProvider := logger_domain.NewMockStackTraceProvider(mockPC, mockTrace)

	logger := logger_domain.NewLoggerWithStackTraceProvider(
		baseLogger,
		otel.Tracer("test"),
		mockProvider,
	)

	logger.Error("test error")

	output := buffer.String()
	assert.Contains(t, output, "/mock/test.go:100", "should use mock stack trace")
	assert.Contains(t, output, "/mock/test.go:200", "should use mock stack trace")
}

func TestLogger_StackTraceProvider_PropagatedToWithLogger(t *testing.T) {
	mockProvider := logger_domain.NewMockStackTraceProvider(
		uintptr(123),
		logger_domain.NewStackTraceFromFrames([]string{"\t/test.go:1"}),
	)

	var buffer bytes.Buffer
	derivedWithHandler := logger_domain.NewLoggerWithStackTraceProvider(
		slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelError})),
		otel.Tracer("test"),
		mockProvider,
	).With(logger_domain.String("key", "value"))

	derivedWithHandler.(interface {
		Error(string, ...logger_domain.Attr)
	}).Error("test")

	output := buffer.String()
	assert.Contains(t, output, "/test.go:1", "derived logger should use same provider")
}

func TestLogger_ErrorLogging_UsesStackTrace(t *testing.T) {
	var buffer bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	logger.Error("test error message", logger_domain.String("key", "value"))

	output := buffer.String()
	assert.Contains(t, output, "test error message", "should contain error message")
	assert.Contains(t, output, "stack_trace", "should include stack trace")
}

func TestLogger_InfoLogging_NoStackTrace(t *testing.T) {
	var buffer bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	logger.Info("test info message")

	output := buffer.String()
	assert.Contains(t, output, "test info message", "should contain info message")
	assert.NotContains(t, output, "stack_trace", "info logs should not include stack trace")
}

func TestLogger_PanicLogging_UsesStackTrace(t *testing.T) {
	var buffer bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	defer func() {
		r := recover()
		require.NotNil(t, r, "should panic")

		output := buffer.String()
		assert.Contains(t, output, "test panic", "should contain panic message")
		assert.Contains(t, output, "stack_trace", "panic should include stack trace")
	}()

	logger.Panic("test panic")
}

func TestStackTrace_EmptyProvider(t *testing.T) {
	emptyProvider := logger_domain.NewMockStackTraceProvider(0, logger_domain.StackTrace{})

	var buffer bytes.Buffer
	baseLogger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	logger := logger_domain.NewLoggerWithStackTraceProvider(
		baseLogger,
		otel.Tracer("test"),
		emptyProvider,
	)

	logger.Error("error with empty stack")

	output := buffer.String()
	assert.Contains(t, output, "error with empty stack", "should still log the message")

}

func BenchmarkCaptureStackTrace_Pooled(b *testing.B) {
	const maxFrames = 32
	const skipFrames = 3

	provider := logger_domain.NewRuntimeStackTraceProvider()

	for range 10 {
		_, trace := provider.CaptureStackTrace(skipFrames, maxFrames)
		trace.Release()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, trace := provider.CaptureStackTrace(skipFrames, maxFrames)
		trace.Release()
	}
}

func BenchmarkCaptureStackTrace_NoRelease(b *testing.B) {
	const maxFrames = 32
	const skipFrames = 3

	provider := logger_domain.NewRuntimeStackTraceProvider()

	for range 10 {
		_, trace := provider.CaptureStackTrace(skipFrames, maxFrames)
		trace.Release()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {

		_, _ = provider.CaptureStackTrace(skipFrames, maxFrames)
	}
}

func BenchmarkStackTrace_Release(b *testing.B) {
	provider := logger_domain.NewRuntimeStackTraceProvider()
	_, trace := provider.CaptureStackTrace(3, 32)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		trace.Release()
		_, trace = provider.CaptureStackTrace(3, 32)
	}
}
