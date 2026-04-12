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
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmitNarrowIntegerTruncationEmitsForNarrowUint(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	location := varLocation{register: 5, kind: registerUint}
	startLength := len(c.function.body)
	c.emitNarrowIntegerTruncation(location, types.Typ[types.Uint8])

	require.Len(t, c.function.body, startLength+1,
		"a single opTruncateNarrow instruction should be emitted for narrow uint8")
	emitted := c.function.body[startLength]
	require.Equal(t, opTruncateNarrow, emitted.op)
	require.Equal(t, uint8(5), emitted.a, "operand A is the register being truncated")
	require.Equal(t, uint8(8), emitted.b, "operand B is the bit width")
	require.Equal(t, uint8(registerUint), emitted.c, "operand C is the bank kind")
}

func TestEmitNarrowIntegerTruncationEmitsForNarrowInt(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	location := varLocation{register: 3, kind: registerInt}
	startLength := len(c.function.body)
	c.emitNarrowIntegerTruncation(location, types.Typ[types.Int16])

	require.Len(t, c.function.body, startLength+1)
	emitted := c.function.body[startLength]
	require.Equal(t, opTruncateNarrow, emitted.op)
	require.Equal(t, uint8(3), emitted.a)
	require.Equal(t, uint8(16), emitted.b)
	require.Equal(t, uint8(registerInt), emitted.c)
}

func TestEmitNarrowIntegerTruncationSkipsFullWidthTypes(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	tests := []struct {
		t    types.Type
		name string
	}{
		{name: "int", t: types.Typ[types.Int]},
		{name: "int64", t: types.Typ[types.Int64]},
		{name: "uint", t: types.Typ[types.Uint]},
		{name: "uint64", t: types.Typ[types.Uint64]},
		{name: "string", t: types.Typ[types.String]},
		{name: "float64", t: types.Typ[types.Float64]},
		{name: "bool", t: types.Typ[types.Bool]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startLength := len(c.function.body)
			c.emitNarrowIntegerTruncation(varLocation{register: 0, kind: registerInt}, tt.t)
			require.Equal(t, startLength, len(c.function.body),
				"no instruction should be emitted for full-width or non-integer types")
		})
	}
}

func TestEmitNarrowIntegerTruncationSkipsWrongBank(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	startLength := len(c.function.body)
	c.emitNarrowIntegerTruncation(varLocation{register: 0, kind: registerGeneral}, types.Typ[types.Uint8])
	require.Equal(t, startLength, len(c.function.body),
		"truncation only applies to int and uint banks; general bank is left untouched because the cell-side wrap is handled by reflect.SetUint when the boxed value is unpacked")
}

func TestEmitNarrowIntegerTruncationSkipsNilType(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	startLength := len(c.function.body)
	c.emitNarrowIntegerTruncation(varLocation{register: 0, kind: registerInt}, nil)
	require.Equal(t, startLength, len(c.function.body),
		"nil static type means the call site has no type info; skip the truncation rather than panic")
}
