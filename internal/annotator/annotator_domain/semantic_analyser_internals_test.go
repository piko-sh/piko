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

package annotator_domain

import (
	"context"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestInternalsAnalyser_AnalyseInternalExpressions_Directives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setupNode          func() *ast_domain.TemplateNode
		name               string
		expectTextResolved bool
		expectHTMLResolved bool
	}{
		{
			name: "p-text directive with expression",
			setupNode: func() *ast_domain.TemplateNode {
				return &ast_domain.TemplateNode{
					DirText: &ast_domain.Directive{
						Type:       ast_domain.DirectiveText,
						Expression: &ast_domain.Identifier{Name: "title"},
						Location:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
					},
				}
			},
			expectTextResolved: true,
		},
		{
			name: "p-html directive with expression",
			setupNode: func() *ast_domain.TemplateNode {
				return &ast_domain.TemplateNode{
					DirHTML: &ast_domain.Directive{
						Type:       ast_domain.DirectiveHTML,
						Expression: &ast_domain.Identifier{Name: "content"},
						Location:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
					},
				}
			},
			expectHTMLResolved: true,
		},
		{
			name: "both directives present",
			setupNode: func() *ast_domain.TemplateNode {
				return &ast_domain.TemplateNode{
					DirText: &ast_domain.Directive{
						Type:       ast_domain.DirectiveText,
						Expression: &ast_domain.Identifier{Name: "text"},
						Location:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
					},
					DirHTML: &ast_domain.Directive{
						Type:       ast_domain.DirectiveHTML,
						Expression: &ast_domain.Identifier{Name: "html"},
						Location:   ast_domain.Location{Line: 2, Column: 1, Offset: 0},
					},
				}
			},
			expectTextResolved: true,
			expectHTMLResolved: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := createMockTypeResolver(t)
			ctx := NewRootAnalysisContext(
				new([]*ast_domain.Diagnostic),
				"test/package",
				"testpkg",
				"test.go",
				"test.piko",
			)

			ctx.Symbols.Define(Symbol{
				Name:     "title",
				TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil, PackageAlias: "", CanonicalPackagePath: "", IsSynthetic: false, IsExportedPackageSymbol: false, InitialPackagePath: "", InitialFilePath: ""},
			})
			ctx.Symbols.Define(Symbol{
				Name:     "content",
				TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil, PackageAlias: "", CanonicalPackagePath: "", IsSynthetic: false, IsExportedPackageSymbol: false, InitialPackagePath: "", InitialFilePath: ""},
			})
			ctx.Symbols.Define(Symbol{
				Name:     "text",
				TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil, PackageAlias: "", CanonicalPackagePath: "", IsSynthetic: false, IsExportedPackageSymbol: false, InitialPackagePath: "", InitialFilePath: ""},
			})
			ctx.Symbols.Define(Symbol{
				Name:     "html",
				TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil, PackageAlias: "", CanonicalPackagePath: "", IsSynthetic: false, IsExportedPackageSymbol: false, InitialPackagePath: "", InitialFilePath: ""},
			})
			analyser := newInternalsAnalyser(resolver)

			node := tt.setupNode()

			analyser.AnalyseInternalExpressions(context.Background(), node, ctx, nil)

			if tt.expectTextResolved && node.DirText != nil {
				if node.DirText.GoAnnotations == nil {
					t.Error("Expected DirText.GoAnnotations to be set, but it was nil")
				}
			}
			if tt.expectHTMLResolved && node.DirHTML != nil {
				if node.DirHTML.GoAnnotations == nil {
					t.Error("Expected DirHTML.GoAnnotations to be set, but it was nil")
				}
			}
		})
	}
}

func TestInternalsAnalyser_AnalyseInternalExpressions_RichText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		richTextParts    []ast_domain.TextPart
		expectedResolved []bool
	}{
		{
			name: "single literal part",
			richTextParts: []ast_domain.TextPart{
				{
					IsLiteral: true,
					Literal:   "Hello World",
					Location:  ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				},
			},
			expectedResolved: []bool{false},
		},
		{
			name: "single expression part",
			richTextParts: []ast_domain.TextPart{
				{
					IsLiteral:  false,
					Expression: &ast_domain.Identifier{Name: "username"},
					Location:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				},
			},
			expectedResolved: []bool{true},
		},
		{
			name: "mixed literal and expression parts",
			richTextParts: []ast_domain.TextPart{
				{
					IsLiteral: true,
					Literal:   "Hello ",
					Location:  ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				},
				{
					IsLiteral:  false,
					Expression: &ast_domain.Identifier{Name: "username"},
					Location:   ast_domain.Location{Line: 1, Column: 7, Offset: 0},
				},
				{
					IsLiteral: true,
					Literal:   ", you have ",
					Location:  ast_domain.Location{Line: 1, Column: 20, Offset: 0},
				},
				{
					IsLiteral:  false,
					Expression: &ast_domain.Identifier{Name: "count"},
					Location:   ast_domain.Location{Line: 1, Column: 32, Offset: 0},
				},
				{
					IsLiteral: true,
					Literal:   " messages",
					Location:  ast_domain.Location{Line: 1, Column: 40, Offset: 0},
				},
			},
			expectedResolved: []bool{false, true, false, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := createMockTypeResolver(t)
			ctx := NewRootAnalysisContext(
				new([]*ast_domain.Diagnostic),
				"test/package",
				"testpkg",
				"test.go",
				"test.piko",
			)

			ctx.Symbols.Define(Symbol{
				Name:     "username",
				TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil, PackageAlias: "", CanonicalPackagePath: "", IsSynthetic: false, IsExportedPackageSymbol: false, InitialPackagePath: "", InitialFilePath: ""},
			})
			ctx.Symbols.Define(Symbol{
				Name:     "count",
				TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil, PackageAlias: "", CanonicalPackagePath: "", IsSynthetic: false, IsExportedPackageSymbol: false, InitialPackagePath: "", InitialFilePath: ""},
			})

			analyser := newInternalsAnalyser(resolver)

			node := &ast_domain.TemplateNode{
				RichText: tt.richTextParts,
			}

			analyser.AnalyseInternalExpressions(context.Background(), node, ctx, nil)

			if len(node.RichText) != len(tt.expectedResolved) {
				t.Fatalf("Test setup error: expected %d parts, got %d", len(tt.expectedResolved), len(node.RichText))
			}

			for i, shouldBeResolved := range tt.expectedResolved {
				part := &node.RichText[i]
				if shouldBeResolved {
					if part.GoAnnotations == nil {
						t.Errorf("Part %d: Expected GoAnnotations to be set for expression, but it was nil", i)
					}
				} else {

					if part.GoAnnotations != nil {
						t.Errorf("Part %d: Expected GoAnnotations to be nil for literal, but it was set", i)
					}
				}
			}
		})
	}
}

func TestInternalsAnalyser_AnalyseInternalExpressions_NilNode(t *testing.T) {
	t.Parallel()

	resolver := createMockTypeResolver(t)
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(
		&diagnostics,
		"test/package",
		"testpkg",
		"test.go",
		"test.piko",
	)

	analyser := newInternalsAnalyser(resolver)

	analyser.AnalyseInternalExpressions(context.Background(), nil, ctx, nil)

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics for nil node, got %d", len(diagnostics))
	}
}

func createMockTypeResolver(t *testing.T) *TypeResolver {
	t.Helper()

	return &TypeResolver{

		inspector: nil,
	}
}
