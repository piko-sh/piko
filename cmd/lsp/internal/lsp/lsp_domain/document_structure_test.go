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

func TestNodeRangeToLSP(t *testing.T) {
	testCases := []struct {
		name string
		r    ast_domain.Range
		want protocol.Range
	}{
		{
			name: "converts 1-based to 0-based",
			r: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 1},
				End:   ast_domain.Location{Line: 1, Column: 10},
			},
			want: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 9},
			},
		},
		{
			name: "multi-line range",
			r: ast_domain.Range{
				Start: ast_domain.Location{Line: 5, Column: 3},
				End:   ast_domain.Location{Line: 10, Column: 20},
			},
			want: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 2},
				End:   protocol.Position{Line: 9, Character: 19},
			},
		},
		{
			name: "zero values remain zero after underflow protection",
			r: ast_domain.Range{
				Start: ast_domain.Location{Line: 0, Column: 0},
				End:   ast_domain.Location{Line: 0, Column: 0},
			},
			want: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := nodeRangeToLSP(tc.r)
			if got != tc.want {
				t.Errorf("nodeRangeToLSP(%v) = %v, want %v", tc.r, got, tc.want)
			}
		})
	}
}

func TestExtractIDSymbol(t *testing.T) {
	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		wantName string
		wantNil  bool
	}{
		{
			name:    "no attributes",
			node:    newTestNode("div", 1, 1),
			wantNil: true,
		},
		{
			name: "has id attribute",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				addAttribute(n, "id", "main-header")
				return n
			}(),
			wantNil:  false,
			wantName: "main-header",
		},
		{
			name: "has class but no id",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				addAttribute(n, "class", "container")
				return n
			}(),
			wantNil: true,
		},
		{
			name: "id is first among multiple attributes",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("section", 1, 1)
				addAttribute(n, "id", "sidebar")
				addAttribute(n, "class", "panel")
				return n
			}(),
			wantNil:  false,
			wantName: "sidebar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractIDSymbol(tc.node)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.name != tc.wantName {
				t.Errorf("name = %q, want %q", got.name, tc.wantName)
			}
			if got.kind != protocol.SymbolKindClass {
				t.Errorf("kind = %v, want SymbolKindClass", got.kind)
			}
		})
	}
}

func TestExtractIsAttrSymbol(t *testing.T) {
	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		wantName string
		wantNil  bool
	}{
		{
			name:    "no attributes",
			node:    newTestNode("div", 1, 1),
			wantNil: true,
		},
		{
			name: "has is attribute",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				addAttribute(n, "is", "card")
				return n
			}(),
			wantNil:  false,
			wantName: "card",
		},
		{
			name: "has other attributes but no is",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				addAttribute(n, "id", "test")
				return n
			}(),
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractIsAttrSymbol(tc.node)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.name != tc.wantName {
				t.Errorf("name = %q, want %q", got.name, tc.wantName)
			}
			if got.kind != protocol.SymbolKindModule {
				t.Errorf("kind = %v, want SymbolKindModule", got.kind)
			}
		})
	}
}

func TestExtractPartialInfoSymbol(t *testing.T) {
	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		wantName string
		wantNil  bool
	}{
		{
			name:    "nil GoAnnotations",
			node:    newTestNode("div", 1, 1),
			wantNil: true,
		},
		{
			name: "nil PartialInfo",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
				return n
			}(),
			wantNil: true,
		},
		{
			name: "PartialInfo with alias",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialAlias: "card",
					},
				}
				return n
			}(),
			wantNil:  false,
			wantName: "card",
		},
		{
			name: "PartialInfo with invocation key only",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey: "inv_123",
					},
				}
				return n
			}(),
			wantNil:  false,
			wantName: "inv_123",
		},
		{
			name: "PartialInfo with empty alias and key",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{},
				}
				return n
			}(),
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractPartialInfoSymbol(tc.node)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.name != tc.wantName {
				t.Errorf("name = %q, want %q", got.name, tc.wantName)
			}
		})
	}
}

func TestExtractStructuralBlockSymbol(t *testing.T) {
	testCases := []struct {
		name     string
		tagName  string
		wantName string
		wantNil  bool
	}{
		{name: "template block", tagName: "template", wantName: "<template>"},
		{name: "script block", tagName: "script", wantName: "<script>"},
		{name: "style block", tagName: "style", wantName: "<style>"},
		{name: "i18n block", tagName: "i18n", wantName: "<i18n>"},
		{name: "unknown tag", tagName: "div", wantNil: true},
		{name: "empty tag", tagName: "", wantNil: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := newTestNode(tc.tagName, 1, 1)
			got := extractStructuralBlockSymbol(node)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.name != tc.wantName {
				t.Errorf("name = %q, want %q", got.name, tc.wantName)
			}
			if got.kind != protocol.SymbolKindNamespace {
				t.Errorf("kind = %v, want SymbolKindNamespace", got.kind)
			}
		})
	}
}

func TestExtractSymbolInfo(t *testing.T) {
	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		wantName string
		wantNil  bool
	}{
		{
			name: "prefers id over is",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				addAttribute(n, "id", "header")
				addAttribute(n, "is", "card")
				return n
			}(),
			wantName: "header",
		},
		{
			name: "falls back to is attribute",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				addAttribute(n, "is", "card")
				return n
			}(),
			wantName: "card",
		},
		{
			name: "falls back to partial info",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				n.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialAlias: "sidebar",
					},
				}
				return n
			}(),
			wantName: "sidebar",
		},
		{
			name:     "falls back to structural block",
			node:     newTestNode("template", 1, 1),
			wantName: "<template>",
		},
		{
			name:    "returns nil for plain div",
			node:    newTestNode("div", 1, 1),
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractSymbolInfo(tc.node)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.name != tc.wantName {
				t.Errorf("name = %q, want %q", got.name, tc.wantName)
			}
		})
	}
}

func TestExtractSymbolFromNode(t *testing.T) {
	document := &document{}

	testCases := []struct {
		node    *ast_domain.TemplateNode
		name    string
		wantNil bool
	}{
		{
			name:    "nil node",
			node:    nil,
			wantNil: true,
		},
		{
			name: "zero line",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 0, 0)
				addAttribute(n, "id", "test")
				return n
			}(),
			wantNil: true,
		},
		{
			name: "non-element node type",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("text", 1, 1)
				n.NodeType = ast_domain.NodeText
				return n
			}(),
			wantNil: true,
		},
		{
			name:    "element with no symbol info",
			node:    newTestNode("div", 1, 1),
			wantNil: true,
		},
		{
			name: "valid element with id",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 1, 1)
				addAttribute(n, "id", "content")
				return n
			}(),
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := document.extractSymbolFromNode(tc.node)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

func TestIsSameSymbol(t *testing.T) {
	testCases := []struct {
		targetRef   *symbolReference
		name        string
		sourcePath  string
		defLocation ast_domain.Location
		want        bool
	}{
		{
			name:        "matching",
			defLocation: ast_domain.Location{Line: 10, Column: 5},
			sourcePath:  "/test/file.pk",
			targetRef:   &symbolReference{line: 10, column: 5, sourcePath: "/test/file.pk"},
			want:        true,
		},
		{
			name:        "different line",
			defLocation: ast_domain.Location{Line: 11, Column: 5},
			sourcePath:  "/test/file.pk",
			targetRef:   &symbolReference{line: 10, column: 5, sourcePath: "/test/file.pk"},
			want:        false,
		},
		{
			name:        "different column",
			defLocation: ast_domain.Location{Line: 10, Column: 6},
			sourcePath:  "/test/file.pk",
			targetRef:   &symbolReference{line: 10, column: 5, sourcePath: "/test/file.pk"},
			want:        false,
		},
		{
			name:        "different path",
			defLocation: ast_domain.Location{Line: 10, Column: 5},
			sourcePath:  "/test/other.pk",
			targetRef:   &symbolReference{line: 10, column: 5, sourcePath: "/test/file.pk"},
			want:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isSameSymbol(tc.defLocation, tc.sourcePath, tc.targetRef)
			if got != tc.want {
				t.Errorf("isSameSymbol() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetSourcePath(t *testing.T) {
	testCases := []struct {
		name string
		ann  *ast_domain.GoGeneratorAnnotation
		want string
	}{
		{
			name: "nil OriginalSourcePath",
			ann:  &ast_domain.GoGeneratorAnnotation{},
			want: "",
		},
		{
			name: "non-nil OriginalSourcePath",
			ann: func() *ast_domain.GoGeneratorAnnotation {
				return &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("/test/file.pk"),
				}
			}(),
			want: "/test/file.pk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getSourcePath(tc.ann)
			if got != tc.want {
				t.Errorf("getSourcePath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMatchSymbolHighlight(t *testing.T) {
	testCases := []struct {
		expression ast_domain.Expression
		target     *symbolReference
		rngMap     map[ast_domain.Expression]protocol.Range
		name       string
		wantNil    bool
	}{
		{
			name:       "nil expression",
			expression: nil,
			target:     &symbolReference{line: 1, column: 1},
			rngMap:     map[ast_domain.Expression]protocol.Range{},
			wantNil:    true,
		},
		{
			name:       "nil annotation",
			expression: &ast_domain.Identifier{Name: "x"},
			target:     &symbolReference{line: 1, column: 1},
			rngMap:     map[ast_domain.Expression]protocol.Range{},
			wantNil:    true,
		},
		{
			name: "nil symbol",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "x"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
				return id
			}(),
			target:  &symbolReference{line: 1, column: 1},
			rngMap:  map[ast_domain.Expression]protocol.Range{},
			wantNil: true,
		},
		{
			name: "synthetic location",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "x"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					Symbol: &ast_domain.ResolvedSymbol{
						Name:              "x",
						ReferenceLocation: ast_domain.Location{Line: 0, Column: 0},
					},
				}
				return id
			}(),
			target:  &symbolReference{line: 1, column: 1},
			rngMap:  map[ast_domain.Expression]protocol.Range{},
			wantNil: true,
		},
		{
			name: "non-matching symbol",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "x"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					Symbol: &ast_domain.ResolvedSymbol{
						Name:              "x",
						ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
					},
				}
				return id
			}(),
			target:  &symbolReference{line: 1, column: 1},
			rngMap:  map[ast_domain.Expression]protocol.Range{},
			wantNil: true,
		},
		{
			name: "matching symbol found in range map",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "x"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					Symbol: &ast_domain.ResolvedSymbol{
						Name:              "x",
						ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
					},
				}
				return id
			}(),
			target:  &symbolReference{line: 5, column: 3, sourcePath: ""},
			wantNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rngMap := tc.rngMap
			if rngMap == nil {
				rngMap = map[ast_domain.Expression]protocol.Range{}
			}

			if tc.expression != nil && !tc.wantNil {
				rngMap[tc.expression] = protocol.Range{
					Start: protocol.Position{Line: 4, Character: 2},
					End:   protocol.Position{Line: 4, Character: 5},
				}
			}
			got := matchSymbolHighlight(tc.expression, tc.target, rngMap)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

func TestGetDocumentHighlights_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "nil AnnotationResult",
			document: newTestDocumentBuilder().Build(),
		},
		{
			name: "nil AnnotatedAST",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			highlights, err := tc.document.GetDocumentHighlights(context.Background(), protocol.Position{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(highlights) != 0 {
				t.Errorf("expected empty highlights, got %d", len(highlights))
			}
		})
	}
}

func TestGetFoldingRanges_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "nil AnnotationResult",
			document: newTestDocumentBuilder().Build(),
		},
		{
			name: "nil AnnotatedAST",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ranges, err := tc.document.GetFoldingRanges()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(ranges) != 0 {
				t.Errorf("expected empty ranges, got %d", len(ranges))
			}
		})
	}
}

func TestGetFoldingRanges_MultiLineElements(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 6)

	ast := newTestAnnotatedAST(node)
	document := newTestDocumentBuilder().
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: ast,
		}).
		Build()

	ranges, err := document.GetFoldingRanges()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ranges) == 0 {
		t.Fatal("expected at least one folding range for multi-line element")
	}
}

func TestPrepareRename_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "nil AnnotationResult",
			document: newTestDocumentBuilder().Build(),
		},
		{
			name: "nil AnnotatedAST",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.document.PrepareRename(context.Background(), protocol.Position{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != nil {
				t.Errorf("expected nil result, got %+v", result)
			}
		})
	}
}

func TestGetDocumentSymbols_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "nil AnnotationResult",
			document: newTestDocumentBuilder().Build(),
		},
		{
			name: "nil AnnotatedAST",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			symbols, err := tc.document.GetDocumentSymbols()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(symbols) != 0 {
				t.Errorf("expected empty symbols, got %d", len(symbols))
			}
		})
	}
}

func TestGetDocumentSymbols_WithStructuralBlocks(t *testing.T) {
	templateNode := newTestNode("template", 1, 1)
	scriptNode := newTestNode("script", 5, 1)
	styleNode := newTestNode("style", 10, 1)

	ast := newTestAnnotatedAST(templateNode)
	templateNode.Children = []*ast_domain.TemplateNode{scriptNode, styleNode}

	document := newTestDocumentBuilder().
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: ast,
		}).
		Build()

	symbols, err := document.GetDocumentSymbols()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(symbols) < 1 {
		t.Errorf("expected at least 1 symbol, got %d", len(symbols))
	}
}

func TestGetDocumentSymbols_WithIDElements(t *testing.T) {
	templateNode := newTestNode("template", 1, 1)
	divNode := newTestNode("div", 2, 3)
	addAttribute(divNode, "id", "main-content")

	templateNode.Children = []*ast_domain.TemplateNode{divNode}

	ast := newTestAnnotatedAST(templateNode)
	document := newTestDocumentBuilder().
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: ast,
		}).
		Build()

	symbols, err := document.GetDocumentSymbols()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(symbols) == 0 {
		t.Fatal("expected at least 1 symbol for div with id")
	}
}

func TestGetSymbolReferenceAtPosition(t *testing.T) {
	testCases := []struct {
		document  *document
		name      string
		position  protocol.Position
		expectNil bool
	}{
		{
			name: "returns nil when no expression found at position",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				Build(),
			position:  protocol.Position{Line: 10, Character: 5},
			expectNil: true,
		},
		{
			name: "returns nil when expression has no Go annotation",
			document: func() *document {
				node := newTestNodeMultiLine("div", 1, 1, 3, 10)
				node.DirIf = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name:             "visible",
						RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
						SourceLength:     7,
					},
				}
				return newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
			}(),
			position:  protocol.Position{Line: 0, Character: 2},
			expectNil: true,
		},
		{
			name: "returns nil when symbol has synthetic reference location",
			document: func() *document {
				node := newTestNodeMultiLine("div", 1, 1, 3, 10)
				node.DirIf = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name:             "visible",
						RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
						SourceLength:     7,
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "visible",
								ReferenceLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
					},
				}
				return newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
			}(),
			position:  protocol.Position{Line: 0, Character: 2},
			expectNil: true,
		},
		{
			name: "returns nil when expression has nil Go annotation symbol",
			document: func() *document {
				node := newTestNodeMultiLine("div", 1, 1, 3, 10)
				node.DirIf = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name:             "visible",
						RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
						SourceLength:     7,
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							Symbol: nil,
						},
					},
				}
				return newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
			}(),
			position:  protocol.Position{Line: 0, Character: 2},
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ref := tc.document.getSymbolReferenceAtPosition(context.Background(), tc.position)
			if tc.expectNil && ref != nil {
				t.Errorf("expected nil, got %+v", ref)
			}
			if !tc.expectNil && ref == nil {
				t.Error("expected non-nil symbol reference")
			}
		})
	}
}

func TestCollectHighlights(t *testing.T) {
	testCases := []struct {
		document           *document
		targetRef          *symbolReference
		expressionRangeMap map[ast_domain.Expression]protocol.Range
		name               string
		expectCount        int
	}{
		{
			name: "returns empty for AST with no matching symbols",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				Build(),
			targetRef: &symbolReference{
				line:       5,
				column:     3,
				sourcePath: "/test/main.go",
			},
			expressionRangeMap: map[ast_domain.Expression]protocol.Range{},
			expectCount:        0,
		},
		{
			name: "returns empty when no expressions match target",
			document: func() *document {
				node := newTestNodeMultiLine("div", 1, 1, 3, 10)
				node.DirIf = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name:             "visible",
						RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
						SourceLength:     7,
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "visible",
								ReferenceLocation: ast_domain.Location{
									Line:   10,
									Column: 5,
								},
							},
						},
					},
				}
				return newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
			}(),
			targetRef: &symbolReference{
				line:       99,
				column:     1,
				sourcePath: "",
			},
			expressionRangeMap: map[ast_domain.Expression]protocol.Range{},
			expectCount:        0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			highlights := tc.document.collectHighlights(tc.targetRef, tc.expressionRangeMap)
			if len(highlights) != tc.expectCount {
				t.Errorf("got %d highlights, want %d", len(highlights), tc.expectCount)
			}
		})
	}
}
