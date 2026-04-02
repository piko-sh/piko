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

package fields

type Text string

func (t Text) String() string {
	return string(t)
}

type Pair[K, V any] struct {
	Key   K `json:"key"`
	Value V `json:"value"`
}

func (p Pair[K, V]) GetKey() K {
	return p.Key
}

func (p Pair[K, V]) GetValue() V {
	return p.Value
}

func (p Pair[K, V]) Swap() Pair[V, K] {
	return Pair[V, K]{Key: p.Value, Value: p.Key}
}

type Triple[A, B, C any] struct {
	First  A `json:"first"`
	Second B `json:"second"`
	Third  C `json:"third"`
}

func (t Triple[A, B, C]) GetFirst() A {
	return t.First
}

func (t Triple[A, B, C]) GetSecond() B {
	return t.Second
}

func (t Triple[A, B, C]) GetThird() C {
	return t.Third
}

type Result[T, E any] struct {
	value *T
	err   *E
}

func NewSuccess[T, E any](v T) Result[T, E] {
	return Result[T, E]{value: &v}
}

func NewError[T, E any](e E) Result[T, E] {
	return Result[T, E]{err: &e}
}

func (r Result[T, E]) IsSuccess() bool {
	return r.value != nil
}

func (r Result[T, E]) GetValue() T {
	if r.value != nil {
		return *r.value
	}
	var zero T
	return zero
}

func (r Result[T, E]) GetError() E {
	if r.err != nil {
		return *r.err
	}
	var zero E
	return zero
}
