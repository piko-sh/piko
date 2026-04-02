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

func TestEvalMethodCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "struct field access via method-like selector",
			code: `type Point struct {
	X int
	Y int
}
p := Point{X: 3, Y: 4}
p.X + p.Y`,
			expect: int64(7),
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

func TestEvalMethodValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "value receiver method call",
			code: `type Counter struct { V int }
func (c Counter) Get() int { return c.V }
c := Counter{V: 42}
c.Get()`,
			expect: int64(42),
		},
		{
			name: "pointer receiver method call",
			code: `type Counter struct { V int }
func (c *Counter) Inc() { c.V++ }
func (c *Counter) Get() int { return c.V }
c := Counter{V: 10}
c.Inc()
c.Inc()
c.Get()`,
			expect: int64(12),
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

func TestEvalMethodCalls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
		skip   string
	}{
		{
			name: "method on value receiver",
			code: `type Counter struct{ N int }
func (c Counter) Value() int { return c.N }
c := Counter{N: 42}
c.Value()`,
			expect: int64(42),
			skip:   "",
		},
		{
			name: "method on pointer receiver",
			code: `type Counter struct{ N int }
func (c *Counter) Inc() int { c.N++; return c.N }
c := &Counter{N: 10}
c.Inc()`,
			expect: int64(11),
			skip:   "",
		},
		{
			name: "multiple method calls with mutation",
			code: `type Acc struct{ V int }
func (a *Acc) Add(n int) int { a.V += n; return a.V }
a := &Acc{V: 0}
a.Add(5)
a.Add(3)
a.V`,
			expect: int64(8),
			skip:   "",
		},
		{
			name: "method returning string",
			code: `type Greeter struct{ Name string }
func (g Greeter) Hello() string { return "hello " + g.Name }
g := Greeter{Name: "world"}
g.Hello()`,
			expect: "hello world",
			skip:   "",
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
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestEvalInterfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
		skip   string
	}{
		{
			name: "basic interface satisfaction and method call",
			code: `type Stringer interface { String() string }
type MyType struct{ V string }
func (m MyType) String() string { return m.V }
var s Stringer = MyType{V: "test"}
s.String()`,
			expect: "test",
			skip:   "",
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
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestPromotedMethodCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "promoted_method_one_level",
			source: `package main

type Inner struct { V int }
func (i Inner) Double() int { return i.V * 2 }

type Outer struct { Inner }

func run() int {
	o := Outer{Inner: Inner{V: 21}}
	return o.Double()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
		},
		{
			name: "promoted_method_two_levels",
			source: `package main

type Base struct { N int }
func (b Base) Value() int { return b.N }

type Mid struct { Base }
type Top struct { Mid }

func run() int {
	t := Top{Mid: Mid{Base: Base{N: 42}}}
	return t.Value()
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
		},
		{
			name: "pointer_receiver_promotion",
			source: `package main

type Base struct { N int }
func (b *Base) Increment() { b.N++ }

type Wrapper struct { *Base }

func run() int {
	b := &Base{N: 41}
	w := Wrapper{Base: b}
	w.Increment()
	return b.N
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
