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

func TestVarInspectionInt(t *testing.T) {
	t.Parallel()

	dbg := interp_domain.NewDebugger()
	svc := interp_domain.NewService(interp_domain.WithDebugger(dbg))

	code := `package main

func run() int {
	x := 42
	y := x + 1
	return y
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	dbg.SetBreakpoint("main.go", 5)

	done := make(chan error, 1)
	go func() {
		_, execErr := svc.ExecuteEntrypoint(context.Background(), cfs, "run")
		done <- execErr
	}()

	snap := dbg.WaitForPause()
	if snap == nil {
		t.Fatal("expected pause snapshot")
	}

	vars := dbg.Variables(0)
	found := false
	for _, v := range vars {
		if v.Name == "x" {
			found = true
			if val, ok := v.Value.(int64); !ok || val != 42 {
				t.Errorf("expected x=42 (int64), got %v (%T)", v.Value, v.Value)
			}
		}
	}
	if !found {
		t.Error("variable 'x' not found in inspection results")
	}

	dbg.Continue()
	if execErr := <-done; execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}
}

func TestVarInspectionMultipleTypes(t *testing.T) {
	t.Parallel()

	dbg := interp_domain.NewDebugger()
	svc := interp_domain.NewService(interp_domain.WithDebugger(dbg))

	code := `package main

func run() int {
	n := 100
	s := "hello"
	b := true
	_ = s
	_ = b
	return n
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	dbg.SetBreakpoint("main.go", 9)

	done := make(chan error, 1)
	go func() {
		_, execErr := svc.ExecuteEntrypoint(context.Background(), cfs, "run")
		done <- execErr
	}()

	dbg.WaitForPause()
	vars := dbg.Variables(0)

	varMap := make(map[string]interp_domain.VariableInfo)
	for _, v := range vars {
		varMap[v.Name] = v
	}

	if v, ok := varMap["n"]; ok {
		if val, isInt := v.Value.(int64); !isInt || val != 100 {
			t.Errorf("expected n=100, got %v", v.Value)
		}
	}
	if v, ok := varMap["s"]; ok {
		if val, isStr := v.Value.(string); !isStr || val != "hello" {
			t.Errorf("expected s=\"hello\", got %v", v.Value)
		}
	}
	if v, ok := varMap["b"]; ok {
		if val, isBool := v.Value.(bool); !isBool || !val {
			t.Errorf("expected b=true, got %v", v.Value)
		}
	}

	dbg.Continue()
	if execErr := <-done; execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}
}

func TestVarInspectionAfterReassignment(t *testing.T) {
	t.Parallel()

	dbg := interp_domain.NewDebugger()
	svc := interp_domain.NewService(interp_domain.WithDebugger(dbg))

	code := `package main

func run() int {
	x := 10
	x = 20
	return x
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	dbg.SetBreakpoint("main.go", 6)

	done := make(chan error, 1)
	go func() {
		_, execErr := svc.ExecuteEntrypoint(context.Background(), cfs, "run")
		done <- execErr
	}()

	dbg.WaitForPause()
	vars := dbg.Variables(0)

	for _, v := range vars {
		if v.Name == "x" {
			if val, ok := v.Value.(int64); !ok || val != 20 {
				t.Errorf("expected x=20 after reassignment, got %v", v.Value)
			}
		}
	}

	dbg.Continue()
	if execErr := <-done; execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}
}
