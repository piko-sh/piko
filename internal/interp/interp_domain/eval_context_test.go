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
	"time"
)

func TestEvalCancelledContext(t *testing.T) {
	t.Parallel()
	service := NewService()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test: pre-cancelled eval context"))

	_, err := service.Eval(ctx, `1 + 2`)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, errExecutionCancelled) {
		t.Fatalf("expected errExecutionCancelled, got: %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled in chain, got: %v", err)
	}
}

func TestEvalTimeout(t *testing.T) {
	t.Parallel()
	service := NewService()
	service.UseSymbols(newOSSymbols())

	ctx, cancel := context.WithTimeoutCause(context.Background(), 100*time.Millisecond, errors.New("test: eval timeout"))
	defer cancel()

	_, err := service.Eval(ctx, `
import "os"
x := 0
for {
	x++
	os.Getenv("PATH")
}
`)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestCompileCancelledContext(t *testing.T) {
	t.Parallel()
	service := NewService()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test: pre-cancelled compile context"))

	_, err := service.Compile(ctx, `1 + 2`)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, errExecutionCancelled) {
		t.Fatalf("expected errExecutionCancelled, got: %v", err)
	}
}

func TestExecuteEntrypointCancelledContext(t *testing.T) {
	t.Parallel()
	service := NewService()

	source := `package main
func entrypoint() int { return 42 }
`
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test: pre-cancelled entrypoint context"))

	_, err = service.ExecuteEntrypoint(ctx, cfs, "entrypoint")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, errExecutionCancelled) {
		t.Fatalf("expected errExecutionCancelled, got: %v", err)
	}
}

func TestMaxExecutionTime(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxExecutionTime(100 * time.Millisecond))
	service.UseSymbols(newOSSymbols())

	_, err := service.Eval(context.Background(), `
import "os"
x := 0
for {
	x++
	os.Getenv("PATH")
}
`)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestEvalCancelDuringExecution(t *testing.T) {
	t.Parallel()
	service := NewService()
	service.UseSymbols(newOSSymbols())

	ctx, cancel := context.WithCancelCause(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel(errors.New("test: cancel during execution"))
	}()

	_, err := service.Eval(ctx, `
import "os"
x := 0
for {
	x++
	os.Getenv("PATH")
}
`)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}
