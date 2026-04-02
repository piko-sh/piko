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
	"log/slog"
	"os"
	"strings"
	"testing"

	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/logger/logger_state"
)

func TestLoggerConfigFromEnvironment(t *testing.T) {
	tests := []struct {
		name           string
		envLogLevel    string
		shouldSeeDebug bool
		shouldSeeInfo  bool
	}{
		{
			name:           "LOG_LEVEL=debug shows debug messages",
			envLogLevel:    "debug",
			shouldSeeDebug: true,
			shouldSeeInfo:  true,
		},
		{
			name:           "LOG_LEVEL=info hides debug messages",
			envLogLevel:    "info",
			shouldSeeDebug: false,
			shouldSeeInfo:  true,
		},
		{
			name:           "LOG_LEVEL=warn hides debug and info messages",
			envLogLevel:    "warn",
			shouldSeeDebug: false,
			shouldSeeInfo:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			logger_state.ResetState()

			var buffer bytes.Buffer

			logConfig := logger_dto.Config{
				Level: tt.envLogLevel,
				Outputs: []logger_dto.OutputConfig{
					{
						Type:   "stdout",
						Format: "json",
					},
				},
			}

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ctx := context.Background()
			_, shutdown, err := logger.Initialise(ctx, logConfig, driver_handlers.OtelSetupConfig{}, nil)
			if err != nil {
				t.Fatalf("Failed to initialise logger: %v", err)
			}

			log := logger_domain.GetLogger("test")

			log.Debug("debug message")
			log.Info("info message")
			log.Warn("warn message")

			if shutdown != nil {
				_ = shutdown(ctx)
			}
			_ = w.Close()
			os.Stdout = oldStdout

			_, _ = buffer.ReadFrom(r)
			output := buffer.String()

			hasDebug := strings.Contains(output, "debug message")
			if hasDebug != tt.shouldSeeDebug {
				t.Errorf("Debug message visibility incorrect. Expected: %v, Got: %v\nOutput:\n%s",
					tt.shouldSeeDebug, hasDebug, output)
			}

			hasInfo := strings.Contains(output, "info message")
			if hasInfo != tt.shouldSeeInfo {
				t.Errorf("Info message visibility incorrect. Expected: %v, Got: %v\nOutput:\n%s",
					tt.shouldSeeInfo, hasInfo, output)
			}

			hasWarn := strings.Contains(output, "warn message")
			if !hasWarn {
				t.Errorf("Warn message should always be visible\nOutput:\n%s", output)
			}
		})
	}
}

func TestLoggerConfigParsesLevels(t *testing.T) {
	tests := []struct {
		levelString string
		expected    slog.Level
	}{
		{levelString: "trace", expected: slog.Level(-8)},
		{levelString: "debug", expected: slog.LevelDebug},
		{levelString: "info", expected: slog.LevelInfo},
		{levelString: "notice", expected: slog.Level(2)},
		{levelString: "warn", expected: slog.LevelWarn},
		{levelString: "error", expected: slog.LevelError},
		{levelString: "invalid", expected: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.levelString, func(t *testing.T) {
			result := logger_dto.ParseLogLevel(tt.levelString, slog.LevelInfo)
			if result != tt.expected {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.levelString, result, tt.expected)
			}
		})
	}
}
