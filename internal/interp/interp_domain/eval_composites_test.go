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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalSliceLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "slice literal len",
			code: `s := []int{10, 20, 30}
len(s)`,
			expect: int64(3),
		},
		{
			name: "slice index",
			code: `s := []int{10, 20, 30}
s[1]`,
			expect: int64(20),
		},
		{
			name: "slice index last",
			code: `s := []int{5, 10, 15, 20}
s[3]`,
			expect: int64(20),
		},
		{
			name: "string slice",
			code: `s := []string{"hello", "world"}
s[0]`,
			expect: "hello",
		},
		{
			name: "slice append and len",
			code: `s := []int{1, 2}
s = append(s, 3, 4)
len(s)`,
			expect: int64(4),
		},
		{
			name: "slice append value",
			code: `s := []int{1, 2}
s = append(s, 3)
s[2]`,
			expect: int64(3),
		},
		{
			name: "empty slice make",
			code: `s := make([]int, 3)
s[0] + s[1] + s[2]`,
			expect: int64(0),
		},
		{
			name: "slice set and get",
			code: `s := make([]int, 3)
s[0] = 10
s[1] = 20
s[2] = 30
s[0] + s[1] + s[2]`,
			expect: int64(60),
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

func TestEvalMapLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "map literal len",
			code: `m := map[string]int{"a": 1, "b": 2}
len(m)`,
			expect: int64(2),
		},
		{
			name: "map index",
			code: `m := map[string]int{"x": 42}
m["x"]`,
			expect: int64(42),
		},
		{
			name: "map set and get",
			code: `m := make(map[string]int)
m["key"] = 99
m["key"]`,
			expect: int64(99),
		},
		{
			name: "map delete",
			code: `m := map[string]int{"a": 1, "b": 2, "c": 3}
delete(m, "b")
len(m)`,
			expect: int64(2),
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

func TestEvalForRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "range slice sum",
			code: `s := []int{1, 2, 3, 4, 5}
sum := 0
for _, v := range s {
	sum = sum + v
}
sum`,
			expect: int64(15),
		},
		{
			name: "range slice index sum",
			code: `s := []int{10, 20, 30}
sum := 0
for i := range s {
	sum = sum + i
}
sum`,
			expect: int64(3),
		},
		{
			name: "range string",
			code: `s := "hello"
count := 0
for range s {
	count++
}
count`,
			expect: int64(5),
		},
		{
			name: "range slice compound assign",
			code: `s := make([]int, 3)
s[0] = 10
s[1] = 20
s[2] = 30
for j := range s {
	s[j] += 5
}
s[0] + s[1] + s[2]`,
			expect: int64(75),
		},
		{
			name: "range slice float value",
			code: `s := []float64{1.5, 2.5, 3.5}
sum := 0.0
for _, v := range s {
	sum += v
}
sum`,
			expect: 7.5,
		},
		{
			name: "range slice key only",
			code: `s := []int{10, 20, 30}
sum := 0
for i := range s {
	sum += s[i]
}
sum`,
			expect: int64(60),
		},
		{
			name: "range slice no vars",
			code: `s := []int{1, 2, 3}
count := 0
for range s {
	count++
}
count`,
			expect: int64(3),
		},
		{
			name: "range bool slice count",
			code: `s := []bool{true, false, true, true, false}
count := 0
for _, v := range s {
	if v {
		count++
	}
}
count`,
			expect: int64(3),
		},
		{
			name: "range uint slice sum",
			code: `s := []uint{10, 20, 30}
var total uint = 0
for _, v := range s {
	total += v
}
total`,
			expect: uint64(60),
		},
		{
			name: "iterations benchmark pattern",
			code: `const size = 10
s := make([]int, size)
for i := 0; i < size; i++ {
	s[i] = i
}
for _, x := range s {
	for j := range s {
		s[j] += x
	}
}
s[0]`,
			expect: int64(1013),
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

func TestEvalBuiltins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "len string",
			code:   `len("hello")`,
			expect: int64(5),
		},
		{
			name: "len slice",
			code: `s := []int{1, 2, 3}
len(s)`,
			expect: int64(3),
		},
		{
			name: "cap slice",
			code: `s := make([]int, 3, 10)
cap(s)`,
			expect: int64(10),
		},
		{
			name: "copy slices",
			code: `src := []int{1, 2, 3}
dst := make([]int, 3)
copy(dst, src)
dst[0] + dst[1] + dst[2]`,
			expect: int64(6),
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

func TestEvalSliceExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "slice low high",
			code: `s := []int{10, 20, 30, 40, 50}
t := s[1:3]
len(t)`,
			expect: int64(2),
		},
		{
			name: "slice low high value",
			code: `s := []int{10, 20, 30, 40, 50}
t := s[1:4]
t[0] + t[1] + t[2]`,
			expect: int64(90),
		},
		{
			name: "slice from start",
			code: `s := []int{10, 20, 30}
t := s[:2]
t[0] + t[1]`,
			expect: int64(30),
		},
		{
			name: "slice to end",
			code: `s := []int{10, 20, 30}
t := s[1:]
t[0] + t[1]`,
			expect: int64(50),
		},
		{
			name: "slice full copy",
			code: `s := []int{10, 20, 30}
t := s[:]
len(t)`,
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

func TestEvalStructLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "struct field access",
			code: `type Point struct { X int; Y int }
p := Point{X: 3, Y: 4}
p.X + p.Y`,
			expect: int64(7),
		},
		{
			name: "struct field set",
			code: `type Point struct { X int; Y int }
p := Point{X: 1, Y: 2}
p.X = 10
p.X + p.Y`,
			expect: int64(12),
		},
		{
			name: "struct string field",
			code: `type Named struct { Name string; Age int }
n := Named{Name: "alice", Age: 30}
n.Name`,
			expect: "alice",
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

func TestEvalArrayLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "array index",
			code:   "a := [3]int{10, 20, 30}\na[1]",
			expect: int64(20),
		},
		{
			name:   "array first",
			code:   "a := [3]int{1, 2, 3}\na[0]",
			expect: int64(1),
		},
		{
			name:   "array last",
			code:   "a := [3]int{1, 2, 3}\na[2]",
			expect: int64(3),
		},
		{
			name:   "array string",
			code:   `a := [2]string{"hello", "world"}` + "\na[1]",
			expect: "world",
		},
		{
			name:   "array len",
			code:   "len([5]int{1, 2, 3, 4, 5})",
			expect: int64(5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalTypeConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "int to float",
			code: `x := 5
float64(x)`,
			expect: float64(5),
		},
		{
			name: "float to int",
			code: `x := 3.7
int(x)`,
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

func TestEvalTypeConversionExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "int var to string rune",
			code:   "x := 65\nstring(x)",
			expect: "A",
		},
		{
			name:   "string var to bytes len",
			code:   "s := \"hello\"\nb := []byte(s)\nlen(b)",
			expect: int64(5),
		},
		{
			name:   "bytes var to string",
			code:   "b := []byte{72, 105}\nstring(b)",
			expect: "Hi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalForRangeMapTyped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "range map float values",
			code: `m := map[string]float64{"a": 1.5, "b": 2.5}
sum := 0.0
for _, v := range m {
	sum += v
}
sum`,
			expect: 4.0,
		},
		{
			name: "range map bool values",
			code: `m := map[string]bool{"a": true, "b": false, "c": true}
count := 0
for _, v := range m {
	if v {
		count++
	}
}
count`,
			expect: int64(2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalForRangeMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "range map value sum",
			code: `m := map[string]int{"a": 1, "b": 2, "c": 3}
sum := 0
for _, v := range m {
	sum = sum + v
}
sum`,
			expect: int64(6),
		},
		{
			name: "range map count keys",
			code: `m := map[string]int{"x": 10, "y": 20}
count := 0
for range m {
	count++
}
count`,
			expect: int64(2),
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

func TestEvalStructFuncFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "func_field_call_basic",
			code: `type Handler struct { Run func() int }
h := Handler{Run: func() int { return 42 }}
h.Run()`,
			expect: int64(42),
		},
		{
			name: "func_field_call_with_args",
			code: `type Op struct { Compute func(int, int) int }
op := Op{Compute: func(a, b int) int { return a + b }}
op.Compute(10, 32)`,
			expect: int64(42),
		},
		{
			name: "func_field_call_returns_string",
			code: `type Greeter struct { Greet func(string) string }
g := Greeter{Greet: func(name string) string { return "hi " + name }}
g.Greet("world")`,
			expect: "hi world",
		},
		{
			name: "func_field_create_pattern",
			code: `type Factory struct { Create func() any }
f := Factory{Create: func() any { return 99 }}
f.Create()`,
			expect: int64(99),
		},
		{
			name: "func_field_in_map",
			code: `type Entry struct {
	Name   string
	Invoke func(int) int
}
registry := map[string]Entry{
	"double": {Name: "double", Invoke: func(x int) int { return x * 2 }},
}
registry["double"].Invoke(21)`,
			expect: int64(42),
		},
		{
			name: "func_field_reassign",
			code: `type Runner struct { Do func() int }
r := Runner{Do: func() int { return 1 }}
r.Do = func() int { return 2 }
r.Do()`,
			expect: int64(2),
		},
		{
			name: "func_field_nil_check",
			code: `type Handler struct { Run func() int }
h := Handler{}
h.Run == nil`,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalStructTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "struct_with_tags_constructs",
			code:   "type Props struct {\n\tArticle string `prop:\"article\"`\n\tCount int `prop:\"count\"`\n}\np := Props{Article: \"hello\", Count: 3}\np.Article",
			expect: "hello",
		},
		{
			name:   "struct_with_json_tags",
			code:   "type User struct {\n\tName string `json:\"name\"`\n\tAge int `json:\"age\"`\n}\nu := User{Name: \"alice\", Age: 30}\nu.Name",
			expect: "alice",
		},
		{
			name:   "struct_with_multiple_tags",
			code:   "type Config struct {\n\tHost string `json:\"host\" yaml:\"host\"`\n\tPort int `json:\"port\" yaml:\"port\"`\n}\nc := Config{Host: \"localhost\", Port: 8080}\nc.Port",
			expect: int64(8080),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalSelfReferentialStruct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect     any
		name       string
		source     string
		entrypoint string
	}{
		{
			name: "self_referential_slice_field_access",
			source: `package main

type Node struct {
	Name     string
	Children []Node
}

func run() string {
	root := Node{Name: "root"}
	return root.Name
}

func main() {}
`,
			entrypoint: "run",
			expect:     "root",
		},
		{
			name: "self_referential_slice_field_with_children",
			source: `package main

type Node struct {
	Name     string
	Children []Node
}

func run() string {
	child := Node{Name: "child"}
	root := Node{Name: "root", Children: []Node{child}}
	return root.Children[0].Name
}

func main() {}
`,
			entrypoint: "run",
			expect:     "child",
		},
		{
			name: "self_referential_deeply_nested",
			source: `package main

type Node struct {
	Name     string
	Children []Node
}

func run() string {
	leaf := Node{Name: "leaf"}
	mid := Node{Name: "mid", Children: []Node{leaf}}
	root := Node{Name: "root", Children: []Node{mid}}
	return root.Children[0].Children[0].Name
}

func main() {}
`,
			entrypoint: "run",
			expect:     "leaf",
		},
		{
			name: "self_referential_pointer_field",
			source: `package main

type TreeNode struct {
	Value int
	Left  *TreeNode
	Right *TreeNode
}

func run() int {
	left := &TreeNode{Value: 1}
	right := &TreeNode{Value: 3}
	root := TreeNode{Value: 2, Left: left, Right: right}
	return root.Left.Value + root.Value + root.Right.Value
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(6),
		},
		{
			name: "self_referential_nil_children",
			source: `package main

type Node struct {
	Name     string
	Children []Node
}

func run() int {
	root := Node{Name: "root"}
	return len(root.Children)
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(0),
		},
		{
			name: "self_referential_append_children",
			source: `package main

type Node struct {
	Name     string
	Children []Node
}

func run() int {
	root := Node{Name: "root"}
	root.Children = append(root.Children, Node{Name: "a"})
	root.Children = append(root.Children, Node{Name: "b"})
	return len(root.Children)
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(2),
		},
		{
			name: "self_referential_many_string_fields",
			source: `package main

type PageField struct {
	ID             string
	Label          string
	Type           string
	HelpText       string
	Value          string
	EmbeddedFields []PageField
}

func run() string {
	child := PageField{ID: "child-1", Label: "Child", Type: "text", HelpText: "help", Value: "val"}
	parent := PageField{ID: "parent-1", Label: "Parent", Type: "group", HelpText: "", Value: "", EmbeddedFields: []PageField{child}}
	return parent.EmbeddedFields[0].Label
}

func main() {}
`,
			entrypoint: "run",
			expect:     "Child",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}
