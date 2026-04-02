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

package layouter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveCalcLength(t *testing.T) {
	ctx := defaultResolutionContext()

	tests := []struct {
		name     string
		unit     string
		value    float64
		expected float64
	}{
		{

			name:     "pixels to points",
			value:    100,
			unit:     "px",
			expected: 75,
		},
		{

			name:     "em units",
			value:    2,
			unit:     "em",
			expected: 24,
		},
		{

			name:     "rem units",
			value:    2,
			unit:     "rem",
			expected: 32,
		},
		{

			name:     "centimetres",
			value:    1,
			unit:     "cm",
			expected: 28.3465,
		},
		{

			name:     "millimetres",
			value:    10,
			unit:     "mm",
			expected: 28.3465,
		},
		{

			name:     "inches",
			value:    1,
			unit:     "in",
			expected: 72,
		},
		{

			name:     "picas",
			value:    1,
			unit:     "pc",
			expected: 12,
		},
		{

			name:     "viewport width units",
			value:    50,
			unit:     "vw",
			expected: 297.64,
		},
		{

			name:     "viewport height units",
			value:    50,
			unit:     "vh",
			expected: 420.945,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveCalcLength(tt.value, tt.unit, ctx)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestParseCalc_SimpleNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "integer",
			input:    "42",
			expected: 42,
		},
		{
			name:     "decimal",
			input:    "3.14",
			expected: 3.14,
		},
		{
			name:     "negative",
			input:    "-5",
			expected: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := parseCalc(tt.input)
			assert.NotNil(t, expr)
			assert.Equal(t, CalcNodeNumber, expr.Type)
			assert.InDelta(t, tt.expected, expr.Value, 0.001)
		})
	}
}

func TestParseCalc_Length(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedUnit string
		expectedVal  float64
	}{
		{
			name:         "pixels",
			input:        "100px",
			expectedVal:  100,
			expectedUnit: "px",
		},
		{
			name:         "em units",
			input:        "2em",
			expectedVal:  2,
			expectedUnit: "em",
		},
		{
			name:         "rem units",
			input:        "1.5rem",
			expectedVal:  1.5,
			expectedUnit: "rem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := parseCalc(tt.input)
			assert.NotNil(t, expr)
			assert.Equal(t, CalcNodeLength, expr.Type)
			assert.InDelta(t, tt.expectedVal, expr.Value, 0.001)
			assert.Equal(t, tt.expectedUnit, expr.Unit)
		})
	}
}

func TestParseCalc_Percentage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "fifty percent",
			input:    "50%",
			expected: 50,
		},
		{
			name:     "one hundred percent",
			input:    "100%",
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := parseCalc(tt.input)
			assert.NotNil(t, expr)
			assert.Equal(t, CalcNodePercentage, expr.Type)
			assert.InDelta(t, tt.expected, expr.Value, 0.001)
		})
	}
}

func TestParseCalc_Addition(t *testing.T) {

	expr := parseCalc("100px + 50px")
	assert.NotNil(t, expr)
	assert.Equal(t, CalcNodeAdd, expr.Type)

	ctx := defaultResolutionContext()
	result := expr.resolveCalc(ctx, ctx.ContainingBlockWidth)
	assert.InDelta(t, 112.5, result, 0.001)
}

func TestParseCalc_Subtraction(t *testing.T) {

	expr := parseCalc("200px - 50px")
	assert.NotNil(t, expr)
	assert.Equal(t, CalcNodeSubtract, expr.Type)

	ctx := defaultResolutionContext()
	result := expr.resolveCalc(ctx, ctx.ContainingBlockWidth)
	assert.InDelta(t, 112.5, result, 0.001)
}

func TestParseCalc_Multiplication(t *testing.T) {

	expr := parseCalc("10 * 5")
	assert.NotNil(t, expr)
	assert.Equal(t, CalcNodeMultiply, expr.Type)

	ctx := defaultResolutionContext()
	result := expr.resolveCalc(ctx, ctx.ContainingBlockWidth)
	assert.InDelta(t, 50.0, result, 0.001)
}

func TestParseCalc_Division(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "normal division",
			input:    "100 / 4",
			expected: 25,
		},
		{

			name:     "division by zero",
			input:    "100 / 0",
			expected: 0,
		},
	}

	ctx := defaultResolutionContext()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := parseCalc(tt.input)
			assert.NotNil(t, expr)
			result := expr.resolveCalc(ctx, ctx.ContainingBlockWidth)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestParseCalc_Parentheses(t *testing.T) {

	expr := parseCalc("(10 + 5) * 2")
	assert.NotNil(t, expr)

	ctx := defaultResolutionContext()
	result := expr.resolveCalc(ctx, ctx.ContainingBlockWidth)
	assert.InDelta(t, 30.0, result, 0.001)
}

func TestParseCalc_NestedCalc(t *testing.T) {

	expr := parseCalc("calc(10 + 5)")
	assert.NotNil(t, expr)

	ctx := defaultResolutionContext()
	result := expr.resolveCalc(ctx, ctx.ContainingBlockWidth)
	assert.InDelta(t, 15.0, result, 0.001)
}

func TestParseCalc_OperatorPrecedence(t *testing.T) {

	expr := parseCalc("2 + 3 * 4")
	assert.NotNil(t, expr)

	ctx := defaultResolutionContext()
	result := expr.resolveCalc(ctx, ctx.ContainingBlockWidth)
	assert.InDelta(t, 14.0, result, 0.001)
}

func TestParseCalc_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "non-numeric text",
			input: "abc",
		},
		{

			name:  "dangling operator",
			input: "10px +",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := parseCalc(tt.input)
			assert.Nil(t, expr)
		})
	}
}

func TestResolveCalc_NilExpression(t *testing.T) {

	var expr *calcExpression
	ctx := defaultResolutionContext()
	result := expr.resolveCalc(ctx, ctx.ContainingBlockWidth)
	assert.InDelta(t, 0.0, result, 0.001)
}

func TestResolveCalc_Percentage(t *testing.T) {

	expr := &calcExpression{Type: CalcNodePercentage, Value: 50}
	ctx := defaultResolutionContext()
	result := expr.resolveCalc(ctx, 200)
	assert.InDelta(t, 100.0, result, 0.001)
}

func TestResolveCalc_MixedUnits(t *testing.T) {

	expr := parseCalc("2em + 100px")
	assert.NotNil(t, expr)

	ctx := defaultResolutionContext()
	result := expr.resolveCalc(ctx, ctx.ContainingBlockWidth)
	assert.InDelta(t, 99.0, result, 0.001)
}
