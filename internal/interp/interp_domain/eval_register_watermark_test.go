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
	"strings"
	"testing"
)

func TestRegisterWatermarkManyFieldAssignments(t *testing.T) {
	t.Parallel()

	const assignmentCount = 300

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("type Item struct {\n\tValue string\n}\n\n")
	source.WriteString("func run() string {\n")
	source.WriteString("\tresult := make([]Item, 0, 300)\n")
	for i := range assignmentCount {
		fmt.Fprintf(&source, "\tresult = append(result, Item{Value: \"v%d\"})\n", i)
	}
	source.WriteString("\treturn result[0].Value + result[299].Value\n")
	source.WriteString("}\n\nfunc main() {}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "v0v299" {
		t.Fatalf("expected \"v0v299\", got %v", result)
	}
}

func TestRegisterWatermarkManyExpressionStatements(t *testing.T) {
	t.Parallel()

	const callCount = 300

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("var counter int\n\n")
	source.WriteString("func inc() int {\n\tcounter++\n\treturn counter\n}\n\n")
	source.WriteString("func run() int {\n")
	for range callCount {
		source.WriteString("\tinc()\n")
	}
	source.WriteString("\treturn counter\n")
	source.WriteString("}\n\nfunc main() {}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(callCount) {
		t.Fatalf("expected %d, got %v", callCount, result)
	}
}

func TestRegisterWatermarkPreservesLocalVariables(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.EvalFile(context.Background(), `package main

func noop(x int) {}

func run() int {
	a := 1
	noop(a + 2)
	b := 3
	noop(b + 4)
	c := 5
	noop(c + 6)
	return a + b + c
}

func main() {}
`, "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(9) {
		t.Fatalf("expected 9, got %v", result)
	}
}

func TestRegisterWatermarkPreservesVarDecl(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.EvalFile(context.Background(), `package main

func noop(x int) {}

func run() int {
	var a int = 10
	noop(a + 1)
	var b int = 20
	noop(b + 2)
	return a + b
}

func main() {}
`, "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(30) {
		t.Fatalf("expected 30, got %v", result)
	}
}

func TestRegisterWatermarkMultiReturnDefine(t *testing.T) {
	t.Parallel()

	const extraStatements = 300

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("var counter int\n\n")
	source.WriteString("func twoValues() (int, int) { return 10, 20 }\n")
	source.WriteString("func inc() int { counter++; return counter }\n\n")
	source.WriteString("func run() int {\n")
	source.WriteString("\ta, b := twoValues()\n")
	for range extraStatements {
		source.WriteString("\tinc()\n")
	}
	source.WriteString("\treturn a + b\n")
	source.WriteString("}\n\nfunc main() {}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(30) {
		t.Fatalf("expected 30, got %v", result)
	}
}

func TestRegisterWatermarkForLoopWithDefine(t *testing.T) {
	t.Parallel()

	const extraStatements = 200

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("var counter int\n\n")
	source.WriteString("func inc() int { counter++; return counter }\n\n")
	source.WriteString("func run() int {\n")
	for range extraStatements {
		source.WriteString("\tinc()\n")
	}
	source.WriteString("\tsum := 0\n")
	source.WriteString("\tfor i := 0; i < 5; i++ {\n")
	source.WriteString("\t\tsum += i\n")
	source.WriteString("\t}\n")
	for range extraStatements {
		source.WriteString("\tinc()\n")
	}
	source.WriteString("\treturn sum\n")
	source.WriteString("}\n\nfunc main() {}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(10) {
		t.Fatalf("expected 10, got %v", result)
	}
}

func TestRegisterWatermarkIfWithInit(t *testing.T) {
	t.Parallel()

	const extraStatements = 200

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("var counter int\n\n")
	source.WriteString("func inc() int { counter++; return counter }\n")
	source.WriteString("func getErr() error { return nil }\n\n")
	source.WriteString("func run() int {\n")
	for range extraStatements {
		source.WriteString("\tinc()\n")
	}
	source.WriteString("\tresult := 42\n")
	source.WriteString("\tif err := getErr(); err != nil {\n")
	source.WriteString("\t\tresult = -1\n")
	source.WriteString("\t}\n")
	for range extraStatements {
		source.WriteString("\tinc()\n")
	}
	source.WriteString("\treturn result\n")
	source.WriteString("}\n\nfunc main() {}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(42) {
		t.Fatalf("expected 42, got %v", result)
	}
}

func TestVarInitFunctionRegisterOverflow(t *testing.T) {
	t.Parallel()

	const globalCount = 300

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("func compute(n int) int { return n * 2 }\n\n")
	for i := range globalCount {
		fmt.Fprintf(&source, "var g%d = compute(%d)\n", i, i)
	}
	source.WriteString("\nfunc run() int { return g0 + g299 }\n")
	source.WriteString("func main() {}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(598) {
		t.Fatalf("expected 598, got %v", result)
	}
}

func TestRegisterWatermarkShortVarDeclDoesNotLeak(t *testing.T) {
	t.Parallel()

	const declCount = 200

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("func compute(n int) int { return n * 2 }\n\n")
	source.WriteString("func run() int {\n")
	source.WriteString("\tsum := 0\n")
	for i := range declCount {
		fmt.Fprintf(&source, "\tv%d := compute(%d)\n", i, i)
		fmt.Fprintf(&source, "\tsum += v%d\n", i)
	}
	source.WriteString("\treturn sum\n")
	source.WriteString("}\n\nfunc main() {}\n")

	var expectedSum int64
	for i := range declCount {
		expectedSum += int64(i * 2)
	}

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedSum {
		t.Fatalf("expected %d, got %v", expectedSum, result)
	}
}

func TestRegisterWatermarkShortVarRedeclDoesNotLeak(t *testing.T) {
	t.Parallel()

	const reassignCount = 200

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("func compute(n int) int { return n * 2 }\n\n")
	source.WriteString("func run() int {\n")
	source.WriteString("\tx := 0\n")
	source.WriteString("\tsum := 0\n")
	for i := range reassignCount {
		fmt.Fprintf(&source, "\tx, v%d := compute(%d), compute(%d)\n", i, i, i+1)
		fmt.Fprintf(&source, "\tsum += v%d\n", i)
	}
	source.WriteString("\tsum += x\n")
	source.WriteString("\treturn sum\n")
	source.WriteString("}\n\nfunc main() {}\n")

	var expectedSum int64
	for i := range reassignCount {
		expectedSum += int64((i + 1) * 2)
	}
	expectedSum += int64((reassignCount - 1) * 2)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedSum {
		t.Fatalf("expected %d, got %v", expectedSum, result)
	}
}

func TestRegisterWatermarkShortVarDeclPreservesValues(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.EvalFile(context.Background(), `package main

func compute(n int) int { return n * 10 }

func run() int {
	a := compute(1)
	b := compute(2)
	c := compute(3)
	d := compute(4)
	e := compute(5)
	return a + b + c + d + e
}

func main() {}
`, "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(150) {
		t.Fatalf("expected 150, got %v", result)
	}
}

func TestRegisterRecyclingPreventsOverflow(t *testing.T) {
	t.Parallel()

	const declCount = 400

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("func compute(n int) int { return n * 2 }\n\n")
	source.WriteString("func run() int {\n")
	source.WriteString("\tsum := 0\n")
	for i := range declCount {
		fmt.Fprintf(&source, "\tv%d := compute(%d)\n", i, i)
		fmt.Fprintf(&source, "\tsum += v%d\n", i)
	}
	source.WriteString("\treturn sum\n")
	source.WriteString("}\n\nfunc main() {}\n")

	var expectedSum int64
	for i := range declCount {
		expectedSum += int64(i * 2)
	}

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedSum {
		t.Fatalf("expected %d, got %v", expectedSum, result)
	}
}

func TestRegisterRecyclingWithGeneralBank(t *testing.T) {
	t.Parallel()

	const declCount = 400

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("type Node struct { Value int }\n\n")
	source.WriteString("func makeNode(n int) *Node { return &Node{Value: n} }\n\n")
	source.WriteString("func run() int {\n")
	source.WriteString("\tsum := 0\n")
	for i := range declCount {
		fmt.Fprintf(&source, "\tn%d := makeNode(%d)\n", i, i)
		fmt.Fprintf(&source, "\tsum += n%d.Value\n", i)
	}
	source.WriteString("\treturn sum\n")
	source.WriteString("}\n\nfunc main() {}\n")

	var expectedSum int64
	for i := range declCount {
		expectedSum += int64(i)
	}

	service := NewService()
	result, err := service.EvalFile(context.Background(), source.String(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedSum {
		t.Fatalf("expected %d, got %v", expectedSum, result)
	}
}

func TestRegisterRecyclingClosureSafety(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.EvalFile(context.Background(), `package main

func run() int {
	x := 10
	f := func() int { return x }
	y := 20
	_ = y
	return f()
}

func main() {}
`, "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(10) {
		t.Fatalf("expected 10, got %v", result)
	}
}

func TestRegisterOverflowErrorIncludesFunctionName(t *testing.T) {
	t.Parallel()

	allocator := &registerAllocator{functionName: "heavyFunction"}
	for range 257 {
		allocator.alloc(registerGeneral)
	}

	if allocator.overflowErr == nil {
		t.Fatal("expected overflow error after 257 allocations")
	}
	if !strings.Contains(allocator.overflowErr.Error(), "heavyFunction") {
		t.Fatalf("expected error to contain function name, got: %v", allocator.overflowErr)
	}
	if !strings.Contains(allocator.overflowErr.Error(), "general") {
		t.Fatalf("expected error to contain register bank name, got: %v", allocator.overflowErr)
	}
}

func TestRegisterAllocatorRestorePanicsOnInvalidSnapshot(t *testing.T) {
	t.Parallel()

	allocator := &registerAllocator{functionName: "testFunc"}
	for range 5 {
		allocator.alloc(registerInt)
	}

	var invalidSnapshot [NumRegisterKinds]uint32
	invalidSnapshot[registerInt] = 10

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic from restore with invalid snapshot")
		}
		message, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", recovered, recovered)
		}
		if !strings.Contains(message, "exceeds") {
			t.Fatalf("expected panic message about exceeding, got: %s", message)
		}
	}()

	allocator.restore(invalidSnapshot)
}
