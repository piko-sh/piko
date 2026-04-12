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

func TestEvalGenericSpecialisationTypedSliceSurvivors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{

			name: "typed_slice_survives_index_read",
			code: `func sum[T ~int64](xs []T) T {
	var total T
	for i := 0; i < len(xs); i++ {
		total = total + xs[i]
	}
	return total
}
xs := []int64{1, 2, 3, 4, 5}
sum(xs)`,
			expect: int64(15),
		},
		{

			name: "typed_slice_demoted_by_spread_append",
			code: `func appendSpread[T ~int64](xs []T, more []T) []T {
	return append(xs, more...)
}
xs := []int64{1, 2, 3}
more := []int64{4, 5}
result := appendSpread(xs, more)
result[4]`,
			expect: int64(5),
		},
		{

			name: "typed_slice_typed_direct_append_returns_grown",
			code: `func appendOne[T ~int64](xs []T, v T) []T {
	return append(xs, v)
}
xs := []int64{1, 2, 3}
result := appendOne(xs, 4)
result[3]`,
			expect: int64(4),
		},
		{

			name: "typed_slice_demoted_by_addressof",
			code: `func passThrough[T ~int64](xs []T) []T {
	p := &xs
	return *p
}
xs := []int64{7, 8, 9}
passThrough(xs)[0]`,
			expect: int64(7),
		},
		{

			name: "bench18_int64_shape",
			code: `func gsum[T ~int64](xs []T) T {
	var total T
	for i := 0; i < len(xs); i++ {
		total = total + xs[i]
	}
	return total
}
xs := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
gsum(xs)`,
			expect: int64(55),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestSpecialisationParameterKindsTypedSliceSurvivor(t *testing.T) {
	t.Parallel()

	source := `package main

func sumTypedSurvives(xs []int64) int64 {
	var total int64
	for i := 0; i < len(xs); i++ {
		total = total + xs[i]
	}
	return total
}

func sumGenericSurvives[T ~int64](xs []T) T {
	var total T
	for i := 0; i < len(xs); i++ {
		total = total + xs[i]
	}
	return total
}

func sumGenericDemotedBySpread[T ~int64](xs []T, more []T) []T {
	return append(xs, more...)
}

func main() {
	xs := []int64{1, 2, 3}
	more := []int64{4, 5}
	_ = sumTypedSurvives(xs)
	_ = sumGenericSurvives(xs)
	_ = sumGenericDemotedBySpread(xs, more)
}
`
	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoError(t, err)

	baseline := findCompiledFunctionByName(t, cfs, "sumTypedSurvives")
	require.NotNil(t, baseline)
	require.NotEmpty(t, baseline.parameterKinds)
	require.Equalf(t,
		registerSliceInt,
		baseline.parameterKinds[0],
		"sumTypedSurvives.xs should stay on registerSliceInt (kind=%d) - sanity check for the non-specialised path",
		baseline.parameterKinds[0])

	specSurvivor := findCompiledFunctionBySpecPrefix(t, cfs, "sumGenericSurvives")
	require.NotNil(t, specSurvivor, "expected a specialisation of sumGenericSurvives in the compiled file set")
	require.NotEmpty(t, specSurvivor.parameterKinds)
	require.Equalf(t,
		registerSliceInt,
		specSurvivor.parameterKinds[0],
		"specialised sumGenericSurvives.xs should stay on registerSliceInt (kind=%d)",
		specSurvivor.parameterKinds[0])

	specDemoted := findCompiledFunctionBySpecPrefix(t, cfs, "sumGenericDemotedBySpread")
	require.NotNil(t, specDemoted, "expected a specialisation of sumGenericDemotedBySpread in the compiled file set")
	require.NotEmpty(t, specDemoted.parameterKinds)
	require.Equalf(t,
		registerGeneral,
		specDemoted.parameterKinds[0],
		"append-disqualified sumGenericDemotedBySpread.xs should demote to registerGeneral (kind=%d)",
		specDemoted.parameterKinds[0])
}

func findCompiledFunctionByName(t *testing.T, cfs *CompiledFileSet, name string) *CompiledFunction {
	t.Helper()
	fn, err := cfs.FindFunction(name)
	if err != nil {
		return nil
	}
	return fn
}

func findCompiledFunctionBySpecPrefix(t *testing.T, cfs *CompiledFileSet, origin string) *CompiledFunction {
	t.Helper()
	prefix := origin + "[spec:"
	for _, fn := range cfs.root.functions {
		if fn == nil {
			continue
		}
		if strings.HasPrefix(fn.name, prefix) {
			return fn
		}
	}
	return nil
}
