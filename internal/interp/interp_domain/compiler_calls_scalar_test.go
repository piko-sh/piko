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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func compileSourceForOpcodeCheck(t *testing.T, source string) string {
	t.Helper()
	service := NewService()
	compiled, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoErrorf(t, err, "compile failed")
	return compiled.DisassembleAssembly()
}

func TestSelectDirectCallOpcodeGate_PositiveScalar(t *testing.T) {
	t.Parallel()
	dump := compileSourceForOpcodeCheck(t, `package main
func fib(n int) int {
	if n < 2 { return n }
	return fib(n-1) + fib(n-2)
}
func EntrypointRun() int { return fib(10) }`)

	require.Positivef(t, strings.Count(dump, "CALL_SCALAR"),
		"expected opCallScalar to be emitted for a scalar-only recursive call; disasm:\n%s", dump)
	require.Zerof(t, strings.Count(dump, ":CALL "),
		"expected zero plain opCall instructions in a scalar-only program; disasm:\n%s", dump)
}

func TestSelectDirectCallOpcodeGate_RejectsVariadic(t *testing.T) {
	t.Parallel()
	dump := compileSourceForOpcodeCheck(t, `package main
func sumAll(values ...int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
func EntrypointRun() int { return sumAll(1, 2, 3, 4, 5) }`)

	require.Zerof(t, strings.Count(dump, "CALL_SCALAR"),
		"expected NO opCallScalar for a variadic callee; disasm:\n%s", dump)
}

func TestSelectDirectCallOpcodeGate_RejectsStructParam(t *testing.T) {
	t.Parallel()
	dump := compileSourceForOpcodeCheck(t, `package main
type Pair struct { Left int; Right int }
func add(pair Pair) int { return pair.Left + pair.Right }
func EntrypointRun() int { return add(Pair{Left: 3, Right: 4}) }`)

	require.Zerof(t, strings.Count(dump, "CALL_SCALAR"),
		"expected NO opCallScalar for a struct-parameter callee; disasm:\n%s", dump)
}

func TestSelectDirectCallOpcodeGate_RejectsStructResult(t *testing.T) {
	t.Parallel()
	dump := compileSourceForOpcodeCheck(t, `package main
type Point struct { X int; Y int }
func origin() Point { return Point{X: 0, Y: 0} }
func EntrypointRun() int {
	point := origin()
	return point.X + point.Y
}`)

	require.Zerof(t, strings.Count(dump, "CALL_SCALAR"),
		"expected NO opCallScalar for a struct-return callee; disasm:\n%s", dump)
}

func TestSelectDirectCallOpcodeGate_RejectsSliceParam(t *testing.T) {
	t.Parallel()
	dump := compileSourceForOpcodeCheck(t, `package main
func sumSlice(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
func EntrypointRun() int {
	values := []int{1, 2, 3, 4, 5}
	return sumSlice(values)
}`)

	require.Zerof(t, strings.Count(dump, "CALL_SCALAR"),
		"expected NO opCallScalar for a slice-parameter callee; disasm:\n%s", dump)
}

func TestSelectDirectCallOpcodeGate_RejectsClosureCall(t *testing.T) {
	t.Parallel()
	dump := compileSourceForOpcodeCheck(t, `package main
func makeAdd(delta int) func(int) int {
	return func(value int) int { return value + delta }
}
func EntrypointRun() int {
	addOne := makeAdd(1)
	return addOne(41)
}`)

	require.Positivef(t, strings.Count(dump, ":CALL "),
		"expected plain opCall for the closure call; disasm:\n%s", dump)
}

func TestSelectDirectCallOpcodeGate_AcceptsMixedScalarKinds(t *testing.T) {
	t.Parallel()

	dump := compileSourceForOpcodeCheck(t, `package main
func mix(intValue int, uintValue uint, floatValue float64, boolValue bool, textValue string) int {
	if boolValue {
		return intValue + int(uintValue) + int(floatValue) + len(textValue)
	}
	return 0
}
func EntrypointRun() int {
	value := mix(1, 2, 3.0, true, "abcd")
	return value + 1
}`)

	require.Positivef(t, strings.Count(dump, "CALL_SCALAR"),
		"expected opCallScalar for an all-scalar mixed-kind callee; disasm:\n%s", dump)
}

func TestCalleeUsesScalarBanksOnly_Direct(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		paramKinds  []registerKind
		resultKinds []registerKind
		want        bool
	}{
		{name: "empty signature", paramKinds: nil, resultKinds: nil, want: true},
		{name: "int to int", paramKinds: []registerKind{registerInt}, resultKinds: []registerKind{registerInt}, want: true},
		{name: "mixed scalar", paramKinds: []registerKind{registerInt, registerString, registerBool}, resultKinds: []registerKind{registerFloat}, want: true},
		{name: "general param", paramKinds: []registerKind{registerGeneral}, resultKinds: []registerKind{registerInt}, want: false},
		{name: "general result", paramKinds: []registerKind{registerInt}, resultKinds: []registerKind{registerGeneral}, want: false},
		{name: "general buried in mixed params", paramKinds: []registerKind{registerInt, registerGeneral, registerString}, resultKinds: []registerKind{registerInt}, want: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			callee := &CompiledFunction{
				parameterKinds: testCase.paramKinds,
				resultKinds:    testCase.resultKinds,
			}
			require.Equal(t, testCase.want, calleeUsesScalarBanksOnly(callee))
		})
	}
}
