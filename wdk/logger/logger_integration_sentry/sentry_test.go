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

package logger_integration_sentry

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
)

func TestParseSlogLevels_EmptyReturnsNil(t *testing.T) {
	t.Parallel()

	require.Nil(t, parseSlogLevels(""))
}

func TestParseSlogLevels_KnownLevels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  []slog.Level
	}{
		{"trace", []slog.Level{slog.Level(-8)}},
		{"internal", []slog.Level{slog.Level(-6)}},
		{"debug", []slog.Level{slog.LevelDebug}},
		{"info", []slog.Level{slog.LevelInfo}},
		{"notice", []slog.Level{slog.Level(2)}},
		{"warn", []slog.Level{slog.LevelWarn}},
		{"error", []slog.Level{slog.LevelError}},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, parseSlogLevels(tc.input))
		})
	}
}

func TestParseSlogLevels_CommaSeparatedAndCaseInsensitive(t *testing.T) {
	t.Parallel()

	got := parseSlogLevels(" Debug, INFO ,WARN")

	require.Equal(t, []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn}, got)
}

func TestParseSlogLevels_UnknownValuesIgnored(t *testing.T) {
	t.Parallel()

	got := parseSlogLevels("nope,info,bogus")

	require.Equal(t, []slog.Level{slog.LevelInfo}, got)
}

func TestParseSentryConfig_FromPublicConfig(t *testing.T) {
	t.Parallel()

	cfg := Config{
		DSN:              "https://test@example.test/1",
		Environment:      "test",
		Release:          "v1.2.3",
		TracesSampleRate: 0.25,
		SampleRate:       0.5,
		Debug:            true,
	}

	parsed, err := parseSentryConfig(cfg)

	require.NoError(t, err)
	require.Equal(t, cfg.DSN, parsed.dsn)
	require.Equal(t, cfg.Environment, parsed.environment)
	require.Equal(t, cfg.Release, parsed.release)
	require.InDelta(t, cfg.TracesSampleRate, parsed.tracesSampleRate, 0.0001)
	require.InDelta(t, cfg.SampleRate, parsed.sampleRate, 0.0001)
	require.True(t, parsed.debug)
	require.True(t, parsed.enableTracing)
	require.False(t, parsed.sendDefaultPII)
	require.True(t, parsed.addSource)
}

func TestParseSentryConfig_FromDtoConfig(t *testing.T) {
	t.Parallel()

	cfg := &logger_dto.SentryConfig{
		DSN:              "https://test@example.test/2",
		Environment:      "production",
		Release:          "v2.0.0",
		EventLevel:       "warn,error",
		BreadcrumbLevel:  "info",
		IgnoreErrors:     []string{"context canceled"},
		TracesSampleRate: 0.1,
		SampleRate:       1.0,
		Debug:            false,
		EnableTracing:    true,
		SendDefaultPII:   true,
		AddSource:        false,
	}

	parsed, err := parseSentryConfig(cfg)

	require.NoError(t, err)
	require.Equal(t, cfg.DSN, parsed.dsn)
	require.Equal(t, cfg.Environment, parsed.environment)
	require.Equal(t, []string{"context canceled"}, parsed.ignoreErrors)
	require.Equal(t, []slog.Level{slog.LevelWarn, slog.LevelError}, parsed.eventLevels)
	require.Equal(t, []slog.Level{slog.LevelInfo}, parsed.breadcrumbLevels)
	require.True(t, parsed.enableTracing)
	require.True(t, parsed.sendDefaultPII)
	require.False(t, parsed.addSource)
}

func TestParseSentryConfig_NilDtoReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parseSentryConfig((*logger_dto.SentryConfig)(nil))

	require.Error(t, err)
}

func TestParseSentryConfig_UnsupportedTypeReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parseSentryConfig(42)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported config type")
}

func TestSentryIntegration_TypeIsSentry(t *testing.T) {
	t.Parallel()

	require.Equal(t, "sentry", (&sentryIntegration{}).Type())
}

func TestSentryIntegration_OtelComponents_NonNil(t *testing.T) {
	t.Parallel()

	processor, propagator := (&sentryIntegration{}).OtelComponents()

	require.NotNil(t, processor)
	require.NotNil(t, propagator)
}

func TestSentryIntegration_RegisteredOnImport(t *testing.T) {
	t.Parallel()

	require.NotNil(t, logger_domain.GetIntegration("sentry"))
}

func TestSentryIntegration_CreateHandler_RejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	integration := &sentryIntegration{}
	_, err := integration.CreateHandler(123)

	require.Error(t, err)
}

func TestFlushTimeoutFromContext_NoDeadlineUsesDefault(t *testing.T) {
	t.Parallel()

	got := flushTimeoutFromContext(context.Background())
	assert.Equal(t, defaultSentryFlushTimeout, got)
}

func TestFlushTimeoutFromContext_RemainingWithinCap(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 100*time.Millisecond,
		assert.AnError)
	defer cancel()

	got := flushTimeoutFromContext(ctx)
	assert.LessOrEqual(t, got, 100*time.Millisecond)
	assert.Greater(t, got, time.Duration(0))
}

func TestFlushTimeoutFromContext_CapsAtMaximum(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Minute,
		assert.AnError)
	defer cancel()

	got := flushTimeoutFromContext(ctx)
	assert.Equal(t, maxSentryFlushTimeout, got)
}

func TestFlushTimeoutFromContext_ExpiredDeadlineReturnsTinyPositive(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithDeadlineCause(context.Background(),
		time.Now().Add(-time.Second), assert.AnError)
	defer cancel()

	got := flushTimeoutFromContext(ctx)
	assert.Greater(t, got, time.Duration(0),
		"expired deadlines must still yield a positive timeout for sentry.Flush")
	assert.LessOrEqual(t, got, time.Millisecond)
}

func TestSentryShutdown_RespectsContextDeadline(t *testing.T) {
	t.Parallel()

	deadline := 100 * time.Millisecond
	ctx, cancel := context.WithTimeoutCause(context.Background(), deadline,
		assert.AnError)
	defer cancel()

	got := flushTimeoutFromContext(ctx)
	assert.LessOrEqual(t, got, deadline,
		"derived flush budget must not exceed the shutdown deadline")
}
