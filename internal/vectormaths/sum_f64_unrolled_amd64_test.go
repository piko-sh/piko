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

//go:build !safe && !(js && wasm) && amd64

package vectormaths

import (
	"fmt"
	"math"
	"testing"
)

var (
	sumF64KernelBoundaryDimsAMD64 = []int{
		0, 1, 2, 3, 4, 7, 8, 9,
		15, 16, 17, 31, 32, 33,
		127, 128, 129, 255, 256,
		1535, 1536, 1537,
		4095, 4096, 4097,
	}
)

func TestSumF64KernelBoundariesAMD64(t *testing.T) {
	t.Parallel()

	kernels := []struct {
		name string
		fn   func(a []float64) float64
	}{
		{"SSE", sumF64SSE},
		{"AVX2", sumF64AVX2},
	}

	for _, k := range kernels {
		t.Run(k.name, func(t *testing.T) {
			t.Parallel()
			for _, length := range sumF64KernelBoundaryDimsAMD64 {
				t.Run(fmt.Sprintf("dim=%d", length), func(t *testing.T) {
					t.Parallel()
					a := makeSumF64Input(length, uint64(length)*0x9E3779B97F4A7C15+6, 0xBADF00DCAFEBABE)
					want := kahanSumF64(a)
					got := k.fn(a)
					tolerance := math.Max(1e-9, math.Abs(want)*1e-12)
					if math.Abs(got-want) > tolerance {
						t.Fatalf("%s dim=%d: got %v want %v (diff=%g, tol=%g)",
							k.name, length, got, want, math.Abs(got-want), tolerance)
					}
				})
			}
		})
	}
}
