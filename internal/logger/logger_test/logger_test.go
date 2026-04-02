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

package logger_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
)

func loadConfigFromYAML(t *testing.T, path string) logger_dto.Config {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var loggerConfig logger_dto.Config
	err = yaml.Unmarshal(data, &loggerConfig)
	require.NoError(t, err)
	return loggerConfig
}

type mockErrorTransport struct {
	t           *testing.T
	lastPayload []byte
	sendCalled  int
	mu          sync.Mutex
}

func (m *mockErrorTransport) SendGroupedErrors(ctx context.Context, batch map[string]*logger_domain.GroupedError) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalled++

	summary := make(map[string]int)
	for _, errInfo := range batch {
		summary[errInfo.LogRecord.Message] += errInfo.Count
	}
	payload, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	m.lastPayload = payload
	return nil
}

func (m *mockErrorTransport) getSendCalled() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendCalled
}

func (m *mockErrorTransport) getLastPayload() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]byte(nil), m.lastPayload...)
}

type capturingHandler struct {
	slog.Handler
	record slog.Record
	mu     sync.Mutex
}

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	h.record = r
	h.mu.Unlock()
	return nil
}

func (h *capturingHandler) GetRecord() slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.record
}

func TestFullConfiguration_InitialisationAndOutput(t *testing.T) {
	loggerConfig := loadConfigFromYAML(t, "testdata/full_config.yaml")

	consoleBuffer := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var wg sync.WaitGroup
	wg.Go(func() {
		_, _ = io.Copy(consoleBuffer, r)
	})
	defer func() {
		os.Stdout = originalStdout
		_ = r.Close()
	}()

	tempDir := t.TempDir()
	logFilePath := filepath.Join(tempDir, "app.log")
	loggerConfig.Outputs[1].File.Path = logFilePath

	slogLogger, shutdown, err := Initialise(context.Background(), loggerConfig, &config.ServerConfig{})
	require.NoError(t, err)

	InitDefaultFactory(slogLogger)
	log := logger_domain.GetLogger("test-integration")

	log.Debug("This is a debug message", logger_domain.String("user", "tester"))
	log.Info("This is an info message", logger_domain.Int("code", 200))
	log.Warn("This is a warning message", logger_domain.Error(io.EOF))
	log.Error("This is an error message", logger_domain.String("critical", "true"))

	err = shutdown(context.Background())
	assert.NoError(t, err)
	_ = w.Close()
	wg.Wait()

	consoleOutput := consoleBuffer.String()
	t.Logf("Captured Console Output:\n%s", consoleOutput)
	assert.Contains(t, consoleOutput, "This is a debug message")

	fileContent, err := os.ReadFile(logFilePath)
	require.NoError(t, err)
	fileOutput := string(fileContent)
	t.Logf("Captured File Output:\n%s", fileOutput)
	assert.NotContains(t, fileOutput, `"level":"DEBUG"`)

	lines := strings.Split(strings.TrimSpace(fileOutput), "\n")
	require.Len(t, lines, 3)
	for _, line := range lines {
		var entry map[string]any
		err := json.Unmarshal([]byte(line), &entry)
		assert.NoError(t, err)
	}
}

func TestOtelSlogHandler_TraceContextInjection(t *testing.T) {
	tp := &testRecordingTracerProvider{}
	tracer := tp.Tracer("test-tracer")

	discardHandler := slog.NewTextHandler(io.Discard, nil)
	capture := &capturingHandler{Handler: discardHandler}
	otelHandler := logger_domain.NewOTelSlogHandler(capture)
	slogLogger := slog.New(otelHandler)

	ctx, span := tracer.Start(context.Background(), "test-span")
	slogLogger.InfoContext(ctx, "hello from within a span", slog.String("attr1", "value1"))
	span.End()

	record := capture.GetRecord()
	foundTraceID, foundSpanID := false, false
	record.Attrs(func(a slog.Attr) bool {
		if a.Key == "trace_id" {
			foundTraceID = true
			assert.Equal(t, span.SpanContext().TraceID().String(), a.Value.String())
		}
		if a.Key == "span_id" {
			foundSpanID = true
			assert.Equal(t, span.SpanContext().SpanID().String(), a.Value.String())
		}
		return true
	})
	assert.True(t, foundTraceID, "trace_id should be added to the log record")
	assert.True(t, foundSpanID, "span_id should be added to the log record")

	spans := tp.getSpans()
	require.Len(t, spans, 1)
	testSpan := spans[0]
	require.Len(t, testSpan.Events, 1)
	event := testSpan.Events[0]
	assert.Equal(t, "hello from within a span", event.Name)
	assert.Equal(t, "value1", event.Attributes[0].Value.AsString())
}

func TestNotificationHandler(t *testing.T) {

	logWarnFromSameLocation := func(l *slog.Logger, message string, arguments ...any) {
		l.Warn(message, arguments...)
	}

	t.Run("GroupsAndDebouncesMessages", func(t *testing.T) {
		mockTransport := &mockErrorTransport{t: t}
		discardHandler := slog.NewJSONHandler(io.Discard, nil)
		notificationHandler := logger_domain.NewNotificationHandler(discardHandler, mockTransport, slog.LevelWarn)
		notificationHandler.SetDebounceDuration(20 * time.Millisecond)
		slogLogger := slog.New(notificationHandler)

		slogLogger.Info("This should be ignored by the notifier")
		logWarnFromSameLocation(slogLogger, "First warning", slog.String("id", "A"))
		logWarnFromSameLocation(slogLogger, "First warning", slog.String("id", "B"))
		slogLogger.Error("A critical error", slog.String("id", "C"))
		slogLogger.Warn("A different warning", slog.String("id", "D"))

		require.Eventually(t, func() bool {
			return mockTransport.getSendCalled() >= 1
		}, 2*time.Second, 5*time.Millisecond, "debounce flush should have fired")

		assert.Equal(t, 1, mockTransport.getSendCalled(), "Send should have been called exactly once")

		var payload map[string]int
		err := json.Unmarshal(mockTransport.getLastPayload(), &payload)
		require.NoError(t, err)

		require.Len(t, payload, 3, "Payload should contain 3 unique error messages")
		assert.Equal(t, 2, payload["First warning"], "Should have grouped two 'First warning' messages")
		assert.Equal(t, 1, payload["A critical error"])
		assert.Equal(t, 1, payload["A different warning"])
	})

	t.Run("FlushesPendingMessagesOnShutdown", func(t *testing.T) {
		mockTransport := &mockErrorTransport{t: t}
		discardHandler := slog.NewJSONHandler(io.Discard, nil)
		notificationHandler := logger_domain.NewNotificationHandler(discardHandler, mockTransport, slog.LevelWarn)

		notificationHandler.SetDebounceDuration(1 * time.Minute)
		slogLogger := slog.New(notificationHandler)

		slogLogger.Error("An error that should be flushed by shutdown")

		notificationHandler.Shutdown()

		assert.Equal(t, 1, mockTransport.getSendCalled(), "Send should have been called by Shutdown()")
		var payload map[string]int
		err := json.Unmarshal(mockTransport.getLastPayload(), &payload)
		require.NoError(t, err)

		require.Len(t, payload, 1)
		assert.Equal(t, 1, payload["An error that should be flushed by shutdown"])
	})
}
