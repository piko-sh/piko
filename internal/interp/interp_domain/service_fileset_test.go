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
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalFile_Basic(t *testing.T) {
	t.Parallel()
	source := `package main

func entrypoint() int {
	return 42
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "42", fmt.Sprint(result))
}

func TestEvalFile_WithHelpers(t *testing.T) {
	t.Parallel()
	source := `package main

func double(x int) int {
	return x * 2
}

func entrypoint() int {
	return double(21)
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "42", fmt.Sprint(result))
}

func TestEvalFile_StringReturn(t *testing.T) {
	t.Parallel()
	source := `package main

func entrypoint() string {
	return "hello world"
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "hello world", fmt.Sprint(result))
}

func TestEvalFile_EntrypointNotFound(t *testing.T) {
	t.Parallel()
	source := `package main

func other() int {
	return 1
}

func main() {}
`
	service := NewService()
	_, err := service.EvalFile(context.Background(), source, "nonexistent")
	require.ErrorIs(t, err, errEntrypointNotFound)
}

func TestEvalFile_ParseError(t *testing.T) {
	t.Parallel()
	service := NewService()
	_, err := service.EvalFile(context.Background(), "not valid go {{{", "entrypoint")
	require.ErrorIs(t, err, errParse)
}

func TestEvalFile_TypeCheckError(t *testing.T) {
	t.Parallel()
	source := `package main

func entrypoint() int {
	return "not an int"
}

func main() {}
`
	service := NewService()
	_, err := service.EvalFile(context.Background(), source, "entrypoint")
	require.ErrorIs(t, err, errTypeCheck)
}

func TestCompileFileSet_SingleFile(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
		"main.go": `package main

func entrypoint() int {
	return 100
}

func main() {}
`,
	}

	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "100", fmt.Sprint(result))
}

func TestCompileFileSet_MultiFile(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
		"helpers.go": `package main

func triple(x int) int {
	return x * 3
}
`,
		"main.go": `package main

func entrypoint() int {
	return triple(14)
}

func main() {}
`,
	}

	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "42", fmt.Sprint(result))
}

func TestCompileFileSet_MultiFile_BidirectionalCalls(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
		"a.go": `package main

func addOne(x int) int {
	return x + 1
}

func entrypoint() int {
	return doubleAndAdd(20)
}
`,
		"b.go": `package main

func doubleAndAdd(x int) int {
	return addOne(x * 2)
}
`,
	}

	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "41", fmt.Sprint(result))
}

func TestCompileFileSet_ReExecute(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
		"main.go": `package main

func entrypoint() int {
	return 7 * 6
}

func main() {}
`,
	}

	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	for range 3 {
		result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
		require.NoError(t, err)
		require.Equal(t, "42", fmt.Sprint(result))
	}
}

func TestCompileFileSet_DifferentEntrypoints(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
		"main.go": `package main

func alpha() string {
	return "alpha"
}

func beta() string {
	return "beta"
}

func main() {}
`,
	}

	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	r1, err := service.ExecuteEntrypoint(context.Background(), cfs, "alpha")
	require.NoError(t, err)
	require.Equal(t, "alpha", fmt.Sprint(r1))

	r2, err := service.ExecuteEntrypoint(context.Background(), cfs, "beta")
	require.NoError(t, err)
	require.Equal(t, "beta", fmt.Sprint(r2))
}

func TestEvalFile_PackageLevelVars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect string
	}{
		{
			name: "global int var",
			source: `package main

var counter int

func inc() { counter++ }

func Main() int {
	inc()
	inc()
	inc()
	return counter
}
`,
			expect: "3",
		},
		{
			name: "global with initialiser",
			source: `package main

var base int = 100

func add(n int) int { return base + n }

func Main() int {
	return add(42)
}
`,
			expect: "142",
		},
		{
			name: "multiple global types",
			source: `package main

var x int = 10
var y float64 = 2.5
var name string = "piko"

func Main() string {
	return name
}
`,
			expect: "piko",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, "Main")
			require.NoError(t, err)
			require.Equal(t, tt.expect, fmt.Sprint(result))
		})
	}
}

func TestCompileFileSet_ErrorInSource(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
		"main.go": `package main

func entrypoint() int {
	return "not an int"
}
`,
	}

	service := NewService()
	_, err := service.CompileFileSet(context.Background(), sources)
	require.Error(t, err)
	require.ErrorIs(t, err, errTypeCheck)
}

func TestCompileProgram_CrossPackage(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/lib"

func entrypoint() int {
	return lib.Add(1, 2)
}

func main() {}
`,
		},
		"lib": {
			"lib.go": `package lib

func Add(a, b int) int {
	return a + b
}
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "3", fmt.Sprint(result))
}

func TestCompileProgram_CrossPackageConstants(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/lib"

func entrypoint() int {
	return lib.Answer
}

func main() {}
`,
		},
		"lib": {
			"lib.go": `package lib

const Answer = 42
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "42", fmt.Sprint(result))
}

func TestCompileProgram_CrossPackageFloatConst(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/lib"

func entrypoint() float64 {
	return lib.Pi
}

func main() {}
`,
		},
		"lib": {
			"lib.go": `package lib

const Pi = 3.14
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "3.14", fmt.Sprint(result))
}

func TestCompileProgram_CrossPackageBoolConst(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/lib"

func entrypoint() bool {
	return lib.Flag
}

func main() {}
`,
		},
		"lib": {
			"lib.go": `package lib

const Flag = true
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "true", fmt.Sprint(result))
}

func TestCompileProgram_CrossPackageStringConst(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/lib"

func entrypoint() string {
	return lib.Greeting
}

func main() {}
`,
		},
		"lib": {
			"lib.go": `package lib

const Greeting = "hello"
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "hello", fmt.Sprint(result))
}

func TestCompileProgram_DependencyChain(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/b"

func entrypoint() int {
	return b.Double(21)
}

func main() {}
`,
		},
		"a": {
			"a.go": `package a

func Add(x, y int) int {
	return x + y
}
`,
		},
		"b": {
			"b.go": `package b

import "testmod/a"

func Double(x int) int {
	return a.Add(x, x)
}
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "42", fmt.Sprint(result))
}

func TestCompileProgram_CyclicImport(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/a"

func entrypoint() int { return a.X() }
func main() {}
`,
		},
		"a": {
			"a.go": `package a

import "testmod/b"

func X() int { return b.Y() }
`,
		},
		"b": {
			"b.go": `package b

import "testmod/a"

func Y() int { return a.X() }
`,
		},
	}

	service := NewService()
	_, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.ErrorIs(t, err, errCyclicImport)
}

func TestCompileProgram_NoMainPackage(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"lib": {
			"lib.go": `package lib

func Hello() string { return "hello" }
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	err = service.ExecuteInits(context.Background(), cfs)
	require.NoError(t, err)
}

func TestExecuteEntrypoint_NotFound(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
		"main.go": `package main

func other() int { return 1 }
func main() {}
`,
	}

	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	_, err = service.ExecuteEntrypoint(context.Background(), cfs, "nonexistent")
	require.ErrorIs(t, err, errEntrypointNotFound)
}

func TestEvalFile_VariadicBasic(t *testing.T) {
	t.Parallel()
	source := `package main

func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func run() int {
	return sum(10, 20, 12)
}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "42", fmt.Sprint(result))
}

func TestEvalFile_VariadicEmpty(t *testing.T) {
	t.Parallel()
	source := `package main

func count(arguments ...int) int {
	return len(arguments)
}

func run() int {
	return count()
}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "0", fmt.Sprint(result))
}

func TestEvalFile_VariadicSpread(t *testing.T) {
	t.Parallel()
	source := `package main

func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func run() int {
	s := []int{1, 2, 3, 4, 5}
	return sum(s...)
}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "15", fmt.Sprint(result))
}

func TestEvalFile_VariadicMixed(t *testing.T) {
	t.Parallel()
	source := `package main

func first(prefix string, nums ...int) int {
	r := len(prefix)
	for _, n := range nums {
		r += n
	}
	return r
}

func run() int {
	return first("abc", 10, 20)
}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "33", fmt.Sprint(result))
}

func TestEvalFile_NativeVariadic(t *testing.T) {
	t.Parallel()
	source := `package main

import "fmt"

func run() string {
	return fmt.Sprintf("hello %s, you are %d", "world", 42)
}
`
	symbols := NewSymbolRegistry(SymbolExports{
		"fmt": {
			"Sprintf": reflect.ValueOf(fmt.Sprintf),
		},
	})
	service := NewService()
	service.UseSymbols(symbols)
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoError(t, err)
	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "run")
	require.NoError(t, err)
	require.Equal(t, "hello world, you are 42", result)
}

func TestCompileProgram_BlankImport(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import _ "testmod/sideeffect"

func entrypoint() int {
	return 1
}

func main() {}
`,
		},
		"sideeffect": {
			"sideeffect.go": `package sideeffect

func helper() int { return 42 }

func init() {
	_ = helper()
}
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "1", fmt.Sprint(result))
}

func TestCompileProgram_CrossPackageMethodCall(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/lib"

func entrypoint() int {
	c := lib.NewCounter(10)
	c = c.Add(5)
	return c.Value()
}

func main() {}
`,
		},
		"lib": {
			"lib.go": `package lib

type Counter struct { V int }

func NewCounter(v int) Counter { return Counter{V: v} }

func (c Counter) Add(x int) Counter { return Counter{V: c.V + x} }

func (c Counter) Value() int { return c.V }
`,
		},
	}

	service := NewService()
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)
	require.Equal(t, "15", fmt.Sprint(result))
}

func TestCompileFileSet_BuildTagExclude(t *testing.T) {
	t.Parallel()

	sources := map[string]string{
		"main.go": `package main

func Run() int { return 1 }
`,
		"extra.go": "//go:build !custom\n\npackage main\n\nfunc Run() int { return 2 }\n",
	}

	service := NewService(WithBuildTags("custom"))
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "Run")
	require.NoError(t, err)
	require.Equal(t, "1", fmt.Sprint(result))
}

func TestCompileFileSet_BuildTagInclude(t *testing.T) {
	t.Parallel()

	sources := map[string]string{
		"main.go": `package main

func Entry() int { return helper() }
`,
		"impl_custom.go":  "//go:build custom\n\npackage main\n\nfunc helper() int { return 42 }\n",
		"impl_default.go": "//go:build !custom\n\npackage main\n\nfunc helper() int { return 0 }\n",
	}

	service := NewService(WithBuildTags("custom"))
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "Entry")
	require.NoError(t, err)
	require.Equal(t, "42", fmt.Sprint(result))
}

func TestCompileFileSet_BuildTagIgnore(t *testing.T) {
	t.Parallel()

	sources := map[string]string{
		"main.go": `package main

func Run() int { return 1 }
`,
		"ignored.go": "//go:build ignore\n\npackage main\n\nfunc Run() int { return 999 }\n",
	}

	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "Run")
	require.NoError(t, err)
	require.Equal(t, "1", fmt.Sprint(result))
}

func TestCompileFileSet_BuildTagGOOS(t *testing.T) {
	t.Parallel()

	sources := map[string]string{
		"main.go": `package main

func Run() string { return "base" }
`,
		"platform.go": "//go:build " + runtime.GOOS + "\n\npackage main\n\nfunc Platform() string { return \"" + runtime.GOOS + "\" }\n",
		"other.go":    "//go:build fakeos\n\npackage main\n\nfunc Platform() string { return \"fake\" }\n",
	}

	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "Platform")
	require.NoError(t, err)
	require.Equal(t, runtime.GOOS, fmt.Sprint(result))
}

func TestCompileFileSet_BuildTagAllExcluded(t *testing.T) {
	t.Parallel()

	sources := map[string]string{
		"a.go": "//go:build ignore\n\npackage main\n\nfunc Run() int { return 1 }\n",
	}

	service := NewService()
	_, err := service.CompileFileSet(context.Background(), sources)
	require.ErrorIs(t, err, errParse)
	require.ErrorContains(t, err, "all files excluded")
}

func TestCompileProgram_BuildTags(t *testing.T) {
	t.Parallel()

	sources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "testmod/util"

func Run() int { return util.Val() }

func main() {}
`,
		},
		"util": {
			"val_tagged.go":  "//go:build interp_test\n\npackage util\n\nfunc Val() int { return 100 }\n",
			"val_default.go": "//go:build !interp_test\n\npackage util\n\nfunc Val() int { return 0 }\n",
		},
	}

	service := NewService(WithBuildTags("interp_test"))
	cfs, err := service.CompileProgram(context.Background(), "testmod", sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "Run")
	require.NoError(t, err)
	require.Equal(t, "100", fmt.Sprint(result))
}

func TestCompileFileSet_BuildTagBooleanLogic(t *testing.T) {
	t.Parallel()

	sources := map[string]string{
		"main.go": `package main

func Run() int { return helper() }
`,
		"impl_ab.go":   "//go:build tagA && tagB\n\npackage main\n\nfunc helper() int { return 1 }\n",
		"impl_a.go":    "//go:build tagA && !tagB\n\npackage main\n\nfunc helper() int { return 2 }\n",
		"impl_none.go": "//go:build !tagA && !tagB\n\npackage main\n\nfunc helper() int { return 3 }\n",
	}

	service := NewService(WithBuildTags("tagA"))
	cfs, err := service.CompileFileSet(context.Background(), sources)
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "Run")
	require.NoError(t, err)
	require.Equal(t, "2", fmt.Sprint(result))
}
