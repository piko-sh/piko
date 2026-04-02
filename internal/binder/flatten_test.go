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

package binder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlattenMapToFormData(t *testing.T) {
	testCases := []struct {
		input    map[string]any
		expected map[string][]string
		name     string
	}{
		{
			name:     "empty map",
			input:    map[string]any{},
			expected: map[string][]string{},
		},
		{
			name:  "simple string values",
			input: map[string]any{"name": "Alice", "city": "London"},
			expected: map[string][]string{
				"name": {"Alice"},
				"city": {"London"},
			},
		},
		{
			name:  "boolean values",
			input: map[string]any{"active": true, "disabled": false},
			expected: map[string][]string{
				"active":   {"true"},
				"disabled": {"false"},
			},
		},
		{
			name:  "integer float64 values",
			input: map[string]any{"count": float64(42), "min_height": float64(200)},
			expected: map[string][]string{
				"count":      {"42"},
				"min_height": {"200"},
			},
		},
		{
			name:  "fractional float64 values",
			input: map[string]any{"price": float64(3.14), "ratio": float64(0.5)},
			expected: map[string][]string{
				"price": {"3.14"},
				"ratio": {"0.5"},
			},
		},
		{
			name:  "nil values are skipped",
			input: map[string]any{"name": "Alice", "address": nil},
			expected: map[string][]string{
				"name": {"Alice"},
			},
		},
		{
			name: "nested map uses bracket notation",
			input: map[string]any{
				"address": map[string]any{
					"city":    "London",
					"country": "UK",
				},
			},
			expected: map[string][]string{
				"address['city']":    {"London"},
				"address['country']": {"UK"},
			},
		},
		{
			name: "array uses index notation",
			input: map[string]any{
				"tags": []any{"go", "web", "framework"},
			},
			expected: map[string][]string{
				"tags[0]": {"go"},
				"tags[1]": {"web"},
				"tags[2]": {"framework"},
			},
		},
		{
			name: "nested array of maps",
			input: map[string]any{
				"items": []any{
					map[string]any{"id": float64(1), "name": "First"},
					map[string]any{"id": float64(2), "name": "Second"},
				},
			},
			expected: map[string][]string{
				"items[0]['id']":   {"1"},
				"items[0]['name']": {"First"},
				"items[1]['id']":   {"2"},
				"items[1]['name']": {"Second"},
			},
		},
		{
			name: "deeply nested structure",
			input: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"address": map[string]any{
							"city": "London",
						},
					},
				},
			},
			expected: map[string][]string{
				"user['profile']['address']['city']": {"London"},
			},
		},
		{
			name: "bracket notation keys passed through",
			input: map[string]any{
				"fields['title']":       "Hello",
				"fields['description']": "World",
			},
			expected: map[string][]string{
				"fields['title']":       {"Hello"},
				"fields['description']": {"World"},
			},
		},
		{
			name: "mixed types at same level",
			input: map[string]any{
				"name":     "Alice",
				"age":      float64(30),
				"active":   true,
				"score":    float64(9.5),
				"nickname": nil,
			},
			expected: map[string][]string{
				"name":   {"Alice"},
				"age":    {"30"},
				"active": {"true"},
				"score":  {"9.5"},
			},
		},
		{
			name: "empty nested map",
			input: map[string]any{
				"config": map[string]any{},
			},
			expected: map[string][]string{},
		},
		{
			name: "empty array",
			input: map[string]any{
				"tags": []any{},
			},
			expected: map[string][]string{},
		},
		{
			name: "string boolean values are preserved as strings",
			input: map[string]any{
				"add_another": "true",
				"redirect":    "false",
			},
			expected: map[string][]string{
				"add_another": {"true"},
				"redirect":    {"false"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := flattenMapToFormData(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLeafToString(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected string
	}{
		{name: "string", input: "hello", expected: "hello"},
		{name: "empty string", input: "", expected: ""},
		{name: "true", input: true, expected: "true"},
		{name: "false", input: false, expected: "false"},
		{name: "float64 integer", input: float64(42), expected: "42"},
		{name: "float64 zero", input: float64(0), expected: "0"},
		{name: "float64 negative", input: float64(-5), expected: "-5"},
		{name: "float64 fractional", input: float64(3.14), expected: "3.14"},
		{name: "int", input: 42, expected: "42"},
		{name: "int64", input: int64(100), expected: "100"},
		{name: "uint64", input: uint64(255), expected: "255"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := leafToString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
