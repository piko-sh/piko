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

package goastutil_test

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func parse(t *testing.T, expressionString string) ast.Expr {
	expression, err := parser.ParseExpr(expressionString)
	require.NoError(t, err, "Test setup failed: could not parse expression '%s'", expressionString)
	return expression
}

func TestTypeStringToAST(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedType   any
		expectedString string
	}{
		{name: "Primitive", input: "string", expectedType: &ast.Ident{}, expectedString: "string"},
		{name: "Pointer", input: "*string", expectedType: &ast.StarExpr{}, expectedString: "*string"},
		{name: "Selector", input: "models.User", expectedType: &ast.SelectorExpr{}, expectedString: "models.User"},
		{name: "Pointer to Selector", input: "*models.User", expectedType: &ast.StarExpr{}, expectedString: "*models.User"},
		{name: "Slice of Primitives", input: "[]int", expectedType: &ast.ArrayType{}, expectedString: "[]int"},
		{name: "Array of Primitives", input: "[5]int", expectedType: &ast.ArrayType{}, expectedString: "[5]int"},
		{name: "Slice of Pointers", input: "[]*User", expectedType: &ast.ArrayType{}, expectedString: "[]*User"},
		{name: "Map", input: "map[string]bool", expectedType: &ast.MapType{}, expectedString: "map[string]bool"},
		{name: "Simple Generic", input: "MyType[string]", expectedType: &ast.IndexExpr{}, expectedString: "MyType[string]"},
		{name: "Function Type Simple", input: "func()", expectedType: &ast.FuncType{}, expectedString: "func()"},
		{name: "Interface Empty", input: "interface{}", expectedType: &ast.InterfaceType{}, expectedString: "interface{}"},
		{name: "Interface Any", input: "any", expectedType: &ast.Ident{}, expectedString: "any"},
		{name: "Channel", input: "chan int", expectedType: &ast.ChanType{}, expectedString: "chan int"},
		{name: "Receive-only Channel", input: "<-chan bool", expectedType: &ast.ChanType{}, expectedString: "<-chan bool"},
		{name: "Send-only Channel", input: "chan<- bool", expectedType: &ast.ChanType{}, expectedString: "chan<- bool"},
		{name: "Empty String", input: "", expectedType: nil, expectedString: ""},

		{name: "Pointer to Slice", input: "*[]int", expectedType: &ast.StarExpr{}, expectedString: "*[]int"},
		{name: "Pointer to Slice of Pointers", input: "*[]*User", expectedType: &ast.StarExpr{}, expectedString: "*[]*User"},
		{name: "Slice of Slice of Pointers", input: "[][]*User", expectedType: &ast.ArrayType{}, expectedString: "[][]*User"},
		{name: "Slice of Maps", input: "[]map[string]int", expectedType: &ast.ArrayType{}, expectedString: "[]map[string]int"},
		{name: "Pointer Key in Map", input: "map[*User]string", expectedType: &ast.MapType{}, expectedString: "map[*User]string"},
		{name: "Pointer Value in Map", input: "map[string]*User", expectedType: &ast.MapType{}, expectedString: "map[string]*User"},
		{name: "Nested Map", input: "map[string]map[int]bool", expectedType: &ast.MapType{}, expectedString: "map[string]map[int]bool"},
		{name: "Complex Generic", input: "map[MyType[T]]AnotherType[U, V]", expectedType: &ast.MapType{}, expectedString: "map[MyType[T]]AnotherType[U, V]"},
		{name: "Generic with Qualified Type", input: "Box[models.User]", expectedType: &ast.IndexExpr{}, expectedString: "Box[models.User]"},
		{name: "Function Type Complex", input: "func(int, string) (bool, error)", expectedType: &ast.FuncType{}, expectedString: "func(int, string) (bool, error)"},
		{name: "Function Type Variadic", input: "func(keys ...string)", expectedType: &ast.FuncType{}, expectedString: "func(keys ...string)"},
		{name: "Function Type Named Params", input: "func(ctx context.Context, id int) error", expectedType: &ast.FuncType{}, expectedString: "func(ctx context.Context, id int) error"},
		{name: "Function Type as Parameter", input: "func(cb func(error))", expectedType: &ast.FuncType{}, expectedString: "func(cb func(error))"},
		{name: "Function returning a Function", input: "func() func()", expectedType: &ast.FuncType{}, expectedString: "func() func()"},
		{name: "Function with blank identifier param", input: "func(int, _ string)", expectedType: &ast.FuncType{}, expectedString: "func(int, _ string)"},
		{name: "Function with multiple named returns of same type", input: "func() (a, b int)", expectedType: &ast.FuncType{}, expectedString: "func() (a, b int)"},
		{name: "Channel of Struct Pointer", input: "chan *models.Event", expectedType: &ast.ChanType{}, expectedString: "chan *models.Event"},
		{name: "Send-only channel of func", input: "chan<- func()", expectedType: &ast.ChanType{}, expectedString: "chan<- func()"},
		{name: "Interface with Methods", input: "interface{ Read([]byte) (int, error) }", expectedType: &ast.InterfaceType{}, expectedString: "interface{ Read([]byte) (int, error) }"},
		{name: "Interface with Embedding", input: "interface{ io.Reader; io.Closer }", expectedType: &ast.InterfaceType{}, expectedString: `interface {
	io.Reader
	io.Closer
}`},
		{name: "Interface embedding comparable", input: "interface{ comparable; String() string }", expectedType: &ast.InterfaceType{}, expectedString: `interface {
	comparable
	String() string
}`},
		{name: "Struct Literal", input: "struct{ Name string `json:\"name\"` }", expectedType: &ast.StructType{}, expectedString: `struct {
	Name string ` + "`json:\"name\"`" + `
}`},
		{name: "Struct with blank identifier field", input: "struct{ _ string; Name string }", expectedType: &ast.StructType{}, expectedString: `struct {
	_	string
	Name	string
}`},
		{name: "Unsafe Pointer", input: "unsafe.Pointer", expectedType: &ast.SelectorExpr{}, expectedString: "unsafe.Pointer"},

		{name: "Parenthesized Type", input: "(string)", expectedType: &ast.ParenExpr{}, expectedString: "(string)"},
		{name: "Parenthesized Func Type", input: "(func())", expectedType: &ast.ParenExpr{}, expectedString: "(func())"},
		{name: "Whitespace", input: " map[ string ] * User ", expectedType: &ast.MapType{}, expectedString: "map[string]*User"},
		{name: "Multi-level Pointer", input: "***int", expectedType: &ast.StarExpr{}, expectedString: "***int"},
		{name: "Invalid Syntax", input: "func(broken", expectedType: &ast.Ident{}, expectedString: "any /* failed to parse type string: func(broken */"},
		{name: "Incomplete Type", input: "map[string]", expectedType: &ast.Ident{}, expectedString: "any /* failed to parse type string: map[string] */"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression := goastutil.TypeStringToAST(tc.input)

			if tc.expectedType == nil {
				assert.Nil(t, expression)
				return
			}

			require.NotNil(t, expression)
			assert.IsType(t, tc.expectedType, expression)

			assert.Equal(t, tc.expectedString, goastutil.ASTToTypeString(expression))
		})
	}
}

func TestASTToTypeString_Qualification(t *testing.T) {
	testCases := []struct {
		name            string
		inputExprString string
		pkgAlias        string
		expectedString  string
	}{
		{name: "Qualify Simple Identifier", inputExprString: "User", pkgAlias: "api", expectedString: "api.User"},
		{name: "Do Not Qualify Primitive", inputExprString: "string", pkgAlias: "api", expectedString: "string"},
		{name: "Do Not Qualify Builtin", inputExprString: "error", pkgAlias: "api", expectedString: "error"},
		{name: "Qualify Pointer to Identifier", inputExprString: "*User", pkgAlias: "api", expectedString: "*api.User"},
		{name: "Qualify Multi-Pointer", inputExprString: "**User", pkgAlias: "api", expectedString: "**api.User"},
		{name: "Qualify Slice of Identifier", inputExprString: "[]User", pkgAlias: "api", expectedString: "[]api.User"},
		{name: "Qualify Array of Pointer", inputExprString: "[5]*User", pkgAlias: "api", expectedString: "[5]*api.User"},
		{name: "Qualify Slice of Slice of Pointer", inputExprString: "[][]*User", pkgAlias: "api", expectedString: "[][]*api.User"},
		{name: "Qualify Map Key and Value", inputExprString: "map[Request]Response", pkgAlias: "api", expectedString: "map[api.Request]api.Response"},
		{name: "Qualify Only Unqualified Parts", inputExprString: "map[string]Response", pkgAlias: "api", expectedString: "map[string]api.Response"},
		{name: "Qualify Map with Pointer Key", inputExprString: "map[*Request]Response", pkgAlias: "api", expectedString: "map[*api.Request]api.Response"},
		{name: "Qualify Map with key only", inputExprString: "map[Request]string", pkgAlias: "api", expectedString: "map[api.Request]string"},
		{name: "Qualify Channel", inputExprString: "chan User", pkgAlias: "api", expectedString: "chan api.User"},
		{name: "Qualify Receive-Only Channel of Pointers", inputExprString: "<-chan *User", pkgAlias: "api", expectedString: "<-chan *api.User"},
		{name: "Do Not Re-qualify Selector", inputExprString: "models.User", pkgAlias: "api", expectedString: "models.User"},
		{name: "Qualify Generic Type Base", inputExprString: "Box[T]", pkgAlias: "models", expectedString: "models.Box[models.T]"},
		{name: "Qualify Multiple Generic Params", inputExprString: "Map[K, V]", pkgAlias: "api", expectedString: "api.Map[api.K, api.V]"},
		{name: "Qualify Generic Type Arguments", inputExprString: "Box[User]", pkgAlias: "api", expectedString: "api.Box[api.User]"},
		{name: "Qualify Nested Generic Arguments", inputExprString: "Wrapper[Box[User]]", pkgAlias: "api", expectedString: "api.Wrapper[api.Box[api.User]]"},
		{name: "Qualify Generic with already qualified argument", inputExprString: "Wrapper[models.User]", pkgAlias: "api", expectedString: "api.Wrapper[models.User]"},
		{name: "Qualify Function Parameters", inputExprString: "func(u User)", pkgAlias: "api", expectedString: "func(u api.User)"},
		{name: "Qualify Function Results", inputExprString: "func() (User, error)", pkgAlias: "api", expectedString: "func() (api.User, error)"},
		{name: "Qualify Multiple Named Returns", inputExprString: "func() (a, b User)", pkgAlias: "api", expectedString: "func() (a, b api.User)"},
		{name: "Qualify Blank Identifier Param", inputExprString: "func(_ User)", pkgAlias: "api", expectedString: "func(_ api.User)"},
		{name: "Qualify Nested Function", inputExprString: "func(cb func(User))", pkgAlias: "api", expectedString: "func(cb func(api.User))"},
		{name: "Qualify Return of Nested Function", inputExprString: "func() func() User", pkgAlias: "api", expectedString: "func() func() api.User"},
		{name: "Qualify Variadic Parameter", inputExprString: "func(users ...User)", pkgAlias: "api", expectedString: "func(users ...api.User)"},
		{name: "Qualify Parenthesized Type", inputExprString: "(User)", pkgAlias: "api", expectedString: "(api.User)"},
		{name: "Qualify Pointer in Paren", inputExprString: "(*User)", pkgAlias: "api", expectedString: "(*api.User)"},
		{name: "Qualify Interface Method", inputExprString: "interface{ M(User) }", pkgAlias: "api", expectedString: "interface{ M(api.User) }"},
		{name: "Qualify Struct Field", inputExprString: "struct{ F User }", pkgAlias: "api", expectedString: "struct{ F api.User }"},
		{name: "Do not qualify with empty alias", inputExprString: "User", pkgAlias: "", expectedString: "User"},
		{name: "Qualify Complex Composition", inputExprString: "[]*map[string]func() Ctx", pkgAlias: "api", expectedString: "[]*map[string]func() api.Ctx"},
		{name: "Qualify Slice of Variadic Funcs", inputExprString: "[]func(...User)", pkgAlias: "api", expectedString: "[]func(...api.User)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression := parse(t, tc.inputExprString)
			originalString := goastutil.ASTToTypeString(expression)

			result := goastutil.ASTToTypeString(expression, tc.pkgAlias)

			assert.Equal(t, tc.expectedString, result)
			assert.Equal(t, originalString, goastutil.ASTToTypeString(expression), "Original AST node should not be modified")
		})
	}
}

func TestASTToTypeString_Immutability(t *testing.T) {
	t.Run("Qualification does not modify original AST", func(t *testing.T) {
		originalAST := parse(t, "map[Key]Value")
		originalString := goastutil.ASTToTypeString(originalAST)

		_ = goastutil.ASTToTypeString(originalAST, "api")

		stringAfterQualification := goastutil.ASTToTypeString(originalAST)
		assert.Equal(t, originalString, stringAfterQualification, "The original AST node should be immutable")
	})
}

func TestIsPrimitiveOrBuiltin(t *testing.T) {
	t.Run("Should be true for primitives and builtins", func(t *testing.T) {
		primitives := []string{
			"bool", "string", "any", "error", "nil",
			"int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"byte", "rune",
			"float32", "float64",
			"complex64", "complex128",
			"any",
		}
		for _, p := range primitives {
			assert.True(t, goastutil.IsPrimitiveOrBuiltin(p), "Expected '%s' to be a primitive/builtin", p)
		}
	})

	t.Run("Should be false for user-defined types and keywords", func(t *testing.T) {
		userTypes := []string{
			"MyType", "User", "Request", "MyInt",
			"t", "t1", "Kv",
			"My_Type", "T_",
			"1T",
		}
		for _, ut := range userTypes {
			assert.False(t, goastutil.IsPrimitiveOrBuiltin(ut), "Expected '%s' to NOT be a primitive/builtin", ut)
		}
	})
}

func TestUnqualifyTypeExpr(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "Nil expression", input: "", expected: ""},
		{name: "Simple identifier no change", input: "User", expected: "User"},
		{name: "Primitive no change", input: "string", expected: "string"},
		{name: "Selector expression", input: "models.User", expected: "User"},
		{name: "Nested selector", input: "pkg.sub.Type", expected: "Type"},
		{name: "Pointer to selector", input: "*models.User", expected: "*User"},
		{name: "Double pointer to selector", input: "**models.User", expected: "**User"},
		{name: "Pointer to primitive no change", input: "*int", expected: "*int"},
		{name: "Slice of selector", input: "[]models.User", expected: "[]User"},
		{name: "Array of selector", input: "[5]models.User", expected: "[5]User"},
		{name: "Slice of pointer to selector", input: "[]*models.User", expected: "[]*User"},
		{name: "Slice of primitive no change", input: "[]int", expected: "[]int"},
		{name: "Map with qualified key", input: "map[models.Key]string", expected: "map[Key]string"},
		{name: "Map with qualified value", input: "map[string]models.Value", expected: "map[string]Value"},
		{name: "Map with both qualified", input: "map[models.Key]models.Value", expected: "map[Key]Value"},
		{name: "Map with primitive no change", input: "map[string]int", expected: "map[string]int"},
		{name: "Generic with qualified base", input: "models.Box[string]", expected: "Box[string]"},
		{name: "Generic with qualified argument", input: "Box[models.User]", expected: "Box[User]"},
		{name: "Generic with both qualified", input: "models.Box[models.User]", expected: "Box[User]"},
		{name: "Multi-param generic qualified base", input: "models.Map[string, int]", expected: "Map[string, int]"},
		{name: "Multi-param generic qualified arguments", input: "Map[models.K, models.V]", expected: "Map[K, V]"},
		{name: "Multi-param generic all qualified", input: "models.Map[models.K, models.V]", expected: "Map[K, V]"},
		{name: "Func with qualified param", input: "func(models.User)", expected: "func(User)"},
		{name: "Func with qualified return", input: "func() models.User", expected: "func() User"},
		{name: "Func with qualified param and return", input: "func(models.Request) models.Response", expected: "func(Request) Response"},
		{name: "Func with multiple qualified params", input: "func(models.A, models.B)", expected: "func(A, B)"},
		{name: "Func with multiple qualified returns", input: "func() (models.A, models.B)", expected: "func() (A, B)"},
		{name: "Func with no qualification needed", input: "func(int) string", expected: "func(int) string"},
		{name: "Slice of map with qualified value", input: "[]map[string]models.User", expected: "[]map[string]User"},
		{name: "Pointer to slice of selector", input: "*[]models.User", expected: "*[]User"},
		{name: "Map of slice of selector", input: "map[string][]models.User", expected: "map[string][]User"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var expression ast.Expr
			if tc.input != "" {
				expression = parse(t, tc.input)
			}

			result := goastutil.UnqualifyTypeExpr(expression)

			if tc.expected == "" {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			resultString := goastutil.ASTToTypeString(result)
			assert.Equal(t, tc.expected, resultString)
		})
	}
}

func TestASTToTypeString_TypeAssertExpr(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		pkgAlias string
		expected string
	}{
		{name: "Type assertion without alias", input: "x.(User)", pkgAlias: "", expected: "x.(User)"},
		{name: "Type assertion with alias", input: "x.(User)", pkgAlias: "api", expected: "x.(api.User)"},
		{name: "Type assertion with primitive", input: "x.(string)", pkgAlias: "api", expected: "x.(string)"},
		{name: "Type assertion with selector", input: "x.(models.User)", pkgAlias: "api", expected: "x.(models.User)"},
		{name: "Type assertion with pointer", input: "x.(*User)", pkgAlias: "api", expected: "x.(*api.User)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expression := parse(t, tc.input)

			var result string
			if tc.pkgAlias != "" {
				result = goastutil.ASTToTypeString(expression, tc.pkgAlias)
			} else {
				result = goastutil.ASTToTypeString(expression)
			}

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestASTToTypeString_EdgeCases(t *testing.T) {
	t.Run("Nil expression returns empty string", func(t *testing.T) {
		result := goastutil.ASTToTypeString(nil)
		assert.Equal(t, "", result)
	})

	t.Run("Nil expression with alias returns empty string", func(t *testing.T) {
		result := goastutil.ASTToTypeString(nil, "pkg")
		assert.Equal(t, "", result)
	})

	t.Run("Empty alias does not qualify", func(t *testing.T) {
		expression := parse(t, "User")
		result := goastutil.ASTToTypeString(expression, "")
		assert.Equal(t, "User", result)
	})

	t.Run("Selector with non-ident X falls back to slow path", func(t *testing.T) {

		expression := parse(t, "(*pkg).Type")
		result := goastutil.ASTToTypeString(expression)
		assert.Contains(t, result, "Type")
	})

	t.Run("Star expression with non-selector X", func(t *testing.T) {
		expression := parse(t, "*User")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "*User", result)
	})

	t.Run("Array type with length (not slice)", func(t *testing.T) {
		expression := parse(t, "[5]models.User")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "[5]models.User", result)
	})

	t.Run("Slice with non-selector element", func(t *testing.T) {
		expression := parse(t, "[]int")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "[]int", result)
	})

	t.Run("Pointer to non-selector non-star", func(t *testing.T) {
		expression := parse(t, "*[]int")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "*[]int", result)
	})

	t.Run("Slice of simple ident (not selector)", func(t *testing.T) {
		expression := parse(t, "[]User")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "[]User", result)
	})

	t.Run("Pointer to pointer to ident (not selector)", func(t *testing.T) {
		expression := parse(t, "**User")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "**User", result)
	})

	t.Run("Complex nested type with alias", func(t *testing.T) {
		expression := parse(t, "map[string][]User")
		result := goastutil.ASTToTypeString(expression, "api")
		assert.Equal(t, "map[string][]api.User", result)
	})

	t.Run("Ellipsis type with alias", func(t *testing.T) {
		expression := parse(t, "func(users ...User)")
		result := goastutil.ASTToTypeString(expression, "api")
		assert.Equal(t, "func(users ...api.User)", result)
	})

	t.Run("Slice of selector fast path", func(t *testing.T) {
		expression := parse(t, "[]models.User")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "[]models.User", result)
	})

	t.Run("Pointer to selector fast path", func(t *testing.T) {
		expression := parse(t, "*models.User")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "*models.User", result)
	})

	t.Run("Chan type without alias", func(t *testing.T) {
		expression := parse(t, "chan models.User")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "chan models.User", result)
	})

	t.Run("Paren type without alias", func(t *testing.T) {
		expression := parse(t, "(models.User)")
		result := goastutil.ASTToTypeString(expression)
		assert.Equal(t, "(models.User)", result)
	})
}

func TestUnqualifyTypeExpr_EdgeCases(t *testing.T) {
	t.Run("IndexExpr with no changes needed", func(t *testing.T) {
		expression := parse(t, "Box[string]")
		result := goastutil.UnqualifyTypeExpr(expression)

		resultString := goastutil.ASTToTypeString(result)
		assert.Equal(t, "Box[string]", resultString)
	})

	t.Run("IndexListExpr with no changes needed", func(t *testing.T) {
		expression := parse(t, "Map[string, int]")
		result := goastutil.UnqualifyTypeExpr(expression)
		resultString := goastutil.ASTToTypeString(result)
		assert.Equal(t, "Map[string, int]", resultString)
	})

	t.Run("Star expression with no changes needed", func(t *testing.T) {
		expression := parse(t, "*User")
		result := goastutil.UnqualifyTypeExpr(expression)
		resultString := goastutil.ASTToTypeString(result)
		assert.Equal(t, "*User", resultString)
	})

	t.Run("Array type with no changes needed", func(t *testing.T) {
		expression := parse(t, "[]int")
		result := goastutil.UnqualifyTypeExpr(expression)
		resultString := goastutil.ASTToTypeString(result)
		assert.Equal(t, "[]int", resultString)
	})

	t.Run("Map type with no changes needed", func(t *testing.T) {
		expression := parse(t, "map[string]int")
		result := goastutil.UnqualifyTypeExpr(expression)
		resultString := goastutil.ASTToTypeString(result)
		assert.Equal(t, "map[string]int", resultString)
	})

	t.Run("Func type with no qualified params", func(t *testing.T) {
		expression := parse(t, "func(int) string")
		result := goastutil.UnqualifyTypeExpr(expression)
		resultString := goastutil.ASTToTypeString(result)
		assert.Equal(t, "func(int) string", resultString)
	})

	t.Run("Func type with nil params and results", func(t *testing.T) {
		expression := parse(t, "func()")
		result := goastutil.UnqualifyTypeExpr(expression)
		resultString := goastutil.ASTToTypeString(result)
		assert.Equal(t, "func()", resultString)
	})

	t.Run("Deeply nested unqualification", func(t *testing.T) {
		expression := parse(t, "map[models.K][]models.V")
		result := goastutil.UnqualifyTypeExpr(expression)
		resultString := goastutil.ASTToTypeString(result)
		assert.Equal(t, "map[K][]V", resultString)
	})
}

func TestParseStructTag(t *testing.T) {
	testCases := []struct {
		expectedMap map[string]string
		name        string
		inputTag    string
	}{
		{name: "Known tag: prop", inputTag: `prop:"userID"`, expectedMap: map[string]string{"prop": "userID"}},
		{name: "Known tag: validate", inputTag: `validate:"required,uuid"`, expectedMap: map[string]string{"validate": "required,uuid"}},
		{name: "Known tag: default", inputTag: `default:"guest"`, expectedMap: map[string]string{"default": "guest"}},
		{name: "Known tag: factory", inputTag: `factory:"NewUser"`, expectedMap: map[string]string{"factory": "NewUser"}},
		{name: "Known tag: coerce with value", inputTag: `coerce:"true"`, expectedMap: map[string]string{"coerce": "true"}},
		{name: "Known tag: coerce with empty string value", inputTag: `coerce:""`, expectedMap: map[string]string{"coerce": ""}},

		{name: "All known tags present", inputTag: `prop:"id" validate:"required" default:"0" factory:"New" coerce:""`,
			expectedMap: map[string]string{
				"prop":     "id",
				"validate": "required",
				"default":  "0",
				"factory":  "New",
				"coerce":   "",
			}},
		{name: "Mix of known and unknown tags", inputTag: `prop:"name" json:"userName" validate:"required"`,
			expectedMap: map[string]string{
				"prop":     "name",
				"validate": "required",
			}},
		{name: "Only unknown tags", inputTag: `json:"name" xml:"id"`, expectedMap: map[string]string{}},
		{name: "Empty tag string", inputTag: ``, expectedMap: map[string]string{}},
		{name: "Tag string with backticks", inputTag: "`prop:\"name\"`", expectedMap: map[string]string{"prop": "name"}},
		{name: "Duplicate known tag", inputTag: `prop:"a" prop:"b"`, expectedMap: map[string]string{"prop": "a"}},
		{name: "Malformed tag (ignored by reflect)", inputTag: `prop:name`, expectedMap: map[string]string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := inspector_dto.ParseStructTag(tc.inputTag)
			assert.Equal(t, tc.expectedMap, result)
		})
	}
}
