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

package ast_test

import (
	"errors"
	"flag"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

var update = flag.Bool("update", false, "update golden files")

func goldenFileHelper(t *testing.T, actualContent string, goldenPath ...string) {
	t.Helper()

	fullGoldenPath := filepath.Join(append([]string{"testdata", "compiler"}, goldenPath...)...)

	if *update {

		err := os.MkdirAll(filepath.Dir(fullGoldenPath), 0755)
		require.NoError(t, err, "failed to create directory for golden file")
		err = os.WriteFile(fullGoldenPath, []byte(actualContent), 0644)
		require.NoError(t, err, "failed to update golden file")
		t.Logf("Golden file updated: %s", fullGoldenPath)
		return
	}

	goldenContent, err := os.ReadFile(fullGoldenPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {

			t.Fatalf("Golden file not found. To create it, run: go test -v ./... -update\nPath: %s", fullGoldenPath)
		}
		t.Fatalf("Failed to read golden file: %v", err)
	}

	assert.Equal(t, string(goldenContent), actualContent, "Output does not match golden file: %s", fullGoldenPath)
}

func createComplexTestAST() *ast_domain.TemplateAST {
	sourcePath := "/path/to/source.pkc"
	resolvedType, err := parser.ParseExpr("map[string]int")
	if err != nil {
		panic("test setup failed: could not parse type expression")
	}

	return &ast_domain.TemplateAST{
		SourcePath: &sourcePath,
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Location: ast_domain.Location{Line: 1, Column: 1},
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "id", Value: "main-container", Location: ast_domain.Location{Line: 1, Column: 6}},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:          "class",
						RawExpression: "`card ${theme}`",
						Expression: &ast_domain.TemplateLiteral{Parts: []ast_domain.TemplateLiteralPart{
							{IsLiteral: true, Literal: "card "},
							{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "theme"}},
						}},
						Location: ast_domain.Location{Line: 2, Column: 5},
					},
				},
				DirIf: &ast_domain.Directive{
					Type:          ast_domain.DirectiveIf,
					RawExpression: "user.isActive",
					Expression:    &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "user"}, Property: &ast_domain.Identifier{Name: "isActive"}},
					Location:      ast_domain.Location{Line: 3, Column: 5},
				},
				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{
							Type:          ast_domain.DirectiveOn,
							Arg:           "click",
							Modifier:      "prevent",
							RawExpression: "doSomething(event)",
							Expression:    &ast_domain.CallExpression{Callee: &ast_domain.Identifier{Name: "doSomething"}, Args: []ast_domain.Expression{&ast_domain.Identifier{Name: "event"}}},
							Location:      ast_domain.Location{Line: 4, Column: 5},
						},
					},
				},
				Binds: map[string]*ast_domain.Directive{
					"prop": {
						Type:          ast_domain.DirectiveBind,
						Arg:           "prop",
						RawExpression: "propValue",
						Expression:    &ast_domain.Identifier{Name: "propValue"},
						Location:      ast_domain.Location{Line: 5, Column: 5},
					},
				},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: resolvedType, PackageAlias: "models"},
					Symbol:       &ast_domain.ResolvedSymbol{Name: "MyComponent", ReferenceLocation: ast_domain.Location{Line: 10, Column: 1}},
					NeedsCSRF:    true,
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey: "key123",
						PassedProps: map[string]ast_domain.PropValue{
							"title": {Expression: &ast_domain.StringLiteral{Value: "Dynamic Title"}, Location: ast_domain.Location{Line: 1, Column: 1}},
						},
					},
					OriginalPackageAlias:    new("main"),
					OriginalSourcePath:      &sourcePath,
					DynamicAttributeOrigins: map[string]string{"class": "parent"},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{Line: 6, Column: 7},
						RichText: []ast_domain.TextPart{
							{IsLiteral: true, Literal: "Hello, "},
							{
								IsLiteral:     false,
								RawExpression: "user.name",
								Expression:    &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "user"}, Property: &ast_domain.Identifier{Name: "name"}},
								Location:      ast_domain.Location{Line: 6, Column: 16},
							},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "ul",
						Location: ast_domain.Location{Line: 7, Column: 7},
						DirFor: &ast_domain.Directive{
							Type:          ast_domain.DirectiveFor,
							RawExpression: "(i, item) in items",
							Expression: &ast_domain.ForInExpression{
								IndexVariable: &ast_domain.Identifier{Name: "i"},
								ItemVariable:  &ast_domain.Identifier{Name: "item"},
								Collection:    &ast_domain.Identifier{Name: "items"},
							},
							Location: ast_domain.Location{Line: 7, Column: 11},
						},
						Key: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "item"}, Property: &ast_domain.Identifier{Name: "id"}},
					},
				},
			},
		},
		Diagnostics: []*ast_domain.Diagnostic{
			{
				Message:    "This is a test warning",
				Severity:   ast_domain.Warning,
				Location:   ast_domain.Location{Line: 1, Column: 1},
				Expression: "someExpr",
				SourcePath: "test.pkc",
			},
		},
	}
}

func createUltraComplexTestAST() *ast_domain.TemplateAST {

	pkgAliasModels := "models"
	sourcePath := "/path/to/ultra_complex.pkc"

	typeExprPointerToString, err := parser.ParseExpr("*string")
	if err != nil {
		panic("test setup failed: could not parse type expression for *string")
	}
	typeExprComplexMap, err := parser.ParseExpr("map[string][]pkg.MyType")
	if err != nil {
		panic("test setup failed: could not parse type expression for map")
	}
	typeExprInterface, err := parser.ParseExpr("io.Reader")
	if err != nil {
		panic("test setup failed: could not parse type expression for io.Reader")
	}
	typeExprBuiltin, err := parser.ParseExpr("int")
	if err != nil {
		panic("test setup failed: could not parse type expression for int")
	}

	expressionUserIsActive := &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "user"}, Property: &ast_domain.Identifier{Name: "isActive"}}
	expressionTrue := &ast_domain.BooleanLiteral{Value: true}

	return &ast_domain.TemplateAST{
		SourcePath: &sourcePath,

		Diagnostics: []*ast_domain.Diagnostic{
			{Message: "Root-level diagnostic message.", Severity: ast_domain.Info, Location: ast_domain.Location{Line: 1, Column: 1}},
		},

		RootNodes: []*ast_domain.TemplateNode{

			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Location: ast_domain.Location{Line: 2, Column: 1},

				Attributes: []ast_domain.HTMLAttribute{
					{Name: "id", Value: "kitchen-sink"},
					{Name: "class", Value: "container theme-dark"},
					{Name: "disabled", Value: ""},

					{Name: "data-ação", Value: "teste"},
				},

				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "aria-label", Expression: &ast_domain.StringLiteral{Value: "Main container"}},

					{Name: "data-state", Expression: &ast_domain.TernaryExpression{
						Condition:  expressionUserIsActive,
						Consequent: &ast_domain.StringLiteral{Value: "active"},
						Alternate:  &ast_domain.StringLiteral{Value: "inactive"},
					}},

					{Name: "data-dynamic-string", Expression: &ast_domain.TemplateLiteral{Parts: []ast_domain.TemplateLiteralPart{
						{IsLiteral: true, Literal: "Outer: "},
						{IsLiteral: false, Expression: &ast_domain.TemplateLiteral{Parts: []ast_domain.TemplateLiteralPart{
							{IsLiteral: true, Literal: "Inner: "},
							{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "user"}},
						}}},
					}}},
				},

				DirIf: &ast_domain.Directive{Type: ast_domain.DirectiveIf, Expression: &ast_domain.UnaryExpression{Operator: ast_domain.OpNot, Right: &ast_domain.Identifier{Name: "isLoading"}}},

				DirShow: &ast_domain.Directive{Type: ast_domain.DirectiveShow, Expression: &ast_domain.Identifier{Name: "shouldDisplay"}},

				DirClass: &ast_domain.Directive{Type: ast_domain.DirectiveClass, Expression: &ast_domain.ObjectLiteral{Pairs: map[string]ast_domain.Expression{
					"is-active":   expressionUserIsActive,
					"has-warning": &ast_domain.TernaryExpression{Condition: &ast_domain.Identifier{Name: "errorCount"}, Consequent: expressionTrue, Alternate: &ast_domain.BooleanLiteral{Value: false}},
				}}},

				DirStyle: &ast_domain.Directive{Type: ast_domain.DirectiveStyle, Expression: &ast_domain.ObjectLiteral{Pairs: map[string]ast_domain.Expression{
					"color":      &ast_domain.Identifier{Name: "fontColor"},
					"fontSize":   &ast_domain.StringLiteral{Value: "16px"},
					"background": &ast_domain.StringLiteral{Value: "blue"},
				}}},

				Binds: map[string]*ast_domain.Directive{
					"data-user-id": {Type: ast_domain.DirectiveBind, Arg: "data-user-id", Expression: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "user"}, Property: &ast_domain.Identifier{Name: "id"}}},
					"title":        {Type: ast_domain.DirectiveBind, Arg: "title", Expression: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "page"}, Property: &ast_domain.Identifier{Name: "title"}}},
				},

				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{Type: ast_domain.DirectiveOn, Arg: "click", Expression: &ast_domain.Identifier{Name: "handleClick"}},
						{Type: ast_domain.DirectiveOn, Arg: "click", Expression: &ast_domain.Identifier{Name: "trackClick"}},
					},
					"submit": {
						{Type: ast_domain.DirectiveOn, Arg: "submit", Modifier: "prevent", Expression: &ast_domain.Identifier{Name: "handleSubmit"}},
					},
				},

				CustomEvents: map[string][]ast_domain.Directive{
					"custom-update": {{Type: ast_domain.DirectiveEvent, Arg: "custom-update", Expression: &ast_domain.Identifier{Name: "onCustomUpdate"}}},
				},

				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType:         &ast_domain.ResolvedTypeInfo{TypeExpression: typeExprComplexMap, PackageAlias: pkgAliasModels},
					Symbol:               &ast_domain.ResolvedSymbol{Name: "KitchenSinkComponent", ReferenceLocation: ast_domain.Location{Line: 100, Column: 1}},
					NeedsCSRF:            true,
					OriginalPackageAlias: new("main"),
					OriginalSourcePath:   &sourcePath,
					DynamicAttributeOrigins: map[string]string{
						"data-state": "parent-component",
						"aria-label": "local",
					},
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "partial-1",
						PartialAlias:        "widgets",
						PartialPackageName:  "github.com/user/widgets",
						InvokerPackageAlias: "main",
						RequestOverrides: map[string]ast_domain.PropValue{
							"override_prop": {Expression: expressionTrue, Location: ast_domain.Location{Line: 1, Column: 1}},
						},
						PassedProps: map[string]ast_domain.PropValue{
							"passed_prop": {Expression: &ast_domain.StringLiteral{Value: "value"}, Location: ast_domain.Location{Line: 1, Column: 1}},
						},
					},
				},

				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement, TagName: "div",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement, TagName: "div",
								Children: []*ast_domain.TemplateNode{
									{
										NodeType: ast_domain.NodeElement, TagName: "p",

										DirIf: &ast_domain.Directive{Type: ast_domain.DirectiveIf, Expression: &ast_domain.Identifier{Name: "showParagraph"}},
										Children: []*ast_domain.TemplateNode{
											{NodeType: ast_domain.NodeElement, TagName: "span",

												Children: []*ast_domain.TemplateNode{
													{NodeType: ast_domain.NodeText, RichText: []ast_domain.TextPart{
														{IsLiteral: true, Literal: "Link: "},
														{IsLiteral: false, Expression: &ast_domain.TemplateLiteral{Parts: []ast_domain.TemplateLiteralPart{
															{IsLiteral: true, Literal: "/users/"},
															{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "userId"}},
														}}},
													}},
												}},
										},
									},
								},
							},
						},
					},

					{NodeType: ast_domain.NodeComment, TextContent: " this is a comment "},

					{NodeType: ast_domain.NodeElement, TagName: "section", Directives: []ast_domain.Directive{{Type: ast_domain.DirectiveIf, RawExpression: ""}}},
				},
			},

			{
				NodeType: ast_domain.NodeFragment,
				Location: ast_domain.Location{Line: 20, Column: 1},
				Children: []*ast_domain.TemplateNode{

					{
						NodeType: ast_domain.NodeElement, TagName: "template",
						DirFor: &ast_domain.Directive{Type: ast_domain.DirectiveFor, Expression: &ast_domain.ForInExpression{ItemVariable: &ast_domain.Identifier{Name: "item"}, Collection: &ast_domain.Identifier{Name: "items"}}},
						Children: []*ast_domain.TemplateNode{
							{NodeType: ast_domain.NodeElement, TagName: "h2", Children: []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeText, TextContent: "Title"}}},
							{NodeType: ast_domain.NodeElement, TagName: "p", Children: []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeText, TextContent: "Content"}}},
						},
					},

					{
						NodeType: ast_domain.NodeElement, TagName: "template",
						DirIf: &ast_domain.Directive{Type: ast_domain.DirectiveIf, Expression: &ast_domain.Identifier{Name: "condition"}},
					},
					{
						NodeType: ast_domain.NodeElement, TagName: "div",
						DirElse: &ast_domain.Directive{Type: ast_domain.DirectiveElse},
					},

					{
						NodeType: ast_domain.NodeElement, TagName: "div", DirIf: &ast_domain.Directive{Type: ast_domain.DirectiveIf, Expression: &ast_domain.Identifier{Name: "c1"}},
					},
					{
						NodeType:   ast_domain.NodeElement,
						TagName:    "div",
						DirElseIf:  &ast_domain.Directive{Type: ast_domain.DirectiveElseIf, Expression: &ast_domain.Identifier{Name: "c2"}},
						DirFor:     &ast_domain.Directive{Type: ast_domain.DirectiveFor, Expression: &ast_domain.ForInExpression{ItemVariable: &ast_domain.Identifier{Name: "i"}, Collection: &ast_domain.Identifier{Name: "items"}}},
						DirKey:     &ast_domain.Directive{Type: ast_domain.DirectiveKey, Expression: &ast_domain.Identifier{Name: "i"}},
						DirContext: &ast_domain.Directive{Type: ast_domain.DirectiveContext, Expression: &ast_domain.StringLiteral{Value: "loop"}},
					},
				},
			},

			{
				NodeType: ast_domain.NodeElement, TagName: "input",
				Location: ast_domain.Location{Line: 30, Column: 1},

				DirModel: &ast_domain.Directive{Type: ast_domain.DirectiveModel, Expression: &ast_domain.Identifier{Name: "form.value"}},

				DirKey: &ast_domain.Directive{Type: ast_domain.DirectiveKey, Expression: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "form"}, Property: &ast_domain.Identifier{Name: "id"}}},

				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: typeExprBuiltin, PackageAlias: ""},
					Symbol:       &ast_domain.ResolvedSymbol{Name: "formInput", ReferenceLocation: ast_domain.Location{Line: 0, Column: 0}},

					OriginalPackageAlias: nil,
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{

					{Name: "data-calc", Expression: &ast_domain.BinaryExpression{
						Left:     &ast_domain.BinaryExpression{Left: &ast_domain.Identifier{Name: "a"}, Operator: ast_domain.OpMinus, Right: &ast_domain.Identifier{Name: "b"}},
						Operator: ast_domain.OpMinus, Right: &ast_domain.Identifier{Name: "c"},
					}},

					{Name: "data-for", Expression: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "data"}, Property: &ast_domain.Identifier{Name: "for"}}},
				},
			},

			{NodeType: ast_domain.NodeComment, TextContent: " Final Section "},

			{NodeType: ast_domain.NodeText, RichText: []ast_domain.TextPart{

				{IsLiteral: false, RawExpression: "  finalMessage  ", Expression: &ast_domain.Identifier{Name: "finalMessage"}},
			}},

			{
				NodeType: ast_domain.NodeElement, TagName: "textarea",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "  Line 1\n  <span>not a tag</span>"},
				},
			},

			{
				NodeType: ast_domain.NodeElement, TagName: "p",
				Location: ast_domain.Location{Line: 40, Column: 1},
				DirText:  &ast_domain.Directive{Type: ast_domain.DirectiveText, Expression: &ast_domain.Identifier{Name: "optionalMessage"}},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: typeExprPointerToString, PackageAlias: ""},
					Symbol:       &ast_domain.ResolvedSymbol{Name: "optionalMessage"},
				},
			},

			{
				NodeType: ast_domain.NodeElement, TagName: "span",
				Location: ast_domain.Location{Line: 41, Column: 1},

				Directives: []ast_domain.Directive{
					{
						Type: ast_domain.DirectiveOn, Arg: "load", Expression: &ast_domain.Identifier{Name: "loadData"},

						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: typeExprInterface, PackageAlias: "io"},
							Symbol:       &ast_domain.ResolvedSymbol{Name: "dataSource"},

							PartialInfo: &ast_domain.PartialInvocationInfo{},
						},
					},
				},

				OnEvents: map[string][]ast_domain.Directive{
					"load": {
						{
							Type: ast_domain.DirectiveOn, Arg: "load", Expression: &ast_domain.Identifier{Name: "loadData"},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: typeExprInterface, PackageAlias: "io"},
								Symbol:       &ast_domain.ResolvedSymbol{Name: "dataSource"},
								PartialInfo:  &ast_domain.PartialInvocationInfo{},
							},
						},
					},
				},
			},
		},
	}
}

func TestASTCompilation(t *testing.T) {

	simpleNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "p",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "class", Value: "simple"},
		},
		DirShow: &ast_domain.Directive{
			Type:       ast_domain.DirectiveShow,
			Expression: &ast_domain.Identifier{Name: "isVisible"},
		},
	}

	testCases := []struct {
		generator        func() string
		goFileGenerator  func() string
		name             string
		goldenFilename   string
		goGoldenFilename string
		isHardcoded      bool
	}{
		{
			name:             "Compile a complex AST to string",
			goldenFilename:   "complex_ast_domain.golden",
			goGoldenFilename: "complex_ast_domain.golden.go",
			generator: func() string {
				complexAST := createComplexTestAST()
				return ast_domain.SerialiseASTString(complexAST)
			},
			goFileGenerator: func() string {
				complexAST := createComplexTestAST()
				return ast_domain.SerialiseASTToGoFileContent(complexAST, "testgolden")
			},
		},
		{
			name:             "Compile a simple node to string",
			goldenFilename:   "simple_node.golden",
			goGoldenFilename: "simple_node.golden.go",
			generator: func() string {
				return ast_domain.SerialiseNodeString(simpleNode)
			},
			goFileGenerator: func() string {
				ultraComplexAST := createUltraComplexTestAST()
				return ast_domain.SerialiseASTToGoFileContent(ultraComplexAST, "testgolden")
			},
		},
		{
			name:        "Compile a nil AST",
			generator:   func() string { return ast_domain.SerialiseASTString(nil) },
			isHardcoded: true,
		},
		{
			name:        "Compile a nil node",
			generator:   func() string { return ast_domain.SerialiseNodeString(nil) },
			isHardcoded: true,
		},
		{
			name:             "Compile an ultra-complex AST to string",
			goldenFilename:   "ultra_complex_ast_domain.golden",
			goGoldenFilename: "ultra_complex_ast_domain.golden.go",
			generator: func() string {
				ultraComplexAST := createUltraComplexTestAST()
				return ast_domain.SerialiseASTString(ultraComplexAST)
			},
			goFileGenerator: func() string {
				ultraComplexAST := createUltraComplexTestAST()
				return ast_domain.SerialiseASTToGoFileContent(ultraComplexAST, "testgolden")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			generatedCode := tc.generator()

			if tc.isHardcoded {
				if strings.Contains(tc.name, "nil AST") {
					assert.Equal(t, "/* AST is nil */", generatedCode)
				} else if strings.Contains(tc.name, "nil node") {
					assert.Equal(t, "/* Node is nil */", generatedCode)
				}
				return
			}

			require.NotEmpty(t, generatedCode, "Generator function produced empty output")

			_, err := parser.ParseExpr(generatedCode)
			require.NoError(t, err, "The generated Go code is not a valid expression.\nGenerated code:\n%s", generatedCode)

			goldenFileHelper(t, generatedCode, t.Name(), tc.goldenFilename)

			if tc.goFileGenerator != nil {
				generatedGoFileContent := tc.goFileGenerator()
				require.NotEmpty(t, generatedGoFileContent, "Go file generator produced empty output")

				_, err := parser.ParseFile(token.NewFileSet(), "", generatedGoFileContent, parser.AllErrors)
				require.NoError(t, err, "The generated Go file content is not valid Go source code.\nContent:\n%s", generatedGoFileContent)

				goldenFileHelper(t, generatedGoFileContent, t.Name(), tc.goGoldenFilename)
			}
		})
	}
}
