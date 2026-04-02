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

func TestEvalClosures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "simple read-only capture",
			code: `x := 10
add := func(n int) int {
	return x + n
}
add(5)`,
			expect: int64(15),
		},
		{
			name: "mutable capture counter",
			code: `count := 0
increment := func() int {
	count = count + 1
	return count
}
increment()
increment()
increment()`,
			expect: int64(3),
		},
		{
			name: "multiple captures",
			code: `a := 10
b := 20
sum := func() int {
	return a + b
}
sum()`,
			expect: int64(30),
		},
		{
			name: "capture string",
			code: `prefix := "hello "
greet := func(name string) string {
	return prefix + name
}
greet("world")`,
			expect: "hello world",
		},
		{
			name: "closure with params and capture",
			code: `base := 100
compute := func(x int, y int) int {
	return base + x*y
}
compute(3, 4)`,
			expect: int64(112),
		},
		{
			name: "closure called multiple times read-only",
			code: `factor := 5
mul := func(x int) int {
	return x * factor
}
mul(3) + mul(7)`,
			expect: int64(50),
		},
		{
			name: "immediately invoked function literal",
			code: `func(x int) int {
	return x * 2
}(21)`,
			expect: int64(42),
		},
		{
			name: "immediately invoked with capture",
			code: `b := 0
func(x int) {
	b = b + x
}(10)
b`,
			expect: int64(10),
		},
		{
			name: "immediately invoked in loop",
			code: `var b int
for i := 0; i < 5; i++ {
	func(x int) {
		b = b + x
	}(i)
}
b`,
			expect: int64(10),
		},
		{
			name: "immediately invoked with return value",
			code: `result := func(a int, b int) int {
	return a + b
}(3, 4)
result`,
			expect: int64(7),
		},
		{
			name: "immediately invoked nested",
			code: `y := func(n int) int {
	return func(m int) int {
		return m + 1
	}(n * 2)
}(5)
y`,
			expect: int64(11),
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

func TestEvalClosureTypedUpvalues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "float upvalue",
			code: `f := 1.5
inc := func() float64 {
	f += 1.0
	return f
}
inc()
inc()`,
			expect: 3.5,
		},
		{
			name: "bool upvalue",
			code: `b := false
toggle := func() bool {
	b = !b
	return b
}
toggle()`,
			expect: true,
		},
		{
			name: "uint upvalue",
			code: `var u uint = 10
inc := func() uint {
	u += 5
	return u
}
inc()
inc()`,
			expect: uint64(20),
		},
		{
			name: "complex upvalue",
			code: `c := 1+2i
add := func() complex128 {
	c += 3+4i
	return c
}
add()`,
			expect: complex128(4 + 6i),
		},
		{
			name: "multiple typed upvalues",
			code: `x := 10
y := 2.5
fn := func() float64 {
	x += 1
	y += 0.5
	return float64(x) + y
}
fn()
fn()`,
			expect: 15.5,
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

func TestClosureCaptureUintAndComplex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "capture_uint",
			source: `package main

func run() uint {
	var u uint = 42
	f := func() uint { return u }
	return f()
}

func main() {}
`,
			entrypoint: "run",
			expect:     uint64(42),
		},
		{
			name: "capture_complex",
			source: `package main

func run() complex128 {
	var c complex128 = 3 + 4i
	f := func() complex128 { return c }
	return f()
}

func main() {}
`,
			entrypoint: "run",
			expect:     complex128(3 + 4i),
		},
		{
			name: "mutate_captured_uint",
			source: `package main

func run() uint {
	var u uint = 10
	f := func() { u += 32 }
	f()
	return u
}

func main() {}
`,
			entrypoint: "run",
			expect:     uint64(42),
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

func TestClosureCaptureInLoopBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "range_shadow_captures_last",
			source: `package main

func run() int {
	var f func() int
	for i := range []int{10, 20, 30} {
		i := i
		f = func() int { return i }
	}
	return f()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(2),
		},
		{
			name: "for_loop_shadow_captures_last",
			source: `package main

func run() int {
	var f func() int
	for i := 0; i < 5; i++ {
		x := i
		f = func() int { return x }
	}
	return f()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(4),
		},
		{
			name: "range_shadow_all_closures",
			source: `package main

func run() int {
	fns := make([]func() int, 3)
	for i := range []int{10, 20, 30} {
		i := i
		fns[i] = func() int { return i }
	}
	return fns[0]() + fns[1]() + fns[2]()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(3),
		},
		{
			name: "two_closures_same_iteration_share",
			source: `package main

func run() int {
	var f, g func() int
	for i := 0; i < 3; i++ {
		x := i
		f = func() int { return x }
		g = func() int { return x + 10 }
	}
	return f() + g()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(14),
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
