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

package ast_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_tidyGoLiteral(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simplest struct one field",
			input:    `&T{A:1}`,
			expected: "&T{\nA:1,\n}",
		},
		{
			name:     "Simplest slice one element",
			input:    `[]int{1}`,
			expected: "[]int{\n1,\n}",
		},
		{
			name:     "Simplest map one pair",
			input:    `map[int]int{1:2}`,
			expected: "map[int]int{\n1:2,\n}",
		},
		{
			name:     "Empty input",
			input:    ``,
			expected: ``,
		},
		{
			name:     "Non-literal input",
			input:    `x := y + z`,
			expected: `x := y + z`,
		},
		{
			name:     "Just a string",
			input:    `"hello world"`,
			expected: `"hello world"`,
		},
		{
			name:     "Struct field with bool",
			input:    `&T{Ok:true}`,
			expected: "&T{\nOk:true,\n}",
		},
		{
			name:     "Slice of strings",
			input:    `[]string{"a","b"}`,
			expected: "[]string{\n\"a\",\n\"b\",\n}",
		},
		{
			name:     "Single identifier",
			input:    `myVar`,
			expected: `myVar`,
		},
		{
			name:     "Basic binary expression",
			input:    `1 + 2`,
			expected: `1 + 2`,
		},
		{
			name:     "Struct with two string fields",
			input:    `&T{S1:"a",S2:"b"}`,
			expected: "&T{\nS1:\"a\",\nS2:\"b\",\n}",
		},
		{
			name:     "Slice with two numbers",
			input:    `[]int{100, 200}`,
			expected: "[]int{\n100,\n200,\n}",
		},
		{
			name:     "Map with two bool pairs",
			input:    `map[bool]bool{true:false, false:true}`,
			expected: "map[bool]bool{\ntrue:false,\nfalse:true,\n}",
		},
		{
			name:     "Function call with no arguments",
			input:    `doSomething()`,
			expected: `doSomething()`,
		},
		{
			name:     "Function call with one argument",
			input:    `doSomething(1)`,
			expected: `doSomething(1)`,
		},
		{
			name:     "Struct with a nil value",
			input:    `&T{Ptr:nil}`,
			expected: "&T{\nPtr:nil,\n}",
		},
		{
			name:     "Slice containing nil",
			input:    `[]*T{nil}`,
			expected: "[]*T{\nnil,\n}",
		},
		{
			name:     "Map with nil value",
			input:    `map[string]*T{"key":nil}`,
			expected: "map[string]*T{\n\"key\":nil,\n}",
		},
		{
			name:     "Empty struct literal (no pointer)",
			input:    `T{}`,
			expected: "T{}",
		},
		{
			name:     "Empty slice literal (no type)",
			input:    `[]{}`,
			expected: "[]{}",
		},
		{
			name:     "Just a single line comment",
			input:    `// hello`,
			expected: `// hello`,
		},
		{
			name:     "Just a multi-line comment",
			input:    `/* hello */`,
			expected: `/* hello */`,
		},
		{
			name:     "Struct with float",
			input:    `&T{F:3.14}`,
			expected: "&T{\nF:3.14,\n}",
		},
		{
			name:     "Slice with floats",
			input:    `[]float64{1.0, 2.5}`,
			expected: "[]float64{\n1.0,\n2.5,\n}",
		},
		{
			name:     "Map with floats",
			input:    `map[int]float64{1:1.1}`,
			expected: "map[int]float64{\n1:1.1,\n}",
		},
		{
			name:     "Parentheses are ignored",
			input:    `(1 + 2)`,
			expected: `(1 + 2)`,
		},
		{
			name:     "Struct field is a simple identifier",
			input:    `&T{Child:childNode}`,
			expected: "&T{\nChild:childNode,\n}",
		},
		{
			name:     "Slice element is an identifier",
			input:    `[]string{myString}`,
			expected: "[]string{\nmyString,\n}",
		},
		{
			name:     "Map value is an identifier",
			input:    `map[string]string{"key":myValue}`,
			expected: "map[string]string{\n\"key\":myValue,\n}",
		},
		{
			name:     "Map key is an identifier",
			input:    `map[string]string{myKey:"value"}`,
			expected: "map[string]string{\nmyKey:\"value\",\n}",
		},
		{
			name:     "Assignment expression",
			input:    `x = &T{}`,
			expected: "x = &T{}",
		},
		{
			name:     "var declaration",
			input:    `var x = &T{}`,
			expected: "var x = &T{}",
		},
		{
			name:     "const declaration",
			input:    `const y = "hello"`,
			expected: `const y = "hello"`,
		},
		{
			name:     "type declaration",
			input:    `type MyString string`,
			expected: `type MyString string`,
		},
		{
			name:  "struct type definition",
			input: `type T struct { Name string }`,
			expected: `type T struct {
Name string
}`,
		},
		{
			name:     "Empty parentheses",
			input:    `foo()`,
			expected: `foo()`,
		},
		{
			name:     "Empty square brackets (indexing)",
			input:    `a[i]`,
			expected: `a[i]`,
		},
		{
			name:     "Chained calls",
			input:    `a.b().c()`,
			expected: `a.b().c()`,
		},
		{
			name:     "Pointer access",
			input:    `p.Name`,
			expected: `p.Name`,
		},
		{
			name:     "Address of operator",
			input:    `&myVar`,
			expected: `&myVar`,
		},
		{
			name:     "Dereference operator",
			input:    `*myPtr`,
			expected: `*myPtr`,
		},
		{
			name:     "Struct with one field no comma",
			input:    `&T{A:1}`,
			expected: "&T{\nA:1,\n}",
		},
		{
			name:     "Slice with one element no comma",
			input:    `[]int{1}`,
			expected: "[]int{\n1,\n}",
		},
		{
			name:     "Map with one pair no comma",
			input:    `map[int]int{1:2}`,
			expected: "map[int]int{\n1:2,\n}",
		},
		{
			name:     "Struct with trailing whitespace",
			input:    `&T{A:1}  `,
			expected: "&T{\nA:1,\n}  ",
		},
		{
			name:     "Slice with leading whitespace",
			input:    `  []int{1}`,
			expected: "  []int{\n1,\n}",
		},
		{
			name:     "Map with newlines in input string",
			input:    "map[string]string{\"a\":`\n`}",
			expected: "map[string]string{\n\"a\":`\n`,\n}",
		},
		{
			name:     "Just a comma",
			input:    `,`,
			expected: `,`,
		},
		{
			name:     "Just braces",
			input:    `{}`,
			expected: "{}",
		},
		{
			name:     "Simple Struct",
			input:    `&T{Name:"foo",Value:123}`,
			expected: "&T{\nName:\"foo\",\nValue:123,\n}",
		},
		{
			name:     "Empty Struct",
			input:    `&T{}`,
			expected: "&T{}",
		},
		{
			name:     "Struct with existing trailing comma",
			input:    `&T{Name:"foo",}`,
			expected: "&T{\nName:\"foo\",\n}",
		},
		{
			name:     "Struct with multiline string field",
			input:    "&T{Description:`line1\nline2`}",
			expected: "&T{\nDescription:`line1\nline2`,\n}",
		},
		{
			name:     "Struct with single quotes",
			input:    `&T{Char:'a'}`,
			expected: "&T{\nChar:'a',\n}",
		},
		{
			name:     "Simple Slice",
			input:    `[]int{1,2,3}`,
			expected: "[]int{\n1,\n2,\n3,\n}",
		},
		{
			name:     "Empty Slice",
			input:    `[]int{}`,
			expected: "[]int{}",
		},
		{
			name:     "Slice of Structs",
			input:    `[]T{{Name:"a"},{Name:"b"}}`,
			expected: "[]T{\n{\nName:\"a\",\n},\n{\nName:\"b\",\n},\n}",
		},
		{
			name:     "Slice with existing trailing comma",
			input:    `[]int{1,2,}`,
			expected: "[]int{\n1,\n2,\n}",
		},
		{
			name:     "Array type",
			input:    `[2]int{1,2}`,
			expected: "[2]int{\n1,\n2,\n}",
		},
		{
			name:     "Simple Map",
			input:    `map[string]int{"a":1,"b":2}`,
			expected: "map[string]int{\n\"a\":1,\n\"b\":2,\n}",
		},
		{
			name:     "Empty Map",
			input:    `map[string]int{}`,
			expected: "map[string]int{}",
		},
		{
			name:     "Map with struct values",
			input:    `map[string]T{"a":{Name:"foo"}}`,
			expected: "map[string]T{\n\"a\":{\nName:\"foo\",\n},\n}",
		},
		{
			name:     "Map type should not be broken up by gofmt",
			input:    `map[string, *SomeType]{}`,
			expected: "map[string,*SomeType]{}",
		},
		{
			name:     "Map with complex key type",
			input:    `map[[2]string]int{[2]string{"a","b"}:1}`,
			expected: "map[[2]string]int{\n[2]string{\n\"a\",\n\"b\",\n}:1,\n}",
		},
		{
			name:     "Function Literal Body is ignored",
			input:    `func(){a:=1;b:=2}`,
			expected: `func(){a:=1;b:=2}`,
		},
		{
			name:     "Function call with multiple arguments is ignored",
			input:    `foo(a, b, c)`,
			expected: `foo(a,b,c)`,
		},
		{
			name:     "If statement is ignored",
			input:    `if x>0{return true}`,
			expected: `if x>0{return true}`,
		},
		{
			name:     "For loop is ignored",
			input:    `for i:=0;i<10;i++{sum+=i}`,
			expected: `for i:=0;i<10;i++{sum+=i}`,
		},
		{
			name:     "Switch statement is ignored",
			input:    `switch x{case 1:foo()}`,
			expected: `switch x{case 1:foo()}`,
		},
		{
			name:     "Nested Structs",
			input:    `&T{Name:"parent",Child:&T{Name:"child"}}`,
			expected: "&T{\nName:\"parent\",\nChild:&T{\nName:\"child\",\n},\n}",
		},
		{
			name:     "Struct with Slice Field",
			input:    `&T{Items:[]string{"a","b"}}`,
			expected: "&T{\nItems:[]string{\n\"a\",\n\"b\",\n},\n}",
		},
		{
			name:     "Slice of Maps",
			input:    `[]map[string]int{{"a":1},{"b":2}}`,
			expected: "[]map[string]int{\n{\n\"a\":1,\n},\n{\n\"b\":2,\n},\n}",
		},
		{
			name:     "Map with Slice Values",
			input:    `map[string][]int{"a":{1,2}}`,
			expected: "map[string][]int{\n\"a\":{\n1,\n2,\n},\n}",
		},
		{
			name:     "Extremely Nested Structure",
			input:    `&T{M:map[string][]T{"key":{{Items:[]string{"item1"}}}}}`,
			expected: "&T{\nM:map[string][]T{\n\"key\":{\n{\nItems:[]string{\n\"item1\",\n},\n},\n},\n},\n}",
		},
		{
			name:     "String containing braces and commas",
			input:    `&T{Name:"{foo,bar}"}`,
			expected: "&T{\nName:\"{foo,bar}\",\n}",
		},
		{
			name:     "Single line comment",
			input:    `&T{Name:"foo" //comment\n}`,
			expected: "&T{\nName:\"foo\",\n//comment\n}",
		},

		{
			name:     "Comment inside a struct",
			input:    `&T{/*comment*/Name:"foo"}`,
			expected: "&T{\n/*comment*/Name:\"foo\",\n}",
		},
		{
			name:     "String with escaped quote",
			input:    `&T{Name:"foo\"bar"}`,
			expected: "&T{\nName:\"foo\\\"bar\",\n}",
		},
		{
			name:     "Function call inside a struct field",
			input:    `&T{Alias:stringPtr("main")}`,
			expected: "&T{\nAlias:stringPtr(\"main\"),\n}",
		},
		{
			name:     "IIFE structure with function literal",
			input:    `func() *T {p := func(s string) *string {return &s}; return &T{Name:*p("a")}}()`,
			expected: "func() *T {p := func(s string) *string {return &s}; return &T{\nName:*p(\"a\"),\n}}()",
		},
		{
			name:     "Map with complex type",
			input:    `map[string][]*Directive{}`,
			expected: "map[string][]*Directive{}",
		},
		{
			name:     "Selector expression for type",
			input:    `&ast_domain.GoGeneratorAnnotation{Symbol:&ast_domain.ResolvedSymbol{Name:"MyComponent"}}`,
			expected: "&ast_domain.GoGeneratorAnnotation{\nSymbol:&ast_domain.ResolvedSymbol{\nName:\"MyComponent\",\n},\n}",
		},

		{
			name:     "No trailing comma on empty literal",
			input:    `&T{}`,
			expected: "&T{}",
		},

		{
			name:     "Compact one-liner",
			input:    `&T{A:1,B:2}`,
			expected: "&T{\nA:1,\nB:2,\n}",
		},
		{
			name:     "Preserve internal newlines in backtick strings",
			input:    "&T{Code:`func main() {\n\tfmt.Println(\"Hi\")\n}`}",
			expected: "&T{\nCode:`func main() {\n\tfmt.Println(\"Hi\")\n}`,\n}",
		},
		{
			name:     "Pointer to a struct",
			input:    `&*T{A:1}`,
			expected: "&*T{\nA:1,\n}",
		},
		{
			name:     "Type conversion",
			input:    `T(U{A:1})`,
			expected: "T(U{\nA:1,\n})",
		},
		{
			name:     "Star expression",
			input:    `*ast_domain.Something`,
			expected: `*ast_domain.Something`,
		},
		{
			name:     "Composite literal without address",
			input:    `ast_domain.TemplateAST{RootNodes:[]*ast_domain.TemplateNode{}}`,
			expected: "ast_domain.TemplateAST{\nRootNodes:[]*ast_domain.TemplateNode{},\n}",
		},
		{
			name:     "Function call returning a struct, used in a literal",
			input:    `&T{Child: createT()}`,
			expected: "&T{\nChild: createT(),\n}",
		},
		{
			name:     "Index expression as field value",
			input:    `&T{Value:items[0]}`,
			expected: "&T{\nValue:items[0],\n}",
		},
		{
			name:     "Slice of interface type",
			input:    `[]interface{}{1, "a"}`,
			expected: "[]interface{}{\n1,\n\"a\",\n}",
		},
		{
			name:     "Pointers in slice",
			input:    `[]*T{&T{}}`,
			expected: "[]*T{\n&T{},\n}",
		},

		{
			name:     "Return statement in a function block should not get a comma",
			input:    `func() *string { s := "a"; return &s }`,
			expected: `func() *string { s := "a"; return &s }`,
		},
		{
			name:     "Multi-statement function block",
			input:    `func() { a(); b() }`,
			expected: `func() { a(); b() }`,
		},
		{
			name:     "If block should not be formatted",
			input:    `if x { y() }`,
			expected: `if x { y() }`,
		},
		{
			name:     "Trailing comma should not be added to function block",
			input:    `func() *ast_domain.TemplateAST { return &ast_domain.TemplateAST{} }()`,
			expected: `func() *ast_domain.TemplateAST { return &ast_domain.TemplateAST{} }()`,
		},
		{
			name: "Complex IIFE with helpers and return statement",
			input: `func() *T {
	stringPtr := func(s string) *string { return &s }
	_ = stringPtr
	return &T{Name: "test"}
}()`,
			expected: `func() *T {
	stringPtr := func(s string) *string { return &s }
	_ = stringPtr
	return &T{
Name: "test",
}
}()`,
		},
		{
			name:     "Function literal inside a struct field",
			input:    `&T{F: func(){ a() }}`,
			expected: "&T{\nF: func(){ a() },\n}",
		},
		{
			name:     "Struct literal inside a function block",
			input:    `func(){ v := &T{A:1} }`,
			expected: "func(){ v := &T{\nA:1,\n} }",
		},
		{
			name:     "Simple if statement",
			input:    `if true { return }`,
			expected: `if true { return }`,
		},
		{
			name:     "Simple for loop",
			input:    `for i := 0; i < 1; i++ { doWork() }`,
			expected: `for i := 0; i < 1; i++ { doWork() }`,
		},
		{
			name:     "If-else statement",
			input:    `if false { a() } else { b() }`,
			expected: `if false { a() } else { b() }`,
		},
		{
			name:  "Composite literal inside an if block",
			input: `if true { v := &T{A:1} }`,
			expected: `if true { v := &T{
A:1,
} }`,
		},
		{
			name:     "The exact failing pattern (if err != nil)",
			input:    `if err != nil { return nil }`,
			expected: `if err != nil { return nil }`,
		},
		{
			name:     "Function literal in assignment",
			input:    `var myFunc = func(s string) { return }`,
			expected: `var myFunc = func(s string) { return }`,
		},
		{
			name:     "'if' statement inside a function literal",
			input:    `var myFunc = func(err error) { if err != nil { return } }`,
			expected: `var myFunc = func(err error) { if err != nil { return } }`,
		},
		{
			name:     "IIFE with nested func and if",
			input:    `var _ = func() int { helper := func() { if true { return } }; return 1 }()`,
			expected: `var _ = func() int { helper := func() { if true { return } }; return 1 }()`,
		},
		{
			name: "IIFE with nested func containing a composite literal",
			input: `var _ = func() {
	_ = func() {
		if true {
			_ = &T{Name: "test"}
		}
	}
}()`,
			expected: `var _ = func() {
	_ = func() {
		if true {
			_ = &T{
Name: "test",
}
		}
	}
}()`,
		},
		{
			name: "The exact failing IIFE pattern with 'if err != nil'",
			input: `var _ = func() {
	_ = func(err error) *string {
		if err != nil {
			return nil
		}
		s := "ok"
		return &s
	}
}()`,
			expected: `var _ = func() {
	_ = func(err error) *string {
		if err != nil {
			return nil
		}
		s := "ok"
		return &s
	}
}()`,
		},
		{
			name:     "Two simple statements in a function block",
			input:    `func() { a(); b() }`,
			expected: `func() { a(); b() }`,
		},
		{
			name:     "Assignment before an if statement",
			input:    `func() { x := 1; if x > 0 { return } }`,
			expected: `func() { x := 1; if x > 0 { return } }`,
		},

		{
			name:  "If statement before a composite literal",
			input: `func() { if true { return }; v := &T{A:1} }`,
			expected: `func() { if true { return }; v := &T{
A:1,
} }`,
		},
		{
			name: "The exact failing IIFE pattern",
			input: `func() {
	_ = func(err error) {
		s := "ok"
		if err != nil {
			return
		}
		_ = s
	}
}()`,
			expected: `func() {
	_ = func(err error) {
		s := "ok"
		if err != nil {
			return
		}
		_ = s
	}
}()`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			removeSpaces := func(s string) string {
				s = strings.ReplaceAll(s, " ", "")

				s = strings.ReplaceAll(s, "\r\n", "\n")
				return s
			}

			actual := tidyGoLiteral(tc.input)
			assert.Equal(t, removeSpaces(tc.expected), removeSpaces(actual), "Input:\n%s", tc.input)
		})
	}
}
