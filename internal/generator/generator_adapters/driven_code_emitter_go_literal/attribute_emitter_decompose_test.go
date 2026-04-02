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
	goast "go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDecomposeEmitter(t *testing.T) *attributeEmitter {
	t.Helper()
	em := requireEmitter(t)
	ne := requireNodeEmitter(t, em)
	return requireAttributeEmitter(t, ne)
}

func verifyAppendCall(t *testing.T, statement goast.Stmt, dwVarName, expectedMethod string) {
	t.Helper()
	expressionStatement, ok := statement.(*goast.ExprStmt)
	require.True(t, ok, "expected *goast.ExprStmt, got %T", statement)

	callExpr, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok, "expected call expression")

	selExpr, ok := callExpr.Fun.(*goast.SelectorExpr)
	require.True(t, ok, "expected selector expression")

	xIdent, ok := selExpr.X.(*goast.Ident)
	require.True(t, ok, "expected ident for receiver")
	assert.Equal(t, dwVarName, xIdent.Name)
	assert.Equal(t, expectedMethod, selExpr.Sel.Name)
}

func TestDecomposeGoExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression    goast.Expr
		name          string
		wantMethod    string
		wantStmtCount int
		ctx           decomposeContext
	}{
		{
			name:          "binary_expr_add_decomposes",
			expression:    &goast.BinaryExpr{Op: token.ADD, X: strLit("a"), Y: strLit("b")},
			ctx:           decomposeContextAttribute,
			wantStmtCount: 2,
		},
		{
			name:          "basic_lit_string",
			expression:    &goast.BasicLit{Kind: token.STRING, Value: `"hello"`},
			ctx:           decomposeContextAttribute,
			wantStmtCount: 1,
			wantMethod:    "AppendString",
		},
		{
			name:          "basic_lit_int",
			expression:    &goast.BasicLit{Kind: token.INT, Value: "42"},
			ctx:           decomposeContextAttribute,
			wantStmtCount: 1,
			wantMethod:    "AppendInt",
		},
		{
			name:          "basic_lit_float",
			expression:    &goast.BasicLit{Kind: token.FLOAT, Value: "3.14"},
			ctx:           decomposeContextAttribute,
			wantStmtCount: 1,
			wantMethod:    "AppendFloat",
		},
		{
			name:          "paren_expr_unwraps",
			expression:    &goast.ParenExpr{X: strLit("wrapped")},
			ctx:           decomposeContextAttribute,
			wantStmtCount: 1,
			wantMethod:    "AppendString",
		},
		{
			name:          "ident_uses_escape_string_in_attribute_ctx",
			expression:    cachedIdent("someVar"),
			ctx:           decomposeContextAttribute,
			wantStmtCount: 1,
			wantMethod:    "AppendEscapeString",
		},
		{
			name:          "ident_uses_fnv_string_in_pkey_ctx",
			expression:    cachedIdent("someVar"),
			ctx:           decomposeContextPKey,
			wantStmtCount: 1,
			wantMethod:    "AppendFNVString",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ae := setupDecomposeEmitter(t)
			dwVar := cachedIdent("dw")

			statements := ae.decomposeGoExpr(dwVar, tc.expression, tc.ctx)

			require.Len(t, statements, tc.wantStmtCount)
			if tc.wantMethod != "" && tc.wantStmtCount == 1 {
				verifyAppendCall(t, statements[0], "dw", tc.wantMethod)
			}
		})
	}
}

func TestDecomposeBinaryExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression    *goast.BinaryExpr
		name          string
		wantMethod    string
		wantStmtCount int
		ctx           decomposeContext
	}{
		{
			name: "add_operator_decomposes_both_sides",
			expression: &goast.BinaryExpr{
				Op: token.ADD,
				X:  strLit("prefix"),
				Y:  strLit("suffix"),
			},
			ctx:           decomposeContextAttribute,
			wantStmtCount: 2,
		},
		{
			name: "nested_add_flattens",
			expression: &goast.BinaryExpr{
				Op: token.ADD,
				X: &goast.BinaryExpr{
					Op: token.ADD,
					X:  strLit("a"),
					Y:  strLit("b"),
				},
				Y: strLit("c"),
			},
			ctx:           decomposeContextAttribute,
			wantStmtCount: 3,
		},
		{
			name: "non_add_operator_emits_as_string",
			expression: &goast.BinaryExpr{
				Op: token.MUL,
				X:  cachedIdent("x"),
				Y:  cachedIdent("y"),
			},
			ctx:           decomposeContextAttribute,
			wantStmtCount: 1,
			wantMethod:    "AppendString",
		},
		{
			name: "sub_operator_emits_as_string",
			expression: &goast.BinaryExpr{
				Op: token.SUB,
				X:  cachedIdent("a"),
				Y:  cachedIdent("b"),
			},
			ctx:           decomposeContextAttribute,
			wantStmtCount: 1,
			wantMethod:    "AppendString",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ae := setupDecomposeEmitter(t)
			dwVar := cachedIdent("dw")

			statements := ae.decomposeBinaryExpr(dwVar, tc.expression, tc.ctx)

			require.Len(t, statements, tc.wantStmtCount)
			if tc.wantMethod != "" && tc.wantStmtCount == 1 {
				verifyAppendCall(t, statements[0], "dw", tc.wantMethod)
			}
		})
	}
}

func TestDecomposeBasicLit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		lit        *goast.BasicLit
		wantMethod string
	}{
		{
			name:       "string_literal",
			lit:        &goast.BasicLit{Kind: token.STRING, Value: `"test"`},
			wantMethod: "AppendString",
		},
		{
			name:       "int_literal",
			lit:        &goast.BasicLit{Kind: token.INT, Value: "123"},
			wantMethod: "AppendInt",
		},
		{
			name:       "float_literal",
			lit:        &goast.BasicLit{Kind: token.FLOAT, Value: "1.5"},
			wantMethod: "AppendFloat",
		},
		{
			name:       "char_literal_falls_through_to_string",
			lit:        &goast.BasicLit{Kind: token.CHAR, Value: "'a'"},
			wantMethod: "AppendString",
		},
		{
			name:       "imag_literal_falls_through_to_string",
			lit:        &goast.BasicLit{Kind: token.IMAG, Value: "1i"},
			wantMethod: "AppendString",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ae := setupDecomposeEmitter(t)
			dwVar := cachedIdent("dw")

			statements := ae.decomposeBasicLit(dwVar, tc.lit, decomposeContextAttribute)

			require.Len(t, statements, 1)
			verifyAppendCall(t, statements[0], "dw", tc.wantMethod)
		})
	}
}

func TestDecomposeCallExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		call       *goast.CallExpr
		name       string
		wantMethod string
		ctx        decomposeContext
	}{
		{
			name: "strconv_format_int_optimises",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent("strconv"),
					Sel: cachedIdent("FormatInt"),
				},
				Args: []goast.Expr{cachedIdent("x"), &goast.BasicLit{Kind: token.INT, Value: "10"}},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendInt",
		},
		{
			name: "strconv_itoa_optimises_with_int64_cast",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent("strconv"),
					Sel: cachedIdent("Itoa"),
				},
				Args: []goast.Expr{cachedIdent("n")},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendInt",
		},
		{
			name: "strconv_format_uint_optimises",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent("strconv"),
					Sel: cachedIdent("FormatUint"),
				},
				Args: []goast.Expr{cachedIdent("u"), &goast.BasicLit{Kind: token.INT, Value: "10"}},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendUint",
		},
		{
			name: "strconv_format_float_optimises",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent("strconv"),
					Sel: cachedIdent("FormatFloat"),
				},
				Args: []goast.Expr{cachedIdent("f")},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendFloat",
		},
		{
			name: "strconv_format_float_uses_fnv_in_pkey_ctx",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent("strconv"),
					Sel: cachedIdent("FormatFloat"),
				},
				Args: []goast.Expr{cachedIdent("f")},
			},
			ctx:        decomposeContextPKey,
			wantMethod: "AppendFNVFloat",
		},
		{
			name: "strconv_format_bool_optimises",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cachedIdent("strconv"),
					Sel: cachedIdent("FormatBool"),
				},
				Args: []goast.Expr{cachedIdent("b")},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendBool",
		},
		{
			name: "non_strconv_call_uses_escape_string",
			call: &goast.CallExpr{
				Fun:  cachedIdent("someFunc"),
				Args: []goast.Expr{cachedIdent("x")},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendEscapeString",
		},
		{
			name: "non_strconv_call_uses_fnv_string_in_pkey_ctx",
			call: &goast.CallExpr{
				Fun:  cachedIdent("someFunc"),
				Args: []goast.Expr{cachedIdent("x")},
			},
			ctx:        decomposeContextPKey,
			wantMethod: "AppendFNVString",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ae := setupDecomposeEmitter(t)
			dwVar := cachedIdent("dw")

			statements := ae.decomposeCallExpr(dwVar, tc.call, tc.ctx)

			require.Len(t, statements, 1)
			verifyAppendCall(t, statements[0], "dw", tc.wantMethod)
		})
	}
}

func TestTryEmitStrconvOptimisation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		call       *goast.CallExpr
		name       string
		wantMethod string
		ctx        decomposeContext
		wantNil    bool
	}{
		{
			name: "format_int_returns_append_int",
			call: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("FormatInt")},
				Args: []goast.Expr{cachedIdent("x")},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendInt",
		},
		{
			name: "itoa_converts_to_int64_and_returns_append_int",
			call: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("Itoa")},
				Args: []goast.Expr{cachedIdent("n")},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendInt",
		},
		{
			name: "format_uint_returns_append_uint",
			call: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("FormatUint")},
				Args: []goast.Expr{cachedIdent("u")},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendUint",
		},
		{
			name: "format_float_returns_append_float_in_attr_ctx",
			call: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("FormatFloat")},
				Args: []goast.Expr{cachedIdent("f")},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendFloat",
		},
		{
			name: "format_float_returns_fnv_float_in_pkey_ctx",
			call: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("FormatFloat")},
				Args: []goast.Expr{cachedIdent("f")},
			},
			ctx:        decomposeContextPKey,
			wantMethod: "AppendFNVFloat",
		},
		{
			name: "format_bool_returns_append_bool",
			call: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("FormatBool")},
				Args: []goast.Expr{cachedIdent("b")},
			},
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendBool",
		},
		{
			name: "unknown_strconv_func_returns_nil",
			call: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("UnknownFunc")},
				Args: []goast.Expr{cachedIdent("x")},
			},
			ctx:     decomposeContextAttribute,
			wantNil: true,
		},
		{
			name: "non_strconv_package_returns_nil",
			call: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("fmt"), Sel: cachedIdent("Sprint")},
				Args: []goast.Expr{cachedIdent("x")},
			},
			ctx:     decomposeContextAttribute,
			wantNil: true,
		},
		{
			name: "strconv_with_no_args_returns_nil",
			call: &goast.CallExpr{
				Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("FormatInt")},
				Args: []goast.Expr{},
			},
			ctx:     decomposeContextAttribute,
			wantNil: true,
		},
		{
			name: "plain_function_call_returns_nil",
			call: &goast.CallExpr{
				Fun:  cachedIdent("someFunc"),
				Args: []goast.Expr{cachedIdent("x")},
			},
			ctx:     decomposeContextAttribute,
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ae := setupDecomposeEmitter(t)
			dwVar := cachedIdent("dw")

			statement := ae.tryEmitStrconvOptimisation(dwVar, tc.call, tc.ctx)

			if tc.wantNil {
				assert.Nil(t, statement)
				return
			}

			require.NotNil(t, statement)
			verifyAppendCall(t, statement, "dw", tc.wantMethod)
		})
	}
}

func TestGetStrconvFuncName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		call     *goast.CallExpr
		wantName string
	}{
		{
			name: "strconv_format_int",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("FormatInt")},
			},
			wantName: "FormatInt",
		},
		{
			name: "strconv_itoa",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("Itoa")},
			},
			wantName: "Itoa",
		},
		{
			name: "non_strconv_package_returns_empty",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: cachedIdent("fmt"), Sel: cachedIdent("Sprint")},
			},
			wantName: "",
		},
		{
			name: "plain_func_call_returns_empty",
			call: &goast.CallExpr{
				Fun: cachedIdent("someFunc"),
			},
			wantName: "",
		},
		{
			name: "selector_with_non_ident_x_returns_empty",
			call: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   &goast.CallExpr{Fun: cachedIdent("getPackage")},
					Sel: cachedIdent("Method"),
				},
			},
			wantName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ae := setupDecomposeEmitter(t)

			got := ae.getStrconvFuncName(tc.call)

			assert.Equal(t, tc.wantName, got)
		})
	}
}

func TestEmitDynamicExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		wantMethod string
		ctx        decomposeContext
	}{
		{
			name:       "attribute_context_uses_escape_string",
			ctx:        decomposeContextAttribute,
			wantMethod: "AppendEscapeString",
		},
		{
			name:       "pkey_context_uses_fnv_string",
			ctx:        decomposeContextPKey,
			wantMethod: "AppendFNVString",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ae := setupDecomposeEmitter(t)
			dwVar := cachedIdent("dw")
			expression := cachedIdent("someVar")

			statement := ae.emitDynamicExpr(dwVar, expression, tc.ctx)

			require.NotNil(t, statement)
			verifyAppendCall(t, statement, "dw", tc.wantMethod)
		})
	}
}

func TestEmitAppendInt(t *testing.T) {
	t.Parallel()

	ae := setupDecomposeEmitter(t)
	dwVar := cachedIdent("dw")
	intExpr := cachedIdent("myInt")

	statement := ae.emitAppendInt(dwVar, intExpr)

	require.NotNil(t, statement)
	verifyAppendCall(t, statement, "dw", "AppendInt")

	expressionStatement, ok := statement.(*goast.ExprStmt)
	require.True(t, ok, "expected *goast.ExprStmt")
	callExpr, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok, "expected *goast.CallExpr")
	require.Len(t, callExpr.Args, 1)
	argIdent, ok := callExpr.Args[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "myInt", argIdent.Name)
}

func TestEmitAppendUint(t *testing.T) {
	t.Parallel()

	ae := setupDecomposeEmitter(t)
	dwVar := cachedIdent("dw")
	uintExpr := cachedIdent("myUint")

	statement := ae.emitAppendUint(dwVar, uintExpr)

	require.NotNil(t, statement)
	verifyAppendCall(t, statement, "dw", "AppendUint")

	expressionStatement, ok := statement.(*goast.ExprStmt)
	require.True(t, ok, "expected *goast.ExprStmt")
	callExpr, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok, "expected *goast.CallExpr")
	require.Len(t, callExpr.Args, 1)
	argIdent, ok := callExpr.Args[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "myUint", argIdent.Name)
}

func TestEmitAppendFloat(t *testing.T) {
	t.Parallel()

	ae := setupDecomposeEmitter(t)
	dwVar := cachedIdent("dw")
	floatExpr := cachedIdent("myFloat")

	statement := ae.emitAppendFloat(dwVar, floatExpr)

	require.NotNil(t, statement)
	verifyAppendCall(t, statement, "dw", "AppendFloat")

	expressionStatement, ok := statement.(*goast.ExprStmt)
	require.True(t, ok, "expected *goast.ExprStmt")
	callExpr, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok, "expected *goast.CallExpr")
	require.Len(t, callExpr.Args, 1)
	argIdent, ok := callExpr.Args[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "myFloat", argIdent.Name)
}

func TestEmitAppendBool(t *testing.T) {
	t.Parallel()

	ae := setupDecomposeEmitter(t)
	dwVar := cachedIdent("dw")
	boolExpr := cachedIdent("myBool")

	statement := ae.emitAppendBool(dwVar, boolExpr)

	require.NotNil(t, statement)
	verifyAppendCall(t, statement, "dw", "AppendBool")

	expressionStatement, ok := statement.(*goast.ExprStmt)
	require.True(t, ok, "expected *goast.ExprStmt")
	callExpr, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok, "expected *goast.CallExpr")
	require.Len(t, callExpr.Args, 1)
	argIdent, ok := callExpr.Args[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "myBool", argIdent.Name)
}

func TestEmitAppendString(t *testing.T) {
	t.Parallel()

	ae := setupDecomposeEmitter(t)
	dwVar := cachedIdent("dw")
	strExpr := strLit("hello")

	statement := ae.emitAppendString(dwVar, strExpr)

	require.NotNil(t, statement)
	verifyAppendCall(t, statement, "dw", "AppendString")

	expressionStatement, ok := statement.(*goast.ExprStmt)
	require.True(t, ok, "expected *goast.ExprStmt")
	callExpr, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok, "expected *goast.CallExpr")
	require.Len(t, callExpr.Args, 1)
	argLit, ok := callExpr.Args[0].(*goast.BasicLit)
	require.True(t, ok)
	assert.Equal(t, `"hello"`, argLit.Value)
}

func TestEmitAppendEscapeString(t *testing.T) {
	t.Parallel()

	ae := setupDecomposeEmitter(t)
	dwVar := cachedIdent("dw")
	strExpr := cachedIdent("userInput")

	statement := ae.emitAppendEscapeString(dwVar, strExpr)

	require.NotNil(t, statement)
	verifyAppendCall(t, statement, "dw", "AppendEscapeString")

	expressionStatement, ok := statement.(*goast.ExprStmt)
	require.True(t, ok, "expected *goast.ExprStmt")
	callExpr, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok, "expected *goast.CallExpr")
	require.Len(t, callExpr.Args, 1)
	argIdent, ok := callExpr.Args[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "userInput", argIdent.Name)
}

func TestEmitAppendFNVString(t *testing.T) {
	t.Parallel()

	ae := setupDecomposeEmitter(t)
	dwVar := cachedIdent("dw")
	strExpr := cachedIdent("dynamicKey")

	statement := ae.emitAppendFNVString(dwVar, strExpr)

	require.NotNil(t, statement)
	verifyAppendCall(t, statement, "dw", "AppendFNVString")

	expressionStatement, ok := statement.(*goast.ExprStmt)
	require.True(t, ok, "expected *goast.ExprStmt")
	callExpr, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok, "expected *goast.CallExpr")
	require.Len(t, callExpr.Args, 1)
	argIdent, ok := callExpr.Args[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "dynamicKey", argIdent.Name)
}

func TestEmitAppendFNVFloat(t *testing.T) {
	t.Parallel()

	ae := setupDecomposeEmitter(t)
	dwVar := cachedIdent("dw")
	floatExpr := cachedIdent("precisionFloat")

	statement := ae.emitAppendFNVFloat(dwVar, floatExpr)

	require.NotNil(t, statement)
	verifyAppendCall(t, statement, "dw", "AppendFNVFloat")

	expressionStatement, ok := statement.(*goast.ExprStmt)
	require.True(t, ok, "expected *goast.ExprStmt")
	callExpr, ok := expressionStatement.X.(*goast.CallExpr)
	require.True(t, ok, "expected *goast.CallExpr")
	require.Len(t, callExpr.Args, 1)
	argIdent, ok := callExpr.Args[0].(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "precisionFloat", argIdent.Name)
}

func TestStrconvHandlersIntegration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		functionName string
		wantMethod   string
		wantConvert  bool
		wantIsFloat  bool
	}{
		{
			functionName: "FormatInt",
			wantConvert:  false,
			wantIsFloat:  false,
			wantMethod:   "AppendInt",
		},
		{
			functionName: "Itoa",
			wantConvert:  true,
			wantIsFloat:  false,
			wantMethod:   "AppendInt",
		},
		{
			functionName: "FormatUint",
			wantConvert:  false,
			wantIsFloat:  false,
			wantMethod:   "AppendUint",
		},
		{
			functionName: "FormatFloat",
			wantConvert:  false,
			wantIsFloat:  true,
			wantMethod:   "AppendFloat",
		},
		{
			functionName: "FormatBool",
			wantConvert:  false,
			wantIsFloat:  false,
			wantMethod:   "AppendBool",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.functionName, func(t *testing.T) {
			t.Parallel()

			handler, ok := strconvHandlers[tc.functionName]
			require.True(t, ok, "handler should exist for %s", tc.functionName)

			assert.Equal(t, tc.wantConvert, handler.convertToInt)
			assert.Equal(t, tc.wantIsFloat, handler.isFloat)

			ae := setupDecomposeEmitter(t)
			dwVar := cachedIdent("dw")
			arg := cachedIdent("x")

			statement := handler.emitter(ae, dwVar, arg)
			require.NotNil(t, statement)
			verifyAppendCall(t, statement, "dw", tc.wantMethod)
		})
	}
}
