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

func TestStackTraceSingleFrame(t *testing.T) {
	t.Parallel()

	dbg := interp_domain.NewDebugger()
	svc := interp_domain.NewService(interp_domain.WithDebugger(dbg))

	code := `package main

func run() int {
	x := 42
	return x
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

	snap := dbg.WaitForPause()
	if snap == nil {
		t.Fatal("expected pause snapshot")
	}
	if len(snap.StackTrace) == 0 {
		t.Fatal("expected non-empty stack trace")
	}
	if snap.StackTrace[0].Line == 0 {
		t.Error("expected non-zero line in top stack frame")
	}

	dbg.Continue()
	if execErr := <-done; execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}
}

func TestStackTraceMultipleFrames(t *testing.T) {
	t.Parallel()

	dbg := interp_domain.NewDebugger()
	svc := interp_domain.NewService(interp_domain.WithDebugger(dbg))

	code := `package main

func inner() int {
	return 42
}

func middle() int {
	v := inner()
	return v
}

func run() int {
	v := middle()
	return v
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

	snap := dbg.WaitForPause()
	if snap == nil {
		t.Fatal("expected pause snapshot")
	}

	if len(snap.StackTrace) < 3 {
		t.Errorf("expected at least 3 stack frames, got %d", len(snap.StackTrace))
	}

	dbg.Continue()
	if execErr := <-done; execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}
}
