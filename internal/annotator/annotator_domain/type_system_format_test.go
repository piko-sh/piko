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
	"testing"

	goast "go/ast"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
)

func TestValidateFormatFuncArgs_WrongArgCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		argCount int
	}{
		{name: "zero arguments", argCount: 0},
		{name: "two arguments", argCount: 2},
		{name: "three arguments", argCount: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createTypeSystemTestContext()
			tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{}}

			arguments := make([]ast_domain.Expression, tc.argCount)
			for i := range arguments {
				arguments[i] = &ast_domain.Identifier{Name: "x"}
			}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "F"},
				Args:   arguments,
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := make([]*ast_domain.GoGeneratorAnnotation, tc.argCount)

			tr.validateFormatFuncArgs(ctx, callExpr, argAnns, baseLocation)

			require.Len(t, *ctx.Diagnostics, 1)
			assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
			assert.Contains(t, (*ctx.Diagnostics)[0].Message, "F/LF expects exactly one argument")
		})
	}
}

func TestValidateFormatFuncArgs_ValidTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		typeExpr     goast.Expr
		packageAlias string
	}{
		{name: "int", typeExpr: goast.NewIdent("int"), packageAlias: ""},
		{name: "float64", typeExpr: goast.NewIdent("float64"), packageAlias: ""},
		{name: "string", typeExpr: goast.NewIdent("string"), packageAlias: ""},
		{name: "bool", typeExpr: goast.NewIdent("bool"), packageAlias: ""},
		{name: "Decimal", typeExpr: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("Decimal")}, packageAlias: "maths"},
		{name: "BigInt", typeExpr: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("BigInt")}, packageAlias: "maths"},
		{name: "Money", typeExpr: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("Money")}, packageAlias: "maths"},
		{name: "time.Time", typeExpr: &goast.SelectorExpr{X: goast.NewIdent("time"), Sel: goast.NewIdent("Time")}, packageAlias: "time"},
		{name: "pointer to Decimal", typeExpr: &goast.StarExpr{X: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("Decimal")}}, packageAlias: "maths"},
		{name: "unresolved (empty)", typeExpr: goast.NewIdent(""), packageAlias: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createTypeSystemTestContext()
			tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{}}

			argument := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "F"},
				Args:   []ast_domain.Expression{argument},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: tc.typeExpr, PackageAlias: tc.packageAlias}},
			}

			tr.validateFormatFuncArgs(ctx, callExpr, argAnns, baseLocation)

			assert.Empty(t, *ctx.Diagnostics, "expected no diagnostics for formattable type %s", tc.name)
		})
	}
}

func TestValidateFormatFuncArgs_UnsupportedType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		typeExpr     goast.Expr
		packageAlias string
	}{
		{name: "slice of string", typeExpr: &goast.ArrayType{Elt: goast.NewIdent("string")}, packageAlias: ""},
		{name: "map", typeExpr: &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")}, packageAlias: ""},
		{name: "custom struct", typeExpr: goast.NewIdent("SomeStruct"), packageAlias: ""},
		{name: "chan", typeExpr: &goast.ChanType{Dir: goast.SEND | goast.RECV, Value: goast.NewIdent("int")}, packageAlias: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createTypeSystemTestContext()
			tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{}}

			argument := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "F"},
				Args:   []ast_domain.Expression{argument},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: tc.typeExpr, PackageAlias: tc.packageAlias}},
			}

			tr.validateFormatFuncArgs(ctx, callExpr, argAnns, baseLocation)

			require.Len(t, *ctx.Diagnostics, 1)
			assert.Equal(t, ast_domain.Warning, (*ctx.Diagnostics)[0].Severity)
			assert.Contains(t, (*ctx.Diagnostics)[0].Message, "may not be formattable")
		})
	}
}

func TestGetFormatFuncReturnType(t *testing.T) {
	t.Parallel()

	result := getFormatFuncReturnType(nil, nil, nil, nil)

	require.NotNil(t, result)

	starExpr, ok := result.TypeExpression.(*goast.StarExpr)
	require.True(t, ok, "expected *StarExpr, got %T", result.TypeExpression)

	selExpr, ok := starExpr.X.(*goast.SelectorExpr)
	require.True(t, ok, "expected SelectorExpr inside star, got %T", starExpr.X)

	pkgIdent, ok := selExpr.X.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "i18n_domain", pkgIdent.Name)
	assert.Equal(t, "FormatBuilder", selExpr.Sel.Name)

	assert.Equal(t, "i18n_domain", result.PackageAlias)
	assert.Equal(t, "piko.sh/piko/internal/i18n/i18n_domain", result.CanonicalPackagePath)
}

func TestIsFormattableType(t *testing.T) {
	t.Parallel()

	formattable := []string{
		"int", "int64", "float64", "string", "bool",
		"Decimal", "maths.Decimal", "*maths.Decimal",
		"BigInt", "Money", "Time", "time.Time",
		"DateTime", "Duration",
		"", "interface{}", "any",
	}

	for _, typeName := range formattable {
		assert.True(t, isFormattableType(typeName), "expected %q to be formattable", typeName)
	}

	notFormattable := []string{
		"[]string", "map[string]int", "SomeStruct", "chan int",
		"MyCustomType", "func()",
	}

	for _, typeName := range notFormattable {
		assert.False(t, isFormattableType(typeName), "expected %q to NOT be formattable", typeName)
	}
}
