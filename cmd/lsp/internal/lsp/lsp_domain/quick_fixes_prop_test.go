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
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestGetDefaultSuggestedValue(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty returns empty string literal",
			input: "",
			want:  `""`,
		},
		{
			name:  "non-empty returns as-is",
			input: `"Default Title"`,
			want:  `"Default Title"`,
		},
		{
			name:  "numeric value returns as-is",
			input: "42",
			want:  "42",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getDefaultSuggestedValue(tc.input)
			if got != tc.want {
				t.Errorf("getDefaultSuggestedValue(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestBuildInsertionEdit(t *testing.T) {
	uri := protocol.DocumentURI("file:///test.pk")
	edit := buildInsertionEdit(uri, 5, 10, " :title=\"hello\"")

	edits, ok := edit.Changes[uri]
	if !ok {
		t.Fatal("expected changes for URI")
	}

	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}

	textEdit := edits[0]
	if textEdit.Range.Start.Line != 5 {
		t.Errorf("Start.Line = %d, want 5", textEdit.Range.Start.Line)
	}
	if textEdit.Range.Start.Character != 10 {
		t.Errorf("Start.Character = %d, want 10", textEdit.Range.Start.Character)
	}
	if textEdit.Range.End.Line != 5 {
		t.Errorf("End.Line = %d, want 5", textEdit.Range.End.Line)
	}
	if textEdit.Range.End.Character != 10 {
		t.Errorf("End.Character = %d, want 10", textEdit.Range.End.Character)
	}
	if textEdit.NewText != " :title=\"hello\"" {
		t.Errorf("NewText = %q, want %q", textEdit.NewText, " :title=\"hello\"")
	}
}

func TestBuildPropFixTitle(t *testing.T) {
	testCases := []struct {
		name          string
		propName      string
		want          string
		isSelfClosing bool
	}{
		{
			name:          "regular tag",
			propName:      "title",
			isSelfClosing: false,
			want:          "Add missing required prop 'title'",
		},
		{
			name:          "self-closing tag",
			propName:      "value",
			isSelfClosing: true,
			want:          "Add missing required prop 'value' (self-closing tag)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildPropFixTitle(tc.propName, tc.isSelfClosing)
			if got != tc.want {
				t.Errorf("buildPropFixTitle(%q, %v) = %q, want %q", tc.propName, tc.isSelfClosing, got, tc.want)
			}
		})
	}
}

func TestFindPropInsertionPoint(t *testing.T) {
	testCases := []struct {
		document        *document
		name            string
		position        protocol.Position
		wantErr         bool
		wantSelfClosing bool
	}{
		{
			name:     "nil annotation result returns error",
			document: newTestDocumentBuilder().WithURI("file:///test.pk").Build(),
			position: protocol.Position{Line: 0, Character: 0},
			wantErr:  true,
		},
		{
			name: "nil annotated AST returns error",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				Build(),
			position: protocol.Position{Line: 0, Character: 0},
			wantErr:  true,
		},
		{
			name: "empty root nodes returns error",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{}},
				}).
				Build(),
			position: protocol.Position{Line: 0, Character: 0},
			wantErr:  true,
		},
		{
			name: "tag not found at position returns error",
			document: func() *document {
				node := newTestNode("MyComponent", 5, 1)
				d := newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithContent("<template>\n  <MyComponent />\n</template>").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
				return d
			}(),
			position: protocol.Position{Line: 99, Character: 0},
			wantErr:  true,
		},
		{
			name: "finds self-closing tag",
			document: func() *document {
				node := newTestNode("MyComponent", 2, 3)
				d := newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithContent("<template>\n  <MyComponent />\n</template>").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
				return d
			}(),
			position:        protocol.Position{Line: 1, Character: 3},
			wantErr:         false,
			wantSelfClosing: true,
		},
		{
			name: "finds regular tag",
			document: func() *document {
				node := newTestNode("MyComponent", 2, 3)
				d := newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithContent("<template>\n  <MyComponent>\n  </MyComponent>\n</template>").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
				return d
			}(),
			position:        protocol.Position{Line: 1, Character: 3},
			wantErr:         false,
			wantSelfClosing: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := findPropInsertionPoint(tc.document, tc.position)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info == nil {
				t.Fatal("expected non-nil propInsertionInfo")
			}

			if info.isSelfClosing != tc.wantSelfClosing {
				t.Errorf("isSelfClosing = %v, want %v", info.isSelfClosing, tc.wantSelfClosing)
			}
		})
	}
}

func TestGenerateAddMissingPropFix(t *testing.T) {
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
			name: "empty PropName returns nil",
			data: map[string]any{
				"prop_name": "",
				"prop_type": "string",
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDocument := &document{
				URI:     "file:///test.pk",
				Content: []byte("<template><div></div></template>"),
			}
			ws := &workspace{
				documents: map[protocol.DocumentURI]*document{testDocument.URI: testDocument},
				docCache:  NewDocumentCache(),
			}

			diagnostic := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 10},
				},
				Data: tc.data,
			}

			action := generateAddMissingPropFix(context.Background(), diagnostic, testDocument, ws)

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

func TestGenerateAddMissingPropFix_ValidCase(t *testing.T) {
	node := newTestNode("MyComponent", 2, 3)

	testDocument := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithContent("<template>\n  <MyComponent />\n</template>").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{testDocument.URI: testDocument},
		docCache:  NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 3},
			End:   protocol.Position{Line: 1, Character: 14},
		},
		Data: map[string]any{
			"prop_name":       "title",
			"prop_type":       "string",
			"suggested_value": `"hello"`,
		},
	}

	action := generateAddMissingPropFix(context.Background(), diagnostic, testDocument, ws)

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
}
