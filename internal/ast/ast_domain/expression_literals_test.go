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

package ast_domain

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiteral_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		literal  Expression
		expected string
	}{
		{
			name:     "StringLiteral basic",
			literal:  &StringLiteral{Value: "hello"},
			expected: `"hello"`,
		},
		{
			name:     "StringLiteral with quotes",
			literal:  &StringLiteral{Value: `a "quoted" string`},
			expected: `"a \"quoted\" string"`,
		},
		{
			name:     "StringLiteral empty",
			literal:  &StringLiteral{Value: ""},
			expected: `""`,
		},
		{
			name:     "IntegerLiteral positive",
			literal:  &IntegerLiteral{Value: 12345},
			expected: "12345",
		},
		{
			name:     "IntegerLiteral zero",
			literal:  &IntegerLiteral{Value: 0},
			expected: "0",
		},
		{
			name:     "IntegerLiteral negative",
			literal:  &IntegerLiteral{Value: -987},
			expected: "-987",
		},
		{
			name:     "FloatLiteral positive",
			literal:  &FloatLiteral{Value: 123.45},
			expected: "123.45",
		},
		{
			name:     "FloatLiteral zero",
			literal:  &FloatLiteral{Value: 0.0},
			expected: "0",
		},
		{
			name:     "FloatLiteral negative",
			literal:  &FloatLiteral{Value: -0.5},
			expected: "-0.5",
		},
		{
			name:     "FloatLiteral whole number",
			literal:  &FloatLiteral{Value: 99.0},
			expected: "99",
		},
		{
			name:     "DecimalLiteral basic",
			literal:  &DecimalLiteral{Value: "123.456"},
			expected: "123.456d",
		},
		{
			name:     "DecimalLiteral integer",
			literal:  &DecimalLiteral{Value: "789"},
			expected: "789d",
		},
		{
			name:     "BigIntLiteral positive",
			literal:  &BigIntLiteral{Value: "12345678901234567890"},
			expected: "12345678901234567890n",
		},
		{
			name:     "BigIntLiteral zero",
			literal:  &BigIntLiteral{Value: "0"},
			expected: "0n",
		},
		{
			name:     "BigIntLiteral negative",
			literal:  &BigIntLiteral{Value: "-987"},
			expected: "-987n",
		},
		{
			name:     "RuneLiteral basic",
			literal:  &RuneLiteral{Value: 'a'},
			expected: "r'a'",
		},
		{
			name:     "RuneLiteral with single quote",
			literal:  &RuneLiteral{Value: '\''},
			expected: `r'\''`,
		},
		{
			name:     "RuneLiteral with newline",
			literal:  &RuneLiteral{Value: '\n'},
			expected: `r'\n'`,
		},
		{
			name:     "RuneLiteral with unicode",
			literal:  &RuneLiteral{Value: '🚀'},
			expected: "r'🚀'",
		},
		{
			name:     "DateTimeLiteral with Z",
			literal:  &DateTimeLiteral{Value: "2025-08-30T10:20:30Z"},
			expected: "dt'2025-08-30T10:20:30Z'",
		},
		{
			name:     "DateTimeLiteral with offset",
			literal:  &DateTimeLiteral{Value: "2025-08-30T10:20:30+01:00"},
			expected: "dt'2025-08-30T10:20:30+01:00'",
		},
		{
			name:     "DateLiteral",
			literal:  &DateLiteral{Value: "2025-08-30"},
			expected: "d'2025-08-30'",
		},
		{
			name:     "TimeLiteral",
			literal:  &TimeLiteral{Value: "15:04:05"},
			expected: "t'15:04:05'",
		},
		{
			name:     "BooleanLiteral true",
			literal:  &BooleanLiteral{Value: true},
			expected: "true",
		},
		{
			name:     "BooleanLiteral false",
			literal:  &BooleanLiteral{Value: false},
			expected: "false",
		},
		{
			name:     "NilLiteral",
			literal:  &NilLiteral{},
			expected: "nil",
		},
		{
			name:     "ObjectLiteral empty",
			literal:  &ObjectLiteral{Pairs: map[string]Expression{}},
			expected: `{}`,
		},
		{
			name: "ObjectLiteral basic",
			literal: &ObjectLiteral{Pairs: map[string]Expression{
				"a": &IntegerLiteral{Value: 1},
				"b": &BooleanLiteral{Value: true},
			}},
			expected: `{"a": 1, "b": true}`,
		},
		{
			name: "ObjectLiteral with new literal types",
			literal: &ObjectLiteral{Pairs: map[string]Expression{
				"cost":     &DecimalLiteral{Value: "99.99"},
				"shipDate": &DateLiteral{Value: "2025-12-25"},
			}},
			expected: `{"cost": 99.99d, "shipDate": d'2025-12-25'}`,
		},
		{
			name:     "ArrayLiteral empty",
			literal:  &ArrayLiteral{Elements: []Expression{}},
			expected: "[]",
		},
		{
			name: "ArrayLiteral with elements",
			literal: &ArrayLiteral{Elements: []Expression{
				&IntegerLiteral{Value: 1},
				&StringLiteral{Value: "two"},
			}},
			expected: `[1, "two"]`,
		},
		{
			name: "ArrayLiteral with new literal types",
			literal: &ArrayLiteral{Elements: []Expression{
				&DecimalLiteral{Value: "1.23"},
				&DateTimeLiteral{Value: "2025-01-01T00:00:00Z"},
			}},
			expected: `[1.23d, dt'2025-01-01T00:00:00Z']`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.literal.String())
		})
	}
}

func TestStrconvQuoteBehaviour(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `"foo"`, strconv.Quote("foo"))
	assert.Equal(t, `"foo \"bar\""`, strconv.Quote(`foo "bar"`))
}

func TestObjectLiteral_TransformIdentifiers(t *testing.T) {
	t.Parallel()

	originalMap := map[string]Expression{
		"a": &Identifier{Name: "varA"},
		"b": &BinaryExpression{
			Left:     &Identifier{Name: "varB"},
			Operator: OpPlus,
			Right:    &IntegerLiteral{Value: 1},
		},
		"c": &StringLiteral{Value: "no change"},
	}
	obj := &ObjectLiteral{Pairs: originalMap}

	transformer := func(s string) string {
		return "PREFIX_" + s
	}

	transformedExpr := obj.TransformIdentifiers(transformer)

	transformedObj, ok := transformedExpr.(*ObjectLiteral)
	require.True(t, ok, "TransformIdentifiers should return an ObjectLiteral")

	assert.NotSame(t, obj, transformedObj, "Should return a new ObjectLiteral instance")

	transformedObj.Pairs["a"] = &StringLiteral{Value: "MODIFIED"}
	assert.IsType(t, &Identifier{}, obj.Pairs["a"], "Modifying the new map should not affect the original map")

	freshTransformedExpr := obj.TransformIdentifiers(transformer)
	freshTransformedObj, ok := freshTransformedExpr.(*ObjectLiteral)
	require.True(t, ok)

	assertExprString(t, "PREFIX_varA", freshTransformedObj.Pairs["a"])
	assertExprString(t, "(PREFIX_varB + 1)", freshTransformedObj.Pairs["b"])
	assertExprString(t, `"no change"`, freshTransformedObj.Pairs["c"])
}

func TestArrayLiteral_TransformIdentifiers(t *testing.T) {
	t.Parallel()

	originalElements := []Expression{
		&Identifier{Name: "varA"},
		&BinaryExpression{
			Left:     &Identifier{Name: "varB"},
			Operator: OpPlus,
			Right:    &IntegerLiteral{Value: 1},
		},
		&StringLiteral{Value: "no change"},
	}
	arr := &ArrayLiteral{Elements: originalElements}

	transformer := func(s string) string {
		return "PREFIX_" + s
	}

	transformedExpr := arr.TransformIdentifiers(transformer)

	transformedArr, ok := transformedExpr.(*ArrayLiteral)
	require.True(t, ok, "TransformIdentifiers should return an ArrayLiteral")

	assert.NotSame(t, arr, transformedArr, "Should return a new ArrayLiteral instance")

	transformedArr.Elements[0] = &StringLiteral{Value: "MODIFIED"}
	assert.IsType(t, &Identifier{}, arr.Elements[0], "Modifying the new slice should not affect the original slice")

	freshTransformedExpr := arr.TransformIdentifiers(transformer)
	freshTransformedArr, ok := freshTransformedExpr.(*ArrayLiteral)
	require.True(t, ok)

	assertExprString(t, "PREFIX_varA", freshTransformedArr.Elements[0])
	assertExprString(t, "(PREFIX_varB + 1)", freshTransformedArr.Elements[1])
	assertExprString(t, `"no change"`, freshTransformedArr.Elements[2])
}

func TestSetLocation_UncoveredLiterals(t *testing.T) {
	t.Parallel()

	newLocation := Location{Line: 5, Column: 10, Offset: 100}
	newLength := 20

	testCases := []struct {
		setAndGet func() (Location, int)
		name      string
	}{
		{
			name: "DateLiteral",
			setAndGet: func() (Location, int) {
				dl := &DateLiteral{Value: "2025-01-15"}
				dl.SetLocation(newLocation, newLength)
				return dl.GetRelativeLocation(), dl.GetSourceLength()
			},
		},
		{
			name: "TimeLiteral",
			setAndGet: func() (Location, int) {
				tl := &TimeLiteral{Value: "14:30:00"}
				tl.SetLocation(newLocation, newLength)
				return tl.GetRelativeLocation(), tl.GetSourceLength()
			},
		},
		{
			name: "RuneLiteral",
			setAndGet: func() (Location, int) {
				rl := &RuneLiteral{Value: 'A'}
				rl.SetLocation(newLocation, newLength)
				return rl.GetRelativeLocation(), rl.GetSourceLength()
			},
		},
		{
			name: "BigIntLiteral",
			setAndGet: func() (Location, int) {
				bil := &BigIntLiteral{Value: "12345678901234567890"}
				bil.SetLocation(newLocation, newLength)
				return bil.GetRelativeLocation(), bil.GetSourceLength()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotLocation, gotLen := tc.setAndGet()
			assert.Equal(t, newLocation, gotLocation, "Location should match after SetLocation")
			assert.Equal(t, newLength, gotLen, "SourceLength should match after SetLocation")
		})
	}
}
