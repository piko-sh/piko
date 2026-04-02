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

func TestEvalGenerics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "generic identity int",
			code: `func identity[T any](v T) T {
	return v
}
identity(42)`,
			expect: int64(42),
		},
		{
			name: "generic identity string",
			code: `func identity[T any](v T) T {
	return v
}
identity("hello")`,
			expect: "hello",
		},
		{
			name: "generic max",
			code: `type Ordered interface {
	~int | ~float64 | ~string
}
func max2[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
max2(3, 7)`,
			expect: int64(7),
		},
		{
			name: "multiple type params",
			code: `func swap[A, B any](a A, b B) (B, A) {
	return b, a
}
b, _ := swap(42, "hello")
b`,
			expect: "hello",
		},
		{
			name: "explicit type arguments",
			code: `func identity[T any](v T) T {
	return v
}
identity[int](42)`,
			expect: int64(42),
		},
		{
			name: "comparable constraint",
			code: `func equal[T comparable](a, b T) bool {
	return a == b
}
equal(42, 42)`,
			expect: true,
		},
		{
			name: "generic recursive",
			code: `func sum[T ~int](s []T) T {
	if len(s) == 0 {
		return 0
	}
	return s[0] + sum(s[1:])
}
sum([]int{1, 2, 3, 4})`,
			expect: int64(10),
		},
		{
			name: "generic closure",
			code: `func makeAdder[T ~int](n T) func(T) T {
	return func(x T) T {
		return x + n
	}
}
add3 := makeAdder(3)
add3(5)`,
			expect: int64(8),
		},
		{
			name: "generic slice param",
			code: `func first[T any](s []T) T {
	return s[0]
}
first([]int{1, 2, 3})`,
			expect: int64(1),
		},
		{
			name: "generic map param",
			code: `func getOrDefault[K comparable, V any](m map[K]V, k K, definition V) V {
	v, ok := m[k]
	if ok {
		return v
	}
	return definition
}
getOrDefault(map[string]int{"a": 1}, "b", 99)`,
			expect: int64(99),
		},
		{
			name: "type assertion on generic",
			code: `func box[T any](v T) any {
	return v
}
box(42).(int)`,
			expect: int(42),
		},
		{
			name: "generic channel",
			code: `func produce[T any](v T) chan T {
	ch := make(chan T, 1)
	ch <- v
	return ch
}
<-produce(42)`,
			expect: int64(42),
		},
		{
			name: "zero value generic",
			code: `func zero[T any]() T {
	var x T
	return x
}
zero[int]()`,
			expect: int64(0),
		},
		{
			name: "nested generic calls",
			code: `func double[T ~int](v T) T {
	return v * 2
}
func quadruple[T ~int](v T) T {
	return double(double(v))
}
quadruple(5)`,
			expect: int64(20),
		},
		{
			name: "multiple returns generic",
			code: `func tryGet[T any](v T) (T, bool) {
	return v, true
}
_, ok := tryGet(42)
ok`,
			expect: true,
		},
		{
			name: "generic struct",
			code: `type Box[T any] struct {
	Value T
}
b := Box[int]{Value: 99}
b.Value`,
			expect: int64(99),
		},
		{
			name: "generic struct method",
			code: `type Box[T any] struct {
	Value T
}
func (b Box[T]) Get() T {
	return b.Value
}
b := Box[int]{Value: 42}
b.Get()`,
			expect: int64(42),
		},
		{
			name: "pointer receiver generic method",
			code: `type Box[T any] struct {
	Value T
}
func (b *Box[T]) Set(v T) {
	b.Value = v
}
func (b Box[T]) Get() T {
	return b.Value
}
b := &Box[int]{Value: 0}
b.Set(99)
b.Get()`,
			expect: int64(99),
		},
		{
			name: "generic interface",
			code: `type Doubler interface {
	Double(int) int
}
type MyDoubler struct{}
func (d MyDoubler) Double(v int) int {
	return v * 2
}
func applyDouble[T Doubler](d T, v int) int {
	return d.Double(v)
}
applyDouble(MyDoubler{}, 21)`,
			expect: int64(42),
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

func TestEvalGenericsEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "generic identity bool",
			code: `func identity[T any](v T) T {
	return v
}
identity(true)`,
			expect: true,
		},
		{
			name: "generic identity float",
			code: `func identity[T any](v T) T {
	return v
}
identity(3.14)`,
			expect: 3.14,
		},
		{
			name: "generic identity multiple calls",
			code: `func identity[T any](v T) T {
	return v
}
identity(42)
identity("hello")`,
			expect: "hello",
		},
		{
			name: "generic with named return",
			code: `func f[T any]() (result T) {
	return
}
f[int]()`,
			expect: int64(0),
		},
		{
			name: "generic constraint method",
			code: `type HasLen interface {
	Len() int
}
type MySlice struct {
	data []int
}
func (s MySlice) Len() int {
	return len(s.data)
}
func getLen[T HasLen](v T) int {
	return v.Len()
}
getLen(MySlice{data: []int{1, 2, 3, 4, 5}})`,
			expect: int64(5),
		},
		{
			name: "type alias simple",
			code: `type MyInt = int
var x MyInt = 42
x`,
			expect: int64(42),
		},
		{
			name: "type alias chained",
			code: `type A = int
type B = A
var x B = 99
x`,
			expect: int64(99),
		},
		{
			name: "named type in func body",
			code: `type MyInt int
var x MyInt = 42
int(x)`,
			expect: int64(42),
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
