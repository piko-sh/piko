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
	"go/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestShouldResolveModulePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		tagName       string
		attributeName string
		want          bool
	}{
		{
			name:          "piko:img with src returns true",
			tagName:       "piko:img",
			attributeName: "src",
			want:          true,
		},
		{
			name:          "piko:svg with src returns true",
			tagName:       "piko:svg",
			attributeName: "src",
			want:          true,
		},
		{
			name:          "pml-img with src returns true",
			tagName:       "pml-img",
			attributeName: "src",
			want:          true,
		},
		{
			name:          "piko:video with src returns true",
			tagName:       "piko:video",
			attributeName: "src",
			want:          true,
		},
		{
			name:          "piko:img with href returns false",
			tagName:       "piko:img",
			attributeName: "href",
			want:          false,
		},
		{
			name:          "regular img with src returns false",
			tagName:       "img",
			attributeName: "src",
			want:          false,
		},
		{
			name:          "div with src returns false",
			tagName:       "div",
			attributeName: "src",
			want:          false,
		},
		{
			name:          "piko:img with SRC (case insensitive) returns true",
			tagName:       "piko:img",
			attributeName: "SRC",
			want:          true,
		},
		{
			name:          "piko:img with Src (mixed case) returns true",
			tagName:       "piko:img",
			attributeName: "Src",
			want:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldResolveModulePath(tc.tagName, tc.attributeName)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetSpecialisedHelperName(t *testing.T) {
	t.Parallel()

	specificHelpers := map[string]string{
		"int":    "AppendInt",
		"int64":  "AppendInt64",
		"string": "AppendString",
	}

	testCases := []struct {
		name        string
		ann         *ast_domain.GoGeneratorAnnotation
		genericName string
		want        string
	}{
		{
			name:        "nil annotation returns generic",
			ann:         nil,
			genericName: "GenericAppend",
			want:        "GenericAppend",
		},
		{
			name: "nil ResolvedType returns generic",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: nil,
			},
			genericName: "GenericAppend",
			want:        "GenericAppend",
		},
		{
			name:        "int type returns specific helper",
			ann:         createMockAnnotation("int", inspector_dto.StringablePrimitive),
			genericName: "GenericAppend",
			want:        "AppendInt",
		},
		{
			name:        "int64 type returns specific helper",
			ann:         createMockAnnotation("int64", inspector_dto.StringablePrimitive),
			genericName: "GenericAppend",
			want:        "AppendInt64",
		},
		{
			name:        "string type returns specific helper",
			ann:         createMockAnnotation("string", inspector_dto.StringablePrimitive),
			genericName: "GenericAppend",
			want:        "AppendString",
		},
		{
			name:        "unknown type returns generic",
			ann:         createMockAnnotation("float64", inspector_dto.StringablePrimitive),
			genericName: "GenericAppend",
			want:        "GenericAppend",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := getSpecialisedHelperName(tc.ann, tc.genericName, specificHelpers)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildAttributeIfNotEmpty(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	ne := requireNodeEmitter(t, em)
	ae := requireAttributeEmitter(t, ne)

	attributeSlice := cachedIdent("attrs")
	attributeName := "data-value"
	valueVar := cachedIdent("val")

	result := ae.buildAttributeIfNotEmpty(attributeSlice, attributeName, valueVar)

	require.Len(t, result, 1)

	ifStmt, ok := result[0].(*goast.IfStmt)
	require.True(t, ok, "Expected IfStmt")

	binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.NEQ, binaryExpr.Op)
	assert.Equal(t, valueVar, binaryExpr.X)

	emptyString, ok := binaryExpr.Y.(*goast.BasicLit)
	require.True(t, ok)
	assert.Equal(t, `""`, emptyString.Value)

	require.Len(t, ifStmt.Body.List, 1)
}

func TestWrapWithModulePathResolver(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	em.config.ModuleName = "test-module"
	ne := requireNodeEmitter(t, em)
	ae := requireAttributeEmitter(t, ne)

	inputExpr := cachedIdent("path")

	result := ae.wrapWithModulePathResolver(inputExpr)

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok, "Expected CallExpr")

	selector, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, runtimePackageName, selector.X.(*goast.Ident).Name)
	assert.Equal(t, "ResolveModulePath", selector.Sel.Name)

	require.Len(t, callExpr.Args, 2)
	assert.Equal(t, inputExpr, callExpr.Args[0])

	moduleLit, ok := callExpr.Args[1].(*goast.BasicLit)
	require.True(t, ok)
	assert.Equal(t, `"test-module"`, moduleLit.Value)
}

func TestIsExpressionIntrinsicallySafe_Helpers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		want       bool
	}{
		{
			name:       "nil expression is not safe",
			expression: nil,
			want:       false,
		},
		{
			name:       "string literal is safe",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			want:       true,
		},
		{
			name:       "integer literal is safe",
			expression: &ast_domain.IntegerLiteral{Value: 42},
			want:       true,
		},
		{
			name:       "float literal is safe",
			expression: &ast_domain.FloatLiteral{Value: 3.14},
			want:       true,
		},
		{
			name:       "boolean literal is safe",
			expression: &ast_domain.BooleanLiteral{Value: true},
			want:       true,
		},
		{
			name: "binary expr with safe operands is safe",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			want: true,
		},
		{
			name: "unary expr with safe operand is safe",
			expression: &ast_domain.UnaryExpression{
				Operator: ast_domain.OpNot,
				Right:    &ast_domain.BooleanLiteral{Value: true},
			},
			want: true,
		},
		{
			name: "ternary with safe branches is safe",
			expression: &ast_domain.TernaryExpression{
				Condition:  &ast_domain.BooleanLiteral{Value: true},
				Consequent: &ast_domain.IntegerLiteral{Value: 1},
				Alternate:  &ast_domain.IntegerLiteral{Value: 2},
			},
			want: true,
		},
		{
			name: "ternary with unsafe consequent is not safe",
			expression: &ast_domain.TernaryExpression{
				Condition:  &ast_domain.BooleanLiteral{Value: true},
				Consequent: &ast_domain.Identifier{Name: "userInput"},
				Alternate:  &ast_domain.IntegerLiteral{Value: 2},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isExpressionIntrinsicallySafe(tc.expression)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsSafeBinaryExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression *ast_domain.BinaryExpression
		name       string
		want       bool
	}{
		{
			name: "both operands safe",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			want: true,
		},
		{
			name: "left operand unsafe",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "userInput"},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			want: false,
		},
		{
			name: "right operand unsafe",
			expression: &ast_domain.BinaryExpression{
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.Identifier{Name: "userInput"},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isSafeBinaryExpr(tc.expression)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsSafeCallExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression *ast_domain.CallExpression
		name       string
		want       bool
	}{
		{
			name: "strconv.FormatInt is safe",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "strconv"},
					Property: &ast_domain.Identifier{Name: "FormatInt"},
				},
			},
			want: true,
		},
		{
			name: "strconv.Itoa is safe",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "strconv"},
					Property: &ast_domain.Identifier{Name: "Itoa"},
				},
			},
			want: true,
		},
		{
			name: "strconv.FormatFloat is safe",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "strconv"},
					Property: &ast_domain.Identifier{Name: "FormatFloat"},
				},
			},
			want: true,
		},
		{
			name: "strconv.FormatBool is safe",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "strconv"},
					Property: &ast_domain.Identifier{Name: "FormatBool"},
				},
			},
			want: true,
		},
		{
			name: "unknown function is not safe",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "unknown"},
					Property: &ast_domain.Identifier{Name: "Func"},
				},
			},
			want: false,
		},
		{
			name: "non-member callee is not safe",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "someFunc"},
			},
			want: false,
		},
		{
			name: "strconv with unknown method is not safe",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "strconv"},
					Property: &ast_domain.Identifier{Name: "ParseInt"},
				},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isSafeCallExpr(tc.expression)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsSafeStrconvFunction(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		property ast_domain.Expression
		name     string
		want     bool
	}{
		{
			name:     "FormatInt is safe",
			property: &ast_domain.Identifier{Name: "FormatInt"},
			want:     true,
		},
		{
			name:     "FormatUint is safe",
			property: &ast_domain.Identifier{Name: "FormatUint"},
			want:     true,
		},
		{
			name:     "FormatFloat is safe",
			property: &ast_domain.Identifier{Name: "FormatFloat"},
			want:     true,
		},
		{
			name:     "FormatBool is safe",
			property: &ast_domain.Identifier{Name: "FormatBool"},
			want:     true,
		},
		{
			name:     "Itoa is safe",
			property: &ast_domain.Identifier{Name: "Itoa"},
			want:     true,
		},
		{
			name:     "FormatComplex is safe",
			property: &ast_domain.Identifier{Name: "FormatComplex"},
			want:     true,
		},
		{
			name:     "ParseInt is not safe",
			property: &ast_domain.Identifier{Name: "ParseInt"},
			want:     false,
		},
		{
			name:     "non-identifier is not safe",
			property: &ast_domain.StringLiteral{Value: "FormatInt"},
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isSafeStrconvFunction(tc.property)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsSafePrimitiveIdentifier(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		want       bool
	}{
		{
			name: "int identifier is safe",
			expression: &ast_domain.Identifier{
				Name: "x",
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
			name: "string identifier is not safe",
			expression: &ast_domain.Identifier{
				Name: "s",
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
			name: "rune identifier is not safe",
			expression: &ast_domain.Identifier{
				Name: "r",
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
			name: "identifier without annotation is not safe",
			expression: &ast_domain.Identifier{
				Name: "x",
			},
			want: false,
		},
		{
			name: "non-primitive stringability is not safe",
			expression: &ast_domain.Identifier{
				Name: "x",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("int"),
					},
					Stringability: int(inspector_dto.StringableViaStringer),
				},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isSafePrimitiveIdentifier(tc.expression)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsSafeTemplateLiteral(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression *ast_domain.TemplateLiteral
		name       string
		want       bool
	}{
		{
			name: "all literal parts is safe",
			expression: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "hello "},
					{IsLiteral: true, Literal: "world"},
				},
			},
			want: true,
		},
		{
			name: "safe expression part is safe",
			expression: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "count: "},
					{IsLiteral: false, Expression: &ast_domain.IntegerLiteral{Value: 42}},
				},
			},
			want: true,
		},
		{
			name: "unsafe expression part is not safe",
			expression: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "name: "},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "userInput"}},
				},
			},
			want: false,
		},
		{
			name: "empty parts is safe",
			expression: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isSafeTemplateLiteral(tc.expression)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildDirectWriterAttributeIfNotNil(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	ne := requireNodeEmitter(t, em)
	ae := requireAttributeEmitter(t, ne)

	nodeVar := cachedIdent("node")
	attributeName := "p-on:click"
	dwVar := cachedIdent("dw")
	bufferPointerVar := cachedIdent("bufferPointer")

	result := ae.buildDirectWriterAttributeIfNotNil(nodeVar, attributeName, dwVar, bufferPointerVar)

	require.Len(t, result, 1)

	ifStmt, ok := result[0].(*goast.IfStmt)
	require.True(t, ok, "Expected IfStmt")

	binaryExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, token.NEQ, binaryExpr.Op)
	assert.Equal(t, bufferPointerVar, binaryExpr.X)
	assert.Equal(t, "nil", binaryExpr.Y.(*goast.Ident).Name)

	require.Len(t, ifStmt.Body.List, 2)

	expressionStatement, ok := ifStmt.Body.List[0].(*goast.ExprStmt)
	require.True(t, ok)
	setNameCall, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok)
	selector, ok := setNameCall.Fun.(*goast.SelectorExpr)
	require.True(t, ok)
	assert.Equal(t, "SetName", selector.Sel.Name)
}
