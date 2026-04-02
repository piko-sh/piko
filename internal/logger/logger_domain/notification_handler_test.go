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
	"context"
	"io"
	"log/slog"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

type recordingTransport struct {
	sendError  error
	batches    []map[string]*logger_domain.GroupedError
	sendCalled int
}

func (r *recordingTransport) SendGroupedErrors(ctx context.Context, batch map[string]*logger_domain.GroupedError) error {
	r.sendCalled++
	if r.sendError != nil {
		return r.sendError
	}

	batchCopy := make(map[string]*logger_domain.GroupedError, len(batch))
	maps.Copy(batchCopy, batch)
	r.batches = append(r.batches, batchCopy)
	return nil
}

func makeErrorRecord(message string) slog.Record {
	r := slog.NewRecord(time.Now(), slog.LevelError, message, 0)
	return r
}

func makeWarnRecord(message string) slog.Record {
	r := slog.NewRecord(time.Now(), slog.LevelWarn, message, 0)
	return r
}

func makeInfoRecord(message string) slog.Record {
	r := slog.NewRecord(time.Now(), slog.LevelInfo, message, 0)
	return r
}

func TestNotificationHandler_Batching(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	err = handler.Handle(context.Background(), makeErrorRecord("error 2"))
	require.NoError(t, err)

	assert.Empty(t, transport.batches, "batch should not be sent before debounce duration")

	clk.Advance(10 * time.Second)

	require.Len(t, transport.batches, 1, "one batch should be sent after debounce")
	batch := transport.batches[0]
	assert.Len(t, batch, 2, "batch should contain 2 unique errors")

	assert.Equal(t, 1, transport.sendCalled, "Send should be called once")
}

func TestNotificationHandler_Deduplication(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("duplicate error"))
	require.NoError(t, err)

	clk.Advance(1 * time.Second)

	err = handler.Handle(context.Background(), makeErrorRecord("duplicate error"))
	require.NoError(t, err)

	clk.Advance(1 * time.Second)

	err = handler.Handle(context.Background(), makeErrorRecord("duplicate error"))
	require.NoError(t, err)

	clk.Advance(10 * time.Second)

	require.Len(t, transport.batches, 1, "one batch should be sent")
	batch := transport.batches[0]
	assert.Len(t, batch, 1, "batch should contain 1 unique error")

	for _, grouped := range batch {
		assert.Equal(t, 3, grouped.Count, "error should have count of 3")
		assert.Equal(t, "duplicate error", grouped.LogRecord.Message)
	}
}

func TestNotificationHandler_MinLevel(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeInfoRecord("info message"))
	require.NoError(t, err)

	err = handler.Handle(context.Background(), makeWarnRecord("warn message"))
	require.NoError(t, err)

	err = handler.Handle(context.Background(), makeErrorRecord("error message"))
	require.NoError(t, err)

	clk.Advance(10 * time.Second)

	require.Len(t, transport.batches, 1, "one batch should be sent")
	batch := transport.batches[0]
	assert.Len(t, batch, 1, "batch should contain only the error-level message")

	for _, grouped := range batch {
		assert.Equal(t, "error message", grouped.LogRecord.Message)
		assert.Equal(t, slog.LevelError, grouped.LogRecord.Level)
	}
}

func TestNotificationHandler_Debouncing(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	clk.Advance(5 * time.Second)

	err = handler.Handle(context.Background(), makeErrorRecord("error 2"))
	require.NoError(t, err)

	assert.Empty(t, transport.batches, "batch should not be sent before 10 seconds")

	clk.Advance(5 * time.Second)

	require.Len(t, transport.batches, 1, "batch should be sent after 10 seconds")
	batch := transport.batches[0]
	assert.Len(t, batch, 2, "batch should contain 2 unique errors")
}

func TestNotificationHandler_MultipleBatches(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	clk.Advance(10 * time.Second)

	require.Len(t, transport.batches, 1, "first batch should be sent")

	err = handler.Handle(context.Background(), makeErrorRecord("error 2"))
	require.NoError(t, err)

	clk.Advance(10 * time.Second)

	require.Len(t, transport.batches, 2, "second batch should be sent")

	assert.Len(t, transport.batches[0], 1, "first batch should have 1 error")
	assert.Len(t, transport.batches[1], 1, "second batch should have 1 error")
}

func TestNotificationHandler_Shutdown(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	err = handler.Handle(context.Background(), makeErrorRecord("error 2"))
	require.NoError(t, err)

	assert.Empty(t, transport.batches, "no batches before shutdown")

	handler.Shutdown()

	require.Len(t, transport.batches, 1, "batch should be sent on shutdown")
	batch := transport.batches[0]
	assert.Len(t, batch, 2, "batch should contain both errors")
}

func TestNotificationHandler_GroupedErrorTimestamps(t *testing.T) {
	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	clk := clock.NewMockClock(startTime)
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("duplicate error"))
	require.NoError(t, err)

	clk.Advance(3 * time.Second)

	err = handler.Handle(context.Background(), makeErrorRecord("duplicate error"))
	require.NoError(t, err)

	clk.Advance(10 * time.Second)

	require.Len(t, transport.batches, 1, "one batch should be sent")
	batch := transport.batches[0]
	require.Len(t, batch, 1, "one unique error")

	for _, grouped := range batch {
		assert.Equal(t, startTime, grouped.FirstSeen, "FirstSeen should be 12:00:00")
		assert.Equal(t, startTime.Add(3*time.Second), grouped.LastSeen, "LastSeen should be 12:00:03")
		assert.Equal(t, 2, grouped.Count, "Count should be 2")
	}
}

func TestNotificationHandler_EmptyShutdown(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	handler.Shutdown()

	assert.Empty(t, transport.batches, "no batches should be sent on empty shutdown")
	assert.Equal(t, 0, transport.sendCalled, "Send should not be called")
}

func TestNotificationHandler_WithAttrs(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	handlerWithAttrs := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})

	err := handlerWithAttrs.Handle(context.Background(), makeErrorRecord("error with attrs"))
	require.NoError(t, err)

	clk.Advance(10 * time.Second)

	require.Len(t, transport.batches, 1, "batch should be sent")
}

func TestNotificationHandler_WithGroup(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	handlerWithGroup := handler.WithGroup("group1")

	err := handlerWithGroup.Handle(context.Background(), makeErrorRecord("error with group"))
	require.NoError(t, err)

	clk.Advance(10 * time.Second)

	require.Len(t, transport.batches, 1, "batch should be sent")
}

func TestNotificationHandler_SetDebounceDuration(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	handler.SetDebounceDuration(5 * time.Second)

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	clk.Advance(5 * time.Second)

	require.Len(t, transport.batches, 1, "batch should be sent after custom debounce duration")
}

func TestNewNotificationHandler(t *testing.T) {
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	handler := logger_domain.NewNotificationHandler(baseHandler, transport, slog.LevelError)

	require.NotNil(t, handler)

	err := handler.Handle(context.Background(), makeErrorRecord("test error"))
	require.NoError(t, err)

	assert.True(t, handler.HasPendingBatch())
}

func TestNewNotificationHandlerWithClock(t *testing.T) {
	mockClock := clock.NewMockClock(time.Now())
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	handler := logger_domain.NewNotificationHandlerWithClock(baseHandler, transport, slog.LevelError, mockClock)

	require.NotNil(t, handler)

	err := handler.Handle(context.Background(), makeErrorRecord("test error"))
	require.NoError(t, err)

	mockClock.Advance(11 * time.Second)

	require.Len(t, transport.batches, 1)
}
