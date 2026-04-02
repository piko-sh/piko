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

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
)

func setupCompositionTest(t *testing.T) (consoleBuffer *bytes.Buffer, tempLogPath string, cleanup func()) {
	t.Helper()

	ResetAndApplyConfig(logger_dto.Config{})

	buffer := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var wg sync.WaitGroup
	wg.Go(func() {
		_, _ = io.Copy(buffer, r)
	})

	tempDir := t.TempDir()
	tempFilePath := filepath.Join(tempDir, "composition-test.log")

	cleanupFunc := func() {
		if shutdowner := GetShutdownFunc(); shutdowner != nil {
			_ = shutdowner(context.Background())
		}

		_ = w.Close()
		wg.Wait()

		os.Stdout = originalStdout
		_ = r.Close()
	}

	return buffer, tempFilePath, cleanupFunc
}

func TestSequentialAddOutput_PreservesStructure(t *testing.T) {

	consoleBuffer, tempLogPath, cleanup := setupCompositionTest(t)

	AddPrettyOutput(slog.LevelInfo)
	debugLogPath := strings.Replace(tempLogPath, "composition-test", "debug-log", 1)
	AddFileOutput(context.Background(), "debug-log", debugLogPath, slog.LevelDebug, true)
	errorLogPath := strings.Replace(tempLogPath, "composition-test", "error-log", 1)
	AddFileOutput(context.Background(), "error-log", errorLogPath, slog.LevelError, true)

	log := logger_domain.GetLogger("composition-test")

	log.Error("database connection failed",
		logger_domain.Field(logger_domain.KeyContext, "user-service"),
		logger_domain.Field(logger_domain.KeyMethod, "POST:/api/users"),
		logger_domain.String("db_host", "prod-db-1.us-east-1"),
		logger_domain.Int("attempt", 3),
	)

	cleanup()

	rawPrettyOutput := consoleBuffer.String()
	t.Logf("Captured Raw Pretty Output:\n%q", rawPrettyOutput)
	prettyOutput := stripAnsi(rawPrettyOutput)
	t.Logf("Captured Stripped Pretty Output:\n%s", prettyOutput)

	lines := strings.Split(strings.TrimSpace(prettyOutput), "\n")
	require.GreaterOrEqual(t, len(lines), 1, "Should be at least one line of console output")

	var errorLogLine string
	for _, line := range lines {
		if strings.Contains(line, "database connection failed") {
			errorLogLine = line
			break
		}
	}
	require.NotEmpty(t, errorLogLine, "Should find a line containing 'database connection failed'")

	assert.Contains(t, errorLogLine, "database connection failed")
	assert.Contains(t, errorLogLine, "| user-service                 |")
	assert.Contains(t, errorLogLine, "| POST:/api/users                         |")
	assert.Contains(t, errorLogLine, "db_host=prod-db-1.us-east-1")
	assert.Contains(t, errorLogLine, "attempt=3")

	debugJSONBytes, err := os.ReadFile(debugLogPath)
	require.NoError(t, err, "Debug log file should be readable")
	t.Logf("Captured Debug JSON Output:\n%s", string(debugJSONBytes))

	debugJsonLines := strings.Split(strings.TrimSpace(string(debugJSONBytes)), "\n")
	require.GreaterOrEqual(t, len(debugJsonLines), 1, "Should be at least one line in debug log")
	lastDebugJson := debugJsonLines[len(debugJsonLines)-1]

	var debugData map[string]any
	require.NoError(t, json.Unmarshal([]byte(lastDebugJson), &debugData), "Last line of debug log must be valid JSON")
	assert.Equal(t, "database connection failed", debugData["msg"])
	assert.Equal(t, "user-service", debugData["ctx"])
	assert.Equal(t, "POST:/api/users", debugData["mtd"])
	assert.Equal(t, "prod-db-1.us-east-1", debugData["db_host"])
	assert.Equal(t, float64(3), debugData["attempt"])

	errorJSONBytes, err := os.ReadFile(errorLogPath)
	require.NoError(t, err, "Error log file should be readable")
	t.Logf("Captured Error JSON Output:\n%s", string(errorJSONBytes))

	var errorData map[string]any
	require.NoError(t, json.Unmarshal(errorJSONBytes, &errorData), "Error log must be valid JSON")
	assert.Equal(t, "database connection failed", errorData["msg"])
	assert.Equal(t, "user-service", errorData["ctx"])
	assert.Equal(t, "POST:/api/users", errorData["mtd"])
	assert.Equal(t, "prod-db-1.us-east-1", errorData["db_host"])
	assert.Equal(t, float64(3), errorData["attempt"])
}
