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

package pml_domain

import (
	"bytes"
	"cmp"
	"context"
	"slices"
	"strings"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_AutowrapPass(t *testing.T) {
	transformer := NewTransformer(nil, newMockMediaQueryCollector(), newMockMSOConditionalCollector())

	testCases := []struct {
		name          string
		inputPML      string
		expectedAST   string
		parentTagName string
	}{
		{
			name:          "Root: Wraps single content component",
			inputPML:      `<pml-p>Hello</pml-p>`,
			expectedAST:   "pml-row(pml-col(pml-p(TEXT)))",
			parentTagName: "",
		},
		{
			name:          "Row: Wraps content component in a column",
			inputPML:      `<pml-p>Hello</pml-p>`,
			expectedAST:   "pml-row(pml-col(pml-p(TEXT)))",
			parentTagName: "pml-row",
		},
		{
			name:          "Row: Groups multiple content components in one column",
			inputPML:      `<pml-p>One</pml-p><pml-img/>`,
			expectedAST:   "pml-row(pml-col(pml-p(TEXT), pml-img))",
			parentTagName: "pml-row",
		},
		{
			name:          "Row: Explicit column breaks implicit grouping",
			inputPML:      `<pml-p>One</pml-p><pml-col></pml-col><pml-p>Two</pml-p>`,
			expectedAST:   "pml-row(pml-col(pml-p(TEXT)), pml-col, pml-col(pml-p(TEXT)))",
			parentTagName: "pml-row",
		},
		{
			name:          "No-Op: A valid structure is not changed",
			inputPML:      `<pml-row><pml-col><pml-p>Hello</pml-p></pml-col></pml-row>`,
			expectedAST:   "pml-row(pml-col(pml-p(TEXT)))",
			parentTagName: "",
		},
		{
			name:          "Container: Wraps stray column in a row",
			inputPML:      `<pml-col></pml-col>`,
			expectedAST:   "pml-container(pml-row(pml-col))",
			parentTagName: "pml-container",
		},
		{
			name:          "OrderedList: Wraps content in list item",
			inputPML:      `<pml-p>Item 1</pml-p>`,
			expectedAST:   "pml-row(pml-col(pml-ol(pml-li(pml-p(TEXT)))))",
			parentTagName: "pml-ol",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputToParseWithContext := tc.inputPML
			if tc.parentTagName != "" {
				inputToParseWithContext = "<" + tc.parentTagName + ">" + tc.inputPML + "</" + tc.parentTagName + ">"
			}
			ast, err := parsePML(t, inputToParseWithContext)
			require.NoError(t, err)

			e, ok := transformer.(*engine)
			require.True(t, ok, "Expected transformer to be *engine")
			e.autowrapTree(context.Background(), ast)

			actualASTString := astToDebugString(ast.RootNodes)
			assert.Equal(t, tc.expectedAST, actualASTString)
		})
	}
}

func TestEngine_TransformPass_ValidInput(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	testCases := []struct {
		name         string
		validPML     string
		expectedHTML string
	}{
		{
			name:         "Transforms simple valid structure",
			validPML:     `<pml-row><pml-col><pml-p>Hello</pml-p></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p>Hello</p></div></div>`,
		},
		{
			name:         "Transforms two columns correctly",
			validPML:     `<pml-row><pml-col><pml-p>Left</pml-p></pml-col><pml-col><pml-p>Right</pml-p></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p>Left</p></div><div class="col"><p>Right</p></div></div>`,
		},
		{
			name:         "Transforms nested structure",
			validPML:     `<pml-row><pml-col><pml-p>Paragraph 1</pml-p><pml-p>Paragraph 2</pml-p></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p>Paragraph 1</p><p>Paragraph 2</p></div></div>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := parsePML(t, tc.validPML)
			require.NoError(t, err)

			transformedAST, _, diagnostics := engine.Transform(context.Background(), ast, config)
			require.Empty(t, diagnostics, "Expected no diagnostics, but got: %v", diagnostics)

			actualHTML := renderASTToHTML(transformedAST)
			assert.Equal(t, normaliseHTML(tc.expectedHTML), normaliseHTML(actualHTML))
		})
	}
}

func TestEngine_Transform_EndToEnd(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	testCases := []struct {
		name         string
		lenientPML   string
		expectedHTML string
	}{
		{
			name:         "Autowraps and transforms a single root text node",
			lenientPML:   `Hello World`,
			expectedHTML: `<div class="row"><div class="col"><p>Hello World</p></div></div>`,
		},
		{
			name:         "Autowraps and transforms content inside a row",
			lenientPML:   `<pml-row><pml-p>Content in a row</pml-p></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p>Content in a row</p></div></div>`,
		},
		{
			name:         "Autowraps multiple content items into single column",
			lenientPML:   `<pml-row><pml-p>First</pml-p><pml-p>Second</pml-p></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p>First</p><p>Second</p></div></div>`,
		},
		{
			name:         "Explicit column breaks autowrapping grouping",
			lenientPML:   `<pml-row><pml-p>First</pml-p><pml-col><pml-p>Second</pml-p></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p>First</p></div><div class="col"><p>Second</p></div></div>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := parsePML(t, tc.lenientPML)
			require.NoError(t, err)

			transformedAST, _, diagnostics := engine.Transform(context.Background(), ast, config)
			require.Empty(t, diagnostics, "Expected no diagnostics, but got: %v", diagnostics)

			actualHTML := renderASTToHTML(transformedAST)
			assert.Equal(t, normaliseHTML(tc.expectedHTML), normaliseHTML(actualHTML))
		})
	}
}

func TestEngine_ErrorHandling(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	testCases := []struct {
		inputAST             *ast_domain.TemplateAST
		inputConfig          *pml_dto.Config
		name                 string
		diagnosticSubstrings []string
		expectedDiagnostics  int
		expectError          bool
	}{
		{
			name:        "Nil AST input",
			inputAST:    nil,
			inputConfig: config,
			expectError: true,
		},
		{
			name: "Unknown PML component",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-unknown-component",
					},
				},
			},
			inputConfig:          config,
			expectError:          false,
			expectedDiagnostics:  1,
			diagnosticSubstrings: []string{"Unknown PML component", "pml-unknown-component"},
		},
		{
			name: "Multiple unknown components",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-row",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "pml-col",
								Children: []*ast_domain.TemplateNode{
									{
										NodeType: ast_domain.NodeElement,
										TagName:  "pml-unknown-1",
									},
									{
										NodeType: ast_domain.NodeElement,
										TagName:  "pml-unknown-2",
									},
								},
							},
						},
					},
				},
			},
			inputConfig:          config,
			expectError:          false,
			expectedDiagnostics:  2,
			diagnosticSubstrings: []string{"Unknown PML component"},
		},
		{
			name: "Nil config uses defaults",
			inputAST: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-row",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "pml-col",
								Children: []*ast_domain.TemplateNode{
									{
										NodeType: ast_domain.NodeElement,
										TagName:  "pml-p",
										Children: []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeText, TextContent: "Test"}},
									},
								},
							},
						},
					},
				},
			},
			inputConfig:         nil,
			expectError:         false,
			expectedDiagnostics: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transformedAST, _, diagnostics := engine.Transform(context.Background(), tc.inputAST, tc.inputConfig)

			if tc.expectError {
				require.NotNil(t, diagnostics, "Expected error diagnostics")
				require.Greater(t, len(diagnostics), 0, "Expected at least one diagnostic")
			} else {
				if tc.expectedDiagnostics > 0 {
					require.Len(t, diagnostics, tc.expectedDiagnostics, "Diagnostic count mismatch")
					for _, substr := range tc.diagnosticSubstrings {
						found := false
						for _, diagnostic := range diagnostics {
							if strings.Contains(diagnostic.Message, substr) {
								found = true
								break
							}
						}
						assert.True(t, found, "Expected diagnostic message to contain: %s", substr)
					}
				} else {
					require.Empty(t, diagnostics, "Expected no diagnostics, but got: %v", diagnostics)
					require.NotNil(t, transformedAST, "Expected valid transformed AST")
				}
			}
		})
	}
}

func TestEngine_CSSCollection(t *testing.T) {
	registry := buildMockRegistry()

	_ = registry.(*mockRegistry).Register(context.Background(), &mockComponent{
		tagName:     "pml-responsive",
		isEndingTag: false,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			if ctx.MediaQueryCollector != nil {
				ctx.MediaQueryCollector.RegisterClass("test-responsive", "width: 100% !important;")
			}
			if ctx.MSOConditionalCollector != nil {
				ctx.MSOConditionalCollector.RegisterStyle(".test-mso", "margin: 0 !important;")
			}
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "test-responsive test-mso"},
				},
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	testCases := []struct {
		name            string
		inputPML        string
		mediaSubstrings []string
		msoSubstrings   []string
		expectMediaCSS  bool
		expectMSOCSS    bool
	}{
		{
			name:            "Component registers media query and MSO styles",
			inputPML:        `<pml-responsive><pml-p>Test</pml-p></pml-responsive>`,
			expectMediaCSS:  true,
			expectMSOCSS:    true,
			mediaSubstrings: []string{"@media", "480px", "test-responsive", "width: 100%"},
			msoSubstrings:   []string{"<!--[if mso]>", ".test-mso", "margin: 0"},
		},
		{
			name:            "Multiple components deduplicate CSS",
			inputPML:        `<pml-responsive><pml-p>A</pml-p></pml-responsive><pml-responsive><pml-p>B</pml-p></pml-responsive>`,
			expectMediaCSS:  true,
			expectMSOCSS:    true,
			mediaSubstrings: []string{"test-responsive"},
			msoSubstrings:   []string{".test-mso"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := parsePML(t, tc.inputPML)
			require.NoError(t, err)

			_, cssOutput, diagnostics := engine.Transform(context.Background(), ast, config)
			require.Empty(t, diagnostics, "Expected no diagnostics")

			if tc.expectMediaCSS || tc.expectMSOCSS {
				require.NotEmpty(t, cssOutput, "Expected CSS output")
			}

			if tc.expectMediaCSS {
				for _, substr := range tc.mediaSubstrings {
					assert.Contains(t, cssOutput, substr, "Expected media query CSS to contain: %s", substr)
				}
			}

			if tc.expectMSOCSS {
				for _, substr := range tc.msoSubstrings {
					assert.Contains(t, cssOutput, substr, "Expected MSO CSS to contain: %s", substr)
				}
			}
		})
	}
}

func TestEngine_ConfigurationVariations(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())

	testCases := []struct {
		config     *pml_dto.Config
		verifyFunc func(t *testing.T, ast *ast_domain.TemplateAST, css string, diagnostics []*Error)
		name       string
		inputPML   string
	}{
		{
			name:     "Custom breakpoint in config",
			inputPML: `<pml-row><pml-col><pml-p>Test</pml-p></pml-col></pml-row>`,
			config: &pml_dto.Config{
				Breakpoint: "768px",
			},
			verifyFunc: func(t *testing.T, ast *ast_domain.TemplateAST, css string, diagnostics []*Error) {
				require.Empty(t, diagnostics)
				require.NotNil(t, ast)
			},
		},
		{
			name:     "Default config when nil",
			inputPML: `<pml-row><pml-col><pml-p>Test</pml-p></pml-col></pml-row>`,
			config:   nil,
			verifyFunc: func(t *testing.T, ast *ast_domain.TemplateAST, css string, diagnostics []*Error) {
				require.Empty(t, diagnostics)
				require.NotNil(t, ast)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := parsePML(t, tc.inputPML)
			require.NoError(t, err)

			transformedAST, css, diagnostics := engine.Transform(context.Background(), ast, tc.config)

			tc.verifyFunc(t, transformedAST, css, diagnostics)
		})
	}
}

func TestEngine_ComplexStructures(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	testCases := []struct {
		name         string
		inputPML     string
		expectedHTML string
	}{
		{
			name: "Deeply nested structure",
			inputPML: `<pml-container>
				<pml-row>
					<pml-col>
						<pml-p>Outer</pml-p>
						<pml-row>
							<pml-col>
								<pml-p>Inner</pml-p>
							</pml-col>
						</pml-row>
					</pml-col>
				</pml-row>
			</pml-container>`,
			expectedHTML: `<div class="container"><div class="row"><div class="col"><p>Outer</p><div class="row"><div class="col"><p>Inner</p></div></div></div></div></div>`,
		},
		{
			name: "Mixed content with fragments",
			inputPML: `<pml-row>
				<pml-col>
					<pml-p>Before</pml-p>
					<!-- Comment should be preserved -->
					<pml-p>After</pml-p>
				</pml-col>
			</pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p>Before</p><!-- Comment should be preserved --><p>After</p></div></div>`,
		},
		{
			name:         "Ordered list with nested content",
			inputPML:     `<pml-row><pml-col><pml-ol><pml-li><pml-p>Item 1</pml-p></pml-li><pml-li><pml-p>Item 2</pml-p></pml-li></pml-ol></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><ol><li><p>Item 1</p></li><li><p>Item 2</p></li></ol></div></div>`,
		},
		{
			name:         "Hero component with content",
			inputPML:     `<pml-hero><pml-p>Hero Content</pml-p></pml-hero>`,
			expectedHTML: `<div class="hero"><p>Hero Content</p></div>`,
		},
		{
			name:         "Multiple self-closing elements",
			inputPML:     `<pml-row><pml-col><pml-hr/><pml-p>Middle</pml-p><pml-br/></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><hr /><p>Middle</p><br /></div></div>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := parsePML(t, tc.inputPML)
			require.NoError(t, err)

			transformedAST, _, diagnostics := engine.Transform(context.Background(), ast, config)
			require.Empty(t, diagnostics, "Expected no diagnostics")

			actualHTML := renderASTToHTML(transformedAST)
			assert.Equal(t, normaliseHTML(tc.expectedHTML), normaliseHTML(actualHTML))
		})
	}
}

func TestEngine_EndingTagBehaviour(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	testCases := []struct {
		name         string
		inputPML     string
		expectedHTML string
	}{
		{
			name:         "Ending tag with nested PML is not transformed",
			inputPML:     `<pml-row><pml-col><pml-p><pml-button>Nested</pml-button></pml-p></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p><pml-button>Nested</pml-button></p></div></div>`,
		},
		{
			name:         "Multiple ending tags preserve their content",
			inputPML:     `<pml-row><pml-col><pml-p>First</pml-p><pml-button>Second</pml-button></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p>First</p><button>Second</button></div></div>`,
		},
		{
			name:         "Non-ending tag recurses into children",
			inputPML:     `<pml-row><pml-col><pml-p>Content</pml-p></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><p>Content</p></div></div>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := parsePML(t, tc.inputPML)
			require.NoError(t, err)

			transformedAST, _, diagnostics := engine.Transform(context.Background(), ast, config)
			require.Empty(t, diagnostics, "Expected no diagnostics")

			actualHTML := renderASTToHTML(transformedAST)
			assert.Equal(t, normaliseHTML(tc.expectedHTML), normaliseHTML(actualHTML))
		})
	}
}

func TestEngine_MixedHTMLPML(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	testCases := []struct {
		name         string
		inputPML     string
		expectedHTML string
	}{
		{
			name:         "PML inside HTML div",
			inputPML:     `<div class="wrapper"><pml-row><pml-col><pml-p>Inside HTML</pml-p></pml-col></pml-row></div>`,
			expectedHTML: `<div class="wrapper"><div class="row"><div class="col"><p>Inside HTML</p></div></div></div>`,
		},
		{
			name:         "HTML inside PML components triggers autowrap",
			inputPML:     `<pml-row><pml-col><div class="custom"><pml-p>Mixed</pml-p></div></pml-col></pml-row>`,
			expectedHTML: `<div class="row"><div class="col"><div class="custom"><div class="row"><div class="col"><p>Mixed</p></div></div></div></div></div>`,
		},
		{
			name:         "Multiple levels of mixing with autowrap",
			inputPML:     `<div><pml-row><pml-col><section><pml-p>Deep</pml-p></section></pml-col></pml-row></div>`,
			expectedHTML: `<div><div class="row"><div class="col"><section><div class="row"><div class="col"><p>Deep</p></div></div></section></div></div></div>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := parsePML(t, tc.inputPML)
			require.NoError(t, err)

			transformedAST, _, diagnostics := engine.Transform(context.Background(), ast, config)
			require.Empty(t, diagnostics, "Expected no diagnostics")

			actualHTML := renderASTToHTML(transformedAST)
			assert.Equal(t, normaliseHTML(tc.expectedHTML), normaliseHTML(actualHTML))
		})
	}
}

func TestEngine_TransformForEmail(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	testCases := []struct {
		name                    string
		inputPML                string
		expectedHTML            string
		expectedAssetCount      int
		expectedDiagnosticCount int
	}{
		{
			name:               "Basic transformation",
			inputPML:           `<pml-row><pml-col><pml-p>Hello Email</pml-p></pml-col></pml-row>`,
			expectedHTML:       `<div class="row"><div class="col"><p>Hello Email</p></div></div>`,
			expectedAssetCount: 0,
		},
		{
			name:               "Multiple columns",
			inputPML:           `<pml-row><pml-col><pml-p>Left</pml-p></pml-col><pml-col><pml-p>Right</pml-p></pml-col></pml-row>`,
			expectedHTML:       `<div class="row"><div class="col"><p>Left</p></div><div class="col"><p>Right</p></div></div>`,
			expectedAssetCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := parsePML(t, tc.inputPML)
			require.NoError(t, err)

			transformedAST, _, assetRequests, diagnostics := engine.TransformForEmail(context.Background(), ast, config)

			if tc.expectedDiagnosticCount > 0 {
				require.Len(t, diagnostics, tc.expectedDiagnosticCount)
			} else {
				require.Empty(t, diagnostics, "Expected no diagnostics")
			}

			assert.Len(t, assetRequests, tc.expectedAssetCount)
			assert.NotNil(t, transformedAST)

			actualHTML := renderASTToHTML(transformedAST)
			assert.Equal(t, normaliseHTML(tc.expectedHTML), normaliseHTML(actualHTML))
		})
	}
}

func TestEngine_TransformForEmail_NilAST(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	transformedAST, css, assetRequests, diagnostics := engine.TransformForEmail(context.Background(), nil, config)

	assert.Nil(t, transformedAST)
	assert.Empty(t, css)
	assert.Nil(t, assetRequests)
	require.NotEmpty(t, diagnostics)
	assert.Contains(t, diagnostics[0].Message, "Cannot transform nil AST")
}

func TestEngine_TransformForEmail_NilConfig(t *testing.T) {
	registry := buildMockRegistry()
	engine := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())

	ast := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "pml-row",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-col",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "pml-p",
								Children: []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeText, TextContent: "Test"}},
							},
						},
					},
				},
			},
		},
	}

	transformedAST, _, assetRequests, diagnostics := engine.TransformForEmail(context.Background(), ast, nil)

	require.Empty(t, diagnostics)
	assert.NotNil(t, transformedAST)
	assert.NotNil(t, assetRequests)
}

func TestEngine_TransformNode_NilNode(t *testing.T) {
	registry := buildMockRegistry()
	transformer := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	ctx := newRootTransformationContext(config, 600.0, registry)
	e, ok := transformer.(*engine)
	require.True(t, ok, "Expected transformer to be *engine")

	result, diagnostics := e.transformNode(nil, ctx, nil, nil)

	assert.Nil(t, result)
	assert.Empty(t, diagnostics)
}

func TestEngine_CreateChildContext_WithNoStackParent(t *testing.T) {
	registry := buildMockRegistry()
	transformer := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	config := pml_dto.DefaultConfig()

	ctx := newRootTransformationContext(config, 600.0, registry)
	e, ok := transformer.(*engine)
	require.True(t, ok, "Expected transformer to be *engine")

	node := &ast_domain.TemplateNode{TagName: "pml-col"}
	comp := registry.MustGet("pml-col")
	parentNode := &ast_domain.TemplateNode{
		TagName:  "pml-no-stack",
		Children: []*ast_domain.TemplateNode{node},
	}
	parentComp := registry.MustGet("pml-no-stack")

	childCtx := e.createChildContext(ctx, node, comp, parentNode, parentComp)

	assert.True(t, childCtx.IsInsideGroup)
	assert.Equal(t, 1, childCtx.SiblingCount)
}

func TestEngine_LogFirstChildDetails_LongText(t *testing.T) {
	registry := buildMockRegistry()
	transformer := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	e, ok := transformer.(*engine)
	require.True(t, ok, "Expected transformer to be *engine")

	longText := strings.Repeat("a", 100)
	node := &ast_domain.TemplateNode{
		TagName: "pml-p",
		Children: []*ast_domain.TemplateNode{
			{
				NodeType:    ast_domain.NodeText,
				TextContent: longText,
			},
		},
	}

	e.logFirstChildDetails(context.Background(), node)
}

func TestEngine_LogFirstChildDetails_NoChildren(t *testing.T) {
	registry := buildMockRegistry()
	transformer := NewTransformer(registry, newMockMediaQueryCollector(), newMockMSOConditionalCollector())
	e, ok := transformer.(*engine)
	require.True(t, ok, "Expected transformer to be *engine")

	node := &ast_domain.TemplateNode{
		TagName:  "pml-p",
		Children: nil,
	}

	e.logFirstChildDetails(context.Background(), node)
}

type mockMediaQueryCollector struct {
	classes map[string]string
}

func newMockMediaQueryCollector() *mockMediaQueryCollector {
	return &mockMediaQueryCollector{
		classes: make(map[string]string),
	}
}

func (m *mockMediaQueryCollector) RegisterClass(className string, mobileStyles string) {
	m.classes[className] = mobileStyles
}

func (m *mockMediaQueryCollector) RegisterFluidClass(className string, mobileStyles string) {
	m.classes[className] = mobileStyles
}

func (m *mockMediaQueryCollector) GenerateCSS(breakpoint string) string {
	if len(m.classes) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString("@media only screen and (max-width: ")
	result.WriteString(breakpoint)
	result.WriteString(") {\n")
	for className, styles := range m.classes {
		result.WriteString("  .")
		result.WriteString(className)
		result.WriteString(" { ")
		result.WriteString(styles)
		result.WriteString(" }\n")
	}
	result.WriteString("}")
	return result.String()
}

type mockMSOConditionalCollector struct {
	styles map[string]string
}

func newMockMSOConditionalCollector() *mockMSOConditionalCollector {
	return &mockMSOConditionalCollector{
		styles: make(map[string]string),
	}
}

func (m *mockMSOConditionalCollector) RegisterStyle(selector string, styles string) {
	m.styles[selector] = styles
}

func (m *mockMSOConditionalCollector) GenerateConditionalBlock() string {
	if len(m.styles) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString("<!--[if mso]>\n<style type=\"text/css\">\n")
	for selector, styles := range m.styles {
		result.WriteString("  ")
		result.WriteString(selector)
		result.WriteString(" { ")
		result.WriteString(styles)
		result.WriteString(" }\n")
	}
	result.WriteString("</style>\n<![endif]-->")
	return result.String()
}

func parsePML(t *testing.T, pml string) (*ast_domain.TemplateAST, error) {
	t.Helper()
	return ast_domain.Parse(context.Background(), `<template>`+pml+`</template>`, "test.pml", nil)
}

func renderASTToHTML(ast *ast_domain.TemplateAST) string {
	var buffer bytes.Buffer
	for _, node := range ast.RootNodes {
		renderNode(&buffer, node)
	}
	return buffer.String()
}

func renderNode(buffer *bytes.Buffer, node *ast_domain.TemplateNode) {
	if node == nil {
		return
	}

	switch node.NodeType {
	case ast_domain.NodeElement:
		buffer.WriteString("<")
		buffer.WriteString(node.TagName)
		sortedAttrs := make([]ast_domain.HTMLAttribute, len(node.Attributes))
		copy(sortedAttrs, node.Attributes)
		slices.SortFunc(sortedAttrs, func(a, b ast_domain.HTMLAttribute) int {
			return cmp.Compare(a.Name, b.Name)
		})
		for _, attr := range sortedAttrs {
			buffer.WriteString(" ")
			buffer.WriteString(attr.Name)
			buffer.WriteString(`="`)
			buffer.WriteString(attr.Value)
			buffer.WriteString(`"`)
		}
		isVoid := isVoidElement(node.TagName)
		if isVoid && len(node.Children) == 0 {
			buffer.WriteString(" />")
			return
		}
		buffer.WriteString(">")
		for _, child := range node.Children {
			renderNode(buffer, child)
		}
		if !isVoid {
			buffer.WriteString("</")
			buffer.WriteString(node.TagName)
			buffer.WriteString(">")
		}

	case ast_domain.NodeText, ast_domain.NodeRawHTML:
		buffer.WriteString(node.TextContent)
	case ast_domain.NodeComment:
		buffer.WriteString("<!--")
		buffer.WriteString(node.TextContent)
		buffer.WriteString("-->")
	case ast_domain.NodeFragment:
		for _, child := range node.Children {
			renderNode(buffer, child)
		}
	}
}

func normaliseHTML(html string) string {
	s := strings.ReplaceAll(html, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, "> <", "><")
	return s
}

func isVoidElement(tagName string) bool {
	switch tagName {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

type mockComponent struct {
	transformFunc     func(*ast_domain.TemplateNode, *TransformationContext) (*ast_domain.TemplateNode, []*Error)
	allowedAttributes map[string]AttributeDefinition
	defaultAttributes map[string]string
	tagName           string
	allowedParents    []string
	precedence        []AttributeSource
	isEndingTag       bool
}

func (m *mockComponent) TagName() string {
	return m.tagName
}

func (m *mockComponent) IsEndingTag() bool {
	return m.isEndingTag
}

func (m *mockComponent) AllowedParents() []string {
	return m.allowedParents
}

func (m *mockComponent) AllowedAttributes() map[string]AttributeDefinition {
	if m.allowedAttributes == nil {
		return make(map[string]AttributeDefinition)
	}
	return m.allowedAttributes
}

func (m *mockComponent) DefaultAttributes() map[string]string {
	if m.defaultAttributes == nil {
		return make(map[string]string)
	}
	return m.defaultAttributes
}

func (m *mockComponent) GetAttributePrecedence() []AttributeSource {
	if m.precedence == nil {
		return []AttributeSource{SourceDefault, SourceInline}
	}
	return m.precedence
}

func (m *mockComponent) GetStyleTargets() []StyleTarget {
	return nil
}

func (m *mockComponent) Transform(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
	if m.transformFunc != nil {
		return m.transformFunc(node, ctx)
	}
	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "data-pml", Value: m.tagName},
		},
		Children: node.Children,
		Location: node.Location,
	}, nil
}

type mockRegistry struct {
	components map[string]Component
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		components: make(map[string]Component),
	}
}

func (r *mockRegistry) Register(_ context.Context, comp Component) error {
	r.components[comp.TagName()] = comp
	return nil
}

func (r *mockRegistry) Get(tagName string) (Component, bool) {
	comp, found := r.components[tagName]
	return comp, found
}

func (r *mockRegistry) GetAll() []Component {
	result := make([]Component, 0, len(r.components))
	for _, comp := range r.components {
		result = append(result, comp)
	}
	return result
}

func (r *mockRegistry) MustGet(tagName string) Component {
	comp, found := r.components[tagName]
	if !found {
		panic("component not found: " + tagName)
	}
	return comp
}

func buildMockRegistry() ComponentRegistry {
	registry := newMockRegistry()

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-row",
		isEndingTag: false,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "row"},
				},
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-col",
		isEndingTag: false,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "col"},
				},
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-p",
		isEndingTag: true,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "p",
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-img",
		isEndingTag: true,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType:   ast_domain.NodeElement,
				TagName:    "img",
				Attributes: []ast_domain.HTMLAttribute{{Name: "src", Value: "test.jpg"}},
				Location:   node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-button",
		isEndingTag: true,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "button",
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-container",
		isEndingTag: false,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "container"},
				},
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-hero",
		isEndingTag: false,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "hero"},
				},
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-ol",
		isEndingTag: false,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "ol",
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-li",
		isEndingTag: false,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "li",
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-hr",
		isEndingTag: true,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "hr",
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-br",
		isEndingTag: true,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "br",
				Location: node.Location,
			}, nil
		},
	})

	_ = registry.Register(context.Background(), &mockComponent{
		tagName:     "pml-no-stack",
		isEndingTag: false,
		transformFunc: func(node *ast_domain.TemplateNode, ctx *TransformationContext) (*ast_domain.TemplateNode, []*Error) {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "no-stack"},
				},
				Children: node.Children,
				Location: node.Location,
			}, nil
		},
	})

	return registry
}
