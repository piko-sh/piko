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
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalDeclarations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "const declaration int",
			code: `const x = 42
x`,
			expect: int64(42),
		},
		{
			name: "multiple const grouped",
			code: `const (
	a = 1
	b = 2
)
a + b`,
			expect: int64(3),
		},
		{
			name: "const string",
			code: `const s = "hello"
s`,
			expect: "hello",
		},
		{
			name: "const float typed",
			code: `const f float64 = 3.14
f`,
			expect: 3.14,
		},
		{
			name: "var with type no value",
			code: `var x int
x`,
			expect: int64(0),
		},
		{
			name: "var with type and value",
			code: `var x int = 42
x`,
			expect: int64(42),
		},
		{
			name: "multiple var grouped",
			code: `var (
	x int = 1
	y int = 2
)
x + y`,
			expect: int64(3),
		},
		{
			name: "var string zero value",
			code: `var s string
s`,
			expect: "",
		},
		{
			name: "var float64 zero value",
			code: `var f float64
f`,
			expect: 0.0,
		},
		{
			name: "var bool zero value",
			code: `var b bool
b`,
			expect: false,
		},
		{
			name: "const used in arithmetic",
			code: `const base = 100
offset := 42
base + offset`,
			expect: int64(142),
		},
		{
			name: "multiple const separate declarations",
			code: `const a = 10
const b = 20
a * b`,
			expect: int64(200),
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

	t.Run("untyped const float", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), "const f = 3.14\nf")
		require.NoError(t, err)
		require.Equal(t, 3.14, result)
	})
}

func TestEvalScoping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "closure shadows outer variable",
			code: `x := 1
f := func() int {
	x := 2
	return x
}
_ = x
f()`,
			expect: int64(2),
		},
		{
			name: "outer unaffected by if-block shadow",
			code: `x := 1
if true {
	x := 2
	_ = x
}
x`,
			expect: int64(1),
		},
		{
			name: "mutation in if block visible outside",
			code: `x := 10
if true {
	x = x + 5
}
x`,
			expect: int64(15),
		},
		{
			name: "for loop variable scoped to loop",
			code: `sum := 0
for i := 0; i < 5; i++ {
	sum = sum + i
}
sum`,
			expect: int64(10),
		},
		{
			name: "nested closure captures outer",
			code: `a := 10
b := 20
f := func() int {
	return a + b
}
f()`,
			expect: int64(30),
		},
		{
			name: "if-else both branches assign outer",
			code: `x := 0
cond := true
if cond {
	x = 42
} else {
	x = 99
}
x`,
			expect: int64(42),
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

	t.Run("closure mutation visible in outer scope", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		code := "x := 0\ninc := func() {\n\tx = x + 1\n}\ninc()\ninc()\ninc()\nx"
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(3), result)
	})
}

func TestEvalImportsWithSymbols(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"fmt": {
			"Sprintf": reflect.ValueOf(fmt.Sprintf),
		},
		"strings": {
			"ToUpper":   reflect.ValueOf(strings.ToUpper),
			"Contains":  reflect.ValueOf(strings.Contains),
			"HasPrefix": reflect.ValueOf(strings.HasPrefix),
		},
	})

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "fmt Sprintf with int",
			code: `import "fmt"
result := fmt.Sprintf("hello %d", 42)
result`,
			expect: "hello 42",
		},
		{
			name: "strings Contains true",
			code: `import "strings"
result := strings.Contains("hello world", "ell")
result`,
			expect: true,
		},
		{
			name: "strings HasPrefix",
			code: `import "strings"
result := strings.HasPrefix("hello", "hel")
result`,
			expect: true,
		},
		{
			name: "grouped import both packages",
			code: `import (
	"fmt"
	"strings"
)
name := strings.ToUpper("piko")
result := fmt.Sprintf("hello %s", name)
result`,
			expect: "hello PIKO",
		},
		{
			name: "import result used in expression",
			code: `import "fmt"
a := 10
b := 20
result := fmt.Sprintf("%d", a + b)
result`,
			expect: "30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			service.UseSymbols(symbols)
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalBasicLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "hex literal",
			code:   `x := 0x1F; x`,
			expect: int64(31),
		},
		{
			name:   "binary literal",
			code:   `x := 0b1010; x`,
			expect: int64(10),
		},
		{
			name:   "octal literal",
			code:   `x := 0o17; x`,
			expect: int64(15),
		},
		{
			name:   "underscored int literal",
			code:   `x := 1_000_000; x`,
			expect: int64(1000000),
		},
		{
			name:   "scientific notation float",
			code:   `x := 3.14e2; x`,
			expect: 314.0,
		},
		{
			name:   "rune literal",
			code:   `x := 'A'; x`,
			expect: int64(65),
		},
		{
			name:   "negative hex via variable",
			code:   `x := 0xFF; y := -x; y`,
			expect: int64(-255),
		},
		{
			name:   "large binary literal",
			code:   `x := 0b11111111; x`,
			expect: int64(255),
		},
		{
			name:   "underscored float literal",
			code:   `x := 1_000.5; x`,
			expect: 1000.5,
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

func TestClosurePassedToNativeWithInnerCall(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"sort": {
			"Slice": reflect.ValueOf(sort.Slice),
		},
	})

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "sort.Slice with closure calling inner function",
			source: `package main

import "sort"

func compare(a, b int) bool {
	return a < b
}

func run() int {
	s := []int{3, 1, 4, 1, 5}
	sort.Slice(s, func(i, j int) bool {
		return compare(s[i], s[j])
	})
	return s[0]*10000 + s[1]*1000 + s[2]*100 + s[3]*10 + s[4]
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(11345),
		},
		{
			name: "sort.Slice with direct closure no inner call",
			source: `package main

import "sort"

func run() int {
	s := []int{5, 2, 8}
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
	return s[0]*100 + s[1]*10 + s[2]
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(258),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			service.UseSymbols(symbols)
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}
