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

package lsp_domain

import (
	"context"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

func TestGenerateAddImportFix(t *testing.T) {
	testCases := []struct {
		data    any
		name    string
		wantNil bool
	}{
		{
			name:    "nil data returns nil",
			data:    nil,
			wantNil: true,
		},
		{
			name:    "wrong type returns nil",
			data:    "not a map",
			wantNil: true,
		},
		{
			name: "empty alias returns nil",
			data: map[string]any{
				"alias":       "",
				"import_path": "fmt",
			},
			wantNil: true,
		},
		{
			name: "empty import_path returns nil",
			data: map[string]any{
				"alias":       "fmt",
				"import_path": "",
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDocument := &document{
				URI:     "file:///test.pk",
				Content: []byte(`<template><div></div></template>`),
			}
			ws := &workspace{
				documents: map[protocol.DocumentURI]*document{testDocument.URI: testDocument},
				docCache:  NewDocumentCache(),
			}

			diagnostic := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 5},
				},
				Data: tc.data,
			}

			action := generateAddImportFix(context.Background(), diagnostic, testDocument, ws)

			if tc.wantNil {
				if action != nil {
					t.Error("expected nil action")
				}
				return
			}

			if action == nil {
				t.Fatal("expected non-nil action")
			}
		})
	}
}

func TestGenerateAddImportFix_ValidCase(t *testing.T) {
	sfcContent := `<template><div></div></template>
<script lang="go">
package main

func Render() {}
</script>`

	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(sfcContent),
	}
	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{testDocument.URI: testDocument},
		docCache:  NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 5},
		},
		Data: map[string]any{
			"alias":       "fmt",
			"import_path": "fmt",
		},
	}

	action := generateAddImportFix(context.Background(), diagnostic, testDocument, ws)

	if action == nil {
		t.Fatal("expected non-nil action")
	}

	if action.Kind != protocol.QuickFix {
		t.Errorf("Kind = %v, want %v", action.Kind, protocol.QuickFix)
	}

	if !action.IsPreferred {
		t.Error("expected IsPreferred to be true")
	}

	if action.Edit == nil {
		t.Fatal("expected non-nil Edit")
	}

	if !strings.Contains(action.Title, "fmt") {
		t.Errorf("title %q should contain import name", action.Title)
	}
}

func TestPrepareAddImportModification(t *testing.T) {
	testCases := []struct {
		fixData      missingImportData
		name         string
		content      string
		wantContains string
		wantErr      bool
	}{
		{
			name: "adds import to existing file",
			content: `<template><div></div></template>
<script lang="go">
package main

func Render() {}
</script>`,
			fixData: missingImportData{
				Alias:      "fmt",
				ImportPath: "fmt",
			},
			wantErr:      false,
			wantContains: `"fmt"`,
		},
		{
			name:    "empty content returns error",
			content: "",
			fixData: missingImportData{
				Alias:      "fmt",
				ImportPath: "fmt",
			},
			wantErr: true,
		},
		{
			name: "no Go script block returns error",
			content: `<template><div></div></template>
<script type="text/javascript">
console.log("hello");
</script>`,
			fixData: missingImportData{
				Alias:      "fmt",
				ImportPath: "fmt",
			},
			wantErr: true,
		},
		{
			name: "invalid Go script returns error",
			content: `<template><div></div></template>
<script lang="go">
not valid go {{{
</script>`,
			fixData: missingImportData{
				Alias:      "fmt",
				ImportPath: "fmt",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code, script, err := prepareAddImportModification([]byte(tc.content), tc.fixData)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if script == nil {
				t.Fatal("expected non-nil script")
			}

			if tc.wantContains != "" && !strings.Contains(code, tc.wantContains) {
				t.Errorf("code does not contain %q: %s", tc.wantContains, code)
			}
		})
	}
}
