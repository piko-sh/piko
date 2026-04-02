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
	"errors"
	"testing"

	"piko.sh/piko/internal/interp/interp_domain"
)

func TestFullDebugSession(t *testing.T) {
	t.Parallel()

	dbg := interp_domain.NewDebugger()
	svc := interp_domain.NewService(interp_domain.WithDebugger(dbg))

	code := `package main

func run() int {
	a := 1
	b := 2
	c := a + b
	return c
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	dbg.SetBreakpoint("main.go", 5)

	done := make(chan struct {
		result any
		err    error
	}, 1)
	go func() {
		result, execErr := svc.ExecuteEntrypoint(context.Background(), cfs, "run")
		done <- struct {
			result any
			err    error
		}{result: result, err: execErr}
	}()

	snap := dbg.WaitForPause()
	if snap.Line != 5 {
		t.Errorf("expected line 5, got %d", snap.Line)
	}

	vars := dbg.Variables(0)
	_ = vars

	dbg.StepIn()
	snap = dbg.WaitForPause()

	dbg.Continue()

	r := <-done
	if r.err != nil {
		t.Fatalf("execute: %v", r.err)
	}
	if r.result != nil {
		if val, ok := r.result.(int64); ok && val != 3 {
			t.Errorf("expected result 3, got %d", val)
		}
	}
}

func TestDebugActionStopTerminates(t *testing.T) {
	t.Parallel()

	dbg := interp_domain.NewDebugger()
	svc := interp_domain.NewService(interp_domain.WithDebugger(dbg))

	code := `package main

func run() int {
	x := 1
	y := 2
	return x + y
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	dbg.SetBreakpoint("main.go", 4)

	done := make(chan error, 1)
	go func() {
		_, execErr := svc.ExecuteEntrypoint(context.Background(), cfs, "run")
		done <- execErr
	}()

	dbg.WaitForPause()

	dbg.Stop()

	execErr := <-done
	if execErr == nil {
		t.Fatal("expected error from DebugActionStop")
	}
	if !errors.Is(execErr, interp_domain.ErrDebuggerStop) {
		t.Errorf("expected ErrDebuggerStop, got: %v", execErr)
	}
}

func TestCancelContextDuringDebug(t *testing.T) {
	t.Parallel()

	dbg := interp_domain.NewDebugger()
	svc := interp_domain.NewService(interp_domain.WithDebugger(dbg))

	code := `package main

func run() int {
	sum := 0
	for i := 0; i < 100000; i++ {
		sum += i
	}
	return sum
}
`
	cfs, err := svc.CompileFileSet(context.Background(), map[string]string{"main.go": code})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	dbg.SetBreakpoint("main.go", 4)

	ctx, cancel := context.WithCancelCause(context.Background())

	done := make(chan error, 1)
	go func() {
		_, execErr := svc.ExecuteEntrypoint(ctx, cfs, "run")
		done <- execErr
	}()

	dbg.WaitForPause()

	cancel(errors.New("test cancellation"))

	dbg.Continue()

	execErr := <-done
	if execErr == nil {
		t.Fatal("expected error from context cancellation")
	}
}
