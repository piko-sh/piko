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

	"github.com/stretchr/testify/require"
)

func requireSpillOccurred(t *testing.T, source string, bank registerKind) {
	t.Helper()
	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoError(t, err)
	fn, err := cfs.FindFunction("run")
	require.NoError(t, err)
	counts := fn.RegisterCounts()
	require.Greater(t, counts[bank], uint32(spillAreaOffset),
		"expected register count to exceed spillAreaOffset (%d), proving spill occurred; got %d for bank %d",
		spillAreaOffset, counts[bank], bank)
}

func generateAllAliveProgram(n int) string {
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}
	b.WriteString("\tresult := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tresult += v%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	return b.String()
}

func generateIntSpillProgram(n int) string {
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}
	b.WriteString("\tsum := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tsum += v%d\n", i)
	}
	mid := n / 2
	fmt.Fprintf(&b, "\treturn v0 + v%d + v%d + sum\n", mid, n-1)
	b.WriteString("}\n")
	return b.String()
}

func triangular(n int) int64 {
	return int64(n) * int64(n-1) / 2
}

func TestRegisterSpillAllAliveInt(t *testing.T) {
	t.Parallel()
	const n = 260
	source := generateAllAliveProgram(n)

	requireSpillOccurred(t, source, registerInt)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, triangular(n), result)
}

func TestRegisterSpillAllAliveString(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() string {\n")
	for i := range n {
		fmt.Fprintf(&b, "\ts%d := \"%d\"\n", i, i)
	}

	b.WriteString("\tresult := \"\"\n")
	for i := range n {
		if i > 0 {
			b.WriteString("\tresult += \"-\"\n")
		}
		fmt.Fprintf(&b, "\tresult += s%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	source := b.String()

	requireSpillOccurred(t, source, registerString)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)

	parts := make([]string, n)
	for i := range n {
		parts[i] = fmt.Sprintf("%d", i)
	}
	require.Equal(t, strings.Join(parts, "-"), result)
}

func TestRegisterSpillBasicInt(t *testing.T) {
	t.Parallel()
	const n = 260
	source := generateIntSpillProgram(n)
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)

	sumAll := n * (n - 1) / 2
	expected := int64(sumAll + 0 + n/2 + (n - 1))
	require.Equal(t, expected, result)
}

func TestRegisterSpillStringBank(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() string {\n")
	for i := range n {
		fmt.Fprintf(&b, "\ts%d := \"%d\"\n", i, i)
	}

	b.WriteString("\tdiscard := \"\"\n")
	for i := range n {
		fmt.Fprintf(&b, "\tdiscard += s%d\n", i)
	}
	b.WriteString("\t_ = discard\n")
	fmt.Fprintf(&b, "\treturn s0 + \"-\" + s%d + \"-\" + s%d\n", n/2, n-1)
	b.WriteString("}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), b.String(), "run")
	require.NoError(t, err)
	expected := fmt.Sprintf("0-%d-%d", n/2, n-1)
	require.Equal(t, expected, result)
}

func TestRegisterSpillReassignment(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}
	b.WriteString("\tsum := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tsum += v%d\n", i)
	}

	b.WriteString("\tv0 = 999\n")
	fmt.Fprintf(&b, "\tv%d = 888\n", n-1)
	fmt.Fprintf(&b, "\treturn v0 + v%d\n", n-1)
	b.WriteString("}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), b.String(), "run")
	require.NoError(t, err)
	require.Equal(t, int64(999+888), result)
}

func TestRegisterSpillInExpression(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}
	b.WriteString("\tsum := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tsum += v%d\n", i)
	}
	b.WriteString("\t_ = sum\n")
	last := n - 1
	secondLast := n - 2
	fmt.Fprintf(&b, "\treturn v%d + v%d\n", secondLast, last)
	b.WriteString("}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), b.String(), "run")
	require.NoError(t, err)
	require.Equal(t, int64(secondLast+last), result)
}

func TestRegisterSpillAsFunctionArgument(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc double(x int) int { return x * 2 }\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}
	b.WriteString("\tsum := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tsum += v%d\n", i)
	}
	b.WriteString("\t_ = sum\n")
	fmt.Fprintf(&b, "\treturn double(v%d)\n", n-1)
	b.WriteString("}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), b.String(), "run")
	require.NoError(t, err)
	require.Equal(t, int64((n-1)*2), result)
}

func TestRegisterSpillInIfElse(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}
	b.WriteString("\tsum := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tsum += v%d\n", i)
	}
	b.WriteString("\t_ = sum\n")
	fmt.Fprintf(&b, "\tif v%d > 100 {\n", n-1)
	fmt.Fprintf(&b, "\t\treturn v%d\n", n-1)
	b.WriteString("\t}\n")
	b.WriteString("\treturn -1\n")
	b.WriteString("}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), b.String(), "run")
	require.NoError(t, err)
	require.Equal(t, int64(n-1), result)
}

func TestRegisterSpillInForLoop(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}
	b.WriteString("\tsum := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tsum += v%d\n", i)
	}
	b.WriteString("\t_ = sum\n")
	b.WriteString("\tloopSum := 0\n")
	b.WriteString("\tfor i := 0; i < 3; i++ {\n")
	fmt.Fprintf(&b, "\t\tloopSum += v%d\n", n-1)
	b.WriteString("\t}\n")
	b.WriteString("\treturn loopSum\n")
	b.WriteString("}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), b.String(), "run")
	require.NoError(t, err)
	require.Equal(t, int64((n-1)*3), result)
}

func TestRegisterSpillMinimal(t *testing.T) {
	t.Parallel()

	const n = 250
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}

	b.WriteString("\tsum := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tsum += v%d\n", i)
	}

	fmt.Fprintf(&b, "\treturn v%d\n", n-1)
	b.WriteString("}\n")

	service := NewService()
	result, err := service.EvalFile(context.Background(), b.String(), "run")
	require.NoError(t, err)
	require.Equal(t, int64(n-1), result)
}

func TestRegisterSpillDeadVariableRecyclingStillWorks(t *testing.T) {
	t.Parallel()
	source := `
package main

func run() int {
	a := 1
	b := 2
	c := a + b
	_ = c
	d := 10
	e := 20
	f := d + e
	return f
}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(30), result)
}

func TestRegisterSpillClosureCapture(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}

	fmt.Fprintf(&b, "\tf := func() int { return v%d }\n", n-1)

	b.WriteString("\tresult := f()\n")
	for i := range n - 1 {
		fmt.Fprintf(&b, "\tresult += v%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	source := b.String()

	requireSpillOccurred(t, source, registerInt)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)

	expected := int64(n-1) + triangular(n-1)
	require.Equal(t, expected, result)
}

func TestRegisterSpillAddressOf(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}

	fmt.Fprintf(&b, "\tp := &v%d\n", n-1)
	b.WriteString("\t*p = 999\n")

	b.WriteString("\tresult := *p\n")
	for i := range n - 1 {
		fmt.Fprintf(&b, "\tresult += v%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	source := b.String()

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)

	expected := int64(999) + triangular(n-1)
	require.Equal(t, expected, result)
}

func TestRegisterSpillIncDec(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}

	fmt.Fprintf(&b, "\tv%d++\n", n-1)
	b.WriteString("\tresult := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tresult += v%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	source := b.String()

	requireSpillOccurred(t, source, registerInt)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)

	require.Equal(t, triangular(n)+1, result)
}

func TestRegisterSpillDecrement(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}
	fmt.Fprintf(&b, "\tv%d--\n", n-1)
	b.WriteString("\tresult := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tresult += v%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	source := b.String()

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, triangular(n)-1, result)
}

func TestRegisterSpillMultipleBanks(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")

	for i := range n {
		fmt.Fprintf(&b, "\ti%d := %d\n", i, i)
	}

	for i := range n {
		fmt.Fprintf(&b, "\ts%d := \"%d\"\n", i, i)
	}

	b.WriteString("\tconcat := \"\"\n")
	for i := range n {
		fmt.Fprintf(&b, "\tconcat += s%d\n", i)
	}
	b.WriteString("\t_ = concat\n")

	b.WriteString("\tresult := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tresult += i%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	source := b.String()

	requireSpillOccurred(t, source, registerInt)
	requireSpillOccurred(t, source, registerString)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, triangular(n), result)
}

func TestRegisterSpillScopeReclaim(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	b.WriteString("\tresult := 0\n")

	b.WriteString("\t{\n")
	for i := range n {
		fmt.Fprintf(&b, "\t\tv%d := %d\n", i, i)
	}

	for i := range n {
		fmt.Fprintf(&b, "\t\tresult += v%d\n", i)
	}
	b.WriteString("\t}\n")

	b.WriteString("\ta := 1\n")
	b.WriteString("\tb := 2\n")
	b.WriteString("\treturn result + a + b\n")
	b.WriteString("}\n")
	source := b.String()

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, triangular(n)+3, result)
}

func TestRegisterSpillCompoundAssignAllOps(t *testing.T) {
	t.Parallel()
	const n = 260
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}

	last := n - 1
	fmt.Fprintf(&b, "\tv%d += 10\n", last)
	fmt.Fprintf(&b, "\tv%d -= 5\n", last)
	fmt.Fprintf(&b, "\tv%d *= 2\n", last)
	fmt.Fprintf(&b, "\tv%d <<= 1\n", last-1)

	b.WriteString("\tresult := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tresult += v%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	source := b.String()

	requireSpillOccurred(t, source, registerInt)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)

	expected := triangular(n) + 269 + 258
	require.Equal(t, expected, result)
}

func TestRegisterSpillStress500(t *testing.T) {
	t.Parallel()
	const n = 500
	source := generateAllAliveProgram(n)

	requireSpillOccurred(t, source, registerInt)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, triangular(n), result)
}

func TestRegisterSpillStress1000(t *testing.T) {
	t.Parallel()
	const n = 1000
	source := generateAllAliveProgram(n)

	requireSpillOccurred(t, source, registerInt)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, triangular(n), result)
}

func TestRegisterSpillStressMixedOps(t *testing.T) {
	t.Parallel()
	const n = 300
	var b strings.Builder
	b.WriteString("package main\n\nfunc identity(x int) int { return x }\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}

	fmt.Fprintf(&b, "\tv%d = 1000\n", n-1)
	fmt.Fprintf(&b, "\tv%d = 2000\n", n-2)

	fmt.Fprintf(&b, "\tv%d += 100\n", n-3)

	fmt.Fprintf(&b, "\tv%d = identity(v%d)\n", n-4, n-4)

	fmt.Fprintf(&b, "\tv%d++\n", n-5)
	fmt.Fprintf(&b, "\tv%d--\n", n-6)

	fmt.Fprintf(&b, "\tif v%d > 0 {\n", n-7)
	fmt.Fprintf(&b, "\t\tv%d += 1\n", n-7)
	b.WriteString("\t}\n")

	b.WriteString("\tfor i := 0; i < 3; i++ {\n")
	fmt.Fprintf(&b, "\t\tv%d += 1\n", n-8)
	b.WriteString("\t}\n")

	b.WriteString("\tresult := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tresult += v%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	source := b.String()

	requireSpillOccurred(t, source, registerInt)

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)

	expected := triangular(n) + 701 + 1702 + 100 + 0 + 1 - 1 + 1 + 3
	require.Equal(t, expected, result)
}
