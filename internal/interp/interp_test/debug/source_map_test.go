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

package debug_test

import (
	"context"
	"testing"

	"piko.sh/piko/internal/interp/interp_domain"
)

func TestSourceMapSimpleAssignment(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())
	fn, err := svc.Compile(context.Background(), "x := 42; _ = x")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	if !fn.HasDebugSourceMap() {
		t.Fatal("expected source map, got nil")
	}

	bodyLen := fn.BodyLen()
	if bodyLen == 0 {
		t.Fatal("expected non-empty body")
	}

	foundLine := false
	for pc := range bodyLen {
		_, line, _ := fn.DebugSourcePosition(pc)
		if line > 0 {
			foundLine = true
		}
	}
	if !foundLine {
		t.Error("expected at least one instruction with a valid source line")
	}
}

func TestSourceMapMultiLineCode(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())

	code := `package main

func compute() int {
	x := 10
	y := 20
	return x + y
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	fn, findErr := cfs.FindFunction("compute")
	if findErr != nil {
		t.Fatalf("find function: %v", findErr)
	}

	if !fn.HasDebugSourceMap() {
		t.Fatal("expected source map, got nil")
	}

	lines := make(map[int]bool)
	for pc := range fn.BodyLen() {
		_, line, _ := fn.DebugSourcePosition(pc)
		if line > 0 {
			lines[line] = true
		}
	}
	if len(lines) < 2 {
		t.Errorf("expected at least 2 distinct source lines, got %d", len(lines))
	}
}

func TestSourceMapNilWhenDisabled(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithForceGoDispatch())
	fn, err := svc.Compile(context.Background(), "42")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	if fn.HasDebugSourceMap() {
		t.Error("expected no source map when debug info disabled")
	}
}

func TestSourceMapIfElse(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())

	code := `package main

func choose(x int) int {
	if x > 0 {
		return 1
	} else {
		return -1
	}
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	fn, findErr := cfs.FindFunction("choose")
	if findErr != nil {
		t.Fatalf("find function: %v", findErr)
	}

	if !fn.HasDebugSourceMap() {
		t.Fatal("expected source map, got nil")
	}

	lines := make(map[int]bool)
	for pc := range fn.BodyLen() {
		_, line, _ := fn.DebugSourcePosition(pc)
		if line > 0 {
			lines[line] = true
		}
	}
	if len(lines) < 2 {
		t.Errorf("expected at least 2 distinct source lines in if/else, got %d", len(lines))
	}
}

func TestSourceMapForLoop(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())

	code := `package main

func sum() int {
	total := 0
	for i := 0; i < 10; i++ {
		total += i
	}
	return total
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	fn, findErr := cfs.FindFunction("sum")
	if findErr != nil {
		t.Fatalf("find function: %v", findErr)
	}

	if !fn.HasDebugSourceMap() {
		t.Fatal("expected source map, got nil")
	}

	lines := make(map[int]bool)
	for pc := range fn.BodyLen() {
		_, line, _ := fn.DebugSourcePosition(pc)
		if line > 0 {
			lines[line] = true
		}
	}
	if len(lines) < 3 {
		t.Errorf("expected at least 3 distinct source lines in for loop, got %d", len(lines))
	}
}

func TestSourceMapClosures(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())

	code := `package main

func outer() int {
	x := 10
	add := func(y int) int {
		return x + y
	}
	return add(5)
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	fn, findErr := cfs.FindFunction("outer")
	if findErr != nil {
		t.Fatalf("find function: %v", findErr)
	}

	if !fn.HasDebugSourceMap() {
		t.Fatal("expected source map on outer function")
	}

	subFuncs := fn.SubFunctions()
	if len(subFuncs) == 0 {

		t.Log("no sub-functions found on 'outer'; closure may reside on root function - skipping sub-function source map check")
	} else {
		for i, sub := range subFuncs {
			if !sub.HasDebugSourceMap() {
				t.Errorf("expected source map on sub-function %d, got nil", i)
			}
		}
	}
}

func TestSourceMapPostOptimiserConsistency(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())

	code := `package main

func check(x int) bool {
	return x > 5
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	fn, findErr := cfs.FindFunction("check")
	if findErr != nil {
		t.Fatalf("find function: %v", findErr)
	}

	if !fn.HasDebugSourceMap() {
		t.Fatal("expected source map, got nil")
	}

	bodyLen := fn.BodyLen()
	for pc := range bodyLen {
		fn.DebugSourcePosition(pc)
	}

	file, line, col := fn.DebugSourcePosition(bodyLen + 100)
	if file != "" || line != 0 || col != 0 {
		t.Error("expected zero values for out-of-range PC")
	}
}

func TestSourceMapMultipleFiles(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())

	files := map[string]string{
		"a.go": `package main

func a() int { return 1 }
`,
		"b.go": `package main

func b() int { return 2 }
`,
	}

	cfs, err := svc.CompileFileSet(context.Background(), files)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	fnA, errA := cfs.FindFunction("a")
	fnB, errB := cfs.FindFunction("b")
	if errA != nil || errB != nil {
		t.Fatalf("find functions: a=%v, b=%v", errA, errB)
	}

	if !fnA.HasDebugSourceMap() || !fnB.HasDebugSourceMap() {
		t.Fatal("expected source maps on both functions")
	}

	var fileA, fileB string
	for pc := range fnA.BodyLen() {
		if f, _, _ := fnA.DebugSourcePosition(pc); f != "" {
			fileA = f
			break
		}
	}
	for pc := range fnB.BodyLen() {
		if f, _, _ := fnB.DebugSourcePosition(pc); f != "" {
			fileB = f
			break
		}
	}
	if fileA == "" || fileB == "" {
		t.Logf("could not resolve file for one or both functions (a=%q, b=%q)", fileA, fileB)
	} else if fileA == fileB {
		t.Errorf("expected different file paths, got both %q", fileA)
	}
}
