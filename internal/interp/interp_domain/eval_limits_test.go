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
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMaxAllocSizeSlice(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxAllocSize(100))

	_, err := service.Eval(context.Background(), `make([]int, 1000)`)
	if err == nil {
		t.Fatal("expected error for oversized slice allocation")
	}
	if !errors.Is(err, errAllocationLimit) {
		t.Fatalf("expected errAllocationLimit, got: %v", err)
	}
}

func TestMaxAllocSizeSliceBelowLimit(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxAllocSize(100))

	result, err := service.Eval(context.Background(), `len(make([]int, 50))`)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := result.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", result)
	}
	if got != 50 {
		t.Fatalf("expected 50, got %v", got)
	}
}

func TestMaxAllocSizeChan(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxAllocSize(100))

	_, err := service.Eval(context.Background(), `make(chan int, 1000)`)
	if err == nil {
		t.Fatal("expected error for oversized chan allocation")
	}
	if !errors.Is(err, errAllocationLimit) {
		t.Fatalf("expected errAllocationLimit, got: %v", err)
	}
}

func TestMaxAllocSizeUnlimited(t *testing.T) {
	t.Parallel()
	service := NewService()

	result, err := service.Eval(context.Background(), `len(make([]int, 10000))`)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := result.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", result)
	}
	if got != 10000 {
		t.Fatalf("expected 10000, got %v", got)
	}
}

func TestMaxGoroutines(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxGoroutines(3))

	_, err := service.Eval(context.Background(), `
for i := 0; i < 10; i++ {
	go func() {}()
}
`)
	if err == nil {
		t.Fatal("expected error for goroutine limit")
	}
	if !errors.Is(err, errGoroutineLimit) {
		t.Fatalf("expected errGoroutineLimit, got: %v", err)
	}

	runtime.Gosched()
}

func TestMaxGoroutinesUnlimited(t *testing.T) {
	t.Parallel()
	service := NewService()

	_, err := service.Eval(context.Background(), `
func noop() {}
for i := 0; i < 10; i++ {
	go noop()
}
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMaxCallDepth(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxCallDepth(50))

	_, err := service.Eval(context.Background(), `
func recurse(n int) int {
	return 1 + recurse(n + 1)
}
recurse(0)
`)
	if err == nil {
		t.Fatal("expected stack overflow error")
	}
	if !errors.Is(err, errStackOverflow) {
		t.Fatalf("expected errStackOverflow, got: %v", err)
	}
}

func TestMaxCallDepthDefault(t *testing.T) {
	t.Parallel()
	service := NewService()

	_, err := service.Eval(context.Background(), `
func recurse(n int) int {
	if n <= 0 { return 0 }
	return recurse(n - 1)
}
recurse(100)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMaxOutputSize(t *testing.T) {
	t.Parallel()
	service := NewService(WithMaxOutputSize(50))

	var buffer bytes.Buffer
	svcVM := newVM(context.Background(), service.globals, service.symbols)
	svcVM.stderrWriter = &buffer
	svcVM.limits = service.limits

	compiledFunction, err := service.Compile(context.Background(), `
for i := 0; i < 1000; i++ {
	println("hello world")
}
`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svcVM.execute(compiledFunction)
	if err == nil {
		t.Fatal("expected error for output limit")
	}
	if !errors.Is(err, errOutputLimit) {
		t.Fatalf("expected errOutputLimit, got: %v", err)
	}
}

func TestMaxOutputSizeUnlimited(t *testing.T) {
	t.Parallel()
	service := NewService()

	var buffer bytes.Buffer
	svcVM := newVM(context.Background(), service.globals, service.symbols)
	svcVM.stderrWriter = &buffer
	svcVM.limits = service.limits

	compiledFunction, err := service.Compile(context.Background(), `
for i := 0; i < 10; i++ {
	println("hello")
}
`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = svcVM.execute(compiledFunction)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buffer.Len() == 0 {
		t.Fatal("expected some output")
	}
}

func TestAtomicCancellationBackgroundCtx(t *testing.T) {
	t.Parallel()
	vm := newVM(context.Background(), newGlobalStore(), NewSymbolRegistry(nil))
	if vm.cancelled.Load() != 0 {
		t.Fatal("expected cancelled flag to be 0 for background context")
	}
}

func TestAtomicCancellationTimeout(t *testing.T) {
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

func TestAtomicCancellationFlagSet(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancelCause(context.Background())
	vm := newVM(ctx, newGlobalStore(), NewSymbolRegistry(nil))

	if vm.cancelled.Load() != 0 {
		t.Fatal("expected cancelled flag to be 0 before cancel")
	}

	cancel(errors.New("test: cancellation flag propagation"))
	deadline := time.After(time.Second)
	for vm.cancelled.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for cancelled flag")
		default:
		}
	}
}

func TestWideGlobalVariables(t *testing.T) {
	t.Parallel()

	const globalCount = 300

	t.Run("int globals exceed uint8 boundary", func(t *testing.T) {
		t.Parallel()

		var source strings.Builder
		source.WriteString("package main\n\n")
		for i := range globalCount {
			fmt.Fprintf(&source, "var g%d int\n", i)
		}
		source.WriteString("\nfunc run() int {\n")
		source.WriteString("\tg255 = 100\n")
		source.WriteString("\tg256 = 200\n")
		source.WriteString("\tg299 = 300\n")
		source.WriteString("\treturn g255 + g256 + g299\n")
		source.WriteString("}\nfunc main() {}\n")

		service := NewService()
		result, err := service.EvalFile(context.Background(), source.String(), "run")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != int64(600) {
			t.Fatalf("expected 600, got %v", result)
		}
	})

	t.Run("string globals exceed uint8 boundary", func(t *testing.T) {
		t.Parallel()

		var source strings.Builder
		source.WriteString("package main\n\n")
		for i := range globalCount {
			fmt.Fprintf(&source, "var s%d string\n", i)
		}
		source.WriteString("\nfunc run() string {\n")
		source.WriteString("\ts256 = \"hello\"\n")
		source.WriteString("\ts257 = \" world\"\n")
		source.WriteString("\treturn s256 + s257\n")
		source.WriteString("}\nfunc main() {}\n")

		service := NewService()
		result, err := service.EvalFile(context.Background(), source.String(), "run")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "hello world" {
			t.Fatalf("expected \"hello world\", got %v", result)
		}
	})

	t.Run("bool globals exceed uint8 boundary", func(t *testing.T) {
		t.Parallel()

		var source strings.Builder
		source.WriteString("package main\n\n")
		for i := range globalCount {
			fmt.Fprintf(&source, "var b%d bool\n", i)
		}
		source.WriteString("\nfunc run() bool {\n")
		source.WriteString("\tb256 = true\n")
		source.WriteString("\treturn b256\n")
		source.WriteString("}\nfunc main() {}\n")

		service := NewService()
		result, err := service.EvalFile(context.Background(), source.String(), "run")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != true {
			t.Fatalf("expected true, got %v", result)
		}
	})

	t.Run("global initialiser with wide index", func(t *testing.T) {
		t.Parallel()

		var source strings.Builder
		source.WriteString("package main\n\n")
		for i := range globalCount {
			if i == 260 {
				fmt.Fprintf(&source, "var g%d int = 42\n", i)
			} else {
				fmt.Fprintf(&source, "var g%d int\n", i)
			}
		}
		source.WriteString("\nfunc run() int { return g260 }\n")
		source.WriteString("func main() {}\n")

		service := NewService()
		result, err := service.EvalFile(context.Background(), source.String(), "run")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != int64(42) {
			t.Fatalf("expected 42, got %v", result)
		}
	})
}

func TestGoroutineTrackerDecrement(t *testing.T) {
	t.Parallel()
	tracker := &resourceTracker{}
	tracker.goroutineCount.Store(5)
	tracker.goroutineCount.Add(-1)
	if tracker.goroutineCount.Load() != 4 {
		t.Fatalf("expected 4, got %d", tracker.goroutineCount.Load())
	}
}
