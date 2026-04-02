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

func TestEvalRangeSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
		delta  float64
	}{
		{
			name: "sum elements",
			code: `s := []int{1, 2, 3}
sum := 0
for _, v := range s { sum += v }
sum`,
			expect: int64(6),
		},
		{
			name: "count elements",
			code: `s := []int{10, 20, 30}
count := 0
for range s { count++ }
count`,
			expect: int64(3),
		},
		{
			name: "index only",
			code: `s := []string{"a", "b", "c"}
last := 0
for i := range s { last = i }
last`,
			expect: int64(2),
		},
		{
			name: "range over float slice",
			code: `s := []float64{1.1, 2.2, 3.3}
sum := 0.0
for _, v := range s { sum += v }
sum`,
			expect: 6.6,
			delta:  1e-10,
		},
		{
			name: "range string slice captures value",
			code: `s := []string{"hello", "world"}
result := ""
for _, v := range s { result += v }
result`,
			expect: "helloworld",
		},
		{
			name: "range with break",
			code: `s := []int{1, 2, 3, 4, 5}
sum := 0
for _, v := range s {
	if v > 3 { break }
	sum += v
}
sum`,
			expect: int64(6),
		},
		{
			name: "range with continue",
			code: `s := []int{1, 2, 3, 4, 5}
sum := 0
for _, v := range s {
	if v == 3 { continue }
	sum += v
}
sum`,
			expect: int64(12),
		},
		{
			name: "range empty slice",
			code: `s := []int{}
sum := 0
for _, v := range s { sum += v }
sum`,
			expect: int64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			if tt.delta > 0 {
				require.InDelta(t, tt.expect, result, tt.delta, "code: %s", tt.code)
			} else {
				require.Equal(t, tt.expect, result, "code: %s", tt.code)
			}
		})
	}
}

func TestEvalRangeMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "count entries",
			code: `m := map[string]int{"a": 1, "b": 2}
count := 0
for range m { count++ }
count`,
			expect: int64(2),
		},
		{
			name: "sum values",
			code: `m := map[string]int{"x": 10, "y": 20}
sum := 0
for _, v := range m { sum += v }
sum`,
			expect: int64(30),
		},
		{
			name: "collect keys length",
			code: `m := map[string]int{"alpha": 1, "beta": 2}
total := 0
for k := range m { total += len(k) }
total`,
			expect: int64(9),
		},
		{
			name: "empty map range",
			code: `m := map[string]int{}
count := 0
for range m { count++ }
count`,
			expect: int64(0),
		},
		{
			name: "map key and value both captured",
			code: `m := map[string]int{"a": 1, "b": 2, "c": 3}
sum := 0
for _, v := range m { sum += v }
sum`,
			expect: int64(6),
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

func TestEvalRangeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
		skip   string
	}{
		{
			name: "count runes",
			code: `s := "hello"
count := 0
for range s { count++ }
count`,
			expect: int64(5),
		},
		{
			name: "sum rune values",
			code: `s := "AB"
sum := 0
for _, r := range s { sum += int(r) }
sum`,
			expect: int64(131),
		},
		{
			name: "empty string range",
			code: `s := ""
count := 0
for range s { count++ }
count`,
			expect: int64(0),
		},
		{
			name: "range index tracks rune position",
			code: `s := "abc"
last := 0
for i := range s { last = i }
last`,
			expect: int64(2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.skip != "" {
				t.Skip(tt.skip)
			}
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalRangeInteger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "range int sum",
			code:   `sum := 0; for i := range 10 { sum += i }; sum`,
			expect: int64(45),
		},
		{
			name:   "range int no key",
			code:   `c := 0; for range 5 { c++ }; c`,
			expect: int64(5),
		},
		{
			name:   "range int zero",
			code:   `c := 0; for range 0 { c++ }; c`,
			expect: int64(0),
		},
		{
			name:   "range int one",
			code:   `c := 0; for range 1 { c++ }; c`,
			expect: int64(1),
		},
		{
			name:   "range int break",
			code:   `s := 0; for i := range 20 { if i >= 5 { break }; s += i }; s`,
			expect: int64(10),
		},
		{
			name:   "range int continue",
			code:   `s := 0; for i := range 10 { if i%2 == 0 { continue }; s += i }; s`,
			expect: int64(25),
		},
		{
			name:   "range int variable",
			code:   `n := 7; s := 0; for i := range n { s += i }; s`,
			expect: int64(21),
		},
		{
			name:   "range int blank key",
			code:   `c := 0; for _ = range 3 { c++ }; c`,
			expect: int64(3),
		},
		{
			name: "range int closure capture",
			code: `var f func() int
for i := range 3 {
	f = func() int { return i }
}
f()`,
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

func TestEvalRangeTypedSliceValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "range_string_slice_values",
			source: `package main

func run() string {
	result := ""
	for _, v := range []string{"a", "b", "c"} {
		result += v
	}
	return result
}

func main() {}
`,
			entrypoint: "run",
			expect:     "abc",
		},
		{
			name: "range_float_slice_values",
			source: `package main

func run() float64 {
	sum := 0.0
	for _, v := range []float64{1.0, 2.0, 3.0} {
		sum += v
	}
	return sum
}

func main() {}
`,
			entrypoint: "run",
			expect:     float64(6),
		},
		{
			name: "range_bool_slice_values",
			source: `package main

func run() int {
	count := 0
	for _, v := range []bool{true, false, true, true} {
		if v {
			count++
		}
	}
	return count
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(3),
		},
		{
			name: "range_uint_slice_values",
			source: `package main

func run() uint {
	var sum uint
	for _, v := range []uint{10, 20, 30} {
		sum += v
	}
	return sum
}

func main() {}
`,
			entrypoint: "run",
			expect:     uint64(60),
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

func TestEvalRangeChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "sum buffered channel values",
			code: `ch := make(chan int, 3)
ch <- 1
ch <- 2
ch <- 3
close(ch)
sum := 0
for v := range ch { sum += v }
sum`,
			expect: int64(6),
		},
		{
			name: "empty closed channel",
			code: `ch := make(chan int)
close(ch)
count := 0
for _ = range ch { count++ }
count`,
			expect: int64(0),
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
