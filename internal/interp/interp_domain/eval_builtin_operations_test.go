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
		{expect: int64(3), name: "append_int_result", code: `s := []int{1}; s = append(s, 2, 3); s[2]`},
		{expect: "b", name: "append_string", code: `s := []string{"a"}; s = append(s, "b"); s[1]`},
		{expect: float64(2.5), name: "append_float", code: `s := []float64{1.0}; s = append(s, 2.5); s[1]`},
		{expect: false, name: "append_bool", code: `s := []bool{true}; s = append(s, false); s[1]`},
		{expect: int64(5), name: "append_len", code: `s := []int{1}; s = append(s, 2, 3, 4, 5); len(s)`},
		{expect: int64(42), name: "append_empty", code: `s := make([]int, 0); s = append(s, 42); s[0]`},
		{expect: int64(10), name: "append_to_nil", code: `var s []int; s = append(s, 10); s[0]`},
	}

	runEvalTable(t, nil, tests)
}

func TestBuiltinCap(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(5), name: "cap_make_slice", code: `s := make([]int, 3, 5); cap(s)`},
		{expect: int64(3), name: "cap_literal", code: `s := []int{1, 2, 3}; cap(s)`},
		{expect: int64(10), name: "cap_channel", code: `ch := make(chan int, 10); cap(ch)`},
	}

	runEvalTable(t, nil, tests)
}

func TestBuiltinCopy(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(6), name: "copy_full", code: `
src := []int{1, 2, 3}
dst := make([]int, 3)
copy(dst, src)
dst[0] + dst[1] + dst[2]`},
		{expect: int64(2), name: "copy_partial", code: `
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
		{expect: int64(0), name: "clear_slice", code: `s := []int{1, 2, 3}; clear(s); s[0]`},
		{expect: int64(0), name: "clear_map", code: `m := map[string]int{"a": 1, "b": 2}; clear(m); len(m)`},
	}

	runEvalTable(t, nil, tests)
}

func TestBuiltinDelete(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(1), name: "delete_existing", code: `m := map[string]int{"a": 1, "b": 2}; delete(m, "a"); len(m)`},
		{expect: int64(2), name: "delete_nonexistent", code: `m := map[string]int{"a": 1, "b": 2}; delete(m, "z"); len(m)`},
	}

	runEvalTable(t, nil, tests)
}

func TestMapCommaOkExtended(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: true, name: "map_comma_ok_exists", code: `
m := map[string]int{"key": 42}
_, ok := m["key"]
ok`},
		{expect: false, name: "map_comma_ok_missing", code: `
m := map[string]int{"key": 42}
_, ok := m["other"]
ok`},
		{expect: int64(42), name: "map_comma_ok_value", code: `
m := map[string]int{"key": 42}
v, ok := m["key"]
result := 0
if ok { result = v }
result`},
		{expect: "hello", name: "map_string_value_comma_ok", code: `
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
		{expect: int64(42), name: "method_expr_call", code: `
type Num struct { V int }
func (n Num) Value() int { return n.V }
x := Num{V: 42}
f := Num.Value
f(x)`},
		{expect: int64(10), name: "method_expr_pointer", code: `
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
		{expect: "hello", name: "slice_string_set_get", code: `s := make([]string, 2); s[0] = "hello"; s[0]`},
		{expect: true, name: "slice_bool_set_get", code: `s := make([]bool, 2); s[0] = true; s[0]`},
		{expect: false, name: "slice_bool_default", code: `s := make([]bool, 2); s[1]`},
	}

	runEvalTable(t, nil, tests)
}

func TestMapIntKeys(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(30), name: "map_int_int", code: `m := map[int]int{1: 10}; m[2] = 20; m[1] + m[2]`},
		{expect: int64(99), name: "map_int_overwrite", code: `m := map[int]int{1: 10}; m[1] = 99; m[1]`},
		{expect: int64(3), name: "map_int_len", code: `m := map[int]int{}; m[1] = 1; m[2] = 2; m[3] = 3; len(m)`},
	}

	runEvalTable(t, nil, tests)
}

func TestIndirectCalls(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(42), name: "call_func_var", code: `
f := func(x int) int { return x * 2 }
f(21)`},
		{expect: int64(31), name: "call_func_in_slice", code: `
fns := []func(int) int{
	func(x int) int { return x + 1 },
	func(x int) int { return x * 2 },
}
fns[0](10) + fns[1](10)`},
		{expect: int64(15), name: "call_func_from_map", code: `
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
		{expect: int64(42), name: "select_send", code: `
ch := make(chan int, 1)
select { case ch <- 42: }
<-ch`},
		{expect: int64(-1), name: "select_default", code: `
ch := make(chan int)
x := 0
select {
case v := <-ch:
	x = v
default:
	x = -1
}
x`},
		{expect: int64(42), name: "select_recv_value", code: `
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
		{expect: "hello world", name: "compound_string_concat", code: `s := "hello"; s += " world"; s`},
		{expect: int64(0xFF), name: "compound_bitor", code: `x := 0xF0; x |= 0x0F; x`},
		{expect: int64(0x0F), name: "compound_bitand", code: `x := 0xFF; x &= 0x0F; x`},
		{expect: int64(16), name: "compound_shl", code: `x := 1; x <<= 4; x`},
		{expect: int64(4), name: "compound_shr", code: `x := 16; x >>= 2; x`},
		{expect: int64(0xF0), name: "compound_xor", code: `x := 0xFF; x ^= 0x0F; x`},
	}

	runEvalTable(t, nil, tests)
}

func TestInterfaceMethodCalls(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(42), name: "interface_method_call", code: `
type Getter interface { Get() int }
type Box struct { V int }
func (b Box) Get() int { return b.V }
var g Getter = Box{V: 42}
g.Get()`},
		{expect: "hello", name: "interface_string_method", code: `
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
		expect any
		name   string
		code   string
	}{
		{name: "any_to_int", code: `var x any = 42; x.(int)`, expect: int(42)},
		{name: "any_to_string", code: `var x any = "hello"; x.(string)`, expect: "hello"},
		{name: "any_to_float", code: `var x any = 3.14; x.(float64)`, expect: float64(3.14)},
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
		expect any
		name   string
		code   string
	}{

		{name: "strings_Replace", code: "import \"strings\"\nstrings.Replace(\"aaa\", \"a\", \"b\", 2)", expect: "bba"},
		{name: "strings_ToTitle", code: "import \"strings\"\nstrings.ToTitle(\"hello world\")", expect: "HELLO WORLD"},
		{name: "strings_TrimLeft", code: "import \"strings\"\nstrings.TrimLeft(\"  hello\", \" \")", expect: "hello"},
		{name: "strings_TrimRight", code: "import \"strings\"\nstrings.TrimRight(\"hello  \", \" \")", expect: "hello"},

		{name: "strconv_Atoi", code: "import \"strconv\"\nv, _ := strconv.Atoi(\"42\")\nv", expect: int64(42)},
		{name: "strconv_ParseBool", code: "import \"strconv\"\nv, _ := strconv.ParseBool(\"true\")\nv", expect: true},

		{name: "fmt_Sprintf_int", code: "import \"fmt\"\nfmt.Sprintf(\"%d\", 42)", expect: "42"},
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
		{expect: int64(42), name: "basic_eval", code: `42`},
		{expect: int64(3), name: "simple_addition", code: `1 + 2`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchCompositeOps(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(60), name: "range_sum", code: `sum := 0; for _, v := range []int{10, 20, 30} { sum += v }; sum`},
		{expect: int64(2), name: "range_index", code: `last := 0; for i := range []int{10, 20, 30} { last = i }; last`},
		{expect: "hi", name: "slice_string", code: `s := make([]string, 1); s[0] = "hi"; s[0]`},
		{expect: int64(30), name: "map_int_int", code: `m := map[int]int{1: 10}; m[2] = 20; m[1] + m[2]`},
		{expect: int64(42), name: "addr_deref", code: `x := 42; p := &x; *p`},
		{expect: int64(20), name: "addr_modify", code: `x := 10; p := &x; *p = 20; x`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestGoDispatchControlFlow(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(10), name: "switch_multi_case", code: `
x := 3
y := 0
switch x {
case 1, 2, 3:
	y = 10
case 4, 5:
	y = 20
}
y`},
		{expect: "other", name: "type_switch_default", code: `
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
		{expect: int64(42), name: "map_comma_ok", code: `
m := map[string]int{"a": 42}
v, ok := m["a"]
result := 0
if ok { result = v }
result`},
		{expect: int64(6), name: "compound_selector", code: `
type S struct { X int }
s := S{X: 3}
s.X *= 2
s.X`},
		{expect: int64(42), name: "interface_method", code: `
type Getter interface { Get() int }
type Box struct { V int }
func (b Box) Get() int { return b.V }
var g Getter = Box{V: 42}
g.Get()`},
		{expect: int64(21), name: "multi_return", code: `
func swap(a, b int) (int, int) { return b, a }
x, y := swap(1, 2)
x*10 + y`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestGoDispatchNamedReturns(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(42), name: "named_return", code: `
func f() (result int) {
	result = 42
	return
}
f()`},
		{expect: int64(30), name: "named_return_multi", code: `
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
		{expect: int64(3), name: "counter_closure", code: `
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
		{expect: int64(42), name: "captured_var", code: `
func f() int {
	x := 42
	g := func() int { return x }
	return g()
}
f()`},
		{expect: int64(42), name: "func_var_call", code: `
f := func(x int) int { return x * 2 }
f(21)`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestGoDispatchDefer(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{expect: int64(10), name: "defer_named_return", code: `
func f() (x int) {
	defer func() { x += 10 }()
	x = 0
	return
}
f()`},
		{expect: "", name: "recover_from_panic", code: `
func f() string {
	defer func() { recover() }()
	panic("boom")
}
f()`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}
