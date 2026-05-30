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

func BenchmarkNilJumpFusion(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"err_eq_nil_taken",
			`package main

func run() int {
	var err error
	hits := 0
	for i := 0; i < 1000; i++ {
		if err == nil {
			hits++
		}
	}
	return hits
}
`,
		},
		{
			"err_ne_nil_not_taken",
			`package main

func run() int {
	var err error
	hits := 0
	for i := 0; i < 1000; i++ {
		if err != nil {
			hits++
		}
	}
	return hits
}
`,
		},
		{
			"interface_eq_nil_alternating",
			`package main

type Greeter interface{ Greet() int }

type En struct{ n int }

func (e En) Greet() int { return e.n }

func run() int {
	hits := 0
	var g Greeter
	for i := 0; i < 1000; i++ {
		if i&1 == 0 {
			g = nil
		} else {
			g = En{n: i}
		}
		if g == nil {
			hits++
		}
	}
	return hits
}
`,
		},
		{
			"pointer_ne_nil_taken",
			`package main

type Node struct{ next *Node }

func run() int {
	var head *Node
	hits := 0
	for i := 0; i < 1000; i++ {
		if head != nil {
			hits++
		}
	}
	return hits
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkUnaryOps(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"neg_int_loop",
			`package main

func run() int {
	x := 0
	for i := 0; i < 1000; i++ {
		x = -i
	}
	return x
}
`,
		},
		{
			"bit_not_int_loop",
			`package main

func run() int {
	x := 0
	for i := 0; i < 1000; i++ {
		x = ^i
	}
	return x
}
`,
		},
		{
			"neg_float_loop",
			`package main

func run() float64 {
	var x float64
	for i := 0; i < 1000; i++ {
		x = -float64(i)
	}
	return x
}
`,
		},
		{
			"neg_int_accumulate",
			`package main

func run() int {
	s := 0
	for i := 0; i < 1000; i++ {
		s += -i
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkMathIntrinsics(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"sqrt_loop",
			`package main

import "math"

func run() float64 {
	var s float64
	for i := 1; i < 1000; i++ {
		s += math.Sqrt(float64(i))
	}
	return s
}
`,
		},
		{
			"abs_loop",
			`package main

import "math"

func run() float64 {
	var s float64
	for i := -500; i < 500; i++ {
		s += math.Abs(float64(i))
	}
	return s
}
`,
		},
		{
			"floor_ceil_alternating",
			`package main

import "math"

func run() float64 {
	var s float64
	for i := 0; i < 1000; i++ {
		f := float64(i) + 0.5
		s += math.Floor(f) + math.Ceil(f)
	}
	return s
}
`,
		},
		{
			"trunc_round_alternating",
			`package main

import "math"

func run() float64 {
	var s float64
	for i := 0; i < 1000; i++ {
		f := float64(i) + 0.3
		s += math.Trunc(f) + math.Round(f)
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarksWithStdlib(b, benchmarks)
}

func BenchmarkStringLenLoop(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"len_short_string",
			`package main

func run() int {
	s := "hello world"
	total := 0
	for i := 0; i < 1000; i++ {
		total += len(s)
	}
	return total
}
`,
		},
		{
			"len_long_string",
			`package main

func run() int {
	s := "the quick brown fox jumps over the lazy dog the quick brown fox jumps over the lazy dog"
	total := 0
	for i := 0; i < 1000; i++ {
		total += len(s)
	}
	return total
}
`,
		},
		{
			"len_string_in_range_cond",
			`package main

func run() int {
	s := "abcdefghij"
	total := 0
	for j := 0; j < 100; j++ {
		for i := 0; i < len(s); i++ {
			total += int(s[i])
		}
	}
	return total
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkTypedSliceRouting(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"float_slice_sum",
			`package main

var s = func() []float64 {
	r := make([]float64, 1000)
	for i := 0; i < 1000; i++ {
		r[i] = float64(i) * 0.5
	}
	return r
}()

func run() float64 {
	var total float64
	for _, v := range s {
		total += v
	}
	return total
}
`,
		},
		{
			"string_slice_index",
			`package main

var s = func() []string {
	r := make([]string, 100)
	for i := 0; i < 100; i++ {
		r[i] = "x"
	}
	return r
}()

func run() int {
	total := 0
	for i := 0; i < len(s); i++ {
		total += len(s[i])
	}
	return total
}
`,
		},
		{
			"bool_slice_count",
			`package main

var flags = func() []bool {
	r := make([]bool, 1000)
	for i := 0; i < 1000; i++ {
		r[i] = i%3 == 0
	}
	return r
}()

func run() int {
	n := 0
	for _, b := range flags {
		if b {
			n++
		}
	}
	return n
}
`,
		},
		{
			"uint_slice_bitmask_fold",
			`package main

var masks = func() []uint64 {
	r := make([]uint64, 1000)
	for i := 0; i < 1000; i++ {
		r[i] = uint64(i)
	}
	return r
}()

func run() uint64 {
	var combined uint64
	for _, m := range masks {
		combined |= m
	}
	return combined
}
`,
		},
		{
			"float_slice_dotproduct",
			`package main

var a = func() []float64 {
	r := make([]float64, 1000)
	for i := 0; i < 1000; i++ {
		r[i] = float64(i) * 0.5
	}
	return r
}()

var b = func() []float64 {
	r := make([]float64, 1000)
	for i := 0; i < 1000; i++ {
		r[i] = float64(i) * 0.25
	}
	return r
}()

func run() float64 {
	var sum float64
	for i := 0; i < 1000; i++ {
		sum += a[i] * b[i]
	}
	return sum
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkMoveString(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"move_string_in_loop",
			`package main

func run() string {
	src := "hello"
	dst := ""
	for i := 0; i < 1000; i++ {
		dst = src
	}
	return dst
}
`,
		},
		{
			"swap_strings_in_loop",
			`package main

func run() string {
	a := "alpha"
	b := "beta"
	for i := 0; i < 500; i++ {
		tmp := a
		a = b
		b = tmp
	}
	return a + b
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkDispatchOverhead(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"inc_int_chain",
			`package main

func run() int {
	x := 0
	x++; x++; x++; x++; x++; x++; x++; x++; x++; x++
	x++; x++; x++; x++; x++; x++; x++; x++; x++; x++
	x++; x++; x++; x++; x++; x++; x++; x++; x++; x++
	x++; x++; x++; x++; x++; x++; x++; x++; x++; x++
	x++; x++; x++; x++; x++; x++; x++; x++; x++; x++
	return x
}
`,
		},
		{
			"add_const_chain",
			`package main

func run() int {
	x := 0
	x += 1; x += 1; x += 1; x += 1; x += 1
	x += 1; x += 1; x += 1; x += 1; x += 1
	x += 1; x += 1; x += 1; x += 1; x += 1
	x += 1; x += 1; x += 1; x += 1; x += 1
	return x
}
`,
		},
		{
			"empty_call_chain",
			`package main

func noop() {}

func run() {
	noop(); noop(); noop(); noop(); noop()
	noop(); noop(); noop(); noop(); noop()
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkErrNilIdiom(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"never_returns_err",
			`package main

func process(x int) (int, error) {
	return x * 2, nil
}

func run() int {
	total := 0
	for i := 0; i < 1000; i++ {
		v, err := process(i)
		if err != nil {
			return -1
		}
		total += v
	}
	return total
}
`,
		},
		{
			"sometimes_returns_err",
			`package main

import "errors"

var errBad = errors.New("bad")

func process(x int) (int, error) {
	if x%7 == 0 {
		return 0, errBad
	}
	return x * 2, nil
}

func run() int {
	total := 0
	errors := 0
	for i := 0; i < 1000; i++ {
		v, err := process(i)
		if err != nil {
			errors++
			continue
		}
		total += v
	}
	return total + errors
}
`,
		},
	}
	runHotspotBenchmarksWithStdlib(b, benchmarks)
}

func BenchmarkSliceSort(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"insertion_sort_int_50",
			`package main

func run() int {
	xs := make([]int, 50)
	for i := 0; i < 50; i++ {
		xs[i] = (50 - i) * 7 % 50
	}
	for i := 1; i < len(xs); i++ {
		v := xs[i]
		j := i - 1
		for j >= 0 && xs[j] > v {
			xs[j+1] = xs[j]
			j--
		}
		xs[j+1] = v
	}
	return xs[0] + xs[len(xs)-1]
}
`,
		},
		{
			"bubble_sort_string_20",
			`package main

func run() string {
	xs := []string{"j", "d", "g", "a", "e", "h", "c", "i", "f", "b", "t", "q", "k", "l", "m", "n", "o", "p", "r", "s"}
	for i := 0; i < len(xs); i++ {
		for j := 0; j < len(xs)-1-i; j++ {
			if xs[j] > xs[j+1] {
				tmp := xs[j]
				xs[j] = xs[j+1]
				xs[j+1] = tmp
			}
		}
	}
	return xs[0] + xs[len(xs)-1]
}
`,
		},
	}
	runHotspotBenchmarks(b, benchmarks)
}

func BenchmarkParseFormat(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"itoa_round_trip_loop",
			`package main

import "strconv"

func run() int {
	total := 0
	for i := 0; i < 100; i++ {
		s := strconv.Itoa(i)
		v, err := strconv.Atoi(s)
		if err != nil {
			return -1
		}
		total += v
	}
	return total
}
`,
		},
		{
			"format_int_base16_loop",
			`package main

import "strconv"

func run() string {
	s := ""
	for i := 0; i < 100; i++ {
		s = strconv.FormatInt(int64(i*1000), 16)
	}
	return s
}
`,
		},
	}
	runHotspotBenchmarksWithStdlib(b, benchmarks)
}
