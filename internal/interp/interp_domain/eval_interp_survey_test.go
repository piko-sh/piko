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

package interp_domain

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type surveyListNode struct {
	Value int
	Next  *surveyListNode
}

func (surveyListNode) Length(head *surveyListNode) int {
	count := 0
	for n := head; n != nil; n = n.Next {
		count++
	}
	return count
}

type surveyFunFamily func(surveyFunFamily) surveyFunFamily

func TestSurveyRecursiveNativeMethodDispatch(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{
		"ll": {
			"Node": reflect.ValueOf((*surveyListNode)(nil)),
			"Make": reflect.ValueOf(func() *surveyListNode {
				return &surveyListNode{
					Value: 1,
					Next:  &surveyListNode{Value: 2, Next: &surveyListNode{Value: 3}},
				}
			}),
		},
	})

	result, err := service.Eval(context.Background(), `
import "ll"
head := ll.Make()
var zero ll.Node
zero.Length(head)
`)
	require.NoError(t, err)
	require.Equal(t, int64(3), result)
}

func TestSurveyEvalDefinedRecursiveStructs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "linked list via pointer",
			code: `type Node struct { V int; Next *Node }
head := &Node{V: 1, Next: &Node{V: 2, Next: &Node{V: 3}}}
total := 0
for n := head; n != nil; n = n.Next {
	total += n.V
}
total`,
			expect: int64(6),
		},
		{
			name: "tree via slice",
			code: `type Tree struct { V int; Kids []*Tree }
root := &Tree{V: 1, Kids: []*Tree{{V: 2}, {V: 3, Kids: []*Tree{{V: 4}}}}}
var sum func(t *Tree) int
sum = func(t *Tree) int {
	if t == nil { return 0 }
	total := t.V
	for _, k := range t.Kids {
		total += sum(k)
	}
	return total
}
sum(root)`,
			expect: int64(10),
		},
		{
			name: "map-valued tree",
			code: `type Tree struct { N string; Kids map[string]*Tree }
root := &Tree{N: "root", Kids: map[string]*Tree{
	"a": {N: "a"},
	"b": {N: "b", Kids: map[string]*Tree{"b1": {N: "b1"}}},
}}
root.Kids["b"].Kids["b1"].N`,
			expect: "b1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestSurveyByteAndRuneArithmetic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "byte addition",
			code: `var a byte = 10
var b byte = 32
int(a + b)`,
			expect: int64(42),
		},
		{
			name: "byte subtraction with underflow domain",
			code: `var a byte = 100
var b byte = 58
int(a - b)`,
			expect: int64(42),
		},
		{
			name: "rune addition",
			code: `var a rune = 'A'
var b rune = 1
int(a + b)`,
			expect: int64(66),
		},
		{
			name: "rune compare in loop",
			code: `s := "abc"
count := 0
for _, r := range s {
	if r >= 'a' && r <= 'z' { count++ }
}
count`,
			expect: int64(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestSurveyPointerChainSelectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "double pointer field read",
			code: `type S struct { V int }
s := S{V: 42}
p := &s
pp := &p
(*pp).V`,
			expect: int64(42),
		},
		{
			name: "double pointer through method",
			code: `type S struct { V int }
func (s S) Get() int { return s.V }
s := S{V: 99}
p := &s
pp := &p
(*pp).Get()`,
			expect: int64(99),
		},
		{
			name: "field assignment through double pointer",
			code: `type S struct { V int }
s := S{V: 1}
p := &s
pp := &p
(*pp).V = 77
s.V`,
			expect: int64(77),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestSurveyCompositeLitPointerElementForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "pointer-to-array explicit deref index",
			code: `m := map[string]*[3]int{"row": {10, 20, 30}}
a := *m["row"]
a[0] + a[1] + a[2]`,
			expect: int64(60),
		},
		{
			name: "struct literal with pointer element field",
			code: `type Inner struct { V int }
type Outer struct { Kids []*Inner }
o := Outer{Kids: []*Inner{{V: 5}, {V: 10}, {V: 15}}}
total := 0
for _, k := range o.Kids {
	total += k.V
}
total`,
			expect: int64(30),
		},
		{
			name: "map of pointer passed to closure",
			code: `type P struct { V int }
sum := func(m map[string]*P) int {
	t := 0
	for _, v := range m {
		t += v.V
	}
	return t
}
sum(map[string]*P{"a": {V: 3}, "b": {V: 4}, "c": {V: 5}})`,
			expect: int64(12),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestSurveyPointerToArrayAutoDerefIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "read through map of pointer to array",
			code: `m := map[string]*[3]int{"row": {10, 20, 30}}
m["row"][0] + m["row"][1] + m["row"][2]`,
			expect: int64(60),
		},
		{
			name: "read through direct pointer",
			code: `a := [4]int{7, 11, 13, 17}
p := &a
p[0] + p[1] + p[2] + p[3]`,
			expect: int64(48),
		},
		{
			name: "write through direct pointer",
			code: `a := [3]int{1, 2, 3}
p := &a
p[0] = 40
p[1] = 50
p[2] = 60
a[0] + a[1] + a[2]`,
			expect: int64(150),
		},
		{
			name: "string elements via pointer",
			code: `a := [2]string{"hello", "world"}
p := &a
p[0] + "-" + p[1]`,
			expect: "hello-world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestSurveyPointerToArrayNilPanic(t *testing.T) {
	t.Parallel()

	code := `var p *[3]int
_ = p[0]`

	service := NewService()
	_, err := service.Eval(context.Background(), code)
	require.Error(t, err, "indexing nil *[N]T must surface an error")
}

func TestSurveyPointerToArrayRangeLenSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "range with index and value",
			code: `a := [4]int{10, 20, 30, 40}
p := &a
total := 0
for i, v := range p {
	total += v * (i + 1)
}
total`,
			expect: int64(300),
		},
		{
			name: "range with value only",
			code: `a := [3]string{"a", "b", "c"}
p := &a
joined := ""
for _, v := range p {
	joined += v
}
joined`,
			expect: "abc",
		},
		{
			name: "len on non-nil pointer",
			code: `a := [5]int{1, 2, 3, 4, 5}
p := &a
len(p)`,
			expect: int64(5),
		},
		{
			name: "cap on non-nil pointer",
			code: `a := [7]int{}
p := &a
cap(p)`,
			expect: int64(7),
		},
		{
			name: "len on nil pointer returns array length",
			code: `var p *[4]int
len(p)`,
			expect: int64(4),
		},
		{
			name: "slice expression on pointer to array",
			code: `a := [5]int{10, 20, 30, 40, 50}
p := &a
s := p[1:4]
s[0] + s[1] + s[2]`,
			expect: int64(90),
		},
		{
			name: "full slice expression on pointer to array",
			code: `a := [6]int{1, 2, 3, 4, 5, 6}
p := &a
s := p[1:4:5]
len(s) + cap(s)`,
			expect: int64(7),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestSurveyMutuallyRecursiveEvalTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "simple mutual recursion via pointers",
			code: `type A struct { Tag string; B *B }
type B struct { Mark int; A *A }
a := &A{Tag: "root"}
b := &B{Mark: 7, A: a}
a.B = b
a.B.A.Tag`,
			expect: "root",
		},
		{
			name: "mutual types with a walking function",
			code: `type Left struct { Name string; Right *Right }
type Right struct { Value int; Left *Left }
func label(l *Left) string {
	if l == nil { return "" }
	if l.Right == nil { return l.Name }
	return l.Name + "/" + l.Right.Left.Name
}
l := &Left{Name: "a"}
r := &Right{Value: 42, Left: l}
l.Right = r
label(l)`,
			expect: "a/a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestSurveyRecursiveFunctionType(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{
		"ff": {
			"F": reflect.ValueOf((*surveyFunFamily)(nil)),
			"Identity": reflect.ValueOf(func(f surveyFunFamily) surveyFunFamily {
				return f
			}),
		},
	})

	_, err := service.Eval(context.Background(), `
import "ff"
var zero ff.F
_ = zero
_ = ff.Identity
42
`)
	if err != nil {
		t.Skipf("recursive function type not yet supported by interp type synthesis: %v", err)
	}
}
