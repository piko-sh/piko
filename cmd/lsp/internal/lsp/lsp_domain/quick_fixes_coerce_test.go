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

func TestGenerateCoerceFix_GuardClauses(t *testing.T) {
	testCases := []struct {
		data any
		name string
	}{
		{
			name: "nil data returns nil",
			data: nil,
		},
		{
			name: "wrong type returns nil",
			data: "not a map",
		},
		{
			name: "CanCoerce false returns nil",
			data: map[string]any{
				"can_coerce":    false,
				"prop_def_path": "/path/to/file.pk",
				"prop_name":     "myProp",
			},
		},
		{
			name: "empty PropDefPath returns nil",
			data: map[string]any{
				"can_coerce":    true,
				"prop_def_path": "",
				"prop_name":     "myProp",
			},
		},
		{
			name: "empty PropName returns nil",
			data: map[string]any{
				"can_coerce":    true,
				"prop_def_path": "/path/to/file.pk",
				"prop_name":     "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := &document{URI: "file:///test.pk"}
			ws := &workspace{
				docCache: NewDocumentCache(),
			}

			diagnostic := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 10},
				},
				Data: tc.data,
			}

			action := generateCoerceFix(context.Background(), diagnostic, document, ws)

			if action != nil {
				t.Error("expected nil action")
			}
		})
	}
}

func TestGenerateCoerceFix_ValidCase(t *testing.T) {
	targetURI := protocol.DocumentURI("file:///path/to/component.pk")
	sfcContent := `<template><div></div></template>
<script lang="go">
package main

type Props struct {
	Title string
}
</script>`

	docCache := NewDocumentCache()
	docCache.Set(targetURI, []byte(sfcContent))

	document := &document{URI: "file:///test.pk"}
	ws := &workspace{
		docCache: docCache,
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Data: map[string]any{
			"can_coerce":    true,
			"prop_def_path": "/path/to/component.pk",
			"prop_name":     "Title",
		},
	}

	action := generateCoerceFix(context.Background(), diagnostic, document, ws)

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

	if !strings.Contains(action.Title, "coerce") {
		t.Errorf("title %q should contain 'coerce'", action.Title)
	}

	if !strings.Contains(action.Title, "Title") {
		t.Errorf("title %q should contain prop name 'Title'", action.Title)
	}
}

func TestPrepareCoerceTagModification(t *testing.T) {
	sfcContent := `<template><div></div></template>
<script lang="go">
package main

type Props struct {
	Title string
}
</script>`

	targetURI := protocol.DocumentURI("file:///component.pk")
	docCache := NewDocumentCache()
	docCache.Set(targetURI, []byte(sfcContent))

	ws := &workspace{
		docCache: docCache,
	}

	code, script, err := prepareCoerceTagModification(ws, targetURI, "Title")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if script == nil {
		t.Fatal("expected non-nil script")
	}

	if !strings.Contains(code, "coerce") {
		t.Errorf("modified code should contain coerce tag: %s", code)
	}
}

func TestPrepareCoerceTagModification_Errors(t *testing.T) {
	testCases := []struct {
		setup     func(*workspace)
		name      string
		targetURI protocol.DocumentURI
		propName  string
	}{
		{
			name:      "file not in cache",
			targetURI: "file:///missing.pk",
			propName:  "Title",
			setup:     func(_ *workspace) {},
		},
		{
			name:      "field not found in Props",
			targetURI: "file:///component.pk",
			propName:  "NonExistent",
			setup: func(ws *workspace) {
				ws.docCache.Set("file:///component.pk", []byte(`<template><div></div></template>
<script lang="go">
package main

type Props struct {
	Title string
}
</script>`))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ws := &workspace{
				docCache: NewDocumentCache(),
			}
			tc.setup(ws)

			_, _, err := prepareCoerceTagModification(ws, tc.targetURI, tc.propName)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestParsePropsFromURI(t *testing.T) {
	testCases := []struct {
		name      string
		targetURI protocol.DocumentURI
		content   string
		wantErr   bool
	}{
		{
			name:      "valid SFC with Go script",
			targetURI: "file:///test.pk",
			content: `<template><div></div></template>
<script lang="go">
package main

type Props struct {
	Name string
}
</script>`,
			wantErr: false,
		},
		{
			name:      "file not in cache",
			targetURI: "file:///missing.pk",
			content:   "",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			docCache := NewDocumentCache()
			if tc.content != "" {
				docCache.Set(tc.targetURI, []byte(tc.content))
			}
			ws := &workspace{
				docCache: docCache,
			}

			script, goFile, fset, err := parsePropsFromURI(ws, tc.targetURI)

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
				t.Error("expected non-nil script")
			}
			if goFile == nil {
				t.Error("expected non-nil goFile")
			}
			if fset == nil {
				t.Error("expected non-nil fset")
			}
		})
	}
}

func TestFindPropsField(t *testing.T) {
	testCases := []struct {
		name     string
		source   string
		propName string
		wantErr  bool
	}{
		{
			name: "finds existing field",
			source: `package main

type Props struct {
	Title string
	Count int
}
`,
			propName: "Title",
			wantErr:  false,
		},
		{
			name: "field not found",
			source: `package main

type Props struct {
	Title string
}
`,
			propName: "Missing",
			wantErr:  true,
		},
		{
			name: "no Props struct",
			source: `package main

type Config struct {
	Value string
}
`,
			propName: "Value",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			goFile := parseGoSource(t, tc.source)

			field, err := findPropsField(goFile, tc.propName)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if field == nil {
				t.Fatal("expected non-nil field")
			}
		})
	}
}
