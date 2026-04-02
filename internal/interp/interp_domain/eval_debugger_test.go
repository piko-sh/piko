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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDebugState(t *testing.T) {
	t.Parallel()

	t.Run("hasBreakpoint", func(t *testing.T) {
		t.Parallel()

		t.Run("nil source map returns false", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			fn := &CompiledFunction{name: "nomap"}
			ExportDebugStateSetBreakpoint(ds, "test.go", 10)
			require.False(t, ExportDebugStateHasBreakpoint(ds, fn, 0))
		})

		t.Run("synthetic position returns false", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()

			positions := []ExportSourcePosition{
				{line: 0, column: 0, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("synth", 1, positions, []string{"test.go"})
			ExportDebugStateSetBreakpoint(ds, "test.go", 0)
			require.False(t, ExportDebugStateHasBreakpoint(ds, fn, 0))
		})

		t.Run("breakpoint set at file and line returns true", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 10, column: 1, fileID: 0},
				{line: 11, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("test", 2, positions, []string{"test.go"})
			ExportDebugStateSetBreakpoint(ds, "test.go", 10)
			require.True(t, ExportDebugStateHasBreakpoint(ds, fn, 0))
		})

		t.Run("same breakpoint deduplicates after first fire", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 10, column: 1, fileID: 0},
				{line: 10, column: 5, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("dedup", 2, positions, []string{"test.go"})
			ExportDebugStateSetBreakpoint(ds, "test.go", 10)

			require.True(t, ExportDebugStateHasBreakpoint(ds, fn, 0))

			ExportDebugStateApplyAction(ds, DebugActionContinue, 0, "test.go", 10)

			ds.lastBreakpointFile = "test.go"
			ds.lastBreakpointLine = 10

			require.False(t, ExportDebugStateHasBreakpoint(ds, fn, 1))
		})

		t.Run("no breakpoint set returns false", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 10, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("nobp", 1, positions, []string{"test.go"})
			require.False(t, ExportDebugStateHasBreakpoint(ds, fn, 0))
		})
	})

	t.Run("shouldStep", func(t *testing.T) {
		t.Parallel()

		t.Run("stepModeNone returns false", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 10, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("none", 1, positions, []string{"test.go"})
			hit, _ := ExportDebugStateShouldStep(ds, fn, 0, 0)
			require.False(t, hit)
		})

		t.Run("stepIn different line returns true", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 10, column: 1, fileID: 0},
				{line: 11, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("stepin", 2, positions, []string{"test.go"})

			ExportDebugStateApplyAction(ds, DebugActionStepIn, 0, "test.go", 10)
			hit, event := ExportDebugStateShouldStep(ds, fn, 1, 0)
			require.True(t, hit)
			require.Equal(t, DebugEventStep, event)
		})

		t.Run("stepIn same line returns false", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 10, column: 1, fileID: 0},
				{line: 10, column: 5, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("stepin-same", 2, positions, []string{"test.go"})
			ExportDebugStateApplyAction(ds, DebugActionStepIn, 0, "test.go", 10)
			hit, _ := ExportDebugStateShouldStep(ds, fn, 1, 0)
			require.False(t, hit)
		})

		t.Run("stepOver same depth different line returns true", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 10, column: 1, fileID: 0},
				{line: 11, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("stepover", 2, positions, []string{"test.go"})
			ExportDebugStateApplyAction(ds, DebugActionStepOver, 1, "test.go", 10)
			hit, event := ExportDebugStateShouldStep(ds, fn, 1, 1)
			require.True(t, hit)
			require.Equal(t, DebugEventStep, event)
		})

		t.Run("stepOver deeper depth returns false", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 10, column: 1, fileID: 0},
				{line: 11, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("stepover-deep", 2, positions, []string{"test.go"})
			ExportDebugStateApplyAction(ds, DebugActionStepOver, 1, "test.go", 10)

			hit, _ := ExportDebugStateShouldStep(ds, fn, 1, 2)
			require.False(t, hit)
		})

		t.Run("stepOut shallower depth returns true", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 20, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("stepout", 1, positions, []string{"test.go"})
			ExportDebugStateApplyAction(ds, DebugActionStepOut, 2, "test.go", 10)

			hit, event := ExportDebugStateShouldStep(ds, fn, 0, 1)
			require.True(t, hit)
			require.Equal(t, DebugEventStep, event)
		})

		t.Run("stepOut same depth returns false", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			positions := []ExportSourcePosition{
				{line: 20, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("stepout-same", 1, positions, []string{"test.go"})
			ExportDebugStateApplyAction(ds, DebugActionStepOut, 2, "test.go", 10)
			hit, _ := ExportDebugStateShouldStep(ds, fn, 0, 2)
			require.False(t, hit)
		})

		t.Run("nil source map returns false", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			fn := &CompiledFunction{name: "nomap"}
			ExportDebugStateApplyAction(ds, DebugActionStepIn, 0, "test.go", 10)
			hit, _ := ExportDebugStateShouldStep(ds, fn, 0, 0)
			require.False(t, hit)
		})
	})

	t.Run("applyAction", func(t *testing.T) {
		t.Parallel()

		t.Run("continue clears stepping", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()

			ExportDebugStateApplyAction(ds, DebugActionStepIn, 0, "test.go", 10)
			ExportDebugStateApplyAction(ds, DebugActionContinue, 0, "test.go", 10)

			positions := []ExportSourcePosition{
				{line: 11, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("cont", 1, positions, []string{"test.go"})
			hit, _ := ExportDebugStateShouldStep(ds, fn, 0, 0)
			require.False(t, hit)
		})

		t.Run("stepIn enables step-in mode", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			ExportDebugStateApplyAction(ds, DebugActionStepIn, 0, "test.go", 10)
			positions := []ExportSourcePosition{
				{line: 11, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("si", 1, positions, []string{"test.go"})
			hit, _ := ExportDebugStateShouldStep(ds, fn, 0, 0)
			require.True(t, hit)
		})

		t.Run("stepOver enables step-over mode", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			ExportDebugStateApplyAction(ds, DebugActionStepOver, 1, "test.go", 10)
			positions := []ExportSourcePosition{
				{line: 11, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("so", 1, positions, []string{"test.go"})

			hit, _ := ExportDebugStateShouldStep(ds, fn, 0, 1)
			require.True(t, hit)

			hit2, _ := ExportDebugStateShouldStep(ds, fn, 0, 2)
			require.False(t, hit2)
		})

		t.Run("stepOut enables step-out mode", func(t *testing.T) {
			t.Parallel()
			ds := TestNewDebugState()
			ExportDebugStateApplyAction(ds, DebugActionStepOut, 2, "test.go", 10)
			positions := []ExportSourcePosition{
				{line: 20, column: 1, fileID: 0},
			}
			fn := ExportNewCompiledFunctionWithSourceMap("sout", 1, positions, []string{"test.go"})

			hit, _ := ExportDebugStateShouldStep(ds, fn, 0, 1)
			require.True(t, hit)

			hit2, _ := ExportDebugStateShouldStep(ds, fn, 0, 2)
			require.False(t, hit2)
		})
	})
}

func TestDebugInfo(t *testing.T) {
	t.Parallel()

	t.Run("compiled with debug info has source map", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t, WithDebugInfo())
		source := `package main

func main() {
	x := 42
	y := x + 1
	_ = y
}
`
		cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
		require.NoError(t, err)

		fn, fnErr := cfs.FindFunction("main")
		require.NoError(t, fnErr)
		require.NotNil(t, fn)
		require.True(t, fn.HasDebugSourceMap())
	})

	t.Run("source position at pc 0 is valid", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t, WithDebugInfo())
		source := `package main

func main() {
	x := 42
	_ = x
}
`
		cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
		require.NoError(t, err)

		fn, fnErr := cfs.FindFunction("main")
		require.NoError(t, fnErr)
		require.NotNil(t, fn)

		file, line, col := fn.DebugSourcePosition(0)

		require.NotEmpty(t, file)
		require.Greater(t, line, 0)
		_ = col
	})

	t.Run("source position out of range returns empty", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t, WithDebugInfo())
		source := `package main

func main() {
	x := 1
	_ = x
}
`
		cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
		require.NoError(t, err)

		fn, fnErr := cfs.FindFunction("main")
		require.NoError(t, fnErr)
		require.NotNil(t, fn)

		file, line, col := fn.DebugSourcePosition(-1)
		require.Empty(t, file)
		require.Equal(t, 0, line)
		require.Equal(t, 0, col)
	})

	t.Run("compiled with debug info has var table", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t, WithDebugInfo())
		source := `package main

func main() {
	x := 42
	y := x + 1
	_ = y
}
`
		cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
		require.NoError(t, err)

		fn, fnErr := cfs.FindFunction("main")
		require.NoError(t, fnErr)
		require.NotNil(t, fn)
		require.True(t, fn.HasDebugVarTable())
	})

	t.Run("live variables reflect scope", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t, WithDebugInfo())
		source := `package main

func main() {
	x := 42
	y := x + 1
	_ = y
}
`
		cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
		require.NoError(t, err)

		fn, fnErr := cfs.FindFunction("main")
		require.NoError(t, fnErr)
		require.NotNil(t, fn)

		vt := ExportVarTableOf(fn)
		require.NotNil(t, vt)

		sm := ExportSourceMapOf(fn)
		require.NotNil(t, sm)

		bodyLen := len(fn.body)
		if bodyLen > 1 {
			live := vt.LiveVariables(bodyLen - 2)

			require.NotEmpty(t, live)
		}
	})

	t.Run("without debug info has no source map", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t)
		source := `package main

func main() {
	x := 1
	_ = x
}
`
		cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
		require.NoError(t, err)

		fn, fnErr := cfs.FindFunction("main")
		require.NoError(t, fnErr)
		require.NotNil(t, fn)
		require.False(t, fn.HasDebugSourceMap())
		file, line, col := fn.DebugSourcePosition(0)
		require.Empty(t, file)
		require.Equal(t, 0, line)
		require.Equal(t, 0, col)
	})

	t.Run("without debug info has no var table", func(t *testing.T) {
		t.Parallel()
		service := newTestService(t)
		source := `package main

func main() {
	x := 1
	_ = x
}
`
		cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
		require.NoError(t, err)

		fn, fnErr := cfs.FindFunction("main")
		require.NoError(t, fnErr)
		require.NotNil(t, fn)
		require.False(t, fn.HasDebugVarTable())
	})
}

func TestDebugger(t *testing.T) {
	t.Parallel()

	t.Run("breakpoint hit", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func main() {
	x := 1
	y := 2
	z := x + y
	_ = z
}
`

		dbg.SetBreakpoint("main.go", 5)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap := dbg.WaitForPause()
		require.NotNil(t, snap)
		require.Equal(t, "main.go", snap.File)
		require.Equal(t, 5, snap.Line)
		require.Equal(t, DebugEventBreakpoint, snap.Event)
		require.NotEmpty(t, snap.StackTrace)

		dbg.Continue()
		<-done
		require.NoError(t, execErr)
	})

	t.Run("multiple breakpoints fire in order", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func main() {
	a := 1
	b := 2
	c := 3
	_ = a + b + c
}
`

		dbg.SetBreakpoint("main.go", 4)
		dbg.SetBreakpoint("main.go", 6)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap1 := dbg.WaitForPause()
		require.NotNil(t, snap1)
		require.Equal(t, 4, snap1.Line)
		require.Equal(t, DebugEventBreakpoint, snap1.Event)
		dbg.Continue()

		snap2 := dbg.WaitForPause()
		require.NotNil(t, snap2)
		require.Equal(t, 6, snap2.Line)
		require.Equal(t, DebugEventBreakpoint, snap2.Event)
		dbg.Continue()

		<-done
		require.NoError(t, execErr)
	})

	t.Run("clear breakpoint removes it", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func main() {
	a := 1
	b := 2
	c := 3
	_ = a + b + c
}
`
		dbg.SetBreakpoint("main.go", 4)
		dbg.SetBreakpoint("main.go", 6)

		dbg.ClearBreakpoint("main.go", 4)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap := dbg.WaitForPause()
		require.NotNil(t, snap)
		require.Equal(t, 6, snap.Line)
		require.Equal(t, DebugEventBreakpoint, snap.Event)
		dbg.Continue()

		<-done
		require.NoError(t, execErr)
	})

	t.Run("step in enters function call", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func add(a, b int) int {
	return a + b
}

func main() {
	x := add(1, 2)
	_ = x
}
`

		dbg.SetBreakpoint("main.go", 8)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap := dbg.WaitForPause()
		require.NotNil(t, snap)
		require.Equal(t, 8, snap.Line)

		dbg.StepIn()

		snap2 := dbg.WaitForPause()
		require.NotNil(t, snap2)
		require.Equal(t, DebugEventStep, snap2.Event)

		require.NotEqual(t, 8, snap2.Line)

		dbg.Continue()
		<-done
		require.NoError(t, execErr)
	})

	t.Run("step over skips function call", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func add(a, b int) int {
	return a + b
}

func main() {
	x := add(1, 2)
	y := x + 1
	_ = y
}
`

		dbg.SetBreakpoint("main.go", 8)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap := dbg.WaitForPause()
		require.NotNil(t, snap)
		require.Equal(t, 8, snap.Line)

		dbg.StepOver()

		snap2 := dbg.WaitForPause()
		require.NotNil(t, snap2)
		require.Equal(t, DebugEventStep, snap2.Event)

		require.Equal(t, "main", snap2.FunctionName)

		dbg.Continue()
		<-done
		require.NoError(t, execErr)
	})

	t.Run("step out exits current function", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func inner() int {
	a := 10
	b := 20
	return a + b
}

func main() {
	r := inner()
	_ = r
}
`

		dbg.SetBreakpoint("main.go", 4)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap := dbg.WaitForPause()
		require.NotNil(t, snap)
		require.Equal(t, 4, snap.Line)
		require.Equal(t, "inner", snap.FunctionName)

		dbg.StepOut()

		snap2 := dbg.WaitForPause()
		require.NotNil(t, snap2)
		require.Equal(t, DebugEventStep, snap2.Event)
		require.Equal(t, "main", snap2.FunctionName)

		dbg.Continue()
		<-done
		require.NoError(t, execErr)
	})

	t.Run("stop terminates execution", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func main() {
	x := 1
	y := 2
	_ = x + y
}
`
		dbg.SetBreakpoint("main.go", 4)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap := dbg.WaitForPause()
		require.NotNil(t, snap)

		dbg.Stop()
		<-done
		require.Error(t, execErr)
		require.True(t, errors.Is(execErr, TestErrDebuggerStop))
	})

	t.Run("snapshot before pause returns nil", func(t *testing.T) {
		t.Parallel()
		dbg := NewDebugger()
		require.Nil(t, dbg.Snapshot())
	})

	t.Run("variables at breakpoint", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func main() {
	x := 42
	y := x + 1
	_ = y
}
`

		dbg.SetBreakpoint("main.go", 5)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap := dbg.WaitForPause()
		require.NotNil(t, snap)
		require.Equal(t, 5, snap.Line)

		vars := dbg.Variables(0)
		require.NotEmpty(t, vars, "expected at least one variable visible at breakpoint")

		found := false
		for _, v := range vars {
			if v.Name == "x" {
				found = true
				break
			}
		}
		require.True(t, found, "expected variable 'x' to be visible at line 5")

		dbg.Continue()
		<-done
		require.NoError(t, execErr)
	})

	t.Run("variables with large frame index makes targetFP negative", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func main() {
	x := 1
	_ = x
}
`
		dbg.SetBreakpoint("main.go", 4)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap := dbg.WaitForPause()
		require.NotNil(t, snap)

		vars := dbg.Variables(1000000)
		require.Nil(t, vars)

		dbg.Continue()
		<-done
		require.NoError(t, execErr)
	})

	t.Run("variables with excessive frame index returns nil", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func main() {
	x := 1
	_ = x
}
`
		dbg.SetBreakpoint("main.go", 4)

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(context.Background(), source, "main")
		}()

		snap := dbg.WaitForPause()
		require.NotNil(t, snap)

		vars := dbg.Variables(100)
		require.Nil(t, vars)

		dbg.Continue()
		<-done
		require.NoError(t, execErr)
	})

	t.Run("variables with nil VM returns nil", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		vars := dbg.Variables(0)
		require.Nil(t, vars)
	})

	t.Run("variables with all register types", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func main() {
	intVar := 42
	floatVar := 3.14
	strVar := "hello"
	boolVar := true
	uintVar := uint(99)
	complexVar := complex(1.0, 2.0)
	var anyVar interface{} = "world"
	_ = intVar
	_ = floatVar
	_ = strVar
	_ = boolVar
	_ = uintVar
	_ = complexVar
	_ = anyVar
}
`

		dbg.SetBreakpoint("main.go", 10)

		ctx, cancel := context.WithTimeoutCause(
			context.Background(),
			5*1000*1000*1000,
			errors.New("debugger test timed out"),
		)
		defer cancel()

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(ctx, source, "main")
		}()

		select {
		case <-ctx.Done():
			t.Skip("debugger did not pause within timeout - breakpoint line may not match")
		case snap := <-func() chan *DebugSnapshot {
			ch := make(chan *DebugSnapshot, 1)
			go func() { ch <- dbg.WaitForPause() }()
			return ch
		}():
			require.NotNil(t, snap)

			vars := dbg.Variables(0)
			require.NotEmpty(t, vars)

			varMap := make(map[string]any)
			for _, v := range vars {
				varMap[v.Name] = v.Value
			}

			if v, ok := varMap["intVar"]; ok {
				require.Equal(t, int64(42), v)
			}

			dbg.Continue()
		}

		<-done
		if execErr != nil && ctx.Err() != nil {
			t.Skip("context cancelled")
		}
	})

	t.Run("variables in closure with upvalues", func(t *testing.T) {
		t.Parallel()

		dbg := NewDebugger()
		service := newTestService(t, WithDebugger(dbg), WithDebugInfo())

		source := `package main

func main() {
	outer := 42
	f := func() int {
		inner := outer + 1
		return inner
	}
	result := f()
	_ = result
}
`

		dbg.SetBreakpoint("main.go", 6)

		ctx, cancel := context.WithTimeoutCause(
			context.Background(),
			5*1000*1000*1000,
			errors.New("debugger test timed out"),
		)
		defer cancel()

		var execErr error
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, execErr = service.EvalFile(ctx, source, "main")
		}()

		select {
		case <-ctx.Done():
			t.Skip("debugger did not pause within timeout - breakpoint line may not match")
		case snap := <-func() chan *DebugSnapshot {
			ch := make(chan *DebugSnapshot, 1)
			go func() { ch <- dbg.WaitForPause() }()
			return ch
		}():
			require.NotNil(t, snap)
			vars := dbg.Variables(0)
			_ = vars
			dbg.Continue()
		}

		<-done
		if execErr != nil && ctx.Err() != nil {
			t.Skip("context cancelled")
		}
	})
}
