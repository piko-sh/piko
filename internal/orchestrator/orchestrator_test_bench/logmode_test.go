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

package orchestrator_test_bench

import (
	"io"
	"log/slog"
	"testing"

	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	logModes = []string{"off", "info", "trace"}
)

func applyLogMode(tb testing.TB, mode string) {
	tb.Helper()

	var level slog.Level
	switch mode {
	case "off":
		level = slog.LevelError + 100
	case "info":
		level = slog.LevelInfo
	case "trace":
		level = logger_domain.LevelTrace
	default:
		tb.Fatalf("unknown log mode %q", mode)
	}

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: level})))
	tb.Cleanup(func() { slog.SetDefault(previous) })
}
