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

package driven_code_emitter_go_literal

import (
	"testing"

	goast "go/ast"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestEmitStaticAttributes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		attrs     []ast_domain.HTMLAttribute
		wantCount int
	}{
		{
			name:      "no attributes",
			attrs:     []ast_domain.HTMLAttribute{},
			wantCount: 0,
		},
		{
			name: "single attribute",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "id", Value: "main"},
			},
			wantCount: 1,
		},
		{
			name: "multiple attributes",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "id", Value: "main"},
				{Name: "data-value", Value: "test"},
			},
			wantCount: 2,
		},
		{
			name: "class attribute filtered",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
			wantCount: 0,
		},
		{
			name: "style attribute filtered",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "style", Value: "color: red"},
			},
			wantCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ae := &attributeEmitter{
				emitter:           &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()},
				expressionEmitter: &mockExpressionEmitter{},
			}

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			node.Attributes = tc.attrs
			attributeSliceExpression := &goast.SelectorExpr{X: cachedIdent("node"), Sel: cachedIdent("Attributes")}

			statements := ae.emitStaticAttributes(attributeSliceExpression, node)

			assert.Len(t, statements, tc.wantCount)
		})
	}
}

func TestEmitClassAttribute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		staticClass     string
		dynamicType     string
		wantHelper      string
		wantStmts       int
		hasDynamicClass bool
	}{
		{
			name:        "no class",
			staticClass: "",
			wantStmts:   0,
		},
		{
			name:        "static class only",
			staticClass: "container mx-auto",
			wantStmts:   1,
		},
		{
			name:            "dynamic string only",
			hasDynamicClass: true,
			dynamicType:     "string",
			wantStmts:       2,
			wantHelper:      "ClassesFromStringBytesArena",
		},
		{
			name:            "static + dynamic string",
			staticClass:     "base",
			hasDynamicClass: true,
			dynamicType:     "string",
			wantStmts:       2,
			wantHelper:      "MergeClassesBytesArena",
		},
		{
			name:            "dynamic []string",
			hasDynamicClass: true,
			dynamicType:     "[]string",
			wantHelper:      "ClassesFromSliceBytesArena",
		},
		{
			name:            "dynamic map[string]bool",
			hasDynamicClass: true,
			dynamicType:     "map[string]bool",
			wantHelper:      "MergeClassesBytesArena",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockExpr := &mockExpressionEmitter{
				emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
					return cachedIdent("dynamicClasses"), nil, nil
				},
			}

			ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			if tc.staticClass != "" {
				node.Attributes = []ast_domain.HTMLAttribute{{Name: "class", Value: tc.staticClass}}
			}
			if tc.hasDynamicClass {
				node.DirClass = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name:          "dynamicClasses",
						GoAnnotations: createMockAnnotation(tc.dynamicType, inspector_dto.StringablePrimitive),
					},
				}
			}

			nodeVar := cachedIdent("node")
			attributeSliceExpression := &goast.SelectorExpr{X: nodeVar, Sel: cachedIdent("Attributes")}
			statements, diagnostics := ae.emitClassAttribute(nodeVar, attributeSliceExpression, node)

			assert.Empty(t, diagnostics)

			if tc.wantStmts > 0 {
				assert.GreaterOrEqual(t, len(statements), tc.wantStmts)
			}

			if tc.wantHelper != "" {

				foundHelper := false
				for _, statement := range statements {
					if containsHelperCall(statement, tc.wantHelper) {
						foundHelper = true
						break
					}
				}
				assert.True(t, foundHelper, "Should call %s helper", tc.wantHelper)
			}
		})
	}
}

func TestBuildActionPayload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		expression   ast_domain.Expression
		modifier     string
		wantFuncName string
		wantArgCount int
		wantErr      bool
	}{
		{
			name:         "identifier normalised to CallExpr",
			expression:   &ast_domain.Identifier{Name: "handleClick"},
			modifier:     "action",
			wantFuncName: "handleClick",
			wantArgCount: 0,
		},
		{
			name: "CallExpr with no arguments",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "submitForm"},
				Args:   []ast_domain.Expression{},
			},
			wantFuncName: "submitForm",
			wantArgCount: 0,
		},
		{
			name: "CallExpr with static string argument",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "showAlert"},
				Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "Hello"}},
			},
			wantFuncName: "showAlert",
			wantArgCount: 1,
		},
		{
			name: "CallExpr with dynamic argument",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "updateUser"},
				Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "userId"}},
			},
			wantFuncName: "updateUser",
			wantArgCount: 1,
		},
		{
			name: "invalid expression type",
			expression: &ast_domain.BinaryExpression{
				Operator: ast_domain.OpPlus,
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			modifier: "action",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockExpr := &mockExpressionEmitter{
				emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
					return cachedIdent("dynamicValue"), nil, nil
				},
			}

			ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

			directive := ast_domain.Directive{
				Expression:    tc.expression,
				Modifier:      tc.modifier,
				RawExpression: "",
				Location:      ast_domain.Location{},
			}

			statements, dwVar, bufferPointerVar, diagnostics := ae.buildActionPayload(directive, nil)

			if tc.wantErr {
				assert.NotEmpty(t, diagnostics, "Should have diagnostics for invalid expression")
				return
			}

			assert.Empty(t, diagnostics)
			assert.NotEmpty(t, statements, "Should generate encoding statements")
			assert.NotNil(t, dwVar, "Should return DirectWriter variable")
			assert.NotNil(t, bufferPointerVar, "Should return buffer pointer variable")

			foundEncodePayload := false
			foundGetDirectWriter := false
			for _, statement := range statements {

				if containsCall(statement, "EncodeActionPayloadBytesArena") ||
					containsCall(statement, "EncodeActionPayloadBytes0Arena") ||
					containsCall(statement, "EncodeActionPayloadBytes1Arena") ||
					containsCall(statement, "EncodeActionPayloadBytes2Arena") ||
					containsCall(statement, "EncodeActionPayloadBytes3Arena") ||
					containsCall(statement, "EncodeActionPayloadBytes4Arena") {
					foundEncodePayload = true
				}
				if containsCall(statement, "GetDirectWriter") {
					foundGetDirectWriter = true
				}
			}

			assert.True(t, foundEncodePayload, "Should generate EncodeActionPayloadBytesArena call (fixed-arity or variadic)")
			assert.True(t, foundGetDirectWriter, "Should generate GetDirectWriter call")
		})
	}
}

func TestEmitActionArgValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		argument    ast_domain.Expression
		wantArgType string
		wantPrereqs int
	}{
		{name: "string literal (static)", argument: &ast_domain.StringLiteral{Value: "test"}, wantArgType: "s", wantPrereqs: 0},
		{name: "integer literal (static)", argument: &ast_domain.IntegerLiteral{Value: 42}, wantArgType: "s", wantPrereqs: 0},
		{name: "float literal (static)", argument: &ast_domain.FloatLiteral{Value: 3.14}, wantArgType: "s", wantPrereqs: 0},
		{name: "boolean literal (static)", argument: &ast_domain.BooleanLiteral{Value: true}, wantArgType: "s", wantPrereqs: 0},
		{name: "identifier (dynamic)", argument: &ast_domain.Identifier{Name: "userId"}, wantArgType: "v", wantPrereqs: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockExpr := &mockExpressionEmitter{}
			ae := newAttributeEmitter(&emitter{}, mockExpr)

			argType, value, prereqs, diagnostics := ae.emitActionArgumentValue(tc.argument)

			assert.Equal(t, tc.wantArgType, argType)
			assert.NotNil(t, value)
			assert.Empty(t, diagnostics)

			if tc.wantPrereqs > 0 {
				assert.Len(t, prereqs, tc.wantPrereqs)
			}
		})
	}
}

func TestEmitStyleAttribute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		staticStyle string
		hasDynamic  bool
		hasShow     bool
		wantStmts   int
	}{
		{name: "no style", staticStyle: "", hasDynamic: false, hasShow: false, wantStmts: 0},
		{name: "static only", staticStyle: "color: red", hasDynamic: false, hasShow: false, wantStmts: 1},
		{name: "dynamic only", staticStyle: "", hasDynamic: true, hasShow: false, wantStmts: 2},
		{name: "static + dynamic", staticStyle: "color: red", hasDynamic: true, hasShow: false, wantStmts: 2},
		{name: "with p-show", staticStyle: "color: red", hasDynamic: false, hasShow: true, wantStmts: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockExpr := &mockExpressionEmitter{
				emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
					return cachedIdent("dynamicStyle"), nil, nil
				},
			}

			ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			if tc.staticStyle != "" {
				node.Attributes = []ast_domain.HTMLAttribute{{Name: "style", Value: tc.staticStyle}}
			}
			if tc.hasDynamic {
				node.DirStyle = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{Name: "dynamicStyles"},
				}
			}
			if tc.hasShow {
				node.DirShow = &ast_domain.Directive{
					Expression: &ast_domain.BooleanLiteral{Value: true},
				}
			}

			nodeVar := cachedIdent("node")
			statements, diagnostics := ae.emitStyleAttribute(nodeVar, node)

			assert.Empty(t, diagnostics)

			if tc.wantStmts > 0 {
				assert.GreaterOrEqual(t, len(statements), tc.wantStmts, "Expected at least %d statements", tc.wantStmts)
			}
		})
	}
}

func TestEmitStyleAttribute_TemplateLiteral(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		parts          []ast_domain.TemplateLiteralPart
		expectedHelper string
		mockParts      []goast.Expr
	}{
		{
			name: "3 parts - CSS variable",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "--gradient-colour: var("},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "colour"}},
				{IsLiteral: true, Literal: ")"},
			},
			expectedHelper: "BuildStyleStringBytes3Arena",
			mockParts: []goast.Expr{
				strLit("--gradient-colour: var("),
				cachedIdent("colour"),
				strLit(")"),
			},
		},
		{
			name: "4 parts - two properties",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "color: "},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "fg"}},
				{IsLiteral: true, Literal: "; background: "},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "bg"}},
			},
			expectedHelper: "BuildStyleStringBytes4Arena",
			mockParts: []goast.Expr{
				strLit("color: "),
				cachedIdent("fg"),
				strLit("; background: "),
				cachedIdent("bg"),
			},
		},
		{
			name: "2 parts - simple",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "color: "},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "colour"}},
			},
			expectedHelper: "BuildStyleStringBytes2Arena",
			mockParts: []goast.Expr{
				strLit("color: "),
				cachedIdent("colour"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			template := &ast_domain.TemplateLiteral{Parts: tc.parts}

			mockExpr := &mockExpressionEmitter{
				emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
					if identifier, ok := expression.(*ast_domain.Identifier); ok {
						return cachedIdent(identifier.Name), nil, nil
					}
					return cachedIdent("expr"), nil, nil
				},
				emitTemplateLiteralPartsFunc: func(_ *ast_domain.TemplateLiteral) ([]goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
					return tc.mockParts, nil, nil
				},
			}

			ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			node.DirStyle = &ast_domain.Directive{Expression: template}

			nodeVar := cachedIdent("node")
			statements, diagnostics := ae.emitStyleAttribute(nodeVar, node)

			assert.Empty(t, diagnostics)
			assert.GreaterOrEqual(t, len(statements), 2, "Expected at least 2 statements for dynamic style")

			var foundHelper bool
			for _, statement := range statements {
				if containsCall(statement, tc.expectedHelper) {
					foundHelper = true
					break
				}
			}
			assert.True(t, foundHelper, "Expected to find %s call in generated code", tc.expectedHelper)
		})
	}
}

func TestEmitKeyAttribute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		keyExpr   ast_domain.Expression
		name      string
		wantStmts int
	}{
		{
			name:      "no key",
			keyExpr:   nil,
			wantStmts: 0,
		},
		{
			name:      "static string key",
			keyExpr:   &ast_domain.StringLiteral{Value: "item-1"},
			wantStmts: 1,
		},
		{
			name: "dynamic key expression",
			keyExpr: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "item"},
				Property: &ast_domain.Identifier{Name: "ID"},
			},
			wantStmts: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockExpr := &mockExpressionEmitter{
				emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
					return cachedIdent("keyValue"), nil, nil
				},
				valueToStringFunc: func(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
					return &goast.CallExpr{Fun: &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("Itoa")}}
				},
			}

			ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			if tc.keyExpr != nil {

				node.Key = tc.keyExpr
			}

			nodeVar := cachedIdent("node")
			statements, diagnostics := ae.emitKeyAttribute(nodeVar, node)

			assert.Empty(t, diagnostics)
			assert.Len(t, statements, tc.wantStmts)
		})
	}
}

func TestIsExpressionIntrinsicallySafe(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		want       bool
	}{

		{name: "string literal", expression: &ast_domain.StringLiteral{Value: "test"}, want: true},
		{name: "integer literal", expression: &ast_domain.IntegerLiteral{Value: 42}, want: true},
		{name: "boolean literal", expression: &ast_domain.BooleanLiteral{Value: true}, want: true},

		{name: "identifier (could be user input)", expression: &ast_domain.Identifier{Name: "userId"}, want: false},
		{name: "member expr (could be user data)", expression: &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "user"}, Property: &ast_domain.Identifier{Name: "id"}}, want: false},

		{
			name: "int literal + int literal",
			expression: &ast_domain.BinaryExpression{
				Operator: ast_domain.OpPlus,
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			want: true,
		},

		{
			name: "string literal + unsafe identifier (XSS risk)",
			expression: &ast_domain.BinaryExpression{
				Operator: ast_domain.OpPlus,
				Left:     &ast_domain.StringLiteral{Value: "prefix-"},
				Right: &ast_domain.Identifier{
					Name: "userId",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
						Stringability: int(inspector_dto.StringablePrimitive),
					},
				},
			},
			want: false,
		},

		{
			name: "fmt.Sprintf call (not in safe list)",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "fmt"},
					Property: &ast_domain.Identifier{Name: "Sprintf"},
				},
				Args: []ast_domain.Expression{
					&ast_domain.StringLiteral{Value: "%s"},
					&ast_domain.Identifier{Name: "value"},
				},
			},
			want: false,
		},

		{
			name: "strconv.Itoa call (in safe list)",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "strconv"},
					Property: &ast_domain.Identifier{Name: "Itoa"},
				},
				Args: []ast_domain.Expression{
					&ast_domain.IntegerLiteral{Value: 42},
				},
			},
			want: true,
		},

		{
			name: "strconv.FormatInt call (in safe list)",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "strconv"},
					Property: &ast_domain.Identifier{Name: "FormatInt"},
				},
				Args: []ast_domain.Expression{
					&ast_domain.IntegerLiteral{Value: 123},
					&ast_domain.IntegerLiteral{Value: 10},
				},
			},
			want: true,
		},

		{
			name: "string-typed identifier (unsafe - could contain user input)",
			expression: &ast_domain.Identifier{
				Name: "username",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("string"),
					},
					Stringability: int(inspector_dto.StringablePrimitive),
				},
			},
			want: false,
		},

		{
			name: "rune-typed identifier (unsafe - could contain user input)",
			expression: &ast_domain.Identifier{
				Name: "char",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("rune"),
					},
					Stringability: int(inspector_dto.StringablePrimitive),
				},
			},
			want: false,
		},

		{
			name: "int-typed identifier (safe primitive)",
			expression: &ast_domain.Identifier{
				Name: "count",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("int"),
					},
					Stringability: int(inspector_dto.StringablePrimitive),
				},
			},
			want: true,
		},

		{
			name: "bool-typed identifier (safe primitive)",
			expression: &ast_domain.Identifier{
				Name: "isActive",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
					Stringability: int(inspector_dto.StringablePrimitive),
				},
			},
			want: true,
		},

		{
			name: "unary negation of safe number",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNeg,
				Right:    &ast_domain.IntegerLiteral{Value: 123},
			},
			want: true,
		},
		{
			name: "unary not of safe boolean",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right:    &ast_domain.BooleanLiteral{Value: true},
			},
			want: true,
		},
		{
			name: "unary negation of unsafe string",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNeg,
				Right: &ast_domain.Identifier{
					Name: "userInput",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
						Stringability: int(inspector_dto.StringablePrimitive),
					},
				},
			},
			want: false,
		},

		{
			name: "ternary with safe branches",
			expression: &ast_domain.TernaryExpression{
				Condition:  &ast_domain.BooleanLiteral{Value: true},
				Consequent: &ast_domain.IntegerLiteral{Value: 1},
				Alternate:  &ast_domain.IntegerLiteral{Value: 2},
			},
			want: true,
		},
		{
			name: "ternary with unsafe consequent",
			expression: &ast_domain.TernaryExpression{
				Condition: &ast_domain.BooleanLiteral{Value: true},
				Consequent: &ast_domain.Identifier{
					Name: "userInput",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
						Stringability: int(inspector_dto.StringablePrimitive),
					},
				},
				Alternate: &ast_domain.IntegerLiteral{Value: 0},
			},
			want: false,
		},
		{
			name: "ternary with unsafe alternate",
			expression: &ast_domain.TernaryExpression{
				Condition:  &ast_domain.BooleanLiteral{Value: true},
				Consequent: &ast_domain.IntegerLiteral{Value: 1},
				Alternate: &ast_domain.Identifier{
					Name: "userInput",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
						Stringability: int(inspector_dto.StringablePrimitive),
					},
				},
			},
			want: false,
		},

		{
			name: "template literal with safe parts",
			expression: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "prefix-"},
					{
						IsLiteral: false,
						Expression: &ast_domain.Identifier{
							Name: "count",
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression: cachedIdent("int"),
								},
								Stringability: int(inspector_dto.StringablePrimitive),
							},
						},
					},
					{IsLiteral: true, Literal: "-suffix"},
				},
			},
			want: true,
		},
		{
			name: "template literal with unsafe part",
			expression: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "prefix-"},
					{
						IsLiteral: false,
						Expression: &ast_domain.Identifier{
							Name: "userInput",
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression: cachedIdent("string"),
								},
								Stringability: int(inspector_dto.StringablePrimitive),
							},
						},
					},
				},
			},
			want: false,
		},

		{name: "nil expression", expression: nil, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isExpressionIntrinsicallySafe(tc.expression)

			assert.Equal(t, tc.want, got, "Safety detection for %s", tc.name)
		})
	}
}

func TestEmitEventHandlers_EmptyEvents(t *testing.T) {
	t.Parallel()

	ae := &attributeEmitter{
		emitter:           &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()},
		expressionEmitter: &mockExpressionEmitter{},
	}

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
	}

	statements, diagnostics := ae.emitEventHandlers(cachedIdent("node"), node)

	assert.Empty(t, statements, "Empty event lists should generate no statements")
	assert.Empty(t, diagnostics, "Empty event lists should have no diagnostics")
}

func TestEmitEventHandlers_DirectiveFiltering(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		modifier       string
		shouldGenerate bool
	}{
		{
			name:           "action modifier (should generate)",
			modifier:       "action",
			shouldGenerate: true,
		},
		{
			name:           "helper modifier (should generate)",
			modifier:       "helper",
			shouldGenerate: true,
		},
		{
			name:           "no modifier - standard Go handler (should skip)",
			modifier:       "",
			shouldGenerate: false,
		},
		{
			name:           "prevent modifier (should skip)",
			modifier:       "prevent",
			shouldGenerate: false,
		},
		{
			name:           "stop modifier (should skip)",
			modifier:       "stop",
			shouldGenerate: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockExprEmitter := &mockExpressionEmitter{
				emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {

					return &goast.CallExpr{
						Fun:  cachedIdent("handleClick"),
						Args: []goast.Expr{},
					}, nil, nil
				},
			}

			ae := &attributeEmitter{
				emitter:           &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()},
				expressionEmitter: mockExprEmitter,
			}

			node := &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "button",
				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{
							Type:     ast_domain.DirectiveOn,
							Modifier: tc.modifier,
							Expression: &ast_domain.CallExpression{
								Callee: &ast_domain.Identifier{Name: "handleClick"},
							},
						},
					},
				},
			}

			statements, _ := ae.emitEventHandlers(cachedIdent("node"), node)

			if tc.shouldGenerate {
				assert.NotEmpty(t, statements, "Directive with %q modifier should generate statements", tc.modifier)
			} else {
				assert.Empty(t, statements, "Directive with %q modifier should NOT generate statements (standard Go handler)", tc.modifier)
			}
		})
	}
}

func TestEmitEventHandlers_Sorting(t *testing.T) {
	t.Parallel()

	var processedEvents []string
	mockExprEmitter := &mockExpressionEmitter{
		emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {

			if identifier, ok := expression.(*ast_domain.Identifier); ok {
				processedEvents = append(processedEvents, identifier.Name)
			}
			return cachedIdent("emittedValue"), nil, nil
		},
	}

	ae := &attributeEmitter{
		emitter:           &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()},
		expressionEmitter: mockExprEmitter,
	}

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		OnEvents: map[string][]ast_domain.Directive{
			"submit": {
				{
					Type:     ast_domain.DirectiveOn,
					Modifier: "action",
					Expression: &ast_domain.CallExpression{
						Callee: &ast_domain.Identifier{Name: "handleSubmit"},
						Args: []ast_domain.Expression{
							&ast_domain.Identifier{Name: "submitArg"},
						},
					},
				},
			},
			"click": {
				{
					Type:     ast_domain.DirectiveOn,
					Modifier: "action",
					Expression: &ast_domain.CallExpression{
						Callee: &ast_domain.Identifier{Name: "handleClick"},
						Args: []ast_domain.Expression{
							&ast_domain.Identifier{Name: "clickArg"},
						},
					},
				},
			},
			"blur": {
				{
					Type:     ast_domain.DirectiveOn,
					Modifier: "action",
					Expression: &ast_domain.CallExpression{
						Callee: &ast_domain.Identifier{Name: "handleBlur"},
						Args: []ast_domain.Expression{
							&ast_domain.Identifier{Name: "blurArg"},
						},
					},
				},
			},
		},
	}

	ae.emitEventHandlers(cachedIdent("node"), node)

	expectedOrder := []string{"blurArg", "clickArg", "submitArg"}
	assert.Equal(t, expectedOrder, processedEvents, "Events must be processed in alphabetical order")
}

func TestEmitEventHandlers_OnEventsAndCustomEvents(t *testing.T) {
	t.Parallel()

	mockExprEmitter := &mockExpressionEmitter{
		emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
			return cachedIdent("emittedValue"), nil, nil
		},
	}

	ae := &attributeEmitter{
		emitter:           &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()},
		expressionEmitter: mockExprEmitter,
	}

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		OnEvents: map[string][]ast_domain.Directive{
			"click": {
				{
					Type:     ast_domain.DirectiveOn,
					Modifier: "action",
					Expression: &ast_domain.CallExpression{
						Callee: &ast_domain.Identifier{Name: "handleClick"},
						Args: []ast_domain.Expression{
							&ast_domain.IntegerLiteral{Value: 1},
						},
					},
				},
			},
		},
		CustomEvents: map[string][]ast_domain.Directive{
			"custom-event": {
				{
					Type:     ast_domain.DirectiveEvent,
					Modifier: "helper",
					Expression: &ast_domain.CallExpression{
						Callee: &ast_domain.Identifier{Name: "handleCustom"},
						Args: []ast_domain.Expression{
							&ast_domain.StringLiteral{Value: "test"},
						},
					},
				},
			},
		},
	}

	statements, diagnostics := ae.emitEventHandlers(cachedIdent("node"), node)

	assert.NotEmpty(t, statements, "Should generate statements for both OnEvents and CustomEvents")
	assert.Empty(t, diagnostics, "Should have no diagnostics for valid directives")

	assert.Greater(t, len(statements), 0, "Should generate at least one statement per event")
}

func containsHelperCall(statement goast.Stmt, helperName string) bool {
	var found bool
	goast.Inspect(statement, func(n goast.Node) bool {
		if call, ok := n.(*goast.CallExpr); ok {
			if selectorExpression, ok := call.Fun.(*goast.SelectorExpr); ok {
				if pkg, ok := selectorExpression.X.(*goast.Ident); ok && pkg.Name == "pikoruntime" {
					if selectorExpression.Sel.Name == helperName {
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	return found
}

func containsCall(statement goast.Stmt, functionName string) bool {
	var found bool
	goast.Inspect(statement, func(n goast.Node) bool {
		if call, ok := n.(*goast.CallExpr); ok {
			if selectorExpression, ok := call.Fun.(*goast.SelectorExpr); ok {
				if selectorExpression.Sel.Name == functionName {
					found = true
					return false
				}
			}
			if identifier, ok := call.Fun.(*goast.Ident); ok {
				if identifier.Name == functionName {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

func TestIsBindBooleanType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		directive *ast_domain.Directive
		name      string
		want      bool
	}{
		{
			name: "GoAnnotations with bool ResolvedType",
			directive: &ast_domain.Directive{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
			want: true,
		},
		{
			name: "GoAnnotations with string ResolvedType",
			directive: &ast_domain.Directive{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("string"),
					},
				},
			},
			want: false,
		},
		{
			name: "nil GoAnnotations but Expression has GoAnnotations with bool type",
			directive: &ast_domain.Directive{
				GoAnnotations: nil,
				Expression: &ast_domain.Identifier{
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "nil GoAnnotations and no expression annotations",
			directive: &ast_domain.Directive{
				GoAnnotations: nil,
				Expression:    &ast_domain.Identifier{Name: "something"},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isBindBooleanType(tc.directive)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestShouldSkipDynAttrForWriter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		dynAttr *ast_domain.DynamicAttribute
		name    string
		want    bool
	}{
		{
			name: "class attribute",
			dynAttr: &ast_domain.DynamicAttribute{
				Name: "class",
			},
			want: true,
		},
		{
			name: "style attribute",
			dynAttr: &ast_domain.DynamicAttribute{
				Name: "style",
			},
			want: true,
		},
		{
			name: "src attribute with no bool annotation",
			dynAttr: &ast_domain.DynamicAttribute{
				Name: "src",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("string"),
					},
				},
			},
			want: false,
		},
		{
			name: "disabled attribute with bool type annotation",
			dynAttr: &ast_domain.DynamicAttribute{
				Name: "disabled",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldSkipDynAttrForWriter(tc.dynAttr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSelectBuildClassBytesHelper(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		want      string
		partCount int
	}{
		{name: "partCount 2 returns fixed-arity 2", partCount: 2, want: "BuildClassBytes2Arena"},
		{name: "partCount 4 returns fixed-arity 4", partCount: 4, want: "BuildClassBytes4Arena"},
		{name: "partCount 6 returns fixed-arity 6", partCount: 6, want: "BuildClassBytes6Arena"},
		{name: "partCount 8 returns fixed-arity 8", partCount: 8, want: "BuildClassBytes8Arena"},
		{name: "partCount 1 falls back to variadic", partCount: 1, want: "BuildClassBytesVArena"},
		{name: "partCount 3 falls back to variadic", partCount: 3, want: "BuildClassBytesVArena"},
		{name: "partCount 10 falls back to variadic", partCount: 10, want: "BuildClassBytesVArena"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := selectBuildClassBytesHelper(tc.partCount)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSelectActionEncoderFunc(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		want     string
		argCount int
	}{
		{name: "argCount 0 returns fixed-arity 0", argCount: 0, want: "EncodeActionPayloadBytes0Arena"},
		{name: "argCount 1 returns fixed-arity 1", argCount: 1, want: "EncodeActionPayloadBytes1Arena"},
		{name: "argCount 2 returns fixed-arity 2", argCount: 2, want: "EncodeActionPayloadBytes2Arena"},
		{name: "argCount 3 returns fixed-arity 3", argCount: 3, want: "EncodeActionPayloadBytes3Arena"},
		{name: "argCount 4 returns fixed-arity 4", argCount: 4, want: "EncodeActionPayloadBytes4Arena"},
		{name: "argCount 5 falls back to variadic", argCount: 5, want: "EncodeActionPayloadBytesArena"},
		{name: "argCount 10 falls back to variadic", argCount: 10, want: "EncodeActionPayloadBytesArena"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := selectActionEncoderFunc(tc.argCount)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDirectiveHasClientScript_AttributeEmitter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		directive                 *ast_domain.Directive
		node                      *ast_domain.TemplateNode
		sourcePathHasClientScript map[string]bool
		name                      string
		configHasClientScript     bool
		want                      bool
	}{
		{
			name: "directive has GoAnnotations with OriginalSourcePath in map (true)",
			directive: &ast_domain.Directive{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("comp.pk"),
				},
			},
			node: &ast_domain.TemplateNode{},
			sourcePathHasClientScript: map[string]bool{
				"comp.pk": true,
			},
			configHasClientScript: false,
			want:                  true,
		},
		{
			name:      "directive has no GoAnnotations but node has OriginalSourcePath in map (true)",
			directive: &ast_domain.Directive{},
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("node-comp.pk"),
				},
			},
			sourcePathHasClientScript: map[string]bool{
				"node-comp.pk": true,
			},
			configHasClientScript: false,
			want:                  true,
		},
		{
			name:                      "neither has source path falls back to config HasClientScript true",
			directive:                 &ast_domain.Directive{},
			node:                      &ast_domain.TemplateNode{},
			sourcePathHasClientScript: map[string]bool{},
			configHasClientScript:     true,
			want:                      true,
		},
		{
			name:                      "neither has source path falls back to config HasClientScript false",
			directive:                 &ast_domain.Directive{},
			node:                      &ast_domain.TemplateNode{},
			sourcePathHasClientScript: map[string]bool{},
			configHasClientScript:     false,
			want:                      false,
		},
		{
			name: "directive has source path not in map but node has source path in map",
			directive: &ast_domain.Directive{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("unknown.pk"),
				},
			},
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("node-comp.pk"),
				},
			},
			sourcePathHasClientScript: map[string]bool{
				"node-comp.pk": true,
			},
			configHasClientScript: false,
			want:                  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			em.config.SourcePathHasClientScript = tc.sourcePathHasClientScript
			em.config.HasClientScript = tc.configHasClientScript

			ne := requireNodeEmitter(t, em)
			ae := requireAttributeEmitter(t, ne)

			got := ae.directiveHasClientScript(tc.directive, tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEmitDynamicOnlyClassWriter(t *testing.T) {
	t.Parallel()

	t.Run("TemplateLiteral expression produces BuildClassBytesNArena call", func(t *testing.T) {
		t.Parallel()

		mockParts := []goast.Expr{
			strLit("cls1"),
			strLit("cls2"),
		}

		mockExpr := &mockExpressionEmitter{
			emitTemplateLiteralPartsFunc: func(_ *ast_domain.TemplateLiteral) ([]goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return mockParts, nil, nil
			},
		}

		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		template := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "cls1"},
				{IsLiteral: false, Expression: &ast_domain.StringLiteral{Value: "cls2"}},
			},
		}

		nodeVar := cachedIdent("node")
		statements, diagnostics := ae.emitDynamicOnlyClassWriter(nodeVar, template)

		assert.Empty(t, diagnostics)
		assert.NotEmpty(t, statements, "Should generate statements for template literal class")

		foundHelper := false
		for _, statement := range statements {
			if containsHelperCall(statement, "BuildClassBytes2Arena") {
				foundHelper = true
				break
			}
		}
		assert.True(t, foundHelper, "Should call BuildClassBytes2Arena for 2 parts")
	})

	t.Run("regular Identifier expression produces helper call", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("dynClass"), nil, nil
			},
		}

		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		identifier := &ast_domain.Identifier{
			Name:          "dynClass",
			GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
		}

		nodeVar := cachedIdent("node")
		statements, diagnostics := ae.emitDynamicOnlyClassWriter(nodeVar, identifier)

		assert.Empty(t, diagnostics)
		assert.NotEmpty(t, statements, "Should generate statements for identifier class")

		foundHelper := false
		for _, statement := range statements {
			if containsHelperCall(statement, "ClassesFromStringBytesArena") {
				foundHelper = true
				break
			}
		}
		assert.True(t, foundHelper, "Should call ClassesFromStringBytesArena for string-typed identifier")
	})
}

func TestEmitMergedClassWriter(t *testing.T) {
	t.Parallel()

	t.Run("TemplateLiteral with static prefix produces BuildClassBytesNArena", func(t *testing.T) {
		t.Parallel()

		mockParts := []goast.Expr{
			strLit("cls1"),
			strLit("cls2"),
		}

		mockExpr := &mockExpressionEmitter{
			emitTemplateLiteralPartsFunc: func(_ *ast_domain.TemplateLiteral) ([]goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return mockParts, nil, nil
			},
		}

		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		template := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "cls1"},
				{IsLiteral: false, Expression: &ast_domain.StringLiteral{Value: "cls2"}},
			},
		}

		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitMergedClassWriter(nodeVar, "base", template)

		assert.Empty(t, diagnostics)
		assert.NotEmpty(t, statements, "Should generate statements for merged class")

		foundHelper := false
		for _, statement := range statements {
			if containsHelperCall(statement, "BuildClassBytes4Arena") {
				foundHelper = true
				break
			}
		}
		assert.True(t, foundHelper, "Should call BuildClassBytes4Arena for 4 total parts (static + space + 2 dynamic)")
	})

	t.Run("regular expression produces MergeClassesBytesArena call", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("dynClass"), nil, nil
			},
		}

		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		identifier := &ast_domain.Identifier{
			Name:          "dynClass",
			GoAnnotations: createMockAnnotation("map[string]bool", inspector_dto.StringablePrimitive),
		}

		nodeVar := cachedIdent("node")
		statements, diagnostics := ae.emitMergedClassWriter(nodeVar, "base", identifier)

		assert.Empty(t, diagnostics)
		assert.NotEmpty(t, statements, "Should generate statements for merged class with regular expression")

		foundHelper := false
		for _, statement := range statements {
			if containsHelperCall(statement, "MergeClassesBytesArena") {
				foundHelper = true
				break
			}
		}
		assert.True(t, foundHelper, "Should call MergeClassesBytesArena for non-template-literal expression")
	})
}

func TestEmitDynamicAttributesOnly(t *testing.T) {
	t.Parallel()

	t.Run("no dynamic content produces empty result", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitDynamicAttributesOnly(nodeVar, node)

		assert.Empty(t, statements, "No dynamic content should produce no statements")
		assert.Empty(t, diagnostics, "No dynamic content should produce no diagnostics")
	})

	t.Run("bind directives present produce statements", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("mockValue"), nil, nil
			},
		}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.Binds = map[string]*ast_domain.Directive{
			"title": {
				Expression: &ast_domain.Identifier{
					Name:          "titleVar",
					GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
				},
				GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
			},
		}
		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitDynamicAttributesOnly(nodeVar, node)

		assert.NotEmpty(t, statements, "Bind directives should produce statements")
		assert.Empty(t, diagnostics)
	})

	t.Run("dynamic class and style produce statements", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("dynValue"), nil, nil
			},
		}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.DirClass = &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name:          "dynClass",
				GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
			},
		}
		node.DirStyle = &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name:          "dynStyle",
				GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
			},
		}
		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitDynamicAttributesOnly(nodeVar, node)

		assert.NotEmpty(t, statements, "Dynamic class and style should produce statements")
		assert.Empty(t, diagnostics)
	})
}

func TestEmitBindAttributes(t *testing.T) {
	t.Parallel()

	t.Run("empty binds produce empty result", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitBindAttributes(nodeVar, node)

		assert.Empty(t, statements, "Empty binds should produce no statements")
		assert.Empty(t, diagnostics)
	})

	t.Run("boolean bind is skipped", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		node := createMockTemplateNode(ast_domain.NodeElement, "input", "")
		node.Binds = map[string]*ast_domain.Directive{
			"disabled": {
				Expression: &ast_domain.Identifier{
					Name:          "isDisabled",
					GoAnnotations: createMockAnnotation("bool", inspector_dto.StringablePrimitive),
				},
				GoAnnotations: createMockAnnotation("bool", inspector_dto.StringablePrimitive),
			},
		}
		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitBindAttributes(nodeVar, node)

		assert.Empty(t, statements, "Boolean bind should be skipped and produce no statements")
		assert.Empty(t, diagnostics)
	})

	t.Run("non-boolean bind generates writer statements", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("titleValue"), nil, nil
			},
		}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.Binds = map[string]*ast_domain.Directive{
			"title": {
				Expression: &ast_domain.Identifier{
					Name:          "titleVar",
					GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
				},
				GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
			},
		}
		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitBindAttributes(nodeVar, node)

		assert.NotEmpty(t, statements, "Non-boolean bind should generate writer statements")
		assert.Empty(t, diagnostics)

		foundGetDW := false
		for _, statement := range statements {
			if containsCall(statement, "GetDirectWriter") {
				foundGetDW = true
				break
			}
		}
		assert.True(t, foundGetDW, "Should generate GetDirectWriter call for non-boolean bind")
	})
}

func TestEmitNonClassStyleDynamicAttrs(t *testing.T) {
	t.Parallel()

	t.Run("empty dynamic attributes produce empty result", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitNonClassStyleDynamicAttrs(nodeVar, node)

		assert.Empty(t, statements, "Empty dynamic attributes should produce no statements")
		assert.Empty(t, diagnostics)
	})

	t.Run("class and style dynamic attributes are skipped", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.DynamicAttributes = []ast_domain.DynamicAttribute{
			{
				Name:       "class",
				Expression: &ast_domain.Identifier{Name: "cls"},
			},
			{
				Name:       "style",
				Expression: &ast_domain.Identifier{Name: "stl"},
			},
		}
		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitNonClassStyleDynamicAttrs(nodeVar, node)

		assert.Empty(t, statements, "Class and style dynamic attributes should be skipped")
		assert.Empty(t, diagnostics)
	})

	t.Run("non-class-style non-boolean attribute generates statements", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("hrefValue"), nil, nil
			},
		}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		node := createMockTemplateNode(ast_domain.NodeElement, "a", "")
		node.DynamicAttributes = []ast_domain.DynamicAttribute{
			{
				Name:          "href",
				Expression:    &ast_domain.Identifier{Name: "link"},
				GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
			},
		}
		nodeVar := cachedIdent("node")

		statements, diagnostics := ae.emitNonClassStyleDynamicAttrs(nodeVar, node)

		assert.NotEmpty(t, statements, "Non-class/style non-boolean attribute should generate statements")
		assert.Empty(t, diagnostics)

		foundGetDW := false
		for _, statement := range statements {
			if containsCall(statement, "GetDirectWriter") {
				foundGetDW = true
				break
			}
		}
		assert.True(t, foundGetDW, "Should generate GetDirectWriter call for non-class/style attribute")
	})
}

func TestEmitNonBooleanDynamicAttribute(t *testing.T) {
	t.Parallel()

	t.Run("asset src on piko:img delegates to asset handler", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("imagePath"), nil, nil
			},
		}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		dynAttr := &ast_domain.DynamicAttribute{
			Name:          "src",
			Expression:    &ast_domain.Identifier{Name: "imgSrc"},
			GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
		}

		nodeVar := cachedIdent("node")
		statements, diagnostics := ae.emitNonBooleanDynamicAttribute(nodeVar, dynAttr, "piko:img")

		assert.NotEmpty(t, statements, "Asset src attribute should produce statements")
		assert.Empty(t, diagnostics)

		foundResolve := false
		for _, statement := range statements {
			if containsCall(statement, "ResolveModulePath") {
				foundResolve = true
				break
			}
		}
		assert.True(t, foundResolve, "Should generate ResolveModulePath call for asset src attribute")
	})

	t.Run("standard attribute generates writer statements", func(t *testing.T) {
		t.Parallel()

		mockExpr := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("titleValue"), nil, nil
			},
		}
		ae := newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)

		dynAttr := &ast_domain.DynamicAttribute{
			Name:          "title",
			Expression:    &ast_domain.Identifier{Name: "titleVar"},
			GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
		}

		nodeVar := cachedIdent("node")
		statements, diagnostics := ae.emitNonBooleanDynamicAttribute(nodeVar, dynAttr, "div")

		assert.NotEmpty(t, statements, "Standard attribute should produce statements")
		assert.Empty(t, diagnostics)

		foundGetDW := false
		for _, statement := range statements {
			if containsCall(statement, "GetDirectWriter") {
				foundGetDW = true
				break
			}
		}
		assert.True(t, foundGetDW, "Should generate GetDirectWriter call for standard attribute")

		foundSetName := false
		for _, statement := range statements {
			if containsCall(statement, "SetName") {
				foundSetName = true
				break
			}
		}
		assert.True(t, foundSetName, "Should generate SetName call for standard attribute")
	})
}

func BenchmarkEmitStaticAttributes(b *testing.B) {
	ae := &attributeEmitter{
		emitter:           &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()},
		expressionEmitter: &mockExpressionEmitter{},
	}

	node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
	node.Attributes = []ast_domain.HTMLAttribute{
		{Name: "id", Value: "main"},
		{Name: "data-value", Value: "test"},
	}
	attributeSliceExpression := &goast.SelectorExpr{X: cachedIdent("node"), Sel: cachedIdent("Attributes")}

	b.ResetTimer()
	for b.Loop() {
		_ = ae.emitStaticAttributes(attributeSliceExpression, node)
	}
}

func TestEmitRefAttribute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		dirRef     *ast_domain.Directive
		hashedName string
		wantStmts  int
		wantNilAll bool
	}{
		{
			name:       "nil DirRef returns nil",
			dirRef:     nil,
			hashedName: "",
			wantNilAll: true,
		},
		{
			name: "empty RawExpression returns nil",
			dirRef: &ast_domain.Directive{
				RawExpression: "",
			},
			hashedName: "",
			wantNilAll: true,
		},
		{
			name: "valid ref without HashedName produces one statement",
			dirRef: &ast_domain.Directive{
				RawExpression: "myRef",
			},
			hashedName: "",
			wantStmts:  1,
		},
		{
			name: "valid ref with HashedName produces two statements",
			dirRef: &ast_domain.Directive{
				RawExpression: "myRef",
			},
			hashedName: "abc123",
			wantStmts:  2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockExpr := &mockExpressionEmitter{}
			em := &emitter{config: EmitterConfig{HashedName: tc.hashedName}, ctx: NewEmitterContext()}
			ae := newAttributeEmitter(em, mockExpr)

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			node.DirRef = tc.dirRef

			nodeVar := cachedIdent("node")
			statements, diagnostics := ae.emitRefAttribute(nodeVar, node)

			assert.Empty(t, diagnostics, "emitRefAttribute should never return diagnostics")

			if tc.wantNilAll {
				assert.Nil(t, statements, "Expected nil statements")
				return
			}

			require.Len(t, statements, tc.wantStmts)

			foundPRef := false
			for _, statement := range statements {
				if containsStringLiteral(statement, "p-ref") {
					foundPRef = true
					break
				}
			}
			assert.True(t, foundPRef, "Should contain p-ref attribute")

			if tc.hashedName != "" {
				foundPartial := false
				for _, statement := range statements {
					if containsStringLiteral(statement, "data-pk-partial") {
						foundPartial = true
						break
					}
				}
				assert.True(t, foundPartial, "Should contain data-pk-partial attribute when HashedName is set")
			}
		})
	}
}

func containsStringLiteral(statement goast.Stmt, value string) bool {
	found := false
	goast.Inspect(statement, func(n goast.Node) bool {
		if lit, ok := n.(*goast.BasicLit); ok {
			if lit.Value == `"`+value+`"` {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func TestGetSourcePath(t *testing.T) {
	t.Parallel()

	t.Run("nil GoAnnotations returns empty string", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		ne := requireNodeEmitter(t, em)
		ae := requireAttributeEmitter(t, ne)

		result := ae.getSourcePath(nil)
		assert.Equal(t, "", result)
	})

	t.Run("nil OriginalSourcePath returns empty string", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		ne := requireNodeEmitter(t, em)
		ae := requireAttributeEmitter(t, ne)

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalSourcePath: nil,
		}

		result := ae.getSourcePath(ann)
		assert.Equal(t, "", result)
	})

	t.Run("valid OriginalSourcePath returns computed relative path", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.config.BaseDir = "/project"
		ne := requireNodeEmitter(t, em)
		ae := requireAttributeEmitter(t, ne)

		srcPath := "/project/components/card.pk"
		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalSourcePath: &srcPath,
		}

		result := ae.getSourcePath(ann)
		assert.Equal(t, "components/card.pk", result)
	})
}
