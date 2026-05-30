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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoopUnroll_SmallConstBody(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func process(value int) {}
func EntrypointRun() int {
	total := 0
	for i := 0; i < 4; i++ {
		total += i * 2
	}
	return total
}`)
	require.Zerof(t, strings.Count(disasm, "JUMP "),
		"expected unrolled loop (no back-edge JUMP); got:\n%s", disasm)
}

func TestLoopUnroll_RefusesLargeTripCount(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() int {
	total := 0
	for i := 0; i < 32; i++ {
		total += i * 3
	}
	return total
}`)
	require.Positivef(t, strings.Count(disasm, "JUMP "),
		"expected scalar loop fallthrough for N=32; got:\n%s", disasm)
}

func TestLoopUnroll_RefusesBreak(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() int {
	total := 0
	for i := 0; i < 4; i++ {
		if i == 2 {
			break
		}
		total += i
	}
	return total
}`)
	require.Positivef(t, strings.Count(disasm, "JUMP "),
		"expected scalar loop fallthrough when body contains break; got:\n%s", disasm)
}

func TestLoopUnroll_SimdWinsPriority(t *testing.T) {
	t.Parallel()
	disasm := compileForSimdSmoke(t, `package main
func EntrypointRun() float64 {
	a := make([]float64, 4)
	b := make([]float64, 4)
	for i := 0; i < len(a); i++ {
		a[i] = float64(i + 1)
		b[i] = float64(4 - i)
	}
	sum := 0.0
	for i := 0; i < 4; i++ {
		sum += a[i] * b[i]
	}
	return sum
}`)
	require.Positivef(t, strings.Count(disasm, "SIMD_DOT_PRODUCT_FLOAT64"),
		"expected SIMD path for const-bound dot product; got:\n%s", disasm)
}
