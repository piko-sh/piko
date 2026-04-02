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

package config_domain

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestSummarise_Comprehensive(t *testing.T) {
	testCases := []struct {
		ctx            *LoadContext
		name           string
		wantContains   []string
		wantNotContain []string
		wantErr        bool
		wantEmpty      bool
	}{
		{
			name:    "Nil context",
			ctx:     nil,
			wantErr: true,
		},
		{
			name: "Nil target",
			ctx: &LoadContext{
				FieldSources: make(map[string]string),
				Target:       nil,
				Context:      context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Empty field sources",
			ctx: &LoadContext{
				FieldSources: make(map[string]string),
				Target:       &struct{ Name string }{Name: "test"},
				Context:      context.Background(),
			},
			wantEmpty: true,
		},
		{
			name: "Only default sources",
			ctx: &LoadContext{
				FieldSources: map[string]string{
					"Name": sourceDefault,
					"Port": sourceDefault,
				},
				Target:  &struct{ Name string }{Name: "test"},
				Context: context.Background(),
			},
			wantEmpty: true,
		},
		{
			name: "Single env source",
			ctx: &LoadContext{
				FieldSources: map[string]string{
					"Port": sourceEnv,
				},
				Target: &struct {
					Port int
				}{Port: 8080},
				Context: context.Background(),
			},
			wantContains: []string{"Port", "8080", sourceEnv},
		},
		{
			name: "Multiple sources grouped",
			ctx: &LoadContext{
				FieldSources: map[string]string{
					"Host": sourceEnv,
					"Port": sourceEnv,
					"Name": "file: config.yaml",
				},
				Target: &struct {
					Host string
					Name string
					Port int
				}{Host: "localhost", Port: 8080, Name: "myapp"},
				Context: context.Background(),
			},
			wantContains: []string{
				"Host", "localhost",
				"Port", "8080",
				"Name", "myapp",
				"Source: env",
				"Source: file: config.yaml",
			},
		},
		{
			name: "Nested struct field",
			ctx: &LoadContext{
				FieldSources: map[string]string{
					"Server.Port": sourceFlag,
				},
				Target: &struct {
					Server struct {
						Port int
					}
				}{
					Server: struct {
						Port int
					}{Port: 3000},
				},
				Context: context.Background(),
			},
			wantContains: []string{"Server.Port", "3000", sourceFlag},
		},
		{
			name: "Redacted field",
			ctx: &LoadContext{
				FieldSources: map[string]string{
					"Password": sourceEnv,
				},
				Target: &struct {
					Password string `summary:"hide"`
				}{Password: "secret123"},
				Context: context.Background(),
			},
			wantContains:   []string{"Password", "[REDACTED]"},
			wantNotContain: []string{"secret123"},
		},
		{
			name: "Mixed redacted and visible",
			ctx: &LoadContext{
				FieldSources: map[string]string{
					"Username": sourceEnv,
					"Password": sourceEnv,
				},
				Target: &struct {
					Username string
					Password string `summary:"hide"`
				}{Username: "admin", Password: "secret123"},
				Context: context.Background(),
			},
			wantContains:   []string{"Username", "admin", "Password", "[REDACTED]"},
			wantNotContain: []string{"secret123"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Summarise(tc.ctx)

			if (err != nil) != tc.wantErr {
				t.Fatalf("Summarise() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if tc.wantEmpty {
				if !strings.Contains(result, "No user-configured values") {
					t.Errorf("Expected empty summary message, got: %s", result)
				}
				return
			}

			for _, want := range tc.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected summary to contain %q, got: %s", want, result)
				}
			}

			for _, notWant := range tc.wantNotContain {
				if strings.Contains(result, notWant) {
					t.Errorf("Expected summary NOT to contain %q, got: %s", notWant, result)
				}
			}
		})
	}
}

func TestSummarise_InvalidFieldPath(t *testing.T) {
	ctx := &LoadContext{
		FieldSources: map[string]string{
			"NonExistent": sourceEnv,
		},
		Target:  &struct{ Name string }{Name: "test"},
		Context: context.Background(),
	}

	_, err := Summarise(ctx)
	if err == nil {
		t.Error("Expected error for non-existent field path, got nil")
	}
}

func TestSummarise_NilPointerInPath(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Config struct {
		Inner *Inner
	}

	ctx := &LoadContext{
		FieldSources: map[string]string{
			"Inner.Value": sourceEnv,
		},
		Target:  &Config{Inner: nil},
		Context: context.Background(),
	}

	_, err := Summarise(ctx)
	if err == nil {
		t.Error("Expected error for nil pointer in path, got nil")
	}
}

func TestGetFieldAndValueByPath(t *testing.T) {
	type Nested struct {
		Value string
	}
	type Config struct {
		Name   string
		Nested Nested
		Port   int
	}

	config := Config{
		Name: "test",
		Port: 8080,
		Nested: Nested{
			Value: "nested_value",
		},
	}

	testCases := []struct {
		wantValue any
		name      string
		path      string
		wantErr   bool
	}{
		{
			name:      "Simple field",
			path:      "Name",
			wantValue: "test",
		},
		{
			name:      "Nested field",
			path:      "Nested.Value",
			wantValue: "nested_value",
		},
		{
			name:    "Non-existent field",
			path:    "NonExistent",
			wantErr: true,
		},
		{
			name:    "Non-existent nested field",
			path:    "Nested.NonExistent",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configVal := reflect.Indirect(reflect.ValueOf(&config))
			_, value, err := getFieldAndValueByPath(configVal, tc.path)

			if (err != nil) != tc.wantErr {
				t.Fatalf("getFieldAndValueByPath() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if value.Interface() != tc.wantValue {
				t.Errorf("getFieldAndValueByPath() = %v, want %v", value.Interface(), tc.wantValue)
			}
		})
	}
}

func TestFormatSummary(t *testing.T) {
	entries := []summaryEntry{
		{
			KeyPath: "Host",
			Value:   "localhost",
			Source:  sourceEnv,
		},
		{
			KeyPath: "Port",
			Value:   8080,
			Source:  sourceEnv,
		},
		{
			KeyPath: "Name",
			Value:   "myapp",
			Source:  "file: config.yaml",
		},
	}

	result := formatSummary(entries)

	if !strings.Contains(result, "Applied Configuration Summary") {
		t.Error("Expected summary header")
	}

	if !strings.Contains(result, "[Source: env]") {
		t.Error("Expected env source header")
	}
	if !strings.Contains(result, "[Source: file: config.yaml]") {
		t.Error("Expected file source header")
	}

	if !strings.Contains(result, "Host") || !strings.Contains(result, "localhost") {
		t.Error("Expected Host field with localhost value")
	}
	if !strings.Contains(result, "Port") || !strings.Contains(result, "8080") {
		t.Error("Expected Port field with 8080 value")
	}
}
