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

func TestEvalFunctionDeclaration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "simple function call",
			code: `func add(a int, b int) int {
	return a + b
}
add(3, 4)`,
			expect: int64(7),
		},
		{
			name: "recursive function",
			code: `func fib(n int) int {
	if n <= 1 {
		return n
	}
	return fib(n-1) + fib(n-2)
}
fib(10)`,
			expect: int64(55),
		},
		{
			name: "function as expression",
			code: `func square(x int) int {
	return x * x
}
square(5) + square(3)`,
			expect: int64(34),
		},
		{
			name: "nested function calls",
			code: `func double(x int) int {
	return x * 2
}
func triple(x int) int {
	return x * 3
}
double(triple(5))`,
			expect: int64(30),
		},
		{
			name: "function with string parameter",
			code: `func greet(name string) string {
	return "hello " + name
}
greet("world")`,
			expect: "hello world",
		},
		{
			name: "function returning bool-like",
			code: `func isPositive(x int) int {
	if x > 0 {
		return 1
	}
	return 0
}
isPositive(5) + isPositive(-3)`,
			expect: int64(1),
		},
		{
			name: "function with multiple params same type",
			code: `func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
max(10, 20)`,
			expect: int64(20),
		},
		{
			name: "void function and side effect via pointer-like",
			code: `func compute(a int, b int) int {
	result := a * b + a + b
	return result
}
compute(3, 7)`,
			expect: int64(31),
		},
		{
			name: "multiple return values",
			code: `func divmod(a int, b int) (int, int) {
	return a / b, a % b
}
q, r := divmod(17, 5)
q*10 + r`,
			expect: int64(32),
		},
		{
			name: "multiple return values mixed use",
			code: `func minmax(a int, b int) (int, int) {
	if a < b {
		return a, b
	}
	return b, a
}
lo, hi := minmax(42, 7)
hi - lo`,
			expect: int64(35),
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
