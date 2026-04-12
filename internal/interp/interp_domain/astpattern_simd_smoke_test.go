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

func compileForSimdSmoke(t *testing.T, source string) string {
	t.Helper()
	service := NewService()
	compiled, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoErrorf(t, err, "compile failed: %v", err)
	return compiled.DisassembleAssembly()
}

func TestSimdRecogniser_DotProduct(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() float64 {
	a := make([]float64, 8)
	b := make([]float64, 8)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
		b[i] = float64(8 - i)
	}
	sum := 0.0
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}`)
	require.Positivef(t, strings.Count(disasm, "SIMD_DOT_PRODUCT_FLOAT64"),
		"expected SIMD_DOT_PRODUCT_FLOAT64 in disasm; got:\n%s", disasm)
}

func TestSimdRecogniser_SumSlice(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() float64 {
	a := make([]float64, 4)
	a[0] = 1.5
	a[1] = 2.5
	a[2] = 3.5
	a[3] = 4.5
	sum := 0.0
	for i := 0; i < len(a); i++ {
		sum += a[i]
	}
	return sum
}`)
	require.Positivef(t, strings.Count(disasm, "SIMD_SUM_SLICE_FLOAT64"),
		"expected SIMD_SUM_SLICE_FLOAT64 in disasm; got:\n%s", disasm)
}

func TestSimdRecogniser_AddSlice(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() float64 {
	a := make([]float64, 4)
	b := make([]float64, 4)
	dst := make([]float64, 4)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
		b[i] = float64((i + 1) * 10)
	}
	for i := 0; i < len(a); i++ {
		dst[i] = a[i] + b[i]
	}
	return dst[0] + dst[1] + dst[2] + dst[3]
}`)
	require.Positivef(t, strings.Count(disasm, "SIMD_ADD_SLICE_FLOAT64"),
		"expected SIMD_ADD_SLICE_FLOAT64 in disasm; got:\n%s", disasm)
}

func TestSimdRecogniser_ScaleSlice(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() float64 {
	a := make([]float64, 4)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
	}
	k := 2.5
	for i := 0; i < len(a); i++ {
		a[i] *= k
	}
	return a[0] + a[1] + a[2] + a[3]
}`)
	require.Positivef(t, strings.Count(disasm, "SIMD_SCALE_SLICE_FLOAT64"),
		"expected SIMD_SCALE_SLICE_FLOAT64 in disasm; got:\n%s", disasm)
}

func TestSimdRecogniser_RefusesNonFloat64(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() int {
	a := make([]int, 4)
	b := make([]int, 4)
	for i := 0; i < len(a); i++ {
		a[i] = i + 1
		b[i] = 4 - i
	}
	sum := 0
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}`)
	require.Zerof(t, strings.Count(disasm, "SIMD_"),
		"expected NO SIMD opcode in int-slice loop; got:\n%s", disasm)
}

func TestSimdRecogniser_RefusesVariableUpperBound(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() float64 {
	a := make([]float64, 5)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
	}
	n := 3
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += a[i]
	}
	return sum
}`)
	require.Zerof(t, strings.Count(disasm, "SIMD_"),
		"expected NO SIMD opcode when upper bound is a variable: got:\n%s", disasm)
}

func TestSimdRecogniser_DotProductConstBound(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
const N = 1024
func EntrypointRun() float64 {
	a := make([]float64, N)
	b := make([]float64, N)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
		b[i] = float64(N - i)
	}
	sum := 0.0
	for i := 0; i < N; i++ {
		sum += a[i] * b[i]
	}
	return sum
}`)
	require.Positivef(t, strings.Count(disasm, "SIMD_DOT_PRODUCT_FLOAT64"),
		"expected SIMD_DOT_PRODUCT_FLOAT64 for const-bound loop; got:\n%s", disasm)
}

func TestSimdRecogniser_SumSliceConstBound(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
const N = 64
func EntrypointRun() float64 {
	a := make([]float64, N)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
	}
	sum := 0.0
	for i := 0; i < N; i++ {
		sum += a[i]
	}
	return sum
}`)
	require.Positivef(t, strings.Count(disasm, "SIMD_SUM_SLICE_FLOAT64"),
		"expected SIMD_SUM_SLICE_FLOAT64 for const-bound loop; got:\n%s", disasm)
}

func TestSimdRecogniser_AdoptsGeneralBankOperands(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() float64 {
	a := []float64{1, 2, 3, 4}
	b := []float64{4, 3, 2, 1}
	sum := 0.0
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}`)
	require.Positivef(t, strings.Count(disasm, "ADOPT_GENERAL_TO_SLICES_FLOAT"),
		"expected ADOPT_GENERAL_TO_SLICES_FLOAT prologue for general-bank operands; got:\n%s", disasm)
	require.Positivef(t, strings.Count(disasm, "SIMD_DOT_PRODUCT_FLOAT64"),
		"expected SIMD opcode after general-bank adoption; got:\n%s", disasm)
}

func TestSimdRecogniser_FiresOnFunctionParameterSlices(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func dotRow(row []float64, input []float64) float64 {
	sum := 0.0
	for j := 0; j < len(row); j++ {
		sum += row[j] * input[j]
	}
	return sum
}
func EntrypointRun() float64 {
	a := make([]float64, 8)
	b := make([]float64, 8)
	return dotRow(a, b)
}`)
	require.Positivef(t, strings.Count(disasm, "SIMD_DOT_PRODUCT_FLOAT64"),
		"expected SIMD opcode on typed-bank parameter slices; got:\n%s", disasm)
}
