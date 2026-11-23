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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestArithmeticMatrixCartesianProduct(t *testing.T) {
	c := casefile.Case{Files: map[string]string{}}
	require.NoError(t, addArithmeticMatrix(&c))
	require.NotEmpty(t, c.Probes)
	probe := c.Probes[0]
	require.Equal(t, "/arithmetic", probe.Path)
	require.Len(t, probe.Output, 486)
	cells := map[string]map[string]string{}
	for _, output := range probe.Output {
		if cells[output.Value] == nil {
			cells[output.Value] = map[string]string{}
		}
		_, exists := cells[output.Value][output.ID]
		require.False(t, exists, "duplicate cell %+v", output)
		require.True(t, output.HasValue, "unbound cell %+v", output)
		cells[output.Value][output.ID] = output.Text
	}
	operands := []int{-2, 0, 3}
	for _, first := range operands {
		for _, second := range operands {
			for _, third := range operands {
				label := fmt.Sprintf("%d/%d/%d", first, second, third)
				assert.Len(t, cells[label], 18, label)
			}
		}
	}
	source := c.Files["pages/arithmetic.pk"]
	testCases := []struct{ id, source, answer string }{
		{id: "expr-110", source: "(r.A - r.B) - r.C", answer: "-5"},
		{id: "expr-111", source: "r.A - (r.B - r.C)", answer: "1"},
		{id: "expr-021", source: "r.A + (r.B * r.C)", answer: "-2"},
		{id: "expr-200", source: "(r.A + r.B) * r.C", answer: "-6"},
	}
	for _, testCase := range testCases {
		selector := fmt.Sprintf(`data-fuzz-output="%s" :data-value="r.Label">{{ %s }}</output>`, testCase.id, testCase.source)
		assert.Contains(t, source, selector)
		assert.Equal(t, testCase.answer, cells["-2/0/3"][testCase.id], testCase.id)
	}
}

func TestApplyRejectsUnknownOperator(t *testing.T) {
	_, err := apply("/", 1, 2)
	require.ErrorIs(t, err, errMalformedProgram)
	_, err = arithmeticShape{id: "bad", outer: "+", inner: "^", left: true}.result(1, 2, 3)
	require.ErrorIs(t, err, errMalformedProgram)
	_, err = arithmeticShape{id: "bad", outer: "^", inner: "+"}.result(1, 2, 3)
	require.ErrorIs(t, err, errMalformedProgram)
}
