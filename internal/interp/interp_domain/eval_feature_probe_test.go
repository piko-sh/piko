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
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalMethodValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "method value on instance",
			code: `type Counter struct { V int }
func (c Counter) Get() int { return c.V }
co := Counter{V: 42}
f := co.Get
f()`,
			expect: int64(42),
		},
		{
			name: "method value with arguments",
			code: `type Adder struct { Base int }
func (a Adder) Add(x int) int { return a.Base + x }
ad := Adder{Base: 10}
f := ad.Add
f(5)`,
			expect: int64(15),
		},
		{
			name: "method value string return",
			code: `type Named struct { Name string }
func (n Named) Greet() string { return n.Name }
nn := Named{Name: "alice"}
f := nn.Greet
f()`,
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

func TestEvalPromotedMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "promoted method direct call",
			code: `type Inner struct { V int }
func (i Inner) Get() int { return i.V }
type Outer struct { Inner }
o := Outer{Inner{V: 99}}
o.Get()`,
			expect: int64(99),
		},
		{
			name: "promoted field access",
			code: `type Inner struct { V int }
type Outer struct { Inner }
o := Outer{Inner{V: 55}}
o.V`,
			expect: int64(55),
		},
		{
			name: "interface via promoted method",
			code: `type Getter interface { Get() int }
type Base struct { V int }
func (b Base) Get() int { return b.V }
type Wrapper struct { Base }
var g Getter = Wrapper{Base{V: 77}}
g.Get()`,
			expect: int64(77),
		},
		{
			name: "promoted method with arguments",
			code: `type Base struct { V int }
func (b Base) Add(x int) int { return b.V + x }
type Extended struct { Base }
e := Extended{Base{V: 10}}
e.Add(5)`,
			expect: int64(15),
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

func TestEvalFuncsInCollections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "func in slice",
			code: `fns := []func(int) int{func(x int) int { return x * 2 }}
fns[0](5)`,
			expect: int64(10),
		},
		{
			name: "func in map",
			code: `m := map[string]func(int) int{"double": func(x int) int { return x * 2 }}
m["double"](5)`,
			expect: int64(10),
		},
		{
			name: "variadic spread",
			code: `func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}
arguments := []int{1, 2, 3, 4, 5}
sum(arguments...)`,
			expect: int64(15),
		},
		{
			name: "variadic spread with fixed params",
			code: `func join(sep string, parts ...string) string {
	result := ""
	for i, p := range parts {
		if i > 0 { result = result + sep }
		result = result + p
	}
	return result
}
words := []string{"a", "b", "c"}
join("-", words...)`,
			expect: "a-b-c",
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

func TestEvalMapCommaOkIfInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "if-init map hit",
			code: `m := map[string]int{"a": 42}
result := 0
if v, ok := m["a"]; ok {
	result = v
}
result`,
			expect: int64(42),
		},
		{
			name: "if-init map miss",
			code: `m := map[string]int{"a": 42}
result := -1
if _, ok := m["b"]; ok {
	result = 99
}
result`,
			expect: int64(-1),
		},
		{
			name: "map comma-ok in assignment",
			code: `m := map[string]int{"x": 7}
v, ok := m["x"]
_ = ok
v`,
			expect: int64(7),
		},
		{
			name: "map comma-ok false",
			code: `m := map[string]int{"x": 7}
_, ok := m["y"]
ok`,
			expect: false,
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

func TestEvalMethodExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "basic value receiver",
			code: `type S struct { V int }
func (s S) Get() int { return s.V }
f := S.Get
f(S{V: 42})`,
			expect: int64(42),
		},
		{
			name: "with arguments",
			code: `type A struct { Base int }
func (a A) Add(x int) int { return a.Base + x }
f := A.Add
f(A{Base: 10}, 5)`,
			expect: int64(15),
		},
		{
			name: "direct call no variable",
			code: `type S struct { V int }
func (s S) Get() int { return s.V }
S.Get(S{V: 7})`,
			expect: int64(7),
		},
		{
			name: "string return",
			code: `type N struct { Name string }
func (n N) Greet() string { return "hello " + n.Name }
f := N.Greet
f(N{Name: "world"})`,
			expect: "hello world",
		},
		{
			name: "promoted method",
			code: `type Inner struct { V int }
func (i Inner) Get() int { return i.V }
type Outer struct { Inner }
f := Outer.Get
f(Outer{Inner{V: 99}})`,
			expect: int64(99),
		},
		{
			name: "passed as argument",
			code: `type P struct { A int; B int }
func (p P) Sum() int { return p.A + p.B }
apply := func(f func(P) int, p P) int { return f(p) }
apply(P.Sum, P{A: 3, B: 4})`,
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

func TestEvalMultipleReturnValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "three return values int",
			code: `func triple(x int) (int, string, bool) {
	return x * 2, "ok", true
}
a, b, c := triple(5)
_ = b
_ = c
a`,
			expect: int64(10),
		},
		{
			name: "three return values string",
			code: `func triple(x int) (int, string, bool) {
	return x * 2, "ok", true
}
a, b, c := triple(5)
_ = a
_ = c
b`,
			expect: "ok",
		},
		{
			name: "three return values bool",
			code: `func triple(x int) (int, string, bool) {
	return x * 2, "ok", true
}
a, b, c := triple(5)
_ = a
_ = b
c`,
			expect: true,
		},
		{
			name: "four return values",
			code: `func quad() (int, int, int, int) {
	return 1, 2, 3, 4
}
a, b, c, d := quad()
a + b + c + d`,
			expect: int64(10),
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

func TestEvalChainedMethodCalls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "chained builder pattern",
			code: `type Builder struct { V int }
func (b Builder) Add(x int) Builder { return Builder{V: b.V + x} }
func (b Builder) Result() int { return b.V }
Builder{V: 0}.Add(3).Add(4).Result()`,
			expect: int64(7),
		},
		{
			name: "chain with double",
			code: `type S struct { V int }
func (s S) Double() S { return S{V: s.V * 2} }
func (s S) Get() int { return s.V }
S{V: 5}.Double().Double().Get()`,
			expect: int64(20),
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

func TestEvalCommaOkTypeAssertionInIfInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "type assertion hit",
			code: `var x interface{} = 42
result := 0
if v, ok := x.(int); ok {
	result = v
}
result`,
			expect: int64(42),
		},
		{
			name: "type assertion miss",
			code: `var x interface{} = "hello"
result := -1
if _, ok := x.(int); ok {
	result = 99
}
result`,
			expect: int64(-1),
		},
		{
			name: "type assertion string",
			code: `var x interface{} = "hello"
result := ""
if v, ok := x.(string); ok {
	result = v
}
result`,
			expect: "hello",
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

func TestEvalComplexCompositeLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "struct with func field in map",
			code: `type Handler struct {
	Invoke func(int) int
}
m := map[string]Handler{
	"double": {Invoke: func(x int) int { return x * 2 }},
	"triple": {Invoke: func(x int) int { return x * 3 }},
}
m["double"].Invoke(5) + m["triple"].Invoke(3)`,
			expect: int64(19),
		},
		{
			name: "nested struct with func fields",
			code: `type Inner struct { Fn func() int }
type Outer struct { A Inner; B Inner }
o := Outer{
	A: Inner{Fn: func() int { return 10 }},
	B: Inner{Fn: func() int { return 20 }},
}
o.A.Fn() + o.B.Fn()`,
			expect: int64(30),
		},
		{
			name: "map of pointer-to-struct with elided literals",
			code: `type Info struct { Icon string; Title string }
m := map[string]*Info{
	"a": {Icon: "x", Title: "Alpha"},
	"b": {Icon: "y", Title: "Beta"},
}
m["a"].Title + "-" + m["b"].Icon`,
			expect: "Alpha-y",
		},
		{
			name: "slice of pointer-to-struct with elided literals",
			code: `type P struct { V int }
s := []*P{{V: 10}, {V: 20}, {V: 30}}
s[0].V + s[1].V + s[2].V`,
			expect: int64(60),
		},
		{
			name: "array of pointer-to-struct with elided literals",
			code: `type P struct { V int }
a := [2]*P{{V: 7}, {V: 35}}
a[0].V + a[1].V`,
			expect: int64(42),
		},
		{
			name: "elided and explicit address-of forms coexist",
			code: `type P struct { V int }
m := map[string]*P{
	"elided":   {V: 1},
	"explicit": &P{V: 10},
}
m["elided"].V + m["explicit"].V`,
			expect: int64(11),
		},
		{
			name: "mutate field through pointer retrieved from map",
			code: `type P struct { V int }
m := map[string]*P{"k": {V: 5}}
m["k"].V = 99
m["k"].V`,
			expect: int64(99),
		},
		{
			name: "nested struct with pointer-elided inner values",
			code: `type Inner struct { V int }
type Outer struct { A *Inner; B *Inner }
o := Outer{A: &Inner{V: 3}, B: &Inner{V: 4}}
o.A.V * o.B.V`,
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

func TestEvalElidedPointerCompositeLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "empty map of pointer-to-struct",
			code: `type P struct { V int }
m := map[string]*P{}
len(m)`,
			expect: int64(0),
		},
		{
			name: "empty slice of pointer-to-struct",
			code: `type P struct { V int }
s := []*P{}
len(s)`,
			expect: int64(0),
		},
		{
			name: "single-field elided struct pointer",
			code: `type P struct { V int }
m := map[string]*P{"only": {V: 7}}
m["only"].V`,
			expect: int64(7),
		},
		{
			name: "pointer-to-array element with elided literal",
			code: `m := map[string]*[3]int{"row": {10, 20, 30}}
a := *m["row"]
a[0] + a[1] + a[2]`,
			expect: int64(60),
		},
		{
			name: "pointer-to-slice element with elided literal",
			code: `m := map[string]*[]int{"row": {1, 2, 3, 4}}
s := *m["row"]
s[0] + s[1] + s[2] + s[3]`,
			expect: int64(10),
		},
		{
			name: "pointer-to-map element with elided literal",
			code: `m := map[string]*map[string]int{"grp": {"a": 5, "b": 9}}
inner := *m["grp"]
inner["a"] + inner["b"]`,
			expect: int64(14),
		},
		{
			name: "slice of pointer-to-array with elided literal",
			code: `s := []*[3]int{{1, 2, 3}, {4, 5, 6}}
first := *s[0]
second := *s[1]
first[0] + first[1] + first[2] + second[0] + second[1] + second[2]`,
			expect: int64(21),
		},
		{
			name: "array of pointer-to-struct iterated",
			code: `type P struct { V int }
a := [4]*P{{V: 1}, {V: 2}, {V: 3}, {V: 4}}
total := 0
for i := 0; i < len(a); i++ {
	total += a[i].V
}
total`,
			expect: int64(10),
		},
		{
			name: "slice of pointer-to-struct range iteration",
			code: `type P struct { Name string; Qty int }
items := []*P{
	{Name: "a", Qty: 2},
	{Name: "b", Qty: 3},
	{Name: "c", Qty: 5},
}
total := 0
for _, it := range items {
	total += it.Qty
}
total`,
			expect: int64(10),
		},
		{
			name: "map of pointer-to-struct with not-ok branch",
			code: `type P struct { V int }
m := map[string]*P{"a": {V: 42}}
result := 0
if info, ok := m["missing"]; ok {
	result = info.V
} else {
	result = m["a"].V
}
result`,
			expect: int64(42),
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

func TestEvalUintSliceIndex(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `s := []int{10, 20, 30}
var i uint = 1
s[i]`)
	require.NoError(t, err)
	require.Equal(t, int64(20), result)
}

func TestRegisterArenaSlabGrowth(t *testing.T) {
	t.Parallel()

	arena := GetRegisterArena()
	defer PutRegisterArena(arena)

	boolRegs := [NumRegisterKinds]uint32{}
	boolRegs[registerBool] = 8
	for range 20 {
		arena.AllocRegisters(boolRegs)
	}

	uintRegs := [NumRegisterKinds]uint32{}
	uintRegs[registerUint] = 8
	for range 20 {
		arena.AllocRegisters(uintRegs)
	}

	complexRegs := [NumRegisterKinds]uint32{}
	complexRegs[registerComplex] = 8
	for range 12 {
		arena.AllocRegisters(complexRegs)
	}
}

func TestNewRegistersForBench(t *testing.T) {
	t.Parallel()

	regs := NewRegistersForBench([NumRegisterKinds]uint32{4, 2, 2, 3, 1, 1, 1})
	require.Len(t, regs.ints, 4)
	require.Len(t, regs.floats, 2)
	require.Len(t, regs.strings, 2)
	require.Len(t, regs.general, 3)
	require.Len(t, regs.bools, 1)
	require.Len(t, regs.uints, 1)
	require.Len(t, regs.complex, 1)
}

func TestEvalBoolComparison(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "bool equals true",
			code:   `a := true; b := true; a == b`,
			expect: true,
		},
		{
			name:   "bool equals false",
			code:   `a := true; b := false; a == b`,
			expect: false,
		},
		{
			name:   "bool not equals",
			code:   `a := true; b := false; a != b`,
			expect: true,
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

func TestEvalUintNegation(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `var u uint = 1
-u`)
	require.NoError(t, err)
	require.Equal(t, ^uint64(0), result)
}

func TestEvalRangeOverUint(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `var total uint
for i := range uint(5) {
	total += i
}
total`)
	require.NoError(t, err)
	require.Equal(t, uint64(10), result)
}

func TestEvalSliceSetOutOfBounds(t *testing.T) {
	t.Parallel()

	service := NewService()
	_, err := service.Eval(context.Background(), `s := []int{1, 2, 3}
s[-1] = 99
s[0]`)
	require.Error(t, err)
}

func TestEvalSliceGetOutOfBounds(t *testing.T) {
	t.Parallel()

	service := NewService()
	_, err := service.Eval(context.Background(), `s := []int{1, 2, 3}
s[5]`)
	require.Error(t, err)
}

func TestEvalCapChannel(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `ch := make(chan int, 10)
cap(ch)`)
	require.NoError(t, err)
	require.Equal(t, int64(10), result)
}

func TestEvalMapMissDefault(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `m := map[string]int{"a": 1}
m["missing"]`)
	require.NoError(t, err)
	require.Equal(t, int64(0), result)
}

func TestEvalRangeValueTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "range float slice",
			code: `s := []float64{1.0, 2.0, 3.0}
total := 0.0
for _, v := range s {
	total += v
}
total`,
			expect: 6.0,
		},
		{
			name: "range bool slice",
			code: `s := []bool{true, false, true}
count := 0
for _, v := range s {
	if v { count++ }
}
count`,
			expect: int64(2),
		},
		{
			name: "range string map values",
			code: `m := map[int]string{1: "a", 2: "b"}
result := ""
for _, v := range m {
	result += v
}
len(result)`,
			expect: int64(2),
		},
		{
			name: "range uint slice",
			code: `s := []uint{10, 20, 30}
var total uint
for _, v := range s {
	total += v
}
total`,
			expect: uint64(60),
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

func TestEvalCrossPackageStringConst(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/lib"

func entrypoint() bool {
	return lib.Flag
}

func main() {}
`,
		},
		"lib": {
			"lib.go": `package lib

const Flag = true
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "true", fmt.Sprint(result))
}

func TestEvalCrossPackageFloatConstUsage(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/lib"

func entrypoint() float64 {
	return lib.Pi * 2
}

func main() {}
`,
		},
		"lib": {
			"lib.go": `package lib

const Pi = 3.14
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "6.28", fmt.Sprint(result))
}

func TestEvalComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "single-line comment after code",
			code:   "x := 42 // this is a comment\nx",
			expect: int64(42),
		},
		{
			name:   "comment before statement",
			code:   "// compute\nx := 10 + 5\nx",
			expect: int64(15),
		},
		{
			name:   "multi-line comment inline",
			code:   "x := /* inline */ 7\nx",
			expect: int64(7),
		},
		{
			name:   "comment-only lines",
			code:   "// a\n// b\nx := 1\nx",
			expect: int64(1),
		},
		{
			name:   "comment inside function",
			code:   "func f() int {\n// body\nreturn 3\n}\nf()",
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

func newOSSymbols() *SymbolRegistry {
	return NewSymbolRegistry(SymbolExports{
		"os": {
			"Getenv":    reflect.ValueOf(os.Getenv),
			"LookupEnv": reflect.ValueOf(os.LookupEnv),
			"Environ":   reflect.ValueOf(os.Environ),
			"Setenv":    reflect.ValueOf(os.Setenv),
			"Unsetenv":  reflect.ValueOf(os.Unsetenv),
		},
	})
}

func TestEvalWithEnv(t *testing.T) {
	t.Parallel()

	t.Run("Getenv reads from configured map", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithEnv(map[string]string{"MY_VAR": "hello"}))
		service.UseSymbols(newOSSymbols())
		result, err := service.Eval(context.Background(), "import \"os\"\nos.Getenv(\"MY_VAR\")")
		require.NoError(t, err)
		require.Equal(t, "hello", result)
	})

	t.Run("Getenv returns empty for missing key", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithEnv(map[string]string{"A": "1"}))
		service.UseSymbols(newOSSymbols())
		result, err := service.Eval(context.Background(), "import \"os\"\nos.Getenv(\"MISSING\")")
		require.NoError(t, err)
		require.Equal(t, "", result)
	})

	t.Run("Setenv then Getenv", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithEnv(map[string]string{}))
		service.UseSymbols(newOSSymbols())
		source := `package main
import "os"
func Run() string {
	os.Setenv("NEW_KEY", "new_val")
	return os.Getenv("NEW_KEY")
}
`
		result, err := service.EvalFile(context.Background(), source, "Run")
		require.NoError(t, err)
		require.Equal(t, "new_val", fmt.Sprint(result))
	})

	t.Run("env isolation from host", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithEnv(map[string]string{"SANDBOXED": "yes"}))
		service.UseSymbols(newOSSymbols())
		result, err := service.Eval(context.Background(), "import \"os\"\nos.Getenv(\"SANDBOXED\")")
		require.NoError(t, err)
		require.Equal(t, "yes", result)
	})

	t.Run("Environ returns configured entries", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithEnv(map[string]string{"X": "1", "Y": "2"}))
		service.UseSymbols(newOSSymbols())
		result, err := service.Eval(context.Background(), "import \"os\"\nlen(os.Environ())")
		require.NoError(t, err)
		require.Equal(t, int64(2), result)
	})
}

func newStrconvSymbols() *SymbolRegistry {
	return NewSymbolRegistry(SymbolExports{
		"strconv": {
			"Atoi":      reflect.ValueOf(strconv.Atoi),
			"Itoa":      reflect.ValueOf(strconv.Itoa),
			"ParseBool": reflect.ValueOf(strconv.ParseBool),
		},
	})
}

func TestMultiReturnSelectorCall(t *testing.T) {
	t.Parallel()

	t.Run("LookupEnv found", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithEnv(map[string]string{"TEST_KEY": "test_val"}))
		service.UseSymbols(newOSSymbols())
		result, err := service.Eval(context.Background(), "import \"os\"\nv, _ := os.LookupEnv(\"TEST_KEY\")\nv")
		require.NoError(t, err)
		require.Equal(t, "test_val", result)
	})

	t.Run("LookupEnv ok flag", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithEnv(map[string]string{"TEST_KEY": "test_val"}))
		service.UseSymbols(newOSSymbols())
		result, err := service.Eval(context.Background(), "import \"os\"\n_, ok := os.LookupEnv(\"TEST_KEY\")\nok")
		require.NoError(t, err)
		require.Equal(t, true, result)
	})

	t.Run("LookupEnv missing", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithEnv(map[string]string{}))
		service.UseSymbols(newOSSymbols())
		result, err := service.Eval(context.Background(), "import \"os\"\n_, ok := os.LookupEnv(\"MISSING\")\nok")
		require.NoError(t, err)
		require.Equal(t, false, result)
	})

	t.Run("Atoi success", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		service.UseSymbols(newStrconvSymbols())
		result, err := service.Eval(context.Background(), "import \"strconv\"\nn, _ := strconv.Atoi(\"42\")\nn")
		require.NoError(t, err)
		require.Equal(t, int64(42), result)
	})

	t.Run("Atoi error", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		service.UseSymbols(newStrconvSymbols())
		result, err := service.Eval(context.Background(), "import \"strconv\"\n_, err := strconv.Atoi(\"bad\")\nerr != nil")
		require.NoError(t, err)
		require.Equal(t, true, result)
	})

	t.Run("ParseBool success", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		service.UseSymbols(newStrconvSymbols())
		result, err := service.Eval(context.Background(), "import \"strconv\"\nv, _ := strconv.ParseBool(\"true\")\nv")
		require.NoError(t, err)
		require.Equal(t, true, result)
	})
}

func TestMultiReturnMethodCall(t *testing.T) {
	t.Parallel()

	t.Run("value receiver multi-return", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		code := `type Point struct { X int; Y int }
func (p Point) Coords() (int, int) { return p.X, p.Y }
p := Point{X: 3, Y: 7}
a, b := p.Coords()
a + b`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(10), result)
	})

	t.Run("pointer receiver multi-return", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		code := `type Pair struct { A string; B string }
func (p *Pair) Values() (string, string) { return p.A, p.B }
pp := &Pair{A: "hello", B: "world"}
a, b := pp.Values()
a + " " + b`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, "hello world", result)
	})

	t.Run("method multi-return with comma-ok pattern", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		code := `type Parser struct{ Prefix string }
func (p Parser) Parse(s string) (int, bool) {
	if s == p.Prefix { return 1, true }
	return 0, false
}
pr := Parser{Prefix: "ok"}
v, _ := pr.Parse("ok")
v`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(1), result)
	})
}

func TestMultiReturnNonIdentLHS(t *testing.T) {
	t.Parallel()

	t.Run("struct field LHS", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		code := `type Result struct { Val int; Ok bool }
func split() (int, bool) { return 42, true }
var r Result
r.Val, r.Ok = split()
r.Val`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(42), result)
	})

	t.Run("struct field LHS second value", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		code := `type Result struct { Val int; Ok bool }
func split() (int, bool) { return 42, true }
var r Result
r.Val, r.Ok = split()
r.Ok`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, true, result)
	})

	t.Run("map index LHS", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		code := `func pair() (string, bool) { return "yes", true }
m := map[string]string{}
var ok bool
m["k"], ok = pair()
m["k"]`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, "yes", result)
	})

	t.Run("mixed ident and selector LHS", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		code := `type State struct { N int }
func work() (int, bool) { return 99, true }
var s State
var ok bool
s.N, ok = work()
s.N`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(99), result)
	})

	t.Run("blank and map index", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		code := `func pair() (int, string) { return 1, "v" }
m := map[string]string{}
_, m["k"] = pair()
m["k"]`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, "v", result)
	})

	t.Run("selector call with struct field LHS", func(t *testing.T) {
		t.Parallel()
		service := NewService(WithEnv(map[string]string{"KEY": "value"}))
		service.UseSymbols(newOSSymbols())
		code := `import "os"
type Res struct { Val string; Ok bool }
var r Res
r.Val, r.Ok = os.LookupEnv("KEY")
r.Val`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, "value", result)
	})
}
