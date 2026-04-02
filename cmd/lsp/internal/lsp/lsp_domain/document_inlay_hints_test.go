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
	goast "go/ast"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestExtractLoopVariables(t *testing.T) {
	testCases := []struct {
		name      string
		directive *ast_domain.Directive
		want      []string
	}{
		{
			name:      "nil directive",
			directive: nil,
			want:      nil,
		},
		{
			name:      "empty expression",
			directive: &ast_domain.Directive{RawExpression: ""},
			want:      nil,
		},
		{
			name:      "no assignment operator",
			directive: &ast_domain.Directive{RawExpression: "items"},
			want:      nil,
		},
		{
			name:      "single variable",
			directive: &ast_domain.Directive{RawExpression: "item := range items"},
			want:      []string{"item"},
		},
		{
			name:      "two variables",
			directive: &ast_domain.Directive{RawExpression: "i, item := range items"},
			want:      []string{"i", "item"},
		},
		{
			name:      "blank identifier ignored",
			directive: &ast_domain.Directive{RawExpression: "_, item := range items"},
			want:      []string{"item"},
		},
		{
			name:      "all blank identifiers",
			directive: &ast_domain.Directive{RawExpression: "_, _ := range items"},
			want:      nil,
		},
		{
			name:      "whitespace handling",
			directive: &ast_domain.Directive{RawExpression: "  i , item  := range items"},
			want:      []string{"i", "item"},
		},
		{
			name:      "empty vars part",
			directive: &ast_domain.Directive{RawExpression: " := range items"},
			want:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractLoopVariables(tc.directive)
			if len(got) != len(tc.want) {
				t.Fatalf("extractLoopVariables() = %v (len %d), want %v (len %d)", got, len(got), tc.want, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("extractLoopVariables()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestGetDirectiveHintPosition(t *testing.T) {
	testCases := []struct {
		name      string
		directive *ast_domain.Directive
		varName   string
		want      protocol.Position
	}{
		{
			name:      "nil directive",
			directive: nil,
			varName:   "item",
			want:      protocol.Position{},
		},
		{
			name: "zero line",
			directive: &ast_domain.Directive{
				RawExpression: "item := range items",
				Location:      ast_domain.Location{Line: 0, Column: 5},
			},
			varName: "item",
			want:    protocol.Position{},
		},
		{
			name: "variable not found",
			directive: &ast_domain.Directive{
				RawExpression: "item := range items",
				Location:      ast_domain.Location{Line: 3, Column: 10},
			},
			varName: "nonexistent",
			want:    protocol.Position{},
		},
		{
			name: "variable found at start",
			directive: &ast_domain.Directive{
				RawExpression: "item := range items",
				Location:      ast_domain.Location{Line: 3, Column: 10},
			},
			varName: "item",
			want: protocol.Position{
				Line:      2,
				Character: 9 + 0 + uint32(4),
			},
		},
		{
			name: "second variable in expression",
			directive: &ast_domain.Directive{
				RawExpression: "i, item := range items",
				Location:      ast_domain.Location{Line: 5, Column: 1},
			},
			varName: "item",
			want: protocol.Position{
				Line:      4,
				Character: 0 + uint32(3) + uint32(4),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getDirectiveHintPosition(tc.directive, tc.varName, "/test.pk")
			if got != tc.want {
				t.Errorf("getDirectiveHintPosition() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestFormatInlayTypeExpr(t *testing.T) {
	testCases := []struct {
		name     string
		typeInfo *ast_domain.ResolvedTypeInfo
		want     string
	}{
		{
			name:     "nil typeInfo",
			typeInfo: nil,
			want:     "",
		},
		{
			name:     "nil TypeExpr",
			typeInfo: &ast_domain.ResolvedTypeInfo{},
			want:     "",
		},
		{
			name: "simple ident type",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.Ident{Name: "string"},
			},
			want: "string",
		},
		{
			name: "pointer type",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StarExpr{
					X: &goast.Ident{Name: "User"},
				},
			},
			want: "*User",
		},
		{
			name: "array type",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{
					Elt: &goast.Ident{Name: "int"},
				},
			},
			want: "[]int",
		},
		{
			name: "selector type",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "time"},
					Sel: &goast.Ident{Name: "Duration"},
				},
			},
			want: "time.Duration",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatInlayTypeExpr(tc.typeInfo)
			if got != tc.want {
				t.Errorf("formatInlayTypeExpr() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInlayHintPositionInRange(t *testing.T) {
	textRange := protocol.Range{
		Start: protocol.Position{Line: 5, Character: 10},
		End:   protocol.Position{Line: 10, Character: 20},
	}

	testCases := []struct {
		name     string
		position protocol.Position
		want     bool
	}{
		{
			name:     "before range (earlier line)",
			position: protocol.Position{Line: 4, Character: 15},
			want:     false,
		},
		{
			name:     "before range (same line, earlier char)",
			position: protocol.Position{Line: 5, Character: 9},
			want:     false,
		},
		{
			name:     "at range start",
			position: protocol.Position{Line: 5, Character: 10},
			want:     true,
		},
		{
			name:     "in the middle",
			position: protocol.Position{Line: 7, Character: 15},
			want:     true,
		},
		{
			name:     "at range end",
			position: protocol.Position{Line: 10, Character: 20},
			want:     true,
		},
		{
			name:     "after range (same line, later char)",
			position: protocol.Position{Line: 10, Character: 21},
			want:     false,
		},
		{
			name:     "after range (later line)",
			position: protocol.Position{Line: 11, Character: 0},
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := inlayHintPositionInRange(tc.position, textRange)
			if got != tc.want {
				t.Errorf("inlayHintPositionInRange(%v, %v) = %v, want %v", tc.position, textRange, got, tc.want)
			}
		})
	}
}

func TestGetInlayHints_GuardClauses(t *testing.T) {
	fullRange := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 100, Character: 0},
	}

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
		{
			name: "nil AnalysisMap",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hints, err := tc.document.GetInlayHints(context.Background(), fullRange)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(hints) != 0 {
				t.Errorf("expected empty hints, got %d", len(hints))
			}
		})
	}
}

func TestCollectForLoopTypeHints(t *testing.T) {
	fullRange := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 100, Character: 100},
	}

	testCases := []struct {
		node    *ast_domain.TemplateNode
		ctx     *annotator_domain.AnalysisContext
		name    string
		wantLen int
	}{
		{
			name:    "nil node",
			node:    nil,
			ctx:     &annotator_domain.AnalysisContext{},
			wantLen: 0,
		},
		{
			name:    "node without DirFor",
			node:    newTestNode("div", 1, 1),
			ctx:     &annotator_domain.AnalysisContext{},
			wantLen: 0,
		},
		{
			name: "node with DirFor and matching symbol",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 3, 1)
				n.DirFor = &ast_domain.Directive{
					RawExpression: "item := range items",
					Location:      ast_domain.Location{Line: 3, Column: 10},
				}
				return n
			}(),
			ctx: func() *annotator_domain.AnalysisContext {
				st := annotator_domain.NewSymbolTable(nil)
				st.Define(annotator_domain.Symbol{
					Name: "item",
					TypeInfo: &ast_domain.ResolvedTypeInfo{
						TypeExpression: &goast.Ident{Name: "string"},
					},
				})
				return &annotator_domain.AnalysisContext{Symbols: st}
			}(),
			wantLen: 1,
		},
		{
			name: "symbol without type info produces no hint",
			node: func() *ast_domain.TemplateNode {
				n := newTestNode("div", 3, 1)
				n.DirFor = &ast_domain.Directive{
					RawExpression: "item := range items",
					Location:      ast_domain.Location{Line: 3, Column: 10},
				}
				return n
			}(),
			ctx: func() *annotator_domain.AnalysisContext {
				st := annotator_domain.NewSymbolTable(nil)
				st.Define(annotator_domain.Symbol{
					Name: "item",
				})
				return &annotator_domain.AnalysisContext{Symbols: st}
			}(),
			wantLen: 0,
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		Build()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hints := document.collectForLoopTypeHints(tc.node, tc.ctx, fullRange)
			if len(hints) != tc.wantLen {
				t.Errorf("collectForLoopTypeHints() returned %d hints, want %d", len(hints), tc.wantLen)
			}
		})
	}
}

func TestGetInlayHints_WithForLoop(t *testing.T) {
	node := newTestNode("div", 3, 1)
	node.DirFor = &ast_domain.Directive{
		RawExpression: "item := range items",
		Location:      ast_domain.Location{Line: 3, Column: 10},
	}

	st := annotator_domain.NewSymbolTable(nil)
	st.Define(annotator_domain.Symbol{
		Name: "item",
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.Ident{Name: "string"},
		},
	})

	analysisMap := map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
		node: {Symbols: st},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(analysisMap).
		Build()

	fullRange := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 100, Character: 100},
	}

	hints, err := document.GetInlayHints(context.Background(), fullRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(hints) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(hints))
	}

	if hints[0].Kind != InlayHintKindType {
		t.Errorf("hint kind = %v, want InlayHintKindType", hints[0].Kind)
	}

	if hints[0].Label != ": string" {
		t.Errorf("hint label = %q, want %q", hints[0].Label, ": string")
	}
}
