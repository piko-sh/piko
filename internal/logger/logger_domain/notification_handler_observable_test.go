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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

func TestNotificationHandler_GetPendingErrorCount(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	assert.Equal(t, 0, handler.GetPendingErrorCount(), "should start with 0 pending errors")

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	assert.Equal(t, 1, handler.GetPendingErrorCount(), "should have 1 pending error")

	err = handler.Handle(context.Background(), makeErrorRecord("error 2"))
	require.NoError(t, err)

	assert.Equal(t, 2, handler.GetPendingErrorCount(), "should have 2 pending errors")

	err = handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	assert.Equal(t, 2, handler.GetPendingErrorCount(), "duplicate should not increase count")

	clk.Advance(10 * time.Second)

	assert.Equal(t, 0, handler.GetPendingErrorCount(), "should have 0 pending errors after flush")
}

func TestNotificationHandler_HasPendingBatch(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	assert.False(t, handler.HasPendingBatch(), "should not have pending batch initially")

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	assert.True(t, handler.HasPendingBatch(), "should have pending batch after logging")

	clk.Advance(10 * time.Second)

	assert.False(t, handler.HasPendingBatch(), "should not have pending batch after flush")
}

func TestNotificationHandler_GetPendingErrors(t *testing.T) {
	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	clk := clock.NewMockClock(startTime)
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	clk.Advance(2 * time.Second)

	err = handler.Handle(context.Background(), makeErrorRecord("error 2"))
	require.NoError(t, err)

	pending := handler.GetPendingErrors()

	assert.Len(t, pending, 2, "should have 2 pending errors")

	messages := make([]string, 0, 2)
	for _, grouped := range pending {
		messages = append(messages, grouped.LogRecord.Message)
		assert.Equal(t, 1, grouped.Count, "each error should have count 1")

		assert.True(t, grouped.FirstSeen.Sub(startTime) >= 0 && grouped.FirstSeen.Sub(startTime) <= 3*time.Second,
			"FirstSeen should be set within the test time range")
	}

	assert.Contains(t, messages, "error 1")
	assert.Contains(t, messages, "error 2")
}

func TestNotificationHandler_GetPendingErrors_IsCopy(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	pending := handler.GetPendingErrors()
	for k := range pending {
		delete(pending, k)
	}

	assert.Equal(t, 1, handler.GetPendingErrorCount(), "handler state should not be affected by modifying returned map")
}

func TestNotificationHandler_GetDebounceDuration(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	assert.Equal(t, 10*time.Second, handler.GetDebounceDuration(), "default debounce should be 10 seconds")

	handler.SetDebounceDuration(5 * time.Second)

	assert.Equal(t, 5*time.Second, handler.GetDebounceDuration(), "should return custom debounce duration")
}

func TestNotificationHandler_GetMinLevel(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()

	handlerError := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)
	assert.Equal(t, slog.LevelError, handlerError.GetMinLevel(), "should return Error level")

	handlerWarn := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelWarn, clk, lifecycle)
	assert.Equal(t, slog.LevelWarn, handlerWarn.GetMinLevel(), "should return Warn level")
}

func TestNotificationHandler_ObservableState_DuringBatching(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("error 1"))
	require.NoError(t, err)

	assert.True(t, handler.HasPendingBatch(), "should have pending batch")
	assert.Equal(t, 1, handler.GetPendingErrorCount(), "should have 1 error")

	clk.Advance(3 * time.Second)

	err = handler.Handle(context.Background(), makeErrorRecord("error 2"))
	require.NoError(t, err)

	assert.True(t, handler.HasPendingBatch(), "should still have pending batch")
	assert.Equal(t, 2, handler.GetPendingErrorCount(), "should have 2 errors")

	clk.Advance(7 * time.Second)

	assert.False(t, handler.HasPendingBatch(), "should not have pending batch after flush")
	assert.Equal(t, 0, handler.GetPendingErrorCount(), "should have 0 errors after flush")

	require.Len(t, transport.batches, 1, "one batch should be sent")
	assert.Len(t, transport.batches[0], 2, "batch should contain 2 errors")
}

func TestNotificationHandler_ObservableState_Deduplication(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	for range 3 {
		err := handler.Handle(context.Background(), makeErrorRecord("duplicate error"))
		require.NoError(t, err)
		clk.Advance(1 * time.Second)
	}

	assert.Equal(t, 1, handler.GetPendingErrorCount(), "should have 1 unique error")

	pending := handler.GetPendingErrors()
	require.Len(t, pending, 1, "should have 1 pending error")

	for _, grouped := range pending {
		assert.Equal(t, 3, grouped.Count, "error should have count of 3")
		assert.Equal(t, "duplicate error", grouped.LogRecord.Message)
	}
}

func TestNotificationHandler_ObservableState_ThreadSafe(t *testing.T) {
	clk := clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	transport := &recordingTransport{}
	baseHandler := slog.NewJSONHandler(io.Discard, nil)

	lifecycle := logger_domain.NewLifecycleManager()
	handler := logger_domain.NewNotificationHandlerWithOptions(baseHandler, transport, slog.LevelError, clk, lifecycle)

	err := handler.Handle(context.Background(), makeErrorRecord("error"))
	require.NoError(t, err)

	done := make(chan bool, 5)

	go func() {
		handler.GetPendingErrorCount()
		done <- true
	}()

	go func() {
		handler.HasPendingBatch()
		done <- true
	}()

	go func() {
		handler.GetPendingErrors()
		done <- true
	}()

	go func() {
		handler.GetDebounceDuration()
		done <- true
	}()

	go func() {
		handler.GetMinLevel()
		done <- true
	}()

	for range 5 {
		<-done
	}

	assert.True(t, true, "all getters should be thread-safe")
}
