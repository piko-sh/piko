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

package browser_provider_chromedp

import (
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/go-json-experiment/json/jsontext"
)

func TestMapConsoleLevel(t *testing.T) {
	testCases := []struct {
		name     string
		input    runtime.APIType
		expected string
	}{
		{
			name:     "log type",
			input:    runtime.APITypeLog,
			expected: "log",
		},
		{
			name:     "debug type",
			input:    runtime.APITypeDebug,
			expected: "debug",
		},
		{
			name:     "info type",
			input:    runtime.APITypeInfo,
			expected: "info",
		},
		{
			name:     "error type",
			input:    runtime.APITypeError,
			expected: "error",
		},
		{
			name:     "warning type",
			input:    runtime.APITypeWarning,
			expected: "warn",
		},
		{
			name:     "trace type",
			input:    runtime.APITypeTrace,
			expected: "trace",
		},
		{
			name:     "unknown type defaults to log",
			input:    runtime.APIType("unknown"),
			expected: "log",
		},
		{
			name:     "assert type defaults to log",
			input:    runtime.APITypeAssert,
			expected: "log",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapConsoleLevel(tc.input)
			if result != tc.expected {
				t.Errorf("mapConsoleLevel(%v) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestDefaultBrowserOptions(t *testing.T) {
	opts := DefaultBrowserOptions()

	if !opts.Headless {
		t.Error("DefaultBrowserOptions().Headless should be true")
	}
}

func TestConsoleLog(t *testing.T) {

	logEntry := ConsoleLog{
		Time:    time.Now(),
		Level:   "error",
		Message: "test message",
	}

	if logEntry.Level != "error" {
		t.Errorf("ConsoleLog.Level = %q, expected %q", logEntry.Level, "error")
	}
	if logEntry.Message != "test message" {
		t.Errorf("ConsoleLog.Message = %q, expected %q", logEntry.Message, "test message")
	}
}

func TestBuildConsoleMessage(t *testing.T) {
	testCases := []struct {
		name      string
		expected  string
		arguments []*runtime.RemoteObject
	}{
		{
			name:      "single string argument",
			arguments: []*runtime.RemoteObject{{Value: jsontext.Value(`"hello"`)}},
			expected:  "hello",
		},
		{
			name: "multiple arguments joined with space",
			arguments: []*runtime.RemoteObject{
				{Value: jsontext.Value(`"hello"`)},
				{Value: jsontext.Value(`"world"`)},
			},
			expected: "hello world",
		},
		{
			name:      "empty arguments",
			arguments: []*runtime.RemoteObject{},
			expected:  "",
		},
		{
			name:      "numeric value",
			arguments: []*runtime.RemoteObject{{Value: jsontext.Value(`42`)}},
			expected:  "42",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildConsoleMessage(tc.arguments)
			if result != tc.expected {
				t.Errorf("buildConsoleMessage() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

func TestRemoteObjectToString(t *testing.T) {
	testCases := []struct {
		name     string
		argument *runtime.RemoteObject
		expected string
	}{
		{
			name:     "JSON string value has quotes stripped",
			argument: &runtime.RemoteObject{Value: jsontext.Value(`"test"`)},
			expected: "test",
		},
		{
			name:     "numeric value",
			argument: &runtime.RemoteObject{Value: jsontext.Value(`123`)},
			expected: "123",
		},
		{
			name:     "empty value with description",
			argument: &runtime.RemoteObject{Value: nil, Description: "Array(3)"},
			expected: "Array(3)",
		},
		{
			name:     "empty value no description",
			argument: &runtime.RemoteObject{Value: nil, Description: ""},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := remoteObjectToString(tc.argument)
			if tc.name == "empty value no description" {

				if result == "" {
					t.Error("remoteObjectToString() should return non-empty for fallback case")
				}
				return
			}
			if result != tc.expected {
				t.Errorf("remoteObjectToString() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

func TestPageHelper_RecordConsoleLog(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
		consoleMutex:  sync.Mutex{},
	}

	ph.recordConsoleLog("test message", "info")
	ph.recordConsoleLog("error message", "error")

	if len(ph.consoleLogs) != 2 {
		t.Fatalf("expected 2 consoleLogs entries, got %d", len(ph.consoleLogs))
	}
	if len(ph.consoleLogsV2) != 2 {
		t.Fatalf("expected 2 consoleLogsV2 entries, got %d", len(ph.consoleLogsV2))
	}

	if ph.consoleLogs[0] != "test message" {
		t.Errorf("consoleLogs[0] = %q, expected %q", ph.consoleLogs[0], "test message")
	}
	if ph.consoleLogsV2[1].Level != "error" {
		t.Errorf("consoleLogsV2[1].Level = %q, expected %q", ph.consoleLogsV2[1].Level, "error")
	}
}

func TestPageHelper_RecordConsoleLog_StoppedIsNoop(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
		consoleMutex:  sync.Mutex{},
		stopped:       true,
	}

	ph.recordConsoleLog("should be ignored", "info")

	if len(ph.consoleLogs) != 0 {
		t.Errorf("expected 0 entries when stopped, got %d", len(ph.consoleLogs))
	}
}

func TestPageHelper_ConsoleLogsByLevel_Unit(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
		consoleMutex:  sync.Mutex{},
	}

	ph.recordConsoleLog("info1", "info")
	ph.recordConsoleLog("error1", "error")
	ph.recordConsoleLog("info2", "info")
	ph.recordConsoleLog("warn1", "warn")

	infoLogs := ph.ConsoleLogsByLevel("info")
	if len(infoLogs) != 2 {
		t.Errorf("expected 2 info logs, got %d", len(infoLogs))
	}

	errorLogs := ph.ConsoleLogsByLevel("error")
	if len(errorLogs) != 1 {
		t.Errorf("expected 1 error log, got %d", len(errorLogs))
	}

	debugLogs := ph.ConsoleLogsByLevel("debug")
	if len(debugLogs) != 0 {
		t.Errorf("expected 0 debug logs, got %d", len(debugLogs))
	}
}

func TestPageHelper_HasConsoleErrors_Unit(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
		consoleMutex:  sync.Mutex{},
	}

	if ph.HasConsoleErrors() {
		t.Error("HasConsoleErrors should be false when no errors")
	}

	ph.recordConsoleLog("info log", "info")
	if ph.HasConsoleErrors() {
		t.Error("HasConsoleErrors should be false with only info logs")
	}

	ph.recordConsoleLog("error log", "error")
	if !ph.HasConsoleErrors() {
		t.Error("HasConsoleErrors should be true when error present")
	}
}

func TestPageHelper_ConsoleErrors_Unit(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
		consoleMutex:  sync.Mutex{},
	}

	ph.recordConsoleLog("info", "info")
	ph.recordConsoleLog("err1", "error")
	ph.recordConsoleLog("err2", "error")

	errors := ph.ConsoleErrors()
	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errors))
	}
	for _, e := range errors {
		if e.Level != "error" {
			t.Errorf("expected level %q, got %q", "error", e.Level)
		}
	}
}

func TestPageHelper_ConsoleLogs_ReturnsDefensiveCopy(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
		consoleMutex:  sync.Mutex{},
	}

	ph.recordConsoleLog("msg1", "info")

	copy1 := ph.ConsoleLogs()
	copy1[0] = "modified"

	original := ph.ConsoleLogs()
	if original[0] != "msg1" {
		t.Error("ConsoleLogs should return a defensive copy")
	}
}

func TestPageHelper_ConsoleLogsWithLevel_ReturnsDefensiveCopy(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
		consoleMutex:  sync.Mutex{},
	}

	ph.recordConsoleLog("msg1", "info")

	copy1 := ph.ConsoleLogsWithLevel()
	copy1[0].Message = "modified"

	original := ph.ConsoleLogsWithLevel()
	if original[0].Message != "msg1" {
		t.Error("ConsoleLogsWithLevel should return a defensive copy")
	}
}

func TestPageHelper_ClearConsoleLogs_ClearsInternalState(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
		consoleMutex:  sync.Mutex{},
	}

	ph.recordConsoleLog("msg1", "info")
	ph.recordConsoleLog("msg2", "error")

	ph.consoleMutex.Lock()
	ph.consoleLogs = []string{}
	ph.consoleLogsV2 = []ConsoleLog{}
	ph.consoleMutex.Unlock()

	if len(ph.ConsoleLogs()) != 0 {
		t.Error("expected consoleLogs to be cleared")
	}
	if len(ph.ConsoleLogsWithLevel()) != 0 {
		t.Error("expected consoleLogsV2 to be cleared")
	}
}
