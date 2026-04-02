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

func TestEdgeCaseRangeClosureCapture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "map_range_closure_capture",
			source: `package main

func run() int {
	m := map[int]int{0: 10, 1: 20, 2: 30}
	fns := make([]func() int, 3)
	for k, v := range m {
		v := v
		fns[k] = func() int { return v }
	}
	return fns[0]() + fns[1]() + fns[2]()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(60),
		},
		{
			name: "channel_range_closure_capture",
			source: `package main

func run() int {
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	close(ch)
	fns := make([]func() int, 3)
	idx := 0
	for v := range ch {
		v := v
		i := idx
		fns[i] = func() int { return v }
		idx++
	}
	return fns[0]() + fns[1]() + fns[2]()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(60),
		},
		{
			name: "string_range_closure_capture",
			source: `package main

func run() int {
	fns := make([]func() int, 3)
	idx := 0
	for _, r := range "abc" {
		r := r
		i := idx
		fns[i] = func() int { return int(r) }
		idx++
	}
	return fns[0]() + fns[1]() + fns[2]()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(294),
		},
		{
			name: "map_range_closure_no_shadow",
			source: `package main

func run() int {
	m := map[int]int{0: 100, 1: 200, 2: 300}
	fns := make([]func() int, 3)
	for k, v := range m {
		fns[k] = func() int { return v }
	}
	return fns[0]() + fns[1]() + fns[2]()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(600),
		},
		{
			name: "slice_range_value_closure",
			source: `package main

func run() int {
	fns := make([]func() int, 3)
	for i, v := range []int{10, 20, 30} {
		fns[i] = func() int { return v }
	}
	return fns[0]() + fns[1]() + fns[2]()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(60),
		},
		{
			name: "slice_range_kv_closure",
			source: `package main

func run() int {
	fns := make([]func() int, 3)
	for i, v := range []int{10, 20, 30} {
		fns[i] = func() int { return i*100 + v }
	}
	return fns[0]() + fns[1]() + fns[2]()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(360),
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

func TestEdgeCaseRecursiveClosures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "recursive_closure_variable",
			source: `package main

func run() int {
	var f func(int) int
	f = func(n int) int {
		if n <= 1 {
			return 1
		}
		return n * f(n-1)
	}
	return f(5)
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(120),
		},
		{
			name: "mutual_recursion_closures",
			source: `package main

func run() bool {
	var isEven, isOdd func(int) bool
	isEven = func(n int) bool {
		if n == 0 { return true }
		return isOdd(n - 1)
	}
	isOdd = func(n int) bool {
		if n == 0 { return false }
		return isEven(n - 1)
	}
	return isEven(10)
}

func main() {}
`,
			entrypoint: "run",
			expect:     true,
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

func TestEdgeCaseTypeSwitch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "multi_types_per_case",
			source: `package main

func run() string {
	var x any = 42
	switch x.(type) {
	case int, float64:
		return "numeric"
	case string:
		return "text"
	default:
		return "other"
	}
}

func main() {}
`,
			entrypoint: "run",
			expect:     "numeric",
		},
		{
			name: "nil_case",
			source: `package main

func run() string {
	var x any
	switch x.(type) {
	case nil:
		return "nil"
	case int:
		return "int"
	default:
		return "other"
	}
}

func main() {}
`,
			entrypoint: "run",
			expect:     "nil",
		},
		{
			name: "binding_multi_type",
			source: `package main

func run() int {
	var x any = 3.14
	switch v := x.(type) {
	case int, float64:
		_ = v
		return 1
	case string:
		return 2
	default:
		return 3
	}
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(1),
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

func TestEdgeCaseCompositeNesting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "map_of_slices",
			source: `package main

func run() int {
	m := map[string][]int{
		"a": {1, 2, 3},
		"b": {4, 5},
	}
	m["a"] = append(m["a"], 4)
	return len(m["a"]) + m["b"][1]
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(9),
		},
		{
			name: "slice_of_maps",
			source: `package main

func run() int {
	s := []map[string]int{
		{"x": 1},
		{"x": 2, "y": 3},
	}
	s[1]["z"] = 4
	return s[0]["x"] + s[1]["y"] + s[1]["z"]
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(8),
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

func TestEdgeCaseDeepNestedClosure(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	x := 0
	f := func() func() func() int {
		return func() func() int {
			return func() int {
				x++
				return x
			}
		}
	}
	g := f()()
	g()
	g()
	return g()
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(3), result)
}

func TestEdgeCaseDeferNamedReturns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "defer_multi_named_return",
			source: `package main

func helper() (a int, b int) {
	a = 10
	b = 100
	defer func() {
		a *= 2
		b *= 2
	}()
	return
}

func run() int {
	a, b := helper()
	return a + b
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(220),
		},
		{
			name: "defer_modify_explicit_return",
			source: `package main

func modified() (result int) {
	defer func() { result += 10 }()
	return 32
}

func run() int {
	return modified()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
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

func TestEdgeCaseSelectChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "select_nil_channel_default",
			source: `package main

func run() int {
	var ch chan int
	select {
	case <-ch:
		return 1
	default:
		return 2
	}
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(2),
		},
		{
			name: "closure_in_select_body",
			source: `package main

func run() int {
	ch := make(chan int, 1)
	ch <- 42
	var f func() int
	select {
	case v := <-ch:
		f = func() int { return v }
	}
	return f()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
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

func TestEdgeCaseGoroutines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "goroutine_loop_capture",
			source: `package main

func run() int {
	ch := make(chan int, 5)
	for i := range 5 {
		go func() {
			ch <- i
		}()
	}
	sum := 0
	for range 5 {
		sum += <-ch
	}
	return sum
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(10),
		},
		{
			name: "goroutine_panic_recover",
			source: `package main

func run() int {
	ch := make(chan int, 1)
	go func() {
		defer func() {
			recover()
			ch <- 42
		}()
		panic("boom")
	}()
	return <-ch
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
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

func TestEdgeCaseNumeric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{

			name: "uint_max_plus_one_no_subword_mask",
			source: `package main

func run() uint8 {
	var x uint8 = 255
	x += 1
	return x
}

func main() {}
`,
			entrypoint: "run",
			expect:     uint64(256),
		},
		{
			name: "float_div_zero_inf",
			source: `package main

func run() bool {
	x := 1.0
	y := 0.0
	z := x / y
	return z > 1e308
}

func main() {}
`,
			entrypoint: "run",
			expect:     true,
		},
		{
			name: "const_shift_expr",
			source: `package main

const x = 1 << 10

func run() int {
	return x
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(1024),
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

func TestEdgeCasePointers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "pointer_to_pointer_deref",
			source: `package main

func run() int {
	x := 42
	p := &x
	pp := &p
	return **pp
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
		},
		{
			name: "double_pointer_mutation",
			source: `package main

func run() int {
	x := 1
	p := &x
	pp := &p
	**pp = 99
	return x
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(99),
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

func TestEdgeCaseInterfaceMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "interface_reassignment",
			source: `package main

type Stringer interface{ String() string }
type A struct{}
func (A) String() string { return "A" }
type B struct{}
func (B) String() string { return "B" }

func run() string {
	var s Stringer
	s = A{}
	r1 := s.String()
	s = B{}
	r2 := s.String()
	return r1 + r2
}

func main() {}
`,
			entrypoint: "run",
			expect:     "AB",
		},
		{
			name: "method_value_receiver_copy",
			source: `package main

type Box struct{ V int }
func (b Box) Get() int { return b.V }

func run() int {
	b := Box{V: 10}
	f := b.Get
	b.V = 99
	return f()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(10),
		},
		{
			name: "field_shadow_embedding",
			source: `package main

type Base struct{ V int }
type Derived struct {
	Base
	V int
}

func run() int {
	d := Derived{Base: Base{V: 1}, V: 2}
	return d.V + d.Base.V
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(3),
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

func TestEdgeCaseScopeVariable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "short_var_redecl",
			source: `package main

func run() int {
	a := 10
	a, b := 20, 30
	return a + b
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(50),
		},
		{
			name: "range_string_runes",
			source: `package main

func run() int {
	sum := 0
	for _, r := range "ABC" {
		sum += int(r)
	}
	return sum
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(198),
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
