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

package driver_handlers

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
)

func newTestHandler(buffer *bytes.Buffer, level slog.Level) slog.Handler {
	return NewPrettyHandler(buffer, &Options{
		Level:    &slog.LevelVar{},
		NoColour: true,
	})
}

func newTestHandlerWithLevel(buffer *bytes.Buffer, level slog.Level) slog.Handler {
	levelVar := new(slog.LevelVar)
	levelVar.Set(level)
	return NewPrettyHandler(buffer, &Options{
		Level:    levelVar,
		NoColour: true,
	})
}

func newTestRecord(level slog.Level, message string) slog.Record {
	return slog.NewRecord(time.Date(2026, 3, 27, 14, 30, 45, 0, time.UTC), level, message, 0)
}

func TestStrPad_RightPadding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "shorter than target",
			input:    "hi",
			length:   5,
			expected: "hi   ",
		},
		{
			name:     "exact length",
			input:    "hello",
			length:   5,
			expected: "hello",
		},
		{
			name:     "longer than target",
			input:    "hello world",
			length:   5,
			expected: "hello world",
		},
		{
			name:     "empty input",
			input:    "",
			length:   3,
			expected: "   ",
		},
		{
			name:     "zero pad length",
			input:    "hi",
			length:   0,
			expected: "hi",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := strPad(testCase.input, testCase.length, " ", "RIGHT")
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestStrPad_LeftPadding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "shorter than target",
			input:    "hi",
			length:   5,
			expected: "   hi",
		},
		{
			name:     "exact length",
			input:    "hello",
			length:   5,
			expected: "hello",
		},
		{
			name:     "longer than target",
			input:    "hello world",
			length:   5,
			expected: "hello world",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := strPad(testCase.input, testCase.length, " ", "LEFT")
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestStrPad_BothPadding(t *testing.T) {
	t.Parallel()

	result := strPad("hi", 6, " ", "BOTH")
	assert.Len(t, result, 6)
	assert.Contains(t, result, "hi")
}

func TestStrPad_CustomPadCharacter(t *testing.T) {
	t.Parallel()

	result := strPad("hi", 6, "-", "RIGHT")
	assert.Equal(t, "hi----", result)
}

func TestHandle_IncludesMessageText(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "something happened")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "something happened")
}

func TestHandle_DebugLevel(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandlerWithLevel(&buffer, slog.LevelDebug)
	record := newTestRecord(slog.LevelDebug, "debug message")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "DEBUG")
	assert.Contains(t, output, "debug message")
}

func TestHandle_InfoLevel(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "info message")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "info message")
}

func TestHandle_WarnLevel(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelWarn, "warn message")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "WARN")
	assert.Contains(t, output, "warn message")
}

func TestHandle_ErrorLevel(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelError, "error message")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "ERROR")
	assert.Contains(t, output, "error message")
}

func TestHandle_IncludesTimeFormatted(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "time check")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "14:30:45")
}

func TestHandle_IncludesErrorField(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelError, "operation failed")
	record.AddAttrs(slog.String(logger_domain.KeyError, "connection refused"))

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "connection refused")
}

func TestHandle_IncludesMultipleAttrs(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "request")
	record.AddAttrs(
		slog.String("request_id", "abc-123"),
		slog.String("user_id", "usr-456"),
	)

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "request_id")
	assert.Contains(t, output, "abc-123")
	assert.Contains(t, output, "user_id")
	assert.Contains(t, output, "usr-456")
}

func TestHandle_IncludesContextAttr(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "hello")
	record.AddAttrs(slog.String(logger_domain.KeyContext, "my_package"))

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "my_package")
}

func TestHandle_IncludesMethodAttr(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "hello")
	record.AddAttrs(slog.String(logger_domain.KeyMethod, "HandleRequest"))

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "HandleRequest")
}

func TestHandle_OutputEndsWithNewline(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "newline check")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.True(t, buffer.Len() > 0)
	assert.Equal(t, byte('\n'), buffer.Bytes()[buffer.Len()-1])
}

func TestHandle_FieldSeparator(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "separator check")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), " | ")
}

func TestHandle_SourceIncludedWhenEnabled(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	levelVar := new(slog.LevelVar)
	levelVar.Set(slog.LevelInfo)
	handler := NewPrettyHandler(&buffer, &Options{
		Level:     levelVar,
		AddSource: true,
		NoColour:  true,
	})

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "source check", 0)

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.NotEmpty(t, buffer.String())
}

func TestHandle_SourceOmittedWhenDisabled(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)

	record := newTestRecord(slog.LevelInfo, "no source")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	output := buffer.String()
	assert.NotContains(t, output, ".go:")
}

func TestHandle_ExtraAttrsAreSorted(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "sorted")
	record.AddAttrs(
		slog.String("zebra", "z"),
		slog.String("alpha", "a"),
	)

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	output := buffer.String()
	alphaIndex := bytes.Index([]byte(output), []byte("alpha"))
	zebraIndex := bytes.Index([]byte(output), []byte("zebra"))
	assert.Greater(t, alphaIndex, -1)
	assert.Greater(t, zebraIndex, -1)
	assert.Less(t, alphaIndex, zebraIndex)
}

func TestWithAttrs_ReturnsNewHandler(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)

	newHandler := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})

	assert.NotNil(t, newHandler)
	assert.NotSame(t, handler, newHandler)
}

func TestWithAttrs_EmptySliceReturnsSameHandler(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)

	newHandler := handler.WithAttrs([]slog.Attr{})

	assert.Equal(t, handler, newHandler)
}

func TestWithAttrs_AttrsAppearInOutput(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	handlerWithAttrs := handler.WithAttrs([]slog.Attr{slog.String("service", "api")})

	record := newTestRecord(slog.LevelInfo, "with attrs")
	err := handlerWithAttrs.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "service")
	assert.Contains(t, buffer.String(), "api")
}

func TestWithGroup_ReturnsNewHandler(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)

	newHandler := handler.WithGroup("request")

	assert.NotNil(t, newHandler)
	assert.NotSame(t, handler, newHandler)
}

func TestWithGroup_GroupPrefixAppearsInOutput(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	handlerWithGroup := handler.WithGroup("request")
	handlerWithGroupAndAttrs := handlerWithGroup.WithAttrs([]slog.Attr{slog.String("id", "42")})

	record := newTestRecord(slog.LevelInfo, "group test")
	err := handlerWithGroupAndAttrs.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "request.id")
}

func TestEnabled_RespectsMinimumLevel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		minimumLevel slog.Level
		checkLevel   slog.Level
		expected     bool
	}{
		{
			name:         "info handler enables info",
			minimumLevel: slog.LevelInfo,
			checkLevel:   slog.LevelInfo,
			expected:     true,
		},
		{
			name:         "info handler enables warn",
			minimumLevel: slog.LevelInfo,
			checkLevel:   slog.LevelWarn,
			expected:     true,
		},
		{
			name:         "info handler enables error",
			minimumLevel: slog.LevelInfo,
			checkLevel:   slog.LevelError,
			expected:     true,
		},
		{
			name:         "info handler disables debug",
			minimumLevel: slog.LevelInfo,
			checkLevel:   slog.LevelDebug,
			expected:     false,
		},
		{
			name:         "warn handler disables info",
			minimumLevel: slog.LevelWarn,
			checkLevel:   slog.LevelInfo,
			expected:     false,
		},
		{
			name:         "error handler disables warn",
			minimumLevel: slog.LevelError,
			checkLevel:   slog.LevelWarn,
			expected:     false,
		},
		{
			name:         "debug handler enables debug",
			minimumLevel: slog.LevelDebug,
			checkLevel:   slog.LevelDebug,
			expected:     true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			handler := newTestHandlerWithLevel(&buffer, testCase.minimumLevel)

			result := handler.Enabled(context.Background(), testCase.checkLevel)

			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestNewPrettyHandler_NilOptionsUsesDefaults(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := NewPrettyHandler(&buffer, nil)

	require.NotNil(t, handler)
	assert.True(t, handler.Enabled(context.Background(), slog.LevelInfo))
	assert.False(t, handler.Enabled(context.Background(), slog.LevelDebug))
}

func TestNewPrettyHandler_ImplementsHandlerInterface(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := NewPrettyHandler(&buffer, nil)

	var _ slog.Handler = handler
}

func TestHandle_StackTraceStringValue(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelError, "panic")
	record.AddAttrs(slog.String("stack_trace", "frame1.go:10\nframe2.go:20"))

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "Stack trace:")
	assert.Contains(t, output, "frame1.go:10")
	assert.Contains(t, output, "frame2.go:20")
}

func TestHandle_CustomLevelNames(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		level         slog.Level
		expectedLabel string
	}{
		{
			name:          "trace level",
			level:         logger_domain.LevelTrace,
			expectedLabel: "TRACE",
		},
		{
			name:          "internal level",
			level:         logger_domain.LevelInternal,
			expectedLabel: "INTERNAL",
		},
		{
			name:          "notice level",
			level:         logger_domain.LevelNotice,
			expectedLabel: "NOTICE",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			handler := newTestHandlerWithLevel(&buffer, logger_domain.LevelTrace)
			record := newTestRecord(testCase.level, "custom level test")

			err := handler.Handle(context.Background(), record)

			require.NoError(t, err)
			assert.Contains(t, buffer.String(), testCase.expectedLabel)
		})
	}
}

func TestHandle_MissingFieldShowsTilde(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "tilde test")

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "~")
}

func TestWithAttrs_DoesNotMutateOriginal(t *testing.T) {
	t.Parallel()

	var bufferOriginal bytes.Buffer
	var bufferNew bytes.Buffer
	original := newTestHandler(&bufferOriginal, slog.LevelInfo)
	_ = original.WithAttrs([]slog.Attr{slog.String("added", "value")})

	record := newTestRecord(slog.LevelInfo, "original handler")
	err := original.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.NotContains(t, bufferOriginal.String(), "added")
	_ = bufferNew
}

func TestWithGroup_DoesNotMutateOriginal(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	original := newTestHandler(&buffer, slog.LevelInfo)
	_ = original.WithGroup("new_group")

	record := newTestRecord(slog.LevelInfo, "original handler")
	err := original.Handle(context.Background(), record)

	require.NoError(t, err)
	assert.NotContains(t, buffer.String(), "new_group")
}

func TestHandle_GroupedAttrFromRecord(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	handler := newTestHandler(&buffer, slog.LevelInfo)
	record := newTestRecord(slog.LevelInfo, "nested")
	record.AddAttrs(slog.Group("outer", slog.String("inner", "value")))

	err := handler.Handle(context.Background(), record)

	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "outer")
	assert.Contains(t, output, "inner")
	assert.Contains(t, output, "value")
}
