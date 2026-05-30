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

//go:build bench

package interp_test_bench

import (
	"context"
	"testing"

	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
)

func runHotspotBenchmarks(b *testing.B, benchmarks []struct {
	name   string
	source string
}) {
	b.Helper()
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			service := interp_domain.NewService()
			cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": bm.source})
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err = service.ExecuteEntrypoint(context.Background(), cfs, "run")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func runHotspotBenchmarksWithStdlib(b *testing.B, benchmarks []struct {
	name   string
	source string
}) {
	b.Helper()
	symbols := interp_domain.NewSymbolRegistry(driven_system_symbols.NewProvider().Exports())
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			service := interp_domain.NewService()
			service.UseSymbols(symbols)
			cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": bm.source})
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err = service.ExecuteEntrypoint(context.Background(), cfs, "run")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkBoundaryCopy(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"struct_arg_8_field",
			`package main

type Wide struct{ a, b, c, d, e, f, g, h int }

func consume(w Wide) int { return w.a + w.h }

func run() int {
	w := Wide{a: 1, h: 8}
	s := 0
	for i := 0; i < 1000; i++ {
		s += consume(w)
	}
	return s
}
`,
		},
		{
			"struct_arg_2_field",
			`package main

type Pair struct{ a, b int }

func consume(p Pair) int { return p.a + p.b }

func run() int {
	p := Pair{a: 1, b: 2}
	s := 0
	for i := 0; i < 1000; i++ {
		s += consume(p)
	}
	return s
}
`,
		},
		{
			"struct_assign_local",
			`package main

type Wide struct{ a, b, c, d, e, f, g, h int }

func run() int {
	src := Wide{a: 1, h: 8}
	s := 0
	for i := 0; i < 1000; i++ {
		dst := src
		dst.a = i
		s += dst.a + dst.h
	}
	return s
}
`,
		},
		{
			"struct_field_read_inner",
			`package main

type Inner struct{ a, b, c, d int }
type Outer struct{ inner Inner }

func run() int {
	o := Outer{inner: Inner{a: 1, d: 4}}
	s := 0
	for i := 0; i < 1000; i++ {
		x := o.inner
		s += x.a + x.d
	}
	return s
}
`,
		},
		{
			"struct_slice_element_read",
			`package main

type Cell struct{ a, b, c, d int }

func run() int {
	cells := make([]Cell, 100)
	for i := 0; i < 100; i++ {
		cells[i] = Cell{a: i, d: i + 1}
	}
	s := 0
	for i := 0; i < 1000; i++ {
		x := cells[i%100]
		s += x.a + x.d
	}
	return s
}
`,
		},
		{
			"struct_map_element_read",
			`package main

type Cell struct{ a, b, c, d int }

func run() int {
	m := make(map[int]Cell, 100)
	for i := 0; i < 100; i++ {
		m[i] = Cell{a: i, d: i + 1}
	}
	s := 0
	for i := 0; i < 1000; i++ {
		x := m[i%100]
		s += x.a + x.d
	}
	return s
}
`,
		},
		{
			"struct_range_value",
			`package main

type Cell struct{ a, b, c, d int }

func run() int {
	cells := make([]Cell, 1000)
	for i := 0; i < 1000; i++ {
		cells[i] = Cell{a: i, d: i + 1}
	}
	s := 0
	for _, v := range cells {
		s += v.a + v.d
	}
	return s
}
`,
		},
		{
			"struct_array_range_value",
			`package main

type Cell struct{ a, b, c, d int }

func run() int {
	var arr [100]Cell
	for i := 0; i < 100; i++ {
		arr[i] = Cell{a: i, d: i + 1}
	}
	s := 0
	for j := 0; j < 10; j++ {
		for _, v := range arr {
			s += v.a + v.d
		}
	}
	return s
}
`,
		},
		{
			"struct_type_assert",
			`package main

type Cell struct{ a, b, c, d int }

func run() int {
	var x interface{} = Cell{a: 7, d: 13}
	s := 0
	for i := 0; i < 1000; i++ {
		c := x.(Cell)
		s += c.a + c.d
	}
	return s
}
`,
		},
		{
			"struct_chan_pingpong",
			`package main

type Cell struct{ a, b, c, d int }

func run() int {
	ch := make(chan Cell, 1)
	s := 0
	for i := 0; i < 1000; i++ {
		ch <- Cell{a: i, d: i + 1}
		c := <-ch
		s += c.a + c.d
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkInterfaceDispatch(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"monomorphic_value_receiver",
			`package main

type Greeter interface{ Greet() int }

type En struct{ n int }
func (e En) Greet() int { return e.n }

func run() int {
	var g Greeter = En{n: 1}
	s := 0
	for i := 0; i < 1000; i++ {
		s += g.Greet()
	}
	return s
}
`,
		},
		{
			"monomorphic_pointer_receiver",
			`package main

type Counter interface{ Inc() }
type C struct{ n int }
func (c *C) Inc() { c.n++ }

func run() int {
	c := &C{}
	var ci Counter = c
	for i := 0; i < 1000; i++ {
		ci.Inc()
	}
	return c.n
}
`,
		},
		{
			"polymorphic_two_types",
			`package main

type Shape interface{ Area() int }

type Square struct{ side int }
func (s Square) Area() int { return s.side * s.side }

type Rect struct{ w, h int }
func (r Rect) Area() int { return r.w * r.h }

func run() int {
	shapes := []Shape{Square{side: 3}, Rect{w: 4, h: 5}}
	s := 0
	for i := 0; i < 1000; i++ {
		s += shapes[i&1].Area()
	}
	return s
}
`,
		},
		{
			"direct_value_receiver",
			`package main

type En struct{ n int }
func (e En) Greet() int { return e.n }

func run() int {
	g := En{n: 1}
	s := 0
	for i := 0; i < 1000; i++ {
		s += g.Greet()
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkGenericDispatch(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"identity_int",
			`package main

func id[T any](x T) T { return x }

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += id(i)
	}
	return s
}
`,
		},
		{
			"identity_string_len",
			`package main

func id[T any](x T) T { return x }

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += len(id("hello"))
	}
	return s
}
`,
		},
		{
			"compare_constrained",
			`package main

type Ordered interface{ ~int | ~int64 | ~float64 }

func max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += max(i, 500)
	}
	return s
}
`,
		},
		{
			"non_generic_baseline",
			`package main

func id(x int) int { return x }

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += id(i)
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkRangePatterns(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"range_over_int",
			`package main

func run() int {
	s := 0
	for i := range 1000 {
		s += i
	}
	return s
}
`,
		},
		{
			"for_i_baseline",
			`package main

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += i
	}
	return s
}
`,
		},
		{
			"range_over_func",
			`package main

func upTo(n int) func(yield func(int) bool) {
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func run() int {
	s := 0
	for v := range upTo(1000) {
		s += v
	}
	return s
}
`,
		},
		{
			"range_over_int_slice",
			`package main

func run() int {
	xs := make([]int, 1000)
	for i := range xs {
		xs[i] = i
	}
	s := 0
	for _, v := range xs {
		s += v
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkDeferOverhead(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"no_defer",
			`package main

func work(x int) int { return x * 2 }

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += work(i)
	}
	return s
}
`,
		},
		{
			"single_defer",
			`package main

func work(x int) int {
	defer func() {}()
	return x * 2
}

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += work(i)
	}
	return s
}
`,
		},
		{
			"defer_with_capture",
			`package main

func work(x int) int {
	y := x + 1
	defer func() { _ = y }()
	return x * 2
}

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += work(i)
	}
	return s
}
`,
		},
		{
			"recover_no_panic",
			`package main

func work(x int) int {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()
	return x * 2
}

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += work(i)
	}
	return s
}
`,
		},
		{
			"single_defer_named_function",
			`package main

func cleanup() {}

func work(x int) int {
	defer cleanup()
	return x * 2
}

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += work(i)
	}
	return s
}
`,
		},
		{
			"single_defer_method_value",
			`package main

type counter struct{ n int }
func (c *counter) inc() { c.n++ }

func work(c *counter, x int) int {
	defer c.inc()
	return x * 2
}

func run() int {
	c := &counter{}
	s := 0
	for i := 0; i < 1000; i++ {
		s += work(c, i)
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkClosureCapture(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"capture_int_read",
			`package main

func make(x int) func() int {
	return func() int { return x }
}

func run() int {
	f := make(42)
	s := 0
	for i := 0; i < 1000; i++ {
		s += f()
	}
	return s
}
`,
		},
		{
			"capture_int_write",
			`package main

func make() func(int) int {
	x := 0
	return func(d int) int {
		x += d
		return x
	}
}

func run() int {
	f := make()
	s := 0
	for i := 0; i < 1000; i++ {
		s = f(1)
	}
	return s
}
`,
		},
		{
			"capture_struct_read",
			`package main

type State struct{ a, b, c, d int }

func make(s State) func() int {
	return func() int { return s.a + s.d }
}

func run() int {
	f := make(State{a: 1, d: 4})
	s := 0
	for i := 0; i < 1000; i++ {
		s += f()
	}
	return s
}
`,
		},
		{
			"capture_pointer_write",
			`package main

func make(p *int) func(int) {
	return func(d int) { *p += d }
}

func run() int {
	x := 0
	f := make(&x)
	for i := 0; i < 1000; i++ {
		f(1)
	}
	return x
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkChannelThroughput(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"buffered_int_pingpong_1000",
			`package main

func run() int {
	ch := make(chan int, 1)
	s := 0
	for i := 0; i < 1000; i++ {
		ch <- i
		s += <-ch
	}
	return s
}
`,
		},
		{
			"buffered_int_throughput_1000",
			`package main

func run() int {
	ch := make(chan int, 1024)
	for i := 0; i < 1000; i++ {
		ch <- i
	}
	s := 0
	for i := 0; i < 1000; i++ {
		s += <-ch
	}
	return s
}
`,
		},
		{
			"unbuffered_goroutine_pingpong_100",
			`package main

func run() int {
	ch := make(chan int)
	go func() {
		for i := 0; i < 100; i++ {
			ch <- i
		}
		close(ch)
	}()
	s := 0
	for v := range ch {
		s += v
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkNestedFieldAccess(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"depth_1",
			`package main

type S1 struct{ x int }

func run() int {
	s := S1{x: 7}
	t := 0
	for i := 0; i < 1000; i++ {
		t += s.x
	}
	return t
}
`,
		},
		{
			"depth_4_value",
			`package main

type Inner struct{ x int }
type Mid struct{ inner Inner }
type Outer struct{ mid Mid }
type Top struct{ outer Outer }

func run() int {
	t := Top{outer: Outer{mid: Mid{inner: Inner{x: 7}}}}
	s := 0
	for i := 0; i < 1000; i++ {
		s += t.outer.mid.inner.x
	}
	return s
}
`,
		},
		{
			"depth_4_pointer_chain",
			`package main

type Inner struct{ x int }
type Mid struct{ inner *Inner }
type Outer struct{ mid *Mid }
type Top struct{ outer *Outer }

func run() int {
	t := &Top{outer: &Outer{mid: &Mid{inner: &Inner{x: 7}}}}
	s := 0
	for i := 0; i < 1000; i++ {
		s += t.outer.mid.inner.x
	}
	return s
}
`,
		},
		{
			"embedded_method_promotion",
			`package main

type Base struct{ n int }
func (b Base) Get() int { return b.n }

type Wrap struct{ Base }

func run() int {
	w := Wrap{Base: Base{n: 7}}
	s := 0
	for i := 0; i < 1000; i++ {
		s += w.Get()
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkTypeSwitchBreadth(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"4_cases",
			`package main

func classify(v interface{}) int {
	switch v.(type) {
	case int:
		return 1
	case string:
		return 2
	case bool:
		return 3
	case float64:
		return 4
	}
	return 0
}

func run() int {
	values := []interface{}{1, "x", true, 3.14}
	s := 0
	for i := 0; i < 1000; i++ {
		s += classify(values[i&3])
	}
	return s
}
`,
		},
		{
			"8_cases_first_match",
			`package main

func classify(v interface{}) int {
	switch v.(type) {
	case int:
		return 1
	case int8:
		return 2
	case int16:
		return 3
	case int32:
		return 4
	case int64:
		return 5
	case string:
		return 6
	case bool:
		return 7
	case float64:
		return 8
	}
	return 0
}

func run() int {
	x := 42
	s := 0
	for i := 0; i < 1000; i++ {
		var v interface{} = x
		s += classify(v)
	}
	return s
}
`,
		},
		{
			"8_cases_last_match",
			`package main

func classify(v interface{}) int {
	switch v.(type) {
	case int8:
		return 1
	case int16:
		return 2
	case int32:
		return 3
	case int64:
		return 4
	case uint:
		return 5
	case uint8:
		return 6
	case uint16:
		return 7
	case float64:
		return 8
	}
	return 0
}

func run() int {
	x := 3.14
	s := 0
	for i := 0; i < 1000; i++ {
		var v interface{} = x
		s += classify(v)
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkSliceOfStruct(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"read_via_index",
			`package main

type Cell struct{ a, b, c, d int }

func run() int {
	cells := make([]Cell, 1000)
	for i := 0; i < 1000; i++ {
		cells[i] = Cell{a: i, d: i + 1}
	}
	s := 0
	for i := 0; i < 1000; i++ {
		s += cells[i].a + cells[i].d
	}
	return s
}
`,
		},
		{
			"write_field_via_index",
			`package main

type Cell struct{ a, b, c, d int }

func run() int {
	cells := make([]Cell, 1000)
	for i := 0; i < 1000; i++ {
		cells[i].a = i
		cells[i].d = i + 1
	}
	s := 0
	for i := 0; i < 1000; i++ {
		s += cells[i].a
	}
	return s
}
`,
		},
		{
			"compound_assign_via_index",
			`package main

type Cell struct{ a, b, c, d int }

func run() int {
	cells := make([]Cell, 1000)
	for i := 0; i < 1000; i++ {
		cells[i].a += 1
		cells[i].d += 2
	}
	s := 0
	for i := 0; i < 1000; i++ {
		s += cells[i].a + cells[i].d
	}
	return s
}
`,
		},
		{
			"pointer_receiver_method_via_index",
			`package main

type Cell struct{ n int }
func (c *Cell) Inc() { c.n++ }

func run() int {
	cells := make([]Cell, 1000)
	for i := 0; i < 1000; i++ {
		cells[i].Inc()
	}
	s := 0
	for i := 0; i < 1000; i++ {
		s += cells[i].n
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkMapStringKey(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"set_then_get_100",
			`package main

func run() int {
	m := make(map[string]int, 100)
	keys := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	for i, k := range keys {
		m[k] = i
	}
	s := 0
	for i := 0; i < 1000; i++ {
		s += m[keys[i%10]]
	}
	return s
}
`,
		},
		{
			"comma_ok",
			`package main

func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	s := 0
	for i := 0; i < 1000; i++ {
		if v, ok := m["a"]; ok {
			s += v
		}
	}
	return s
}
`,
		},
		{
			"delete_then_set",
			`package main

func run() int {
	m := make(map[string]int, 16)
	for i := 0; i < 100; i++ {
		m["k"] = i
		delete(m, "k")
	}
	return len(m)
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkASMTypeConversion(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"bool_to_int_loop",
			`package main

func run() int {
	b := true
	s := 0
	for i := 0; i < 1000; i++ {
		var v int
		if b {
			v = 1
		}
		s += v
	}
	return s
}
`,
		},
		{
			"int_to_uint_loop",
			`package main

func run() uint64 {
	var s uint64
	for i := 0; i < 1000; i++ {
		s += uint64(i)
	}
	return s
}
`,
		},
		{
			"uint_to_int_loop",
			`package main

func run() int64 {
	var u uint64 = 7
	var s int64
	for i := 0; i < 1000; i++ {
		s += int64(u)
	}
	return s
}
`,
		},
		{
			"uint_to_float_loop",
			`package main

func run() float64 {
	var u uint64 = 1234
	var s float64
	for i := 0; i < 1000; i++ {
		s += float64(u)
	}
	return s
}
`,
		},
		{
			"float_to_uint_loop",
			`package main

func run() uint64 {
	f := 3.14
	var s uint64
	for i := 0; i < 1000; i++ {
		s += uint64(f)
	}
	return s
}
`,
		},
		{
			"bitnot_uint_loop",
			`package main

func run() uint64 {
	var x uint64 = 0xFF
	var s uint64
	for i := 0; i < 1000; i++ {
		s += ^x
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkErrorWrap(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"errors_is_direct",
			`package main

import "errors"

var Sentinel = errors.New("boom")

func run() int {
	hits := 0
	for i := 0; i < 1000; i++ {
		if errors.Is(Sentinel, Sentinel) {
			hits++
		}
	}
	return hits
}
`,
		},
		{
			"errors_is_wrapped_depth_3",
			`package main

import (
	"errors"
	"fmt"
)

var Sentinel = errors.New("boom")

func wrap() error {
	return fmt.Errorf("layer3: %w", fmt.Errorf("layer2: %w", fmt.Errorf("layer1: %w", Sentinel)))
}

func run() int {
	hits := 0
	for i := 0; i < 100; i++ {
		err := wrap()
		if errors.Is(err, Sentinel) {
			hits++
		}
	}
	return hits
}
`,
		},
	}
	runHotspotBenchmarksWithStdlib(b, benchmarks)
}
