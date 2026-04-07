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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestNewPartialExpander(t *testing.T) {
	expander := NewPartialExpander(nil, nil, nil)

	require.NotNil(t, expander, "Expected NewPartialExpander to return non-nil expander")
	if expander.resolver != nil {
		t.Error("Expected resolver to be nil when passed nil")
	}
	if expander.cssProcessor != nil {
		t.Error("Expected cssProcessor to be nil when passed nil")
	}
	if expander.fsReader != nil {
		t.Error("Expected fsReader to be nil when passed nil")
	}
}

func TestFillSlotsInTree(t *testing.T) {
	tests := []struct {
		content        map[string][]*ast_domain.TemplateNode
		validateResult func(*testing.T, []*ast_domain.TemplateNode)
		name           string
		nodes          []*ast_domain.TemplateNode
		expectedCount  int
	}{
		{
			name: "no slots in tree",
			nodes: []*ast_domain.TemplateNode{
				{
					TagName:  "div",
					NodeType: ast_domain.NodeElement,
					Children: []*ast_domain.TemplateNode{
						{
							TagName:  "p",
							NodeType: ast_domain.NodeElement,
						},
					},
				},
			},
			content:       map[string][]*ast_domain.TemplateNode{},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*ast_domain.TemplateNode) {
				if len(result) != 1 {
					t.Errorf("Expected 1 node, got %d", len(result))
					return
				}
				if result[0].TagName != "div" {
					t.Errorf("Expected tagName 'div', got '%s'", result[0].TagName)
				}
			},
		},
		{
			name: "default slot with content",
			nodes: []*ast_domain.TemplateNode{
				{
					TagName:  "div",
					NodeType: ast_domain.NodeElement,
					Children: []*ast_domain.TemplateNode{
						{
							TagName:  "piko:slot",
							NodeType: ast_domain.NodeElement,
						},
					},
				},
			},
			content: map[string][]*ast_domain.TemplateNode{
				"": {
					{
						TagName:  "span",
						NodeType: ast_domain.NodeElement,
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*ast_domain.TemplateNode) {
				if len(result) != 1 {
					t.Errorf("Expected 1 root node, got %d", len(result))
					return
				}
				if len(result[0].Children) != 1 {
					t.Errorf("Expected 1 child (the filled slot content), got %d", len(result[0].Children))
					return
				}
				if result[0].Children[0].TagName != "span" {
					t.Errorf("Expected filled slot to contain 'span', got '%s'", result[0].Children[0].TagName)
				}
			},
		},
		{
			name: "named slot with content",
			nodes: []*ast_domain.TemplateNode{
				{
					TagName:  "div",
					NodeType: ast_domain.NodeElement,
					Children: []*ast_domain.TemplateNode{
						{
							TagName:  "piko:slot",
							NodeType: ast_domain.NodeElement,
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "name", Value: "header"},
							},
						},
					},
				},
			},
			content: map[string][]*ast_domain.TemplateNode{
				"header": {
					{
						TagName:  "h1",
						NodeType: ast_domain.NodeElement,
					},
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*ast_domain.TemplateNode) {
				if len(result) != 1 {
					t.Errorf("Expected 1 root node, got %d", len(result))
					return
				}
				if len(result[0].Children) != 1 {
					t.Errorf("Expected 1 child (the filled slot content), got %d", len(result[0].Children))
					return
				}
				if result[0].Children[0].TagName != "h1" {
					t.Errorf("Expected filled slot to contain 'h1', got '%s'", result[0].Children[0].TagName)
				}
			},
		},
		{
			name: "slot with fallback content when no content provided",
			nodes: []*ast_domain.TemplateNode{
				{
					TagName:  "div",
					NodeType: ast_domain.NodeElement,
					Children: []*ast_domain.TemplateNode{
						{
							TagName:  "piko:slot",
							NodeType: ast_domain.NodeElement,
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "name", Value: "header"},
							},
							Children: []*ast_domain.TemplateNode{
								{
									TagName:  "h2",
									NodeType: ast_domain.NodeElement,
								},
							},
						},
					},
				},
			},
			content:       map[string][]*ast_domain.TemplateNode{},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []*ast_domain.TemplateNode) {
				if len(result) != 1 {
					t.Errorf("Expected 1 root node, got %d", len(result))
					return
				}
				if len(result[0].Children) != 1 {
					t.Errorf("Expected 1 child (the fallback content), got %d", len(result[0].Children))
					return
				}
				if result[0].Children[0].TagName != "h2" {
					t.Errorf("Expected fallback content 'h2', got '%s'", result[0].Children[0].TagName)
				}
			},
		},
		{
			name: "multiple nodes with slots",
			nodes: []*ast_domain.TemplateNode{
				{
					TagName:  "header",
					NodeType: ast_domain.NodeElement,
					Children: []*ast_domain.TemplateNode{
						{
							TagName:  "piko:slot",
							NodeType: ast_domain.NodeElement,
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "name", Value: "title"},
							},
						},
					},
				},
				{
					TagName:  "main",
					NodeType: ast_domain.NodeElement,
					Children: []*ast_domain.TemplateNode{
						{
							TagName:  "piko:slot",
							NodeType: ast_domain.NodeElement,
						},
					},
				},
			},
			content: map[string][]*ast_domain.TemplateNode{
				"title": {
					{TagName: "h1", NodeType: ast_domain.NodeElement},
				},
				"": {
					{TagName: "article", NodeType: ast_domain.NodeElement},
				},
			},
			expectedCount: 2,
			validateResult: func(t *testing.T, result []*ast_domain.TemplateNode) {
				if len(result) != 2 {
					t.Errorf("Expected 2 root nodes, got %d", len(result))
					return
				}

				if len(result[0].Children) != 1 || result[0].Children[0].TagName != "h1" {
					t.Error("Expected first node's slot to be filled with h1")
				}

				if len(result[1].Children) != 1 || result[1].Children[0].TagName != "article" {
					t.Error("Expected second node's slot to be filled with article")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fillSlotsInTree(tt.nodes, tt.content)

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d nodes, got %d", tt.expectedCount, len(result))
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestFinaliseAST(t *testing.T) {
	tests := []struct {
		mainComponent *annotator_dto.ParsedComponent
		validateAST   func(*testing.T, *ast_domain.TemplateAST)
		name          string
		rootNodes     []*ast_domain.TemplateNode
	}{
		{
			name: "basic finalisation",
			rootNodes: []*ast_domain.TemplateNode{
				{TagName: "div", NodeType: ast_domain.NodeElement},
			},
			mainComponent: &annotator_dto.ParsedComponent{
				SourcePath: "/test/path.piko",
				Template: &ast_domain.TemplateAST{
					Diagnostics: []*ast_domain.Diagnostic{},
				},
			},
			validateAST: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.NotNil(t, ast, "Expected non-nil AST")
				if len(ast.RootNodes) != 1 {
					t.Errorf("Expected 1 root node, got %d", len(ast.RootNodes))
				}
				if ast.SourcePath == nil || *ast.SourcePath != "/test/path.piko" {
					t.Error("Expected SourcePath to be set from mainComponent")
				}
			},
		},
		{
			name:      "empty root nodes",
			rootNodes: []*ast_domain.TemplateNode{},
			mainComponent: &annotator_dto.ParsedComponent{
				SourcePath: "/test/empty.piko",
				Template: &ast_domain.TemplateAST{
					Diagnostics: []*ast_domain.Diagnostic{},
				},
			},
			validateAST: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.NotNil(t, ast, "Expected non-nil AST")
				if len(ast.RootNodes) != 0 {
					t.Errorf("Expected 0 root nodes, got %d", len(ast.RootNodes))
				}
			},
		},
		{
			name: "multiple root nodes",
			rootNodes: []*ast_domain.TemplateNode{
				{TagName: "header", NodeType: ast_domain.NodeElement},
				{TagName: "main", NodeType: ast_domain.NodeElement},
				{TagName: "footer", NodeType: ast_domain.NodeElement},
			},
			mainComponent: &annotator_dto.ParsedComponent{
				SourcePath: "/test/multi.piko",
				Template: &ast_domain.TemplateAST{
					Diagnostics: []*ast_domain.Diagnostic{},
				},
			},
			validateAST: func(t *testing.T, ast *ast_domain.TemplateAST) {
				require.NotNil(t, ast, "Expected non-nil AST")
				if len(ast.RootNodes) != 3 {
					t.Errorf("Expected 3 root nodes, got %d", len(ast.RootNodes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := finaliseAST(context.Background(), tt.rootNodes, tt.mainComponent)

			if tt.validateAST != nil {
				tt.validateAST(t, result)
			}
		})
	}
}

func TestStampNodesWithPackage(t *testing.T) {
	tests := []struct {
		validate     func(*testing.T, []*ast_domain.TemplateNode)
		name         string
		packageAlias string
		sourcePath   string
		nodes        []*ast_domain.TemplateNode
	}{
		{
			name: "single node stamping",
			nodes: []*ast_domain.TemplateNode{
				{TagName: "div", NodeType: ast_domain.NodeElement},
			},
			packageAlias: "test_pkg",
			sourcePath:   "/test/path.piko",
			validate: func(t *testing.T, nodes []*ast_domain.TemplateNode) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				node := nodes[0]
				if node.GoAnnotations == nil {
					t.Fatal("Expected GoAnnotations to be set")
				}
				if node.GoAnnotations.OriginalPackageAlias == nil || *node.GoAnnotations.OriginalPackageAlias != "test_pkg" {
					t.Error("Expected OriginalPackageAlias to be set to 'test_pkg'")
				}
				if node.GoAnnotations.OriginalSourcePath == nil || *node.GoAnnotations.OriginalSourcePath != "/test/path.piko" {
					t.Error("Expected OriginalSourcePath to be set")
				}
			},
		},
		{
			name: "nested nodes stamping",
			nodes: []*ast_domain.TemplateNode{
				{
					TagName:  "div",
					NodeType: ast_domain.NodeElement,
					Children: []*ast_domain.TemplateNode{
						{
							TagName:  "p",
							NodeType: ast_domain.NodeElement,
							Children: []*ast_domain.TemplateNode{
								{TagName: "span", NodeType: ast_domain.NodeElement},
							},
						},
					},
				},
			},
			packageAlias: "nested_pkg",
			sourcePath:   "/nested/path.piko",
			validate: func(t *testing.T, nodes []*ast_domain.TemplateNode) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 root node, got %d", len(nodes))
				}

				if nodes[0].GoAnnotations == nil || nodes[0].GoAnnotations.OriginalPackageAlias == nil {
					t.Error("Expected root node to be stamped")
				}

				if len(nodes[0].Children) == 0 {
					t.Fatal("Expected root to have children")
				}
				if nodes[0].Children[0].GoAnnotations == nil || nodes[0].Children[0].GoAnnotations.OriginalPackageAlias == nil {
					t.Error("Expected first child to be stamped")
				}

				if len(nodes[0].Children[0].Children) == 0 {
					t.Fatal("Expected first child to have children")
				}
				if nodes[0].Children[0].Children[0].GoAnnotations == nil {
					t.Error("Expected grandchild to be stamped")
				}
			},
		},
		{
			name:         "empty nodes list",
			nodes:        []*ast_domain.TemplateNode{},
			packageAlias: "empty_pkg",
			sourcePath:   "/empty/path.piko",
			validate: func(t *testing.T, nodes []*ast_domain.TemplateNode) {
				if len(nodes) != 0 {
					t.Errorf("Expected 0 nodes, got %d", len(nodes))
				}
			},
		},
		{
			name: "preserves existing annotations",
			nodes: []*ast_domain.TemplateNode{
				{
					TagName:  "div",
					NodeType: ast_domain.NodeElement,
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						NeedsCSRF: true,
					},
				},
			},
			packageAlias: "preserve_pkg",
			sourcePath:   "/preserve/path.piko",
			validate: func(t *testing.T, nodes []*ast_domain.TemplateNode) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				node := nodes[0]
				if node.GoAnnotations == nil {
					t.Fatal("Expected GoAnnotations to be set")
				}
				if !node.GoAnnotations.NeedsCSRF {
					t.Error("Expected existing NeedsCSRF to be preserved")
				}
				if node.GoAnnotations.OriginalPackageAlias == nil {
					t.Error("Expected OriginalPackageAlias to be added")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stampNodesWithPackage(tt.nodes, tt.packageAlias, tt.sourcePath)

			if tt.validate != nil {
				tt.validate(t, tt.nodes)
			}
		})
	}
}

func TestCountErrors(t *testing.T) {
	tests := []struct {
		name          string
		diagnostics   []*ast_domain.Diagnostic
		expectedCount int
	}{
		{
			name:          "no diagnostics",
			diagnostics:   []*ast_domain.Diagnostic{},
			expectedCount: 0,
		},
		{
			name: "only warnings",
			diagnostics: []*ast_domain.Diagnostic{
				{Severity: ast_domain.Warning, Message: "warning 1"},
				{Severity: ast_domain.Warning, Message: "warning 2"},
			},
			expectedCount: 0,
		},
		{
			name: "only errors",
			diagnostics: []*ast_domain.Diagnostic{
				{Severity: ast_domain.Error, Message: "error 1"},
				{Severity: ast_domain.Error, Message: "error 2"},
				{Severity: ast_domain.Error, Message: "error 3"},
			},
			expectedCount: 3,
		},
		{
			name: "mixed severities",
			diagnostics: []*ast_domain.Diagnostic{
				{Severity: ast_domain.Info, Message: "info"},
				{Severity: ast_domain.Warning, Message: "warning"},
				{Severity: ast_domain.Error, Message: "error 1"},
				{Severity: ast_domain.Debug, Message: "debug"},
				{Severity: ast_domain.Error, Message: "error 2"},
			},
			expectedCount: 2,
		},
		{
			name:          "nil slice",
			diagnostics:   nil,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countErrors(tt.diagnostics)

			if result != tt.expectedCount {
				t.Errorf("Expected %d errors, got %d", tt.expectedCount, result)
			}
		})
	}
}

func TestCreateEmptyExpansionResult(t *testing.T) {
	t.Parallel()

	result := createEmptyExpansionResult()

	require.NotNil(t, result)
	require.NotNil(t, result.FlattenedAST)
	assert.Nil(t, result.FlattenedAST.SourcePath)
	assert.Nil(t, result.FlattenedAST.ExpiresAtUnixNano)
	assert.Nil(t, result.FlattenedAST.Metadata)
	assert.Nil(t, result.FlattenedAST.RootNodes)
	assert.Nil(t, result.FlattenedAST.Diagnostics)
	assert.Equal(t, int64(0), result.FlattenedAST.SourceSize)
	assert.False(t, result.FlattenedAST.Tidied)
	assert.Equal(t, "", result.CombinedCSS)
	assert.Nil(t, result.PotentialInvocations)
}

func TestCheckForCriticalErrors(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no diagnostics have errors", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: []*ast_domain.Diagnostic{
				{Severity: ast_domain.Warning, Message: "just a warning"},
			},
		}

		err := checkForCriticalErrors(context.Background(), ec)

		assert.NoError(t, err)
	})

	t.Run("returns nil when no errors at all", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}

		err := checkForCriticalErrors(context.Background(), ec)

		assert.NoError(t, err)
	})

	t.Run("returns CircularDependencyError when circular dependency is found", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: []*ast_domain.Diagnostic{
				{
					Severity: ast_domain.Error,
					Message:  "Circular dependency detected: a -> b -> a",
				},
			},
			expansionPath: []string{"a", "b"},
		}

		err := checkForCriticalErrors(context.Background(), ec)

		require.Error(t, err)
		var circErr *CircularDependencyError
		require.ErrorAs(t, err, &circErr)
		assert.Equal(t, []string{"a", "b"}, circErr.Path)
	})

	t.Run("returns nil for non-circular errors", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: []*ast_domain.Diagnostic{
				{
					Severity: ast_domain.Error,
					Message:  "Failed to load partial 'card'",
				},
			},
		}

		err := checkForCriticalErrors(context.Background(), ec)

		assert.NoError(t, err)
	})
}

func TestStampDynamicAttributesWithSourcePath(t *testing.T) {
	t.Parallel()

	t.Run("stamps source path on attributes without one", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", RawExpression: "state.title"},
				{Name: "visible", RawExpression: "state.visible"},
			},
		}

		stampDynamicAttributesWithSourcePath(node, "/my/source.pk")

		for i := range node.DynamicAttributes {
			attr := &node.DynamicAttributes[i]
			require.NotNil(t, attr.GoAnnotations)
			require.NotNil(t, attr.GoAnnotations.OriginalSourcePath)
			assert.Equal(t, "/my/source.pk", *attr.GoAnnotations.OriginalSourcePath)
		}
	})

	t.Run("does not overwrite existing source path", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:          "title",
					RawExpression: "state.title",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						OriginalSourcePath: new("/existing/path.pk"),
					},
				},
			},
		}

		stampDynamicAttributesWithSourcePath(node, "/new/source.pk")

		assert.Equal(t, "/existing/path.pk", *node.DynamicAttributes[0].GoAnnotations.OriginalSourcePath)
	})

	t.Run("handles empty dynamic attributes", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{},
		}

		assert.NotPanics(t, func() {
			stampDynamicAttributesWithSourcePath(node, "/source.pk")
		})
	})
}

func TestStampDirectivesWithSourcePath(t *testing.T) {
	t.Parallel()

	t.Run("stamps all directive types", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			DirIf:       &ast_domain.Directive{RawExpression: "state.show"},
			DirElseIf:   &ast_domain.Directive{RawExpression: "state.other"},
			DirElse:     &ast_domain.Directive{RawExpression: ""},
			DirFor:      &ast_domain.Directive{RawExpression: "item in items"},
			DirShow:     &ast_domain.Directive{RawExpression: "state.visible"},
			DirModel:    &ast_domain.Directive{RawExpression: "state.value"},
			DirRef:      &ast_domain.Directive{RawExpression: "myRef"},
			DirClass:    &ast_domain.Directive{RawExpression: "state.classes"},
			DirStyle:    &ast_domain.Directive{RawExpression: "state.styles"},
			DirText:     &ast_domain.Directive{RawExpression: "state.text"},
			DirHTML:     &ast_domain.Directive{RawExpression: "state.html"},
			DirKey:      &ast_domain.Directive{RawExpression: "item.id"},
			DirContext:  &ast_domain.Directive{RawExpression: "ctx"},
			DirScaffold: &ast_domain.Directive{RawExpression: "scaffold"},
		}

		stampDirectivesWithSourcePath(node, "/test/source.pk")

		directives := []*ast_domain.Directive{
			node.DirIf, node.DirElseIf, node.DirElse, node.DirFor,
			node.DirShow, node.DirModel, node.DirRef, node.DirClass,
			node.DirStyle, node.DirText, node.DirHTML, node.DirKey,
			node.DirContext, node.DirScaffold,
		}
		for _, d := range directives {
			require.NotNil(t, d.GoAnnotations, "GoAnnotations should be set")
			require.NotNil(t, d.GoAnnotations.OriginalSourcePath, "OriginalSourcePath should be set")
			assert.Equal(t, "/test/source.pk", *d.GoAnnotations.OriginalSourcePath)
		}
	})

	t.Run("does not overwrite existing source path on directives", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			DirIf: &ast_domain.Directive{
				RawExpression: "state.show",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("/existing.pk"),
				},
			},
		}

		stampDirectivesWithSourcePath(node, "/new/source.pk")

		assert.Equal(t, "/existing.pk", *node.DirIf.GoAnnotations.OriginalSourcePath)
	})

	t.Run("stamps nil directives safely", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{}

		assert.NotPanics(t, func() {
			stampDirectivesWithSourcePath(node, "/source.pk")
		})
	})

	t.Run("stamps bind directives", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"value": {RawExpression: "state.val"},
			},
		}

		stampDirectivesWithSourcePath(node, "/source.pk")

		require.NotNil(t, node.Binds["value"].GoAnnotations)
		require.NotNil(t, node.Binds["value"].GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "/source.pk", *node.Binds["value"].GoAnnotations.OriginalSourcePath)
	})

	t.Run("stamps on-event directives", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"click": {
					{RawExpression: "handleClick"},
				},
			},
		}

		stampDirectivesWithSourcePath(node, "/source.pk")

		require.NotNil(t, node.OnEvents["click"][0].GoAnnotations)
		require.NotNil(t, node.OnEvents["click"][0].GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "/source.pk", *node.OnEvents["click"][0].GoAnnotations.OriginalSourcePath)
	})

	t.Run("stamps custom-event directives", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			CustomEvents: map[string][]ast_domain.Directive{
				"custom": {
					{RawExpression: "handleCustom"},
				},
			},
		}

		stampDirectivesWithSourcePath(node, "/source.pk")

		require.NotNil(t, node.CustomEvents["custom"][0].GoAnnotations)
		require.NotNil(t, node.CustomEvents["custom"][0].GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "/source.pk", *node.CustomEvents["custom"][0].GoAnnotations.OriginalSourcePath)
	})
}

func TestStampRichTextWithSourcePath(t *testing.T) {
	t.Parallel()

	t.Run("stamps source path on rich text parts", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			RichText: []ast_domain.TextPart{
				{Literal: "hello ", IsLiteral: true},
				{Literal: "", IsLiteral: false, RawExpression: "state.name"},
			},
		}

		stampRichTextWithSourcePath(node, "/source.pk")

		for i := range node.RichText {
			require.NotNil(t, node.RichText[i].GoAnnotations)
			require.NotNil(t, node.RichText[i].GoAnnotations.OriginalSourcePath)
			assert.Equal(t, "/source.pk", *node.RichText[i].GoAnnotations.OriginalSourcePath)
		}
	})

	t.Run("does not overwrite existing source path", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			RichText: []ast_domain.TextPart{
				{
					Literal:   "hello",
					IsLiteral: true,
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						OriginalSourcePath: new("/existing.pk"),
					},
				},
			},
		}

		stampRichTextWithSourcePath(node, "/new.pk")

		assert.Equal(t, "/existing.pk", *node.RichText[0].GoAnnotations.OriginalSourcePath)
	})

	t.Run("handles empty rich text", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			RichText: []ast_domain.TextPart{},
		}

		assert.NotPanics(t, func() {
			stampRichTextWithSourcePath(node, "/source.pk")
		})
	})
}

func TestStampNodesWithPackageDirectivesAndDynAttrs(t *testing.T) {
	t.Parallel()

	t.Run("stamps dynamic attributes on nested nodes", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "title", RawExpression: "state.title"},
				},
				Children: []*ast_domain.TemplateNode{
					{
						TagName:  "span",
						NodeType: ast_domain.NodeElement,
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{Name: "visible", RawExpression: "state.visible"},
						},
					},
				},
			},
		}

		stampNodesWithPackage(nodes, "test_pkg", "/test/source.pk")

		require.NotNil(t, nodes[0].DynamicAttributes[0].GoAnnotations)
		assert.Equal(t, "/test/source.pk", *nodes[0].DynamicAttributes[0].GoAnnotations.OriginalSourcePath)
		require.NotNil(t, nodes[0].Children[0].DynamicAttributes[0].GoAnnotations)
		assert.Equal(t, "/test/source.pk", *nodes[0].Children[0].DynamicAttributes[0].GoAnnotations.OriginalSourcePath)
	})

	t.Run("stamps directives on nodes", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				DirIf:    &ast_domain.Directive{RawExpression: "state.show"},
			},
		}

		stampNodesWithPackage(nodes, "test_pkg", "/test/source.pk")

		require.NotNil(t, nodes[0].DirIf.GoAnnotations)
		assert.Equal(t, "/test/source.pk", *nodes[0].DirIf.GoAnnotations.OriginalSourcePath)
	})

	t.Run("stamps rich text on nodes", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				TagName:  "p",
				NodeType: ast_domain.NodeElement,
				RichText: []ast_domain.TextPart{
					{Literal: "Hello ", IsLiteral: true},
				},
			},
		}

		stampNodesWithPackage(nodes, "test_pkg", "/test/source.pk")

		require.NotNil(t, nodes[0].RichText[0].GoAnnotations)
		assert.Equal(t, "/test/source.pk", *nodes[0].RichText[0].GoAnnotations.OriginalSourcePath)
	})
}

func TestNewPartialExpanderWithArgs(t *testing.T) {
	t.Parallel()

	t.Run("stores resolver, cssProcessor, and fsReader", func(t *testing.T) {
		t.Parallel()

		cssProc := &CSSProcessor{}
		expander := NewPartialExpander(nil, cssProc, nil)

		require.NotNil(t, expander)
		assert.Nil(t, expander.resolver)
		assert.Same(t, cssProc, expander.cssProcessor)
		assert.Nil(t, expander.fsReader)
	})
}

func TestFillSlotsInTreeNestedSlots(t *testing.T) {
	t.Parallel()

	t.Run("recursively fills nested slot content", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				Children: []*ast_domain.TemplateNode{
					{
						TagName:  "section",
						NodeType: ast_domain.NodeElement,
						Children: []*ast_domain.TemplateNode{
							{
								TagName:  "piko:slot",
								NodeType: ast_domain.NodeElement,
								Attributes: []ast_domain.HTMLAttribute{
									{Name: "name", Value: "deep"},
								},
							},
						},
					},
				},
			},
		}

		content := map[string][]*ast_domain.TemplateNode{
			"deep": {
				{TagName: "article", NodeType: ast_domain.NodeElement},
			},
		}

		result := fillSlotsInTree(nodes, content)

		require.Len(t, result, 1)
		require.Len(t, result[0].Children, 1)
		section := result[0].Children[0]
		require.Len(t, section.Children, 1)
		assert.Equal(t, "article", section.Children[0].TagName)
	})

	t.Run("empty nodes returns empty result", func(t *testing.T) {
		t.Parallel()

		result := fillSlotsInTree(nil, map[string][]*ast_domain.TemplateNode{})

		assert.Empty(t, result)
	})
}

func TestFinaliseASTWithDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("preserves template diagnostics", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*ast_domain.Diagnostic{
			{Severity: ast_domain.Warning, Message: "template warning"},
		}
		mainComponent := &annotator_dto.ParsedComponent{
			SourcePath: "/test/with-diagnostics.pk",
			Template: &ast_domain.TemplateAST{
				Diagnostics: diagnostics,
			},
		}

		result := finaliseAST(context.Background(), []*ast_domain.TemplateNode{}, mainComponent)

		require.NotNil(t, result)
		assert.Equal(t, diagnostics, result.Diagnostics)
	})
}

func TestPartialExpanderCreateEmptyExpansionResult(t *testing.T) {
	t.Parallel()

	t.Run("returns empty result with nil source path", func(t *testing.T) {
		t.Parallel()

		result := createEmptyExpansionResult()

		require.NotNil(t, result)
		require.NotNil(t, result.FlattenedAST)
		assert.Nil(t, result.FlattenedAST.SourcePath)
		assert.Nil(t, result.FlattenedAST.RootNodes)
		assert.Empty(t, result.CombinedCSS)
		assert.Nil(t, result.PotentialInvocations)
	})
}
