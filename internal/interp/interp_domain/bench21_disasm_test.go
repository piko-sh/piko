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
)

func TestBench21DenseLayerHitsSimd(t *testing.T) {
	source := `package main
func dense_forward(weights [][]float64, input []float64, output []float64) {
	for i := 0; i < len(output); i++ {
		row := weights[i]
		sum := 0.0
		for j := 0; j < len(row); j++ {
			sum += row[j] * input[j]
		}
		output[i] = sum
	}
}
func EntrypointRun() {
	weights := make([][]float64, 2)
	weights[0] = make([]float64, 4)
	weights[1] = make([]float64, 4)
	input := make([]float64, 4)
	output := make([]float64, 2)
	dense_forward(weights, input, output)
}`
	service := NewService()
	compiled, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	dis := compiled.DisassembleAssembly()
	if strings.Contains(dis, "SIMD_DOT_PRODUCT_FLOAT64") {
		t.Logf("SIMD dot product fires on dense layer inner loop")
	} else {
		t.Logf("SIMD does NOT fire on dense layer inner loop; disasm follows:\n%s", dis)
		t.Fail()
	}
}
