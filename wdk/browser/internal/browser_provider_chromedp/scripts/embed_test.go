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

package scripts

import (
	"strings"
	"testing"
)

func TestGet_ValidFile(t *testing.T) {
	result := Get("check_focused.js")
	if result == "" {
		t.Error("Get(\"check_focused.js\") returned empty string")
	}
}

func TestGet_MissingFilePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Get with missing file should panic")
		}
	}()

	Get("nonexistent_file.js")
}

func TestMustGet_ReturnsNonEmpty(t *testing.T) {
	result := MustGet("check_focused.js")
	if result == "" {
		t.Error("MustGet(\"check_focused.js\") returned empty string")
	}
}

func TestMustGet_SameAsGet(t *testing.T) {
	get := Get("check_focused.js")
	mustGet := MustGet("check_focused.js")
	if get != mustGet {
		t.Error("MustGet should return the same value as Get")
	}
}

func TestExecute_ValidTemplate(t *testing.T) {
	result, err := Execute("element_bounds_with_padding.js.tmpl", map[string]any{
		"Selector": "#test",
		"Padding":  10.0,
	})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Execute returned empty string for valid template")
	}
}

func TestExecute_MissingTemplate(t *testing.T) {
	_, err := Execute("nonexistent.js.tmpl", nil)
	if err == nil {
		t.Error("Execute with missing template should return error")
	}
}

func TestMustExecute_ValidTemplate(t *testing.T) {
	result := MustExecute("element_bounds_with_padding.js.tmpl", map[string]any{
		"Selector": "#test",
		"Padding":  10.0,
	})
	if result == "" {
		t.Error("MustExecute returned empty string for valid template")
	}
}

func TestMustExecute_PanicsOnMissingTemplate(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("MustExecute with missing template should panic")
		}
	}()

	MustExecute("nonexistent.js.tmpl", nil)
}

func TestGetTemplate_CachesResult(t *testing.T) {
	template1, err := getTemplate("element_bounds_with_padding.js.tmpl")
	if err != nil {
		t.Fatalf("First getTemplate call failed: %v", err)
	}

	template2, err := getTemplate("element_bounds_with_padding.js.tmpl")
	if err != nil {
		t.Fatalf("Second getTemplate call failed: %v", err)
	}

	if template1 != template2 {
		t.Error("getTemplate should return the same cached *template.Template on repeated calls")
	}
}

func TestGetTemplate_MissingFile(t *testing.T) {
	_, err := getTemplate("does_not_exist.js.tmpl")
	if err == nil {
		t.Error("getTemplate with missing file should return error")
	}
}

func TestTemplateFuncs_JsStr(t *testing.T) {
	templateFunction, ok := templateFuncs["jsStr"].(func(string) string)
	if !ok {
		t.Fatal("type assertion failed for jsStr")
	}

	testCases := []struct {
		name     string
		input    string
		contains string
	}{
		{name: "simple string", input: "hello", contains: `"hello"`},
		{name: "string with quotes", input: `say "hi"`, contains: `\"`},
		{name: "string with newline", input: "line1\nline2", contains: `\n`},
		{name: "empty string", input: "", contains: `""`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := templateFunction(tc.input)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("jsStr(%q) = %q, expected to contain %q", tc.input, result, tc.contains)
			}
		})
	}
}

func TestTemplateFuncs_JsInt(t *testing.T) {
	templateFunction, ok := templateFuncs["jsInt"].(func(int64) string)
	if !ok {
		t.Fatal("type assertion failed for jsInt")
	}

	testCases := []struct {
		name     string
		expected string
		input    int64
	}{
		{name: "zero", input: 0, expected: "0"},
		{name: "positive", input: 42, expected: "42"},
		{name: "negative", input: -7, expected: "-7"},
		{name: "large number", input: 1000000, expected: "1000000"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := templateFunction(tc.input)
			if result != tc.expected {
				t.Errorf("jsInt(%d) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestTemplateFuncs_JsFloat(t *testing.T) {
	templateFunction, ok := templateFuncs["jsFloat"].(func(float64) string)
	if !ok {
		t.Fatal("type assertion failed for jsFloat")
	}

	testCases := []struct {
		name     string
		expected string
		input    float64
	}{
		{name: "zero", input: 0.0, expected: "0"},
		{name: "integer-like", input: 1.0, expected: "1"},
		{name: "decimal", input: 3.14, expected: "3.14"},
		{name: "negative", input: -2.5, expected: "-2.5"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := templateFunction(tc.input)
			if result != tc.expected {
				t.Errorf("jsFloat(%f) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestTemplateFuncs_JsBool(t *testing.T) {
	templateFunction, ok := templateFuncs["jsBool"].(func(bool) string)
	if !ok {
		t.Fatal("type assertion failed for jsBool")
	}

	if templateFunction(true) != "true" {
		t.Errorf("jsBool(true) = %q, expected %q", templateFunction(true), "true")
	}
	if templateFunction(false) != "false" {
		t.Errorf("jsBool(false) = %q, expected %q", templateFunction(false), "false")
	}
}

func TestTemplateFuncs_JsRaw(t *testing.T) {
	templateFunction, ok := templateFuncs["jsRaw"].(func(string) string)
	if !ok {
		t.Fatal("type assertion failed for jsRaw")
	}

	input := "document.querySelector('#foo')"
	result := templateFunction(input)
	if result != input {
		t.Errorf("jsRaw(%q) = %q, expected identity pass-through", input, result)
	}
}
