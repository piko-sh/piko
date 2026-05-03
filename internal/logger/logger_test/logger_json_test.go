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
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
)

func setupJSONTestLogger(t *testing.T, loggerConfig logger_dto.Config) (logger_domain.Logger, func() string) {
	t.Helper()

	buffer := new(bytes.Buffer)
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	var wg sync.WaitGroup
	wg.Go(func() {
		_, _ = io.Copy(buffer, r)
	})

	slogLogger, shutdown, err := Initialise(context.Background(), loggerConfig)
	require.NoError(t, err)

	InitDefaultFactory(slogLogger)
	log := logger_domain.GetLogger("json-test-" + t.Name())

	getOutput := func() string {
		_ = shutdown(context.Background())
		_ = w.Close()
		wg.Wait()
		os.Stderr = originalStderr
		return buffer.String()
	}

	return log, getOutput
}

func unmarshalLogLine(t *testing.T, line string) map[string]any {
	t.Helper()
	var data map[string]any
	err := json.Unmarshal([]byte(line), &data)
	require.NoError(t, err, "The logged output line must be valid JSON: %s", line)
	return data
}

func TestJSONOutput_FormatAndFields(t *testing.T) {
	loggerConfig := loadConfigFromYAML(t, "testdata/json_output.yaml")
	log, getOutput := setupJSONTestLogger(t, loggerConfig)

	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	testTime := time.Date(2025, 8, 22, 16, 29, 58, 0, time.UTC)
	testDuration := 15 * time.Second
	testErr := errors.New("this is a test error")

	log.Info("json serialisation test",
		logger_domain.String("string_key", "hello world"),
		logger_domain.Int("int_key", 12345),
		logger_domain.Float64("float_key", 3.14159),
		logger_domain.Bool("bool_key", true),
		logger_domain.Time("time_key", testTime),
		logger_domain.Duration("duration_key", testDuration),
		logger_domain.Error(testErr),
		logger_domain.Field("struct_key", testStruct{Name: "Test", Value: 99}),
	)

	output := getOutput()
	data := unmarshalLogLine(t, output)

	assert.Equal(t, "INFO", data["level"])
	assert.Equal(t, "json serialisation test", data["msg"])
	source, ok := data["source"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, source["file"], "logger_json_test.go")

	assert.Equal(t, "hello world", data["string_key"])
	assert.Equal(t, float64(12345), data["int_key"])
	assert.Equal(t, 3.14159, data["float_key"])
	assert.Equal(t, true, data["bool_key"])
	assert.Equal(t, testTime.Format(time.RFC3339Nano), data["time_key"])
	assert.Equal(t, float64(testDuration.Nanoseconds()), data["duration_key"])
	assert.Equal(t, testErr.Error(), data["error"])

	structData, ok := data["struct_key"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Test", structData["name"])
	assert.Equal(t, float64(99), structData["value"])
}

func TestJSONOutput_WithAddSourceDisabled(t *testing.T) {
	loggerConfig := logger_dto.Config{
		AddSource: false,
		Outputs: []logger_dto.OutputConfig{{
			Type: "stderr", Level: "debug", Format: "json",
		}},
	}
	log, getOutput := setupJSONTestLogger(t, loggerConfig)

	log.Info("this log should not have a source key")

	output := getOutput()
	data := unmarshalLogLine(t, output)

	assert.Equal(t, "this log should not have a source key", data["msg"])
	_, exists := data["source"]
	assert.False(t, exists, "The 'source' key should be absent when addSource is false")
}

func TestJSONOutput_LevelFiltering(t *testing.T) {
	loggerConfig := logger_dto.Config{
		Outputs: []logger_dto.OutputConfig{{
			Type: "stderr", Level: "warn", Format: "json",
		}},
	}
	log, getOutput := setupJSONTestLogger(t, loggerConfig)

	log.Debug("This should be filtered")
	log.Info("This should also be filtered")
	log.Warn("This is a warning", logger_domain.Int("code", 1))
	log.Error("This is an error", logger_domain.Int("code", 2))

	output := getOutput()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 2, "Expected exactly 2 log lines (WARN and ERROR)")

	warnData := unmarshalLogLine(t, lines[0])
	assert.Equal(t, "WARN", warnData["level"])
	assert.Equal(t, "This is a warning", warnData["msg"])

	errorData := unmarshalLogLine(t, lines[1])
	assert.Equal(t, "ERROR", errorData["level"])
	assert.Equal(t, "This is an error", errorData["msg"])
}

func TestJSONOutput_WithAttrs(t *testing.T) {
	loggerConfig := loadConfigFromYAML(t, "testdata/json_output.yaml")
	log, getOutput := setupJSONTestLogger(t, loggerConfig)

	contextualLog := log.With(
		logger_domain.String("service", "api-gateway"),
		logger_domain.String("version", "v2.1.0"),
	)

	contextualLog.Info("request processed", logger_domain.String("path", "/users"), logger_domain.Int("status", 200))

	output := getOutput()
	data := unmarshalLogLine(t, output)

	assert.Equal(t, "api-gateway", data["service"])
	assert.Equal(t, "v2.1.0", data["version"])
	assert.Equal(t, "/users", data["path"])
	assert.Equal(t, float64(200), data["status"])
}

func TestJSONOutput_WithGroupedAttrs(t *testing.T) {
	loggerConfig := loadConfigFromYAML(t, "testdata/json_output.yaml")
	log, getOutput := setupJSONTestLogger(t, loggerConfig)

	log.Info("user action",
		slog.Group("user",
			slog.String("name", "alice"),
			slog.Int("id", 123),
		),
		slog.Group("request",
			slog.String("id", "xyz-789"),
		),
	)

	output := getOutput()
	data := unmarshalLogLine(t, output)

	userData, ok := data["user"].(map[string]any)
	require.True(t, ok, "'user' group should be a JSON object")
	assert.Equal(t, "alice", userData["name"])
	assert.Equal(t, float64(123), userData["id"])

	requestData, ok := data["request"].(map[string]any)
	require.True(t, ok, "'request' group should be a JSON object")
	assert.Equal(t, "xyz-789", requestData["id"])
}

func TestJSONOutput_EmptyAndNilValues(t *testing.T) {
	loggerConfig := loadConfigFromYAML(t, "testdata/json_output.yaml")
	log, getOutput := setupJSONTestLogger(t, loggerConfig)

	log.Info("",
		logger_domain.String("empty_string", ""),
		logger_domain.Error(nil),
		logger_domain.Field("nil_field", nil),
	)

	output := getOutput()
	data := unmarshalLogLine(t, output)

	assert.Equal(t, "", data["msg"], "message should be an empty string")
	assert.Equal(t, "", data["empty_string"])

	assert.Equal(t, "<nil>", data["error"])

	assert.Nil(t, data["nil_field"])
}

func TestJSONOutput_ConcurrencySafety(t *testing.T) {
	loggerConfig := loadConfigFromYAML(t, "testdata/json_output.yaml")
	log, getOutput := setupJSONTestLogger(t, loggerConfig)

	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			log.Info("log from goroutine", logger_domain.Int("goroutine_id", id))
		}(i)
	}

	wg.Wait()

	output := getOutput()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, numGoroutines, "Should have exactly one log line per goroutine")

	seenIDs := make(map[int]bool)
	for _, line := range lines {
		data := unmarshalLogLine(t, line)
		assert.Equal(t, "log from goroutine", data["msg"])
		id, ok := data["goroutine_id"].(float64)
		require.True(t, ok)
		seenIDs[int(id)] = true
	}

	assert.Len(t, seenIDs, numGoroutines, "All goroutine IDs should be present in the output")
}
