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

package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestMatricesNameEachTargetsFiniteCells(t *testing.T) {
	testCases := []struct {
		target string
		names  []string
	}{
		{target: "pages", names: []string{ArithmeticMatrix, TypedMatrix}},
		{target: "components", names: []string{ArithmeticMatrix, TypedMatrix, "PKC action order and family"}},
		{target: "auth-contracts", names: []string{AuthClaimMatrix, AuthPageMatrix, AuthActionMatrix}},
		{target: "form-shim-contracts", names: []string{FormShimMatrix}},
		{target: "graph-rebuild", names: []string{"partial graph and edit"}},
		{target: "partial-components", names: []string{"partial owner, locale and reload order"}},
		{target: "browser-events", names: []string{"event flag rotation"}},
		{target: "checkout-workflow", names: nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.target, func(t *testing.T) {
			c, err := Build(testCase.target, 7, 0, "")
			require.NoError(t, err)
			matrices := Matrices(&c)
			names := make([]string, 0, len(matrices))
			for _, matrix := range matrices {
				names = append(names, matrix.Name)
				assert.NotEmpty(t, matrix.Cells, matrix.Name)
				assert.LessOrEqual(t, len(matrix.Cells), matrix.Feasible, matrix.Name)
				assert.IsNonDecreasing(t, matrix.Cells, matrix.Name)
			}
			if testCase.names == nil {
				assert.Empty(t, names)
				return
			}
			assert.Equal(t, testCase.names, names)
		})
	}
}

func TestMatricesWithoutModelsAreEmpty(t *testing.T) {
	for _, target := range []string{"auth-contracts", "form-shim-contracts", "source-diagnostics", "browser-events"} {
		c := casefile.Case{Target: target}
		assert.Empty(t, Matrices(&c), target)
	}
	assert.Empty(t, Matrices(&casefile.Case{Target: "browser-events", Expected: casefile.Expected{Client: &casefile.ClientTransition{}}}))

	markup := Matrices(&casefile.Case{Target: "markup"})
	require.Len(t, markup, 1)
	assert.Empty(t, markup[0].Cells)
}

func TestGraphAndComponentMatricesNeedBothHalves(t *testing.T) {
	assert.Empty(t, graphMatrix([]string{"partial-graph-diamond-reverse"}).Cells)
	assert.Equal(t, []string{"diamond-reverse/leaf-true-helper-false"},
		graphMatrix([]string{"partial-graph-diamond-reverse", "graph-snapshot-leaf-true-helper-false", "partial-graph-other"}).Cells)
	assert.Empty(t, componentMatrix([]string{"client-recipe-03"}).Cells)
	assert.Equal(t, []string{"03/timer"}, componentMatrix([]string{"client-recipe-03", "client-extra-timer"}).Cells)
}

func TestNewMatrixSortsAndDeduplicatesCells(t *testing.T) {
	matrix := newMatrix("name", matrixServer, 9, []string{"b", "a", "b"})
	assert.Equal(t, Matrix{Name: "name", Environment: matrixServer, Cells: []string{"a", "b"}, Feasible: 9}, matrix)
}

func TestProgramMatrixCellRecognisesProgramPaths(t *testing.T) {
	output := casefile.ProgramOutput{ID: "sum", Value: "4"}
	name, id := ProgramMatrixCell(casefile.Probe{Program: true, Path: "/arithmetic"}, output)
	assert.Equal(t, ArithmeticMatrix, name)
	assert.Equal(t, "sum/4", id)
	name, id = ProgramMatrixCell(casefile.Probe{Program: true, Path: "/typed?mode=a"}, output)
	assert.Equal(t, TypedMatrix, name)
	assert.Equal(t, "/typed?mode=a/sum", id)
	name, id = ProgramMatrixCell(casefile.Probe{Program: false, Path: "/arithmetic"}, output)
	assert.Empty(t, name)
	assert.Empty(t, id)
}
