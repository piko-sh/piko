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

func TestGetFieldType(t *testing.T) {
	testCases := []struct {
		name          string
		suggestedType string
		want          string
	}{
		{
			name:          "empty returns string default",
			suggestedType: "",
			want:          "string",
		},
		{
			name:          "non-empty returns as-is",
			suggestedType: "int",
			want:          "int",
		},
		{
			name:          "complex type returns as-is",
			suggestedType: "map[string]int",
			want:          "map[string]int",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getFieldType(tc.suggestedType)
			if got != tc.want {
				t.Errorf("getFieldType(%q) = %q, want %q", tc.suggestedType, got, tc.want)
			}
		})
	}
}

func TestAddPropFieldToAST(t *testing.T) {
	testCases := []struct {
		name      string
		source    string
		wantField string
		fixData   undefinedVariableData
		wantErr   bool
	}{
		{
			name: "adds field to existing Props struct",
			source: `package main

type Props struct {
	Name string
}
`,
			fixData: undefinedVariableData{
				PropName:      "Age",
				SuggestedType: "int",
			},
			wantErr:   false,
			wantField: "Age",
		},
		{
			name: "uses default string type when SuggestedType is empty",
			source: `package main

type Props struct {}
`,
			fixData: undefinedVariableData{
				PropName:      "Title",
				SuggestedType: "",
			},
			wantErr:   false,
			wantField: "Title",
		},
		{
			name: "returns error when Props struct is missing",
			source: `package main

type Config struct {
	Value string
}
`,
			fixData: undefinedVariableData{
				PropName:      "Missing",
				SuggestedType: "string",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			goFile := parseGoSource(t, tc.source)

			err := addPropFieldToAST(goFile, tc.fixData)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			_, propsStruct, found := findPropsStruct(goFile)
			if !found {
				t.Fatal("Props struct not found after modification")
			}

			fieldFound := false
			for _, field := range propsStruct.Fields.List {
				for _, name := range field.Names {
					if name.Name == tc.wantField {
						fieldFound = true
					}
				}
			}

			if !fieldFound {
				t.Errorf("field %q not found in Props struct", tc.wantField)
			}
		})
	}
}

func TestPrepareAddPropModification(t *testing.T) {
	testCases := []struct {
		name         string
		content      string
		wantContains string
		fixData      undefinedVariableData
		wantErr      bool
	}{
		{
			name: "valid SFC with Props struct",
			content: `<template><div></div></template>
<script lang="go">
package main

type Props struct {
	Name string
}
</script>`,
			fixData: undefinedVariableData{
				PropName:      "Age",
				SuggestedType: "int",
			},
			wantErr:      false,
			wantContains: "Age",
		},
		{
			name:    "invalid SFC content",
			content: "",
			fixData: undefinedVariableData{
				PropName: "Field",
			},
			wantErr: true,
		},
		{
			name: "SFC without Go script block",
			content: `<template><div></div></template>
<script type="text/javascript">
console.log("hello");
</script>`,
			fixData: undefinedVariableData{
				PropName: "Field",
			},
			wantErr: true,
		},
		{
			name: "SFC with invalid Go script",
			content: `<template><div></div></template>
<script lang="go">
not valid go code {{{
</script>`,
			fixData: undefinedVariableData{
				PropName: "Field",
			},
			wantErr: true,
		},
		{
			name: "SFC with Go script but no Props struct",
			content: `<template><div></div></template>
<script lang="go">
package main

type Config struct {
	Value string
}
</script>`,
			fixData: undefinedVariableData{
				PropName: "Field",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code, script, err := prepareAddPropModification([]byte(tc.content), tc.fixData)

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

func TestGenerateUndefinedVariableFixes(t *testing.T) {
	testCases := []struct {
		data    any
		name    string
		wantLen int
	}{
		{
			name:    "nil data returns empty",
			data:    nil,
			wantLen: 0,
		},
		{
			name:    "wrong type returns empty",
			data:    "not a map",
			wantLen: 0,
		},
		{
			name: "suggestion only returns typo correction",
			data: map[string]any{
				"suggestion": "userName",
			},
			wantLen: 1,
		},
		{
			name: "is_prop without prop_name returns only suggestion",
			data: map[string]any{
				"suggestion": "myProp",
				"is_prop":    true,
				"prop_name":  "",
			},
			wantLen: 1,
		},
		{
			name: "no suggestion and no prop returns empty",
			data: map[string]any{
				"is_prop":   false,
				"prop_name": "",
			},
			wantLen: 0,
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

			actions := generateUndefinedVariableFixes(context.Background(), diagnostic, testDocument, ws)

			if len(actions) != tc.wantLen {
				t.Errorf("len(actions) = %d, want %d", len(actions), tc.wantLen)
			}
		})
	}
}

func TestGenerateAddToPropsEdit(t *testing.T) {
	sfcContent := `<template><div></div></template>
<script lang="go">
package main

type Props struct {
	Name string
}
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
	}

	testCases := []struct {
		name    string
		fixData undefinedVariableData
		wantNil bool
	}{
		{
			name: "valid prop addition",
			fixData: undefinedVariableData{
				PropName:      "Age",
				SuggestedType: "int",
			},
			wantNil: false,
		},
		{
			name: "uses default type when SuggestedType is empty",
			fixData: undefinedVariableData{
				PropName:      "Title",
				SuggestedType: "",
			},
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			action := generateAddToPropsEdit(context.Background(), diagnostic, testDocument, ws, tc.fixData)

			if tc.wantNil {
				if action != nil {
					t.Fatal("expected nil action")
				}
				return
			}

			if action == nil {
				t.Fatal("expected non-nil action")
			}

			if action.Kind != protocol.QuickFix {
				t.Errorf("Kind = %v, want %v", action.Kind, protocol.QuickFix)
			}

			if action.IsPreferred {
				t.Error("expected IsPreferred to be false")
			}

			if action.Edit == nil {
				t.Error("expected non-nil Edit")
			}

			if !strings.Contains(action.Title, tc.fixData.PropName) {
				t.Errorf("title %q should contain prop name %q", action.Title, tc.fixData.PropName)
			}
		})
	}
}

func TestGenerateAddToPropsEdit_ReturnsNilOnInvalidContent(t *testing.T) {
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
	}

	action := generateAddToPropsEdit(context.Background(), diagnostic, testDocument, ws, undefinedVariableData{
		PropName: "Missing",
	})

	if action != nil {
		t.Error("expected nil action when document has no Go script block")
	}
}
