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

func TestInterfaceInlineCacheCorrectness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "monomorphic_value_receiver_loop",
			code: `type Greeter interface{ Greet() int }
type En struct{ n int }
func (e En) Greet() int { return e.n }
var g Greeter = En{n: 7}
s := 0
for i := 0; i < 10; i++ {
	s += g.Greet()
}
s`,
			expect: int64(70),
		},
		{
			name: "monomorphic_pointer_receiver_loop",
			code: `type Counter interface{ Inc(); Value() int }
type Cnt struct{ n int }
func (c *Cnt) Inc() { c.n++ }
func (c *Cnt) Value() int { return c.n }
func run() int {
	inst := &Cnt{}
	var ci Counter = inst
	for i := 0; i < 10; i++ {
		ci.Inc()
	}
	return ci.Value()
}
run()`,
			expect: int64(10),
		},
		{
			name: "polymorphic_two_types_alternating",
			code: `type Shape interface{ Area() int }
type Square struct{ side int }
func (s Square) Area() int { return s.side * s.side }
type Rect struct{ w, h int }
func (r Rect) Area() int { return r.w * r.h }
shapes := []Shape{Square{side: 3}, Rect{w: 4, h: 5}}
s := 0
for i := 0; i < 10; i++ {
	s += shapes[i&1].Area()
}
s`,
			expect: int64(5*9 + 5*20),
		},
		{
			name: "polymorphic_three_types_round_robin",
			code: `type S interface{ V() int }
type A struct{ x int }
func (a A) V() int { return a.x }
type B struct{ x int }
func (b B) V() int { return b.x * 10 }
type C struct{ x int }
func (c C) V() int { return c.x * 100 }
items := []S{A{x: 1}, B{x: 2}, C{x: 3}}
s := 0
for i := 0; i < 9; i++ {
	s += items[i%3].V()
}
s`,
			expect: int64(3*1 + 3*20 + 3*300),
		},
		{
			name: "embedded_promoted_method",
			code: `type Speaker interface{ Speak() int }
type Inner struct{ value int }
func (i Inner) Speak() int { return i.value }
type Outer struct{ Inner }
var s Speaker = Outer{Inner: Inner{value: 42}}
s.Speak()`,
			expect: int64(42),
		},
		{
			name: "monomorphic_then_polymorphic_arrives",
			code: `type Greeter interface{ Greet() int }
type En struct{ n int }
func (e En) Greet() int { return e.n }
type De struct{ n int }
func (d De) Greet() int { return d.n * 2 }
var g Greeter = En{n: 5}
s := g.Greet()
s += g.Greet()
g = De{n: 5}
s += g.Greet()
s += g.Greet()
s`,
			expect: int64(5 + 5 + 10 + 10),
		},
		{
			name: "interface_method_returning_string",
			code: `type Namer interface{ Name() string }
type Person struct{ name string }
func (p Person) Name() string { return p.name }
var n Namer = Person{name: "alice"}
n.Name() + "-" + n.Name()`,
			expect: "alice-alice",
		},
		{
			name: "interface_method_with_args",
			code: `type Adder interface{ Plus(int) int }
type N struct{ base int }
func (n N) Plus(x int) int { return n.base + x }
var a Adder = N{base: 100}
a.Plus(1) + a.Plus(2) + a.Plus(3)`,
			expect: int64(303 + 3),
		},
		{
			name: "interface_pointer_in_struct",
			code: `type Resolver interface{ Resolve() int }
type Dat struct{ x int }
func (d *Dat) Resolve() int { return d.x }
type Wrap struct{ r Resolver }
w := Wrap{r: &Dat{x: 99}}
w.r.Resolve() + w.r.Resolve()`,
			expect: int64(198),
		},
		{
			name: "interface_after_assignment_swap",
			code: `type Greeter interface{ Greet() int }
type X struct{ v int }
func (x X) Greet() int { return x.v }
type Y struct{ v int }
func (y Y) Greet() int { return -y.v }
var g Greeter
g = X{v: 7}
a := g.Greet()
g = Y{v: 7}
b := g.Greet()
a + b`,
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

func TestInterfaceInlineCacheStateTransitions(t *testing.T) {
	t.Parallel()

	service := NewService()

	code := `type Greeter interface{ Greet() int }
type A struct{ n int }
func (a A) Greet() int { return a.n + 1 }
type B struct{ n int }
func (b B) Greet() int { return b.n + 100 }
var g Greeter = A{n: 0}
v1 := g.Greet()
v2 := g.Greet()
v3 := g.Greet()
g = B{n: 0}
v4 := g.Greet()
v5 := g.Greet()
g = A{n: 0}
v6 := g.Greet()
v1*1000000 + v2*100000 + v3*10000 + v4*1000 + v5*100 + v6`
	result, err := service.Eval(context.Background(), code)
	require.NoError(t, err)
	require.Equal(t, int64(1*1000000+1*100000+1*10000+100*1000+100*100+1), result)
}

func TestArenaStructEscapesAcrossSlabGrowth(t *testing.T) {
	t.Parallel()
	service := NewService()
	code := `type En struct{ n int }
func (e En) Greet() int { return e.n }
type Greeter interface{ Greet() int }
var g Greeter = En{n: 42}
junk := make([]byte, 0, 1)
for k := 0; k < 100000; k++ {
	junk = append(junk, byte(k))
}
_ = junk
total := 0
for i := 0; i < 100; i++ {
	total += g.Greet()
}
total`
	result, err := service.Eval(context.Background(), code)
	require.NoError(t, err)
	require.Equal(t, int64(4200), result,
		"interface-stored En must survive slab-growth pressure")
}
