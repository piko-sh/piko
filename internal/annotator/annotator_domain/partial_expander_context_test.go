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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestNewExpansionContext(t *testing.T) {
	tests := []struct {
		setupGraph           func() *annotator_dto.ComponentGraph
		validateContext      func(*testing.T, *expansionContext, *annotator_dto.ParsedComponent, *ast_domain.TemplateAST)
		name                 string
		entryPointHashedName string
		expectError          bool
	}{
		{
			name: "valid entry point with template",
			setupGraph: func() *annotator_dto.ComponentGraph {
				return &annotator_dto.ComponentGraph{
					Components: map[string]*annotator_dto.ParsedComponent{
						"main_hash": {
							SourcePath: "/test/main.piko",
							Template: &ast_domain.TemplateAST{
								RootNodes: []*ast_domain.TemplateNode{
									{TagName: "div", NodeType: ast_domain.NodeElement},
								},
							},
						},
					},
				}
			},
			entryPointHashedName: "main_hash",
			expectError:          false,
			validateContext: func(t *testing.T, ec *expansionContext, comp *annotator_dto.ParsedComponent, ast *ast_domain.TemplateAST) {
				if ec == nil {
					t.Fatal("Expected non-nil expansion context")
				}
				if comp == nil {
					t.Fatal("Expected non-nil component")
				}
				if ast == nil {
					t.Fatal("Expected non-nil AST")
				}
				if len(ast.RootNodes) != 1 {
					t.Errorf("Expected 1 root node in cloned AST, got %d", len(ast.RootNodes))
				}
				if len(ec.expansionPath) != 1 {
					t.Errorf("Expected 1 entry in expansion path, got %d", len(ec.expansionPath))
				}
				if ec.scopedCSSBlocks == nil {
					t.Error("Expected scopedCSSBlocks map to be initialised")
				}
				if ec.globalCSSBlocks == nil {
					t.Error("Expected globalCSSBlocks map to be initialised")
				}
				if ec.uniqueInvocations == nil {
					t.Error("Expected uniqueInvocations map to be initialised")
				}
			},
		},
		{
			name: "valid entry point without template",
			setupGraph: func() *annotator_dto.ComponentGraph {
				return &annotator_dto.ComponentGraph{
					Components: map[string]*annotator_dto.ParsedComponent{
						"no_template_hash": {
							SourcePath: "/test/no_template.piko",
							Template:   nil,
						},
					},
				}
			},
			entryPointHashedName: "no_template_hash",
			expectError:          false,
			validateContext: func(t *testing.T, ec *expansionContext, comp *annotator_dto.ParsedComponent, ast *ast_domain.TemplateAST) {
				if ec != nil {
					t.Error("Expected nil expansion context when component has no template")
				}
				if comp == nil {
					t.Fatal("Expected non-nil component")
				}
				if ast != nil {
					t.Error("Expected nil AST when component has no template")
				}
			},
		},
		{
			name: "entry point not found in graph",
			setupGraph: func() *annotator_dto.ComponentGraph {
				return &annotator_dto.ComponentGraph{
					Components: map[string]*annotator_dto.ParsedComponent{},
				}
			},
			entryPointHashedName: "nonexistent_hash",
			expectError:          true,
			validateContext: func(t *testing.T, ec *expansionContext, comp *annotator_dto.ParsedComponent, ast *ast_domain.TemplateAST) {
				if ec != nil {
					t.Error("Expected nil context when entry point not found")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := tt.setupGraph()
			expander := NewPartialExpander(nil, nil, nil)

			ec, comp, ast, err := newExpansionContext(expander, graph, tt.entryPointHashedName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}

			if tt.validateContext != nil {
				tt.validateContext(t, ec, comp, ast)
			}
		})
	}
}

func TestGetPartialImportAliasFromIsAttr(t *testing.T) {
	tests := []struct {
		name          string
		node          *ast_domain.TemplateNode
		expectedAlias string
		expectedFlag  bool
	}{
		{
			name: "piko:partial with is attribute",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:partial",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "card"},
				},
			},
			expectedAlias: "card",
			expectedFlag:  true,
		},
		{
			name: "piko:partial without is attribute",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:partial",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "container"},
				},
			},
			expectedAlias: "",
			expectedFlag:  false,
		},
		{
			name: "piko:partial with empty is attribute",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:partial",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: ""},
				},
			},
			expectedAlias: "",
			expectedFlag:  false,
		},
		{
			name: "piko:partial with multiple attributes including is",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:partial",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "wrapper"},
					{Name: "is", Value: "modal"},
					{Name: "id", Value: "main"},
				},
			},
			expectedAlias: "modal",
			expectedFlag:  true,
		},
		{
			name: "is attribute on non-piko:partial element is ignored",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "card"},
				},
			},
			expectedAlias: "",
			expectedFlag:  false,
		},
		{
			name:          "nil node",
			node:          nil,
			expectedAlias: "",
			expectedFlag:  false,
		},
		{
			name: "node with no attributes",
			node: &ast_domain.TemplateNode{
				TagName:    "div",
				NodeType:   ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{},
			},
			expectedAlias: "",
			expectedFlag:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alias, isPartial := getPartialImportAliasFromIsAttr(tt.node)

			if alias != tt.expectedAlias {
				t.Errorf("Expected alias '%s', got '%s'", tt.expectedAlias, alias)
			}
			if isPartial != tt.expectedFlag {
				t.Errorf("Expected isPartial %v, got %v", tt.expectedFlag, isPartial)
			}
		})
	}
}

func TestValidatePikoPartialElement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		node         *ast_domain.TemplateNode
		name         string
		errorMessage string
		expectError  bool
	}{
		{
			name: "valid piko:partial with is",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:partial",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "card"},
				},
			},
			expectError: false,
		},
		{
			name: "piko:partial missing is attribute",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:partial",
				NodeType: ast_domain.NodeElement,
			},
			expectError:  true,
			errorMessage: "requires an 'is' attribute",
		},
		{
			name: "piko:partial with empty is attribute",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:partial",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: ""},
				},
			},
			expectError:  true,
			errorMessage: "requires an 'is' attribute",
		},
		{
			name: "dynamic :is on piko:partial",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:partial",
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "is", RawExpression: "state.partial"},
				},
			},
			expectError:  true,
			errorMessage: "cannot be dynamic",
		},
		{
			name: "is on non-piko:partial element is valid (just a normal attribute)",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "card"},
				},
			},
			expectError: false,
		},
		{
			name:        "nil node is valid",
			node:        nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diagnostic := validatePikoPartialElement(tt.node, "test.pk")

			if tt.expectError {
				if diagnostic == nil {
					t.Error("Expected error diagnostic but got nil")
				} else if !strings.Contains(diagnostic.Message, tt.errorMessage) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMessage, diagnostic.Message)
				}
			} else if diagnostic != nil {
				t.Errorf("Expected no error, got: %s", diagnostic.Message)
			}
		})
	}
}

func TestCalculatePotentialInvocationKey(t *testing.T) {
	tests := []struct {
		reqOverrides map[string]ast_domain.PropValue
		passedProps  map[string]ast_domain.PropValue
		validateKey  func(*testing.T, string)
		name         string
		partialAlias string
	}{
		{
			name:         "simple alias with no props",
			partialAlias: "card",
			reqOverrides: map[string]ast_domain.PropValue{},
			passedProps:  map[string]ast_domain.PropValue{},
			validateKey: func(t *testing.T, key string) {
				if key == "" {
					t.Error("Expected non-empty key")
				}
				if key != calculatePotentialInvocationKey("card", map[string]ast_domain.PropValue{}, map[string]ast_domain.PropValue{}) {
					t.Error("Expected consistent key generation")
				}
			},
		},
		{
			name:         "alias with request overrides",
			partialAlias: "modal",
			reqOverrides: map[string]ast_domain.PropValue{
				"title": {
					Expression:  &ast_domain.StringLiteral{Value: "Hello"},
					GoFieldName: "Title",
				},
			},
			passedProps: map[string]ast_domain.PropValue{},
			validateKey: func(t *testing.T, key string) {
				if key == "" {
					t.Error("Expected non-empty key")
				}

				simpleKey := calculatePotentialInvocationKey("modal", map[string]ast_domain.PropValue{}, map[string]ast_domain.PropValue{})
				if key == simpleKey {
					t.Error("Expected different key when props are provided")
				}
			},
		},
		{
			name:         "alias with passed props",
			partialAlias: "button",
			reqOverrides: map[string]ast_domain.PropValue{},
			passedProps: map[string]ast_domain.PropValue{
				"label": {
					Expression:  &ast_domain.StringLiteral{Value: "Click me"},
					GoFieldName: "Label",
				},
			},
			validateKey: func(t *testing.T, key string) {
				if key == "" {
					t.Error("Expected non-empty key")
				}
			},
		},
		{
			name:         "same props should produce same key",
			partialAlias: "card",
			reqOverrides: map[string]ast_domain.PropValue{
				"id": {Expression: &ast_domain.StringLiteral{Value: "123"}, GoFieldName: "ID"},
			},
			passedProps: map[string]ast_domain.PropValue{
				"title": {Expression: &ast_domain.StringLiteral{Value: "Test"}, GoFieldName: "Title"},
			},
			validateKey: func(t *testing.T, key string) {
				key2 := calculatePotentialInvocationKey(
					"card",
					map[string]ast_domain.PropValue{
						"id": {Expression: &ast_domain.StringLiteral{Value: "123"}, GoFieldName: "ID"},
					},
					map[string]ast_domain.PropValue{
						"title": {Expression: &ast_domain.StringLiteral{Value: "Test"}, GoFieldName: "Title"},
					},
				)
				if key != key2 {
					t.Error("Expected same key for identical props")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := calculatePotentialInvocationKey(tt.partialAlias, tt.reqOverrides, tt.passedProps)

			if tt.validateKey != nil {
				tt.validateKey(t, key)
			}
		})
	}
}

func TestExtractPropsForLinking(t *testing.T) {
	tests := []struct {
		node                 *ast_domain.TemplateNode
		validateProps        func(*testing.T, map[string]ast_domain.PropValue, map[string]ast_domain.PropValue)
		name                 string
		expectedReqOverrides int
		expectedPassedProps  int
	}{
		{
			name: "node with no prop attributes",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "container"},
				},
			},
			expectedReqOverrides: 0,
			expectedPassedProps:  0,
		},
		{
			name: "node with request override",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "request.userId",
						Expression: &ast_domain.Identifier{Name: "userId"},
					},
				},
			},
			expectedReqOverrides: 1,
			expectedPassedProps:  0,
			validateProps: func(t *testing.T, reqOverrides, passedProps map[string]ast_domain.PropValue) {
				if _, found := reqOverrides["userId"]; !found {
					t.Error("Expected 'userId' in request overrides")
				}
			},
		},
		{
			name: "node with passed prop",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "title",
						Expression: &ast_domain.StringLiteral{Value: "Hello"},
					},
				},
			},
			expectedReqOverrides: 0,
			expectedPassedProps:  1,
			validateProps: func(t *testing.T, reqOverrides, passedProps map[string]ast_domain.PropValue) {
				if _, found := passedProps["title"]; !found {
					t.Error("Expected 'title' in passed props")
				}
			},
		},
		{
			name: "node with both request overrides and passed props",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "request.path",
						Expression: &ast_domain.StringLiteral{Value: "/home"},
					},
					{
						Name:       "caption",
						Expression: &ast_domain.StringLiteral{Value: "Welcome"},
					},
				},
			},
			expectedReqOverrides: 1,
			expectedPassedProps:  1,
			validateProps: func(t *testing.T, reqOverrides, passedProps map[string]ast_domain.PropValue) {
				if _, found := reqOverrides["path"]; !found {
					t.Error("Expected 'path' in request overrides")
				}
				if _, found := passedProps["caption"]; !found {
					t.Error("Expected 'caption' in passed props")
				}
			},
		},
		{
			name: "node with dynamic and static attributes",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "dynamicProp",
						Expression: &ast_domain.StringLiteral{Value: "dynamic"},
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "container"},
					{Name: "title", Value: "Hello"},
				},
			},
			expectedReqOverrides: 0,
			expectedPassedProps:  2,
			validateProps: func(t *testing.T, reqOverrides, passedProps map[string]ast_domain.PropValue) {
				if _, found := passedProps["class"]; found {
					t.Error("Expected 'class' to be excluded (structural attribute)")
				}
				if _, found := passedProps["title"]; !found {
					t.Error("Expected 'title' to be included")
				}
				if _, found := passedProps["dynamicprop"]; !found {
					t.Error("Expected 'dynamicprop' to be included (lowercased)")
				}
			},
		},
		{
			name: "node with server.* prefix in dynamic attributes",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "server.meta",
						Expression: &ast_domain.StringLiteral{Value: "test"},
					},
					{
						Name:       "normalProp",
						Expression: &ast_domain.StringLiteral{Value: "value"},
					},
				},
			},
			expectedReqOverrides: 0,
			expectedPassedProps:  2,
			validateProps: func(t *testing.T, reqOverrides, passedProps map[string]ast_domain.PropValue) {
				if _, found := passedProps["server.meta"]; !found {
					t.Error("Expected 'server.meta' to be included in passedProps")
				}
				if _, found := passedProps["normalprop"]; !found {
					t.Error("Expected 'normalprop' to be included (lowercased)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqOverrides, passedProps := extractPropsForLinking(context.Background(), tt.node, "test")

			if len(reqOverrides) != tt.expectedReqOverrides {
				t.Errorf("Expected %d request overrides, got %d", tt.expectedReqOverrides, len(reqOverrides))
			}
			if len(passedProps) != tt.expectedPassedProps {
				t.Errorf("Expected %d passed props, got %d", tt.expectedPassedProps, len(passedProps))
			}

			if tt.validateProps != nil {
				tt.validateProps(t, reqOverrides, passedProps)
			}
		})
	}
}

func TestAssembleFinalCSS(t *testing.T) {
	t.Parallel()

	t.Run("returns empty string for empty maps", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: make(map[string]string),
			scopedCSSBlocks: make(map[string]string),
		}

		result, err := ec.assembleFinalCSS()

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != "" {
			t.Errorf("Expected empty CSS, got: %q", result)
		}
	})

	t.Run("assembles global CSS blocks sorted by path", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: map[string]string{
				"/z/styles.css": ".z { color: red; }",
				"/a/styles.css": ".a { color: blue; }",
			},
			scopedCSSBlocks: make(map[string]string),
		}

		result, err := ec.assembleFinalCSS()

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !strings.Contains(result, ".a { color: blue; }") {
			t.Error("Expected global CSS block for /a/styles.css")
		}
		if !strings.Contains(result, ".z { color: red; }") {
			t.Error("Expected global CSS block for /z/styles.css")
		}

		aIndex := strings.Index(result, ".a {")
		zIndex := strings.Index(result, ".z {")
		if aIndex > zIndex {
			t.Error("Expected global blocks to be sorted by path (a before z)")
		}
	})

	t.Run("assembles scoped CSS blocks sorted by scope ID", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: make(map[string]string),
			scopedCSSBlocks: map[string]string{
				"scope_z": ".scoped-z { margin: 0; }",
				"scope_a": ".scoped-a { padding: 0; }",
			},
		}

		result, err := ec.assembleFinalCSS()

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		aIndex := strings.Index(result, ".scoped-a")
		zIndex := strings.Index(result, ".scoped-z")
		if aIndex > zIndex {
			t.Error("Expected scoped blocks to be sorted by scope ID (a before z)")
		}
	})

	t.Run("global blocks come before scoped blocks", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: map[string]string{
				"/global.css": ".global { display: block; }",
			},
			scopedCSSBlocks: map[string]string{
				"scope_1": ".scoped { display: flex; }",
			},
		}

		result, err := ec.assembleFinalCSS()

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		globalIndex := strings.Index(result, ".global")
		scopedIndex := strings.Index(result, ".scoped")
		if globalIndex > scopedIndex {
			t.Error("Expected global CSS before scoped CSS")
		}
	})

	t.Run("skips empty CSS values", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: map[string]string{
				"/empty.css":  "",
				"/filled.css": ".filled { color: green; }",
			},
			scopedCSSBlocks: map[string]string{
				"empty_scope":  "",
				"filled_scope": ".scoped { color: green; }",
			},
		}

		result, err := ec.assembleFinalCSS()

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if strings.Contains(result, "\n\n\n") {
			t.Error("Expected no extra blank lines from empty blocks")
		}
	})
}

func TestGetSortedPartialInvocations(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for empty map", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
		}

		result := ec.getSortedPartialInvocations()

		if len(result) != 0 {
			t.Errorf("Expected 0 invocations, got %d", len(result))
		}
	})

	t.Run("returns invocations sorted by key", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			uniqueInvocations: map[string]*annotator_dto.PartialInvocation{
				"z_key": {InvocationKey: "z_key", PartialAlias: "z_partial"},
				"a_key": {InvocationKey: "a_key", PartialAlias: "a_partial"},
				"m_key": {InvocationKey: "m_key", PartialAlias: "m_partial"},
			},
		}

		result := ec.getSortedPartialInvocations()

		if len(result) != 3 {
			t.Fatalf("Expected 3 invocations, got %d", len(result))
		}
		if result[0].InvocationKey != "a_key" {
			t.Errorf("Expected first key 'a_key', got %q", result[0].InvocationKey)
		}
		if result[1].InvocationKey != "m_key" {
			t.Errorf("Expected second key 'm_key', got %q", result[1].InvocationKey)
		}
		if result[2].InvocationKey != "z_key" {
			t.Errorf("Expected third key 'z_key', got %q", result[2].InvocationKey)
		}
	})
}

func TestIsStructuralAttribute(t *testing.T) {
	tests := []struct {
		name          string
		attributeName string
		expected      bool
	}{
		{name: "class attribute", attributeName: "class", expected: true},
		{name: "style attribute", attributeName: "style", expected: false},
		{name: "id attribute", attributeName: "id", expected: false},
		{name: "is attribute", attributeName: "is", expected: true},
		{name: "key attribute", attributeName: "key", expected: false},
		{name: "ref attribute", attributeName: "ref", expected: false},
		{name: "slot attribute", attributeName: "slot", expected: false},
		{name: "p-if directive", attributeName: "p-if", expected: true},
		{name: "p-for directive", attributeName: "p-for", expected: true},
		{name: "p-on:click directive", attributeName: "p-on:click", expected: true},
		{name: "normal attribute", attributeName: "title", expected: false},
		{name: "data attribute", attributeName: "data-value", expected: false},
		{name: "aria attribute", attributeName: "aria-label", expected: false},
		{name: "custom attribute", attributeName: "x-custom", expected: false},
		{name: "empty string", attributeName: "", expected: false},
		{name: "CLASS uppercase", attributeName: "CLASS", expected: true},
		{name: "Is mixed case", attributeName: "Is", expected: true},
		{name: "P-IF uppercase", attributeName: "P-IF", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStructuralAttribute(tt.attributeName)

			if result != tt.expected {
				t.Errorf("Expected isStructuralAttribute(%q) to return %v, got %v", tt.attributeName, tt.expected, result)
			}
		})
	}
}

func TestCreatePartialInfoAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("creates partial info with correct fields", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			graph: &annotator_dto.ComponentGraph{
				Components: map[string]*annotator_dto.ParsedComponent{
					"invoker_hash": {
						SourcePath: "/test/invoker.pk",
					},
				},
			},
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		invokerNode := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			Location: ast_domain.Location{Line: 10, Column: 5, Offset: 100},
		}

		partialInfo := ec.createPartialInfoAnnotation(context.Background(), invokerNode, "card", "target_hash", "invoker_hash")

		require.NotNil(t, partialInfo)
		assert.Equal(t, "card", partialInfo.PartialAlias)
		assert.Equal(t, "target_hash", partialInfo.PartialPackageName)
		assert.Equal(t, "invoker_hash", partialInfo.InvokerPackageAlias)
		assert.Equal(t, ast_domain.Location{Line: 10, Column: 5, Offset: 100}, partialInfo.Location)
		assert.NotEmpty(t, partialInfo.InvocationKey)
		assert.Empty(t, partialInfo.InvokerInvocationKey)
	})

	t.Run("registers invocation in uniqueInvocations on first call", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			graph: &annotator_dto.ComponentGraph{
				Components: map[string]*annotator_dto.ParsedComponent{
					"invoker_hash": {
						SourcePath: "/test/invoker.pk",
					},
				},
			},
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		invokerNode := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
		}

		pInfo := ec.createPartialInfoAnnotation(context.Background(), invokerNode, "card", "target_hash", "invoker_hash")

		assert.Len(t, ec.uniqueInvocations, 1)
		assert.Len(t, ec.invocationOrder, 1)

		invocation := ec.uniqueInvocations[pInfo.InvocationKey]
		require.NotNil(t, invocation)
		assert.Equal(t, "card", invocation.PartialAlias)
		assert.Equal(t, "target_hash", invocation.PartialHashedName)
		assert.Equal(t, "invoker_hash", invocation.InvokerHashedName)
	})

	t.Run("does not duplicate invocation on second call with same key", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			graph: &annotator_dto.ComponentGraph{
				Components: map[string]*annotator_dto.ParsedComponent{
					"invoker_hash": {
						SourcePath: "/test/invoker.pk",
					},
				},
			},
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		invokerNode := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
		}

		pInfo1 := ec.createPartialInfoAnnotation(context.Background(), invokerNode, "card", "target_hash", "invoker_hash")
		pInfo2 := ec.createPartialInfoAnnotation(context.Background(), invokerNode, "card", "target_hash", "invoker_hash")

		assert.Equal(t, pInfo1.InvocationKey, pInfo2.InvocationKey)
		assert.Len(t, ec.uniqueInvocations, 1)
		assert.Len(t, ec.invocationOrder, 1)
	})

	t.Run("handles invoker hash not found in graph", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			graph: &annotator_dto.ComponentGraph{
				Components: map[string]*annotator_dto.ParsedComponent{},
			},
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		invokerNode := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
		}

		pInfo := ec.createPartialInfoAnnotation(context.Background(), invokerNode, "card", "target_hash", "missing_hash")

		require.NotNil(t, pInfo)
		assert.Equal(t, "card", pInfo.PartialAlias)
	})

	t.Run("extracts dynamic attribute props into passed props", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			graph: &annotator_dto.ComponentGraph{
				Components: map[string]*annotator_dto.ParsedComponent{
					"invoker_hash": {
						SourcePath: "/test/invoker.pk",
					},
				},
			},
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		invokerNode := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "title",
					Expression: &ast_domain.StringLiteral{Value: "Hello"},
				},
			},
		}

		pInfo := ec.createPartialInfoAnnotation(context.Background(), invokerNode, "card", "target_hash", "invoker_hash")

		require.NotNil(t, pInfo)
		assert.NotEmpty(t, pInfo.PassedProps)
		_, hasTitleProp := pInfo.PassedProps["title"]
		assert.True(t, hasTitleProp)
	})

	t.Run("extracts request override props", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			graph: &annotator_dto.ComponentGraph{
				Components: map[string]*annotator_dto.ParsedComponent{
					"invoker_hash": {
						SourcePath: "/test/invoker.pk",
					},
				},
			},
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		invokerNode := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "request.userId",
					Expression: &ast_domain.Identifier{Name: "userId"},
				},
			},
		}

		pInfo := ec.createPartialInfoAnnotation(context.Background(), invokerNode, "card", "target_hash", "invoker_hash")

		require.NotNil(t, pInfo)
		assert.NotEmpty(t, pInfo.RequestOverrides)
		_, hasUserID := pInfo.RequestOverrides["userId"]
		assert.True(t, hasUserID)
	})

	t.Run("different props produce different invocation keys", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			graph: &annotator_dto.ComponentGraph{
				Components: map[string]*annotator_dto.ParsedComponent{
					"invoker_hash": {
						SourcePath: "/test/invoker.pk",
					},
				},
			},
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		node1 := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", Expression: &ast_domain.StringLiteral{Value: "Hello"}},
			},
		}
		node2 := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", Expression: &ast_domain.StringLiteral{Value: "World"}},
			},
		}

		pInfo1 := ec.createPartialInfoAnnotation(context.Background(), node1, "card", "target_hash", "invoker_hash")
		pInfo2 := ec.createPartialInfoAnnotation(context.Background(), node2, "card", "target_hash", "invoker_hash")

		assert.NotEqual(t, pInfo1.InvocationKey, pInfo2.InvocationKey)
		assert.Len(t, ec.uniqueInvocations, 2)
		assert.Len(t, ec.invocationOrder, 2)
	})
}

func TestExtractPropsForLinkingStaticAttributeEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("skips is attribute from static attributes", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "is", Value: "card"},
				{Name: "title", Value: "Hello"},
			},
		}

		_, passedProps := extractPropsForLinking(context.Background(), node, "/test/source.pk")

		_, hasIs := passedProps["is"]
		assert.False(t, hasIs, "is attribute should be filtered as structural")
		_, hasTitle := passedProps["title"]
		assert.True(t, hasTitle)
	})

	t.Run("skips class attribute from static attributes", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "data-test", Value: "42"},
			},
		}

		_, passedProps := extractPropsForLinking(context.Background(), node, "/test/source.pk")

		_, hasClass := passedProps["class"]
		assert.False(t, hasClass, "class attribute should be filtered as structural")
		_, hasDataTest := passedProps["data-test"]
		assert.True(t, hasDataTest)
	})

	t.Run("skips p- prefixed attributes from static attributes", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "p-fragment", Value: "abc123"},
				{Name: "title", Value: "Test"},
			},
		}

		_, passedProps := extractPropsForLinking(context.Background(), node, "/test/source.pk")

		_, hasPFragment := passedProps["p-fragment"]
		assert.False(t, hasPFragment, "p-fragment should be filtered as structural")
		_, hasTitle := passedProps["title"]
		assert.True(t, hasTitle)
	})

	t.Run("lowercases static attribute names", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "MyProp", Value: "value"},
			},
		}

		_, passedProps := extractPropsForLinking(context.Background(), node, "/test/source.pk")

		_, hasMyProp := passedProps["myprop"]
		assert.True(t, hasMyProp, "static attribute names should be lowercased")
	})

	t.Run("creates parsed expression for static attribute values", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			Attributes: []ast_domain.HTMLAttribute{
				{
					Name:     "title",
					Value:    "Hello World",
					Location: ast_domain.Location{Line: 5, Column: 10},
				},
			},
		}

		_, passedProps := extractPropsForLinking(context.Background(), node, "/test/source.pk")

		titleProp, found := passedProps["title"]
		require.True(t, found)
		assert.NotNil(t, titleProp.Expression, "static attributes should have parsed expressions")
		assert.Equal(t, ast_domain.Location{Line: 5, Column: 10}, titleProp.Location)
	})

	t.Run("empty node produces empty maps", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
		}

		reqOverrides, passedProps := extractPropsForLinking(context.Background(), node, "/test/source.pk")

		assert.Empty(t, reqOverrides)
		assert.Empty(t, passedProps)
	})
}

func TestValidatePikoPartialElementTextNode(t *testing.T) {
	t.Parallel()

	t.Run("text node type returns nil", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "some text",
		}

		diagnostic := validatePikoPartialElement(node, "/test.pk")

		assert.Nil(t, diagnostic)
	})

	t.Run("comment node type returns nil", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeComment,
		}

		diagnostic := validatePikoPartialElement(node, "/test.pk")

		assert.Nil(t, diagnostic)
	})

	t.Run("dynamic is attribute with case insensitive match", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			TagName:  "piko:partial",
			NodeType: ast_domain.NodeElement,
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "IS", RawExpression: "state.comp", Location: ast_domain.Location{Line: 3, Column: 5}},
			},
		}

		diagnostic := validatePikoPartialElement(node, "/test.pk")

		require.NotNil(t, diagnostic)
		assert.Equal(t, ast_domain.Error, diagnostic.Severity)
		assert.Contains(t, diagnostic.Message, "cannot be dynamic")
	})
}

func TestCalculatePotentialInvocationKeyNilExpression(t *testing.T) {
	t.Parallel()

	t.Run("handles nil expression in props", func(t *testing.T) {
		t.Parallel()

		reqOverrides := map[string]ast_domain.PropValue{}
		passedProps := map[string]ast_domain.PropValue{
			"title": {
				Expression:  nil,
				GoFieldName: "Title",
			},
		}

		key := calculatePotentialInvocationKey("card", reqOverrides, passedProps)

		assert.NotEmpty(t, key)
	})

	t.Run("different aliases produce different keys", func(t *testing.T) {
		t.Parallel()

		emptyOverrides := map[string]ast_domain.PropValue{}
		emptyProps := map[string]ast_domain.PropValue{}

		key1 := calculatePotentialInvocationKey("card", emptyOverrides, emptyProps)
		key2 := calculatePotentialInvocationKey("modal", emptyOverrides, emptyProps)

		assert.NotEqual(t, key1, key2)
	})

	t.Run("request overrides and passed props differ in key prefix", func(t *testing.T) {
		t.Parallel()

		propsMap := map[string]ast_domain.PropValue{
			"title": {Expression: &ast_domain.StringLiteral{Value: "Hello"}},
		}

		key1 := calculatePotentialInvocationKey("card", propsMap, map[string]ast_domain.PropValue{})
		key2 := calculatePotentialInvocationKey("card", map[string]ast_domain.PropValue{}, propsMap)

		assert.NotEqual(t, key1, key2, "request overrides should produce different key than passed props")
	})
}

func TestExpansionContext_assembleFinalCSS(t *testing.T) {
	t.Parallel()

	t.Run("empty maps returns empty string", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: map[string]string{},
			scopedCSSBlocks: map[string]string{},
		}

		css, err := ec.assembleFinalCSS()

		require.NoError(t, err)
		assert.Equal(t, "", css)
	})

	t.Run("global blocks are sorted by path", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: map[string]string{
				"/z.css": "z{}",
				"/a.css": "a{}",
				"/m.css": "m{}",
			},
			scopedCSSBlocks: map[string]string{},
		}

		css, err := ec.assembleFinalCSS()

		require.NoError(t, err)
		assert.True(t, strings.Index(css, "a{}") < strings.Index(css, "m{}"))
		assert.True(t, strings.Index(css, "m{}") < strings.Index(css, "z{}"))
	})

	t.Run("scoped blocks are sorted by scope ID", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: map[string]string{},
			scopedCSSBlocks: map[string]string{
				"scope-z": ".z{color:red}",
				"scope-a": ".a{color:blue}",
			},
		}

		css, err := ec.assembleFinalCSS()

		require.NoError(t, err)
		assert.True(t, strings.Index(css, ".a{color:blue}") < strings.Index(css, ".z{color:red}"))
	})

	t.Run("global blocks come before scoped blocks", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: map[string]string{
				"/global.css": "body{margin:0}",
			},
			scopedCSSBlocks: map[string]string{
				"scope-a": ".a{color:red}",
			},
		}

		css, err := ec.assembleFinalCSS()

		require.NoError(t, err)
		assert.True(t, strings.Index(css, "body{margin:0}") < strings.Index(css, ".a{color:red}"))
	})

	t.Run("empty CSS blocks are skipped", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			globalCSSBlocks: map[string]string{
				"/a.css": "",
				"/b.css": "b{}",
			},
			scopedCSSBlocks: map[string]string{
				"scope-a": "",
				"scope-b": ".b{}",
			},
		}

		css, err := ec.assembleFinalCSS()

		require.NoError(t, err)
		assert.Equal(t, "b{}\n.b{}", css)
	})
}

func TestExpansionContext_getSortedPartialInvocations(t *testing.T) {
	t.Parallel()

	t.Run("empty map returns empty slice", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			uniqueInvocations: map[string]*annotator_dto.PartialInvocation{},
		}

		result := ec.getSortedPartialInvocations()

		assert.Empty(t, result)
	})

	t.Run("returns invocations sorted by key", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			uniqueInvocations: map[string]*annotator_dto.PartialInvocation{
				"zebra": {InvocationKey: "zebra", PartialAlias: "z", PartialHashedName: "zh", PassedProps: nil, RequestOverrides: nil, InvokerHashedName: "", InvokerInvocationKey: "", DependsOn: nil, Location: ast_domain.Location{Line: 0, Column: 0, Offset: 0}},
				"alpha": {InvocationKey: "alpha", PartialAlias: "a", PartialHashedName: "ah", PassedProps: nil, RequestOverrides: nil, InvokerHashedName: "", InvokerInvocationKey: "", DependsOn: nil, Location: ast_domain.Location{Line: 0, Column: 0, Offset: 0}},
				"mango": {InvocationKey: "mango", PartialAlias: "m", PartialHashedName: "mh", PassedProps: nil, RequestOverrides: nil, InvokerHashedName: "", InvokerInvocationKey: "", DependsOn: nil, Location: ast_domain.Location{Line: 0, Column: 0, Offset: 0}},
			},
		}

		result := ec.getSortedPartialInvocations()

		require.Len(t, result, 3)
		assert.Equal(t, "alpha", result[0].InvocationKey)
		assert.Equal(t, "mango", result[1].InvocationKey)
		assert.Equal(t, "zebra", result[2].InvocationKey)
	})
}

func TestNewExpansionContext_NilTemplateReturnsNilAST(t *testing.T) {
	t.Parallel()

	graph := &annotator_dto.ComponentGraph{
		Components: map[string]*annotator_dto.ParsedComponent{
			"my_hash": {
				SourcePath: "/test.pk",
				Template:   nil,
			},
		},
		PathToHashedName:  map[string]string{},
		AllSourceContents: nil,
		HashedNameToPath:  nil,
	}

	ec, comp, ast, err := newExpansionContext(nil, graph, "my_hash")

	assert.Nil(t, ec)
	require.NoError(t, err)
	require.NotNil(t, comp)
	assert.Equal(t, "/test.pk", comp.SourcePath)
	assert.Nil(t, ast)
}

func TestNewExpansionContext_MissingEntryPoint(t *testing.T) {
	t.Parallel()

	graph := &annotator_dto.ComponentGraph{
		Components:        map[string]*annotator_dto.ParsedComponent{},
		PathToHashedName:  map[string]string{},
		AllSourceContents: nil,
		HashedNameToPath:  nil,
	}

	ec, comp, ast, err := newExpansionContext(nil, graph, "missing_hash")

	assert.Nil(t, ec)
	assert.Nil(t, comp)
	assert.Nil(t, ast)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in component graph")
}

func TestValidatePikoElementNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		node         *ast_domain.TemplateNode
		name         string
		errorMessage string
		expectError  bool
	}{
		{
			name: "valid static is=div",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "div"},
				},
			},
			expectError: false,
		},
		{
			name: "valid dynamic :is",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "is", RawExpression: "state.tag"},
				},
			},
			expectError: false,
		},
		{
			name: "missing is attribute",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
			},
			expectError:  true,
			errorMessage: "requires an 'is' attribute",
		},
		{
			name: "empty is attribute",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: ""},
				},
			},
			expectError:  true,
			errorMessage: "requires an 'is' attribute",
		},
		{
			name: "rejected target piko:partial",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "piko:partial"},
				},
			},
			expectError:  true,
			errorMessage: "cannot target 'piko:partial'",
		},
		{
			name: "rejected target piko:slot",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "piko:slot"},
				},
			},
			expectError:  true,
			errorMessage: "cannot target 'piko:slot'",
		},
		{
			name: "rejected target piko:element (no recursion)",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "piko:element"},
				},
			},
			expectError:  true,
			errorMessage: "cannot target 'piko:element'",
		},
		{
			name: "non piko:element node is not validated",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
			},
			expectError: false,
		},
		{
			name:        "nil node is valid",
			node:        nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diagnostic := validatePikoElementNode(tt.node, "test.pk")

			if tt.expectError {
				if diagnostic == nil {
					t.Error("Expected error diagnostic but got nil")
				} else if !strings.Contains(diagnostic.Message, tt.errorMessage) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMessage, diagnostic.Message)
				}
			} else if diagnostic != nil {
				t.Errorf("Expected no error, got: %s", diagnostic.Message)
			}
		})
	}
}

func TestResolvePikoElementStaticIs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		node           *ast_domain.TemplateNode
		name           string
		expectedTag    string
		expectResolved bool
	}{
		{
			name: "resolves static is=h2",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "h2"},
				},
			},
			expectResolved: true,
			expectedTag:    "h2",
		},
		{
			name: "resolves static is=piko:a",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "piko:a"},
				},
			},
			expectResolved: true,
			expectedTag:    "piko:a",
		},
		{
			name: "does not resolve dynamic :is",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "is", RawExpression: "state.tag"},
				},
			},
			expectResolved: false,
			expectedTag:    "piko:element",
		},
		{
			name: "does not resolve rejected targets",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "piko:partial"},
				},
			},
			expectResolved: false,
			expectedTag:    "piko:element",
		},
		{
			name: "does not resolve non-piko:element nodes",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
			},
			expectResolved: false,
			expectedTag:    "div",
		},
		{
			name:           "nil node returns false",
			node:           nil,
			expectResolved: false,
		},
		{
			name: "removes is attribute after resolving",
			node: &ast_domain.TemplateNode{
				TagName:  "piko:element",
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "is", Value: "span"},
					{Name: "class", Value: "highlight"},
				},
			},
			expectResolved: true,
			expectedTag:    "span",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolved := resolvePikoElementStaticIs(tt.node)
			assert.Equal(t, tt.expectResolved, resolved)

			if tt.node != nil {
				assert.Equal(t, tt.expectedTag, tt.node.TagName)
			}

			if tt.name == "removes is attribute after resolving" && resolved {
				_, hasIs := tt.node.GetAttribute("is")
				assert.False(t, hasIs, "is attribute should be removed after resolution")
				classVal, hasClass := tt.node.GetAttribute("class")
				assert.True(t, hasClass, "other attributes should remain")
				assert.Equal(t, "highlight", classVal)
			}
		})
	}
}
