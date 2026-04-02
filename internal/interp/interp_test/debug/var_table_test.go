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

func TestVarTableSimpleDeclarations(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())

	code := `package main

func vars() {
	x := 42
	y := "hello"
	_ = x
	_ = y
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	fn, findErr := cfs.FindFunction("vars")
	if findErr != nil {
		t.Fatalf("find function: %v", findErr)
	}

	if !fn.HasDebugVarTable() {
		t.Fatal("expected var table, got nil")
	}
}

func TestVarTableNilWhenDisabled(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithForceGoDispatch())

	fn, err := svc.Compile(context.Background(), "x := 42; _ = x")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	if fn.HasDebugVarTable() {
		t.Error("expected no var table when debug info disabled")
	}
}

func TestVarTableFunctionParams(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())

	code := `package main

func add(a int, b int) int {
	return a + b
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	fn, findErr := cfs.FindFunction("add")
	if findErr != nil {
		t.Fatalf("find function: %v", findErr)
	}

	if !fn.HasDebugVarTable() {
		t.Fatal("expected var table, got nil")
	}
}

func TestVarTableClosures(t *testing.T) {
	t.Parallel()
	svc := interp_domain.NewService(interp_domain.WithDebugInfo(), interp_domain.WithForceGoDispatch())

	code := `package main

func outer() int {
	x := 10
	inc := func() int {
		return x + 1
	}
	return inc()
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

	if !fn.HasDebugVarTable() {
		t.Fatal("expected var table on outer function")
	}

	subFuncs := fn.SubFunctions()
	if len(subFuncs) == 0 {

		t.Log("no sub-functions found on 'outer'; closure may reside on root function - skipping sub-function var table check")
	} else {
		for i, sub := range subFuncs {
			if !sub.HasDebugVarTable() {
				t.Errorf("expected var table on sub-function %d", i)
			}
		}
	}
}
