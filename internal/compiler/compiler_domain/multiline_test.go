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

package compiler_domain

import (
	"testing"

	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestParseImportAliasesMultiline(t *testing.T) {
	tests := []struct {
		expected map[string]string
		name     string
		snippet  string
	}{
		{
			name: "multi-line import",
			snippet: `import {
				toUpperCase,
				toLowerCase,
				capitalize as capitalizeText
			} from '@/lib/string-utils.js';`,
			expected: map[string]string{
				"toUpperCase":    "toUpperCase",
				"toLowerCase":    "toLowerCase",
				"capitalizeText": "capitalize",
			},
		},
		{
			name: "multi-line with all aliases",
			snippet: `import {
				foo as bar,
				baz as qux
			} from './lib.js';`,
			expected: map[string]string{
				"bar": "foo",
				"qux": "baz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseImportAliases(tt.snippet)
			t.Logf("Result: %v", result)
			for local, expectedExport := range tt.expected {
				if gotExport, ok := result[local]; !ok {
					t.Errorf("missing local name %q in result", local)
				} else if gotExport != expectedExport {
					t.Errorf("for local %q: got export %q, want %q", local, gotExport, expectedExport)
				}
			}
		})
	}
}

func TestBuildImportFromRecordsMultiline(t *testing.T) {
	parser := NewTypeScriptParser()
	code := `import {
		toUpperCase,
		toLowerCase,
		capitalize as capitalizeText
	} from '@/lib/string-utils.js';`

	ast, err := parser.ParseTypeScript(code, "test.ts")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	statement, ok := buildImportFromRecords(ast, code)
	if !ok {
		t.Fatal("buildImportFromRecords returned false")
	}

	imp, ok := statement.Data.(*js_ast.SImport)
	if !ok {
		t.Fatalf("Statement is not SImport: %T", statement.Data)
	}

	if imp.Items == nil {
		t.Fatal("Items is nil")
	}

	items := *imp.Items
	t.Logf("Items count: %d", len(items))
	for i, item := range items {
		t.Logf("  Item %d: Alias=%q, OriginalName=%q", i, item.Alias, item.OriginalName)
	}

	found := false
	for _, item := range items {
		if item.Alias == "capitalize" && item.OriginalName == "capitalizeText" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find item with Alias='capitalize' and OriginalName='capitalizeText'")
	}

	for _, expectedName := range []string{"toUpperCase", "toLowerCase"} {
		found = false
		for _, item := range items {
			if item.Alias == expectedName && item.OriginalName == expectedName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find item with Alias='%s' and OriginalName='%s'", expectedName, expectedName)
		}
	}
}
