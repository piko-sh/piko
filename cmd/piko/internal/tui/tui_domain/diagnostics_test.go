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

package tui_domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestDiagnosticsResult_AllPassed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		passed    int
		failed    int
		connected bool
		want      bool
	}{
		{name: "connected with no failures", connected: true, passed: 3, failed: 0, want: true},
		{name: "connected with failures", connected: true, passed: 2, failed: 1, want: false},
		{name: "not connected", connected: false, passed: 0, failed: 0, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := &DiagnosticsResult{
				Connected: tc.connected,
				Passed:    tc.passed,
				Failed:    tc.failed,
			}
			if got := r.AllPassed(); got != tc.want {
				t.Errorf("AllPassed() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDiagnosticsResult_AddResult(t *testing.T) {
	t.Parallel()

	t.Run("success increments passed", func(t *testing.T) {
		t.Parallel()
		r := &DiagnosticsResult{}
		r.addResult("TestService", "GetFoo", nil, "3 items")

		if r.Passed != 1 {
			t.Errorf("Passed = %d, want 1", r.Passed)
		}
		if r.Failed != 0 {
			t.Errorf("Failed = %d, want 0", r.Failed)
		}
		if len(r.Services) != 1 {
			t.Fatalf("len(Services) = %d, want 1", len(r.Services))
		}

		sr := r.Services[0]
		if !sr.OK {
			t.Error("OK should be true for success")
		}
		if sr.Name != "TestService" {
			t.Errorf("Name = %q, want %q", sr.Name, "TestService")
		}
		if sr.Method != "GetFoo" {
			t.Errorf("Method = %q, want %q", sr.Method, "GetFoo")
		}
		if sr.Details != "3 items" {
			t.Errorf("Details = %q, want %q", sr.Details, "3 items")
		}
	})

	t.Run("error increments failed", func(t *testing.T) {
		t.Parallel()
		r := &DiagnosticsResult{}
		r.addResult("TestService", "GetBar", errors.New("connection refused"), "failed")

		if r.Passed != 0 {
			t.Errorf("Passed = %d, want 0", r.Passed)
		}
		if r.Failed != 1 {
			t.Errorf("Failed = %d, want 1", r.Failed)
		}

		sr := r.Services[0]
		if sr.OK {
			t.Error("OK should be false for error")
		}
		if sr.Error == nil {
			t.Error("Error should not be nil")
		}
	})

	t.Run("multiple results accumulate", func(t *testing.T) {
		t.Parallel()
		r := &DiagnosticsResult{}
		r.addResult("Service", "A", nil, "ok")
		r.addResult("Service", "B", errors.New("fail"), "")
		r.addResult("Service", "C", nil, "ok")

		if r.Passed != 2 {
			t.Errorf("Passed = %d, want 2", r.Passed)
		}
		if r.Failed != 1 {
			t.Errorf("Failed = %d, want 1", r.Failed)
		}
		if len(r.Services) != 3 {
			t.Errorf("len(Services) = %d, want 3", len(r.Services))
		}
	})
}

func TestDiagnosticsResult_Print_Connected(t *testing.T) {
	t.Parallel()

	r := &DiagnosticsResult{
		Endpoint:  "localhost:9091",
		Connected: true,
	}
	r.addResult(serviceHealth, "GetHealth", nil, "liveness=ok")
	r.addResult(serviceMetrics, "GetMetrics", nil, "5 metrics")

	var buffer bytes.Buffer
	r.Print(&buffer)
	output := buffer.String()

	mustContain := []string{
		"TUI Diagnostics",
		"localhost:9091",
		"Status: Connected",
		"HealthService",
		"GetHealth: OK",
		"MetricsService",
		"GetMetrics: OK",
		"2 passed, 0 failed",
		"All services operational",
	}

	for _, want := range mustContain {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\n\nFull output:\n%s", want, output)
		}
	}
}

func TestDiagnosticsResult_Print_ConnectionFailed(t *testing.T) {
	t.Parallel()

	r := &DiagnosticsResult{
		ConnectionError: errors.New("connection refused"),
		Endpoint:        "localhost:9091",
		Connected:       false,
	}

	var buffer bytes.Buffer
	r.Print(&buffer)
	output := buffer.String()

	mustContain := []string{
		"Status: FAILED",
		"connection refused",
		"piko.WithMonitoring()",
	}

	for _, want := range mustContain {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\n\nFull output:\n%s", want, output)
		}
	}
}

func TestDiagnosticsResult_Print_MixedResults(t *testing.T) {
	t.Parallel()

	r := &DiagnosticsResult{
		Endpoint:  "localhost:9091",
		Connected: true,
	}
	r.addResult(serviceHealth, "GetHealth", nil, "ok")
	r.addResult(serviceOrchestrator, "GetTaskSummary", errors.New("unimplemented"), "failed")

	var buffer bytes.Buffer
	r.Print(&buffer)
	output := buffer.String()

	mustContain := []string{
		"1 passed, 1 failed",
		"Some services failed",
		"GetHealth: OK",
		"GetTaskSummary: FAILED",
	}

	for _, want := range mustContain {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\n\nFull output:\n%s", want, output)
		}
	}
}
