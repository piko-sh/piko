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

package logger_dto

import (
	"log/slog"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{input: "trace", want: slog.Level(-8)},
		{input: "TRACE", want: slog.Level(-8)},
		{input: "internal", want: slog.Level(-6)},
		{input: "debug", want: slog.LevelDebug},
		{input: "info", want: slog.LevelInfo},
		{input: "notice", want: slog.Level(2)},
		{input: "warn", want: slog.LevelWarn},
		{input: "warning", want: slog.LevelWarn},
		{input: "error", want: slog.LevelError},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseLogLevel(tt.input, slog.LevelInfo)
			if got != tt.want {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLogLevel_Default(t *testing.T) {
	got := ParseLogLevel("unknown", slog.LevelWarn)
	if got != slog.LevelWarn {
		t.Errorf("ParseLogLevel(unknown) = %v, want %v", got, slog.LevelWarn)
	}
}

func TestParseSlogLevels(t *testing.T) {
	got := parseSlogLevels("debug,info,warn")
	if len(got) != 3 {
		t.Fatalf("parseSlogLevels returned %d levels, want 3", len(got))
	}
	if got[0] != slog.LevelDebug {
		t.Errorf("got[0] = %v, want Debug", got[0])
	}
	if got[1] != slog.LevelInfo {
		t.Errorf("got[1] = %v, want Info", got[1])
	}
	if got[2] != slog.LevelWarn {
		t.Errorf("got[2] = %v, want Warn", got[2])
	}
}

func TestParseSlogLevels_Empty(t *testing.T) {
	if got := parseSlogLevels(""); got != nil {
		t.Errorf("parseSlogLevels(\"\") = %v, want nil", got)
	}
}

func TestParseSlogLevels_IgnoresUnknown(t *testing.T) {
	got := parseSlogLevels("debug,unknown,error")
	if len(got) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(got))
	}
}
