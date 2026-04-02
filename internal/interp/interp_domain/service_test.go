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
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceCompileAndExecute(t *testing.T) {
	t.Parallel()
	service := NewService()
	compiledFunction, err := service.Compile(context.Background(), "1 + 2")
	require.NoError(t, err)
	require.NotNil(t, compiledFunction)

	result, err := service.Execute(context.Background(), compiledFunction)
	require.NoError(t, err)
	require.Equal(t, int64(3), result)
}

func TestServiceCompileError(t *testing.T) {
	t.Parallel()
	service := NewService()
	_, err := service.Compile(context.Background(), "func {{{")
	require.Error(t, err)
}

func TestServiceExecuteMultipleTimes(t *testing.T) {
	t.Parallel()
	service := NewService()
	compiledFunction, err := service.Compile(context.Background(), "2 * 3")
	require.NoError(t, err)

	for range 5 {
		result, err := service.Execute(context.Background(), compiledFunction)
		require.NoError(t, err)
		require.Equal(t, int64(6), result)
	}
}

func TestServiceCompileStatements(t *testing.T) {
	t.Parallel()
	service := NewService()
	compiledFunction, err := service.Compile(context.Background(), "x := 10; y := 20; x + y")
	require.NoError(t, err)
	result, err := service.Execute(context.Background(), compiledFunction)
	require.NoError(t, err)
	require.Equal(t, int64(30), result)
}

func TestServiceCompileMixed(t *testing.T) {
	t.Parallel()
	service := NewService()
	compiledFunction, err := service.Compile(context.Background(), `func double(x int) int { return x * 2 }
double(21)`)
	require.NoError(t, err)
	result, err := service.Execute(context.Background(), compiledFunction)
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestServiceReset(t *testing.T) {
	t.Parallel()
	service := NewService()
	_, err := service.Eval(context.Background(), "1 + 1")
	require.NoError(t, err)
	service.Reset()
	result, err := service.Eval(context.Background(), "2 + 2")
	require.NoError(t, err)
	require.Equal(t, int64(4), result)
}

func TestServiceClone(t *testing.T) {
	t.Parallel()
	service := NewService()
	clone := service.Clone()
	result, err := clone.Eval(context.Background(), "3 + 4")
	require.NoError(t, err)
	require.Equal(t, int64(7), result)
}

type mockSymbolProvider struct {
	exports SymbolExports
}

func (m *mockSymbolProvider) Exports() SymbolExports {
	return m.exports
}

func TestServiceUseSymbolProviders(t *testing.T) {
	t.Parallel()
	provider := &mockSymbolProvider{
		exports: SymbolExports{
			"testpkg": {
				"Add": reflect.ValueOf(func(a, b int) int { return a + b }),
			},
		},
	}
	service := NewService()
	service.UseSymbolProviders(provider)
	require.NotNil(t, service.symbols)
	require.True(t, service.symbols.HasPackage("testpkg"))
}

func TestServicePackageLevelVarInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "variable with literal init",
			source: `package main
var x = 42
func run() int { return x }
func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
		},
		{
			name: "variable with expression init",
			source: `package main
var x = 3 * 14
func run() int { return x }
func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
		},
		{
			name: "multiple variables with init",
			source: `package main
var (
	a = 10
	b = a + 20
)
func run() int { return b }
func main() {}
`,
			entrypoint: "run",
			expect:     int64(30),
		},
		{
			name: "string variable",
			source: `package main
var greeting = "hello"
func run() string { return greeting }
func main() {}
`,
			entrypoint: "run",
			expect:     "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := newTestService(t)
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestServiceMultiFileCompilation(t *testing.T) {
	t.Parallel()

	sources := map[string]string{
		"main.go": `package main
func main() {}
func run() int { return helper() }
`,
		"helper.go": `package main
func helper() int { return 42 }
`,
	}

	service := newTestService(t)
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestServiceCompileErrorWrapping(t *testing.T) {
	t.Parallel()

	t.Run("syntax error", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t)
		_, err := service.EvalFile(context.Background(), `package main
func run() int { return }
func main() {}
`, "run")
		require.Error(t, err)
	})

	t.Run("type error", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t)
		_, err := service.EvalFile(context.Background(), `package main
func run() int { return "hello" }
func main() {}
`, "run")
		require.Error(t, err)
	})

	t.Run("unknown entrypoint", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t)
		_, err := service.EvalFile(context.Background(), `package main
func main() {}
`, "nonexistent")
		require.Error(t, err)
	})
}

func TestServiceInitFunctions(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	result, err := service.EvalFile(context.Background(), `package main

var order []int

func init() {
	order = append(order, 1)
}

func init() {
	order = append(order, 2)
}

func run() int {
	return len(order)
}

func main() {}
`, "run")
	require.NoError(t, err)
	require.Equal(t, int64(2), result)
}

func TestServiceChannelDirectionTypes(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithFunctions(t, "testpkg", map[string]reflect.Value{
		"SendOnly": reflect.ValueOf(func(ch chan<- int, val int) {
			ch <- val
		}),
		"RecvOnly": reflect.ValueOf(func(ch <-chan int) int {
			return <-ch
		}),
	})

	source := `package main
import "testpkg"

func run() int {
	ch := make(chan int, 1)
	testpkg.SendOnly(ch, 42)
	return testpkg.RecvOnly(ch)
}
func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestServiceCloneAndReset(t *testing.T) {
	t.Parallel()

	t.Run("clone preserves symbols", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		_, err := service.Eval(context.Background(), "x := 42; x")
		require.NoError(t, err)

		clone := service.Clone()

		result, err := clone.Eval(context.Background(), "1 + 2")
		require.NoError(t, err)
		require.Equal(t, int64(3), result)
	})

	t.Run("reset clears state", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		_, err := service.Eval(context.Background(), "x := 42; x")
		require.NoError(t, err)

		service.Reset()

		result, err := service.Eval(context.Background(), "1 + 1")
		require.NoError(t, err)
		require.Equal(t, int64(2), result)
	})
}

func TestServiceExecuteInits(t *testing.T) {
	t.Parallel()

	service := newTestService(t)

	sources := map[string]string{
		"main.go": `package main

var initCount int

func init() { initCount++ }
func init() { initCount++ }
func init() { initCount++ }

func run() int { return initCount }
func main() {}
`,
	}

	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	err = service.ExecuteInits(context.Background(), cfs)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "run")
	require.NoError(t, err)
	require.Equal(t, int64(3), result)
}

func TestServiceVarInitWithFunctionCall(t *testing.T) {
	t.Parallel()

	service := newTestService(t)
	source := `package main

func compute() int { return 21 * 2 }

var x = compute()

func run() int { return x }
func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestServiceSelectRecvAssign(t *testing.T) {
	t.Parallel()

	service := newTestService(t)

	source := `package main

func run() int {
	ch := make(chan int, 1)
	ch <- 42
	v := 0
	select {
	case v = <-ch:
	default:
	}
	return v
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestServiceSelectRecvDefine(t *testing.T) {
	t.Parallel()

	service := newTestService(t)

	source := `package main

func run() int {
	ch := make(chan int, 1)
	ch <- 42
	result := 0
	select {
	case v := <-ch:
		result = v
	default:
	}
	return result
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestServiceSendStatement(t *testing.T) {
	t.Parallel()

	service := newTestService(t)

	source := `package main

func run() int {
	ch := make(chan int, 1)
	ch <- 42
	return <-ch
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestServiceDebugInfoSourceMapOptimisation(t *testing.T) {
	t.Parallel()

	service := newTestService(t, WithDebugInfo())
	sources := map[string]string{
		"main.go": `package main

func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func run() int {
	return fibonacci(10)
}

func main() {}
`,
	}

	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	fn, err := cfs.FindFunction("fibonacci")
	require.NoError(t, err)
	require.True(t, fn.HasDebugSourceMap())

	file, line, _ := fn.DebugSourcePosition(0)
	require.NotEmpty(t, file)
	require.Greater(t, line, 0)
}
