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
	"testing"
)

func BenchmarkStructFieldRead(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"read_int64_field",
			`package main

type T struct{ a, b, c, d int64 }

var s = T{a: 7, b: 11, c: 13, d: 17}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += s.a + s.d
	}
	return total
}
`,
		},
		{
			"read_int32_field",
			`package main

type T struct{ a, b, c, d int32 }

var s = T{a: 7, b: 11, c: 13, d: 17}

func run() int32 {
	total := int32(0)
	for i := 0; i < 1000; i++ {
		total += s.a + s.d
	}
	return total
}
`,
		},
		{
			"read_int8_field",
			`package main

type T struct{ a, b, c, d int8 }

var s = T{a: 7, b: 11, c: 13, d: 17}

func run() int8 {
	total := int8(0)
	for i := 0; i < 1000; i++ {
		total += s.a + s.d
	}
	return total
}
`,
		},
		{
			"read_uint64_field",
			`package main

type T struct{ a, b, c, d uint64 }

var s = T{a: 7, b: 11, c: 13, d: 17}

func run() uint64 {
	total := uint64(0)
	for i := 0; i < 1000; i++ {
		total += s.a + s.d
	}
	return total
}
`,
		},
		{
			"read_uint8_field",
			`package main

type T struct{ a, b, c, d uint8 }

var s = T{a: 7, b: 11, c: 13, d: 17}

func run() uint8 {
	total := uint8(0)
	for i := 0; i < 1000; i++ {
		total += s.a + s.d
	}
	return total
}
`,
		},
		{
			"read_float64_field",
			`package main

type T struct{ a, b, c, d float64 }

var s = T{a: 1.5, b: 2.5, c: 3.5, d: 4.5}

func run() float64 {
	total := 0.0
	for i := 0; i < 1000; i++ {
		total += s.a + s.d
	}
	return total
}
`,
		},
		{
			"read_float32_field",
			`package main

type T struct{ a, b, c, d float32 }

var s = T{a: 1.5, b: 2.5, c: 3.5, d: 4.5}

func run() float32 {
	total := float32(0)
	for i := 0; i < 1000; i++ {
		total += s.a + s.d
	}
	return total
}
`,
		},
		{
			"read_bool_field",
			`package main

type T struct{ a, b, c, d bool }

var s = T{a: true, b: false, c: true, d: true}

func run() int {
	count := 0
	for i := 0; i < 1000; i++ {
		if s.a {
			count++
		}
		if s.d {
			count++
		}
	}
	return count
}
`,
		},
		{
			"read_string_field_short",
			`package main

type T struct{ a, b string }

var s = T{a: "abc", b: "xyz"}

func run() int {
	total := 0
	for i := 0; i < 1000; i++ {
		total += len(s.a) + len(s.b)
	}
	return total
}
`,
		},
		{
			"read_string_field_long",
			`package main

type T struct{ a, b string }

var s = T{
	a: "the quick brown fox jumps over the lazy dog over and over and over again ok",
	b: "a different sentence with similarly long contents to keep the string header big",
}

func run() int {
	total := 0
	for i := 0; i < 1000; i++ {
		total += len(s.a) + len(s.b)
	}
	return total
}
`,
		},
		{
			"read_via_pointer_receiver",
			`package main

type T struct{ a, b, c, d int64 }

var s = &T{a: 7, b: 11, c: 13, d: 17}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += s.a + s.d
	}
	return total
}
`,
		},
		{
			"read_via_value_receiver",
			`package main

type T struct{ a, b, c, d int64 }

func run() int64 {
	s := T{a: 7, b: 11, c: 13, d: 17}
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += s.a + s.d
	}
	return total
}
`,
		},
		{
			"read_from_map_value",
			`package main

type T struct{ a, b, c, d int64 }

var m = map[int]T{0: {a: 7, b: 11, c: 13, d: 17}}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		v := m[0]
		total += v.a + v.d
	}
	return total
}
`,
		},
		{
			"read_from_slice_element",
			`package main

type T struct{ a, b, c, d int64 }

var xs = []T{{a: 7, b: 11, c: 13, d: 17}}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += xs[0].a + xs[0].d
	}
	return total
}
`,
		},
		{
			"read_after_type_assert",
			`package main

type T struct{ a, b, c, d int64 }

var x interface{} = T{a: 7, b: 11, c: 13, d: 17}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		v := x.(T)
		total += v.a + v.d
	}
	return total
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkStructFieldWrite(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"write_int64_field",
			`package main

type T struct{ a, b, c, d int64 }

var s = &T{}

func run() int64 {
	for i := 0; i < 1000; i++ {
		s.a = int64(i)
	}
	return s.a
}
`,
		},
		{
			"write_int32_field",
			`package main

type T struct{ a, b, c, d int32 }

var s = &T{}

func run() int32 {
	for i := 0; i < 1000; i++ {
		s.a = int32(i)
	}
	return s.a
}
`,
		},
		{
			"write_int8_field",
			`package main

type T struct{ a, b, c, d int8 }

var s = &T{}

func run() int8 {
	for i := 0; i < 1000; i++ {
		s.a = int8(i)
	}
	return s.a
}
`,
		},
		{
			"write_uint64_field",
			`package main

type T struct{ a, b, c, d uint64 }

var s = &T{}

func run() uint64 {
	for i := 0; i < 1000; i++ {
		s.a = uint64(i)
	}
	return s.a
}
`,
		},
		{
			"write_float64_field",
			`package main

type T struct{ a, b, c, d float64 }

var s = &T{}

func run() float64 {
	for i := 0; i < 1000; i++ {
		s.a = float64(i)
	}
	return s.a
}
`,
		},
		{
			"write_bool_field",
			`package main

type T struct{ a, b, c, d bool }

var s = &T{}

func run() bool {
	for i := 0; i < 1000; i++ {
		s.a = i&1 == 0
	}
	return s.a
}
`,
		},
		{
			"write_string_field_short",
			`package main

type T struct{ a, b string }

var s = &T{}

func run() string {
	for i := 0; i < 1000; i++ {
		s.a = "abc"
	}
	return s.a
}
`,
		},
		{
			"write_string_field_long",
			`package main

type T struct{ a, b string }

var s = &T{}

func run() string {
	for i := 0; i < 1000; i++ {
		s.a = "the quick brown fox jumps over the lazy dog over and over and over again ok"
	}
	return s.a
}
`,
		},
		{
			"write_via_pointer_receiver",
			`package main

type T struct{ a, b, c, d int64 }

var p = &T{}

func run() int64 {
	for i := 0; i < 1000; i++ {
		p.a = int64(i)
	}
	return p.a
}
`,
		},
		{
			"write_via_slice_element",
			`package main

type T struct{ a, b, c, d int64 }

var xs = make([]T, 4)

func run() int64 {
	for i := 0; i < 1000; i++ {
		xs[i&3].a = int64(i)
	}
	return xs[0].a
}
`,
		},
		{
			"write_pointer_field_to_nil",
			`package main

type T struct{ next *T }

var s = &T{next: &T{}}

func run() bool {
	for i := 0; i < 1000; i++ {
		s.next = nil
	}
	return s.next == nil
}
`,
		},
		{
			"write_slice_field",
			`package main

type T struct{ xs []int }

var s = &T{}

func run() int {
	src := []int{1, 2, 3, 4, 5}
	for i := 0; i < 1000; i++ {
		s.xs = src
	}
	return len(s.xs)
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkStructFieldCompound(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"compound_int_add",
			`package main

type T struct{ counter int64 }

var s = &T{}

func run() int64 {
	for i := 0; i < 1000; i++ {
		s.counter += 1
	}
	return s.counter
}
`,
		},
		{
			"compound_int_sub",
			`package main

type T struct{ counter int64 }

var s = &T{counter: 1000000}

func run() int64 {
	for i := 0; i < 1000; i++ {
		s.counter -= 1
	}
	return s.counter
}
`,
		},
		{
			"compound_int_mul",
			`package main

type T struct{ counter int64 }

var s = &T{counter: 1}

func run() int64 {
	for i := 0; i < 1000; i++ {
		s.counter *= 1
	}
	return s.counter
}
`,
		},
		{
			"compound_float_add",
			`package main

type T struct{ total float64 }

var s = &T{}

func run() float64 {
	for i := 0; i < 1000; i++ {
		s.total += 0.5
	}
	return s.total
}
`,
		},
		{
			"compound_string_concat",
			`package main

type T struct{ label string }

var s = &T{}

func run() int {
	s.label = ""
	for i := 0; i < 100; i++ {
		s.label += "x"
	}
	return len(s.label)
}
`,
		},
		{
			"compound_via_slice_element",
			`package main

type T struct{ counter int64 }

var xs = make([]T, 4)

func run() int64 {
	for i := 0; i < 1000; i++ {
		xs[i&3].counter += 1
	}
	return xs[0].counter
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkStructFieldNested(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"depth_2_value",
			`package main

type Inner struct{ x int64 }
type Outer struct{ inner Inner }

var s = Outer{inner: Inner{x: 42}}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += s.inner.x
	}
	return total
}
`,
		},
		{
			"depth_3_value",
			`package main

type Inner struct{ x int64 }
type Mid struct{ inner Inner }
type Top struct{ mid Mid }

var s = Top{mid: Mid{inner: Inner{x: 42}}}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += s.mid.inner.x
	}
	return total
}
`,
		},
		{
			"depth_2_pointer_inner",
			`package main

type Inner struct{ x int64 }
type Outer struct{ inner *Inner }

var s = Outer{inner: &Inner{x: 42}}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += s.inner.x
	}
	return total
}
`,
		},
		{
			"depth_3_pointer_chain",
			`package main

type Inner struct{ x int64 }
type Mid struct{ inner *Inner }
type Top struct{ mid *Mid }

var s = &Top{mid: &Mid{inner: &Inner{x: 42}}}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += s.mid.inner.x
	}
	return total
}
`,
		},
		{
			"embedded_value_field",
			`package main

type Base struct{ n int64 }
type Wrap struct{ Base }

var s = Wrap{Base: Base{n: 42}}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += s.n
	}
	return total
}
`,
		},
		{
			"embedded_pointer_field",
			`package main

type Base struct{ n int64 }
type Wrap struct{ *Base }

var s = Wrap{Base: &Base{n: 42}}

func run() int64 {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += s.n
	}
	return total
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkStructFieldIteration(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"slice_sum_int_field",
			`package main

type Cell struct{ a, b, c, d int64 }

var cells = func() []Cell {
	r := make([]Cell, 1000)
	for i := 0; i < 1000; i++ {
		r[i] = Cell{a: int64(i), d: int64(i + 1)}
	}
	return r
}()

func run() int64 {
	total := int64(0)
	for i := 0; i < len(cells); i++ {
		total += cells[i].a
	}
	return total
}
`,
		},
		{
			"slice_sum_two_fields",
			`package main

type Cell struct{ a, b, c, d int64 }

var cells = func() []Cell {
	r := make([]Cell, 1000)
	for i := 0; i < 1000; i++ {
		r[i] = Cell{a: int64(i), d: int64(i + 1)}
	}
	return r
}()

func run() int64 {
	total := int64(0)
	for i := 0; i < len(cells); i++ {
		total += cells[i].a + cells[i].d
	}
	return total
}
`,
		},
		{
			"slice_write_field_in_loop",
			`package main

type Cell struct{ a, b, c, d int64 }

var cells = make([]Cell, 1000)

func run() int64 {
	for i := 0; i < len(cells); i++ {
		cells[i].a = int64(i)
	}
	return cells[len(cells)-1].a
}
`,
		},
		{
			"slice_read_string_field",
			`package main

type Row struct{ name, kind string }

var rows = func() []Row {
	r := make([]Row, 100)
	for i := 0; i < 100; i++ {
		r[i] = Row{name: "item", kind: "k"}
	}
	return r
}()

func run() int {
	total := 0
	for i := 0; i < len(rows); i++ {
		total += len(rows[i].name) + len(rows[i].kind)
	}
	return total
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}
