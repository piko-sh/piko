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

package generator_domain_test

import (
	"context"
	"strings"
	"testing"

	"piko.sh/piko/internal/generator/generator_domain"
)

func TestJSTranspiler_Transpile(t *testing.T) {
	t.Parallel()

	transpiler := generator_domain.NewJSTranspiler()
	ctx := context.Background()

	testCases := []struct {
		name     string
		source   string
		opts     generator_domain.TranspileOptions
		contains []string
		excludes []string
		wantErr  bool
	}{
		{
			name:   "empty source",
			source: "",
			opts:   generator_domain.TranspileOptions{Filename: "test.ts"},
		},
		{
			name:     "simple typescript with type annotation",
			source:   `const x: number = 42;`,
			opts:     generator_domain.TranspileOptions{Filename: "test.ts"},
			contains: []string{"const x", "42"},
			excludes: []string{": number"},
		},
		{
			name: "typescript interface is stripped",
			source: `
interface User {
	name: string;
	age: number;
}
const user: User = { name: "Alice", age: 30 };
`,
			opts:     generator_domain.TranspileOptions{Filename: "test.ts"},
			contains: []string{"const user", "Alice", "30"},
			excludes: []string{"interface User", ": User"},
		},
		{
			name: "typescript as assertion is handled",
			source: `
const el = document.getElementById("test") as HTMLElement;
el.textContent = "hello";
`,
			opts:     generator_domain.TranspileOptions{Filename: "test.ts"},
			contains: []string{"const el", "getElementById", "textContent", "hello"},
			excludes: []string{"as HTMLElement"},
		},
		{
			name: "typescript function with typed parameters",
			source: `
function greet(name: string): string {
	return "Hello, " + name;
}
`,
			opts:     generator_domain.TranspileOptions{Filename: "test.ts"},
			contains: []string{"function greet", "name", "return", "Hello"},
			excludes: []string{": string"},
		},
		{
			name: "plain javascript passes through",
			source: `
const x = 42;
function add(a, b) { return a + b; }
`,
			opts:     generator_domain.TranspileOptions{Filename: "test.js"},
			contains: []string{"const x", "42", "function add", "return a + b"},
		},
		{
			name: "minified output",
			source: `
const message = "hello";
const value = 42;
`,
			opts:     generator_domain.TranspileOptions{Filename: "test.ts", Minify: true},
			contains: []string{"message", "hello", "value", "42"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := transpiler.Transpile(ctx, tc.source, tc.opts)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, s := range tc.contains {
				if !strings.Contains(result.Code, s) {
					t.Errorf("expected output to contain %q, got:\n%s", s, result.Code)
				}
			}

			for _, s := range tc.excludes {
				if strings.Contains(result.Code, s) {
					t.Errorf("expected output NOT to contain %q, got:\n%s", s, result.Code)
				}
			}
		})
	}
}

func TestJSTranspiler_Transpile_SyntaxError(t *testing.T) {
	t.Parallel()

	transpiler := generator_domain.NewJSTranspiler()
	ctx := context.Background()

	_, err := transpiler.Transpile(ctx, "const x = {{{", generator_domain.TranspileOptions{
		Filename: "broken.ts",
	})

	if err == nil {
		t.Error("expected error for syntax error, got nil")
	}
}
