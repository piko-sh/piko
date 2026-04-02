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

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"piko.sh/piko/cmd/piko/internal/tui"
)

func TestDiagnosticJSONResult(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		wantKeys []string
		result   diagnosticJSONResult
	}{
		{
			name: "connected with services",
			result: diagnosticJSONResult{
				Endpoint:  "localhost:9091",
				Connected: true,
				Passed:    2,
				Failed:    0,
				Services: []diagnosticServiceResult{
					{Name: "HealthService", Method: "GetHealth", OK: true, Details: "healthy"},
					{Name: "MetricsService", Method: "GetMetrics", OK: true},
				},
			},
			wantKeys: []string{
				`"endpoint":"localhost:9091"`,
				`"connected":true`,
				`"passed":2`,
				`"failed":0`,
				`"name":"HealthService"`,
				`"method":"GetHealth"`,
				`"ok":true`,
				`"details":"healthy"`,
				`"name":"MetricsService"`,
			},
		},
		{
			name: "connection error omits empty fields",
			result: diagnosticJSONResult{
				Endpoint:  "localhost:9091",
				Connected: false,
				Error:     "connection refused",
				Passed:    0,
				Failed:    1,
				Services: []diagnosticServiceResult{
					{Name: "HealthService", Method: "GetHealth", OK: false, Error: "unavailable"},
				},
			},
			wantKeys: []string{
				`"connected":false`,
				`"error":"connection refused"`,
				`"failed":1`,
				`"error":"unavailable"`,
			},
		},
		{
			name: "omitempty hides empty details and error",
			result: diagnosticJSONResult{
				Endpoint:  "localhost:9091",
				Connected: true,
				Services: []diagnosticServiceResult{
					{Name: "Service", Method: "M", OK: true},
				},
			},
			wantKeys: []string{`"name":"Service"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tc.result)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}

			output := string(data)
			for _, want := range tc.wantKeys {
				if !contains(output, want) {
					t.Errorf("JSON missing %q\nfull JSON:\n%s", want, output)
				}
			}
		})
	}
}

func TestDiagnosticJSONOmitEmpty(t *testing.T) {
	t.Parallel()

	result := diagnosticJSONResult{
		Endpoint:  "localhost:9091",
		Connected: true,
		Passed:    1,
		Services: []diagnosticServiceResult{
			{Name: "Service", Method: "M", OK: true},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	output := string(data)

	if contains(output, `"error"`) {
		t.Errorf("expected top-level error to be omitted, got:\n%s", output)
	}
}

func TestFormatDiagnosticResult(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		result  *tui.DiagnosticsResult
		name    string
		format  string
		wantAll []string
		wantErr bool
	}{
		{
			name: "json all passed",
			result: &tui.DiagnosticsResult{
				Endpoint:  "localhost:9091",
				Connected: true,
				Passed:    2,
				Services: []tui.ServiceResult{
					{Name: "HealthService", Method: "GetHealth", OK: true, Details: "healthy"},
					{Name: "MetricsService", Method: "GetMetrics", OK: true, Details: "2 metrics"},
				},
			},
			format: "json",
			wantAll: []string{
				`"endpoint"`,
				`"localhost:9091"`,
				`"connected": true`,
				`"passed": 2`,
				`"services"`,
				`"name": "HealthService"`,
				`"method": "GetHealth"`,
				`"ok": true`,
				`"details": "healthy"`,
			},
		},
		{
			name: "json with failures",
			result: &tui.DiagnosticsResult{
				Endpoint:  "localhost:9091",
				Connected: true,
				Passed:    1,
				Failed:    1,
				Services: []tui.ServiceResult{
					{Name: "HealthService", Method: "GetHealth", OK: true, Details: "healthy"},
					{Name: "MetricsService", Method: "GetMetrics", OK: false, Error: errors.New("unavailable")},
				},
			},
			format: "json",
			wantAll: []string{
				`"failed": 1`,
				`"error": "unavailable"`,
				`"ok": false`,
			},
		},
		{
			name: "text all passed",
			result: &tui.DiagnosticsResult{
				Endpoint:  "localhost:9091",
				Connected: true,
				Passed:    1,
				Services: []tui.ServiceResult{
					{Name: "HealthService", Method: "GetHealth", OK: true, Details: "healthy"},
				},
			},
			format: "table",
			wantAll: []string{
				"TUI Diagnostics",
				"localhost:9091",
			},
		},
		{
			name: "text with failures",
			result: &tui.DiagnosticsResult{
				Endpoint:  "localhost:9091",
				Connected: true,
				Passed:    1,
				Failed:    1,
				Services: []tui.ServiceResult{
					{Name: "HealthService", Method: "GetHealth", OK: true, Details: "healthy"},
					{Name: "MetricsService", Method: "GetMetrics", OK: false, Error: errors.New("unavailable")},
				},
			},
			format:  "table",
			wantErr: true,
			wantAll: []string{
				"TUI Diagnostics",
				"1 passed",
				"1 failed",
			},
		},
		{
			name: "json connection error",
			result: &tui.DiagnosticsResult{
				Endpoint:        "localhost:9091",
				Connected:       false,
				ConnectionError: errors.New("connection refused"),
				Failed:          1,
			},
			format: "json",
			wantAll: []string{
				`"connected": false`,
				`"error": "connection refused"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			p := NewPrinter(&buffer, tc.format, true, false)
			err := formatDiagnosticResult(tc.result, p, &buffer)

			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buffer.String()
			for _, want := range tc.wantAll {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, output)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
