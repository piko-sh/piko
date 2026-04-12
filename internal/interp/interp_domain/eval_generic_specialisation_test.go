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

func TestEvalGenericBodySpecialisation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "identity int returns typed",
			code: `func id[T any](x T) T { return x }
id(42)`,
			expect: int64(42),
		},
		{
			name: "identity bool returns typed",
			code: `func id[T any](x T) T { return x }
id(true)`,
			expect: true,
		},
		{
			name: "identity float returns typed",
			code: `func id[T any](x T) T { return x }
id(3.14)`,
			expect: 3.14,
		},
		{
			name: "identity string returns typed",
			code: `func id[T any](x T) T { return x }
id("hello")`,
			expect: "hello",
		},
		{
			name: "identity uint returns typed",
			code: `func id[T any](x T) T { return x }
var u uint = 42
id(u)`,
			expect: uint64(42),
		},
		{
			name: "polymorphic three specialisations",
			code: `func id[T any](x T) T { return x }
a := id(42)
b := id("hello")
c := id(true)
_ = a
_ = b
_ = c
a + 1`,
			expect: int64(43),
		},
		{
			name: "constraint Ordered max int",
			code: `type Ordered interface {
	~int | ~float64 | ~string
}
func max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
max(7, 3)`,
			expect: int64(7),
		},
		{
			name: "constraint Ordered max float",
			code: `type Ordered interface {
	~int | ~float64 | ~string
}
func max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
max(2.5, 9.5)`,
			expect: 9.5,
		},
		{
			name: "constraint Ordered max string",
			code: `type Ordered interface {
	~int | ~float64 | ~string
}
func max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
max("apple", "zebra")`,
			expect: "zebra",
		},
		{
			name: "explicit type argument",
			code: `func id[T any](x T) T { return x }
id[int](100)`,
			expect: int64(100),
		},
		{
			name: "type inference from arg",
			code: `func id[T any](x T) T { return x }
id(7) + id(8)`,
			expect: int64(15),
		},
		{
			name: "generic calling generic",
			code: `func id[T any](x T) T { return x }
func twice[T any](x T) T { return id(id(x)) }
twice(11)`,
			expect: int64(11),
		},
		{
			name: "recursive generic factorial",
			code: `func fact[T ~int](n T) T {
	if n <= 1 {
		return 1
	}
	return n * fact(n-1)
}
fact(5)`,
			expect: int64(120),
		},
		{
			name: "generic with named return",
			code: `func zero[T any]() (out T) {
	return
}
zero[int]()`,
			expect: int64(0),
		},
		{
			name: "generic slice access",
			code: `func first[T any](xs []T) T {
	return xs[0]
}
first([]int{1, 2, 3})`,
			expect: int64(1),
		},
		{
			name: "generic closure captures typed",
			code: `func makeAdder[T ~int](n T) func(T) T {
	return func(x T) T {
		return x + n
	}
}
add3 := makeAdder(3)
add3(10)`,
			expect: int64(13),
		},
		{
			name: "specialisation cap fallback",
			code: `func id[T any](x T) T { return x }
a := id(int8(1))
b := id(int16(2))
c := id(int32(3))
d := id(int64(4))
e := id(uint8(5))
f := id(uint16(6))
g := id(uint32(7))
h := id(uint64(8))
i := id(float32(9.0))
j := id(float64(10.0))
_ = a; _ = b; _ = c; _ = d; _ = e
_ = f; _ = g; _ = h; _ = i; _ = j
"ok"`,
			expect: "ok",
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
