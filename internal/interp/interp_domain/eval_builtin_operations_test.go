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
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltinAppendVariants(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(3), "append_int_result", `s := []int{1}; s = append(s, 2, 3); s[2]`},
		{"b", "append_string", `s := []string{"a"}; s = append(s, "b"); s[1]`},
		{float64(2.5), "append_float", `s := []float64{1.0}; s = append(s, 2.5); s[1]`},
		{false, "append_bool", `s := []bool{true}; s = append(s, false); s[1]`},
		{int64(5), "append_len", `s := []int{1}; s = append(s, 2, 3, 4, 5); len(s)`},
		{int64(42), "append_empty", `s := make([]int, 0); s = append(s, 42); s[0]`},
		{int64(10), "append_to_nil", `var s []int; s = append(s, 10); s[0]`},
	}

	runEvalTable(t, nil, tests)
}

func TestBuiltinCap(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(5), "cap_make_slice", `s := make([]int, 3, 5); cap(s)`},
		{int64(3), "cap_literal", `s := []int{1, 2, 3}; cap(s)`},
		{int64(10), "cap_channel", `ch := make(chan int, 10); cap(ch)`},
	}

	runEvalTable(t, nil, tests)
}

func TestBuiltinCopy(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(6), "copy_full", `
src := []int{1, 2, 3}
dst := make([]int, 3)
copy(dst, src)
dst[0] + dst[1] + dst[2]`},
		{int64(2), "copy_partial", `
src := []int{1, 2, 3}
dst := make([]int, 2)
n := copy(dst, src)
n`},
	}

	runEvalTable(t, nil, tests)
}

func TestBuiltinClear(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(0), "clear_slice", `s := []int{1, 2, 3}; clear(s); s[0]`},
		{int64(0), "clear_map", `m := map[string]int{"a": 1, "b": 2}; clear(m); len(m)`},
	}

	runEvalTable(t, nil, tests)
}

func TestBuiltinDelete(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(1), "delete_existing", `m := map[string]int{"a": 1, "b": 2}; delete(m, "a"); len(m)`},
		{int64(2), "delete_nonexistent", `m := map[string]int{"a": 1, "b": 2}; delete(m, "z"); len(m)`},
	}

	runEvalTable(t, nil, tests)
}

func TestMapCommaOkExtended(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{true, "map_comma_ok_exists", `
m := map[string]int{"key": 42}
_, ok := m["key"]
ok`},
		{false, "map_comma_ok_missing", `
m := map[string]int{"key": 42}
_, ok := m["other"]
ok`},
		{int64(42), "map_comma_ok_value", `
m := map[string]int{"key": 42}
v, ok := m["key"]
result := 0
if ok { result = v }
result`},
		{"hello", "map_string_value_comma_ok", `
m := map[int]string{1: "hello"}
v, ok := m[1]
result := ""
if ok { result = v }
result`},
	}

	runEvalTable(t, nil, tests)
}

func TestMethodExpressions(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "method_expr_call", `
type Num struct { V int }
func (n Num) Value() int { return n.V }
x := Num{V: 42}
f := Num.Value
f(x)`},
		{int64(10), "method_expr_pointer", `
type Box struct { V int }
func (b *Box) Get() int { return b.V }
b := &Box{V: 10}
f := (*Box).Get
f(b)`},
	}

	runEvalTable(t, nil, tests)
}

func TestSliceOperationsExtended(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{"hello", "slice_string_set_get", `s := make([]string, 2); s[0] = "hello"; s[0]`},
		{true, "slice_bool_set_get", `s := make([]bool, 2); s[0] = true; s[0]`},
		{false, "slice_bool_default", `s := make([]bool, 2); s[1]`},
	}

	runEvalTable(t, nil, tests)
}

func TestMapIntKeys(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(30), "map_int_int", `m := map[int]int{1: 10}; m[2] = 20; m[1] + m[2]`},
		{int64(99), "map_int_overwrite", `m := map[int]int{1: 10}; m[1] = 99; m[1]`},
		{int64(3), "map_int_len", `m := map[int]int{}; m[1] = 1; m[2] = 2; m[3] = 3; len(m)`},
	}

	runEvalTable(t, nil, tests)
}

func TestIndirectCalls(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "call_func_var", `
f := func(x int) int { return x * 2 }
f(21)`},
		{int64(31), "call_func_in_slice", `
fns := []func(int) int{
	func(x int) int { return x + 1 },
	func(x int) int { return x * 2 },
}
fns[0](10) + fns[1](10)`},
		{int64(15), "call_func_from_map", `
m := map[string]func(int) int{
	"double": func(x int) int { return x * 2 },
	"add5":   func(x int) int { return x + 5 },
}
m["add5"](10)`},
	}

	runEvalTable(t, nil, tests)
}

func TestSelectStatements(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "select_send", `
ch := make(chan int, 1)
select { case ch <- 42: }
<-ch`},
		{int64(-1), "select_default", `
ch := make(chan int)
x := 0
select {
case v := <-ch:
	x = v
default:
	x = -1
}
x`},
		{int64(42), "select_recv_value", `
ch := make(chan int, 1)
ch <- 42
x := 0
select {
case v := <-ch:
	x = v
}
x`},
	}

	runEvalTable(t, nil, tests)
}

func TestCompoundAssignExtended(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{"hello world", "compound_string_concat", `s := "hello"; s += " world"; s`},
		{int64(0xFF), "compound_bitor", `x := 0xF0; x |= 0x0F; x`},
		{int64(0x0F), "compound_bitand", `x := 0xFF; x &= 0x0F; x`},
		{int64(16), "compound_shl", `x := 1; x <<= 4; x`},
		{int64(4), "compound_shr", `x := 16; x >>= 2; x`},
		{int64(0xF0), "compound_xor", `x := 0xFF; x ^= 0x0F; x`},
	}

	runEvalTable(t, nil, tests)
}

func TestInterfaceMethodCalls(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "interface_method_call", `
type Getter interface { Get() int }
type Box struct { V int }
func (b Box) Get() int { return b.V }
var g Getter = Box{V: 42}
g.Get()`},
		{"hello", "interface_string_method", `
type Stringer interface { String() string }
type Name struct { S string }
func (n Name) String() string { return n.S }
var s Stringer = Name{S: "hello"}
s.String()`},
	}

	runEvalTable(t, nil, tests)
}

func TestCrossBankMoves(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"any_to_int", `var x any = 42; x.(int)`, int(42)},
		{"any_to_string", `var x any = "hello"; x.(string)`, "hello"},
		{"any_to_float", `var x any = 3.14; x.(float64)`, float64(3.14)},
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

func TestNativeFastpathExtended(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{
		"strings": {
			"Contains":     reflect.ValueOf(strings.Contains),
			"ContainsRune": reflect.ValueOf(strings.ContainsRune),
			"Count":        reflect.ValueOf(strings.Count),
			"HasPrefix":    reflect.ValueOf(strings.HasPrefix),
			"HasSuffix":    reflect.ValueOf(strings.HasSuffix),
			"Index":        reflect.ValueOf(strings.Index),
			"IndexRune":    reflect.ValueOf(strings.IndexRune),
			"Repeat":       reflect.ValueOf(strings.Repeat),
			"Replace":      reflect.ValueOf(strings.Replace),
			"ToLower":      reflect.ValueOf(strings.ToLower),
			"ToTitle":      reflect.ValueOf(strings.ToTitle),
			"ToUpper":      reflect.ValueOf(strings.ToUpper),
			"TrimLeft":     reflect.ValueOf(strings.TrimLeft),
			"TrimRight":    reflect.ValueOf(strings.TrimRight),
		},
		"strconv": {
			"Atoi":       reflect.ValueOf(strconv.Atoi),
			"FormatBool": reflect.ValueOf(strconv.FormatBool),
			"FormatInt":  reflect.ValueOf(strconv.FormatInt),
			"Itoa":       reflect.ValueOf(strconv.Itoa),
			"ParseBool":  reflect.ValueOf(strconv.ParseBool),
			"ParseInt":   reflect.ValueOf(strconv.ParseInt),
		},
		"fmt": {
			"Sprintf": reflect.ValueOf(fmt.Sprintf),
		},
		"math": {
			"Abs":   reflect.ValueOf(math.Abs),
			"Ceil":  reflect.ValueOf(math.Ceil),
			"Floor": reflect.ValueOf(math.Floor),
			"Max":   reflect.ValueOf(math.Max),
			"Min":   reflect.ValueOf(math.Min),
			"Pow":   reflect.ValueOf(math.Pow),
			"Sqrt":  reflect.ValueOf(math.Sqrt),
		},
	})

	tests := []struct {
		name   string
		code   string
		expect any
	}{

		{"strings_Replace", "import \"strings\"\nstrings.Replace(\"aaa\", \"a\", \"b\", 2)", "bba"},
		{"strings_ToTitle", "import \"strings\"\nstrings.ToTitle(\"hello world\")", "HELLO WORLD"},
		{"strings_TrimLeft", "import \"strings\"\nstrings.TrimLeft(\"  hello\", \" \")", "hello"},
		{"strings_TrimRight", "import \"strings\"\nstrings.TrimRight(\"hello  \", \" \")", "hello"},

		{"strconv_Atoi", "import \"strconv\"\nv, _ := strconv.Atoi(\"42\")\nv", int64(42)},
		{"strconv_ParseBool", "import \"strconv\"\nv, _ := strconv.ParseBool(\"true\")\nv", true},

		{"fmt_Sprintf_int", "import \"fmt\"\nfmt.Sprintf(\"%d\", 42)", "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			localService := service.Clone()
			result, err := localService.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestBuildTagFiltering(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "basic_eval", `42`},
		{int64(3), "simple_addition", `1 + 2`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchCompositeOps(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(60), "range_sum", `sum := 0; for _, v := range []int{10, 20, 30} { sum += v }; sum`},
		{int64(2), "range_index", `last := 0; for i := range []int{10, 20, 30} { last = i }; last`},
		{"hi", "slice_string", `s := make([]string, 1); s[0] = "hi"; s[0]`},
		{int64(30), "map_int_int", `m := map[int]int{1: 10}; m[2] = 20; m[1] + m[2]`},
		{int64(42), "addr_deref", `x := 42; p := &x; *p`},
		{int64(20), "addr_modify", `x := 10; p := &x; *p = 20; x`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestGoDispatchControlFlow(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(10), "switch_multi_case", `
x := 3
y := 0
switch x {
case 1, 2, 3:
	y = 10
case 4, 5:
	y = 20
}
y`},
		{"other", "type_switch_default", `
var x any = 3.14
result := ""
switch x.(type) {
case int:
	result = "int"
case string:
	result = "string"
default:
	result = "other"
}
result`},
		{int64(42), "map_comma_ok", `
m := map[string]int{"a": 42}
v, ok := m["a"]
result := 0
if ok { result = v }
result`},
		{int64(6), "compound_selector", `
type S struct { X int }
s := S{X: 3}
s.X *= 2
s.X`},
		{int64(42), "interface_method", `
type Getter interface { Get() int }
type Box struct { V int }
func (b Box) Get() int { return b.V }
var g Getter = Box{V: 42}
g.Get()`},
		{int64(21), "multi_return", `
func swap(a, b int) (int, int) { return b, a }
x, y := swap(1, 2)
x*10 + y`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestGoDispatchNamedReturns(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "named_return", `
func f() (result int) {
	result = 42
	return
}
f()`},
		{int64(30), "named_return_multi", `
func f() (a int, b int) {
	a = 10
	b = 20
	return
}
x, y := f()
x + y`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestGoDispatchClosures(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(3), "counter_closure", `
func makeCounter() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}
c := makeCounter()
c()
c()
c()`},
		{int64(42), "captured_var", `
func f() int {
	x := 42
	g := func() int { return x }
	return g()
}
f()`},
		{int64(42), "func_var_call", `
f := func(x int) int { return x * 2 }
f(21)`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestGoDispatchDefer(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(10), "defer_named_return", `
func f() (x int) {
	defer func() { x += 10 }()
	x = 0
	return
}
f()`},
		{"", "recover_from_panic", `
func f() string {
	defer func() { recover() }()
	panic("boom")
}
f()`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}
