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
)

func TestDefineAndAssign(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		rightHandSide goast.Expr
		name          string
		varName       string
	}{
		{name: "simple ident", varName: "x", rightHandSide: cachedIdent("someValue")},
		{name: "temp var", varName: "tempVar1", rightHandSide: &goast.CallExpr{Fun: cachedIdent("GetNode")}},
		{name: "complex expr", varName: "result", rightHandSide: &goast.BinaryExpr{X: cachedIdent("a"), Op: token.ADD, Y: cachedIdent("b")}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := defineAndAssign(tc.varName, tc.rightHandSide)

			assert.Equal(t, token.DEFINE, result.Tok, "should use := operator")
			require.Len(t, result.Lhs, 1)
			assert.Equal(t, tc.varName, result.Lhs[0].(*goast.Ident).Name)
			require.Len(t, result.Rhs, 1)
			assert.Equal(t, tc.rightHandSide, result.Rhs[0])
		})
	}
}

func TestAssignExpression(t *testing.T) {
	t.Parallel()

	result := assignExpression("existing", cachedIdent("newValue"))

	assert.Equal(t, token.ASSIGN, result.Tok, "should use = operator")
	assert.Equal(t, "existing", result.Lhs[0].(*goast.Ident).Name)
	assert.Equal(t, "newValue", result.Rhs[0].(*goast.Ident).Name)
}

func TestAppendToSlice(t *testing.T) {
	t.Parallel()

	sliceExpr := &goast.SelectorExpr{X: cachedIdent("node"), Sel: cachedIdent("Children")}
	elemExpr := cachedIdent("childNode")

	result := appendToSlice(sliceExpr, elemExpr)

	assert.Equal(t, token.ASSIGN, result.Tok)
	assert.Equal(t, sliceExpr, result.Lhs[0])

	callExpr, ok := result.Rhs[0].(*goast.CallExpr)
	require.True(t, ok, "RHS should be CallExpr")
	assert.Equal(t, "append", callExpr.Fun.(*goast.Ident).Name)
	require.Len(t, callExpr.Args, 2)
	assert.Equal(t, sliceExpr, callExpr.Args[0])
	assert.Equal(t, elemExpr, callExpr.Args[1])
}

func TestStrLit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple string", input: "hello", want: `"hello"`},
		{name: "empty string", input: "", want: `""`},
		{name: "with quotes", input: `he"llo`, want: `"he\"llo"`},
		{name: "with backslash", input: `hel\lo`, want: `"hel\\lo"`},
		{name: "with newline", input: "hel\nlo", want: `"hel\nlo"`},
		{name: "with tab", input: "hel\tlo", want: `"hel\tlo"`},
		{name: "unicode", input: "hello 世界", want: `"hello 世界"`},
		{name: "emoji", input: "hello 🎉", want: `"hello 🎉"`},
		{name: "HTML special chars", input: "<div>&amp;</div>", want: `"<div>&amp;</div>"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := strLit(tc.input)

			assert.Equal(t, token.STRING, result.Kind)
			assert.Equal(t, tc.want, result.Value)
		})
	}
}

func TestIntLit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		want  string
		input int
	}{
		{name: "zero", input: 0, want: "0"},
		{name: "positive", input: 42, want: "42"},
		{name: "negative", input: -10, want: "-10"},
		{name: "large", input: 1000000, want: "1000000"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := intLit(tc.input)

			assert.Equal(t, token.INT, result.Kind)
			assert.Equal(t, tc.want, result.Value)
		})
	}
}

func TestCallHelper(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		functionName string
		arguments    []goast.Expr
		wantArgCount int
	}{
		{name: "no arguments", functionName: "GetNode", arguments: []goast.Expr{}, wantArgCount: 0},
		{name: "one argument", functionName: "ValueToString", arguments: []goast.Expr{cachedIdent("x")}, wantArgCount: 1},
		{name: "two arguments", functionName: "EvaluateEquality", arguments: []goast.Expr{cachedIdent("a"), cachedIdent("b")}, wantArgCount: 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := callHelper(tc.functionName, tc.arguments...)

			selector, ok := result.Fun.(*goast.SelectorExpr)
			require.True(t, ok, "expected SelectorExpr")
			assert.Equal(t, "pikoruntime", selector.X.(*goast.Ident).Name)
			assert.Equal(t, tc.functionName, selector.Sel.Name)
			assert.Len(t, result.Args, tc.wantArgCount)
		})
	}
}

func TestWrapInTruthinessCall(t *testing.T) {
	t.Parallel()

	expression := cachedIdent("someValue")
	result := wrapInTruthinessCall(expression)

	callExpr := requireCallExpr(t, result, "wrapInTruthinessCall result")

	selector := requireSelectorExpr(t, callExpr.Fun, "truthiness call function")
	selectorX := requireIdent(t, selector.X, "selector X")
	assert.Equal(t, "pikoruntime", selectorX.Name)
	assert.Equal(t, "EvaluateTruthiness", selector.Sel.Name)
	require.Len(t, callExpr.Args, 1)
	assert.Equal(t, expression, callExpr.Args[0])
}

func TestGetZeroValueExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr    goast.Expr
		name        string
		wantValue   string
		wantIsIdent bool
		wantIsLit   bool
		wantIsComp  bool
	}{

		{name: "nil type", typeExpr: nil, wantIsIdent: true, wantValue: "nil"},

		{name: "*int pointer", typeExpr: &goast.StarExpr{X: cachedIdent("int")}, wantIsIdent: true, wantValue: "nil"},
		{name: "[]int slice", typeExpr: &goast.ArrayType{Len: nil, Elt: cachedIdent("int")}, wantIsIdent: true, wantValue: "nil"},
		{name: "map[string]int", typeExpr: &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")}, wantIsIdent: true, wantValue: "nil"},
		{name: "func()", typeExpr: &goast.FuncType{Params: &goast.FieldList{}}, wantIsIdent: true, wantValue: "nil"},
		{name: "chan int", typeExpr: &goast.ChanType{Value: cachedIdent("int")}, wantIsIdent: true, wantValue: "nil"},
		{name: "interface{}", typeExpr: &goast.InterfaceType{Methods: &goast.FieldList{}}, wantIsIdent: true, wantValue: "nil"},

		{name: "string", typeExpr: cachedIdent("string"), wantIsLit: true, wantValue: `""`},

		{name: "bool", typeExpr: cachedIdent("bool"), wantIsIdent: true, wantValue: "false"},

		{name: "int", typeExpr: cachedIdent("int"), wantIsLit: true, wantValue: "0"},
		{name: "int64", typeExpr: cachedIdent("int64"), wantIsLit: true, wantValue: "0"},
		{name: "float64", typeExpr: cachedIdent("float64"), wantIsLit: true, wantValue: "0"},

		{name: "[5]int array", typeExpr: &goast.ArrayType{Len: &goast.BasicLit{Value: "5"}, Elt: cachedIdent("int")}, wantIsComp: true},

		{name: "MyStruct", typeExpr: cachedIdent("MyStruct"), wantIsComp: true},
		{name: "pkg.MyStruct", typeExpr: &goast.SelectorExpr{X: cachedIdent("pkg"), Sel: cachedIdent("MyStruct")}, wantIsComp: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getZeroValueExpr(tc.typeExpr)
			require.NotNil(t, result)

			switch r := result.(type) {
			case *goast.Ident:
				assert.True(t, tc.wantIsIdent, "Expected Ident for %s", tc.name)
				if tc.wantValue != "" {
					assert.Equal(t, tc.wantValue, r.Name)
				}
			case *goast.BasicLit:
				assert.True(t, tc.wantIsLit, "Expected BasicLit for %s", tc.name)
				if tc.wantValue != "" {
					assert.Equal(t, tc.wantValue, r.Value)
				}
			case *goast.CompositeLit:
				assert.True(t, tc.wantIsComp, "Expected CompositeLit for %s", tc.name)
			default:
				t.Errorf("Unexpected type %T for %s", result, tc.name)
			}
		})
	}
}

func TestCreateHTMLAttributeLiteral(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		attributeName  string
		attributeValue string
	}{
		{name: "simple", attributeName: "class", attributeValue: "container"},
		{name: "empty value", attributeName: "disabled", attributeValue: ""},
		{name: "special chars", attributeName: "data-value", attributeValue: "test&value"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := createHTMLAttributeLiteral(tc.attributeName, tc.attributeValue)

			typeIdent, ok := result.Type.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "pikoruntime.HTMLAttribute", typeIdent.Name)

			require.Len(t, result.Elts, 2)

			nameKV := requireKeyValueExpr(t, result.Elts[0], "name key-value")
			nameKey := requireIdent(t, nameKV.Key, "name key")
			nameValue := requireBasicLit(t, nameKV.Value, "name value")
			assert.Equal(t, "Name", nameKey.Name)
			assert.Equal(t, `"`+tc.attributeName+`"`, nameValue.Value)

			valueKV := requireKeyValueExpr(t, result.Elts[1], "value key-value")
			valueKey := requireIdent(t, valueKV.Key, "value key")
			valueValue := requireBasicLit(t, valueKV.Value, "value value")
			assert.Equal(t, "Value", valueKey.Name)
			assert.Equal(t, `"`+tc.attributeValue+`"`, valueValue.Value)
		})
	}
}

func BenchmarkDefineAndAssign(b *testing.B) {
	rightHandSide := cachedIdent("value")

	b.ResetTimer()
	for b.Loop() {
		_ = defineAndAssign("x", rightHandSide)
	}
}

func BenchmarkStrLit(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		_ = strLit("hello world")
	}
}

func BenchmarkGetZeroValueExpr(b *testing.B) {
	typeExpr := cachedIdent("int64")

	b.ResetTimer()
	for b.Loop() {
		_ = getZeroValueExpr(typeExpr)
	}
}
