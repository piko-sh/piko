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
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserStep_ExpectedString(t *testing.T) {
	testCases := []struct {
		name     string
		expected any
		want     string
	}{
		{
			name:     "nil returns empty string",
			expected: nil,
			want:     "",
		},
		{
			name:     "string returns as-is",
			expected: "hello",
			want:     "hello",
		},
		{
			name:     "empty string",
			expected: "",
			want:     "",
		},
		{
			name:     "int converts to string",
			expected: 42,
			want:     "42",
		},
		{
			name:     "float64 converts to string",
			expected: 3.14,
			want:     "3.14",
		},
		{
			name:     "bool converts to string",
			expected: true,
			want:     "true",
		},
		{
			name:     "negative int",
			expected: -10,
			want:     "-10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := &BrowserStep{Expected: tc.expected}
			got := step.ExpectedString()
			if got != tc.want {
				t.Errorf("ExpectedString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBrowserStep_ExpectedInt(t *testing.T) {
	testCases := []struct {
		expected any
		name     string
		want     int
	}{
		{
			name:     "nil returns zero",
			expected: nil,
			want:     0,
		},
		{
			name:     "int returns as-is",
			expected: 42,
			want:     42,
		},
		{
			name:     "negative int",
			expected: -10,
			want:     -10,
		},
		{
			name:     "float64 truncates to int",
			expected: 3.14,
			want:     3,
		},
		{
			name:     "float64 9.99 truncates to 9",
			expected: 9.99,
			want:     9,
		},
		{
			name:     "string numeric parses to int",
			expected: "123",
			want:     123,
		},
		{
			name:     "string with leading zeros",
			expected: "007",
			want:     7,
		},
		{
			name:     "string non-numeric returns zero",
			expected: "abc",
			want:     0,
		},
		{
			name:     "string empty returns zero",
			expected: "",
			want:     0,
		},
		{
			name:     "bool returns zero",
			expected: true,
			want:     0,
		},
		{
			name:     "slice returns zero",
			expected: []int{1, 2, 3},
			want:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := &BrowserStep{Expected: tc.expected}
			got := step.ExpectedInt()
			if got != tc.want {
				t.Errorf("ExpectedInt() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestBrowserStep_ExpectedMap(t *testing.T) {
	testCases := []struct {
		expected any
		name     string
		wantLen  int
		wantNil  bool
	}{
		{
			name:     "nil returns nil",
			expected: nil,
			wantNil:  true,
		},
		{
			name:     "map returns map",
			expected: map[string]any{"key": "value"},
			wantNil:  false,
			wantLen:  1,
		},
		{
			name:     "empty map returns empty map",
			expected: map[string]any{},
			wantNil:  false,
			wantLen:  0,
		},
		{
			name:     "multi-key map",
			expected: map[string]any{"a": 1, "b": 2, "c": 3},
			wantNil:  false,
			wantLen:  3,
		},
		{
			name:     "string returns nil",
			expected: "not a map",
			wantNil:  true,
		},
		{
			name:     "int returns nil",
			expected: 42,
			wantNil:  true,
		},
		{
			name:     "slice returns nil",
			expected: []string{"a", "b"},
			wantNil:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := &BrowserStep{Expected: tc.expected}
			got := step.ExpectedMap()

			if tc.wantNil {
				if got != nil {
					t.Errorf("ExpectedMap() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("ExpectedMap() = nil, want map with %d elements", tc.wantLen)
				return
			}

			if len(got) != tc.wantLen {
				t.Errorf("ExpectedMap() has %d elements, want %d", len(got), tc.wantLen)
			}
		})
	}
}

func TestBrowserStep_AttributeName(t *testing.T) {
	testCases := []struct {
		name      string
		attribute string
		nameField string
		want      string
	}{
		{
			name:      "attribute takes precedence",
			attribute: "data-custom",
			nameField: "should-be-ignored",
			want:      "data-custom",
		},
		{
			name:      "name used when attribute empty",
			attribute: "",
			nameField: "class",
			want:      "class",
		},
		{
			name:      "both empty returns empty",
			attribute: "",
			nameField: "",
			want:      "",
		},
		{
			name:      "only attribute set",
			attribute: "href",
			nameField: "",
			want:      "href",
		},
		{
			name:      "only name set",
			attribute: "",
			nameField: "title",
			want:      "title",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := &BrowserStep{
				Attribute: tc.attribute,
				Name:      tc.nameField,
			}
			got := step.AttributeName()
			if got != tc.want {
				t.Errorf("AttributeName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLoadTestSpec(t *testing.T) {
	t.Run("valid testspec with all fields", func(t *testing.T) {
		json := `{
			"description": "Test description",
			"requestURL": "/test",
			"expectedStatus": 201,
			"shouldError": false,
			"browserSteps": [
				{"action": "click", "selector": "#btn"},
				{"action": "fill", "selector": "#input", "value": "hello"}
			],
			"pkcComponents": [
				{"tagName": "pp-counter", "selector": "#counter"}
			],
			"networkMocks": [
				{"path": "/api/test", "status": 200, "body": "{}"}
			],
			"partials": {
				"counter": {
					"endpoint": "/partials/counter",
					"stages": [{"file": "counter.pk", "stage": 0}]
				}
			}
		}`

		path := writeTestFile(t, "testspec.json", json)
		spec, err := LoadTestSpec(path)
		if err != nil {
			t.Fatalf("LoadTestSpec() error = %v", err)
		}

		if spec.Description != "Test description" {
			t.Errorf("Description = %q, want %q", spec.Description, "Test description")
		}
		if spec.RequestURL != "/test" {
			t.Errorf("RequestURL = %q, want %q", spec.RequestURL, "/test")
		}
		if spec.ExpectedStatus != 201 {
			t.Errorf("ExpectedStatus = %d, want %d", spec.ExpectedStatus, 201)
		}
		if len(spec.BrowserSteps) != 2 {
			t.Errorf("BrowserSteps length = %d, want 2", len(spec.BrowserSteps))
		}
		if len(spec.PKCComponents) != 1 {
			t.Errorf("PKCComponents length = %d, want 1", len(spec.PKCComponents))
		}
		if len(spec.NetworkMocks) != 1 {
			t.Errorf("NetworkMocks length = %d, want 1", len(spec.NetworkMocks))
		}
		if len(spec.Partials) != 1 {
			t.Errorf("Partials length = %d, want 1", len(spec.Partials))
		}
	})

	t.Run("default status when not specified", func(t *testing.T) {
		json := `{"description": "Test", "requestURL": "/"}`

		path := writeTestFile(t, "testspec.json", json)
		spec, err := LoadTestSpec(path)
		if err != nil {
			t.Fatalf("LoadTestSpec() error = %v", err)
		}

		if spec.ExpectedStatus != DefaultExpectedStatus {
			t.Errorf("ExpectedStatus = %d, want default %d", spec.ExpectedStatus, DefaultExpectedStatus)
		}
	})

	t.Run("minimal valid testspec", func(t *testing.T) {
		json := `{"description": "Minimal", "requestURL": "/minimal"}`

		path := writeTestFile(t, "testspec.json", json)
		spec, err := LoadTestSpec(path)
		if err != nil {
			t.Fatalf("LoadTestSpec() error = %v", err)
		}

		if spec.Description != "Minimal" {
			t.Errorf("Description = %q, want %q", spec.Description, "Minimal")
		}
		if len(spec.BrowserSteps) != 0 {
			t.Errorf("BrowserSteps should be empty, got %v", spec.BrowserSteps)
		}
	})

	t.Run("file not found error", func(t *testing.T) {
		_, err := LoadTestSpec("/nonexistent/path/testspec.json")
		if err == nil {
			t.Error("LoadTestSpec() expected error for nonexistent file")
		}
	})

	t.Run("invalid JSON error", func(t *testing.T) {
		path := writeTestFile(t, "testspec.json", "{ invalid json }")
		_, err := LoadTestSpec(path)
		if err == nil {
			t.Error("LoadTestSpec() expected error for invalid JSON")
		}
	})

	t.Run("shouldError field", func(t *testing.T) {
		json := `{
			"description": "Error test",
			"requestURL": "/error",
			"shouldError": true,
			"errorContains": "expected error"
		}`

		path := writeTestFile(t, "testspec.json", json)
		spec, err := LoadTestSpec(path)
		if err != nil {
			t.Fatalf("LoadTestSpec() error = %v", err)
		}

		if !spec.ShouldError {
			t.Error("ShouldError = false, want true")
		}
		if spec.ErrorContains != "expected error" {
			t.Errorf("ErrorContains = %q, want %q", spec.ErrorContains, "expected error")
		}
	})
}

func TestIsAssertionAction(t *testing.T) {
	assertionActions := []string{
		"checkText",
		"checkValue",
		"checkAttribute",
		"checkClass",
		"checkStyle",
		"checkFocused",
		"checkNotFocused",
		"checkVisible",
		"checkHidden",
		"checkEnabled",
		"checkDisabled",
		"checkChecked",
		"checkUnchecked",
		"checkElementCount",
		"checkHTML",
		"checkFormData",
		"checkConsoleMessage",
		"checkNoConsoleMessage",
		"checkNoConsoleErrors",
		"checkNoConsoleWarnings",
	}

	for _, action := range assertionActions {
		t.Run(action+" is assertion", func(t *testing.T) {
			if !IsAssertionAction(action) {
				t.Errorf("IsAssertionAction(%q) = false, want true", action)
			}
		})
	}

	nonAssertionActions := []string{
		"click",
		"fill",
		"press",
		"navigate",
		"wait",
		"type",
		"select",
		"hover",
		"focus",
		"blur",
		"scroll",
		"captureDOM",
		"unknown",
		"",
	}

	for _, action := range nonAssertionActions {
		t.Run(action+" is not assertion", func(t *testing.T) {
			if IsAssertionAction(action) {
				t.Errorf("IsAssertionAction(%q) = true, want false", action)
			}
		})
	}
}

func writeTestFile(t *testing.T, filename, content string) string {
	t.Helper()

	directory := t.TempDir()
	path := filepath.Join(directory, filename)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	return path
}
